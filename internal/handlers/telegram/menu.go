package telegram

import tele "gopkg.in/telebot.v3"

type Menu struct {
	// Главные меню
	ReplyMain  *tele.ReplyMarkup // Нижнее меню
	InlineMain *tele.ReplyMarkup // Инлайн меню под сообщениями

	// Меню настроек (только Inline)
	InlineSettings *tele.ReplyMarkup

	// Кнопки Reply (Главное меню)
	BtnSettingsReply tele.Btn
	BtnLastMsgReply  tele.Btn
	BtnWhoReply      tele.Btn

	// Кнопки Inline (Главное меню)
	BtnSettingsInline tele.Btn
	BtnLastMsgInline  tele.Btn
	BtnWhoInline      tele.Btn

	// Кнопки настроек (Inline)
	BtnSetBroker    tele.Btn
	BtnSetTopic     tele.Btn
	BtnCancelConfig tele.Btn
}

func NewMenu() *Menu {
	// 1. Инициализация разметок
	replyMain := &tele.ReplyMarkup{ResizeKeyboard: true}
	inlineMain := &tele.ReplyMarkup{}
	inlineSettings := &tele.ReplyMarkup{}

	// 2. Создание кнопок Reply
	btnLastMsgReply := replyMain.Text("📩 Last Message")
	btnSettingsReply := replyMain.Text("⚙️ Settings")
	btnWhoReply := replyMain.Text("👤 WhoAmI")

	// 3. Создание кнопок Inline (Main)
	// Unique ID важны для корректной обработки коллбэков
	btnLastMsgInline := inlineMain.Data("📩 Last Message", "last_msg_btn")
	btnSettingsInline := inlineMain.Data("⚙️ Settings", "settings_btn")
	btnWhoInline := inlineMain.Data("👤 WhoAmI", "who_btn")

	// 4. Создание кнопок Inline (Settings)
	btnSetBroker := inlineSettings.Data("🔌 Set Broker", "set_broker")
	btnSetTopic := inlineSettings.Data("📝 Set Topic", "set_topic")
	btnCancel := inlineSettings.Data("🔙 Back", "cancel_config")

	// 5. Компоновка Reply меню
	replyMain.Reply(
		replyMain.Row(btnLastMsgReply),
		replyMain.Row(btnSettingsReply, btnWhoReply),
	)

	// 6. Компоновка Inline Main меню
	inlineMain.Inline(
		inlineMain.Row(btnLastMsgInline),
		inlineMain.Row(btnSettingsInline, btnWhoInline),
	)

	// 7. Компоновка Inline Settings меню
	inlineSettings.Inline(
		inlineSettings.Row(btnSetBroker, btnSetTopic),
		inlineSettings.Row(btnCancel),
	)

	return &Menu{
		ReplyMain:         replyMain,
		InlineMain:        inlineMain,
		InlineSettings:    inlineSettings,
		BtnSettingsReply:  btnSettingsReply,
		BtnLastMsgReply:   btnLastMsgReply,
		BtnWhoReply:       btnWhoReply,
		BtnSettingsInline: btnSettingsInline,
		BtnLastMsgInline:  btnLastMsgInline,
		BtnWhoInline:      btnWhoInline,
		BtnSetBroker:      btnSetBroker,
		BtnSetTopic:       btnSetTopic,
		BtnCancelConfig:   btnCancel,
	}
}
