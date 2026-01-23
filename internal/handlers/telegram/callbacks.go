package telegram

import (
	"context"
	"encoding/json"
	"fmt"
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
	cmdHandler   *CommandHandler

	// liveSessions хранит функции отмены контекста для активных Live-сессий пользователей.
	// Ключ: int64 (UserID), Значение: context.CancelFunc
	liveSessions sync.Map
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
	case "live_mode":
		return h.onLiveModeStart(c, targetID)
	case "stop_live":
		return h.onStopLive(c, targetID)
	case "del_target":
		return h.onDeleteTarget(c, targetID)
	}
	return nil
}

// --- Specific Handlers ---

func (h *CallbackHandler) onListTargets(c tele.Context) error {
	h.stopUserLiveSession(c.Sender().ID)

	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	targets, err := h.settingsUC.GetTargets(c.Sender().ID)
	if err != nil {
		return c.Send("Error fetching targets: " + err.Error())
	}

	text := fmt.Sprintf("📋 <b>Ваши подключения (%d)</b>\n\nВыберите подключение или создайте новое", len(targets))
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
	h.stopUserLiveSession(c.Sender().ID)
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

// --- Live Mode Handlers ---

func (h *CallbackHandler) onLiveModeStart(c tele.Context, targetID uint) error {
	userID := c.Sender().ID

	h.stopUserLiveSession(userID)

	ctx, cancel := context.WithCancel(context.Background())
	h.liveSessions.Store(userID, cancel)

	target, err := h.settingsUC.GetTargetByID(targetID)
	if err != nil {
		return c.Send("❌ Target not found")
	}

	// Показываем сообщение о загрузке, чтобы пользователь видел реакцию сразу
	initialText := fmt.Sprintf("🔴 <b>LIVE MODE: %s</b>\n\n⏳ Подключение...", target.Name)
	markup := h.menu.BuildLiveView(targetID)

	if err := c.Edit(initialText, markup); err != nil {
		return err
	}

	go h.runLiveUpdateLoop(ctx, c, targetID, target.Name)

	return nil
}

func (h *CallbackHandler) onStopLive(c tele.Context, targetID uint) error {
	h.stopUserLiveSession(c.Sender().ID)
	return h.onViewTarget(c, targetID)
}

func (h *CallbackHandler) runLiveUpdateLoop(ctx context.Context, c tele.Context, targetID uint, targetName string) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var lastContent string

	// Определяем функцию обновления, чтобы вызвать её сразу и в цикле
	update := func() {
		fetchCtx, cancelFetch := context.WithTimeout(context.Background(), 5*time.Second)
		msgRaw, err := h.monitoringUC.FetchLastKafkaMessage(fetchCtx, targetID)
		cancelFetch()

		// Если контекст уже отменен (пользователь вышел пока шел запрос), не обновляем
		if ctx.Err() != nil {
			return
		}

		var displayText string
		timestamp := time.Now().Format("15:04:05")

		if err != nil {
			displayText = fmt.Sprintf("🔴 <b>LIVE MODE: %s</b>\nUpdated: %s\n\n❌ <b>Error:</b> %s", targetName, timestamp, err.Error())
		} else {
			prettyMsg := prettyPrintJSON(msgRaw)
			if len(prettyMsg) > 3500 {
				prettyMsg = prettyMsg[:3500] + "\n...[truncated]"
			}
			displayText = fmt.Sprintf("🔴 <b>LIVE MODE: %s</b>\nUpdated: %s\n\n<pre>%s</pre>", targetName, timestamp, prettyMsg)
		}

		// Избегаем ошибки "message is not modified"
		if displayText == lastContent {
			return
		}

		markup := h.menu.BuildLiveView(targetID)
		if err := c.Edit(displayText, markup); err != nil {
			if strings.Contains(err.Error(), "message to edit not found") || strings.Contains(err.Error(), "chat not found") {
				// Если сообщение удалено или чат недоступен — останавливаем цикл
				h.stopUserLiveSession(c.Sender().ID)
			} else {
				fmt.Printf("Live edit warning (user %d): %v\n", c.Sender().ID, err)
			}
		} else {
			lastContent = displayText
		}
	}

	// 1. Вызываем обновление СРАЗУ (убирает задержку в 3 секунды)
	update()

	// 2. Запускаем цикл
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
		cancelFunc := val.(context.CancelFunc)
		cancelFunc()
		h.liveSessions.Delete(userID)
	}
}

// --- Wizard Handlers ---

func (h *CallbackHandler) onAddTargetStart(c tele.Context) error {
	h.stopUserLiveSession(c.Sender().ID)
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
