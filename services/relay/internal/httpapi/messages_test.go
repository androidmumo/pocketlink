package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/androidmumo/pocketlink/services/relay/internal/protocol"
	"github.com/androidmumo/pocketlink/services/relay/internal/store"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func textFixture(t *testing.T) (*API, *http.Cookie, string, store.Message) {
	t.Helper()
	a, db := fixture(t)
	c := login(t, a)
	ctx := context.Background()
	token := secret()
	if e := db.CreatePairing(ctx, "pair", "test", 10); e != nil {
		t.Fatal(e)
	}
	if e := db.ConsumePairing(ctx, "pair", "device", "SN", digest(token), 11); e != nil {
		t.Fatal(e)
	}
	if e := db.CreateRoom(ctx, "room", "room", 12); e != nil {
		t.Fatal(e)
	}
	if e := db.SetMember(ctx, "room", "device", true, 13); e != nil {
		t.Fatal(e)
	}
	m, e := db.SendText(ctx, "room", "initial", "hello", 14)
	if e != nil {
		t.Fatal(e)
	}
	return a, c, token, m
}
func TestMessageHTTP(t *testing.T) {
	a, c, token, m := textFixture(t)
	if request(t, a, "GET", "/api/v1/rooms", "", "", nil, token).Code != 401 {
		t.Fatal("device became admin")
	}
	if request(t, a, "POST", "/api/v1/rooms/room/messages", "", `{"request_id":"new","text":"test"}`, c, "").Code != 403 {
		t.Fatal("CSRF")
	}
	for range 2 {
		w := request(t, a, "POST", "/api/v1/rooms/room/messages", testOrigin, `{"request_id":"new","text":"test"}`, c, "")
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w := request(t, a, "POST", "/api/v1/rooms/room/messages", testOrigin, `{"request_id":"new","text":"changed"}`, c, "")
	if w.Code != 409 {
		t.Fatal(w.Code)
	}
	w = request(t, a, "GET", "/api/v1/device/inbox", "", "", nil, token)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "hello") {
		t.Fatal(w.Code)
	}
	w = request(t, a, "POST", "/api/v1/device/ack", "", fmt.Sprintf(`{"message_id":%d,"state":"received"}`, m.ID), nil, token)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if request(t, a, "GET", "/api/v1/device/inbox", "", "", c, "").Code != 401 {
		t.Fatal("cookie became device")
	}
	if request(t, a, "GET", "/api/v1/rooms/room/messages?before=-1", "", "", c, "").Code != 400 {
		t.Fatal("invalid cursor")
	}
}
func dialDevice(t *testing.T, url, token string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, e := websocket.Dial(ctx, strings.Replace(url, "http", "ws", 1)+"/api/v1/device/stream", &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": []string{"Bearer " + token}}, Subprotocols: []string{"pocketlink.v1"}})
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func readFrame(t *testing.T, c *websocket.Conn) protocol.Envelope {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, b, e := c.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	v, e := protocol.DecodeControl(b)
	if e != nil {
		t.Fatal(e, string(b))
	}
	return v
}
func TestWebSocketDeliveryReconnectAndDrain(t *testing.T) {
	a, _, token, m := textFixture(t)
	srv := httptest.NewServer(a)
	defer srv.Close()
	defer a.Drain()
	c := dialDevice(t, srv.URL, token)
	v := readFrame(t, c)
	if v.Type != "text.message" {
		t.Fatal(v.Type)
	}
	c.CloseNow()
	// Wait for the disconnected handler to release its single-device slot.
	deadline := time.Now().Add(2 * time.Second)
	for {
		a.auth.mu.Lock()
		n := len(a.auth.streams)
		a.auth.mu.Unlock()
		if n == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("stream leaked")
		}
		time.Sleep(time.Millisecond * 10)
	}
	c = dialDevice(t, srv.URL, token)
	defer c.CloseNow()
	v = readFrame(t, c)
	var message store.Message
	json.Unmarshal(v.Payload, &message)
	if message.ID != m.ID {
		t.Fatal("offline replay lost")
	}
	ack := fmt.Sprintf(`{"version":1,"type":"text.ack","request_id":"ack-1","namespace":"text","payload":{"message_id":%d,"state":"received"}}`, m.ID)
	if e := c.Write(context.Background(), websocket.MessageText, []byte(ack)); e != nil {
		t.Fatal(e)
	}
	if v = readFrame(t, c); v.Type != "ack.result" {
		t.Fatal(v.Type)
	}
	pending, e := a.auth.db.Pending(context.Background(), digest(token))
	if e != nil || len(pending) != 0 {
		t.Fatal("ack not durable")
	}
	done := make(chan struct{})
	go func() { a.Drain(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("drain leaked socket")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, _, e = c.Read(ctx); e == nil {
		t.Fatal("socket survived drain")
	}
}
func TestWebSocketRevocationAndOrigin(t *testing.T) {
	a, _, token, _ := textFixture(t)
	srv := httptest.NewServer(a)
	defer srv.Close()
	defer a.Drain()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, response, e := websocket.Dial(ctx, strings.Replace(srv.URL, "http", "ws", 1)+"/api/v1/device/stream", &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": []string{"Bearer " + token}, "Origin": []string{testOrigin}}, Subprotocols: []string{"pocketlink.v1"}})
	if e == nil || response == nil || response.StatusCode != 403 {
		t.Fatal("browser origin accepted")
	}
	c := dialDevice(t, srv.URL, token)
	defer c.CloseNow()
	readFrame(t, c)
	if e = a.auth.db.RevokeDevice(context.Background(), "device", 20); e != nil {
		t.Fatal(e)
	}
	if _, _, e = c.Read(ctx); e == nil {
		t.Fatal("revoked stream survived")
	}
}

func TestWebSocketRejectsInvalidFrames(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind websocket.MessageType
		data string
	}{
		{"binary", websocket.MessageBinary, "data"},
		{"oversize", websocket.MessageText, strings.Repeat("x", 8193)},
		{"foreign ack", websocket.MessageText, `{"version":1,"type":"text.ack","request_id":"bad","namespace":"text","payload":{"message_id":9999,"state":"read"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _, token, _ := textFixture(t)
			srv := httptest.NewServer(a)
			defer srv.Close()
			defer a.Drain()
			c := dialDevice(t, srv.URL, token)
			defer c.CloseNow()
			readFrame(t, c)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = c.Write(ctx, tc.kind, []byte(tc.data))
			if _, _, e := c.Read(ctx); e == nil {
				t.Fatal("invalid frame accepted")
			}
		})
	}
}
