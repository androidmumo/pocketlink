// Package testfirmware supplies ephemeral signing keys to nonparallel tests only.
package testfirmware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"github.com/androidmumo/pocketlink/services/relay/internal/firmware"
	"testing"
)

type Signer struct{ key *ecdsa.PrivateKey }

func New(t *testing.T) *Signer {
	t.Helper()
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	der, e := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if e != nil {
		t.Fatal(e)
	}
	old := firmware.Trust
	firmware.Trust = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	t.Cleanup(func() { firmware.Trust = old })
	return &Signer{key}
}
func (s *Signer) Release(t *testing.T, seq int64) ([]byte, []byte, firmware.Metadata) {
	t.Helper()
	image := make([]byte, 288)
	image[0] = 0xe9
	image[1] = 1
	binary.LittleEndian.PutUint16(image[12:14], 5)
	binary.LittleEndian.PutUint32(image[32:36], 0xabcd5432)
	image[287] = byte(seq)
	h := sha256.Sum256(image)
	m := firmware.Metadata{Format: 1, Board: "ai-passport-esp32c3", App: "pocketlink", Version: "1.0.0", Sequence: seq, Size: len(image), SHA256: hex.EncodeToString(h[:])}
	return s.Sign(t, m), image, m
}
func (s *Signer) Sign(t *testing.T, m firmware.Metadata) []byte {
	t.Helper()
	payload, _ := json.Marshal(m)
	h := sha256.Sum256(payload)
	sig, e := ecdsa.SignASN1(rand.Reader, s.key, h[:])
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(firmware.Envelope{Payload: base64.StdEncoding.EncodeToString(payload), Signature: base64.StdEncoding.EncodeToString(sig)})
	return raw
}
