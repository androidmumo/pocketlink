package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	console "github.com/androidmumo/pocketlink/apps/console"
	"github.com/androidmumo/pocketlink/services/relay/internal/store"
)

const testOrigin = "https://console.example"
const testPassword = "a long test-only password"

func fixture(t *testing.T) (*API, *store.Store) {
	t.Helper()
	db, e := store.Open(context.Background(), filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	a, e := NewAuth(db, testOrigin, testPassword)
	if e != nil {
		t.Fatal(e)
	}
	api := New(db, "test")
	api.EnableAuth(a, console.Handler())
	return api, db
}
func request(t *testing.T, a http.Handler, method, path, origin, body string, cookie *http.Cookie, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func login(t *testing.T, a http.Handler) *http.Cookie {
	t.Helper()
	w := request(t, a, "POST", "/api/v1/auth/login", testOrigin, `{"password":"`+testPassword+`"}`, nil, "")
	if w.Code != 200 {
		t.Fatalf("login %d %s", w.Code, w.Body.String())
	}
	c := w.Result().Cookies()[0]
	if !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.Domain != "" {
		t.Fatal("unsafe cookie")
	}
	return c
}
func pairing(t *testing.T, a http.Handler, c *http.Cookie) string {
	t.Helper()
	w := request(t, a, "POST", "/api/v1/pairings", testOrigin, `{"name":"desk"}`, c, "")
	if w.Code != 201 {
		t.Fatalf("pairing %d %s", w.Code, w.Body.String())
	}
	var result struct{ Code string }
	json.Unmarshal(w.Body.Bytes(), &result)
	return result.Code
}
func TestAuthLifecycle(t *testing.T) {
	a, _ := fixture(t)
	for _, origin := range []string{"", "https://evil.example"} {
		if w := request(t, a, "POST", "/api/v1/auth/login", origin, `{"password":"`+testPassword+`"}`, nil, ""); w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
	if w := request(t, a, "POST", "/api/v1/auth/login", testOrigin, `{"password":"incorrect"}`, nil, ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	c := login(t, a)
	// Same-origin browser GETs do not carry Origin; they must work with the cookie.
	if w := request(t, a, "GET", "/api/v1/devices", "", "", c, ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := request(t, a, "GET", "/api/v1/devices", "https://evil.example", "", c, ""); w.Code != 403 {
		t.Fatal(w.Code)
	}
	code := pairing(t, a, c)
	payload := `{"code":"` + code + `","sn":"device-001"}`
	w := request(t, a, "POST", "/api/v1/device/pair", "", payload, nil, "")
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		Device     store.Device
		Credential string
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	if !validSecret(result.Credential) {
		t.Fatal("missing credential")
	}
	if request(t, a, "POST", "/api/v1/device/pair", "", payload, nil, "").Code != 401 {
		t.Fatal("pair code reused")
	}
	if request(t, a, "GET", "/api/v1/device/me", "", "", nil, result.Credential).Code != 200 {
		t.Fatal("device cannot authenticate")
	}
	if request(t, a, "GET", "/api/v1/devices", "", "", nil, result.Credential).Code != 401 {
		t.Fatal("device gained admin")
	}
	w = request(t, a, "GET", "/api/v1/devices", "", "", c, "")
	if strings.Contains(w.Body.String(), result.Credential) || strings.Contains(w.Body.String(), "credential_hash") {
		t.Fatal("credential disclosed")
	}
	if request(t, a, "DELETE", "/api/v1/devices/"+result.Device.ID, "", "", c, "").Code != 403 {
		t.Fatal("CSRF accepted")
	}
	if request(t, a, "DELETE", "/api/v1/devices/"+result.Device.ID, testOrigin, "", c, "").Code != 200 {
		t.Fatal("revoke failed")
	}
	if request(t, a, "GET", "/api/v1/device/me", "", "", nil, result.Credential).Code != 401 {
		t.Fatal("revoked credential accepted")
	}
	if request(t, a, "POST", "/api/v1/auth/logout", testOrigin, "", c, "").Code != 200 {
		t.Fatal("logout failed")
	}
	if request(t, a, "GET", "/api/v1/devices", "", "", c, "").Code != 401 {
		t.Fatal("logout session accepted")
	}
}
func TestConcurrentPairingAndExpiry(t *testing.T) {
	a, db := fixture(t)
	c := login(t, a)
	code := pairing(t, a, c)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for _, sn := range []string{"one", "two"} {
		wg.Add(1)
		go func(sn string) {
			defer wg.Done()
			w := request(t, a, "POST", "/api/v1/device/pair", "", `{"code":"`+code+`","sn":"`+sn+`"}`, nil, "")
			results <- w.Code
		}(sn)
	}
	wg.Wait()
	close(results)
	success := 0
	for status := range results {
		if status == 201 {
			success++
		} else if status != 401 {
			t.Fatal(status)
		}
	}
	if success != 1 {
		t.Fatal("non-atomic pairing")
	}
	expired := secret()
	if e := db.CreatePairing(context.Background(), digest(expired), "expired", time.Now().Add(-11*time.Minute).Unix()); e != nil {
		t.Fatal(e)
	}
	if request(t, a, "POST", "/api/v1/device/pair", "", `{"code":"`+expired+`","sn":"old"}`, nil, "").Code != 401 {
		t.Fatal("expired code accepted")
	}
	a.auth.mu.Lock()
	a.auth.sessions[digest(c.Value)] = time.Now().Add(-time.Second)
	a.auth.mu.Unlock()
	if request(t, a, "GET", "/api/v1/devices", "", "", c, "").Code != 401 {
		t.Fatal("expired session accepted")
	}
}
func TestLimitsAndConsole(t *testing.T) {
	a, _ := fixture(t)
	c := login(t, a)
	if w := request(t, a, "GET", "/", "", "", nil, ""); w.Code != 200 || !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatal("UI/security headers missing")
	}
	if request(t, a, "GET", "/assets/app.js", "", "", nil, "").Code != 200 {
		t.Fatal("missing script")
	}
	if request(t, a, "GET", "/assets/unknown", "", "", nil, "").Code != 404 {
		t.Fatal("exposed unknown asset")
	}
	if request(t, a, "POST", "/api/v1/pairings", testOrigin, `{"name":"`+strings.Repeat("x", 5000)+`"}`, c, "").Code != 400 {
		t.Fatal("unbounded request")
	}
	a.auth.mu.Lock()
	a.auth.attempts = 20
	a.auth.window = time.Now()
	a.auth.mu.Unlock()
	if request(t, a, "POST", "/api/v1/auth/login", testOrigin, `{}`, nil, "").Code != 429 {
		t.Fatal("login not limited")
	}
	a.Drain()
	if request(t, a, "GET", "/api/v1/devices", "", "", c, "").Code != 503 {
		t.Fatal("draining accepted business request")
	}
}
