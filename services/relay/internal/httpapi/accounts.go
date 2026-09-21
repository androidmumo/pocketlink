package httpapi

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"github.com/androidmumo/pocketlink/services/relay/internal/store"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type session struct {
	Expires time.Time
	User    store.User
}

var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)

func (a *Auth) principal(r *http.Request) (store.User, bool) {
	c, e := r.Cookie(sessionCookie)
	if e != nil || !validSecret(c.Value) {
		return store.User{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for h, v := range a.sessions {
		if !now.Before(v.Expires) {
			delete(a.sessions, h)
		}
	}
	v, ok := a.sessions[digest(c.Value)]
	return v.User, ok
}
func (a *Auth) startSession(w http.ResponseWriter, r *http.Request, u store.User) {
	token := secret()
	a.mu.Lock()
	now := time.Now()
	for h, v := range a.sessions {
		if !now.Before(v.Expires) {
			delete(a.sessions, h)
		}
	}
	// Replace a browser's current session rather than retaining an abandoned token.
	if c, e := r.Cookie(sessionCookie); e == nil {
		delete(a.sessions, digest(c.Value))
	}
	count := 0
	for _, v := range a.sessions {
		if v.User.ID == u.ID {
			count++
		}
	}
	if len(a.sessions) >= 256 || count >= 16 {
		a.mu.Unlock()
		fail(w, 429, "session_limit")
		return
	}
	a.sessions[digest(token)] = session{now.Add(12 * time.Hour), u}
	a.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 43200})
	write(w, 200, identity(u))
}
func identity(u store.User) map[string]any {
	role := "user"
	if u.ID == "admin" {
		role = "admin"
	}
	return map[string]any{"authenticated": true, "user": u, "role": role}
}
func (a *Auth) accountHTTP(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path == "/api/v1/auth/login" || path == "/api/v1/auth/register" {
		if r.Method != "POST" {
			fail(w, 405, "method_not_allowed")
			return true
		}
		if a.limited() {
			fail(w, 429, "try_later")
			return true
		}
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Code     string `json:"code"`
		}
		if !body(w, r, &input) {
			return true
		}
		input.Username = strings.ToLower(strings.TrimSpace(input.Username))
		register := path == "/api/v1/auth/register"
		if len(input.Password) > 256 {
			fail(w, 400, "invalid_request")
			return true
		}
		if register && (!usernamePattern.MatchString(input.Username) || input.Username == "admin" || utf8.RuneCountInString(input.Password) < 12 || len(input.Password) > 128 || !validSecret(input.Code)) {
			fail(w, 400, "invalid_registration")
			return true
		}
		select {
		case a.verifying <- struct{}{}:
			defer func() { <-a.verifying }()
		default:
			fail(w, 429, "try_later")
			return true
		}
		if register {
			salt := make([]byte, 32)
			if _, e := rand.Read(salt); e != nil {
				fail(w, 503, "unavailable")
				return true
			}
			hash, e := pbkdf2.Key(sha256.New, input.Password, salt, iterations, 32)
			if e != nil {
				fail(w, 400, "invalid_registration")
				return true
			}
			_, e = a.db.Register(r.Context(), secret(), input.Username, digest(input.Code), salt, hash, time.Now().Unix())
			if e != nil {
				if e == store.ErrConflict {
					fail(w, 409, "username_unavailable")
				} else if e == store.ErrDenied {
					fail(w, 400, "invalid_invitation")
				} else {
					a.dbError(w, e)
				}
				return true
			}
			write(w, 201, map[string]bool{"registered": true})
			return true
		}
		user := store.User{ID: "admin", Username: "admin"}
		salt, expected := a.salt, a.passwordHash
		valid := true
		if input.Username != "" && input.Username != "admin" {
			u, s, h, e := a.db.UserCredentials(r.Context(), input.Username)
			if e == nil {
				user, salt, expected = u, s, h
			} else {
				valid = false
			}
		}
		hash, e := pbkdf2.Key(sha256.New, input.Password, salt, iterations, 32)
		if e != nil || subtle.ConstantTimeCompare(hash, expected) != 1 || !valid {
			fail(w, 401, "invalid_credentials")
			return true
		}
		a.startSession(w, r, user)
		return true
	}
	if path != "/api/v1/invitations" && !strings.HasPrefix(path, "/api/v1/invitations/") && path != "/api/v1/rooms/join" {
		return false
	}
	user, ok := a.principal(r)
	if !ok {
		fail(w, 401, "login_required")
		return true
	}
	db := a.db.ForUser(user.ID)
	if r.Method != "GET" && a.messageLimit() {
		fail(w, 429, "try_later")
		return true
	}
	switch {
	case path == "/api/v1/invitations" && r.Method == "GET":
		v, e := db.Invitations(r.Context())
		if e != nil {
			a.dbError(w, e)
		} else {
			write(w, 200, map[string]any{"invitations": v})
		}
	case path == "/api/v1/invitations" && r.Method == "POST":
		var input struct {
			Kind   string `json:"kind"`
			RoomID string `json:"room_id"`
		}
		if !body(w, r, &input) {
			return true
		}
		if input.Kind == "registration" && user.ID != "admin" {
			fail(w, 403, "admin_required")
			return true
		}
		code := secret()
		id := recordID()
		now := time.Now().Unix()
		e := db.CreateInvitation(r.Context(), id, digest(code), code, input.Kind, input.RoomID, now)
		if e != nil {
			a.messageError(w, e)
		} else {
			write(w, 201, map[string]any{"id": id, "code": code, "expires_at": now + 7*86400})
		}
	case strings.HasPrefix(path, "/api/v1/invitations/") && strings.HasSuffix(path, "/code") && r.Method == "GET":
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/api/v1/invitations/"), "/code")
		code, e := db.InvitationCode(r.Context(), id, time.Now().Unix())
		if e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, map[string]string{"code": code})
		}
	case strings.HasPrefix(path, "/api/v1/invitations/") && r.Method == "DELETE":
		e := db.RevokeInvitation(r.Context(), strings.TrimPrefix(path, "/api/v1/invitations/"), time.Now().Unix())
		if e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, map[string]bool{"revoked": true})
		}
	case path == "/api/v1/rooms/join" && r.Method == "POST":
		var input struct {
			Code string `json:"code"`
		}
		if !body(w, r, &input) {
			return true
		}
		if !validSecret(input.Code) {
			fail(w, 400, "invalid_invitation")
			return true
		}
		room, e := db.JoinRoom(r.Context(), digest(input.Code), time.Now().Unix())
		if e != nil {
			fail(w, 400, "invalid_invitation")
		} else {
			write(w, 200, map[string]string{"room_id": room})
		}
	default:
		fail(w, 405, "method_not_allowed")
	}
	return true
}
