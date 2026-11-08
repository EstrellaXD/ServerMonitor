package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/api"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/broadcast"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/manager"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/store"
)

func main() {
	settings := config.LoadSettings()
	logLevel := slog.LevelInfo
	if settings.Debug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
	slog.Info("starting ServerMonitor", "host", settings.Host, "port", settings.Port, "mock", settings.MockMode)

	cfg := config.Load(settings.ConfigPath)
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "err", err)
		os.Exit(1)
	}

	st := store.New()
	bus := broadcast.New()
	mgr := manager.New(st, bus, settings.MockMode)
	mgr.LoadConfig(cfg)

	staticDir := envOr("SERVERMONITOR_STATIC_DIR", "/app/static")
	state := api.NewAppState(st, mgr, settings, bus, cfg.EnabledSystemConfigs())

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mgr.Start(rootCtx)
	slog.Info("monitoring", "systems", mgr.Count())

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", settings.Host, settings.Port),
		Handler:           api.Routes(state, staticDir),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-rootCtx.Done():
		slog.Info("shutting down")
	case err := <-errCh:
		if err != nil {
			slog.Error("server error", "err", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	mgr.Stop()
	slog.Info("stopped monitoring")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
