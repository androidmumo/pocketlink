package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/androidmumo/pocketlink/services/relay/internal/protocol"
	"github.com/coder/websocket"
	"net/http"
	"time"
)

type streamSession struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (a *Auth) stopStreams() {
	a.mu.Lock()
	a.stopping = true
	states := make([]*streamSession, 0, len(a.streams))
	for _, s := range a.streams {
		states = append(states, s)
		s.cancel()
	}
	a.mu.Unlock()
	for _, s := range states {
		<-s.done
	}
}

// One unacknowledged message in flight per device. Re-send every five seconds;
// only a durable client acknowledgement advances the queue. All I/O is bounded.
func (a *Auth) stream(w http.ResponseWriter, r *http.Request, device, hash string) {
	if r.URL.RawQuery != "" {
		fail(w, 400, "query_not_allowed")
		return
	}
	a.mu.Lock()
	busy := a.stopping || a.streams[device] != nil || len(a.streams) >= 64
	a.mu.Unlock()
	if busy {
		fail(w, 409, "stream_limit")
		return
	}
	c, e := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"pocketlink.v1"}, CompressionMode: websocket.CompressionDisabled})
	if e != nil {
		return
	}
	defer c.CloseNow()
	if c.Subprotocol() != "pocketlink.v1" {
		_ = c.Close(websocket.StatusPolicyViolation, "subprotocol required")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	state := &streamSession{cancel: cancel, done: make(chan struct{})}
	a.mu.Lock()
	if a.stopping || a.streams[device] != nil || len(a.streams) >= 64 {
		a.mu.Unlock()
		return
	}
	a.streams[device] = state
	a.mu.Unlock()
	defer func() { a.mu.Lock(); delete(a.streams, device); a.mu.Unlock(); close(state.done) }()
	c.SetReadLimit(protocol.MaxControlBytes)
	incoming := make(chan protocol.Envelope, 1)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		defer cancel()
		for {
			typ, b, e := c.Read(ctx)
			if e != nil {
				return
			}
			if typ != websocket.MessageText {
				return
			}
			v, e := protocol.DecodeControl(b)
			if e != nil || v.Namespace != "text" || v.Type != "text.ack" || v.RoomID != "" {
				return
			}
			select {
			case incoming <- v:
			case <-ctx.Done():
				return
			}
		}
	}()
	defer func() { cancel(); c.CloseNow(); <-readerDone }()
	send := func(kind, key, room string, payload any) error {
		raw, e := json.Marshal(payload)
		if e != nil {
			return e
		}
		data, e := json.Marshal(protocol.Envelope{Version: 1, Type: kind, RequestID: key, Namespace: "text", RoomID: room, Payload: raw})
		if e != nil {
			return e
		}
		writeCtx, stop := context.WithTimeout(ctx, 5*time.Second)
		defer stop()
		return c.Write(writeCtx, websocket.MessageText, data)
	}
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	lastID := int64(0)
	lastSent := time.Time{}
	lastPing := time.Now()
	ackWindow := time.Now()
	ackCount := 0
	// Immediate first delivery, then one-second persistence/permission checks.
	tick := make(chan time.Time, 1)
	tick <- time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-incoming:
			if time.Since(ackWindow) >= time.Minute {
				ackWindow = time.Now()
				ackCount = 0
			}
			ackCount++
			if ackCount > 120 {
				return
			}
			var ack struct {
				MessageID int64  `json:"message_id"`
				State     string `json:"state"`
			}
			decoder := json.NewDecoder(bytes.NewReader(v.Payload))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&ack) != nil {
				return
			}
			if e := a.db.Acknowledge(ctx, hash, ack.MessageID, ack.State, time.Now().Unix()); e != nil {
				return
			}
			if send("ack.result", v.RequestID, "", map[string]any{"message_id": ack.MessageID, "acknowledged": true}) != nil {
				return
			}
			lastSent = time.Time{}
			select {
			case tick <- time.Now():
			default:
			}
		case <-timer.C:
			select {
			case tick <- time.Now():
			default:
			}
		case <-tick:
			if _, e := a.db.DeviceByCredential(ctx, hash); e != nil {
				return
			}
			ms, e := a.db.Pending(ctx, hash)
			if e != nil {
				return
			}
			if len(ms) > 0 && (ms[0].ID != lastID || time.Since(lastSent) >= 5*time.Second) {
				m := ms[0]
				if send("text.message", m.RequestID, m.RoomID, m) != nil {
					return
				}
				lastID = m.ID
				lastSent = time.Now()
			}
			if time.Since(lastPing) >= 20*time.Second {
				pingCtx, stop := context.WithTimeout(ctx, 5*time.Second)
				e := c.Ping(pingCtx)
				stop()
				if e != nil {
					return
				}
				lastPing = time.Now()
			}
		}
	}
}
