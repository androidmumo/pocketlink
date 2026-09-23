package httpapi

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

func (a *Auth) userManagementHTTP(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/api/v1/users" && !strings.HasPrefix(r.URL.Path, "/api/v1/users/") {
		return false
	}
	u, ok := a.principal(r)
	if !ok {
		fail(w, 401, "login_required")
		return true
	}
	if u.ID != "admin" {
		fail(w, 403, "admin_required")
		return true
	}
	if r.URL.Path == "/api/v1/users" && r.Method == "GET" {
		users, e := a.db.Users(r.Context())
		if e != nil {
			a.dbError(w, e)
		} else {
			write(w, 200, map[string]any{"users": users})
		}
		return true
	}
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return true
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/users/"), "/")
	if len(parts) != 2 || parts[0] == "" || len(parts[0]) > 64 {
		fail(w, 400, "invalid_request")
		return true
	}
	id, action := parts[0], parts[1]
	if id == "admin" {
		fail(w, 403, "protected_admin")
		return true
	}
	if action != "disable" && action != "enable" && action != "logout" && action != "password" {
		fail(w, 404, "not_found")
		return true
	}
	if a.messageLimit() {
		fail(w, 429, "try_later")
		return true
	}
	var salt, hash []byte
	if action == "password" {
		var input struct {
			Password string `json:"password"`
		}
		if !body(w, r, &input) {
			return true
		}
		if !utf8.ValidString(input.Password) || utf8.RuneCountInString(input.Password) < 12 || len(input.Password) > 128 {
			fail(w, 400, "invalid_password")
			return true
		}
		select {
		case a.verifying <- struct{}{}:
			defer func() { <-a.verifying }()
		default:
			fail(w, 429, "try_later")
			return true
		}
		salt = make([]byte, 32)
		if _, e := rand.Read(salt); e != nil {
			fail(w, 503, "unavailable")
			return true
		}
		var e error
		hash, e = pbkdf2.Key(sha256.New, input.Password, salt, iterations, 32)
		if e != nil {
			fail(w, 400, "invalid_password")
			return true
		}
	} else {
		var input struct{}
		if !body(w, r, &input) {
			return true
		}
	}
	if e := a.db.ManageUser(r.Context(), id, action, salt, hash, time.Now().Unix()); e != nil {
		a.messageError(w, e)
		return true
	}
	a.mu.Lock()
	for key, s := range a.sessions {
		if s.User.ID == id {
			delete(a.sessions, key)
		}
	}
	a.mu.Unlock()
	write(w, 200, map[string]bool{"updated": true})
	return true
}
