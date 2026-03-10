package main

import (
	"context"
	"fmt"

	"github.com/outfitte/outfitte/internal/api/server"
	"github.com/outfitte/outfitte/internal/config"
)

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("server init: %w", err)
	}
	return server.Run(ctx, srv)
}
