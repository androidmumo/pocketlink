// Package firmware verifies signed OTA metadata independently of HTTP/storage.
package firmware

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"regexp"
)

const MaxImage = 0x300000
const MaxManifest = 4096

//go:embed trust.pem
var Trust []byte
var versionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Metadata struct {
	Format   int    `json:"format"`
	Board    string `json:"board"`
	App      string `json:"app"`
	Version  string `json:"version"`
	Sequence int64  `json:"sequence"`
	Size     int    `json:"size"`
	SHA256   string `json:"sha256"`
}
type Envelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

var ErrInvalid = errors.New("invalid signed firmware")

func decode(raw []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return ErrInvalid
	}
	if d.Decode(new(any)) != io.EOF {
		return ErrInvalid
	}
	return nil
}
func PublicKey() (*ecdsa.PublicKey, error) {
	b, _ := pem.Decode(Trust)
	if b == nil {
		return nil, ErrInvalid
	}
	key, e := x509.ParsePKIXPublicKey(b.Bytes)
	if e != nil {
		return nil, e
	}
	k, ok := key.(*ecdsa.PublicKey)
	if !ok || k.Curve != elliptic.P256() {
		return nil, ErrInvalid
	}
	return k, nil
}
func Parse(raw []byte) (Metadata, error) {
	var m Metadata
	var envelope Envelope
	if len(raw) > MaxManifest || decode(raw, &envelope) != nil {
		return m, ErrInvalid
	}
	payload, e := base64.StdEncoding.Strict().DecodeString(envelope.Payload)
	if e != nil || len(payload) > 1024 {
		return m, ErrInvalid
	}
	signature, e := base64.StdEncoding.Strict().DecodeString(envelope.Signature)
	if e != nil {
		return m, ErrInvalid
	}
	key, e := PublicKey()
	if e != nil {
		return m, e
	}
	hash := sha256.Sum256(payload)
	if !ecdsa.VerifyASN1(key, hash[:], signature) || decode(payload, &m) != nil {
		return m, ErrInvalid
	}
	if m.Format != 1 || m.Board != "ai-passport-esp32c3" || m.App != "pocketlink" || !versionPattern.MatchString(m.Version) || m.Sequence < 1 || m.Sequence > 2147483647 || m.Size < 288 || m.Size > MaxImage || !digestPattern.MatchString(m.SHA256) {
		return m, ErrInvalid
	}
	return m, nil
}
func ValidateImage(m Metadata, image []byte) error {
	hash := sha256.Sum256(image)
	// ESP app descriptor follows the image header and first segment header.
	// This rejects merged/bootloader images. ESP-IDF performs full validation on device.
	if len(image) != m.Size || len(image) < 288 || image[0] != 0xe9 || binary.LittleEndian.Uint16(image[12:14]) != 5 || binary.LittleEndian.Uint32(image[32:36]) != 0xabcd5432 || hex.EncodeToString(hash[:]) != m.SHA256 {
		return ErrInvalid
	}
	return nil
}
