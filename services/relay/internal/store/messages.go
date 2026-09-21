package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/androidmumo/pocketlink/services/relay/internal/protocol"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrConflict = errors.New("idempotency conflict")
var ErrInvalid = errors.New("invalid input")
var ErrNotFound = errors.New("not found")
var ErrEmptyRoom = errors.New("no active recipients")

func ValidName(s string) bool {
	return s == strings.TrimSpace(s) && len(s) > 0 && utf8.ValidString(s) && utf8.RuneCountInString(s) <= 40 && !strings.ContainsFunc(s, unicode.IsControl)
}

type Room struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	OwnerID    string `json:"owner_id"`
	CreatedAt  int64  `json:"created_at"`
	ArchivedAt *int64 `json:"archived_at"`
}
type Message struct {
	Sender    string `json:"sender"`
	ID        int64  `json:"id"`
	RoomID    string `json:"room_id"`
	RequestID string `json:"request_id"`
	Text      string `json:"text"`
	CreatedAt int64  `json:"created_at"`
}
type Receipt struct {
	DeviceID    string `json:"device_id"`
	Name        string `json:"name"`
	DeliveredAt *int64 `json:"delivered_at"`
	ReadAt      *int64 `json:"read_at"`
	WithdrawnAt *int64 `json:"withdrawn_at"`
}

