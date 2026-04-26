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
	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("config_load_failed", "error", err)
		os.Exit(1)
	}
	initLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server, cleanup, err := app.BootstrapWithConfig(ctx, cfg)
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

func initLogger(level slog.Level) {
	levelVar := new(slog.LevelVar)
	levelVar.Set(level)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelVar,
	}))
	slog.SetDefault(logger)
}
