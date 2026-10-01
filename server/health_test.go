package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthMux(t *testing.T) {
	appHit := false
	app := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { appHit = true })
	down := errors.New("dial tcp: connection refused")
	h := healthMux(app, []readinessCheck{
		{name: "licensing_db", ping: func(context.Context) error { return nil }},
		{name: "lms_mysql", ping: func(context.Context) error { return down }},
	})

	serve := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}

	if w := serve("/healthz"); w.Code != http.StatusOK {
		t.Fatalf("/healthz status=%d", w.Code)
	}
	w := serve("/readyz")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("/readyz status=%d want 503", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"lms_mysql":"error"`) || !strings.Contains(body, `"licensing_db":"ok"`) {
		t.Fatalf("unexpected body %s", body)
	}
	if strings.Contains(body, "refused") {
		t.Fatalf("error details leaked: %s", body)
	}
	if serve("/query"); !appHit {
		t.Fatal("other paths must reach the app handler")
	}
}