func (s *Store) CreateRoom(ctx context.Context, id, name string, now int64) error {
	if !ValidName(name) {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var count int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM rooms").Scan(&count); e != nil {
		return e
	}
	if count >= 64 {
		return ErrLimit
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO rooms(id,name,created_at,owner_id) VALUES(?,?,?,?)", id, name, now, s.actorID()); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Rooms(ctx context.Context) ([]Room, error) {
	rows, e := s.db.QueryContext(ctx, "SELECT id,name,owner_id,created_at,archived_at FROM rooms r WHERE owner_id=? OR ?='admin' OR EXISTS(SELECT 1 FROM room_users u WHERE u.room_id=r.id AND u.user_id=?) ORDER BY created_at,id", s.actorID(), s.actorID(), s.actorID())
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []Room{}
	for rows.Next() {
		var v Room
		if e = rows.Scan(&v.ID, &v.Name, &v.OwnerID, &v.CreatedAt, &v.ArchivedAt); e != nil {
			return nil, e
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (s *Store) ArchiveRoom(ctx context.Context, id string, now int64) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = s.roomAllowed(ctx, tx, id, true); e != nil {
		return e
	}
	result, e := tx.ExecContext(ctx, "UPDATE rooms SET archived_at=COALESCE(archived_at,?) WHERE id=?", now, id)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	if _, e = tx.ExecContext(ctx, "UPDATE receipts SET withdrawn_at=COALESCE(withdrawn_at,?) WHERE message_id IN (SELECT id FROM messages WHERE room_id=?)", now, id); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM room_members WHERE room_id=?", id); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) SetMember(ctx context.Context, room, device string, add bool, now int64) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = s.roomAllowed(ctx, tx, room, false); e != nil {
		return e
	}
	var allowed int
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM devices d WHERE id=? AND (owner_id=? OR ?='admin' OR (?=0 AND EXISTS(SELECT 1 FROM rooms WHERE id=? AND owner_id=?)))`, device, s.actorID(), s.actorID(), add, room, s.actorID()).Scan(&allowed); e != nil {
		return e
	}
	if allowed == 0 {
		return ErrNotFound
	}
	var count int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM rooms WHERE id=? AND archived_at IS NULL", room).Scan(&count); e != nil {
		return e
	}
	if count == 0 {
		return ErrNotFound
	}
	if add {
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM devices WHERE id=? AND revoked_at IS NULL", device).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			return ErrNotFound
		}
		_, e = tx.ExecContext(ctx, "INSERT OR IGNORE INTO room_members(room_id,device_id) VALUES(?,?)", room, device)
	} else {
		if _, e = tx.ExecContext(ctx, "DELETE FROM room_members WHERE room_id=? AND device_id=?", room, device); e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, "UPDATE receipts SET withdrawn_at=COALESCE(withdrawn_at,?) WHERE device_id=? AND message_id IN (SELECT id FROM messages WHERE room_id=?)", now, device, room)
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Members(ctx context.Context, room string) ([]string, error) {
	if e := s.roomAllowed(ctx, s.db, room, false); e != nil {
		return nil, e
	}

	rows, e := s.db.QueryContext(ctx, "SELECT device_id FROM room_members WHERE room_id=? ORDER BY device_id", room)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Message and recipient snapshot commit together. Retrying the same key/body
// returns the original message, even after membership changes or room archival.
func (s *Store) SendText(ctx context.Context, room, key, text string, now int64) (Message, error) {
	var m Message
	if protocol.ValidateText(text) != nil || strings.TrimSpace(text) == "" || len(key) == 0 || len(key) > 64 {
		return m, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return m, e
	}
	defer tx.Rollback()
	if e = s.roomAllowed(ctx, tx, room, false); e != nil {
		return m, e
	}
	e = tx.QueryRowContext(ctx, "SELECT id,room_id,request_id,body,created_at,sender_name FROM messages WHERE room_id=? AND request_id=?", room, key).Scan(&m.ID, &m.RoomID, &m.RequestID, &m.Text, &m.CreatedAt, &m.Sender)
	if e == nil {
		var visible int
		if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM rooms r WHERE id=? AND (?='admin' OR owner_id=? OR EXISTS(SELECT 1 FROM room_users u WHERE u.room_id=r.id AND u.user_id=? AND ?>u.since_message_id))`, room, s.actorID(), s.actorID(), s.actorID(), m.ID).Scan(&visible); e != nil {
			return Message{}, e
		}
		if visible == 0 {
			return Message{}, ErrNotFound
		}
		if m.Text != text {
			return Message{}, ErrConflict
		}
		return m, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return m, e
	}
	var count int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM rooms WHERE id=? AND archived_at IS NULL", room).Scan(&count); e != nil {
		return m, e
	}
	if count == 0 {
		return m, ErrNotFound
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM messages").Scan(&count); e != nil {
		return m, e
	}
	if count >= 10000 {
		return m, ErrLimit
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM room_members rm JOIN devices d ON d.id=rm.device_id WHERE rm.room_id=? AND d.revoked_at IS NULL", room).Scan(&count); e != nil {
		return m, e
	}
	if count == 0 {
		return m, ErrEmptyRoom
	}
	var sender string
	if e = tx.QueryRowContext(ctx, "SELECT username FROM users WHERE id=?", s.actorID()).Scan(&sender); e != nil {
		return m, e
	}
	result, e := tx.ExecContext(ctx, "INSERT INTO messages(room_id,request_id,body,created_at,sender_name) VALUES(?,?,?,?,?)", room, key, text, now, sender)
	if e != nil {
		return m, e
	}
	id, e := result.LastInsertId()
	if e != nil {
		return m, e
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO receipts(message_id,device_id) SELECT ?,rm.device_id FROM room_members rm JOIN devices d ON d.id=rm.device_id WHERE rm.room_id=? AND d.revoked_at IS NULL`, id, room); e != nil {
		return m, e
	}
	if e = tx.Commit(); e != nil {
		return m, e
	}
	return Message{ID: id, RoomID: room, RequestID: key, Text: text, CreatedAt: now, Sender: sender}, nil
}
func scanMessages(rows *sql.Rows) ([]Message, error) {
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if e := rows.Scan(&m.ID, &m.RoomID, &m.RequestID, &m.Text, &m.CreatedAt, &m.Sender); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) History(ctx context.Context, room string, before int64) ([]Message, error) {
	if e := s.roomAllowed(ctx, s.db, room, false); e != nil {
		return nil, e
	}

	if before == 0 {
		before = 1<<63 - 1
	}
	rows, e := s.db.QueryContext(ctx, "SELECT id,room_id,request_id,body,created_at,sender_name FROM messages m WHERE room_id=? AND id<? AND (?='admin' OR EXISTS(SELECT 1 FROM rooms WHERE id=m.room_id AND owner_id=?) OR EXISTS(SELECT 1 FROM room_users WHERE room_id=m.room_id AND user_id=? AND m.id>since_message_id)) ORDER BY id DESC LIMIT 20", room, before, s.actorID(), s.actorID(), s.actorID())
	if e != nil {
		return nil, e
	}
	return scanMessages(rows)
}
func (s *Store) MessageReceipts(ctx context.Context, id int64) ([]Receipt, error) {
	var allowed int
	if e := s.db.QueryRowContext(ctx, `SELECT count(*) FROM messages m JOIN rooms r ON r.id=m.room_id WHERE m.id=? AND (?='admin' OR r.owner_id=? OR EXISTS(SELECT 1 FROM room_users u WHERE u.room_id=r.id AND u.user_id=? AND m.id>u.since_message_id))`, id, s.actorID(), s.actorID(), s.actorID()).Scan(&allowed); e != nil {
		return nil, e
	}
	if allowed == 0 {
		return nil, ErrNotFound
	}

	rows, e := s.db.QueryContext(ctx, "SELECT r.device_id,d.name,r.delivered_at,r.read_at,r.withdrawn_at FROM receipts r JOIN devices d ON d.id=r.device_id WHERE r.message_id=? ORDER BY d.id", id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Receipt{}
	for rows.Next() {
		var v Receipt
		if e = rows.Scan(&v.DeviceID, &v.Name, &v.DeliveredAt, &v.ReadAt, &v.WithdrawnAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Authenticate within each database operation; cached socket identity is never
// sufficient after revocation/re-pairing. Device cursors cannot skip unacked rows.
func (s *Store) Pending(ctx context.Context, hash string) ([]Message, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT m.id,m.room_id,m.request_id,m.body,m.created_at,m.sender_name FROM messages m
 JOIN receipts r ON r.message_id=m.id JOIN devices d ON d.id=r.device_id
 JOIN rooms room ON room.id=m.room_id JOIN room_members rm ON rm.room_id=m.room_id AND rm.device_id=d.id
 WHERE d.credential_hash=? AND d.revoked_at IS NULL AND room.archived_at IS NULL AND r.withdrawn_at IS NULL AND r.delivered_at IS NULL
 ORDER BY m.id LIMIT 20`, hash)
	if e != nil {
		return nil, e
	}
	return scanMessages(rows)
}
func (s *Store) Acknowledge(ctx context.Context, hash string, id int64, state string, now int64) error {
	if state != "received" && state != "read" {
		return ErrInvalid
	}
	result, e := s.db.ExecContext(ctx, `UPDATE receipts SET delivered_at=COALESCE(delivered_at,?),read_at=CASE WHEN ?='read' THEN COALESCE(read_at,?) ELSE read_at END
 WHERE message_id=? AND withdrawn_at IS NULL AND device_id IN (SELECT d.id FROM devices d
 JOIN room_members rm ON rm.device_id=d.id JOIN messages m ON m.room_id=rm.room_id JOIN rooms room ON room.id=m.room_id
 WHERE d.credential_hash=? AND d.revoked_at IS NULL AND m.id=? AND room.archived_at IS NULL)`, now, state, now, id, hash, id)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrDenied
	}
	return nil
}
