package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func setupMessages(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "db")
	s, e := Open(context.Background(), path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		if e = s.CreatePairing(ctx, "code-"+id, id, 10); e != nil {
			t.Fatal(e)
		}
		if e = s.ConsumePairing(ctx, "code-"+id, id, id, "hash-"+id, 11); e != nil {
			t.Fatal(e)
		}
	}
	for _, id := range []string{"room1", "room2"} {
		if e = s.CreateRoom(ctx, id, id, 12); e != nil {
			t.Fatal(e)
		}
	}
	if e = s.SetMember(ctx, "room1", "a", true, 13); e != nil {
		t.Fatal(e)
	}
	if e = s.SetMember(ctx, "room2", "b", true, 13); e != nil {
		t.Fatal(e)
	}
	return s, path
}
func TestMessageIsolationAndMonotonicReceipts(t *testing.T) {
	s, _ := setupMessages(t)
	ctx := context.Background()
	m, e := s.SendText(ctx, "room1", "key", "hello", 14)
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"b", "c"} {
		ms, e := s.Pending(ctx, "hash-"+id)
		if e != nil || len(ms) != 0 {
			t.Fatal("cross-room read", e)
		}
		if e = s.Acknowledge(ctx, "hash-"+id, m.ID, "read", 15); !errors.Is(e, ErrDenied) {
			t.Fatal("forged receipt", e)
		}
	}
	ms, e := s.Pending(ctx, "hash-a")
	if e != nil || len(ms) != 1 {
		t.Fatal(ms, e)
	}
	if e = s.Acknowledge(ctx, "hash-a", m.ID, "read", 20); e != nil {
		t.Fatal(e)
	}
	if e = s.Acknowledge(ctx, "hash-a", m.ID, "received", 25); e != nil {
		t.Fatal(e)
	}
	rs, e := s.MessageReceipts(ctx, m.ID)
	if e != nil || len(rs) != 1 || *rs[0].ReadAt != 20 || *rs[0].DeliveredAt != 20 {
		t.Fatal("non-monotonic ack", rs, e)
	}
	ms, e = s.Pending(ctx, "hash-a")
	if e != nil || len(ms) != 0 {
		t.Fatal("acked message still pending")
	}
}
func TestMessageIdempotencyAndMembershipSnapshot(t *testing.T) {
	s, _ := setupMessages(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	ids := make(chan int64, 8)
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, e := s.SendText(ctx, "room1", "same-key", "same-body", 14)
			if e != nil {
				errs <- e
			} else {
				ids <- m.ID
			}
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	var id int64
	for v := range ids {
		if id != 0 && id != v {
			t.Fatal("duplicate message")
		}
		id = v
	}
	if _, e := s.SendText(ctx, "room1", "same-key", "other", 15); !errors.Is(e, ErrConflict) {
		t.Fatal("key collision accepted", e)
	}
	if e := s.SetMember(ctx, "room1", "b", true, 16); e != nil {
		t.Fatal(e)
	}
	ms, _ := s.Pending(ctx, "hash-b")
	if len(ms) != 0 {
		t.Fatal("new member received old message")
	}
	if e := s.SetMember(ctx, "room1", "a", false, 17); e != nil {
		t.Fatal(e)
	}
	if e := s.SetMember(ctx, "room1", "a", true, 18); e != nil {
		t.Fatal(e)
	}
	ms, _ = s.Pending(ctx, "hash-a")
	if len(ms) != 0 {
		t.Fatal("removed message revived")
	}
	if e := s.Acknowledge(ctx, "hash-a", id, "read", 19); !errors.Is(e, ErrDenied) {
		t.Fatal("withdrawn receipt changed")
	}
	m, e := s.SendText(ctx, "room1", "new-key", "new-body", 20)
	if e != nil {
		t.Fatal(e)
	}
	rs, _ := s.MessageReceipts(ctx, m.ID)
	if len(rs) != 2 {
		t.Fatal("wrong recipient snapshot")
	}
}
func TestMessageRestartArchiveAndRevoke(t *testing.T) {
	s, path := setupMessages(t)
	ctx := context.Background()
	m, e := s.SendText(ctx, "room1", "restart", "persist", 14)
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	ms, e := s.Pending(ctx, "hash-a")
	if e != nil || len(ms) != 1 || ms[0].ID != m.ID {
		t.Fatal("lost offline message", ms, e)
	}
	if e = s.RevokeDevice(ctx, "a", 15); e != nil {
		t.Fatal(e)
	}
	members, _ := s.Members(ctx, "room1")
	if len(members) != 0 {
		t.Fatal("revoked member retained")
	}
	s.CreatePairing(ctx, "repair", "a", 16)
	if e = s.ConsumePairing(ctx, "repair", "ignored", "a", "newhash", 17); e != nil {
		t.Fatal(e)
	}
	ms, e = s.Pending(ctx, "newhash")
	if e != nil || len(ms) != 0 {
		t.Fatal("old privilege restored")
	}
	if _, e = s.SendText(ctx, "room1", "empty", "text", 18); !errors.Is(e, ErrEmptyRoom) {
		t.Fatal(e)
	}
	if e = s.ArchiveRoom(ctx, "room1", 19); e != nil {
		t.Fatal(e)
	}
	if _, e = s.SendText(ctx, "room1", "archived", "text", 20); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}
	old, e := s.SendText(ctx, "room1", "restart", "persist", 21)
	if e != nil || old.ID != m.ID {
		t.Fatal("retry after archive failed", e)
	}
	history, e := s.History(ctx, "room1", 0)
	if e != nil || len(history) != 1 {
		t.Fatal("archive lost history")
	}
}
func TestMessagePagination(t *testing.T) {
	s, _ := setupMessages(t)
	ctx := context.Background()
	for i := range 25 {
		if _, e := s.SendText(ctx, "room1", fmt.Sprint(i), "text", int64(i)); e != nil {
			t.Fatal(e)
		}
	}
	first, e := s.History(ctx, "room1", 0)
	if e != nil || len(first) != 20 {
		t.Fatal(e)
	}
	second, e := s.History(ctx, "room1", first[19].ID)
	if e != nil || len(second) != 5 || second[0].ID >= first[19].ID {
		t.Fatal("cursor overlap")
	}
	if _, e = s.SendText(ctx, "room1", "blank", "  ", 30); !errors.Is(e, ErrInvalid) {
		t.Fatal("blank accepted")
	}
}
