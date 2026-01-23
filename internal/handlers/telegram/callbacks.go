package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucClient/internal/interfaces"
	tele "gopkg.in/telebot.v3"
)

type CallbackHandler struct {
	menu         *Menu
	settingsUC   interfaces.SettingsUsecase
	monitoringUC interfaces.MonitoringUsecase
	controlUC    interfaces.ControlUsecase
	cmdHandler   *CommandHandler

	liveSessions sync.Map
}

func NewCallbackHandler(
	menu *Menu,
	sUC interfaces.SettingsUsecase,
	mUC interfaces.MonitoringUsecase,
	cUC interfaces.ControlUsecase,
	cmd *CommandHandler,
) *CallbackHandler {
	return &CallbackHandler{
		menu:         menu,
		settingsUC:   sUC,
		monitoringUC: mUC,
		controlUC:    cUC,
		cmdHandler:   cmd,
	}
}

func (h *CallbackHandler) OnCallback(c tele.Context) error {
	defer c.Respond()
	data := strings.TrimSpace(c.Callback().Data)

	// 1. Static Actions
	switch data {
	// Common
	case "home":
		return h.cmdHandler.OnStart(c)
	case "who_btn":
		return h.cmdHandler.OnWho(c)
	case "cancel_wizard":
		return h.onCancelWizard(c)

	// Kafka Targets
	case "add_target":
		return h.onAddTargetStart(c)
	case "targets_list", "back_to_list":
		return h.onListTargets(c)

	// Services
	case "services_list":
		return h.onListServices(c)
	case "add_service":
		return h.onAddServiceStart(c)
	}

	// 2. Dynamic Actions
	return h.handleDynamicCallback(c, data)
}

func (h *CallbackHandler) handleDynamicCallback(c tele.Context, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil
	}
	action := parts[0]
	// First arg is usually numeric ID (svcID or targetID)
	idVal, _ := strconv.Atoi(parts[1])
	uID := uint(idVal)

	switch action {
	// Kafka
	case "view_target":
		return h.onViewTarget(c, uID)
	case "check_msg":
		return h.onCheckMessage(c, uID)
	case "live_mode":
		return h.onLiveModeStart(c, uID)
	case "stop_live":
		return h.onStopLive(c, uID)
	case "del_target":
		return h.onDeleteTarget(c, uID)

	// Services
	case "view_service":
		return h.onViewService(c, uID)
	case "del_service":
		return h.onDeleteService(c, uID)
	case "svc_machines":
		return h.onListServiceMachines(c, uID)
	case "add_conn":
		return h.onAddConnectionStart(c, uID)

	// Machine Actions (Format: action:svcID:machineID)
	case "vm", "sp", "stp", "gp", "dc":
		if len(parts) < 3 {
			return nil
		}
		machineID := parts[2]
		switch action {
		case "vm": // view machine
			return h.onViewMachine(c, uID, machineID)
		case "sp": // start poll
			return h.onStartPollWizard(c, uID, machineID)
		case "stp": // stop poll
			return h.onStopPoll(c, uID, machineID)
		case "gp": // get program
			return h.onGetProgram(c, uID, machineID)
		case "dc": // delete connection
			return h.onDeleteConnection(c, uID, machineID)
		}
	}
	return nil
}

// --- Service Handlers ---

