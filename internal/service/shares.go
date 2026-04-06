package service

import (
	"context"
	"log/slog"

	"github.com/outfitte/backend/internal/domain"
)

// cleanUpShares removes all shares targeting the given entity. Failures are
// logged but do not propagate — orphaned shares for a deleted entity are
// harmless and must not roll back a successful delete.
func cleanUpShares(ctx context.Context, log *slog.Logger, shares shareAccessChecker, targetType domain.ShareTargetType, targetID string) {
	if err := shares.DeleteByTarget(ctx, targetType, targetID); err != nil {
		log.ErrorContext(ctx, "failed to clean up shares after delete", "error", err, "target_type", targetType, "target_id", targetID)
	}
}
