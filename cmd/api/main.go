package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"pokemon_ai/internal/app"
)

func main() {
	initLogger()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server, cleanup, err := app.Bootstrap(ctx)
	if err != nil {
		slog.Error("bootstrap_failed", "error", err)
		os.Exit(1)
	}
	var cleanupOnce sync.Once
	runCleanup := func() {
		cleanupOnce.Do(cleanup)
	}

	slog.Info("server_starting", "addr", server.Addr)
	serverErrCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	select {
	case <-ctx.Done():
		slog.Info("signal_received", "reason", ctx.Err())
		slog.Info("shutdown_started")
		runCleanup()
		slog.Info("shutdown_completed")
	case serveErr := <-serverErrCh:
		if serveErr != nil {
			slog.Error("server_failed", "error", serveErr)
			runCleanup()
			os.Exit(1)
		}
		slog.Info("server_stopped")
		runCleanup()
	}
}

func initLogger() {
	level := new(slog.LevelVar)
	level.Set(parseLevel(os.Getenv("LOG_LEVEL")))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)
}

func parseLevel(raw string) slog.Level {
	switch raw {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
