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
	"github.com/iwtcode/fanucService"
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
	case "del_target":
		return h.onDeleteTarget(c, uID)

	case "add_key_start":
		return h.onAddKeyStart(c, uID)

	case "view_key":
		if len(parts) < 3 {
			return nil
		}
		keyID, _ := strconv.Atoi(parts[2])
		return h.onViewKey(c, uID, uint(keyID))

	case "del_key":
		if len(parts) < 3 {
			return nil
		}
		keyID, _ := strconv.Atoi(parts[2])
		return h.onDeleteKey(c, uID, uint(keyID))

	case "check_msg":
		if len(parts) < 3 {
			return nil
		}
		keyID, _ := strconv.Atoi(parts[2])
		return h.onCheckMessage(c, uID, uint(keyID))

	case "live_mode":
		if len(parts) < 3 {
			return nil
		}
		keyID, _ := strconv.Atoi(parts[2])
		return h.onLiveModeStart(c, uID, uint(keyID))

	case "stop_live":
		if len(parts) < 3 {
			return nil
		}
		keyID, _ := strconv.Atoi(parts[2])
		return h.onStopLive(c, uID, uint(keyID))

	// Services
	case "view_service", "svc_machines": // Merged action
		return h.onViewService(c, uID)
	case "del_service":
		return h.onDeleteService(c, uID)
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
		return c.Send("❌ Ошибка получения списка сервисов: " + safeErr)
	}

	text := fmt.Sprintf("🌐 <b>Ваши сервисы (%d)</b>\n\nВыберите <code>API Service</code> для управления:", len(services))
	markup := h.menu.BuildServicesList(services)

	if c.Callback() != nil {
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (h *CallbackHandler) onViewService(c tele.Context, svcID uint) error {
	h.stopUserLiveSession(c.Sender().ID)
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)
	c.Notify(tele.Typing)

	// 1. Get Service from DB
	s, err := h.settingsUC.GetServiceByID(svcID)
	if err != nil {
		return h.onListServices(c)
	}

	// 2. Get Machines from API
	machines, errMach := h.controlUC.ListMachines(context.Background(), svcID)

	// Prepare text
	safeName := html.EscapeString(s.Name)
	safeURL := html.EscapeString(s.BaseURL)

	text := fmt.Sprintf("🌐 <b>Сервис: %s</b>\n"+
		"🔗 URL: <code>%s</code>\n",
		safeName, safeURL)

	if errMach != nil {
		safeErr := html.EscapeString(errMach.Error())
		text += fmt.Sprintf("\n⚠️ <b>API Недоступен:</b>\n%s", safeErr)
		// We still show the menu (empty list) so user can delete the service if needed
		machines = []fanucService.MachineDTO{}
	} else {
		text += fmt.Sprintf("\n🔌 <b>Станки: %d</b>", len(machines))
	}

	text += "\n\nВыберите станок или действие:"

	markup := h.menu.BuildServiceView(svcID, machines)

	if c.Callback() != nil {
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (h *CallbackHandler) onDeleteService(c tele.Context, svcID uint) error {
	err := h.settingsUC.DeleteService(c.Sender().ID, svcID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "❌ Ошибка удаления сервиса"})
	} else {
		c.Respond(&tele.CallbackResponse{Text: "✅ Удалено!"})
	}
	return h.onListServices(c)
}

// --- Machine Actions Handlers ---

