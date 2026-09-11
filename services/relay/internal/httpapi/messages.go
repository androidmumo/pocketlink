package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/androidmumo/pocketlink/services/relay/internal/store"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func recordID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "r" + hex.EncodeToString(b)
}
func credential(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	token := strings.TrimPrefix(h, "Bearer ")
	if !validSecret(token) {
		return ""
	}
	return digest(token)
}
func (a *Auth) messageLimit() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	if now.Sub(a.messageWindow) >= time.Minute {
		a.messageWindow = now
		a.messageAttempts = 0
	}
	if a.messageAttempts >= 120 {
		return true
	}
	a.messageAttempts++
	return false
}
func (a *Auth) messageError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, store.ErrInvalid):
		fail(w, 400, "invalid_request")
	case errors.Is(e, store.ErrNotFound):
		fail(w, 404, "not_found")
	case errors.Is(e, store.ErrConflict):
		fail(w, 409, "idempotency_conflict")
	case errors.Is(e, store.ErrEmptyRoom):
		fail(w, 409, "empty_room")
	default:
		a.dbError(w, e)
	}
}
func (a *Auth) messageHTTP(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path == "/api/v1/device/inbox" || path == "/api/v1/device/ack" || path == "/api/v1/device/stream" {
		hash := credential(r)
		if hash == "" {
			fail(w, 401, "invalid_credentials")
			return true
		}
		d, e := a.db.DeviceByCredential(r.Context(), hash)
		if e != nil {
			a.dbError(w, e)
			return true
		}
		switch {
		case path == "/api/v1/device/inbox" && r.Method == "GET":
			messages, e := a.db.Pending(r.Context(), hash)
			if e != nil {
				a.messageError(w, e)
			} else {
				write(w, 200, map[string]any{"messages": messages})
			}
		case path == "/api/v1/device/ack" && r.Method == "POST":
			var v struct {
				MessageID int64  `json:"message_id"`
				State     string `json:"state"`
			}
			if !body(w, r, &v) {
				return true
			}
			if e := a.db.Acknowledge(r.Context(), hash, v.MessageID, v.State, time.Now().Unix()); e != nil {
				a.messageError(w, e)
			} else {
				write(w, 200, map[string]bool{"acknowledged": true})
			}
		case path == "/api/v1/device/stream" && r.Method == "GET":
			a.stream(w, r, d.ID, hash)
		default:
			fail(w, 405, "method_not_allowed")
		}
		return true
	}
	if path != "/api/v1/rooms" && !strings.HasPrefix(path, "/api/v1/rooms/") && !strings.HasPrefix(path, "/api/v1/messages/") {
		return false
	}
	if !a.admin(r) {
		fail(w, 401, "login_required")
		return true
	}
	if r.Method != "GET" && a.messageLimit() {
		w.Header().Set("Retry-After", "60")
		fail(w, 429, "try_later")
		return true
	}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	now := time.Now().Unix()
	switch {
	case path == "/api/v1/rooms" && r.Method == "GET":
		rooms, e := a.db.Rooms(r.Context())
		if e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, map[string]any{"rooms": rooms})
		}
	case path == "/api/v1/rooms" && r.Method == "POST":
		var v struct {
			Name string `json:"name"`
		}
		if !body(w, r, &v) {
			break
		}
		v.Name = strings.TrimSpace(v.Name)
		id := recordID()
		if e := a.db.CreateRoom(r.Context(), id, v.Name, now); e != nil {
			a.messageError(w, e)
		} else {
			write(w, 201, store.Room{ID: id, Name: v.Name, CreatedAt: now})
		}
	case len(parts) == 2 && parts[0] == "rooms" && r.Method == "DELETE":
		if e := a.db.ArchiveRoom(r.Context(), parts[1], now); e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, map[string]bool{"archived": true})
		}
	case len(parts) == 3 && parts[0] == "rooms" && parts[2] == "members" && r.Method == "GET":
		ids, e := a.db.Members(r.Context(), parts[1])
		if e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, map[string]any{"device_ids": ids})
		}
	case len(parts) == 4 && parts[0] == "rooms" && parts[2] == "members" && (r.Method == "PUT" || r.Method == "DELETE"):
		if e := a.db.SetMember(r.Context(), parts[1], parts[3], r.Method == "PUT", now); e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, map[string]bool{"updated": true})
		}
	case len(parts) == 3 && parts[0] == "rooms" && parts[2] == "messages" && r.Method == "POST":
		var v struct {
			RequestID string `json:"request_id"`
			Text      string `json:"text"`
		}
		if !body(w, r, &v) {
			break
		}
		if !serialPattern.MatchString(v.RequestID) {
			fail(w, 400, "invalid_request")
			break
		}
		m, e := a.db.SendText(r.Context(), parts[1], v.RequestID, v.Text, now)
		if e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, m)
		}
	case len(parts) == 3 && parts[0] == "rooms" && parts[2] == "messages" && r.Method == "GET":
		before := int64(0)
		var e error
		if v := r.URL.Query().Get("before"); v != "" {
			before, e = strconv.ParseInt(v, 10, 64)
			if e != nil || before <= 0 {
				fail(w, 400, "invalid_cursor")
				break
			}
		}
		ms, e := a.db.History(r.Context(), parts[1], before)
		if e != nil {
			a.messageError(w, e)
		} else {
			next := int64(0)
			if len(ms) == 20 {
				next = ms[len(ms)-1].ID
			}
			write(w, 200, map[string]any{"messages": ms, "next_before": next})
		}
	case len(parts) == 3 && parts[0] == "messages" && parts[2] == "receipts" && r.Method == "GET":
		id, e := strconv.ParseInt(parts[1], 10, 64)
		if e != nil || id <= 0 {
			fail(w, 400, "invalid_message")
			break
		}
		rs, e := a.db.MessageReceipts(r.Context(), id)
		if e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, map[string]any{"receipts": rs})
		}
	default:
		fail(w, 404, "not_found")
	}
	return true
}
