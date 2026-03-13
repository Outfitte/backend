// Package main is the entry point for the Outfitte server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/outfitte/outfitte/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	logger.Info("server started", "port", cfg.ServerPort)

	if err := run(ctx, cfg, logger); err != nil {
		logger.Error("server terminated", "error", err)
		os.Exit(1)
	}
	logger.Info("server terminated")
}
