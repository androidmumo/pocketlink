package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Keep the recovery key separate from SQLite and include it in data-directory backups.
func invitationKey(ctx context.Context, db *sql.DB, path string) ([]byte, error) {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		var count int
		if err = db.QueryRowContext(ctx, "SELECT count(*) FROM invitations WHERE encrypted_code IS NOT NULL").Scan(&count); err != nil {
			return nil, err
		}
		if count != 0 {
			return nil, fmt.Errorf("invitation recovery key missing; restore matching data-directory backup")
		}
		key := make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
		f, err := os.CreateTemp(filepath.Dir(path), ".invitation-key-*")
		if err != nil {
			return nil, err
		}
		defer os.Remove(f.Name())
		if _, err = f.Write(key); err != nil {
			f.Close()
			return nil, err
		}
		if err = f.Sync(); err != nil {
			f.Close()
			return nil, err
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
		// Publish a complete key atomically; another opener may have won the race.
		if err = os.Link(f.Name(), path); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		dir, syncErr := os.Open(filepath.Dir(path))
		if syncErr != nil {
			return nil, syncErr
		}
		syncErr = dir.Sync()
		closeErr := dir.Close()
		if syncErr != nil {
			return nil, syncErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	} else if err != nil {
		return nil, err
	}
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || st.Size() != 32 {
		return nil, fmt.Errorf("invalid invitation key file or permissions")
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid invitation key")
	}
	return key, nil
}
func invitationDigest(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}
func (s *Store) invitationCipher() (cipher.AEAD, error) {
	b, e := aes.NewCipher(s.invitationKey)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCMWithRandomNonce(b)
}
func (s *Store) sealInvitation(id, code string) ([]byte, error) {
	a, e := s.invitationCipher()
	if e != nil {
		return nil, e
	}
	return a.Seal(nil, nil, []byte(code), []byte(id)), nil
}
func (s *Store) InvitationCode(ctx context.Context, id string, now int64) (string, error) {
	var encrypted []byte
	e := s.db.QueryRowContext(ctx, `SELECT encrypted_code FROM invitations WHERE id=? AND created_by=? AND used_at IS NULL AND revoked_at IS NULL AND expires_at>? AND encrypted_code IS NOT NULL AND (kind='registration' OR EXISTS(SELECT 1 FROM rooms WHERE id=invitations.room_id AND archived_at IS NULL))`, id, s.actorID(), now).Scan(&encrypted)
	if errors.Is(e, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if e != nil {
		return "", e
	}
	a, e := s.invitationCipher()
	if e != nil {
		return "", e
	}
	raw, e := a.Open(nil, nil, encrypted, []byte(id))
	if e != nil {
		return "", fmt.Errorf("cannot decrypt invitation code")
	}
	return string(raw), nil
}
