package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucClient/internal/interfaces"
	tele "gopkg.in/telebot.v3"
)

type Router struct {
	menu         *Menu
	settingsUC   interfaces.SettingsUsecase
	monitoringUC interfaces.MonitoringUsecase
}

func NewRouter(menu *Menu, sUC interfaces.SettingsUsecase, mUC interfaces.MonitoringUsecase) *Router {
	return &Router{
		menu:         menu,
		settingsUC:   sUC,
		monitoringUC: mUC,
	}
}

func (r *Router) Register(b *tele.Bot) {
	// Команды
	b.Handle("/start", r.onStart)

	// Основные функции (регистрируем и на Reply, и на Inline кнопки)
	b.Handle(&r.menu.BtnLastMsgReply, r.onLastMessage)
	b.Handle(&r.menu.BtnLastMsgInline, r.onLastMessage)

	b.Handle(&r.menu.BtnSettingsReply, r.onSettings)
	b.Handle(&r.menu.BtnSettingsInline, r.onSettings)

	b.Handle(&r.menu.BtnWhoReply, r.onWho)
	b.Handle(&r.menu.BtnWhoInline, r.onWho)

	// Навигация внутри настроек (только Inline)
	b.Handle(&r.menu.BtnSetBroker, r.onSetBrokerBtn)
	b.Handle(&r.menu.BtnSetTopic, r.onSetTopicBtn)
	b.Handle(&r.menu.BtnCancelConfig, r.onBackToMain) // Кнопка "Back" возвращает в главное меню

	// Текстовые сообщения (для ввода данных FSM)
	b.Handle(tele.OnText, r.onText)
}

// refreshMessage - универсальная функция ответа.
// Если взаимодействие через Inline кнопку -> редактирует сообщение.
// Если через Reply кнопку или команду -> отправляет новое сообщение.
func (r *Router) refreshMessage(c tele.Context, text string, markup *tele.ReplyMarkup) error {
	// Если это callback (нажатие inline кнопки)
	if c.Callback() != nil {
		// Обязательно отвечаем на callback, чтобы убрать часики загрузки
		c.Respond()
		// Редактируем текущее сообщение
		return c.Edit(text, markup)
	}

	// Если это обычное текстовое сообщение, отправляем ответ с:
	// 1. Текстом и Inline клавиатурой (markup)
	// 2. Reply клавиатурой (она задается в опциях отправки, если нужна, но обычно она ставится один раз при /start)
	// В данном случае мы всегда прикрепляем переданный Inline markup к сообщению.
	return c.Send(text, markup)
}

func (r *Router) onStart(c tele.Context) error {
	user := &entities.User{
		ID:        c.Sender().ID,
		FirstName: c.Sender().FirstName,
		UserName:  c.Sender().Username,
		State:     entities.StateIdle,
	}
	if err := r.settingsUC.RegisterUser(user); err != nil {
		return c.Send("⚠️ Error registering user: " + err.Error())
	}

	// При старте отправляем приветствие и устанавливаем нижнее меню (ReplyMain)
	// А к самому сообщению прикрепляем InlineMain
	return c.Send("👋 <b>Панель управления Fanuc Client</b>\n\nИспользуйте меню для навигации.",
		r.menu.ReplyMain, r.menu.InlineMain)
}

func (r *Router) onSettings(c tele.Context) error {
	user, _ := r.settingsUC.GetUser(c.Sender().ID)

	text := fmt.Sprintf("⚙️ <b>Configuration</b>\n\n"+
		"🔌 Broker: <code>%s</code>\n"+
		"📝 Topic: <code>%s</code>\n\n"+
		"Выберите параметр для изменения:",
		nonEmpty(user.KafkaBroker, "not set"),
		nonEmpty(user.KafkaTopic, "not set"),
	)

	// Переходим в меню настроек (InlineSettings)
	return r.refreshMessage(c, text, r.menu.InlineSettings)
}

