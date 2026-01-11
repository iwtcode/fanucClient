package telegram

import (
	"log"
	"time"

	"github.com/iwtcode/fanucClient"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

type Bot struct {
	Bot    *tele.Bot
	Router *Router
}

func NewBot(cfg *fanucClient.Config, router *Router) *Bot {
	pref := tele.Settings{
		Token:     cfg.TgToken,
		Poller:    &tele.LongPoller{Timeout: 10 * time.Second},
		ParseMode: tele.ModeHTML,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	b.Use(middleware.Recover())

	// ЗАМЕНА: Вместо middleware.Logger() используем свой
	b.Use(LogMiddleware())

	// Регистрируем все хендлеры через роутер
	router.Register(b)

	return &Bot{
		Bot:    b,
		Router: router,
	}
}

func (b *Bot) Start() {
	log.Println("🤖 Bot is running...")
	b.Bot.Start()
}

func (b *Bot) Stop() {
	b.Bot.Stop()
}
