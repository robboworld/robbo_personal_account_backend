package app

import (
	"github.com/skinnykaen/robbo_student_personal_account.git/app/modules"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/config"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/docker_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/logger"
	"github.com/skinnykaen/robbo_student_personal_account.git/server"
	"go.uber.org/fx"
	"log"
)

func InvokeWith(options ...fx.Option) *fx.App {
	if err := config.Init(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	var di = []fx.Option{
		fx.Provide(logger.NewLogger),
		fx.Provide(db_client.NewPostgresClient),
		fx.Provide(modules.SetupGateway),
		fx.Provide(modules.SetupPortalModule),
		fx.Provide(modules.SetupUseCase),
		fx.Provide(modules.SetupDelegate),
		fx.Provide(modules.SetupHandler),
		fx.Provide(modules.SetupGraphQLModule),
		fx.Invoke(modules.StartPortalOutboxWorker),
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

func TestInvokeWith(options ...fx.Option) (*fx.App, func()) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	var cleanerContainer func()
	var di = []fx.Option{
		fx.Provide(docker_client.NewTestDockerClient),
		fx.Provide(logger.NewLogger),
		fx.Provide(db_client.NewTestPostgresClient),
		fx.Provide(modules.SetupGateway),
		fx.Provide(modules.SetupUseCase),
		fx.Provide(modules.SetupDelegate),
		fx.Provide(modules.SetupHandler),
		fx.Provide(modules.SetupGraphQLModule),
		fx.Populate(&cleanerContainer),
	}
	for _, option := range options {
		di = append(di, option)
	}
	return fx.New(di...), cleanerContainer
}

func TestApp() (app *fx.App, cleanerContainer func()) {
	app, cleanerContainer = TestInvokeWith(
		fx.Invoke(server.NewServer),
	)
	return
}
