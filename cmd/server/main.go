package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aipsa-backend/internal/config"
	"aipsa-backend/internal/server"
)

func main() {
	cfg := config.Load()

	logger := setupLogger(cfg.Environment)
	slog.SetDefault(logger)

	srv, err := server.New(cfg, logger)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		slog.Info("shutting down server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server forced to shutdown", "error", err)
		}
	}()

	slog.Info("starting aipsa-backend",
		"version", "1.0.0",
		"environment", cfg.Environment,
	)

	if err := srv.Start(); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}

func setupLogger(env string) *slog.Logger {
	var level slog.Level
	if env == "production" {
		level = slog.LevelInfo
	} else {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}
