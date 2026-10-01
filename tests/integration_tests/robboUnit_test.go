//go:build integration

// Needs Docker (dockertest Postgres); run with: go test -tags integration ./tests/...
package integration_tests

import (
	"context"
	"github.com/go-playground/assert/v2"
	"github.com/hasura/go-graphql-client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/tests/factory"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"testing"
)

func TestCreateRobboUnit(t *testing.T) {
	gqlClient := graphql.NewClient("http://localhost:8001/query", nil).WithRequestModifier(func(request *http.Request) {
		request.Header.Add("Authorization", "Bearer "+viper.GetString("auth.token_super_admin"))
		request.Header.Add("Content-Type", "application/json")
	})
	var m struct {
		CreateRobboUnit struct {
			models.RobboUnitHTTP `graphql:"... on RobboUnitHttp"`
			models.Error         `graphql:"... on Error"`
		} `graphql:"CreateRobboUnit(input: $NewRobboUnit)"`
	}

	testData := factory.DataCreateRobboUnit()

	log.Println("Ok")
	expect := testData[0].ExpectedError
	correct := gqlClient.Mutate(context.Background(), &m, testData[0].Variables)
	assert.Equal(t, expect, correct)
}
