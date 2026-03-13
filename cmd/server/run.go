package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/outfitte/outfitte/internal/api/server"
	"github.com/outfitte/outfitte/internal/config"
)

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	logger.Info("server started", slog.Int("port", cfg.ServerPort))
	return runServer(ctx, cfg, logger)
}

func runServer(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	return server.Run(ctx, server.New(cfg, logger))
}
