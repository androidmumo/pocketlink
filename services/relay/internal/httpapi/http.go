package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Readiness interface{ Ready(context.Context) error }
type API struct {
	store    Readiness
	draining atomic.Bool
	version  string
	auth     *Auth
	console  http.Handler
}

func New(store Readiness, version string) *API             { return &API{store: store, version: version} }
func (a *API) EnableAuth(auth *Auth, console http.Handler) { a.auth = auth; a.console = console }
func (a *API) Drain() {
	a.draining.Store(true)
	if a.auth != nil {
		a.auth.stopStreams()
	}
}
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.auth != nil {
		if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/assets/") {
			a.console.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/") && r.URL.Path != "/api/v1/capabilities" {
			if a.draining.Load() {
				fail(w, 503, "unavailable")
				return
			}
			a.auth.ServeHTTP(w, r)
			return
		}
	}
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
		features := []string{}
		stage := "foundation"
		if a.auth != nil {
			features = []string{"admin_login", "device_pairing", "device_revocation", "rooms", "text_messages", "receipts", "device_websocket", "signed_firmware_ota", "invite_registration", "shared_rooms", "ptt_voice"}
			stage = "text"
		}
		write(w, 200, map[string]any{"protocol_versions": []int{1}, "server_version": a.version, "stage": stage, "enabled_features": features, "limits": map[string]int{"control_bytes": 8192, "text_utf8_bytes": 2048, "text_codepoints": 200, "realtime_plaintext_bytes": 1100}})
	}
}
