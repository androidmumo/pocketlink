package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInvitationRecoveryAndIsolation(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "db")
	s, e := Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { s.Close() }()
	code := "private-code-test"
	hash := invitationDigest(code)
	if e = s.CreateInvitation(ctx, "invite", hash, code, "registration", "", 10); e != nil {
		t.Fatal(e)
	}
	var sealed []byte
	if e = s.db.QueryRow("SELECT encrypted_code FROM invitations WHERE id='invite'").Scan(&sealed); e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(sealed, []byte(code)) {
		t.Fatal("plaintext stored")
	}
	if _, e = s.ForUser("other").InvitationCode(ctx, "invite", 11); !errors.Is(e, ErrNotFound) {
		t.Fatal("cross-user code disclosed", e)
	}
	s.Close()
	s, e = Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	if got, e := s.InvitationCode(ctx, "invite", 11); e != nil || got != code {
		t.Fatal("restart recovery failed", e)
	}
	if _, e = s.InvitationCode(ctx, "invite", 10+7*86400); !errors.Is(e, ErrNotFound) {
		t.Fatal("expired code disclosed", e)
	}
	if e = s.RevokeInvitation(ctx, "invite", 12); e != nil {
		t.Fatal(e)
	}
	if _, e = s.InvitationCode(ctx, "invite", 13); !errors.Is(e, ErrNotFound) {
		t.Fatal("revoked code disclosed", e)
	}
	if e = s.CreateInvitation(ctx, "used", hash+"x", code, "registration", "", 10); !errors.Is(e, ErrInvalid) {
		t.Fatal("accepted mismatched hash")
	}
	if e = s.CreateInvitation(ctx, "used", invitationDigest("second"), "second", "registration", "", 10); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Register(ctx, "user", "alice", invitationDigest("second"), []byte("salt"), []byte("hash"), 12); e != nil {
		t.Fatal(e)
	}
	if _, e = s.InvitationCode(ctx, "used", 13); !errors.Is(e, ErrNotFound) {
		t.Fatal("used code disclosed", e)
	}
	if e = s.CreateInvitation(ctx, "legacy", "old-hash", "", "registration", "", 10); e != nil {
		t.Fatal(e)
	}
	if _, e = s.InvitationCode(ctx, "legacy", 11); !errors.Is(e, ErrNotFound) {
		t.Fatal("invented legacy code", e)
	}
	if e = s.CreateInvitation(ctx, "retained", invitationDigest("retained"), "retained", "registration", "", 10); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if e = os.Remove(path + ".invitation-key"); e != nil {
		t.Fatal(e)
	}
	if reopened, e := Open(ctx, path); e == nil {
		reopened.Close()
		t.Fatal("silently replaced lost recovery key")
	}
}