func (h *CallbackHandler) onListServices(c tele.Context) error {
	h.stopUserLiveSession(c.Sender().ID)
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	services, err := h.settingsUC.GetServices(c.Sender().ID)
	if err != nil {
		safeErr := html.EscapeString(err.Error())
		return c.Send("Error fetching services: " + safeErr)
	}

	text := fmt.Sprintf("🌐 <b>Ваши сервисы (%d)</b>\n\nВыберите сервис для управления:", len(services))
	markup := h.menu.BuildServicesList(services)

	if c.Callback() != nil {
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (h *CallbackHandler) onViewService(c tele.Context, svcID uint) error {
	s, err := h.settingsUC.GetServiceByID(svcID)
	if err != nil {
		return h.onListServices(c)
	}

	safeName := html.EscapeString(s.Name)
	safeURL := html.EscapeString(s.BaseURL)

	text := fmt.Sprintf("🌐 <b>Service: %s</b>\n\n"+
		"🔗 URL: <code>%s</code>\n"+
		"🔑 Key: <code>****</code>\n\n"+
		"Выберите действие:",
		safeName, safeURL)

	markup := h.menu.BuildServiceView(svcID)
	return c.Edit(text, markup)
}

func (h *CallbackHandler) onDeleteService(c tele.Context, svcID uint) error {
	err := h.settingsUC.DeleteService(c.Sender().ID, svcID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "Error deleting service"})
	} else {
		c.Respond(&tele.CallbackResponse{Text: "Deleted!"})
	}
	return h.onListServices(c)
}

func (h *CallbackHandler) onListServiceMachines(c tele.Context, svcID uint) error {
	c.Notify(tele.Typing)

	machines, err := h.controlUC.ListMachines(context.Background(), svcID)
	// Build menu even if error to allow back button
	if err != nil {
		backMarkup := h.menu.BuildServiceView(svcID) // Go back to service view
		safeErr := html.EscapeString(err.Error())
		return c.Edit(fmt.Sprintf("❌ <b>Error calling API:</b>\n%s", safeErr), backMarkup)
	}

	text := fmt.Sprintf("🔌 <b>Список станков (%d):</b>\n\nВыберите станок для управления:", len(machines))
	markup := h.menu.BuildMachinesList(svcID, machines)

	return c.Edit(text, markup)
}

// --- Machine Actions Handlers ---

func (h *CallbackHandler) onViewMachine(c tele.Context, svcID uint, machineID string) error {
	c.Notify(tele.Typing)
	machine, err := h.controlUC.GetMachine(context.Background(), svcID, machineID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "Error refreshing machine"})
		// Fallback to list
		return h.onListServiceMachines(c, svcID)
	}

	safeEP := html.EscapeString(machine.Endpoint)
	safeModel := html.EscapeString(machine.Model)
	safeSeries := html.EscapeString(machine.Series)

	statusIcon := "🟢"
	if machine.Status != "connected" {
		statusIcon = "🔴"
	}

	text := fmt.Sprintf("📟 <b>Станок: %s</b>\n"+
		"ID: <code>%s</code>\n"+
		"Address: <code>%s</code>\n"+
		"Model: %s (Series: %s)\n"+
		"Status: %s <b>%s</b>\n"+
		"Mode: <b>%s</b>",
		safeModel, machine.ID, safeEP, safeModel, safeSeries, statusIcon, machine.Status, machine.Mode)

	if machine.Mode == "polling" {
		text += fmt.Sprintf("\nPolling Interval: %d ms", machine.Interval)
	}

	markup := h.menu.BuildMachineView(svcID, *machine)
	return c.Edit(text, markup)
}

func (h *CallbackHandler) onAddConnectionStart(c tele.Context, svcID uint) error {
	userID := c.Sender().ID
	h.settingsUC.SetState(userID, entities.StateWaitingConnEndpoint)
	h.settingsUC.SetContextSvcID(userID, svcID)

	return c.Edit("🔌 <b>Шаг 1/2: Endpoint</b>\n\nВведите IP адрес и порт станка (например: 192.168.1.10:8193):", h.menu.BuildCancel())
}

func (h *CallbackHandler) onDeleteConnection(c tele.Context, svcID uint, machineID string) error {
	c.Notify(tele.Typing)
	err := h.controlUC.DeleteMachine(context.Background(), svcID, machineID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "Error: " + err.Error()})
	} else {
		c.Respond(&tele.CallbackResponse{Text: "Connection deleted"})
	}
	return h.onListServiceMachines(c, svcID)
}

func (h *CallbackHandler) onStartPollWizard(c tele.Context, svcID uint, machineID string) error {
	userID := c.Sender().ID
	h.settingsUC.SetState(userID, entities.StateWaitingPollInterval)
	h.settingsUC.SetContextSvcID(userID, svcID)
	h.settingsUC.SetContextMachineID(userID, machineID)

	return c.Edit("⏱ <b>Настройка опроса</b>\n\nВведите интервал опроса в миллисекундах (например, 1000):", h.menu.BuildCancel())
}

