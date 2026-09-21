package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestPersistenceAndPragmas(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "nested", "relay ?#.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"journal_mode": "wal", "foreign_keys": "1", "synchronous": "2"} {
		var got string
		if err = s.db.QueryRow("PRAGMA " + key).Scan(&got); err != nil || got != want {
			t.Fatalf("%s: %s %v", key, got, err)
		}
	}
	if _, err = s.db.Exec("INSERT INTO app_metadata VALUES ('test','persisted')"); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var value string
	if err = s.db.QueryRow("SELECT value FROM app_metadata WHERE key='test'").Scan(&value); err != nil || value != "persisted" {
		t.Fatalf("lost data: %s %v", value, err)
	}
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if count != 6 {
		t.Fatal(count)
	}
	if err = s.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0077 != 0 {
		t.Fatal("database accessible to other users")
	}
}
func TestMigrationFailureIsAtomic(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, _ := migrations.ReadFile("migrations/001_metadata.sql")
	second, _ := migrations.ReadFile("migrations/002_devices.sql")
	third, _ := migrations.ReadFile("migrations/003_messages.sql")
	fourth, _ := migrations.ReadFile("migrations/004_firmware.sql")
	fifth, _ := migrations.ReadFile("migrations/005_accounts.sql")
	sixth, _ := migrations.ReadFile("migrations/006_console.sql")
	source := fstest.MapFS{
		"migrations/001_metadata.sql": &fstest.MapFile{Data: first},
		"migrations/002_devices.sql":  &fstest.MapFile{Data: second},
		"migrations/003_messages.sql": &fstest.MapFile{Data: third},
		"migrations/004_firmware.sql": &fstest.MapFile{Data: fourth},
		"migrations/005_accounts.sql": &fstest.MapFile{Data: fifth},
		"migrations/006_console.sql":  &fstest.MapFile{Data: sixth},
		"migrations/007_bad.sql":      &fstest.MapFile{Data: []byte("CREATE TABLE partial (id INTEGER); INSERT INTO nonexistent VALUES(1);")},
	}
	if err = migrate(ctx, s.db, source); err == nil {
		t.Fatal("accepted bad SQL")
	}
	var n int
	_ = s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='partial'").Scan(&n)
	if n != 0 {
		t.Fatal("partial migration committed")
	}
	_ = s.db.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&n)
	if n != 6 {
		t.Fatal("bad migration recorded")
	}
}
func TestMigrationDriftAndDowngradeRejected(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	bad := fstest.MapFS{"migrations/001_metadata.sql": &fstest.MapFile{Data: []byte("SELECT 1;")}}
	if migrate(ctx, s.db, bad) == nil {
		t.Fatal("accepted checksum drift")
	}
	_, err = s.db.Exec("INSERT INTO schema_migrations(name,checksum) VALUES ('migrations/999_future.sql','future')")
	if err != nil {
		t.Fatal(err)
	}
	if migrate(ctx, s.db, migrations) == nil {
		t.Fatal("accepted newer database")
	}
}
func TestClosedDatabaseNotReady(t *testing.T) {
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	if s.Ready(context.Background()) == nil {
		t.Fatal("closed store ready")
	}
}
