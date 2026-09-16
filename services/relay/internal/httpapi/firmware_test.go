package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/androidmumo/pocketlink/services/relay/internal/testfirmware"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFirmwareAuthorizationAndPublish(t *testing.T) {
	signer := testfirmware.New(t)
	a, _ := fixture(t)
	cookie := login(t, a)
	manifest, image, m := signer.Release(t, 1)
	upload := func(origin string, c *http.Cookie) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		p, _ := mw.CreateFormFile("manifest", "manifest.json")
		p.Write(manifest)
		p, _ = mw.CreateFormFile("image", "pocketlink-ota.bin")
		p.Write(image)
		mw.Close()
		r := httptest.NewRequest("POST", "/api/v1/firmware", &buf)
		r.Header.Set("Content-Type", mw.FormDataContentType())
		r.Header.Set("Origin", origin)
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		return w
	}
	if w := upload(testOrigin, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := upload("https://evil.example", cookie); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := upload(testOrigin, cookie); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(t, a, "GET", "/api/v1/device/firmware/latest", "", "", cookie, ""); w.Code != 401 {
		t.Fatal("cookie accepted as device", w.Code)
	}
	code := pairing(t, a, cookie)
	w := request(t, a, "POST", "/api/v1/device/pair", "", `{"code":"`+code+`","sn":"ota-device"}`, nil, "")
	var pair struct{ Credential string }
	json.Unmarshal(w.Body.Bytes(), &pair)
	if w = request(t, a, "GET", "/api/v1/device/firmware/latest", "", "", nil, pair.Credential); w.Code != 404 {
		t.Fatal("unpublished", w.Code)
	}
	if w = request(t, a, "PUT", "/api/v1/firmware/channel", testOrigin, `{"sha256":"`+m.SHA256+`"}`, cookie, ""); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(t, a, "GET", "/api/v1/device/firmware/latest", "", "", nil, pair.Credential); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), manifest) {
		t.Fatal("manifest", w.Code)
	}
	if w = request(t, a, "GET", "/api/v1/device/firmware/"+m.SHA256, "", "", nil, pair.Credential); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), image) {
		t.Fatal("download", w.Code)
	}
	if w = request(t, a, "GET", "/api/v1/device/firmware/latest", testOrigin, "", nil, pair.Credential); w.Code != 403 {
		t.Fatal("browser origin", w.Code)
	}
	manifest[15] ^= 1
	if w = upload(testOrigin, cookie); w.Code != 400 {
		t.Fatal("bad signature", w.Code)
	}
}
