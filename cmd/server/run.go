package main

import (
	"context"
	"log/slog"

	"github.com/outfitte/outfitte/internal/api/server"
	"github.com/outfitte/outfitte/internal/config"
)

func run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	return server.Run(ctx, server.New(cfg, logger))
}
