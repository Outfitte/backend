package ports_test

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// Compile-time assertion: shareRepositoryStub must satisfy ShareRepository.
var _ ports.ShareRepository = (*shareRepositoryStub)(nil)

type shareRepositoryStub struct{}

func (s *shareRepositoryStub) Get(_ context.Context, _ string) (domain.Share, error) {
	return domain.Share{}, nil
}

func (s *shareRepositoryStub) Save(_ context.Context, _ domain.Share) error {
	return nil
}

func (s *shareRepositoryStub) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *shareRepositoryStub) ListByOwner(_ context.Context, _ string) ([]domain.Share, error) {
	return nil, nil
}

func (s *shareRepositoryStub) ListByRecipient(_ context.Context, _ string) ([]domain.Share, error) {
	return nil, nil
}

func (s *shareRepositoryStub) FindByTarget(_ context.Context, _, _ string, _ domain.ShareTargetType, _ string) (*domain.Share, error) {
	return nil, nil
}

func (s *shareRepositoryStub) DeleteByTarget(_ context.Context, _ domain.ShareTargetType, _ string) error {
	return nil
}

func (s *shareRepositoryStub) HasDirectAccess(_ context.Context, _ string, _ domain.ShareTargetType, _ string) (bool, error) {
	return false, nil
}

func (s *shareRepositoryStub) ListByRecipientAndType(_ context.Context, _ string, _ domain.ShareTargetType) ([]domain.Share, error) {
	return nil, nil
}
