package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func jsonCall(t *testing.T, a http.Handler, method, path string, input any, cookie *http.Cookie) *http.Response {
	t.Helper()
	b, _ := json.Marshal(input)
	w := request(t, a, method, "/api/v1/"+path, testOrigin, string(b), cookie, "")
	return w.Result()
}
func responseData(t *testing.T, r *http.Response, status int) map[string]any {
	t.Helper()
	defer r.Body.Close()
	var v map[string]any
	json.NewDecoder(r.Body).Decode(&v)
	if r.StatusCode != status {
		t.Fatalf("status %d want %d: %v", r.StatusCode, status, v)
	}
	return v
}
func invite(t *testing.T, a http.Handler, c *http.Cookie, kind, room string) string {
	t.Helper()
	v := responseData(t, jsonCall(t, a, "POST", "invitations", map[string]string{"kind": kind, "room_id": room}, c), 201)
	return v["code"].(string)
}
func newAccount(t *testing.T, a http.Handler, admin *http.Cookie, username string) *http.Cookie {
	t.Helper()
	code := invite(t, a, admin, "registration", "")
	responseData(t, jsonCall(t, a, "POST", "auth/register", map[string]string{"username": username, "password": "user-password-for-tests", "code": code}, nil), 201)
	r := jsonCall(t, a, "POST", "auth/login", map[string]string{"username": username, "password": "user-password-for-tests"}, nil)
	cookies := r.Cookies()
	data := responseData(t, r, 200)
	if data["role"] != "user" {
		t.Fatal("registration escalated role")
	}
	return cookies[0]
}
func TestInviteAccountsAndRoomIsolation(t *testing.T) {
	a, _ := fixture(t)
	admin := login(t, a)
	alice := newAccount(t, a, admin, "alice")
	bob := newAccount(t, a, admin, "bob")
	// Ordinary users cannot mint registration invitations or manage global firmware.
	responseData(t, jsonCall(t, a, "POST", "invitations", map[string]string{"kind": "registration"}, alice), 403)
	r := jsonCall(t, a, "GET", "firmware", nil, alice)
	if r.StatusCode != 403 {
		t.Fatal(r.StatusCode)
	}
	r.Body.Close()
	createDevice := func(c *http.Cookie, sn string) map[string]any {
		code := pairing(t, a, c)
		w := request(t, a, "POST", "/api/v1/device/pair", "", `{"code":"`+code+`","sn":"`+sn+`"}`, nil, "")
		if w.Code != 201 {
			t.Fatal(w.Body.String())
		}
		var v map[string]any
		json.Unmarshal(w.Body.Bytes(), &v)
		return v
	}
	ad := createDevice(alice, "alice-device")
	bd := createDevice(bob, "bob-device")
	aid := ad["device"].(map[string]any)["id"].(string)
	bid := bd["device"].(map[string]any)["id"].(string)
	devices := responseData(t, jsonCall(t, a, "GET", "devices", nil, bob), 200)["devices"].([]any)
	if len(devices) != 1 || devices[0].(map[string]any)["id"] != bid {
		t.Fatal("device list leaked")
	}
	r = jsonCall(t, a, "DELETE", "devices/"+aid, nil, bob)
	if r.StatusCode == 200 {
		t.Fatal("cross-account revocation")
	}
	r.Body.Close()
	room := responseData(t, jsonCall(t, a, "POST", "rooms", map[string]string{"name": "private"}, alice), 201)["id"].(string)
	responseData(t, jsonCall(t, a, "PUT", "rooms/"+room+"/members/"+aid, nil, alice), 200)
	old := responseData(t, jsonCall(t, a, "POST", "rooms/"+room+"/messages", map[string]string{"request_id": "before-join", "text": "private history"}, alice), 200)
	for _, path := range []string{"rooms/" + room + "/members", "rooms/" + room + "/messages", "rooms/" + room + "/users"} {
		responseData(t, jsonCall(t, a, "GET", path, nil, bob), 404)
	}
	responseData(t, jsonCall(t, a, "DELETE", "rooms/"+room, nil, bob), 404)
	responseData(t, jsonCall(t, a, "POST", "invitations", map[string]string{"kind": "room", "room_id": room}, bob), 404)
	code := invite(t, a, alice, "room", room)
	responseData(t, jsonCall(t, a, "POST", "rooms/join", map[string]string{"code": code}, bob), 200)
	responseData(t, jsonCall(t, a, "POST", "rooms/join", map[string]string{"code": code}, bob), 400)
	history := responseData(t, jsonCall(t, a, "GET", "rooms/"+room+"/messages", nil, bob), 200)["messages"].([]any)
	if len(history) != 0 {
		t.Fatal("pre-invitation history leaked")
	}
	oldID := int(old["id"].(float64))
	idJSON, _ := json.Marshal(oldID)
	responseData(t, jsonCall(t, a, "GET", "messages/"+string(idJSON)+"/receipts", nil, bob), 404)
	responseData(t, jsonCall(t, a, "POST", "rooms/"+room+"/messages", map[string]string{"request_id": "before-join", "text": "private history"}, bob), 404)
	// Consent: a guest can only add their own devices, and cannot archive the room.
	responseData(t, jsonCall(t, a, "PUT", "rooms/"+room+"/members/"+aid, nil, bob), 404)
	responseData(t, jsonCall(t, a, "PUT", "rooms/"+room+"/members/"+bid, nil, bob), 200)
	responseData(t, jsonCall(t, a, "DELETE", "rooms/"+room, nil, bob), 404)
	responseData(t, jsonCall(t, a, "POST", "rooms/"+room+"/messages", map[string]string{"request_id": "shared", "text": "hello together"}, bob), 200)
	buser := responseData(t, jsonCall(t, a, "GET", "auth/me", nil, bob), 200)["user"].(map[string]any)["id"].(string)
	responseData(t, jsonCall(t, a, "DELETE", "rooms/"+room+"/users/"+buser, nil, alice), 200)
	responseData(t, jsonCall(t, a, "GET", "rooms/"+room+"/messages", nil, bob), 404)
	w := request(t, a, "GET", "/api/v1/device/inbox", "", "", nil, bd["credential"].(string))
	var pending struct{ Messages []any }
	json.Unmarshal(w.Body.Bytes(), &pending)
	if w.Code != 200 || len(pending.Messages) != 0 {
		t.Fatal("removed user's pending delivery survived")
	}
	// A revoked serial number cannot be taken over by a different owner.
	responseData(t, jsonCall(t, a, "DELETE", "devices/"+aid, nil, alice), 200)
	code = pairing(t, a, bob)
	w = request(t, a, "POST", "/api/v1/device/pair", "", `{"code":"`+code+`","sn":"alice-device"}`, nil, "")
	if w.Code != 401 {
		t.Fatal("cross-account serial takeover", w.Code)
	}
}
func TestRegistrationInviteConsumption(t *testing.T) {
	a, _ := fixture(t)
	admin := login(t, a)
	code := invite(t, a, admin, "registration", "")
	for _, username := range []string{"admin", "x", "bad name"} {
		responseData(t, jsonCall(t, a, "POST", "auth/register", map[string]string{"username": username, "password": "user-password-for-tests", "code": code}, nil), 400)
	}
	responseData(t, jsonCall(t, a, "POST", "auth/register", map[string]string{"username": "valid_user", "password": "user-password-for-tests", "code": code}, nil), 201)
	responseData(t, jsonCall(t, a, "POST", "auth/register", map[string]string{"username": "other_user", "password": "user-password-for-tests", "code": code}, nil), 400)
	list := responseData(t, jsonCall(t, a, "GET", "invitations", nil, admin), 200)["invitations"].([]any)
	for _, item := range list {
		row := item.(map[string]any)
		if row["code"] != nil || row["code_hash"] != nil {
			t.Fatal("invitation secret leaked")
		}
	}
	code = invite(t, a, admin, "registration", "")
	list = responseData(t, jsonCall(t, a, "GET", "invitations", nil, admin), 200)["invitations"].([]any)
	for _, item := range list {
		row := item.(map[string]any)
		if row["used_at"] == nil {
			responseData(t, jsonCall(t, a, "DELETE", "invitations/"+row["id"].(string), nil, admin), 200)
		}
	}
	responseData(t, jsonCall(t, a, "POST", "auth/register", map[string]string{"username": "revoked_user", "password": "user-password-for-tests", "code": code}, nil), 400)
}
