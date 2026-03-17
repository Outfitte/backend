package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	localmedia "github.com/outfitte/outfitte/internal/adapter/media/local"
	storejson "github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/api/server"
	"github.com/outfitte/outfitte/internal/config"
	"github.com/outfitte/outfitte/internal/domain"
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

func runServer(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	users := storejson.NewProvider[domain.User](cfg.StorageDataPath, "users.json")
	sessions := storejson.NewProvider[domain.Session](cfg.StorageDataPath, "sessions.json")
	items := storejson.NewProvider[domain.Item](cfg.StorageDataPath, "items.json")
	locations := storejson.NewProvider[domain.Location](cfg.StorageDataPath, "locations.json")
	settings := storejson.NewSingletonStore[domain.AppSettings](cfg.StorageDataPath, "app_settings.json")
	media := localmedia.NewProvider(cfg.MediaStoragePath)

	return server.Run(ctx, server.New(cfg, logger, users, sessions, items, locations, settings, media))
}