func (h *CallbackHandler) onViewMachine(c tele.Context, svcID uint, machineID string) error {
	c.Notify(tele.Typing)

	machine, err := h.controlUC.GetMachine(context.Background(), svcID, machineID)

	if machine == nil {
		safeErr := "Неизвестная ошибка"
		if err != nil {
			safeErr = err.Error()
		}
		c.Respond(&tele.CallbackResponse{Text: "❌ Не удалось загрузить станок: " + safeErr})
		// Fallback to service view
		return h.onViewService(c, svcID)
	}

	safeEP := html.EscapeString(machine.Endpoint)
	safeModel := html.EscapeString(machine.Model)
	safeSeries := html.EscapeString(machine.Series)

	// Status Emoji
	statusIcon := "🟢"
	if err != nil || machine.Status != "connected" {
		statusIcon = "🔴"
	}

	// Mode Emoji
	modeIcon := "⏸️"
	if machine.Mode == "polling" {
		modeIcon = "🔄"
	}

	text := fmt.Sprintf("📟 <b>Станок: %s</b>\n"+
		"ID: <code>%s</code>\n"+
		"Endpoint: <code>%s</code>\n"+
		"Model: %s\n"+
		"Series: %s\n"+
		"Timeout: %d ms\n"+
		"Status: %s <b>%s</b>\n"+
		"Mode: %s <b>%s</b>",
		safeModel,
		machine.ID,
		safeEP,
		safeModel,
		safeSeries,
		machine.Timeout,
		statusIcon, machine.Status,
		modeIcon, machine.Mode)

	if machine.Mode == "polling" {
		text += fmt.Sprintf("\nPolling Interval: %d ms", machine.Interval)
	}

	if err != nil {
		safeErr := html.EscapeString(err.Error())
		text += fmt.Sprintf("\n\n⚠️ <b>Внимание:</b>\n%s", safeErr)
	}

	markup := h.menu.BuildMachineView(svcID, *machine)

	if c.Callback() != nil {
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (h *CallbackHandler) onAddConnectionStart(c tele.Context, svcID uint) error {
	userID := c.Sender().ID
	h.settingsUC.SetState(userID, entities.StateWaitingConnEndpoint)
	h.settingsUC.SetContextSvcID(userID, svcID)

	return c.Edit("🔌 <b>Шаг 1/4: Endpoint</b>\n\nВведите IP адрес и порт станка (например: 192.168.1.10:8193):", h.menu.BuildCancel())
}

func (h *CallbackHandler) onDeleteConnection(c tele.Context, svcID uint, machineID string) error {
	c.Notify(tele.Typing)
	err := h.controlUC.DeleteMachine(context.Background(), svcID, machineID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "❌ Ошибка: " + err.Error()})
	} else {
		c.Respond(&tele.CallbackResponse{Text: "✅ Подключение удалено"})
	}
	// Return to service view (machine list)
	return h.onViewService(c, svcID)
}

func (h *CallbackHandler) onStartPollWizard(c tele.Context, svcID uint, machineID string) error {
	userID := c.Sender().ID
	h.settingsUC.SetState(userID, entities.StateWaitingPollInterval)
	h.settingsUC.SetContextSvcID(userID, svcID)
	h.settingsUC.SetContextMachineID(userID, machineID)

	return c.Edit("⏱ <b>Настройка опроса</b>\n\nВведите интервал опроса в миллисекундах (например, 5000):", h.menu.BuildCancel())
}

func (h *CallbackHandler) onStopPoll(c tele.Context, svcID uint, machineID string) error {
	c.Notify(tele.Typing)
	err := h.controlUC.StopPolling(context.Background(), svcID, machineID)
	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "❌ Ошибка остановки опроса: " + err.Error()})
	} else {
		c.Respond(&tele.CallbackResponse{Text: "✅ Опрос остановлен"})
	}
	return h.onViewMachine(c, svcID, machineID)
}

func (h *CallbackHandler) onGetProgram(c tele.Context, svcID uint, machineID string) error {
	c.Notify(tele.UploadingDocument)
	prog, err := h.controlUC.GetProgram(context.Background(), svcID, machineID)

	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "❌ Ошибка получения программы"})
		safeErr := html.EscapeString(err.Error())
		backMarkup := &tele.ReplyMarkup{}
		// Back leads to machine view
		backMarkup.Inline(backMarkup.Row(backMarkup.Data("🔙 Назад", fmt.Sprintf("vm:%d:%s", svcID, machineID))))

		if c.Callback() != nil {
			return c.Edit(fmt.Sprintf("❌ Ошибка:\n%s", safeErr), backMarkup)
		}
		return c.Send(fmt.Sprintf("❌ Ошибка:\n%s", safeErr), backMarkup)
	}

	doc := &tele.Document{
		File:     tele.FromReader(strings.NewReader(prog)),
		FileName: "GCODE.NC",
		Caption:  fmt.Sprintf("📄 Управляющая программа\nID: <code>%s</code> ", machineID),
		MIME:     "text/plain",
	}

	if err := c.Send(doc); err != nil {
		return c.Edit("❌ Не удалось отправить файл: " + err.Error())
	}

	return h.onViewMachine(c, svcID, machineID)
}

