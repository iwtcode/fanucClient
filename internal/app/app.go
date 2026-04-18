package app

import (
	"context"

	"github.com/iwtcode/fanucClient"
	"github.com/iwtcode/fanucClient/internal/handlers/telegram"
	"github.com/iwtcode/fanucClient/internal/handlers/web"
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
			services.NewFanucApiService,

			// Usecases
			usecases.NewSettingsUsecase,
			usecases.NewMonitoringUsecase,
			usecases.NewControlUsecase,

			// Telegram Components
			telegram.NewMenu,
			telegram.NewCommandHandler,
			telegram.NewCallbackHandler,
			telegram.NewRouter,
			telegram.NewBot,

			// Web Components
			web.NewServer,
		),
		fx.Invoke(
			startServices,
		),
	)
}

func startServices(lifecycle fx.Lifecycle, cfg *fanucClient.Config, bot *telegram.Bot, webSrv *web.Server) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if cfg.TgToken != "" && bot.Bot != nil {
				go bot.Start()
			}
			if cfg.AppPort != "" {
				go webSrv.Start(cfg.AppPort)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cfg.TgToken != "" && bot.Bot != nil {
				bot.Stop()
			}
			if cfg.AppPort != "" {
				webSrv.Stop(ctx)
			}
			return nil
		},
	})
}
