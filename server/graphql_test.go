package server

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func postQuery(t *testing.T, query string) string {
	t.Helper()
	srv := newGraphQLServer(generated.NewExecutableSchema(generated.Config{}))
	body, _ := json.Marshal(map[string]string{"query": query})
	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w.Body.String()
}

func TestGraphQLComplexityLimit(t *testing.T) {
	var aliased strings.Builder
	aliased.WriteString("{")
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&aliased, " a%d: GetUser { __typename }", i)
	}
	aliased.WriteString(" }")
	if body := postQuery(t, aliased.String()); !strings.Contains(body, "COMPLEXITY_LIMIT_EXCEEDED") {
		t.Fatalf("300 aliased GetUser accepted: %.300s", body)
	}
	if body := postQuery(t, "{ __typename }"); !strings.Contains(body, `"Query"`) {
		t.Fatalf("small query rejected: %s", body)
	}
}
