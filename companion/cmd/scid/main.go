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

	"github.com/lhw/scid/companion/internal/api"
	"github.com/lhw/scid/companion/internal/config"
	"github.com/lhw/scid/companion/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "err", err)
		os.Exit(1)
	}

	st, err := store.New(cfg.DatabaseSource())
	if err != nil {
		slog.Error("database init failed", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	srv := api.New(cfg, st)

	// Background job: re-verify all verified users weekly.
	jobCtx, cancelJobs := context.WithCancel(context.Background())
	defer cancelJobs()
	srv.StartReverifyJob(jobCtx)

	httpServer := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      srv,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting HTTP server", "addr", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	cancelJobs() // stop background jobs first
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
	slog.Info("server stopped")
}
