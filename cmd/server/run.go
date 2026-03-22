package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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

func newServer(cfg *config.Config, logger *slog.Logger) *http.Server {
	users := storejson.NewProvider[domain.User](cfg.StorageDataPath, "users.json")
	sessions := storejson.NewProvider[domain.Session](cfg.StorageDataPath, "sessions.json")
	items := storejson.NewItemRepository(cfg.StorageDataPath)
	locations := storejson.NewLocationRepository(cfg.StorageDataPath)
	settings := storejson.NewSingletonStore[domain.AppSettings](cfg.StorageDataPath, "app_settings.json")
	media := localmedia.NewProvider(cfg.MediaStoragePath)
	return server.New(cfg, logger, users, sessions, items, locations, settings, media)
}

func runServer(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	return server.Run(ctx, newServer(cfg, logger))
}
