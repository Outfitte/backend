// Package server constructs and configures the HTTP server.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/config"
)

// New builds a configured *http.Server from cfg and logger.
func New(cfg *config.Config, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.NewHealthHandler(logger))

	return &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.ServerPort),
		Handler: mux,
	}
}

// Run listens on srv's configured address and shuts down when ctx is done.
func Run(ctx context.Context, srv *http.Server) error {
	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return serve(ctx, srv, l)
}

// serve runs srv on l, shutting down gracefully when ctx is done.
func serve(ctx context.Context, srv *http.Server, l net.Listener) error {
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
