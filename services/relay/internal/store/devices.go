package store

import (
	"context"
	"database/sql"
	"errors"
)

var ErrDenied = errors.New("credential unavailable")
var ErrLimit = errors.New("device or pairing limit reached")

type Device struct {
	ID        string `json:"id"`
	SN        string `json:"sn"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	RevokedAt *int64 `json:"revoked_at"`
}

// Pairing and device credentials are SHA-256 digests of random 256-bit secrets.
func (s *Store) CreatePairing(ctx context.Context, hash, name string, now int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "DELETE FROM pairings WHERE expires_at<=?", now); err != nil {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM pairings").Scan(&count); err != nil {
		return err
	}
	if count >= 32 {
		return ErrLimit
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO pairings(code_hash,name,expires_at) VALUES(?,?,?)", hash, name, now+600); err != nil {
		return err
	}
	return tx.Commit()
}

// ConsumePairing is atomic: concurrent exchanges cannot create two devices.
// A duplicate SN must be explicitly revoked before it can be paired again.
func (s *Store) ConsumePairing(ctx context.Context, codeHash, id, sn, credentialHash string, now int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name string
	if err = tx.QueryRowContext(ctx, "SELECT name FROM pairings WHERE code_hash=? AND expires_at>?", codeHash, now).Scan(&name); errors.Is(err, sql.ErrNoRows) {
		return ErrDenied
	} else if err != nil {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM devices WHERE sn=? AND revoked_at IS NULL", sn).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return ErrDenied
	}
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM devices WHERE revoked_at IS NULL").Scan(&count); err != nil {
		return err
	}
	if count >= 256 {
		return ErrLimit
	}
	// Bound retained identities as well as active devices.
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM devices WHERE sn<>?", sn).Scan(&count); err != nil {
		return err
	}
	if count >= 1024 {
		return ErrLimit
	}
	// Re-pairing retains the device identity but invalidates the old credential.
	_, err = tx.ExecContext(ctx, `INSERT INTO devices(id,sn,name,credential_hash,created_at) VALUES(?,?,?,?,?)
 ON CONFLICT(sn) DO UPDATE SET name=excluded.name,credential_hash=excluded.credential_hash,created_at=excluded.created_at,revoked_at=NULL`, id, sn, name, credentialHash, now)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM pairings WHERE code_hash=?", codeHash); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) DeviceByCredential(ctx context.Context, hash string) (Device, error) {
	var d Device
	err := s.db.QueryRowContext(ctx, "SELECT id,sn,name,created_at,revoked_at FROM devices WHERE credential_hash=? AND revoked_at IS NULL", hash).Scan(&d.ID, &d.SN, &d.Name, &d.CreatedAt, &d.RevokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrDenied
	}
	return d, err
}
func (s *Store) Devices(ctx context.Context) ([]Device, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,sn,name,created_at,revoked_at FROM devices ORDER BY created_at DESC,id LIMIT 1024")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Device{}
	for rows.Next() {
		var d Device
		if err = rows.Scan(&d.ID, &d.SN, &d.Name, &d.CreatedAt, &d.RevokedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}
func (s *Store) RevokeDevice(ctx context.Context, id string, now int64) error {
	r, err := s.db.ExecContext(ctx, "UPDATE devices SET revoked_at=COALESCE(revoked_at,?) WHERE id=?", now, id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrDenied
	}
	return nil
}
