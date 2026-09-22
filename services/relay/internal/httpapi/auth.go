package httpapi

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/androidmumo/pocketlink/services/relay/internal/store"
	"github.com/androidmumo/pocketlink/services/relay/internal/voice"
)

const sessionCookie = "__Host-pocketlink"
const iterations = 600000

var serialPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)

type Auth struct {
	db                 *store.Store
	origin             string
	salt, passwordHash []byte
	mu                 sync.Mutex
	sessions           map[string]session
	window             time.Time
	attempts           int
	verifying          chan struct{}
	messageWindow      time.Time
	messageAttempts    int
	streams            map[string]*streamSession
	stopping           bool
	voice              *voice.Hub
}

// The bootstrap administrator retains its file-configured password.
// Sessions are bounded, memory-only and invalidated on restart/password rotation.
func NewAuth(db *store.Store, origin, password string) (*Auth, error) {
	if len(password) < 16 || len(password) > 256 {
		return nil, errors.New("administrator password must be 16..256 bytes")
	}
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	hash, err := pbkdf2.Key(sha256.New, password, salt, iterations, 32)
	if err != nil {
		return nil, err
	}
	return &Auth{db: db, origin: origin, salt: salt, passwordHash: hash, sessions: map[string]session{}, verifying: make(chan struct{}, 1), streams: map[string]*streamSession{}, voice: voice.New()}, nil
}
func secret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func validSecret(s string) bool {
	b, e := base64.RawURLEncoding.DecodeString(s)
	return e == nil && len(b) == 32 && base64.RawURLEncoding.EncodeToString(b) == s
}
func fail(w http.ResponseWriter, status int, code string) {
	write(w, status, map[string]string{"error": code})
}
func body(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		fail(w, 415, "json_required")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		fail(w, 400, "invalid_request")
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		fail(w, 400, "invalid_request")
		return false
	}
	return true
}
func (a *Auth) limited() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	if now.Sub(a.window) >= time.Minute {
		a.window = now
		a.attempts = 0
	}
	if a.attempts >= 20 {
		return true
	}
	a.attempts++
	return false
}
func (a *Auth) admin(r *http.Request) bool { u, ok := a.principal(r); return ok && u.ID == "admin" }
func (a *Auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// No forwarded header is trusted. Exact configured Origin is mandatory on
	// browser requests. Device endpoints use bearer credentials, never cookies.
	device := strings.HasPrefix(r.URL.Path, "/api/v1/device/")
	if !device && ((r.Method != "GET" && r.Header.Get("Origin") != a.origin) || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != a.origin) || (r.Header.Get("Sec-Fetch-Site") == "cross-site")) {
		fail(w, 403, "origin_denied")
		return
	}
	if device && r.Header.Get("Origin") != "" {
		fail(w, 403, "device_endpoint")
		return
	}
	if a.voiceHTTP(w, r) || a.accountHTTP(w, r) || a.firmwareHTTP(w, r) || a.messageHTTP(w, r) {
		return
	}
	switch {
	case r.URL.Path == "/api/v1/device/pair" && r.Method == "POST":
		if a.limited() {
			w.Header().Set("Retry-After", "60")
			fail(w, 429, "try_later")
			return
		}
		var req struct {
			Code string `json:"code"`
			SN   string `json:"sn"`
		}
		if !body(w, r, &req) {
			return
		}
		if !validSecret(req.Code) || !serialPattern.MatchString(req.SN) {
			fail(w, 400, "invalid_request")
			return
		}
		credential := secret()
		if err := a.db.ConsumePairing(r.Context(), digest(req.Code), secret(), req.SN, digest(credential), time.Now().Unix()); err != nil {
			a.dbError(w, err)
			return
		}
		d, err := a.db.DeviceByCredential(r.Context(), digest(credential))
		if err != nil {
			a.dbError(w, err)
			return
		}
		write(w, 201, map[string]any{"device": d, "credential": credential})
	case r.URL.Path == "/api/v1/device/me" && r.Method == "GET":
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || !validSecret(token) {
			fail(w, 401, "invalid_credentials")
			return
		}
		d, err := a.db.DeviceByCredential(r.Context(), digest(token))
		if err != nil {
			a.dbError(w, err)
			return
		}
		write(w, 200, d)
	default:
		user, ok := a.principal(r)
		if !ok {
			fail(w, 401, "login_required")
			return
		}
		db := a.db.ForUser(user.ID)
		switch {
		case r.URL.Path == "/api/v1/auth/me" && r.Method == "GET":
			write(w, 200, identity(user))
		case r.URL.Path == "/api/v1/auth/logout" && r.Method == "POST":
			c, _ := r.Cookie(sessionCookie)
			a.mu.Lock()
			delete(a.sessions, digest(c.Value))
			a.mu.Unlock()
			http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
			write(w, 200, map[string]bool{"authenticated": false})
		case r.URL.Path == "/api/v1/devices" && r.Method == "GET":
			ds, err := db.Devices(r.Context())
			if err != nil {
				a.dbError(w, err)
				return
			}
			write(w, 200, map[string]any{"devices": ds})
		case r.URL.Path == "/api/v1/pairings" && r.Method == "POST":
			var req struct {
				Name string `json:"name"`
			}
			if !body(w, r, &req) {
				return
			}
			req.Name = strings.TrimSpace(req.Name)
			if req.Name == "" || !utf8.ValidString(req.Name) || utf8.RuneCountInString(req.Name) > 40 || strings.ContainsFunc(req.Name, unicode.IsControl) {
				fail(w, 400, "invalid_name")
				return
			}
			code := secret()
			now := time.Now().Unix()
			if err := db.CreatePairing(r.Context(), digest(code), req.Name, now); err != nil {
				a.dbError(w, err)
				return
			}
			write(w, 201, map[string]any{"code": code, "expires_at": now + 600})
		case strings.HasPrefix(r.URL.Path, "/api/v1/devices/") && r.Method == "DELETE":
			id := strings.TrimPrefix(r.URL.Path, "/api/v1/devices/")
			if !validSecret(id) {
				fail(w, 400, "invalid_device")
				return
			}
			if err := db.RevokeDevice(r.Context(), id, time.Now().Unix()); err != nil {
				a.dbError(w, err)
				return
			}
			write(w, 200, map[string]bool{"revoked": true})
		default:
			fail(w, 404, "not_found")
		}
	}
}
func (a *Auth) dbError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrDenied) {
		fail(w, 401, "invalid_credentials")
	} else if errors.Is(err, store.ErrLimit) {
		fail(w, 409, "capacity_reached")
	} else {
		fail(w, 503, "unavailable")
	}
}