func (h *CallbackHandler) onStopPoll(c tele.Context, svcID uint, machineID string) error {
	c.Notify(tele.Typing)
	err := h.controlUC.StopPolling(context.Background(), svcID, machineID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "Error stopping polling"})
	} else {
		c.Respond(&tele.CallbackResponse{Text: "Polling stopped"})
	}
	// Refresh view
	return h.onViewMachine(c, svcID, machineID)
}

func (h *CallbackHandler) onGetProgram(c tele.Context, svcID uint, machineID string) error {
	c.Notify(tele.UploadingDocument)
	prog, err := h.controlUC.GetProgram(context.Background(), svcID, machineID)

	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "Error getting program"})
		safeErr := html.EscapeString(err.Error())

		// Rebuild "Back" button to stay on error screen or go back
		backMarkup := &tele.ReplyMarkup{}
		backMarkup.Inline(backMarkup.Row(backMarkup.Data("🔙 Back", fmt.Sprintf("vm:%d:%s", svcID, machineID))))

		return c.Edit(fmt.Sprintf("❌ Error:\n%s", safeErr), backMarkup)
	}

	// Создаем документ из строки
	doc := &tele.Document{
		File:     tele.FromReader(strings.NewReader(prog)),
		FileName: "GCODE.NC",
		Caption:  fmt.Sprintf("📄 Program from %s", machineID),
		MIME:     "text/plain",
	}

	// Отправляем файл НОВЫМ сообщением
	if err := c.Send(doc); err != nil {
		return c.Edit("❌ Failed to send file: " + err.Error())
	}

	// Возвращаем исходное сообщение (меню) в состояние просмотра станка,
	// чтобы у пользователя остался интерфейс управления.
	return h.onViewMachine(c, svcID, machineID)
}

// --- Service Wizard ---

func (h *CallbackHandler) onAddServiceStart(c tele.Context) error {
	h.settingsUC.SetState(c.Sender().ID, entities.StateWaitingSvcName)
	return c.Edit("🖊 <b>Шаг 1/3: Название сервиса</b>\n\nПридумайте название (например, 'Главный цех'):", h.menu.BuildCancel())
}

// --- Kafka Handlers (Existing) ---

