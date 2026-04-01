package json

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

var _ ports.ShareRepository = (*ShareRepository)(nil)

// ShareRepository is a JSON file-backed implementation of ports.ShareRepository.
type ShareRepository struct {
	provider *Provider[domain.Share]
}

// NewShareRepository creates a ShareRepository that stores shares in root/shares.json.
func NewShareRepository(root string) *ShareRepository {
	return &ShareRepository{
		provider: NewProvider[domain.Share](root, "shares.json"),
	}
}

func (r *ShareRepository) Get(ctx context.Context, id string) (domain.Share, error) {
	return r.provider.Get(ctx, id)
}

func (r *ShareRepository) Save(ctx context.Context, share domain.Share) error {
	return r.provider.Save(ctx, share)
}

func (r *ShareRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.provider.Delete(ctx, id)
}

func (r *ShareRepository) ListByOwner(ctx context.Context, ownerID string) ([]domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []domain.Share{}
	for _, s := range all {
		if s.OwnerID == ownerID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *ShareRepository) ListByRecipient(ctx context.Context, recipientID string) ([]domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []domain.Share{}
	for _, s := range all {
		if s.RecipientID == recipientID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *ShareRepository) FindByTarget(ctx context.Context, ownerID, recipientID string, targetType domain.ShareTargetType, targetID string) (*domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range all {
		if s.OwnerID == ownerID && s.RecipientID == recipientID && s.TargetType == targetType && s.TargetID == targetID {
			found := s
			return &found, nil
		}
	}
	return nil, nil
}

func (r *ShareRepository) DeleteByTarget(ctx context.Context, targetType domain.ShareTargetType, targetID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return err
	}
	for _, s := range all {
		if s.TargetType == targetType && s.TargetID == targetID {
			if err := r.provider.Delete(ctx, s.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ShareRepository) HasDirectAccess(ctx context.Context, recipientID string, targetType domain.ShareTargetType, targetID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return false, err
	}
	for _, s := range all {
		if s.RecipientID == recipientID && s.TargetType == targetType && s.TargetID == targetID {
			return true, nil
		}
	}
	return false, nil
}

func (r *ShareRepository) ListByRecipientAndType(ctx context.Context, recipientID string, targetType domain.ShareTargetType) ([]domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []domain.Share{}
	for _, s := range all {
		if s.RecipientID == recipientID && s.TargetType == targetType {
			result = append(result, s)
		}
	}
	return result, nil
}
