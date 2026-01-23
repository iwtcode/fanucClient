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
	idStr := parts[1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil
	}
	uID := uint(id)

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
	}
	return nil
}

// --- Service Handlers ---

func (h *CallbackHandler) onListServices(c tele.Context) error {
	h.stopUserLiveSession(c.Sender().ID)
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	services, err := h.settingsUC.GetServices(c.Sender().ID)
	if err != nil {
		// Экранируем ошибку, чтобы не сломать разметку, если там спецсимволы
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

	// Экранируем данные из БД
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
	backMarkup := h.menu.BuildBackToService(svcID)

	if err != nil {
		// Экранируем текст ошибки, так как он может содержать HTML (например <!DOCTYPE...)
		safeErr := html.EscapeString(err.Error())
		return c.Edit(fmt.Sprintf("❌ <b>Error calling API:</b>\n%s", safeErr), backMarkup)
	}

	// Сериализуем ответ API в JSON для отображения
	jsonBytes, err := json.MarshalIndent(machines, "", "  ")
	if err != nil {
		safeErr := html.EscapeString(err.Error())
		return c.Edit(fmt.Sprintf("❌ <b>JSON Error:</b>\n%s", safeErr), backMarkup)
	}

	jsonString := string(jsonBytes)

	// Обрезаем, если сообщение слишком длинное для Telegram (лимит ~4096 символов)
	// Оставляем запас под заголовок и теги
	if len(jsonString) > 3800 {
		jsonString = jsonString[:3800] + "\n...[truncated]"
	}

	// Экранируем JSON перед вставкой в HTML
	safeJSON := html.EscapeString(jsonString)

	text := fmt.Sprintf("🔌 <b>Список станков:</b>\n<pre>%s</pre>", safeJSON)

	return c.Edit(text, backMarkup)
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

	// Экранирование
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
	// Экранируем JSON перед вставкой в HTML (даже внутри pre)
	safeMsg := html.EscapeString(prettyMsg)
	return c.Edit(fmt.Sprintf("📨 Result:\n<pre>%s</pre>", safeMsg), backMarkup)
}

// --- Live Mode & Wizard (Existing simplified) ---

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
	return h.cmdHandler.OnStart(c) // Return to main menu
}

// Helper
func prettyPrintJSON(input string) string {
	var temp interface{}
	if err := json.Unmarshal([]byte(input), &temp); err != nil {
		return input
	}
	pretty, _ := json.MarshalIndent(temp, "", "  ")
	return string(pretty)
}
