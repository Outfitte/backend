package json

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/domain"
)

// mockShareProvider is a test double for shareProvider.
type mockShareProvider struct {
	listFn   func(ctx context.Context) ([]domain.Share, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockShareProvider) Get(_ context.Context, _ string) (domain.Share, error) {
	return domain.Share{}, nil
}

func (m *mockShareProvider) Save(_ context.Context, _ domain.Share) error {
	return nil
}

func (m *mockShareProvider) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockShareProvider) List(ctx context.Context) ([]domain.Share, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func TestDeleteByTargetShouldReturnErrorWhenDeleteFails(t *testing.T) {
	errDelete := errors.New("delete failed")
	mock := &mockShareProvider{
		listFn: func(_ context.Context) ([]domain.Share, error) {
			s := domain.Share{}
			s.ID = "1"
			s.TargetType = domain.ShareTargetItem
			s.TargetID = "item1"
			return []domain.Share{s}, nil
		},
		deleteFn: func(_ context.Context, _ string) error {
			return errDelete
		},
	}
	r := &ShareRepository{provider: mock}

	err := r.DeleteByTarget(t.Context(), domain.ShareTargetItem, "item1")
	require.ErrorIs(t, err, errDelete)
}
