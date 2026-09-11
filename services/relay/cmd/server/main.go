package main

import (
	"context"
	"errors"
	"fmt"
	console "github.com/androidmumo/pocketlink/apps/console"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/androidmumo/pocketlink/services/relay/internal/config"
	"github.com/androidmumo/pocketlink/services/relay/internal/httpapi"
	"github.com/androidmumo/pocketlink/services/relay/internal/store"
)

var version = "dev"

func main() { os.Exit(entry(os.Args[1:])) }
func entry(args []string) int {
	if len(args) == 1 && args[0] == "--version" {
		fmt.Println(version)
		return 0
	}
	c, err := config.Load(os.Getenv)
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		return 1
	}
	if len(args) > 0 {
		if len(args) == 1 && args[0] == "healthcheck" {
			if healthcheck(c.ListenAddress) != nil {
				return 1
			}
			return 0
		}
		slog.Error("unknown command")
		return 2
	}
	var level slog.Level
	_ = level.UnmarshalText([]byte(c.LogLevel))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err = run(ctx, c, logger); err != nil {
		logger.Error("server stopped", "error", err)
		return 1
	}
	return 0
}
func run(ctx context.Context, c config.Config, logger *slog.Logger) error {
	startup, cancel := context.WithTimeout(ctx, 15*time.Second)
	db, err := store.Open(startup, c.DatabasePath)
	cancel()
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}
	defer db.Close()
	api := httpapi.New(db, version)
	stage := "foundation"
	if c.AdminPasswordFile != "" {
		password, err := readPassword(c.AdminPasswordFile)
		if err != nil {
			return err
		}
		auth, err := httpapi.NewAuth(db, c.PublicOrigin, password)
		if err != nil {
			return err
		}
		api.EnableAuth(auth, console.Handler())
		stage = "device-auth"
	}
	srv := &http.Server{Handler: api, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	listener, err := net.Listen("tcp", c.ListenAddress)
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	logger.Info("server started", "listen", listener.Addr().String(), "version", version, "stage", stage)
	select {
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	api.Drain()
	shutdown, cancel := context.WithTimeout(context.Background(), c.ShutdownTimeout)
	defer cancel()
	if err = srv.Shutdown(shutdown); err != nil {
		_ = srv.Close()
		return fmt.Errorf("shutdown: %w", err)
	}
	err = <-done
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("server stopped gracefully")
	return nil
}
func healthcheck(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	if host == "::" {
		host = "::1"
	}
	// Never route an in-container readiness probe through a configured HTTP proxy.
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}}
	response, err := client.Get("http://" + net.JoinHostPort(host, port) + "/health/ready")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("not ready")
	}
	return nil
}
