package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeStore struct{ err error }

func (f fakeStore) Ready(context.Context) error { return f.err }
func TestHealthAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name, method, path string
		dbErr              error
		drain              bool
		status             int
	}{
		{"live", "GET", "/health/live", nil, false, 200},
		{"ready", "GET", "/health/ready", nil, false, 200},
		{"storage failure", "GET", "/health/ready", errors.New("secret SQL path"), false, 503},
		{"draining", "GET", "/health/ready", nil, true, 503},
		{"still live", "GET", "/health/live", nil, true, 200},
		{"unknown", "GET", "/api/v1/devices", nil, false, 404},
		{"method", "POST", "/health/live", nil, false, 405},
		{"capabilities", "GET", "/api/v1/capabilities", nil, false, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := New(fakeStore{tc.dbErr}, "test")
			if tc.drain {
				a.Drain()
			}
			w := httptest.NewRecorder()
			a.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status {
				t.Fatal(w.Code)
			}
			if strings.Contains(w.Body.String(), "secret") {
				t.Fatal("error leaked")
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("health cached")
			}
			if tc.path == "/api/v1/capabilities" && !strings.Contains(w.Body.String(), `"enabled_features":[]`) {
				t.Fatal("advertises unimplemented features")
			}
		})
	}
}
