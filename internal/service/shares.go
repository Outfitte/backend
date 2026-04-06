package service

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
)

// cleanUpShares removes all shares targeting the given entity.
func cleanUpShares(ctx context.Context, shares shareAccessChecker, targetType domain.ShareTargetType, targetID string) error {
	return shares.DeleteByTarget(ctx, targetType, targetID)
}