// --- Service Wizard ---

func (h *CallbackHandler) onAddServiceStart(c tele.Context) error {
	h.settingsUC.SetState(c.Sender().ID, entities.StateWaitingSvcName)
	return c.Edit("🖊 <b>Шаг 1/3: Название сервиса</b>\n\nПридумайте название (например, 'Главный цех'):", h.menu.BuildCancel())
}

// --- Kafka Handlers ---

func (h *CallbackHandler) onListTargets(c tele.Context) error {
	h.stopUserLiveSession(c.Sender().ID)
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	targets, err := h.settingsUC.GetTargets(c.Sender().ID)
	if err != nil {
		safeErr := html.EscapeString(err.Error())
		return c.Send("❌ Ошибка получения списка Targets: " + safeErr)
	}
	text := fmt.Sprintf("📋 <b>Kafka Targets (%d)</b>\n\nВыберите <code>Kafka Target</code> для управления:", len(targets))
	markup := h.menu.BuildTargetsList(targets)

	if c.Callback() != nil {
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (h *CallbackHandler) onViewTarget(c tele.Context, targetID uint) error {
	h.stopUserLiveSession(c.Sender().ID)
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	t, err := h.settingsUC.GetTargetByID(targetID)
	if err != nil {
		return h.onListTargets(c)
	}

	safeName := html.EscapeString(t.Name)
	safeBroker := html.EscapeString(t.Broker)
	safeTopic := html.EscapeString(t.Topic)

	text := fmt.Sprintf("📋 <b>Target: %s</b>\nBroker: <code>%s</code>\nTopic: <code>%s</code>\n\nВыберите ключ для мониторинга или действие:",
		safeName, safeBroker, safeTopic)
	markup := h.menu.BuildTargetView(*t)

	if c.Callback() != nil {
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (h *CallbackHandler) onDeleteTarget(c tele.Context, targetID uint) error {
	h.settingsUC.DeleteTarget(c.Sender().ID, targetID)
	c.Respond(&tele.CallbackResponse{Text: "✅ Target удален"})
	return h.onListTargets(c)
}

// --- Keys Handlers ---

func (h *CallbackHandler) onAddKeyStart(c tele.Context, targetID uint) error {
	h.settingsUC.SetState(c.Sender().ID, entities.StateWaitingNewKey)
	h.settingsUC.SetContextTargetID(c.Sender().ID, targetID)

	return c.Edit("🔑 <b>Добавление ключа</b>\n\nВведите ключ (фильтр):", h.menu.BuildCancel())
}

func (h *CallbackHandler) onViewKey(c tele.Context, targetID, keyID uint) error {
	h.stopUserLiveSession(c.Sender().ID)

	var text string

	if keyID == 0 {
		// Virtual Default Key
		text = "📂 <b>Просмотр по умолчанию</b>\n(Без фильтрации по ключу)"
	} else {
		// Real Key from DB
		key, err := h.settingsUC.GetKeyByID(keyID)
		if err != nil {
			return h.onViewTarget(c, targetID)
		}
		text = fmt.Sprintf("🔑 <b>Ключ</b>: <code>%s</code>", html.EscapeString(key.Key))
	}

	markup := h.menu.BuildKeyView(targetID, keyID)

	if c.Callback() != nil {
		return c.Edit(text, markup)
	}
	return c.Send(text, markup)
}

func (h *CallbackHandler) onDeleteKey(c tele.Context, targetID, keyID uint) error {
	h.settingsUC.DeleteKey(keyID)
	c.Respond(&tele.CallbackResponse{Text: "✅ Ключ удален"})
	return h.onViewTarget(c, targetID)
}

func (h *CallbackHandler) onCheckMessage(c tele.Context, targetID, keyID uint) error {
	c.Notify(tele.Typing)
	foundKey, msgRaw, err := h.monitoringUC.FetchLastKafkaMessage(context.Background(), targetID, keyID)

	// Always go back to the key view (even if it's default)
	backMarkup := h.menu.BuildKeyView(targetID, keyID)

	if err != nil {
		safeErr := html.EscapeString(err.Error())
		return c.Edit(fmt.Sprintf("❌ Ошибка:\n%s", safeErr), backMarkup)
	}

	prettyMsg := prettyPrintJSON(msgRaw)
	if len(prettyMsg) > 3800 {
		prettyMsg = prettyMsg[:3800] + "\n...[обрезано]"
	}
	safeMsg := html.EscapeString(prettyMsg)

	// Format text
	var textBuilder strings.Builder
	if foundKey != "" {
		textBuilder.WriteString(fmt.Sprintf("🔑 Ключ: <code>%s</code>\n", html.EscapeString(foundKey)))
	}
	textBuilder.WriteString(fmt.Sprintf("📨 Результат:\n<pre>%s</pre>", safeMsg))

	return c.Edit(textBuilder.String(), backMarkup)
}

// --- Live Mode ---

func (h *CallbackHandler) onLiveModeStart(c tele.Context, targetID, keyID uint) error {
	userID := c.Sender().ID
	h.stopUserLiveSession(userID)
	ctx, cancel := context.WithCancel(context.Background())
	h.liveSessions.Store(userID, cancel)

	target, _ := h.settingsUC.GetTargetByID(targetID)

	title := "LIVE"
	if target != nil {
		title = "LIVE: " + html.EscapeString(target.Name)
	}
	if keyID > 0 {
		k, _ := h.settingsUC.GetKeyByID(keyID)
		if k != nil {
			title += fmt.Sprintf(" [%s]", html.EscapeString(k.Key))
		}
	} else {
		title += " [Default]"
	}

	initialText := fmt.Sprintf("🔴 <b>%s</b>\n⏳ Подключение...", title)
	c.Edit(initialText, h.menu.BuildLiveView(targetID, keyID))
	go h.runLiveUpdateLoop(ctx, c, targetID, keyID, title)
	return nil
}

func (h *CallbackHandler) onStopLive(c tele.Context, targetID, keyID uint) error {
	h.stopUserLiveSession(c.Sender().ID)
	// Return to the Key View (works for both default and specific)
	return h.onViewKey(c, targetID, keyID)
}

func (h *CallbackHandler) runLiveUpdateLoop(ctx context.Context, c tele.Context, targetID, keyID uint, title string) {
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()
	var lastContent string

	update := func() {
		fetchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, msgRaw, err := h.monitoringUC.FetchLastKafkaMessage(fetchCtx, targetID, keyID)
		cancel()
		if ctx.Err() != nil {
			return
		}

		timestamp := time.Now().Format("15:04:05")
		var textBuilder strings.Builder
		textBuilder.WriteString(fmt.Sprintf("🔴 <b>%s</b>\nОбновлено: %s\n", title, timestamp))

		if err != nil {
			safeErr := html.EscapeString(err.Error())
			textBuilder.WriteString(fmt.Sprintf("❌ %s", safeErr))
		} else {

			p := prettyPrintJSON(msgRaw)
			if len(p) > 3500 {
				p = p[:3500] + "..."
			}
			safeP := html.EscapeString(p)
			textBuilder.WriteString(fmt.Sprintf("<pre>%s</pre>", safeP))
		}

		text := textBuilder.String()
		if text != lastContent {
			if err := c.Edit(text, h.menu.BuildLiveView(targetID, keyID)); err != nil {
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
	return c.Edit("🖊 <b>Шаг 1/3: Имя Kafka Target</b>\nВведите имя:", h.menu.BuildCancel())
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
