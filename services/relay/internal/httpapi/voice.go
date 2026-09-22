package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/androidmumo/pocketlink/services/relay/internal/voice"
	"github.com/coder/websocket"
)

func (a *Auth) voiceHTTP(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path == "/api/v1/device/rooms" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return true
		}
		rooms, e := a.db.DeviceRooms(r.Context(), credential(r))
		if e != nil {
			a.dbError(w, e)
		} else {
			write(w, 200, map[string]any{"rooms": rooms})
		}
		return true
	}
	device := strings.HasPrefix(path, "/api/v1/device/voice/")
	browser := strings.HasPrefix(path, "/api/v1/voice/")
	if !device && !browser {
		return false
	}
	if r.Method != "GET" || r.URL.RawQuery != "" {
		fail(w, 400, "invalid_request")
		return true
	}
	room := strings.TrimPrefix(path, "/api/v1/voice/")
	hash, user, key := "", "", ""
	if device {
		room = strings.TrimPrefix(path, "/api/v1/device/voice/")
		hash = credential(r)
		if hash == "" {
			fail(w, 401, "invalid_credentials")
			return true
		}
		d, e := a.db.DeviceByCredential(r.Context(), hash)
		if e != nil {
			a.dbError(w, e)
			return true
		}
		key = "voice:device:" + d.ID
	} else {
		if r.Header.Get("Origin") != a.origin {
			fail(w, 403, "origin_denied")
			return true
		}
		u, ok := a.principal(r)
		if !ok {
			fail(w, 401, "login_required")
			return true
		}
		user = u.ID
		c, _ := r.Cookie(sessionCookie)
		key = "voice:web:" + digest(c.Value)
	}
	if len(room) > 64 || strings.Contains(room, "/") {
		fail(w, 400, "invalid_request")
		return true
	}
	name, e := a.db.VoiceIdentity(r.Context(), room, hash, user)
	if e != nil {
		fail(w, 403, "room_denied")
		return true
	}
	a.mu.Lock()
	busy := a.stopping || len(a.streams) >= 64 || a.streams[key] != nil
	a.mu.Unlock()
	if busy {
		fail(w, 409, "stream_limit")
		return true
	}
	c, e := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"pocketlink.voice.v1"}, CompressionMode: websocket.CompressionDisabled, OriginPatterns: []string{strings.TrimPrefix(a.origin, "https://")}})
	if e != nil {
		return true
	}
	defer c.CloseNow()
	if c.Subprotocol() != "pocketlink.voice.v1" {
		return true
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	state := &streamSession{cancel: cancel, done: make(chan struct{})}
	a.mu.Lock()
	if a.stopping || a.streams[key] != nil || len(a.streams) >= 64 {
		a.mu.Unlock()
		return true
	}
	a.streams[key] = state
	a.mu.Unlock()
	defer func() { a.mu.Lock(); delete(a.streams, key); a.mu.Unlock(); close(state.done) }()
	p := &voice.Peer{Room: room, Name: name, Out: make(chan voice.Packet, 16), Stop: cancel}
	if !a.voice.Join(p, time.Now()) {
		return true
	}
	defer func() { a.voice.Leave(p, time.Now()) }()
	c.SetReadLimit(1024)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer cancel()
		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeat.C:
				ping, stop := context.WithTimeout(ctx, 5*time.Second)
				e := c.Ping(ping)
				stop()
				if e != nil {
					return
				}
			case packet := <-p.Out:
				if packet.Binary && time.Since(packet.At) > 160*time.Millisecond {
					continue
				}
				typ := websocket.MessageText
				if packet.Binary {
					typ = websocket.MessageBinary
				}
				wctx, stop := context.WithTimeout(ctx, 500*time.Millisecond)
				e := c.Write(wctx, typ, packet.Data)
				stop()
				if e != nil {
					return
				}
			}
		}
	}()
	defer func() { cancel(); c.CloseNow(); <-done }()
	permission := time.Time{}
	window := time.Now()
	controls := 0
	tickDone := make(chan struct{})
	go func() {
		defer close(tickDone)
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-timer.C:
				a.voice.Tick(now)
				if _, e := a.db.VoiceIdentity(ctx, room, hash, user); e != nil {
					cancel()
					return
				}
				if browser {
					if _, ok := a.principal(r); !ok {
						cancel()
						return
					}
				}
			}
		}
	}()
	defer func() { cancel(); <-tickDone }()
	for {
		typ, b, e := c.Read(ctx)
		if e != nil {
			return true
		}
		now := time.Now()
		if now.Sub(permission) >= time.Second {
			if _, e = a.db.VoiceIdentity(ctx, room, hash, user); e != nil {
				return true
			}
			if browser {
				if _, ok := a.principal(r); !ok {
					return true
				}
			}
			permission = now
		}
		if typ == websocket.MessageBinary {
			if !a.voice.Audio(p, b, now) {
				return true
			}
			continue
		}
		if now.Sub(window) >= time.Second {
			window = now
			controls = 0
		}
		controls++
		if controls > 10 {
			return true
		}
		var v struct {
			Type    string `json:"type"`
			Request uint32 `json:"request_id"`
			Stream  uint32 `json:"stream"`
		}
		d := json.NewDecoder(bytes.NewReader(b))
		d.DisallowUnknownFields()
		if d.Decode(&v) != nil || d.Decode(new(any)) != io.EOF {
			return true
		}
		switch v.Type {
		case "request":
			if v.Request == 0 {
				return true
			}
			a.voice.Request(p, v.Request, now)
		case "release":
			a.voice.Release(p, v.Stream, now)
		default:
			return true
		}
	}
}
