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
	// Commands
	b.Handle("/start", r.commands.OnStart)
	b.Handle("/profile", r.commands.OnWho)

	// Reply Keyboard
	b.Handle(&r.menu.BtnTargets, r.callbacks.onListTargets)
	b.Handle(&r.menu.BtnServices, r.callbacks.onListServices)
	b.Handle(&r.menu.BtnWho, r.commands.OnWho)
	b.Handle(&r.menu.BtnHome, r.commands.OnStart)

	// Callbacks & Text
	b.Handle(tele.OnCallback, r.callbacks.OnCallback)
	b.Handle(tele.OnText, r.commands.OnText)
}
