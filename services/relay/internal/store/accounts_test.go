package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

func TestAccountMigrationPreservesLegacyData(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy.db")
	raw, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	raw.SetMaxOpenConns(1)
	if _, e = raw.Exec("PRAGMA foreign_keys=ON"); e != nil {
		t.Fatal(e)
	}
	source := fstest.MapFS{}
	for _, name := range []string{"001_metadata.sql", "002_devices.sql", "003_messages.sql", "004_firmware.sql"} {
		b, _ := migrations.ReadFile("migrations/" + name)
		source["migrations/"+name] = &fstest.MapFile{Data: b}
	}
	if e = migrate(ctx, raw, source); e != nil {
		t.Fatal(e)
	}
	for _, statement := range []string{
		"INSERT INTO devices(id,sn,name,credential_hash,created_at) VALUES('d','legacy','old device','credential',1)",
		"INSERT INTO pairings(code_hash,name,expires_at) VALUES('pair','pending device',1000)",
		"INSERT INTO rooms(id,name,created_at) VALUES('r','old room',1)",
		"INSERT INTO room_members VALUES('r','d')",
		"INSERT INTO messages(room_id,request_id,body,created_at) VALUES('r','request','retained text',1)",
		"INSERT INTO receipts(message_id,device_id) VALUES(1,'d')",
	} {
		if _, e = raw.Exec(statement); e != nil {
			t.Fatal(e)
		}
	}
	raw.Close()
	db, e := Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	for _, table := range []string{"devices", "pairings", "rooms"} {
		var owner string
		if e = db.db.QueryRow("SELECT owner_id FROM " + table).Scan(&owner); e != nil || owner != "admin" {
			t.Fatal(table, owner, e)
		}
	}
	if e = db.ConsumePairing(ctx, "pair", "new-id", "new-serial", "new-credential", 2); e != nil {
		t.Fatal(e)
	}
	pending, e := db.Pending(ctx, "credential")
	if e != nil || len(pending) != 1 || pending[0].Text != "retained text" || pending[0].Sender != "admin" {
		t.Fatal(pending, e)
	}
}
func TestRegistrationAtomicExpiryAndPersistence(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "accounts.db")
	db, e := Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { db.Close() }()
	if e = db.CreateInvitation(ctx, "i", "single-code", "", "registration", "", 10); e != nil {
		t.Fatal(e)
	}
	var success atomic.Int32
	var wg sync.WaitGroup
	for _, id := range []string{"alice", "bob"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, err := db.Register(ctx, id, id, "single-code", []byte("salt"), []byte("hash"), 11)
			if err == nil {
				success.Add(1)
			} else if !errors.Is(err, ErrDenied) {
				t.Error(err)
			}
		}(id)
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatal(success.Load())
	}
	if e = db.CreateInvitation(ctx, "expired", "expired-code", "", "registration", "", 10); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Register(ctx, "expired-user", "expired_user", "expired-code", nil, nil, 10+7*86400); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	db.Close()
	db, e = Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	var name string
	if e = db.db.QueryRow("SELECT username FROM users WHERE id<>'admin'").Scan(&name); e != nil {
		t.Fatal(e)
	}
	u, salt, hash, e := db.UserCredentials(ctx, name)
	if e != nil || u.Username != name || string(salt) != "salt" || string(hash) != "hash" {
		t.Fatal("credentials did not persist", e)
	}
}
