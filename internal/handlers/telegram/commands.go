package telegram

import (
	"fmt"
	"strings"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucClient/internal/interfaces"
	tele "gopkg.in/telebot.v3"
)

type CommandHandler struct {
	menu       *Menu
	settingsUC interfaces.SettingsUsecase
}

func NewCommandHandler(menu *Menu, settingsUC interfaces.SettingsUsecase) *CommandHandler {
	return &CommandHandler{
		menu:       menu,
		settingsUC: settingsUC,
	}
}

// OnStart обрабатывает команду /start и кнопку "Домой"
func (h *CommandHandler) OnStart(c tele.Context) error {
	// Сброс состояния при возврате в главное меню
	if err := h.settingsUC.SetState(c.Sender().ID, entities.StateIdle); err != nil {
		return c.Send("⚠️ Error resetting state: " + err.Error())
	}

	user := &entities.User{
		ID:        c.Sender().ID,
		FirstName: c.Sender().FirstName,
		UserName:  c.Sender().Username,
		State:     entities.StateIdle,
	}

	if err := h.settingsUC.RegisterUser(user); err != nil {
		return c.Send(fmt.Sprintf("⚠️ Error registering user: %s", err.Error()))
	}

	text := "👋 <b>Fanuc Client</b>\n\n" +
		"Главное меню управления подключениями\n" +
		"Используйте кнопки ниже для навигации"

	inlineMarkup := h.menu.BuildMainMenu()

	// Если вызов пришел из Callback (кнопка "В начало"), редактируем сообщение
	if c.Callback() != nil {
		return c.Edit(text, inlineMarkup)
	}
	// Иначе отправляем новое с клавиатурой
	return c.Send(text, h.menu.ReplyMain, inlineMarkup)
}

// OnWho обрабатывает запрос информации о пользователе
func (h *CommandHandler) OnWho(c tele.Context) error {
	userID := c.Sender().ID

	// 1. Получаем пользователя
	u, err := h.settingsUC.GetUser(userID)
	if err != nil {
		return c.Send("❌ Ошибка получения профиля.")
	}

	// 2. Получаем список целей (таргетов)
	targets, err := h.settingsUC.GetTargets(userID)
	if err != nil {
		targets = []entities.MonitoringTarget{}
	}

	// 3. Формируем сообщение
	var msg strings.Builder

	// Заголовок профиля
	msg.WriteString("🪪 <b>Профиль</b>\n\n")
	msg.WriteString(fmt.Sprintf("🆔 ID: <code>%d</code>\n", u.ID))
	msg.WriteString(fmt.Sprintf("👷 Имя: <b>%s</b>\n", u.FirstName))
	// ИЗМЕНЕНИЕ ЗДЕСЬ: FSM -> State
	msg.WriteString(fmt.Sprintf("⚙️ State: <code>%s</code>\n\n", u.State))

	// Блок подключений
	msg.WriteString(fmt.Sprintf("📡 <b>Активные подключения (%d):</b>\n", len(targets)))

	if len(targets) == 0 {
		msg.WriteString("<i>— Список пуст. Добавьте станки через меню.</i>")
	} else {
		for i, t := range targets {
			keyDisplay := t.Key
			if keyDisplay == "" {
				keyDisplay = "ALL (Без фильтра)"
			}

			// Красивое форматирование списка
			msg.WriteString(fmt.Sprintf("\n<b>%d. 🏭 %s</b>\n", i+1, t.Name))
			msg.WriteString(fmt.Sprintf("   ├ 🌐 <code>%s</code>\n", t.Broker))
			msg.WriteString(fmt.Sprintf("   ├ 📂 <code>%s</code>\n", t.Topic))
			msg.WriteString(fmt.Sprintf("   └ 🔑 <code>%s</code>", keyDisplay))
		}
	}

	markup := h.menu.BuildWhoMenu()

	if c.Callback() != nil {
		return c.Edit(msg.String(), markup)
	}
	return c.Send(msg.String(), markup)
}

