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

type CallbackHandler struct {
	menu         *Menu
	settingsUC   interfaces.SettingsUsecase
	monitoringUC interfaces.MonitoringUsecase
	cmdHandler   *CommandHandler
}

func NewCallbackHandler(menu *Menu, sUC interfaces.SettingsUsecase, mUC interfaces.MonitoringUsecase, cmd *CommandHandler) *CallbackHandler {
	return &CallbackHandler{
		menu:         menu,
		settingsUC:   sUC,
		monitoringUC: mUC,
		cmdHandler:   cmd,
	}
}

// OnCallback - маршрутизатор для всех callback-запросов
func (h *CallbackHandler) OnCallback(c tele.Context) error {
	defer c.Respond()

	unique := c.Callback().Unique
	data := strings.TrimSpace(c.Callback().Data)

	// 1. Проверка Data
	switch data {
	case "add_target":
		return h.onAddTargetStart(c)
	case "cancel_wizard":
		return h.onCancelWizard(c)
	case "targets_list", "back_to_list":
		return h.onListTargets(c)
	case "who_btn":
		return h.cmdHandler.OnWho(c)
	case "home":
		return h.cmdHandler.OnStart(c)
	}

	// 2. Проверка Unique
	switch unique {
	case h.menu.BtnAddTarget.Unique:
		return h.onAddTargetStart(c)
	case h.menu.BtnBack.Unique:
		return h.onListTargets(c)
	case h.menu.BtnCancelWizard.Unique:
		return h.onCancelWizard(c)
	case h.menu.BtnHomeInline.Unique:
		return h.cmdHandler.OnStart(c)
	}

	// 3. Динамические
	return h.handleDynamicCallback(c, data)
}

func (h *CallbackHandler) handleDynamicCallback(c tele.Context, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil
	}

	action := parts[0]
	idStr := parts[1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil
	}
	targetID := uint(id)

	switch action {
	case "view_target":
		return h.onViewTarget(c, targetID)
	case "check_msg":
		return h.onCheckMessage(c, targetID)
	case "del_target":
		return h.onDeleteTarget(c, targetID)
	}
	return nil
}

// --- Specific Handlers ---

func (h *CallbackHandler) onListTargets(c tele.Context) error {
	// Сбрасываем состояние FSM
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	targets, err := h.settingsUC.GetTargets(c.Sender().ID)
	if err != nil {
		return c.Send("Error fetching targets: " + err.Error())
	}

	text := fmt.Sprintf("📋 <b>Ваши настройки (%d)</b>\n\nВыберите настройку для проверки или создайте новую.", len(targets))
	markup := h.menu.BuildTargetsList(targets)

	// ИСПРАВЛЕНИЕ:
	// Если вызов пришел через Callback (Inline кнопка), мы редактируем сообщение.
	// Если через Reply кнопку (текст), мы отправляем новое.
	if c.Callback() != nil {
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (h *CallbackHandler) onViewTarget(c tele.Context, targetID uint) error {
	t, err := h.settingsUC.GetTargetByID(targetID)
	if err != nil {
		return h.onListTargets(c)
	}

	keyDisplay := t.Key
	if keyDisplay == "" {
		keyDisplay = "None (Read Last)"
	}

	text := fmt.Sprintf("🔩 <b>Target: %s</b>\n\n"+
		"🔌 Broker: <code>%s</code>\n"+
		"📝 Topic: <code>%s</code>\n"+
		"🔑 Key: <code>%s</code>\n\n"+
		"📅 Created: %s",
		t.Name, t.Broker, t.Topic, keyDisplay, t.CreatedAt.Format("02 Jan 15:04"))

	markup := h.menu.BuildTargetView(targetID)
	return c.Edit(text, markup)
}

func (h *CallbackHandler) onDeleteTarget(c tele.Context, targetID uint) error {
	err := h.settingsUC.DeleteTarget(c.Sender().ID, targetID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "Error deleting target"})
	} else {
		c.Respond(&tele.CallbackResponse{Text: "Deleted!"})
	}
	return h.onListTargets(c)
}

func (h *CallbackHandler) onCheckMessage(c tele.Context, targetID uint) error {
	c.Notify(tele.Typing)

	msg, err := h.monitoringUC.FetchLastKafkaMessage(context.Background(), targetID)
	backMarkup := h.menu.BuildTargetView(targetID)

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

func (h *CallbackHandler) onAddTargetStart(c tele.Context) error {
	h.settingsUC.SetState(c.Sender().ID, entities.StateWaitingName)
	return c.Edit("🖊 <b>Шаг 1/4: Название</b>\n\nВведите понятное имя для этого станка (например, 'Токарный 1'):", h.menu.BuildCancel())
}

func (h *CallbackHandler) onCancelWizard(c tele.Context) error {
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)
	return h.onListTargets(c)
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
