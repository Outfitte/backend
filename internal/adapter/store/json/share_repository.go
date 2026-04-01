package json

import (
	"context"
	"errors"

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
	return domain.Share{}, errors.New("not implemented")
}

func (r *ShareRepository) Save(ctx context.Context, share domain.Share) error {
	return errors.New("not implemented")
}

func (r *ShareRepository) Delete(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func (r *ShareRepository) ListByOwner(ctx context.Context, ownerID string) ([]domain.Share, error) {
	return nil, errors.New("not implemented")
}

func (r *ShareRepository) ListByRecipient(ctx context.Context, recipientID string) ([]domain.Share, error) {
	return nil, errors.New("not implemented")
}

func (r *ShareRepository) FindByTarget(ctx context.Context, ownerID, recipientID string, targetType domain.ShareTargetType, targetID string) (*domain.Share, error) {
	return nil, errors.New("not implemented")
}

func (r *ShareRepository) DeleteByTarget(ctx context.Context, targetType domain.ShareTargetType, targetID string) error {
	return errors.New("not implemented")
}

func (r *ShareRepository) HasDirectAccess(ctx context.Context, recipientID string, targetType domain.ShareTargetType, targetID string) (bool, error) {
	return false, errors.New("not implemented")
}

func (r *ShareRepository) ListByRecipientAndType(ctx context.Context, recipientID string, targetType domain.ShareTargetType) ([]domain.Share, error) {
	return nil, errors.New("not implemented")
}
