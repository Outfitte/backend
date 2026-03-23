package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/outfitte/outfitte/internal/api/middleware"
)

// callerIDFromContext extracts the authenticated caller ID from ctx.
// On failure it writes a 500 response and returns ok=false.
func callerIDFromContext(ctx context.Context, w http.ResponseWriter, log *slog.Logger) (string, bool) {
	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	return callerID, ok
}
