package voice

import (
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"
)

func peer(room string) *Peer {
	return &Peer{Room: room, Name: "speaker", Out: make(chan Packet, 16), Stop: func() {}}
}
func control(t *testing.T, p *Peer, kind string) map[string]any {
	t.Helper()
	for i := 0; i < 16; i++ {
		select {
		case packet := <-p.Out:
			var v map[string]any
			if json.Unmarshal(packet.Data, &v) == nil && v["type"] == kind {
				return v
			}
		default:
			t.Fatalf("missing %s", kind)
		}
	}
	return nil
}
func TestFloorIsolationAndRelease(t *testing.T) {
	h := New()
	now := time.Unix(100, 0)
	a, b, c := peer("r"), peer("r"), peer("other")
	for _, p := range []*Peer{a, b, c} {
		if !h.Join(p, now) {
			t.Fatal("join")
		}
		control(t, p, "floor")
	}
	h.Request(a, 1, now)
	grant := control(t, a, "grant")
	stream := uint32(grant["stream"].(float64))
	h.Request(b, 2, now)
	control(t, b, "busy")
	frame := make([]byte, FrameBytes)
	binary.BigEndian.PutUint32(frame, stream)
	binary.BigEndian.PutUint32(frame[4:], 1)
	if h.Audio(b, frame, now) {
		t.Fatal("nonholder sent")
	}
	if !h.Audio(a, frame, now) {
		t.Fatal("holder rejected")
	}
	select {
	case packet := <-c.Out:
		t.Fatalf("cross-room disclosure %v", packet.Binary)
	default:
	}
	packet := <-b.Out
	if !packet.Binary {
		t.Fatal("audio not relayed")
	}
	if h.Audio(a, frame, now) {
		t.Fatal("replay accepted")
	}
	h.Release(b, stream, now)
	h.Request(b, 3, now)
	control(t, b, "busy")
	h.Release(a, stream, now)
	h.Request(b, 4, now)
	next := control(t, b, "grant")
	if next["stream"] == grant["stream"] {
		t.Fatal("stream reused")
	}
	h.Leave(b, now)
	h.Request(a, 5, now)
	control(t, a, "grant")
}
func TestExpiryAndBounds(t *testing.T) {
	h := New()
	now := time.Unix(100, 0)
	a := peer("r")
	h.Join(a, now)
	control(t, a, "floor")
	h.Request(a, 1, now)
	g := control(t, a, "grant")
	h.Tick(now.Add(Lease))
	v := control(t, a, "floor")
	if v["stream"].(float64) != 0 {
		t.Fatal("lease stuck")
	}
	h.Request(a, 2, now.Add(Lease))
	if control(t, a, "grant")["stream"] == g["stream"] {
		t.Fatal("old stream reused")
	}
	stopped := false
	b := peer("r")
	b.Out = make(chan Packet, 1)
	b.Stop = func() { stopped = true }
	h.Join(b, now)
	h.Release(a, uint32(h.next), now)
	if !stopped {
		t.Fatal("slow reader unbounded")
	}
}
