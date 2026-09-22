// Package console embeds the device management UI in the relay binary.
package console

import (
	"embed"
	"net/http"
	"strings"
)

//go:embed index.html assets/*
var files embed.FS

func Handler(origin ...string) http.Handler {
	connections := "'self'"
	if len(origin) > 0 && strings.HasPrefix(origin[0], "https://") {
		connections += " wss://" + strings.TrimPrefix(origin[0], "https://")
	}
	static := http.FileServerFS(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src "+connections+"; worker-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path != "/" && r.URL.Path != "/assets/voice.js" && r.URL.Path != "/assets/voice-worklet.js" && r.URL.Path != "/assets/mobile.js" && r.URL.Path != "/assets/app.js" && r.URL.Path != "/assets/room-picker.js" && r.URL.Path != "/assets/style.css" {
			http.NotFound(w, r)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		static.ServeHTTP(w, r)
	})
}
