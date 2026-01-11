package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	// Commands
	b.Handle("/start", r.onStart)

	// Reply Menu Handlers
	b.Handle(&r.menu.BtnTargets, r.onListTargets)
	b.Handle(&r.menu.BtnWho, r.onWho)
	b.Handle(&r.menu.BtnHome, r.onStart)

	// Static Inline handlers
	b.Handle(&r.menu.BtnAddTarget, r.onAddTargetStart)
	b.Handle(&r.menu.BtnBack, r.onListTargets)
	b.Handle(&r.menu.BtnCancelWizard, r.onCancelWizard)
	b.Handle(&r.menu.BtnHomeInline, r.onStart)

	// Callback catch-all
	b.Handle(tele.OnCallback, r.onCallback)

	// Text Input (FSM)
	b.Handle(tele.OnText, r.onText)
}

// onCallback handles dynamic buttons and routing
func (r *Router) onCallback(c tele.Context) error {
	unique := c.Callback().Unique
	data := c.Callback().Data
	data = strings.TrimSpace(data)

	// 1. Проверяем статические кнопки (меню, отмена и т.д.) по Unique ID
	switch unique {
	case r.menu.BtnAddTarget.Unique:
		return r.onAddTargetStart(c)
	case r.menu.BtnBack.Unique:
		return r.onListTargets(c)
	case r.menu.BtnCancelWizard.Unique:
		return r.onCancelWizard(c)
	case r.menu.BtnHomeInline.Unique:
		return r.onStart(c)
	}

	// 2. Проверяем кастомные Data (созданные вручную в BuildMainMenu)
	switch data {
	case "targets_list": // <-- Теперь ловим этот ключ
		return r.onListTargets(c)
	case "back_to_list":
		return r.onListTargets(c)
	case "who_btn":
		return r.onWho(c)
	case "home":
		return r.onStart(c)
	}

	// 3. Парсим динамические кнопки (action:id)
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil
	}

	action := parts[0]
	idStr := parts[1]
	id, _ := strconv.Atoi(idStr)
	targetID := uint(id)

	switch action {
	case "view_target":
		return r.onViewTarget(c, targetID)
	case "check_msg":
		return r.onCheckMessage(c, targetID)
	case "del_target":
		return r.onDeleteTarget(c, targetID)
	}

	return nil
}

func (r *Router) onStart(c tele.Context) error {
	r.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	user := &entities.User{
		ID:        c.Sender().ID,
		FirstName: c.Sender().FirstName,
		UserName:  c.Sender().Username,
		State:     entities.StateIdle,
	}
	if err := r.settingsUC.RegisterUser(user); err != nil {
		return c.Send(fmt.Sprintf("⚠️ Error registering user: %s", err.Error()))
	}

	text := "👋 <b>Fanuc Client Configurator</b>\n\n" +
		"Главное меню управления.\n" +
		"Используйте кнопки ниже для навигации."

	inlineMarkup := r.menu.BuildMainMenu()

	if c.Callback() != nil {
		c.Respond()
		return c.Edit(text, inlineMarkup)
	}
	return c.Send(text, r.menu.ReplyMain, inlineMarkup)
}

func (r *Router) onWho(c tele.Context) error {
	u, _ := r.settingsUC.GetUser(c.Sender().ID)
	text := fmt.Sprintf("👤 <b>User Info</b>\n\n"+
		"🆔 ID: <code>%d</code>\n"+
		"📛 Name: <b>%s</b>\n"+
		"🏷 State: <code>%s</code>",
		u.ID, u.FirstName, u.State)

	markup := r.menu.BuildWhoMenu()

	if c.Callback() != nil {
		c.Respond()
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

// --- Targets List & Management ---

func (r *Router) onListTargets(c tele.Context) error {
	r.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	targets, err := r.settingsUC.GetTargets(c.Sender().ID)
	if err != nil {
		return c.Send("Error fetching targets: " + err.Error())
	}

	text := fmt.Sprintf("📋 <b>Ваши настройки (%d)</b>\n\nВыберите настройку для проверки или создайте новую.", len(targets))
	markup := r.menu.BuildTargetsList(targets)

	if c.Callback() != nil {
		c.Respond()
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (r *Router) onViewTarget(c tele.Context, targetID uint) error {
	t, err := r.settingsUC.GetTargetByID(targetID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "Target not found"})
		return r.onListTargets(c)
	}

	text := fmt.Sprintf("🔩 <b>Target: %s</b>\n\n"+
		"🔌 Broker: <code>%s</code>\n"+
		"📝 Topic: <code>%s</code>\n"+
		"🔑 Key: <code>%s</code>\n\n"+
		"📅 Created: %s",
		t.Name, t.Broker, t.Topic, nonEmpty(t.Key, "None (Read Last)"), t.CreatedAt.Format("02 Jan 15:04"))

	markup := r.menu.BuildTargetView(targetID)
	c.Respond()
	return c.Edit(text, markup)
}

func (r *Router) onDeleteTarget(c tele.Context, targetID uint) error {
	err := r.settingsUC.DeleteTarget(c.Sender().ID, targetID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "Error deleting target"})
	} else {
		c.Respond(&tele.CallbackResponse{Text: "Deleted!"})
	}
	return r.onListTargets(c)
}