func (h *CallbackHandler) onListTargets(c tele.Context) error {
	h.stopUserLiveSession(c.Sender().ID)
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	targets, err := h.settingsUC.GetTargets(c.Sender().ID)
	if err != nil {
		safeErr := html.EscapeString(err.Error())
		return c.Send("Error fetching targets: " + safeErr)
	}
	text := fmt.Sprintf("📋 <b>Kafka Targets (%d)</b>", len(targets))
	markup := h.menu.BuildTargetsList(targets)

	if c.Callback() != nil {
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (h *CallbackHandler) onViewTarget(c tele.Context, targetID uint) error {
	h.stopUserLiveSession(c.Sender().ID)
	t, err := h.settingsUC.GetTargetByID(targetID)
	if err != nil {
		return h.onListTargets(c)
	}
	keyDisplay := t.Key
	if keyDisplay == "" {
		keyDisplay = "None"
	}

	safeName := html.EscapeString(t.Name)
	safeBroker := html.EscapeString(t.Broker)
	safeTopic := html.EscapeString(t.Topic)
	safeKey := html.EscapeString(keyDisplay)

	text := fmt.Sprintf("🔩 <b>Target: %s</b>\nBroker: <code>%s</code>\nTopic: <code>%s</code>\nKey: <code>%s</code>",
		safeName, safeBroker, safeTopic, safeKey)
	markup := h.menu.BuildTargetView(targetID)
	return c.Edit(text, markup)
}

func (h *CallbackHandler) onDeleteTarget(c tele.Context, targetID uint) error {
	h.settingsUC.DeleteTarget(c.Sender().ID, targetID)
	return h.onListTargets(c)
}

func (h *CallbackHandler) onCheckMessage(c tele.Context, targetID uint) error {
	c.Notify(tele.Typing)
	msg, err := h.monitoringUC.FetchLastKafkaMessage(context.Background(), targetID)
	backMarkup := h.menu.BuildTargetView(targetID)
	if err != nil {
		safeErr := html.EscapeString(err.Error())
		return c.Edit(fmt.Sprintf("❌ Error:\n%s", safeErr), backMarkup)
	}
	prettyMsg := prettyPrintJSON(msg)
	if len(prettyMsg) > 3800 {
		prettyMsg = prettyMsg[:3800] + "\n...[truncated]"
	}
	safeMsg := html.EscapeString(prettyMsg)
	return c.Edit(fmt.Sprintf("📨 Result:\n<pre>%s</pre>", safeMsg), backMarkup)
}

// --- Live Mode ---

func (h *CallbackHandler) onLiveModeStart(c tele.Context, targetID uint) error {
	userID := c.Sender().ID
	h.stopUserLiveSession(userID)
	ctx, cancel := context.WithCancel(context.Background())
	h.liveSessions.Store(userID, cancel)

	target, _ := h.settingsUC.GetTargetByID(targetID)
	safeName := html.EscapeString(target.Name)

	initialText := fmt.Sprintf("🔴 <b>LIVE: %s</b>\n⏳ Connecting...", safeName)
	c.Edit(initialText, h.menu.BuildLiveView(targetID))
	go h.runLiveUpdateLoop(ctx, c, targetID, target.Name)
	return nil
}

func (h *CallbackHandler) onStopLive(c tele.Context, targetID uint) error {
	h.stopUserLiveSession(c.Sender().ID)
	return h.onViewTarget(c, targetID)
}

func (h *CallbackHandler) runLiveUpdateLoop(ctx context.Context, c tele.Context, targetID uint, name string) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	var lastContent string
	safeName := html.EscapeString(name)

	update := func() {
		fetchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		msgRaw, err := h.monitoringUC.FetchLastKafkaMessage(fetchCtx, targetID)
		cancel()
		if ctx.Err() != nil {
			return
		}

		timestamp := time.Now().Format("15:04:05")
		var text string
		if err != nil {
			safeErr := html.EscapeString(err.Error())
			text = fmt.Sprintf("🔴 <b>LIVE: %s</b>\nUpdated: %s\n❌ %s", safeName, timestamp, safeErr)
		} else {
			p := prettyPrintJSON(msgRaw)
			if len(p) > 3500 {
				p = p[:3500] + "..."
			}
			safeP := html.EscapeString(p)
			text = fmt.Sprintf("🔴 <b>LIVE: %s</b>\nUpdated: %s\n<pre>%s</pre>", safeName, timestamp, safeP)
		}
		if text != lastContent {
			if err := c.Edit(text, h.menu.BuildLiveView(targetID)); err != nil {
				h.stopUserLiveSession(c.Sender().ID)
			} else {
				lastContent = text
			}
		}
	}
	update()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			update()
		}
	}
}

func (h *CallbackHandler) stopUserLiveSession(userID int64) {
	if val, ok := h.liveSessions.Load(userID); ok {
		val.(context.CancelFunc)()
		h.liveSessions.Delete(userID)
	}
}

func (h *CallbackHandler) onAddTargetStart(c tele.Context) error {
	h.settingsUC.SetState(c.Sender().ID, entities.StateWaitingName)
	return c.Edit("🖊 <b>Шаг 1/4: Kafka Name</b>\nВведите имя:", h.menu.BuildCancel())
}

func (h *CallbackHandler) onCancelWizard(c tele.Context) error {
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)
	return h.cmdHandler.OnStart(c)
}

func prettyPrintJSON(input string) string {
	var temp interface{}
	if err := json.Unmarshal([]byte(input), &temp); err != nil {
		return input
	}
	pretty, _ := json.MarshalIndent(temp, "", "  ")
	return string(pretty)
}
