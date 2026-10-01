package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// pinger is implemented by dependencies /readyz checks (licensing Postgres, LMS MySQL).
type pinger interface {
	Ping(ctx context.Context) error
}

type readinessCheck struct {
	name string
	ping func(ctx context.Context) error
}

const readinessTimeout = 2 * time.Second

// healthMux answers /healthz (process is up) and /readyz (dependencies reachable) before
// the gin router: probes skip auth, CSRF and rate limiting and stay out of the request log.
func healthMux(next http.Handler, checks []readinessCheck) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeHealth(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()
		status := http.StatusOK
		result := map[string]string{}
		for _, check := range checks {
			if err := check.ping(ctx); err != nil {
				// Details go to the log only: the endpoint is reachable without auth.
				log.Printf("readyz: %s: %v", check.name, err)
				result[check.name] = "error"
				status = http.StatusServiceUnavailable
				continue
			}
			result[check.name] = "ok"
		}
		writeHealth(w, status, result)
	})
	mux.Handle("/", next)
	return mux
}

func writeHealth(w http.ResponseWriter, status int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
