package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db    *sql.DB
	actor string
}

func Open(ctx context.Context, path string) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	// Create with owner-only permissions; SQLite preserves the existing mode.
	f, err := os.OpenFile(absolute, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open database file: %w", err)
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	uri := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	fail := func(err error) (*Store, error) { _ = db.Close(); return nil, err }
	if _, err = db.ExecContext(ctx, "PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON; PRAGMA synchronous=FULL;"); err != nil {
		return fail(err)
	}
	var journal string
	if err = db.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&journal); err != nil {
		return fail(err)
	}
	if journal != "wal" {
		return fail(fmt.Errorf("WAL journal mode unavailable"))
	}
	if err = migrate(ctx, db, migrations); err != nil {
		return fail(err)
	}
	return &Store{db: db}, nil
}

func migrate(ctx context.Context, db *sql.DB, source fs.FS) error {
	files, err := fs.Glob(source, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no migrations embedded")
	}
	// A single immediate transaction prevents two processes from racing migrations.
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	if _, err = conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
  name TEXT PRIMARY KEY, checksum TEXT NOT NULL,
  applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
 )`); err != nil {
		return err
	}
	rows, err := conn.QueryContext(ctx, "SELECT name,checksum FROM schema_migrations ORDER BY name")
	if err != nil {
		return err
	}
	applied := map[string]string{}
	for rows.Next() {
		var name, sum string
		if err = rows.Scan(&name, &sum); err != nil {
			rows.Close()
			return err
		}
		applied[name] = sum
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	known := map[string]bool{}
	seenPending := false
	for _, name := range files {
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(data))
		known[name] = true
		if old, ok := applied[name]; ok {
			if old != checksum {
				return fmt.Errorf("migration checksum mismatch: %s", name)
			}
			if seenPending {
				return fmt.Errorf("migration history is not a prefix")
			}
			continue
		}
		seenPending = true
		if strings.TrimSpace(string(data)) == "" {
			return fmt.Errorf("empty migration: %s", name)
		}
		if _, err = conn.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err = conn.ExecContext(ctx, "INSERT INTO schema_migrations(name,checksum) VALUES (?,?)", name, checksum); err != nil {
			return err
		}
	}
	for name := range applied {
		if !known[name] {
			return fmt.Errorf("database has unknown migration: %s; use a compatible image", name)
		}
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}

func (s *Store) Ready(ctx context.Context) error {
	var version string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM app_metadata WHERE key='protocol_version'").Scan(&version)
	if err != nil {
		return err
	}
	if version != "1" {
		return fmt.Errorf("incompatible database protocol")
	}
	return nil
}
func (s *Store) Close() error { return s.db.Close() }
