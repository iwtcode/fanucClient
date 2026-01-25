package telegram

import (
	"fmt"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucService"
	tele "gopkg.in/telebot.v3"
)

type Menu struct {
	// Reply Main
	ReplyMain   *tele.ReplyMarkup
	BtnTargets  tele.Btn
	BtnServices tele.Btn
	BtnWho      tele.Btn
	BtnHome     tele.Btn

	// Inline Main
	InlineMain    *tele.ReplyMarkup
	BtnHomeInline tele.Btn

	// --- Kafka Targets ---
	BtnAddTarget    tele.Btn
	BtnBackTargets  tele.Btn
	BtnCancelWizard tele.Btn

	// --- Fanuc Services ---
	BtnAddService tele.Btn
	BtnBackSvc    tele.Btn
	BtnDeleteSvc  tele.Btn

	// --- Machines Control ---
	BtnAddConnection tele.Btn
}

func NewMenu() *Menu {
	replyMain := &tele.ReplyMarkup{ResizeKeyboard: true}
	inlineMain := &tele.ReplyMarkup{}

	// Reply Buttons
	btnTargets := replyMain.Text("📋 Kafka Targets")
	btnServices := replyMain.Text("🌐 API Services")
	btnWho := replyMain.Text("👤 Профиль")
	btnHome := replyMain.Text("🏠 В начало")

	replyMain.Reply(
		replyMain.Row(btnTargets, btnServices),
		replyMain.Row(btnWho, btnHome),
	)

	// Inline Buttons (Global)
	btnHomeInline := inlineMain.Data("🏠 В начало", "home")
	btnCancelWizard := inlineMain.Data("🚫 Отмена", "cancel_wizard")

	// Kafka
	btnAddTarget := inlineMain.Data("➕ Kafka Target", "add_target")
	btnBackTargets := inlineMain.Data("🔙 К списку Kafka", "targets_list")

	// Services
	btnAddService := inlineMain.Data("➕ API Service", "add_service")
	btnBackSvc := inlineMain.Data("🔙 К списку Сервисов", "services_list")
	btnDeleteSvc := inlineMain.Data("🗑 Удалить сервис", "del_service")

	// Machines
	btnAddConnection := inlineMain.Data("➕ Подключить станок", "add_conn")

	return &Menu{
		ReplyMain:     replyMain,
		InlineMain:    inlineMain,
		BtnTargets:    btnTargets,
		BtnServices:   btnServices,
		BtnWho:        btnWho,
		BtnHome:       btnHome,
		BtnHomeInline: btnHomeInline,

		// Kafka
		BtnAddTarget:    btnAddTarget,
		BtnBackTargets:  btnBackTargets,
		BtnCancelWizard: btnCancelWizard,

		// Services
		BtnAddService:    btnAddService,
		BtnBackSvc:       btnBackSvc,
		BtnDeleteSvc:     btnDeleteSvc,
		BtnAddConnection: btnAddConnection,
	}
}

func (m *Menu) BuildMainMenu() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📋 Kafka Targets", "targets_list")),
		markup.Row(markup.Data("🌐 API Services", "services_list")),
		markup.Row(markup.Data("👤 Профиль", "who_btn")),
	)
	return markup
}

func (m *Menu) BuildWhoMenu() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(m.BtnHomeInline),
	)
	return markup
}

// --- Kafka Menus ---

func (m *Menu) BuildTargetsList(targets []entities.MonitoringTarget) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, t := range targets {
		btn := markup.Data(fmt.Sprintf("📋 %s", t.Name), fmt.Sprintf("view_target:%d", t.ID))
		rows = append(rows, markup.Row(btn))
	}
	rows = append(rows, markup.Row(m.BtnAddTarget))
	rows = append(rows, markup.Row(m.BtnHomeInline))
	markup.Inline(rows...)
	return markup
}

func (m *Menu) BuildTargetView(t entities.MonitoringTarget) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	// 1. Entry Points List
	var entryRows []tele.Row

	// Default (No Key) entry point
	// keyID = 0 is reserved for "No Key"
	btnDefault := markup.Data("📂 Default (No Key)", fmt.Sprintf("view_key:%d:0", t.ID))
	entryRows = append(entryRows, markup.Row(btnDefault))

	// User defined keys
	for _, k := range t.Keys {
		btnKey := markup.Data(fmt.Sprintf("🔑 %s", k.Key), fmt.Sprintf("view_key:%d:%d", t.ID, k.ID))
		entryRows = append(entryRows, markup.Row(btnKey))
	}

	// 2. Management
	btnAddKey := markup.Data("➕ Добавить ключ", fmt.Sprintf("add_key_start:%d", t.ID))
	btnDelTarget := markup.Data("🗑 Удалить Target", fmt.Sprintf("del_target:%d", t.ID))

	entryRows = append(entryRows, markup.Row(btnAddKey))
	entryRows = append(entryRows, markup.Row(btnDelTarget))
	entryRows = append(entryRows, markup.Row(m.BtnBackTargets))

	markup.Inline(entryRows...)
	return markup
}

