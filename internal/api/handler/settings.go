package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/outfitte/outfitte/internal/api/middleware"
	"github.com/outfitte/outfitte/internal/domain"
)

type settingsService interface {
	GetSettings(ctx context.Context) (domain.AppSettings, error)
	UpdateRegistrationEnabled(ctx context.Context, callerID string, enabled bool) error
}

// SettingsHandler handles GET and PATCH /admin/settings.
type SettingsHandler struct {
	settings settingsService
	log      *slog.Logger
}

// NewSettingsHandler creates a SettingsHandler with a logger pre-scoped to handler=settings.
func NewSettingsHandler(settings settingsService, log *slog.Logger) *SettingsHandler {
	return &SettingsHandler{settings: settings, log: log.With("handler", "settings")}
}

type settingsResponse struct {
	RegistrationEnabled bool `json:"registration_enabled"`
}

type updateSettingsRequest struct {
	RegistrationEnabled bool `json:"registration_enabled"`
}

// UpdateSettings handles PATCH /admin/settings.
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "UpdateSettings")
	log.InfoContext(ctx, "started")

	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	callerID, _ := middleware.UserIDFromContext(ctx)

	if err := h.settings.UpdateRegistrationEnabled(ctx, callerID, req.RegistrationEnabled); err != nil {
		log.ErrorContext(ctx, "update settings failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	s, err := h.settings.GetSettings(ctx)
	if err != nil {
		log.ErrorContext(ctx, "get settings after update failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded")
	writeJSON(w, http.StatusOK, settingsResponse{RegistrationEnabled: s.RegistrationEnabled})
}

// GetSettings handles GET /admin/settings.
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "GetSettings")
	log.InfoContext(ctx, "started")

	s, err := h.settings.GetSettings(ctx)
	if err != nil {
		log.ErrorContext(ctx, "get settings failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded")
	writeJSON(w, http.StatusOK, settingsResponse{RegistrationEnabled: s.RegistrationEnabled})
}