// showTargetsList - вспомогательный метод для отображения списка целей (аналог в CallbackHandler)
func (h *CommandHandler) showTargetsList(c tele.Context) error {
	// Сбрасываем состояние
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	targets, err := h.settingsUC.GetTargets(c.Sender().ID)
	if err != nil {
		return c.Send("Error fetching targets: " + err.Error())
	}

	text := fmt.Sprintf("📋 <b>Ваши подключения (%d)</b>\n\nВыберите подключение или создайте новое", len(targets))
	markup := h.menu.BuildTargetsList(targets)

	return c.Send(text, markup)
}

// OnText обрабатывает текстовый ввод (FSM)
func (h *CommandHandler) OnText(c tele.Context) error {
	userID := c.Sender().ID
	user, err := h.settingsUC.GetUser(userID)
	if err != nil || user == nil {
		// Если пользователя нет в базе, отправляем на старт
		return h.OnStart(c)
	}

	input := strings.TrimSpace(c.Text())

	// Проверяем, является ли текст командой меню.
	switch input {
	case h.menu.BtnHome.Text:
		return h.OnStart(c)
	case h.menu.BtnWho.Text:
		return h.OnWho(c)
	case h.menu.BtnTargets.Text:
		return h.showTargetsList(c)
	}

	switch user.State {
	case entities.StateWaitingName:
		return h.processNameStep(c, userID, input)

	case entities.StateWaitingBroker:
		return h.processBrokerStep(c, userID, input)

	case entities.StateWaitingTopic:
		return h.processTopicStep(c, userID, input)

	case entities.StateWaitingKey:
		return h.processKeyStep(c, userID, input, user.DraftName)

	case entities.StateIdle:
		return c.Send("🤖 Я ожидаю команды меню. Нажмите /start для сброса.", h.menu.ReplyMain)

	default:
		// Если состояние неизвестно, сбрасываем его
		h.settingsUC.SetState(userID, entities.StateIdle)
		return c.Send("⚠️ Неизвестное состояние. Сброс...", h.menu.ReplyMain)
	}
}

func (h *CommandHandler) processNameStep(c tele.Context, userID int64, input string) error {
	if err := h.settingsUC.SetDraftName(userID, input); err != nil {
		return c.Send("Error saving name.")
	}
	return c.Send("🔌 <b>Шаг 2/4: Брокер</b>\n\nВведите адрес брокера (IP:PORT):", h.menu.BuildCancel())
}

func (h *CommandHandler) processBrokerStep(c tele.Context, userID int64, input string) error {
	if err := h.settingsUC.SetDraftBroker(userID, input); err != nil {
		return c.Send("Error saving broker.")
	}
	return c.Send("📂 <b>Шаг 3/4: Топик</b>\n\nВведите название Kafka Topic:", h.menu.BuildCancel())
}

func (h *CommandHandler) processTopicStep(c tele.Context, userID int64, input string) error {
	if err := h.settingsUC.SetDraftTopic(userID, input); err != nil {
		return c.Send("Error saving topic.")
	}
	return c.Send("🔑 <b>Шаг 4/4: Ключ (Опционально)</b>\n\nВведите Kafka Key (например, IP станка) или отправьте '0', '-' или 'no', чтобы читать любые последние сообщения:", h.menu.BuildCancel())
}

func (h *CommandHandler) processKeyStep(c tele.Context, userID int64, input, draftName string) error {
	finalKey := input
	if input == "0" || input == "-" || input == "no" {
		finalKey = ""
	}

	if err := h.settingsUC.SetDraftKeyAndSave(userID, finalKey); err != nil {
		return c.Send("❌ Ошибка при сохранении: " + err.Error())
	}

	c.Send(fmt.Sprintf("✅ Настройка <b>%s</b> сохранена!", draftName))

	// Показываем список таргетов после успешного сохранения
	return h.showTargetsList(c)
}
