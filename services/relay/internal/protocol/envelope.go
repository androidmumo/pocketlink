package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"unicode/utf8"
)

const Version = 1
const MaxControlBytes = 8192
const MaxRealtimeBytes = 1100
const HeaderSize = 32

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,63}$`)
var ErrInvalid = errors.New("invalid protocol message")

type Envelope struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	RequestID string          `json:"request_id"`
	Namespace string          `json:"namespace"`
	RoomID    string          `json:"room_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

// DecodeControl rejects unknown envelope fields and duplicate keys at any level.
// Business payload schemas and session authorization are separate checks.
func DecodeControl(data []byte) (Envelope, error) {
	var e Envelope
	if len(data) == 0 || len(data) > MaxControlBytes || !utf8.Valid(data) {
		return e, ErrInvalid
	}
	if err := uniqueJSON(data); err != nil {
		return e, ErrInvalid
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		return Envelope{}, ErrInvalid
	}
	if e.Version != Version {
		return Envelope{}, fmt.Errorf("unsupported_version")
	}
	if !identifier.MatchString(e.Type) || !identifier.MatchString(e.RequestID) || !identifier.MatchString(e.Namespace) {
		return Envelope{}, ErrInvalid
	}
	if e.RoomID != "" && !identifier.MatchString(e.RoomID) {
		return Envelope{}, ErrInvalid
	}
	payload := bytes.TrimSpace(e.Payload)
	if len(payload) < 2 || payload[0] != '{' {
		return Envelope{}, ErrInvalid
	}
	return e, nil
}
func uniqueJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 16 {
			return ErrInvalid
		}
		tok, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				s, ok := key.(string)
				if !ok || seen[s] {
					return ErrInvalid
				}
				seen[s] = true
				if err = walk(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		default:
			return ErrInvalid
		}
		_, err = d.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrInvalid
	}
	return nil
}

func ValidateText(text string) error {
	if !utf8.ValidString(text) || len(text) == 0 || len(text) > 2048 || utf8.RuneCountInString(text) > 200 {
		return ErrInvalid
	}
	for _, r := range text {
		if (r < 32 && r != '\n' && r != '\t') || r == 127 {
			return ErrInvalid
		}
	}
	return nil
}

// Frame is a plaintext record inside DTLS, or one binary WSS message.
// Stream and epoch are issued by the authenticated control service.
type Frame struct {
	Kind      uint8
	StreamID  uint32
	Epoch     uint32
	Sequence  uint32
	Timestamp uint64
	TTL       uint16
	Payload   []byte
}

func DecodeFrame(data []byte) (Frame, error) {
	if len(data) < HeaderSize || len(data) > MaxRealtimeBytes || string(data[:2]) != "PL" || data[2] != Version {
		return Frame{}, ErrInvalid
	}
	if data[3] != 1 && data[3] != 2 {
		return Frame{}, ErrInvalid
	}
	if binary.BigEndian.Uint32(data[28:32]) != 0 {
		return Frame{}, ErrInvalid
	}
	size := int(binary.BigEndian.Uint16(data[26:28]))
	if size == 0 || size != len(data)-HeaderSize {
		return Frame{}, ErrInvalid
	}
	f := Frame{Kind: data[3], StreamID: binary.BigEndian.Uint32(data[4:8]), Epoch: binary.BigEndian.Uint32(data[8:12]), Sequence: binary.BigEndian.Uint32(data[12:16]), Timestamp: binary.BigEndian.Uint64(data[16:24]), TTL: binary.BigEndian.Uint16(data[24:26]), Payload: append([]byte(nil), data[32:]...)}
	if f.StreamID == 0 || f.Epoch == 0 || f.TTL == 0 {
		return Frame{}, ErrInvalid
	}
	return f, nil
}
