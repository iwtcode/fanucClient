package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucClient/internal/interfaces"
	tele "gopkg.in/telebot.v3"
)

type CommandHandler struct {
	menu       *Menu
	settingsUC interfaces.SettingsUsecase
	controlUC  interfaces.ControlUsecase
}

func NewCommandHandler(
	menu *Menu,
	settingsUC interfaces.SettingsUsecase,
	controlUC interfaces.ControlUsecase,
) *CommandHandler {
	return &CommandHandler{
		menu:       menu,
		settingsUC: settingsUC,
		controlUC:  controlUC,
	}
}

func (h *CommandHandler) OnStart(c tele.Context) error {
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	user := &entities.User{
		ID:        c.Sender().ID,
		FirstName: c.Sender().FirstName,
		UserName:  c.Sender().Username,
		State:     entities.StateIdle,
	}
	h.settingsUC.RegisterUser(user)

	text := "👋 <b>Fanuc Client</b>\n\nГлавное меню."
	if c.Callback() != nil {
		return c.Edit(text, h.menu.BuildMainMenu())
	}
	return c.Send(text, h.menu.ReplyMain, h.menu.BuildMainMenu())
}

func (h *CommandHandler) OnWho(c tele.Context) error {
	u, err := h.settingsUC.GetUser(c.Sender().ID)
	if err != nil {
		return c.Send("Error getting user")
	}
	text := fmt.Sprintf("👤 <b>Profile</b>\nID: %d\nState: %s", u.ID, u.State)

	targets, _ := h.settingsUC.GetTargets(u.ID)
	services, _ := h.settingsUC.GetServices(u.ID)

	text += fmt.Sprintf("\n\n📋 Kafka Targets: %d", len(targets))
	text += fmt.Sprintf("\n🌐 API Services: %d", len(services))

	if c.Callback() != nil {
		return c.Edit(text, h.menu.BuildWhoMenu())
	}
	return c.Send(text, h.menu.BuildWhoMenu())
}

func (h *CommandHandler) OnText(c tele.Context) error {
	userID := c.Sender().ID
	user, err := h.settingsUC.GetUser(userID)
	if err != nil {
		return h.OnStart(c)
	}

	input := strings.TrimSpace(c.Text())

	// Menu Commands
	switch input {
	case h.menu.BtnHome.Text:
		return h.OnStart(c)
	case h.menu.BtnWho.Text:
		return h.OnWho(c)
	case h.menu.BtnTargets.Text:
		return (&CallbackHandler{menu: h.menu, settingsUC: h.settingsUC}).onListTargets(c)
	case h.menu.BtnServices.Text:
		return (&CallbackHandler{menu: h.menu, settingsUC: h.settingsUC}).onListServices(c)
	}

	// FSM
	switch user.State {
	// --- Kafka Wizard ---
	case entities.StateWaitingName:
		h.settingsUC.SetDraftName(userID, input)
		return c.Send("🔌 <b>Шаг 2/4: Broker (IP:PORT)</b>", h.menu.BuildCancel())
	case entities.StateWaitingBroker:
		h.settingsUC.SetDraftBroker(userID, input)
		return c.Send("📂 <b>Шаг 3/4: Topic</b>", h.menu.BuildCancel())
	case entities.StateWaitingTopic:
		h.settingsUC.SetDraftTopic(userID, input)
		return c.Send("🔑 <b>Шаг 4/4: Key (Optional)</b>\nОтправьте '0' или 'no' если не нужен.", h.menu.BuildCancel())
	case entities.StateWaitingKey:
		finalKey := input
		if input == "0" || input == "-" || input == "no" {
			finalKey = ""
		}
		h.settingsUC.SetDraftKeyAndSave(userID, finalKey)
		c.Send("✅ Kafka Target Saved!")
		return (&CallbackHandler{menu: h.menu, settingsUC: h.settingsUC}).onListTargets(c)

	// --- Service Registration Wizard ---
	case entities.StateWaitingSvcName:
		h.settingsUC.SetDraftSvcName(userID, input)
		return c.Send("🔗 <b>Шаг 2/3: Host (IP:PORT)</b>\nВведите адрес сервиса (без http://):", h.menu.BuildCancel())
	case entities.StateWaitingSvcHost:
		h.settingsUC.SetDraftSvcHost(userID, input)
		return c.Send("🔐 <b>Шаг 3/3: API Key</b>\nВведите ключ доступа к сервису:", h.menu.BuildCancel())
	case entities.StateWaitingSvcKey:
		h.settingsUC.SetDraftSvcKeyAndSave(userID, input)
		c.Send("✅ Service Saved!")
		return (&CallbackHandler{menu: h.menu, settingsUC: h.settingsUC}).onListServices(c)

	// --- Machine Connection Wizard (Remote API) ---
	case entities.StateWaitingConnEndpoint:
		// Save IP temporarily in draft field
		h.settingsUC.SetDraftConnIP(userID, input)
		return c.Send("🤖 <b>Шаг 2/2: Series</b>\nВведите серию стойки (0i, 30i, 31i, 32i, 35i). Если не знаете, отправьте 'Unknown'.", h.menu.BuildCancel())

	case entities.StateWaitingConnSeries:
		series := input
		// Get context variables
		svcID := user.ContextSvcID
		ip := user.DraftConnIP

		c.Send("⏳ Creating connection on remote service...")

		// Call UseCase
		_, err := h.controlUC.CreateMachine(context.Background(), svcID, ip, series)
		if err != nil {
			c.Send(fmt.Sprintf("❌ Error creating connection: %v", err))
		} else {
			c.Send("✅ Connection established!")
		}

		h.settingsUC.SetState(userID, entities.StateIdle)
		// Redirect to machine list
		cb := &CallbackHandler{menu: h.menu, settingsUC: h.settingsUC, controlUC: h.controlUC}
		return cb.onListServiceMachines(c, svcID)

	// --- Polling Wizard ---
	case entities.StateWaitingPollInterval:
		interval, err := strconv.Atoi(input)
		if err != nil || interval < 100 {
			return c.Send("⚠️ Please enter a valid number (min 100 ms).")
		}

		svcID := user.ContextSvcID
		machineID := user.ContextMachineID

		c.Send("⏳ Starting polling...")
		err = h.controlUC.StartPolling(context.Background(), svcID, machineID, interval)
		if err != nil {
			c.Send(fmt.Sprintf("❌ Error starting polling: %v", err))
		} else {
			c.Send("✅ Polling started!")
		}

		h.settingsUC.SetState(userID, entities.StateIdle)
		// Redirect to machine view
		cb := &CallbackHandler{menu: h.menu, settingsUC: h.settingsUC, controlUC: h.controlUC}
		return cb.onViewMachine(c, svcID, machineID)

	default:
		return h.OnStart(c)
	}
}
