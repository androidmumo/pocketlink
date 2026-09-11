package main

import (
	"errors"
	"io"
	"os"
	"strings"
)

func readPassword(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", errors.New("cannot read administrator password file")
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 259))
	if e != nil || len(b) > 258 {
		return "", errors.New("cannot read administrator password file")
	}
	password := strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r")
	if len(password) < 16 || len(password) > 256 || strings.ContainsAny(password, "\r\n\x00") {
		return "", errors.New("administrator password must be one line of 16..256 bytes")
	}
	return password, nil
}
