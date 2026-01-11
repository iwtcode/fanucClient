package telegram

import (
	"fmt"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	tele "gopkg.in/telebot.v3"
)

type Menu struct {
	// Reply Main
	ReplyMain  *tele.ReplyMarkup
	BtnTargets tele.Btn
	BtnWho     tele.Btn
	BtnHome    tele.Btn

	// Inline Main (Navigation)
	InlineMain    *tele.ReplyMarkup
	BtnHomeInline tele.Btn

	// Inline Targets List
	BtnAddTarget tele.Btn
	BtnBack      tele.Btn

	// Inline Wizard
	BtnCancelWizard tele.Btn

	// Inline Target Actions
	BtnCheckMsg tele.Btn
	BtnDelete   tele.Btn
}

func NewMenu() *Menu {
	replyMain := &tele.ReplyMarkup{ResizeKeyboard: true}
	inlineMain := &tele.ReplyMarkup{}

	// Reply Buttons
	btnTargets := replyMain.Text("📋 Targets")
	btnWho := replyMain.Text("👤 WhoAmI")
	btnHome := replyMain.Text("🏠 Главная")

	replyMain.Reply(
		replyMain.Row(btnTargets, btnWho),
		replyMain.Row(btnHome),
	)

	// Inline Buttons
	// Unique ID (второй аргумент) важен для статических кнопок, которые мы проверяем через switch unique
	btnHomeInline := inlineMain.Data("🏠 Домой", "home")

	btnAddTarget := inlineMain.Data("➕ Add New", "add_target")
	btnBack := inlineMain.Data("🔙 Back to List", "back_to_list")
	btnCancelWizard := inlineMain.Data("🚫 Cancel", "cancel_wizard")

	btnCheckMsg := inlineMain.Data("📨 Check Message", "check_msg")
	btnDelete := inlineMain.Data("🗑 Delete", "del_target")

	return &Menu{
		ReplyMain:       replyMain,
		InlineMain:      inlineMain,
		BtnTargets:      btnTargets,
		BtnWho:          btnWho,
		BtnHome:         btnHome,
		BtnHomeInline:   btnHomeInline,
		BtnAddTarget:    btnAddTarget,
		BtnBack:         btnBack,
		BtnCancelWizard: btnCancelWizard,
		BtnCheckMsg:     btnCheckMsg,
		BtnDelete:       btnDelete,
	}
}

// BuildMainMenu создает инлайн меню для команды /start
func (m *Menu) BuildMainMenu() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	// ИСПРАВЛЕНИЕ: Используем 'targets_list' вместо 'back_to_list' для логирования
	btnTargets := markup.Data("📋 Управление целями", "targets_list")
	btnWho := markup.Data("👤 Информация", "who_btn")

	markup.Inline(
		markup.Row(btnTargets),
		markup.Row(btnWho),
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

	btnCheck := markup.Data("📨 Check Message", fmt.Sprintf("check_msg:%d", targetID))
	btnDel := markup.Data("🗑 Delete", fmt.Sprintf("del_target:%d", targetID))

	markup.Inline(
		markup.Row(btnCheck),
		markup.Row(btnDel),
		markup.Row(m.BtnBack),
		markup.Row(m.BtnHomeInline),
	)

	return markup
}

func (m *Menu) BuildCancel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(m.BtnCancelWizard))
	return markup
}
