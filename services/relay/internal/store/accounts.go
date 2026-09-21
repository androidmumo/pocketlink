package store

import (
	"context"
	"database/sql"
	"errors"
)

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	CreatedAt int64  `json:"created_at"`
}
type Invitation struct {
	ID        string  `json:"id"`
	Kind      string  `json:"kind"`
	RoomID    *string `json:"room_id"`
	CreatedAt int64   `json:"created_at"`
	ExpiresAt int64   `json:"expires_at"`
	UsedAt    *int64  `json:"used_at"`
	RevokedAt *int64  `json:"revoked_at"`
}

func (s *Store) ForUser(id string) *Store { return &Store{db: s.db, actor: id} }
func (s *Store) actorID() string {
	if s.actor == "" {
		return "admin"
	}
	return s.actor
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) roomAllowed(ctx context.Context, q queryer, room string, manage bool) error {
	var n int
	err := q.QueryRowContext(ctx, `SELECT count(*) FROM rooms r WHERE id=? AND (?='admin' OR owner_id=? OR (?=0 AND EXISTS(SELECT 1 FROM room_users u WHERE u.room_id=r.id AND u.user_id=?)))`, room, s.actorID(), s.actorID(), manage, s.actorID()).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) UserCredentials(ctx context.Context, username string) (User, []byte, []byte, error) {
	var u User
	var salt, hash []byte
	e := s.db.QueryRowContext(ctx, "SELECT id,username,created_at,password_salt,password_hash FROM users WHERE username=? AND id<>'admin'", username).Scan(&u.ID, &u.Username, &u.CreatedAt, &salt, &hash)
	if errors.Is(e, sql.ErrNoRows) {
		e = ErrDenied
	}
	return u, salt, hash, e
}
func (s *Store) User(ctx context.Context, id string) (User, error) {
	var u User
	e := s.db.QueryRowContext(ctx, "SELECT id,username,created_at FROM users WHERE id=?", id).Scan(&u.ID, &u.Username, &u.CreatedAt)
	return u, e
}
func (s *Store) Register(ctx context.Context, id, username, code string, salt, hash []byte, now int64) (User, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return User{}, e
	}
	defer tx.Rollback()
	var invite string
	e = tx.QueryRowContext(ctx, "SELECT id FROM invitations WHERE code_hash=? AND kind='registration' AND used_at IS NULL AND revoked_at IS NULL AND expires_at>?", code, now).Scan(&invite)
	if errors.Is(e, sql.ErrNoRows) {
		return User{}, ErrDenied
	}
	if e != nil {
		return User{}, e
	}
	var count int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&count); e != nil {
		return User{}, e
	}
	if count >= 256 {
		return User{}, ErrLimit
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE username=?", username).Scan(&count); e != nil {
		return User{}, e
	}
	if count != 0 {
		return User{}, ErrConflict
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO users(id,username,password_salt,password_hash,created_at) VALUES(?,?,?,?,?)", id, username, salt, hash, now); e != nil {
		return User{}, e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE invitations SET used_at=?,used_by=? WHERE id=?", now, id, invite); e != nil {
		return User{}, e
	}
	return User{id, username, now}, tx.Commit()
}
func (s *Store) CreateInvitation(ctx context.Context, id, hash, kind, room string, now int64) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var roomValue any
	if kind == "registration" {
		if s.actorID() != "admin" {
			return ErrNotFound
		}
	} else if kind == "room" {
		if e = s.roomAllowed(ctx, tx, room, true); e != nil {
			return e
		}
		var archived sql.NullInt64
		if e = tx.QueryRowContext(ctx, "SELECT archived_at FROM rooms WHERE id=?", room).Scan(&archived); e != nil {
			return e
		}
		if archived.Valid {
			return ErrNotFound
		}
		roomValue = room
	} else {
		return ErrInvalid
	}
	// Expired records carry no secrets and need not accumulate forever.
	if _, e = tx.ExecContext(ctx, "DELETE FROM invitations WHERE expires_at<=?", now); e != nil {
		return e
	}
	var count int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM invitations").Scan(&count); e != nil {
		return e
	}
	if count >= 1024 {
		return ErrLimit
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO invitations(id,code_hash,kind,room_id,created_by,created_at,expires_at) VALUES(?,?,?,?,?,?,?)", id, hash, kind, roomValue, s.actorID(), now, now+7*86400)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Invitations(ctx context.Context) ([]Invitation, error) {
	rows, e := s.db.QueryContext(ctx, "SELECT id,kind,room_id,created_at,expires_at,used_at,revoked_at FROM invitations WHERE created_by=? ORDER BY created_at DESC,id", s.actorID())
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Invitation{}
	for rows.Next() {
		var v Invitation
		if e = rows.Scan(&v.ID, &v.Kind, &v.RoomID, &v.CreatedAt, &v.ExpiresAt, &v.UsedAt, &v.RevokedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) RevokeInvitation(ctx context.Context, id string, now int64) error {
	r, e := s.db.ExecContext(ctx, "UPDATE invitations SET revoked_at=COALESCE(revoked_at,?) WHERE id=? AND created_by=?", now, id, s.actorID())
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) JoinRoom(ctx context.Context, hash string, now int64) (string, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	var id, room, owner string
	e = tx.QueryRowContext(ctx, `SELECT i.id,i.room_id,r.owner_id FROM invitations i JOIN rooms r ON r.id=i.room_id WHERE i.code_hash=? AND i.kind='room' AND i.used_at IS NULL AND i.revoked_at IS NULL AND i.expires_at>? AND r.archived_at IS NULL`, hash, now).Scan(&id, &room, &owner)
	if errors.Is(e, sql.ErrNoRows) {
		return "", ErrDenied
	}
	if e != nil {
		return "", e
	}
	if owner == s.actorID() {
		return "", ErrConflict
	}
	if _, e = tx.ExecContext(ctx, `INSERT OR IGNORE INTO room_users(room_id,user_id,since_message_id) VALUES(?,?,(SELECT COALESCE(MAX(id),0) FROM messages))`, room, s.actorID()); e != nil {
		return "", e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE invitations SET used_at=?,used_by=? WHERE id=?", now, s.actorID(), id); e != nil {
		return "", e
	}
	return room, tx.Commit()
}
func (s *Store) RoomUsers(ctx context.Context, room string) ([]User, error) {
	if e := s.roomAllowed(ctx, s.db, room, false); e != nil {
		return nil, e
	}
	rows, e := s.db.QueryContext(ctx, `SELECT id,username,created_at FROM users WHERE id IN (SELECT owner_id FROM rooms WHERE id=? UNION SELECT user_id FROM room_users WHERE room_id=?) ORDER BY username`, room, room)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if e = rows.Scan(&u.ID, &u.Username, &u.CreatedAt); e != nil {
			return nil, e
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
func (s *Store) RemoveRoomUser(ctx context.Context, room, user string, now int64) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = s.roomAllowed(ctx, tx, room, user != s.actorID()); e != nil {
		return e
	}
	var owner string
	if e = tx.QueryRowContext(ctx, "SELECT owner_id FROM rooms WHERE id=?", room).Scan(&owner); e != nil {
		return e
	}
	if user == owner {
		return ErrInvalid
	}
	if _, e = tx.ExecContext(ctx, `UPDATE receipts SET withdrawn_at=COALESCE(withdrawn_at,?) WHERE message_id IN(SELECT id FROM messages WHERE room_id=?) AND device_id IN(SELECT id FROM devices WHERE owner_id=?)`, now, room, user); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM room_members WHERE room_id=? AND device_id IN(SELECT id FROM devices WHERE owner_id=?)", room, user); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM room_users WHERE room_id=? AND user_id=?", room, user); e != nil {
		return e
	}
	return tx.Commit()
}
