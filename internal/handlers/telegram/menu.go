package telegram

import (
	"fmt"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	tele "gopkg.in/telebot.v3"
)

type Menu struct {
	// Reply Main (Нижняя клавиатура)
	ReplyMain  *tele.ReplyMarkup
	BtnTargets tele.Btn
	BtnWho     tele.Btn
	BtnHome    tele.Btn

	// Inline Main (Меню в сообщении)
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

	// Reply Buttons (Названия синхронизированы с Inline)
	btnTargets := replyMain.Text("📋 Управление подключениями")
	btnWho := replyMain.Text("👤 Профиль")
	btnHome := replyMain.Text("🏠 В начало")

	replyMain.Reply(
		replyMain.Row(btnTargets, btnWho),
		replyMain.Row(btnHome),
	)

	// Inline Buttons
	btnHomeInline := inlineMain.Data("🏠 В начало", "home")

	btnAddTarget := inlineMain.Data("➕ Добавить", "add_target")
	btnBack := inlineMain.Data("🔙 Назад", "back_to_list")
	btnCancelWizard := inlineMain.Data("🚫 Отмена", "cancel_wizard")

	btnCheckMsg := inlineMain.Data("📨 Проверить", "check_msg")
	btnDelete := inlineMain.Data("🗑 Удалить", "del_target")

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

	// Используем те же названия, что и в Reply
	btnTargets := markup.Data("📋 Управление подключениями", "targets_list")
	btnWho := markup.Data("👤 Профиль", "who_btn")

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
		// Отображаем имя цели в кнопке
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

	btnCheck := markup.Data("📨 Последнее сообщение", fmt.Sprintf("check_msg:%d", targetID))
	btnDel := markup.Data("🗑 Удалить подключение", fmt.Sprintf("del_target:%d", targetID))

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
