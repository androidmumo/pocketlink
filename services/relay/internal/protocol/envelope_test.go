package protocol

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSharedVectors(t *testing.T) {
	data, err := os.ReadFile("../../../../packages/protocol/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		Control []struct {
			Name  string
			Wire  json.RawMessage
			Valid bool
		}
		Realtime []struct {
			Name  string
			Hex   string
			Valid bool
		}
	}
	if err = json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors.Control {
		t.Run(v.Name, func(t *testing.T) {
			_, err := DecodeControl(v.Wire)
			if (err == nil) != v.Valid {
				t.Fatalf("valid=%v error=%v", v.Valid, err)
			}
		})
	}
	for _, v := range vectors.Realtime {
		t.Run(v.Name, func(t *testing.T) {
			wire, err := hex.DecodeString(v.Hex)
			if err != nil {
				t.Fatal(err)
			}
			f, err := DecodeFrame(wire)
			if (err == nil) != v.Valid {
				t.Fatalf("valid=%v error=%v", v.Valid, err)
			}
			if err == nil {
				wire[32] ^= 255
				if f.Payload[0] == wire[32] {
					t.Fatal("payload aliases receive buffer")
				}
			}
		})
	}
}
func TestMalformedControl(t *testing.T) {
	valid := `{"version":1,"type":"hello","request_id":"r1","namespace":"intercom","payload":{}}`
	for _, wire := range []string{
		valid + valid,
		strings.Replace(valid, `"version":1`, `"version":1,"version":1`, 1),
		strings.Replace(valid, `"payload":{}`, `"payload":{"a":1,"a":2}`, 1),
		strings.Replace(valid, `"payload":{}`, `"payload":null`, 1),
		strings.Replace(valid, `"payload":{}`, `"sender_id":"spoof","payload":{}`, 1),
		strings.Repeat(" ", MaxControlBytes) + valid,
		string([]byte{0xff}),
	} {
		if _, err := DecodeControl([]byte(wire)); err == nil {
			t.Fatal("malformed message accepted")
		}
	}
}
func TestTextBounds(t *testing.T) {
	for _, text := range []string{"hello", strings.Repeat("中", 200), "line\nnext"} {
		if ValidateText(text) != nil {
			t.Fatal("valid text rejected")
		}
	}
	for _, text := range []string{"", strings.Repeat("中", 201), "nul\x00", string([]byte{0xff})} {
		if ValidateText(text) == nil {
			t.Fatal("invalid text accepted")
		}
	}
}
func FuzzControl(f *testing.F) {
	f.Add([]byte(`{"version":1,"type":"hello","request_id":"r1","namespace":"intercom","payload":{}}`))
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = DecodeControl(data) })
}
func FuzzFrame(f *testing.F) {
	f.Add([]byte("PL"))
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = DecodeFrame(data) })
}