func (r *Router) onCheckMessage(c tele.Context, targetID uint) error {
	c.Respond()

	msg, err := r.monitoringUC.FetchLastKafkaMessage(context.Background(), targetID)
	backMarkup := r.menu.BuildTargetView(targetID)

	if err != nil {
		return c.Edit(fmt.Sprintf("❌ <b>Error:</b>\n%s", err.Error()), backMarkup)
	}

	prettyMsg := prettyPrintJSON(msg)
	if len(prettyMsg) > 3800 {
		prettyMsg = prettyMsg[:3800] + "\n...[truncated]"
	}

	text := fmt.Sprintf("📨 <b>Result:</b>\n\n<pre>%s</pre>", prettyMsg)

	return c.Edit(text, backMarkup)
}

// --- Wizard FSM ---

func (r *Router) onAddTargetStart(c tele.Context) error {
	r.settingsUC.SetState(c.Sender().ID, entities.StateWaitingName)
	c.Respond()
	return c.Edit("🖊 <b>Шаг 1/4: Название</b>\n\nВведите понятное имя для этого станка (например, 'Токарный 1'):", r.menu.BuildCancel())
}

func (r *Router) onCancelWizard(c tele.Context) error {
	r.settingsUC.SetState(c.Sender().ID, entities.StateIdle)
	c.Respond()
	return r.onListTargets(c)
}

func (r *Router) onText(c tele.Context) error {
	userID := c.Sender().ID
	user, err := r.settingsUC.GetUser(userID)
	if err != nil || user == nil {
		return r.onStart(c)
	}

	input := strings.TrimSpace(c.Text())

	if input == r.menu.BtnTargets.Text || input == r.menu.BtnWho.Text || input == r.menu.BtnHome.Text {
		return nil
	}

	switch user.State {
	case entities.StateWaitingName:
		if err := r.settingsUC.SetDraftName(userID, input); err != nil {
			return c.Send("Error saving state.")
		}
		return c.Send("🔌 <b>Шаг 2/4: Брокер</b>\n\nВведите адрес брокера (IP:PORT):", r.menu.BuildCancel())

	case entities.StateWaitingBroker:
		if err := r.settingsUC.SetDraftBroker(userID, input); err != nil {
			return c.Send("Error saving state.")
		}
		return c.Send("📝 <b>Шаг 3/4: Топик</b>\n\nВведите название Kafka Topic:", r.menu.BuildCancel())

	case entities.StateWaitingTopic:
		if err := r.settingsUC.SetDraftTopic(userID, input); err != nil {
			return c.Send("Error saving state.")
		}
		return c.Send("🔑 <b>Шаг 4/4: Ключ (Опционально)</b>\n\nВведите Kafka Key (например, IP станка) или отправьте '0', '-' или 'no', чтобы читать любые последние сообщения:", r.menu.BuildCancel())

	case entities.StateWaitingKey:
		finalKey := input
		if input == "0" || input == "-" || input == "no" {
			finalKey = ""
		}

		if err := r.settingsUC.SetDraftKeyAndSave(userID, finalKey); err != nil {
			return c.Send("❌ Ошибка при сохранении: " + err.Error())
		}

		c.Send(fmt.Sprintf("✅ Настройка <b>%s</b> сохранена!", user.DraftName))
		return r.onListTargets(c)

	case entities.StateIdle:
		return c.Send("Я вас не понимаю. Используйте меню или нажмите /start.", r.menu.ReplyMain)

	default:
		return c.Send("Неизвестное состояние. Сброс...", r.menu.ReplyMain)
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
	pretty, err := json.MarshalIndent(temp, "", "  ")
	if err != nil {
		return input
	}
	return string(pretty)
}
