// Package voice implements bounded, ephemeral half-duplex PCM rooms.
package voice

import (
	"encoding/binary"
	"encoding/json"
	"sync"
	"time"
)

const FrameBytes = 648 // stream u32 BE, sequence u32 BE, 320 signed little-endian samples
const Lease = 30 * time.Second

type Packet struct {
	Binary bool
	Data   []byte
	At     time.Time
}
type Peer struct {
	Room, Name string
	Out        chan Packet
	Stop       func()
}
type floor struct {
	owner         *Peer
	stream, seq   uint32
	until, window time.Time
	frames        int
}
type Hub struct {
	mu     sync.Mutex
	peers  map[*Peer]bool
	floors map[string]*floor
	next   uint32
}

func New() *Hub { return &Hub{peers: map[*Peer]bool{}, floors: map[string]*floor{}} }
func (h *Hub) send(p *Peer, v any, now time.Time) {
	b, _ := json.Marshal(v)
	h.queue(p, Packet{Data: b, At: now})
}
func (h *Hub) queue(p *Peer, v Packet) {
	select {
	case p.Out <- v:
	default:
		p.Stop()
	}
}
func (h *Hub) state(room string, now time.Time) {
	f := h.floors[room]
	v := map[string]any{"type": "floor", "stream": 0, "sender": ""}
	if f != nil {
		v["stream"] = f.stream
		v["sender"] = f.owner.Name
	}
	for p := range h.peers {
		if p.Room == room {
			h.send(p, v, now)
		}
	}
}
func (h *Hub) expire(now time.Time) {
	for room, f := range h.floors {
		if !now.Before(f.until) {
			delete(h.floors, room)
			h.state(room, now)
		}
	}
}
func (h *Hub) Join(p *Peer, now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.peers) >= 32 {
		return false
	}
	h.peers[p] = true
	h.expire(now)
	f := h.floors[p.Room]
	v := map[string]any{"type": "floor", "stream": 0, "sender": ""}
	if f != nil {
		v["stream"] = f.stream
		v["sender"] = f.owner.Name
	}
	h.send(p, v, now)
	return true
}
func (h *Hub) Leave(p *Peer, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.peers, p)
	if f := h.floors[p.Room]; f != nil && f.owner == p {
		delete(h.floors, p.Room)
		h.state(p.Room, now)
	}
}
func (h *Hub) Tick(now time.Time) { h.mu.Lock(); defer h.mu.Unlock(); h.expire(now) }
func (h *Hub) Request(p *Peer, request uint32, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.peers[p] {
		return
	}
	h.expire(now)
	if h.floors[p.Room] != nil {
		h.send(p, map[string]any{"type": "busy", "request_id": request}, now)
		return
	}
	h.next++
	if h.next == 0 {
		h.next++
	}
	f := &floor{owner: p, stream: h.next, until: now.Add(Lease), window: now}
	h.floors[p.Room] = f
	h.state(p.Room, now)
	h.send(p, map[string]any{"type": "grant", "request_id": request, "stream": f.stream, "lease_ms": 30000}, now)
}
func (h *Hub) Release(p *Peer, stream uint32, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if f := h.floors[p.Room]; f != nil && f.owner == p && f.stream == stream {
		delete(h.floors, p.Room)
		h.state(p.Room, now)
	}
}
func (h *Hub) Audio(p *Peer, b []byte, now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expire(now)
	if len(b) != FrameBytes {
		return false
	}
	f := h.floors[p.Room]
	if f == nil || f.owner != p || binary.BigEndian.Uint32(b) != f.stream {
		return false
	}
	seq := binary.BigEndian.Uint32(b[4:])
	if seq <= f.seq || seq-f.seq > 100 {
		return false
	}
	if now.Sub(f.window) >= time.Second {
		f.window = now
		f.frames = 0
	}
	f.frames++
	if f.frames > 60 {
		return false
	}
	f.seq = seq
	packet := Packet{Binary: true, Data: append([]byte(nil), b...), At: now}
	for target := range h.peers {
		if target != p && target.Room == p.Room {
			h.queue(target, packet)
		}
	}
	return true
}
