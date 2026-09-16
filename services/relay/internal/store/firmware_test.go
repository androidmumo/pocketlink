package store

import (
	"bytes"
	"context"
	"errors"
	"github.com/androidmumo/pocketlink/services/relay/internal/testfirmware"
	"path/filepath"
	"testing"
)

func TestFirmwareLifecycleAndSequence(t *testing.T) {
	signer := testfirmware.New(t)
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "db")
	s, e := Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { s.Close() }()
	one, image, m := signer.Release(t, 10)
	if e = s.AddFirmware(ctx, one, image, 1); e != nil {
		t.Fatal(e)
	}
	if e = s.AddFirmware(ctx, one, image, 1); e != nil {
		t.Fatal("idempotency", e)
	}
	if _, e = s.FirmwareManifest(ctx); !errors.Is(e, ErrNotFound) {
		t.Fatal("unpublished firmware visible", e)
	}
	if e = s.PublishFirmware(ctx, m.SHA256); e != nil {
		t.Fatal(e)
	}
	if e = s.DeleteFirmware(ctx, m.SHA256); !errors.Is(e, ErrConflict) {
		t.Fatal("deleted active firmware", e)
	}
	if got, e := s.FirmwareImage(ctx, m.SHA256); e != nil || !bytes.Equal(got, image) {
		t.Fatal("image", e)
	}
	s.Close()
	s, e = Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	if got, e := s.FirmwareManifest(ctx); e != nil || !bytes.Equal(got, one) {
		t.Fatal("restart persistence", e)
	}
	if e = s.PublishFirmware(ctx, ""); e != nil {
		t.Fatal(e)
	}
	if _, e = s.FirmwareImage(ctx, m.SHA256); !errors.Is(e, ErrNotFound) {
		t.Fatal("withdrawn download", e)
	}
	if e = s.DeleteFirmware(ctx, m.SHA256); e != nil {
		t.Fatal(e)
	}
	if e = s.AddFirmware(ctx, one, image, 2); !errors.Is(e, ErrConflict) {
		t.Fatal("sequence reuse after delete", e)
	}
	bad, badImage, _ := signer.Release(t, 11)
	badImage[100] ^= 1
	if e = s.AddFirmware(ctx, bad, badImage, 3); !errors.Is(e, ErrInvalid) {
		t.Fatal("bad hash accepted", e)
	}
	for seq := int64(11); seq < 19; seq++ {
		raw, img, _ := signer.Release(t, seq)
		if e = s.AddFirmware(ctx, raw, img, 4); e != nil {
			t.Fatal(e)
		}
	}
	raw, img, _ := signer.Release(t, 19)
	if e = s.AddFirmware(ctx, raw, img, 5); !errors.Is(e, ErrLimit) {
		t.Fatal("capacity", e)
	}
}
