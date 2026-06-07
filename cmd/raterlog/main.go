package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"raterlog-go/internal/config"
	"raterlog-go/internal/httpapi"
	"raterlog-go/internal/postgres"
	"raterlog-go/internal/scylla"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	ctx := context.Background()

	pg, err := retry(ctx, "postgres", func() (*postgres.Store, error) {
		return postgres.Connect(ctx, cfg.PostgresURL)
	})
	if err != nil {
		return err
	}
	defer pg.Close()

	logs, err := retry(ctx, "scylla", func() (*scylla.Store, error) {
		return scylla.Connect(ctx, cfg.ScyllaHosts, cfg.ScyllaKeyspace)
	})
	if err != nil {
		return err
	}
	defer logs.Close()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(cfg, pg, logs),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.HTTPAddr)
		errs <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signals:
		slog.Info("shutting down", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func retry[T any](ctx context.Context, name string, connect func() (T, error)) (T, error) {
	var zero T
	delay := time.Second
	for attempt := 1; attempt <= 20; attempt++ {
		value, err := connect()
		if err == nil {
			return value, nil
		}
		slog.Warn("dependency not ready", "name", name, "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
		if delay < 10*time.Second {
			delay *= 2
		}
	}
	return zero, errors.New(name + " did not become ready")
}
