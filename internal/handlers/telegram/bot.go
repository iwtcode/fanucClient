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
	b.Use(LogMiddleware())

	// Регистрируем хендлеры
	router.Register(b)

	// Устанавливаем команды для меню (удалит старые ping/time)
	err = b.SetCommands([]tele.Command{
		{Text: "start", Description: "Главное меню"},
		{Text: "whoami", Description: "Профиль пользователя"},
	})
	if err != nil {
		log.Printf("⚠️ Не удалось обновить список команд: %v", err)
	}

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
