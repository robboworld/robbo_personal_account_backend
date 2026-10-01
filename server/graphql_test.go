package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/graph/generated"
	"github.com/spf13/viper"
)

func introspect(t *testing.T) string {
	t.Helper()
	srv := newGraphQLServer(generated.NewExecutableSchema(generated.Config{}))
	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(`{"query":"{ __schema { queryType { name } } }"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w.Body.String()
}

func TestGraphQLIntrospectionDisabledByDefault(t *testing.T) {
	viper.Set("graphql.introspection", false)
	if body := introspect(t); !strings.Contains(body, "errors") || strings.Contains(body, `"queryType"`) {
		t.Fatalf("introspection allowed when disabled: %s", body)
	}
}

func TestGraphQLIntrospectionOptIn(t *testing.T) {
	viper.Set("graphql.introspection", true)
	t.Cleanup(func() { viper.Set("graphql.introspection", false) })
	if body := introspect(t); !strings.Contains(body, `"queryType"`) {
		t.Fatalf("introspection blocked when enabled: %s", body)
	}
}
