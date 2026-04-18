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
	if cfg.TgToken == "" {
		log.Println("⚠️ TG_TOKEN не указан, Telegram бот не будет инициализирован")
		return &Bot{}
	}

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

	router.Register(b)

	err = b.SetCommands([]tele.Command{
		{Text: "start", Description: "Главное меню"},
		{Text: "kafka", Description: "Управление Kafka Targets"},
		{Text: "services", Description: "Управление API Services"},
		{Text: "profile", Description: "Профиль пользователя"},
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
	if b.Bot != nil {
		log.Println("🤖 Бот запущен...")
		b.Bot.Start()
	}
}

func (b *Bot) Stop() {
	if b.Bot != nil {
		b.Bot.Stop()
	}
}
