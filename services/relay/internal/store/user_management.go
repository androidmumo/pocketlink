package store

import "context"

type ManagedUser struct {
	User
	DeviceCount int `json:"device_count"`
	RoomCount   int `json:"room_count"`
}

func (s *Store) Users(ctx context.Context) ([]ManagedUser, error) {
	if s.actorID() != "admin" {
		return nil, ErrDenied
	}
	rows, e := s.db.QueryContext(ctx, `SELECT u.id,u.username,u.created_at,u.disabled_at,
 (SELECT count(*) FROM devices d WHERE d.owner_id=u.id AND d.revoked_at IS NULL),
 (SELECT count(*) FROM rooms r WHERE r.owner_id=u.id AND r.archived_at IS NULL)
 FROM users u ORDER BY CASE WHEN u.id='admin' THEN 0 ELSE 1 END,u.created_at DESC,u.id LIMIT 256`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ManagedUser{}
	for rows.Next() {
		var u ManagedUser
		if e = rows.Scan(&u.ID, &u.Username, &u.CreatedAt, &u.DisabledAt, &u.DeviceCount, &u.RoomCount); e != nil {
			return nil, e
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
func (s *Store) ManageUser(ctx context.Context, id, action string, salt, hash []byte, now int64) error {
	if s.actorID() != "admin" || id == "admin" {
		return ErrDenied
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	query := ""
	var args []any
	switch action {
	case "disable":
		query = "UPDATE users SET disabled_at=COALESCE(disabled_at,?),auth_version=auth_version+1 WHERE id=?"
		args = []any{now, id}
	case "enable":
		query = "UPDATE users SET disabled_at=NULL,auth_version=auth_version+1 WHERE id=?"
		args = []any{id}
	case "logout":
		query = "UPDATE users SET auth_version=auth_version+1 WHERE id=?"
		args = []any{id}
	case "password":
		if len(salt) != 32 || len(hash) != 32 {
			return ErrInvalid
		}
		query = "UPDATE users SET password_salt=?,password_hash=?,auth_version=auth_version+1 WHERE id=?"
		args = []any{salt, hash, id}
	default:
		return ErrInvalid
	}
	result, e := tx.ExecContext(ctx, query, args...)
	if e != nil {
		return e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return ErrNotFound
	}
	if action == "disable" {
		if _, e = tx.ExecContext(ctx, "DELETE FROM pairings WHERE owner_id=?", id); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE invitations SET revoked_at=COALESCE(revoked_at,?),encrypted_code=NULL WHERE created_by=? AND used_at IS NULL", now, id); e != nil {
			return e
		}
	}
	return tx.Commit()
}
