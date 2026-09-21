package store

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"github.com/androidmumo/pocketlink/services/relay/internal/firmware"
	"time"
)

type FirmwareRelease struct {
	SHA256      string `json:"sha256"`
	Version     string `json:"version"`
	Sequence    int64  `json:"sequence"`
	Size        int    `json:"size"`
	CreatedAt   int64  `json:"created_at"`
	PublishedAt *int64 `json:"published_at"`
	Active      bool   `json:"active"`
}

func (s *Store) AddFirmware(ctx context.Context, manifest, image []byte, now int64) error {
	m, e := firmware.Parse(manifest)
	if e != nil {
		return ErrInvalid
	}
	if firmware.ValidateImage(m, image) != nil {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var existing []byte
	e = tx.QueryRowContext(ctx, "SELECT manifest FROM firmware_releases WHERE sha256=?", m.SHA256).Scan(&existing)
	if e == nil {
		if bytes.Equal(existing, manifest) {
			return nil
		}
		return ErrConflict
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	var count int
	var latest int64
	if e = tx.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM firmware_releases),max_sequence FROM firmware_channel WHERE id=1").Scan(&count, &latest); e != nil {
		return e
	}
	if count >= 8 {
		return ErrLimit
	}
	if m.Sequence <= latest {
		return ErrConflict
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO firmware_releases(sha256,version,sequence,size,manifest,image,created_at) VALUES(?,?,?,?,?,?,?)", m.SHA256, m.Version, m.Sequence, m.Size, manifest, image, now)
	if e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE firmware_channel SET max_sequence=? WHERE id=1", m.Sequence); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) FirmwareReleases(ctx context.Context) ([]FirmwareRelease, error) {
	rows, e := s.db.QueryContext(ctx, "SELECT f.sha256,f.version,f.sequence,f.size,f.created_at,f.published_at,COALESCE(f.sha256=c.sha256,0) FROM firmware_releases f CROSS JOIN firmware_channel c ORDER BY f.sequence DESC")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	releases := []FirmwareRelease{}
	for rows.Next() {
		var r FirmwareRelease
		if e = rows.Scan(&r.SHA256, &r.Version, &r.Sequence, &r.Size, &r.CreatedAt, &r.PublishedAt, &r.Active); e != nil {
			return nil, e
		}
		releases = append(releases, r)
	}
	return releases, rows.Err()
}
func (s *Store) PublishFirmware(ctx context.Context, hash string) error {
	if hash == "" {
		_, e := s.db.ExecContext(ctx, "UPDATE firmware_channel SET sha256=NULL WHERE id=1")
		return e
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, "UPDATE firmware_releases SET published_at=? WHERE sha256=? AND (published_at IS NULL OR sha256!=COALESCE((SELECT sha256 FROM firmware_channel WHERE id=1),''))", time.Now().Unix(), hash)
	if e != nil {
		return e
	}
	result, e := tx.ExecContext(ctx, "UPDATE firmware_channel SET sha256=? WHERE id=1 AND EXISTS(SELECT 1 FROM firmware_releases WHERE sha256=?)", hash, hash)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}
func (s *Store) FirmwareManifest(ctx context.Context) ([]byte, error) {
	var raw []byte
	e := s.db.QueryRowContext(ctx, "SELECT manifest FROM firmware_releases WHERE sha256=(SELECT sha256 FROM firmware_channel WHERE id=1)").Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return raw, e
}
func (s *Store) FirmwareImage(ctx context.Context, hash string) ([]byte, error) {
	var raw []byte
	e := s.db.QueryRowContext(ctx, "SELECT image FROM firmware_releases WHERE sha256=? AND sha256=(SELECT sha256 FROM firmware_channel WHERE id=1)", hash).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return raw, e
}

func (s *Store) DeleteFirmware(ctx context.Context, hash string) error {
	result, e := s.db.ExecContext(ctx, "DELETE FROM firmware_releases WHERE sha256=? AND sha256!=COALESCE((SELECT sha256 FROM firmware_channel WHERE id=1),'')", hash)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrConflict
	}
	return nil
}
