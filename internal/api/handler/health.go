package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

// HealthHandler handles GET /health. No auth required.
type HealthHandler struct {
	log *slog.Logger
}

// NewHealthHandler creates a HealthHandler with a logger pre-scoped to handler=health.
func NewHealthHandler(logger *slog.Logger) *HealthHandler {
	return &HealthHandler{log: logger.With("handler", "health")}
}

// ServeHTTP handles the health check request.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "ServeHTTP")
	log.InfoContext(ctx, "started")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}
