package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddress   string
	DatabasePath    string
	ShutdownTimeout time.Duration
	LogLevel        string
}

// Load reads only named settings. Configuration failures never print values.
func Load(getenv func(string) string) (Config, error) {
	value := func(key, fallback string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return fallback
	}
	c := Config{
		ListenAddress: value("POCKETLINK_LISTEN", "127.0.0.1:8080"),
		DatabasePath:  value("POCKETLINK_DATABASE", "./data/relay.db"),
		LogLevel:      value("POCKETLINK_LOG_LEVEL", "info"),
	}
	host, port, err := net.SplitHostPort(c.ListenAddress)
	if err != nil || (host != "" && net.ParseIP(host) == nil) {
		return Config{}, fmt.Errorf("POCKETLINK_LISTEN must use a literal IP and port")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("POCKETLINK_LISTEN port must be 1..65535")
	}
	if strings.TrimSpace(c.DatabasePath) == "" || c.DatabasePath == ":memory:" || strings.HasPrefix(c.DatabasePath, "file:") || strings.ContainsRune(c.DatabasePath, 0) {
		return Config{}, fmt.Errorf("POCKETLINK_DATABASE must be a filesystem path")
	}
	c.ShutdownTimeout, err = time.ParseDuration(value("POCKETLINK_SHUTDOWN_TIMEOUT", "10s"))
	if err != nil || c.ShutdownTimeout < time.Second || c.ShutdownTimeout > time.Minute {
		return Config{}, fmt.Errorf("POCKETLINK_SHUTDOWN_TIMEOUT must be 1s..1m")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("POCKETLINK_LOG_LEVEL is unsupported")
	}
	return c, nil
}
