//go:build integration

// Needs Docker (dockertest Postgres); run with: go test -tags integration ./tests/...
package integration_tests

import (
	"context"
	"github.com/skinnykaen/robbo_student_personal_account.git/app/apptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	app, cleanerContainer := apptest.TestApp()
	ctx := context.Background()
	app.Start(ctx)
	code := m.Run()
	app.Stop(ctx)
	cleanerContainer()
	os.Exit(code)
}
