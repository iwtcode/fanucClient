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
		// Trigger callback logic for list
		return (&CallbackHandler{menu: h.menu, settingsUC: h.settingsUC}).onListTargets(c)
	case h.menu.BtnServices.Text:
		return (&CallbackHandler{menu: h.menu, settingsUC: h.settingsUC}).onListServices(c)
	}

	// FSM
	switch user.State {
	// Kafka Wizard
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

	// Service Wizard
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

	default:
		return h.OnStart(c)
	}
}
