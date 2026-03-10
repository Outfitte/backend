// Package server constructs and configures the HTTP server.
package server

import (
	"net/http"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/config"
)

// New builds a configured *http.Server from cfg.
func New(cfg *config.Config) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)

	return &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: mux,
	}, nil
}
