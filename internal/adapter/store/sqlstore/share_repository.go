package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// shareDB is the subset of *sql.DB methods used by ShareRepository.
// Accepting this interface instead of *sql.DB allows test doubles to be injected.
type shareDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.ShareRepository = (*ShareRepository)(nil)

// ShareRepository is a SQL-backed implementation of ports.ShareRepository.
type ShareRepository struct {
	db shareDB
}

// NewShareRepository creates a ShareRepository backed by the given db.
func NewShareRepository(db shareDB) *ShareRepository {
	return &ShareRepository{db: db}
}

// Get retrieves a single share by ID.
func (r *ShareRepository) Get(_ context.Context, _ string) (domain.Share, error) {
	return domain.Share{}, errors.New("not implemented")
}

// Save creates or updates a share entry.
func (r *ShareRepository) Save(_ context.Context, _ domain.Share) error {
	return errors.New("not implemented")
}

// Delete removes the share identified by id.
func (r *ShareRepository) Delete(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

// ListByOwner returns all outgoing shares created by ownerID.
func (r *ShareRepository) ListByOwner(_ context.Context, _ string) ([]domain.Share, error) {
	return nil, errors.New("not implemented")
}

// ListByRecipient returns all incoming shares for recipientID.
func (r *ShareRepository) ListByRecipient(_ context.Context, _ string) ([]domain.Share, error) {
	return nil, errors.New("not implemented")
}

// FindByTarget returns the share matching owner, recipient, target type, and target ID.
func (r *ShareRepository) FindByTarget(_ context.Context, _, _ string, _ domain.ShareTargetType, _ string) (*domain.Share, error) {
	return nil, errors.New("not implemented")
}

// DeleteByTarget removes all shares for the given target type and target ID.
func (r *ShareRepository) DeleteByTarget(_ context.Context, _ domain.ShareTargetType, _ string) error {
	return errors.New("not implemented")
}

// HasDirectAccess reports whether a share entry exists granting recipientID direct access.
func (r *ShareRepository) HasDirectAccess(_ context.Context, _ string, _ domain.ShareTargetType, _ string) (bool, error) {
	return false, errors.New("not implemented")
}

// ListByRecipientAndType returns all incoming shares of a specific type for recipientID.
func (r *ShareRepository) ListByRecipientAndType(_ context.Context, _ string, _ domain.ShareTargetType) ([]domain.Share, error) {
	return nil, errors.New("not implemented")
}
