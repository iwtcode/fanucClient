package app

import (
	"context"
	"log"

	"github.com/iwtcode/fanucClient"
	"github.com/iwtcode/fanucClient/internal/handlers/telegram"
	"github.com/iwtcode/fanucClient/internal/repository"
	"github.com/iwtcode/fanucClient/internal/services"
	"github.com/iwtcode/fanucClient/internal/usecases"
	"go.uber.org/fx"
)

func New() *fx.App {
	return fx.New(
		fx.Provide(
			// Config
			fanucClient.LoadConfig,

			// Repository
			repository.NewPostgresRepository,
			repository.NewUserRepository,

			// Services
			services.NewKafkaService,
			services.NewFanucApiService, // Новый сервис API

			// Usecases
			usecases.NewSettingsUsecase,
			usecases.NewMonitoringUsecase,
			usecases.NewControlUsecase, // Новый юзкейс

			// Telegram Components
			telegram.NewMenu,
			telegram.NewCommandHandler,
			telegram.NewCallbackHandler,
			telegram.NewRouter,
			telegram.NewBot,
		),
		fx.Invoke(
			startBot,
		),
	)
}

func startBot(lifecycle fx.Lifecycle, bot *telegram.Bot) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Println("🔥 Бот запускается...")
				bot.Start()
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			bot.Stop()
			return nil
		},
	})
}
