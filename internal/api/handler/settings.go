package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/outfitte/outfitte/internal/domain"
)

type settingsGetter interface {
	GetSettings(ctx context.Context) (domain.AppSettings, error)
}

// SettingsHandler handles GET /admin/settings.
type SettingsHandler struct {
	settings settingsGetter
	log      *slog.Logger
}

// NewSettingsHandler creates a SettingsHandler with a logger pre-scoped to handler=settings.
func NewSettingsHandler(settings settingsGetter, log *slog.Logger) *SettingsHandler {
	return &SettingsHandler{settings: settings, log: log.With("handler", "settings")}
}

type settingsResponse struct {
	RegistrationEnabled bool `json:"registration_enabled"`
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
