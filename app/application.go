package app

import (
	"github.com/skinnykaen/robbo_student_personal_account.git/app/modules"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/config"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/logger"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/oidc"
	"github.com/skinnykaen/robbo_student_personal_account.git/server"
	"go.uber.org/fx"
	"log"
	"time"
)

// StopTimeout covers HTTP draining (server.shutdownTimeout) plus closing pools.
const StopTimeout = 28 * time.Second

func InvokeWith(options ...fx.Option) *fx.App {
	if err := config.Init(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	if err := config.ValidateSecrets(); err != nil {
		log.Fatalf("config: %s", err.Error())
	}
	if err := oidc.InitSharedStoreFromConfig(); err != nil {
		log.Printf("[oidc] PKCE store fallback to memory: %v", err)
	}
	var di = []fx.Option{
		// Stop hooks run in reverse: the HTTP server drains first, LMS pools close last.
		fx.StopTimeout(StopTimeout),
		fx.Invoke(func(lc fx.Lifecycle) { lc.Append(fx.Hook{OnStop: lmsdb.CloseAll}) }),
		fx.Provide(logger.NewLogger),
		fx.Provide(db_client.NewPostgresClient),
		fx.Provide(modules.SetupGateway),
		fx.Provide(modules.SetupPortalModule),
		fx.Provide(modules.SetupUseCase),
		fx.Provide(modules.SetupDelegate),
		fx.Provide(modules.SetupUserSearchService),
		fx.Provide(modules.SetupHandler),
		fx.Provide(modules.SetupGraphQLModule),
		fx.Invoke(modules.StartPortalOutboxWorker),
		fx.Invoke(modules.StartUserSearchSync),
		fx.Invoke(modules.StartBanExpiryWorker),
	}
	for _, option := range options {
		di = append(di, option)
	}
	return fx.New(di...)
}

func RunApp() {
	InvokeWith(
		fx.Invoke(server.NewServer),
	).Run()
}
