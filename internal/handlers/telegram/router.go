package telegram

import (
	tele "gopkg.in/telebot.v3"
)

type Router struct {
	menu      *Menu
	commands  *CommandHandler
	callbacks *CallbackHandler
}

func NewRouter(menu *Menu, cmd *CommandHandler, cb *CallbackHandler) *Router {
	return &Router{
		menu:      menu,
		commands:  cmd,
		callbacks: cb,
	}
}

func (r *Router) Register(b *tele.Bot) {
	// --- Commands ---
	b.Handle("/start", r.commands.OnStart)
	b.Handle("/whoami", r.commands.OnWho) // Добавили обработчик для команды из меню

	// --- Reply Keyboard Buttons ---
	b.Handle(&r.menu.BtnTargets, r.callbacks.onListTargets)
	b.Handle(&r.menu.BtnWho, r.commands.OnWho)
	b.Handle(&r.menu.BtnHome, r.commands.OnStart)

	// --- Callback Queries (Inline Buttons) ---
	b.Handle(tele.OnCallback, r.callbacks.OnCallback)

	// --- Text Input (FSM) ---
	b.Handle(tele.OnText, r.commands.OnText)
}
