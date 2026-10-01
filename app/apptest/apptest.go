// Package apptest wires the application against a dockertest Postgres for integration tests.
// It lives apart from package app so the server binary does not link dockertest.
package apptest

import (
	"log"

	"github.com/skinnykaen/robbo_student_personal_account.git/app/modules"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/config"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/docker_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/logger"
	"github.com/skinnykaen/robbo_student_personal_account.git/server"
	"go.uber.org/fx"
)

func TestInvokeWith(options ...fx.Option) (*fx.App, func()) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	var cleanerContainer func()
	var di = []fx.Option{
		fx.Provide(docker_client.NewTestDockerClient),
		fx.Provide(logger.NewLogger),
		fx.Provide(docker_client.NewTestPostgresClient),
		fx.Provide(modules.SetupGateway),
		fx.Provide(modules.SetupUserSearchService),
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
