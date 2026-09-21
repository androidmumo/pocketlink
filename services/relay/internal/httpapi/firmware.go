package httpapi

import (
	"github.com/androidmumo/pocketlink/services/relay/internal/firmware"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var firmwareHash = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Bound simultaneous image copies independently of the small normal API bodies.
var firmwareTransfers = make(chan struct{}, 2)

func (a *Auth) firmwareHTTP(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/device/firmware/") {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return true
		}
		hash := credential(r)
		if hash == "" {
			fail(w, 401, "invalid_credentials")
			return true
		}
		if _, e := a.db.DeviceByCredential(r.Context(), hash); e != nil {
			a.dbError(w, e)
			return true
		}
		var data []byte
		var e error
		if path == "/api/v1/device/firmware/latest" {
			data, e = a.db.FirmwareManifest(r.Context())
			w.Header().Set("Content-Type", "application/json")
		} else {
			id := strings.TrimPrefix(path, "/api/v1/device/firmware/")
			if !firmwareHash.MatchString(id) {
				fail(w, 400, "invalid_request")
				return true
			}
			select {
			case firmwareTransfers <- struct{}{}:
				defer func() { <-firmwareTransfers }()
			default:
				fail(w, 429, "try_later")
				return true
			}
			data, e = a.db.FirmwareImage(r.Context(), id)
			w.Header().Set("Content-Type", "application/octet-stream")
			_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(2 * time.Minute))
		}
		if e != nil {
			a.messageError(w, e)
			return true
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(200)
		_, _ = w.Write(data)
		return true
	}
	if path != "/api/v1/firmware" && !strings.HasPrefix(path, "/api/v1/firmware/") {
		return false
	}
	user, ok := a.principal(r)
	if !ok {
		fail(w, 401, "login_required")
		return true
	}
	if user.ID != "admin" {
		fail(w, 403, "admin_required")
		return true
	}
	switch {
	case path == "/api/v1/firmware" && r.Method == "GET":
		releases, e := a.db.FirmwareReleases(r.Context())
		if e != nil {
			a.dbError(w, e)
		} else {
			write(w, 200, map[string]any{"releases": releases})
		}
	case path == "/api/v1/firmware" && r.Method == "POST":
		if a.messageLimit() {
			fail(w, 429, "try_later")
			return true
		}
		select {
		case firmwareTransfers <- struct{}{}:
			defer func() { <-firmwareTransfers }()
		default:
			fail(w, 429, "try_later")
			return true
		}
		_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(2 * time.Minute))
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(2 * time.Minute))
		r.Body = http.MaxBytesReader(w, r.Body, firmware.MaxImage+8192)
		reader, e := r.MultipartReader()
		if e != nil {
			fail(w, 400, "invalid_firmware")
			return true
		}
		part, e := reader.NextPart()
		if e != nil || part.FormName() != "manifest" {
			fail(w, 400, "invalid_firmware")
			return true
		}
		manifest, e := io.ReadAll(io.LimitReader(part, firmware.MaxManifest+1))
		if e != nil {
			fail(w, 400, "invalid_firmware")
			return true
		}
		if _, e = firmware.Parse(manifest); e != nil {
			fail(w, 400, "invalid_firmware")
			return true
		}
		part, e = reader.NextPart()
		if e != nil || part.FormName() != "image" {
			fail(w, 400, "invalid_firmware")
			return true
		}
		image, e := io.ReadAll(io.LimitReader(part, firmware.MaxImage+1))
		if e != nil || len(image) > firmware.MaxImage {
			fail(w, 400, "invalid_firmware")
			return true
		}
		if _, e = reader.NextPart(); e != io.EOF {
			fail(w, 400, "invalid_firmware")
			return true
		}
		if e = a.db.AddFirmware(r.Context(), manifest, image, time.Now().Unix()); e != nil {
			a.messageError(w, e)
		} else {
			write(w, 201, map[string]bool{"stored": true})
		}
	case path == "/api/v1/firmware/channel" && r.Method == "PUT":
		var input struct {
			SHA256 string `json:"sha256"`
		}
		if !body(w, r, &input) {
			return true
		}
		if input.SHA256 != "" && !firmwareHash.MatchString(input.SHA256) {
			fail(w, 400, "invalid_request")
			return true
		}
		if e := a.db.PublishFirmware(r.Context(), input.SHA256); e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, map[string]bool{"published": input.SHA256 != ""})
		}
	case r.Method == "DELETE":
		hash := strings.TrimPrefix(path, "/api/v1/firmware/")
		if !firmwareHash.MatchString(hash) {
			fail(w, 400, "invalid_request")
			return true
		}
		if e := a.db.DeleteFirmware(r.Context(), hash); e != nil {
			a.messageError(w, e)
		} else {
			write(w, 200, map[string]bool{"deleted": true})
		}
	default:
		fail(w, 405, "method_not_allowed")
	}
	return true
}
