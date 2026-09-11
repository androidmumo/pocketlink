package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPasswordFile(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{{"a long test password\n", true}, {"a long test password\r\n", true}, {strings.Repeat("x", 256) + "\r\ntrailing", false}, {"short", false}, {strings.Repeat("x", 300), false}, {"a long password\nsecond line", false}} {
		path := filepath.Join(t.TempDir(), "secret")
		if e := os.WriteFile(path, []byte(tc.body), 0600); e != nil {
			t.Fatal(e)
		}
		_, e := readPassword(path)
		if (e == nil) != tc.valid {
			t.Fatal("unexpected secret validation")
		}
	}
}
