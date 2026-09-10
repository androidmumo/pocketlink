package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type Readiness interface{ Ready(context.Context) error }
type API struct {
	store    Readiness
	draining atomic.Bool
	version  string
}

func New(store Readiness, version string) *API { return &API{store: store, version: version} }
func (a *API) Drain()                          { a.draining.Store(true) }
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health/live", "/health/ready", "/api/v1/capabilities":
	default:
		write(w, 404, map[string]string{"error": "not_found"})
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		write(w, 405, map[string]string{"error": "method_not_allowed"})
		return
	}
	switch r.URL.Path {
	case "/health/live":
		write(w, 200, map[string]string{"status": "ok"})
	case "/health/ready":
		if a.draining.Load() {
			write(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if a.store.Ready(ctx) != nil {
			write(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	case "/api/v1/capabilities":
		write(w, 200, map[string]any{"protocol_versions": []int{1}, "server_version": a.version, "stage": "foundation", "enabled_features": []string{}, "limits": map[string]int{"control_bytes": 8192, "text_utf8_bytes": 2048, "text_codepoints": 200, "realtime_plaintext_bytes": 1100}})
	}
}