func (m *Menu) BuildKeyView(targetID, keyID uint) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	btnMsg := markup.Data("📨 Check Msg", fmt.Sprintf("check_msg:%d:%d", targetID, keyID))
	btnLive := markup.Data("🔴 Live Mode", fmt.Sprintf("live_mode:%d:%d", targetID, keyID))
	btnBack := markup.Data("🔙 К Target", fmt.Sprintf("view_target:%d", targetID))

	// Control rows
	rows := []tele.Row{
		markup.Row(btnMsg, btnLive),
	}

	// Delete button only for real keys (ID > 0)
	if keyID > 0 {
		btnDelKey := markup.Data("🗑 Удалить ключ", fmt.Sprintf("del_key:%d:%d", targetID, keyID))
		rows = append(rows, markup.Row(btnDelKey))
	}

	rows = append(rows, markup.Row(btnBack))
	markup.Inline(rows...)
	return markup
}

func (m *Menu) BuildLiveView(targetID, keyID uint) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	btnStop := markup.Data("⏹ Стоп", fmt.Sprintf("stop_live:%d:%d", targetID, keyID))
	markup.Inline(markup.Row(btnStop))
	return markup
}

// --- Services Menus ---

func (m *Menu) BuildServicesList(services []entities.FanucService) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, s := range services {
		btn := markup.Data(fmt.Sprintf("🌐 %s", s.Name), fmt.Sprintf("view_service:%d", s.ID))
		rows = append(rows, markup.Row(btn))
	}
	rows = append(rows, markup.Row(m.BtnAddService))
	rows = append(rows, markup.Row(m.BtnHomeInline))
	markup.Inline(rows...)
	return markup
}

func (m *Menu) BuildServiceView(svcID uint, machines []fanucService.MachineDTO) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	// 1. Machine List
	for _, mach := range machines {
		statusIcon := "🟢"
		if mach.Status != "connected" {
			statusIcon = "🔴"
		} else if mach.Mode == "polling" {
			statusIcon = "🔄"
		}
		btn := markup.Data(fmt.Sprintf("%s %s (%s)", statusIcon, mach.Endpoint, mach.Model),
			fmt.Sprintf("vm:%d:%s", svcID, mach.ID))
		rows = append(rows, markup.Row(btn))
	}

	// 2. Service Management
	btnAdd := markup.Data("➕ Подключить станок", fmt.Sprintf("add_conn:%d", svcID))
	btnDel := markup.Data("🗑 Удалить сервис", fmt.Sprintf("del_service:%d", svcID))

	rows = append(rows, markup.Row(btnAdd))
	rows = append(rows, markup.Row(btnDel))
	rows = append(rows, markup.Row(m.BtnBackSvc))

	markup.Inline(rows...)
	return markup
}

// --- Machine Menus ---

func (m *Menu) BuildMachineView(svcID uint, machine fanucService.MachineDTO) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	var btnPoll tele.Btn
	if machine.Mode == "polling" {
		btnPoll = markup.Data("⏹ Остановить опрос", fmt.Sprintf("stp:%d:%s", svcID, machine.ID))
	} else {
		btnPoll = markup.Data("▶ Запустить опрос", fmt.Sprintf("sp:%d:%s", svcID, machine.ID))
	}

	btnProg := markup.Data("📄 Скачать программу", fmt.Sprintf("gp:%d:%s", svcID, machine.ID))
	btnDel := markup.Data("🗑 Удалить", fmt.Sprintf("dc:%d:%s", svcID, machine.ID))
	// Кнопка назад теперь ведет на просмотр сервиса (список станков), а не на старый промежуточный список
	btnBack := markup.Data("🔙 К сервису", fmt.Sprintf("view_service:%d", svcID))

	markup.Inline(
		markup.Row(btnPoll),
		markup.Row(btnProg),
		markup.Row(btnDel),
		markup.Row(btnBack),
	)
	return markup
}

func (m *Menu) BuildCancel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(m.BtnCancelWizard))
	return markup
}
