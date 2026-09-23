package httpapi

import (
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func managedFixture(t *testing.T) (*API, *http.Cookie, *http.Cookie, string, string, string) {
	t.Helper()
	a, _ := fixture(t)
	admin := login(t, a)
	user := newAccount(t, a, admin, "managed")
	id := responseData(t, jsonCall(t, a, "GET", "auth/me", nil, user), 200)["user"].(map[string]any)["id"].(string)
	code := pairing(t, a, user)
	w := request(t, a, "POST", "/api/v1/device/pair", "", `{"code":"`+code+`","sn":"managed-device"}`, nil, "")
	var d map[string]any
	json.Unmarshal(w.Body.Bytes(), &d)
	if w.Code != 201 {
		t.Fatal(w.Code)
	}
	device := d["device"].(map[string]any)["id"].(string)
	token := d["credential"].(string)
	room := responseData(t, jsonCall(t, a, "POST", "rooms", map[string]string{"name": "managed room"}, user), 201)["id"].(string)
	responseData(t, jsonCall(t, a, "PUT", "rooms/"+room+"/members/"+device, nil, user), 200)
	return a, admin, user, id, token, room
}
func TestManageUsersAccessDisableRestore(t *testing.T) {
	a, admin, user, id, token, room := managedFixture(t)
	responseData(t, jsonCall(t, a, "GET", "users", nil, nil), 401)
	responseData(t, jsonCall(t, a, "GET", "users", nil, user), 403)
	responseData(t, jsonCall(t, a, "POST", "users/"+id+"/disable", map[string]any{}, user), 403)
	for _, action := range []string{"disable", "enable", "logout", "password"} {
		responseData(t, jsonCall(t, a, "POST", "users/admin/"+action, map[string]string{"password": "another-long-password"}, admin), 403)
	}
	responseData(t, jsonCall(t, a, "POST", "users/missing/logout", map[string]any{}, admin), 404)
	rows := responseData(t, jsonCall(t, a, "GET", "users", nil, admin), 200)["users"].([]any)
	for _, row := range rows {
		u := row.(map[string]any)
		for _, key := range []string{"password_hash", "password_salt", "auth_version"} {
			if _, exists := u[key]; exists {
				t.Fatal("secret field", key)
			}
		}
		if u["id"] == id && (u["device_count"] != float64(1) || u["room_count"] != float64(1)) {
			t.Fatal("counts", u)
		}
	}
	pendingCode := pairing(t, a, user)
	roomCode := invite(t, a, user, "room", room)
	responseData(t, jsonCall(t, a, "POST", "users/"+id+"/disable", map[string]any{}, admin), 200)
	responseData(t, jsonCall(t, a, "GET", "auth/me", nil, user), 401)
	responseData(t, jsonCall(t, a, "POST", "auth/login", map[string]string{"username": "managed", "password": "user-password-for-tests"}, nil), 401)
	if w := request(t, a, "GET", "/api/v1/device/me", "", "", nil, token); w.Code != 401 {
		t.Fatal("disabled device", w.Code)
	}
	responseData(t, jsonCall(t, a, "POST", "users/"+id+"/enable", map[string]any{}, admin), 200)
	responseData(t, jsonCall(t, a, "GET", "auth/me", nil, user), 401)
	if w := request(t, a, "GET", "/api/v1/device/me", "", "", nil, token); w.Code != 200 {
		t.Fatal("restored device", w.Code)
	}
	if w := request(t, a, "POST", "/api/v1/device/pair", "", `{"code":"`+pendingCode+`","sn":"later"}`, nil, ""); w.Code != 401 {
		t.Fatal("old pairing survived", w.Code)
	}
	responseData(t, jsonCall(t, a, "POST", "rooms/join", map[string]string{"code": roomCode}, admin), 400)
	responseData(t, jsonCall(t, a, "GET", "rooms/"+room+"/members", nil, admin), 200)
}
func TestManageLogoutAndPassword(t *testing.T) {
	a, admin, user, id, token, _ := managedFixture(t)
	oldUser, e := a.auth.db.User(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	responseData(t, jsonCall(t, a, "POST", "users/"+id+"/logout", map[string]any{}, admin), 200)
	responseData(t, jsonCall(t, a, "GET", "auth/me", nil, user), 401)
	recorder := httptest.NewRecorder()
	a.auth.startSession(recorder, httptest.NewRequest("POST", "/", nil), oldUser)
	if recorder.Code != 401 {
		t.Fatal("stale login created session")
	}
	response := jsonCall(t, a, "POST", "auth/login", map[string]string{"username": "managed", "password": "user-password-for-tests"}, nil)
	user = response.Cookies()[0]
	responseData(t, response, 200)
	responseData(t, jsonCall(t, a, "POST", "users/"+id+"/password", map[string]string{"password": "short"}, admin), 400)
	responseData(t, jsonCall(t, a, "POST", "users/"+id+"/password", map[string]string{"password": "new-password-for-tests"}, admin), 200)
	responseData(t, jsonCall(t, a, "GET", "auth/me", nil, user), 401)
	responseData(t, jsonCall(t, a, "POST", "auth/login", map[string]string{"username": "managed", "password": "user-password-for-tests"}, nil), 401)
	responseData(t, jsonCall(t, a, "POST", "auth/login", map[string]string{"username": "managed", "password": "new-password-for-tests"}, nil), 200)
	if w := request(t, a, "GET", "/api/v1/device/me", "", "", nil, token); w.Code != 200 {
		t.Fatal("reset changed device", w.Code)
	}
}
func TestDisableClosesVoicePeers(t *testing.T) {
	a, admin, user, id, token, room := managedFixture(t)
	srv := httptest.NewServer(a)
	defer srv.Close()
	defer a.Drain()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	headers := []http.Header{{"Origin": []string{testOrigin}, "Cookie": []string{user.String()}}, {"Authorization": []string{"Bearer " + token}}}
	paths := []string{"/api/v1/voice/", "/api/v1/device/voice/"}
	var peers []*websocket.Conn
	for i, h := range headers {
		c, _, e := websocket.Dial(ctx, strings.Replace(srv.URL, "http", "ws", 1)+paths[i]+room, &websocket.DialOptions{HTTPHeader: h, Subprotocols: []string{"pocketlink.voice.v1"}})
		if e != nil {
			t.Fatal(e)
		}
		defer c.CloseNow()
		voiceControl(t, c, "floor")
		peers = append(peers, c)
	}
	responseData(t, jsonCall(t, a, "POST", "users/"+id+"/disable", map[string]any{}, admin), 200)
	for _, c := range peers {
		deadline, stop := context.WithTimeout(ctx, 3*time.Second)
		_, _, e := c.Read(deadline)
		stop()
		if e == nil {
			t.Fatal("disabled peer remained open")
		}
	}
}
