package telegram

import (
	"fmt"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
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
	BtnCheckMsg     tele.Btn
	BtnLiveMode     tele.Btn
	BtnDelete       tele.Btn
	BtnStopLive     tele.Btn

	// --- Fanuc Services ---
	BtnAddService  tele.Btn
	BtnBackSvc     tele.Btn
	BtnDeleteSvc   tele.Btn
	BtnSvcMachines tele.Btn // Список подключений на сервисе
}

func NewMenu() *Menu {
	replyMain := &tele.ReplyMarkup{ResizeKeyboard: true}
	inlineMain := &tele.ReplyMarkup{}

	// Reply Buttons
	btnTargets := replyMain.Text("📋 Kafka Reader")
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
	btnCheckMsg := inlineMain.Data("📨 Последнее сообщение", "check_msg")
	btnLiveMode := inlineMain.Data("🔴 Live Mode", "live_mode")
	btnDelete := inlineMain.Data("🗑 Удалить", "del_target")
	btnStopLive := inlineMain.Data("⏹ Стоп", "stop_live")

	// Services
	btnAddService := inlineMain.Data("➕ API Service", "add_service")
	btnBackSvc := inlineMain.Data("🔙 К списку Сервисов", "services_list")
	btnDeleteSvc := inlineMain.Data("🗑 Удалить сервис", "del_service")
	btnSvcMachines := inlineMain.Data("🔌 Список станков", "svc_machines")

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
		BtnCheckMsg:     btnCheckMsg,
		BtnLiveMode:     btnLiveMode,
		BtnDelete:       btnDelete,
		BtnStopLive:     btnStopLive,

		// Services
		BtnAddService:  btnAddService,
		BtnBackSvc:     btnBackSvc,
		BtnDeleteSvc:   btnDeleteSvc,
		BtnSvcMachines: btnSvcMachines,
	}
}

func (m *Menu) BuildMainMenu() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📋 Kafka Reader", "targets_list")),
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
		btn := markup.Data(fmt.Sprintf("🔩 %s", t.Name), fmt.Sprintf("view_target:%d", t.ID))
		rows = append(rows, markup.Row(btn))
	}
	rows = append(rows, markup.Row(m.BtnAddTarget))
	rows = append(rows, markup.Row(m.BtnHomeInline))
	markup.Inline(rows...)
	return markup
}

func (m *Menu) BuildTargetView(targetID uint) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	btnCheck := markup.Data("📨 Msg", fmt.Sprintf("check_msg:%d", targetID))
	btnLive := markup.Data("🔴 Live", fmt.Sprintf("live_mode:%d", targetID))
	btnDel := markup.Data("🗑 Del", fmt.Sprintf("del_target:%d", targetID))

	markup.Inline(
		markup.Row(btnCheck, btnLive),
		markup.Row(btnDel),
		markup.Row(m.BtnBackTargets),
	)
	return markup
}

func (m *Menu) BuildLiveView(targetID uint) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	btnStop := markup.Data("⏹ Стоп", fmt.Sprintf("stop_live:%d", targetID))
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

func (m *Menu) BuildServiceView(svcID uint) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	btnList := markup.Data("🔌 Список станков", fmt.Sprintf("svc_machines:%d", svcID))
	btnDel := markup.Data("🗑 Удалить сервис", fmt.Sprintf("del_service:%d", svcID))

	markup.Inline(
		markup.Row(btnList),
		markup.Row(btnDel),
		markup.Row(m.BtnBackSvc),
	)
	return markup
}

func (m *Menu) BuildBackToService(svcID uint) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	btnBack := markup.Data("🔙 Назад к сервису", fmt.Sprintf("view_service:%d", svcID))
	markup.Inline(markup.Row(btnBack))
	return markup
}

func (m *Menu) BuildCancel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(m.BtnCancelWizard))
	return markup
}
