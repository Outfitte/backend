package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	store "github.com/outfitte/outfitte/internal/adapter/store"
	localmedia "github.com/outfitte/outfitte/internal/adapter/media/local"
	"github.com/outfitte/outfitte/internal/api/server"
	"github.com/outfitte/outfitte/internal/config"
)

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	logger.Info("server started", "port", cfg.ServerPort)
	return runServer(ctx, cfg, logger)
}

func newServer(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*http.Server, func(), error) {
	repos, closer, err := store.NewRepositories(ctx, *cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open repositories: %w", err)
	}
	media := localmedia.NewProvider(cfg.MediaStoragePath)
	cleanup := func() { closer.Close() } //nolint:errcheck
	return server.New(cfg, logger, repos, media), cleanup, nil
}

func runServer(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	srv, cleanup, err := newServer(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer cleanup()
	return server.Run(ctx, srv)
}
