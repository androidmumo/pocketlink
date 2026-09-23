package store

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Store) VoiceIdentity(ctx context.Context, room, hash, user string) (string, error) {
	var name string
	var e error
	if hash != "" {
		e = s.db.QueryRowContext(ctx, `SELECT d.name FROM devices d JOIN room_members m ON m.device_id=d.id JOIN rooms r ON r.id=m.room_id WHERE d.credential_hash=? AND d.revoked_at IS NULL AND EXISTS(SELECT 1 FROM users enabled WHERE enabled.id=d.owner_id AND enabled.disabled_at IS NULL) AND r.id=? AND r.archived_at IS NULL AND (d.owner_id='admin' OR d.owner_id=r.owner_id OR EXISTS(SELECT 1 FROM room_users u WHERE u.room_id=r.id AND u.user_id=d.owner_id))`, hash, room).Scan(&name)
	} else {
		e = s.db.QueryRowContext(ctx, `SELECT u.username FROM users u JOIN rooms r ON r.id=? WHERE u.id=? AND u.disabled_at IS NULL AND r.archived_at IS NULL AND (u.id='admin' OR r.owner_id=u.id OR EXISTS(SELECT 1 FROM room_users m WHERE m.room_id=r.id AND m.user_id=u.id))`, room, user).Scan(&name)
	}
	if errors.Is(e, sql.ErrNoRows) {
		return "", ErrDenied
	}
	return name, e
}
func (s *Store) DeviceRooms(ctx context.Context, hash string) ([]Room, error) {
	d, e := s.DeviceByCredential(ctx, hash)
	if e != nil {
		return nil, e
	}
	rows, e := s.db.QueryContext(ctx, `SELECT r.id,r.name,r.owner_id,r.created_at,r.archived_at FROM rooms r JOIN room_members m ON m.room_id=r.id WHERE m.device_id=? AND r.archived_at IS NULL AND (?='admin' OR r.owner_id=? OR EXISTS(SELECT 1 FROM room_users u WHERE u.room_id=r.id AND u.user_id=?)) ORDER BY r.created_at,r.id`, d.ID, d.OwnerID, d.OwnerID, d.OwnerID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Room{}
	for rows.Next() {
		var r Room
		if e = rows.Scan(&r.ID, &r.Name, &r.OwnerID, &r.CreatedAt, &r.ArchivedAt); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
