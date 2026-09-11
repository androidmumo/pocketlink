package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestDeviceRestartAndRepair(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "db")
	s, e := Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CreatePairing(ctx, "code", "name", 100); e != nil {
		t.Fatal(e)
	}
	if e = s.ConsumePairing(ctx, "code", "id", "serial", "credential", 101); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = s.DeviceByCredential(ctx, "credential"); e != nil {
		t.Fatal("credential did not persist", e)
	}
	s.CreatePairing(ctx, "code2", "new name", 102)
	if e = s.ConsumePairing(ctx, "code2", "newid", "serial", "newcredential", 103); !errors.Is(e, ErrDenied) {
		t.Fatal("overwrote live serial", e)
	}
	if e = s.RevokeDevice(ctx, "id", 104); e != nil {
		t.Fatal(e)
	}
	if e = s.ConsumePairing(ctx, "code2", "newid", "serial", "newcredential", 105); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DeviceByCredential(ctx, "credential"); !errors.Is(e, ErrDenied) {
		t.Fatal("old credential accepted")
	}
	d, e := s.DeviceByCredential(ctx, "newcredential")
	if e != nil || d.ID != "id" || d.Name != "new name" {
		t.Fatal("repair failed", d, e)
	}
	if e = s.ConsumePairing(ctx, "code2", "other", "other", "other", 106); !errors.Is(e, ErrDenied) {
		t.Fatal("pairing reused")
	}
}
func TestPairingQuota(t *testing.T) {
	ctx := context.Background()
	s, e := Open(ctx, filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for i := range 32 {
		if e = s.CreatePairing(ctx, string(rune('a'+i)), "name", 100); e != nil {
			t.Fatal(e)
		}
	}
	if e = s.CreatePairing(ctx, "overflow", "name", 100); !errors.Is(e, ErrLimit) {
		t.Fatal("quota bypassed")
	}
	if e = s.CreatePairing(ctx, "afterexpiry", "name", 701); e != nil {
		t.Fatal("expired codes not pruned", e)
	}
}
