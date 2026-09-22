package httpapi

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/androidmumo/pocketlink/services/relay/internal/voice"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func voiceControl(t *testing.T, c *websocket.Conn, kind string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		typ, b, e := c.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if typ != websocket.MessageText {
			continue
		}
		var v map[string]any
		if json.Unmarshal(b, &v) != nil {
			t.Fatal("json")
		}
		if v["type"] == kind {
			return v
		}
	}
}
func TestVoiceDeviceBrowserAndRevocation(t *testing.T) {
	a, cookie, token, _ := textFixture(t)
	srv := httptest.NewServer(a)
	defer srv.Close()
	defer a.Drain()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(path string, h http.Header) *websocket.Conn {
		t.Helper()
		c, _, e := websocket.Dial(ctx, strings.Replace(srv.URL, "http", "ws", 1)+path, &websocket.DialOptions{HTTPHeader: h, Subprotocols: []string{"pocketlink.voice.v1"}})
		if e != nil {
			t.Fatal(e)
		}
		return c
	}
	device := dial("/api/v1/device/voice/room", http.Header{"Authorization": []string{"Bearer " + token}})
	defer device.CloseNow()
	browser := dial("/api/v1/voice/room", http.Header{"Origin": []string{testOrigin}, "Cookie": []string{cookie.String()}})
	defer browser.CloseNow()
	voiceControl(t, device, "floor")
	voiceControl(t, browser, "floor")
	if e := device.Write(ctx, websocket.MessageText, []byte(`{"type":"request","request_id":1}`)); e != nil {
		t.Fatal(e)
	}
	grant := voiceControl(t, device, "grant")
	voiceControl(t, browser, "floor")
	frame := make([]byte, voice.FrameBytes)
	binary.BigEndian.PutUint32(frame, uint32(grant["stream"].(float64)))
	binary.BigEndian.PutUint32(frame[4:], 1)
	frame[8] = 42
	if e := device.Write(ctx, websocket.MessageBinary, frame); e != nil {
		t.Fatal(e)
	}
	typ, b, e := browser.Read(ctx)
	if e != nil || typ != websocket.MessageBinary || len(b) != voice.FrameBytes || b[8] != 42 {
		t.Fatal("PCM relay", e)
	}
	w := request(t, a, "DELETE", "/api/v1/rooms/room/members/device", testOrigin, "", cookie, "")
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	// Reader wakes on permission revocation even with no further client traffic.
	rctx, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	if _, _, e = device.Read(rctx); e == nil {
		t.Fatal("revoked peer remains connected")
	}
}
func TestVoiceHandshakeBoundaries(t *testing.T) {
	a, cookie, token, _ := textFixture(t)
	srv := httptest.NewServer(a)
	defer srv.Close()
	defer a.Drain()
	for _, tc := range []struct {
		path    string
		headers http.Header
		status  int
	}{
		{"/api/v1/voice/room", http.Header{"Cookie": []string{cookie.String()}}, 403},
		{"/api/v1/device/voice/room", http.Header{"Cookie": []string{cookie.String()}}, 401},
		{"/api/v1/device/voice/other", http.Header{"Authorization": []string{"Bearer " + token}}, 403},
		{"/api/v1/voice/room?token=secret", http.Header{"Origin": []string{testOrigin}, "Cookie": []string{cookie.String()}}, 400},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		c, r, e := websocket.Dial(ctx, strings.Replace(srv.URL, "http", "ws", 1)+tc.path, &websocket.DialOptions{HTTPHeader: tc.headers, Subprotocols: []string{"pocketlink.voice.v1"}})
		cancel()
		if e == nil {
			c.CloseNow()
			t.Fatal("invalid handshake accepted")
		}
		if r == nil || r.StatusCode != tc.status {
			t.Fatal(tc.path, r)
		}
	}
}
