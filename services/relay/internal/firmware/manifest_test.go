package firmware_test

import (
	"encoding/base64"
	"encoding/json"
	"github.com/androidmumo/pocketlink/services/relay/internal/firmware"
	"github.com/androidmumo/pocketlink/services/relay/internal/testfirmware"
	"testing"
)

func TestSignedManifestAndImage(t *testing.T) {
	signer := testfirmware.New(t)
	manifest, image, m := signer.Release(t, 1)
	if parsed, e := firmware.Parse(manifest); e != nil || parsed != m {
		t.Fatal(parsed, e)
	}
	if e := firmware.ValidateImage(m, image); e != nil {
		t.Fatal(e)
	}
	image[100] ^= 1
	if firmware.ValidateImage(m, image) == nil {
		t.Fatal("tampered image accepted")
	}
	var envelope firmware.Envelope
	json.Unmarshal(manifest, &envelope)
	payload, _ := base64.StdEncoding.DecodeString(envelope.Payload)
	payload[0] ^= 1
	envelope.Payload = base64.StdEncoding.EncodeToString(payload)
	tampered, _ := json.Marshal(envelope)
	if _, e := firmware.Parse(tampered); e == nil {
		t.Fatal("tampered payload accepted")
	}
	for _, change := range []func(*firmware.Metadata){func(m *firmware.Metadata) { m.Board = "another-board" }, func(m *firmware.Metadata) { m.Size = firmware.MaxImage + 1 }, func(m *firmware.Metadata) { m.Sequence = 0 }, func(m *firmware.Metadata) { m.App = "board-check" }} {
		bad := m
		change(&bad)
		if _, e := firmware.Parse(signer.Sign(t, bad)); e == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
	old := append([]byte{}, firmware.Trust...)
	testfirmware.New(t)
	if _, e := firmware.Parse(manifest); e == nil {
		t.Fatal("foreign key accepted")
	}
	firmware.Trust = old
}

func TestEmbeddedPublicKey(t *testing.T) {
	if _, err := firmware.PublicKey(); err != nil {
		t.Fatalf("invalid embedded trust key: %v", err)
	}
}