func (r *Router) onLastMessage(c tele.Context) error {
	// Если это нажатие кнопки, можно показать уведомление "Загрузка..." через c.Respond
	// Но для простоты сразу делаем запрос
	ctx := context.Background()
	msg, err := r.monitoringUC.FetchLastKafkaMessage(ctx, c.Sender().ID)

	if err != nil {
		return r.refreshMessage(c, fmt.Sprintf("❌ <b>Error:</b>\n%s", err.Error()), r.menu.InlineMain)
	}

	prettyMsg := prettyPrintJSON(msg)
	text := fmt.Sprintf("📨 <b>Last Kafka Message:</b>\n\n<pre>%s</pre>", prettyMsg)

	return r.refreshMessage(c, text, r.menu.InlineMain)
}

func (r *Router) onWho(c tele.Context) error {
	u, _ := r.settingsUC.GetUser(c.Sender().ID)
	text := fmt.Sprintf("👤 <b>User Info</b>\n\n🆔 ID: <code>%d</code>\n📛 Name: <b>%s</b>\n🏷 State: <code>%s</code>",
		u.ID, u.FirstName, u.State)

	return r.refreshMessage(c, text, r.menu.InlineMain)
}

// Inline handlers for Settings

func (r *Router) onSetBrokerBtn(c tele.Context) error {
	r.settingsUC.SetState(c.Sender().ID, entities.StateWaitingBroker)
	text := "🔌 <b>Setting Broker</b>\n\nОтправьте IP:PORT брокера (например, <code>192.168.1.50:9092</code>):"

	// Здесь мы убираем кнопки, чтобы пользователь сосредоточился на вводе,
	// или можно оставить кнопку "Отмена"
	return r.refreshMessage(c, text, r.menu.InlineSettings)
}

func (r *Router) onSetTopicBtn(c tele.Context) error {
	r.settingsUC.SetState(c.Sender().ID, entities.StateWaitingTopic)
	text := "📝 <b>Setting Topic</b>\n\nОтправьте название Topic:"

	return r.refreshMessage(c, text, r.menu.InlineSettings)
}

func (r *Router) onBackToMain(c tele.Context) error {
	// Сбрасываем стейт и возвращаемся в главное меню
	r.settingsUC.SetState(c.Sender().ID, entities.StateIdle)
	return r.refreshMessage(c, "👋 <b>Главное меню</b>", r.menu.InlineMain)
}

// State Machine Handler (Text Input)

func (r *Router) onText(c tele.Context) error {
	user, err := r.settingsUC.GetUser(c.Sender().ID)
	if err != nil || user == nil {
		return r.onStart(c)
	}

	input := strings.TrimSpace(c.Text())

	// Игнорируем текст, если он совпадает с кнопками Reply меню,
	// так как они обрабатываются отдельными хендлерами
	if input == r.menu.BtnSettingsReply.Text ||
		input == r.menu.BtnLastMsgReply.Text ||
		input == r.menu.BtnWhoReply.Text {
		return nil
	}

	switch user.State {
	case entities.StateWaitingBroker:
		if err := r.settingsUC.SetBroker(user.ID, input); err != nil {
			return c.Send("❌ Ошибка сохранения.", r.menu.InlineSettings)
		}
		return c.Send(fmt.Sprintf("✅ Broker сохранен: <code>%s</code>", input), r.menu.InlineSettings)

	case entities.StateWaitingTopic:
		if err := r.settingsUC.SetTopic(user.ID, input); err != nil {
			return c.Send("❌ Ошибка сохранения.", r.menu.InlineSettings)
		}
		return c.Send(fmt.Sprintf("✅ Topic сохранен: <code>%s</code>", input), r.menu.InlineSettings)

	default:
		// Если состояние idle и текст не команда
		return c.Send("🤔 Я не понимаю это сообщение. Используйте кнопки меню.", r.menu.InlineMain)
	}
}

// Helpers

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func prettyPrintJSON(input string) string {
	var temp interface{}
	if err := json.Unmarshal([]byte(input), &temp); err != nil {
		return input
	}
	pretty, err := json.MarshalIndent(temp, "", "    ")
	if err != nil {
		return input
	}
	return string(pretty)
}
