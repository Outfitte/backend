package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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
func (r *ShareRepository) Get(ctx context.Context, id string) (domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return domain.Share{}, err
	}
	const q = `SELECT id, owner_id, recipient_id, target_type, target_id, created_at FROM shares WHERE id = ?`
	return scanShareRow(r.db.QueryRowContext(ctx, q, id))
}

// Save creates or updates a share entry.
func (r *ShareRepository) Save(ctx context.Context, share domain.Share) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO shares (id, owner_id, recipient_id, target_type, target_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			owner_id     = excluded.owner_id,
			recipient_id = excluded.recipient_id,
			target_type  = excluded.target_type,
			target_id    = excluded.target_id`
	_, err := r.db.ExecContext(ctx, q,
		share.ID, share.OwnerID, share.RecipientID,
		share.TargetType, share.TargetID,
		share.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// Delete removes the share identified by id.
func (r *ShareRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM shares WHERE id = ?`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if n == 0 {
		return fmt.Errorf("%w: id %s", domain.ErrNotFound, id)
	}
	return nil
}

// ListByOwner returns all outgoing shares created by ownerID.
func (r *ShareRepository) ListByOwner(ctx context.Context, ownerID string) ([]domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT id, owner_id, recipient_id, target_type, target_id, created_at FROM shares WHERE owner_id = ?`
	rows, err := r.db.QueryContext(ctx, q, ownerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()
	return scanShareRows(rows)
}

// ListByRecipient returns all incoming shares for recipientID.
func (r *ShareRepository) ListByRecipient(ctx context.Context, recipientID string) ([]domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT id, owner_id, recipient_id, target_type, target_id, created_at FROM shares WHERE recipient_id = ?`
	rows, err := r.db.QueryContext(ctx, q, recipientID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()
	return scanShareRows(rows)
}

// FindByTarget returns the share matching owner, recipient, target type, and target ID.
func (r *ShareRepository) FindByTarget(ctx context.Context, ownerID, recipientID string, targetType domain.ShareTargetType, targetID string) (*domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT id, owner_id, recipient_id, target_type, target_id, created_at FROM shares WHERE owner_id = ? AND recipient_id = ? AND target_type = ? AND target_id = ?`
	share, err := scanShareRow(r.db.QueryRowContext(ctx, q, ownerID, recipientID, targetType, targetID))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &share, nil
}

// DeleteByTarget removes all shares for the given target type and target ID.
func (r *ShareRepository) DeleteByTarget(ctx context.Context, targetType domain.ShareTargetType, targetID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM shares WHERE target_type = ? AND target_id = ?`
	_, err := r.db.ExecContext(ctx, q, targetType, targetID)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// HasDirectAccess reports whether a share entry exists granting recipientID direct access.
func (r *ShareRepository) HasDirectAccess(ctx context.Context, recipientID string, targetType domain.ShareTargetType, targetID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	const q = `SELECT EXISTS(SELECT 1 FROM shares WHERE recipient_id = ? AND target_type = ? AND target_id = ?)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, q, recipientID, targetType, targetID).Scan(&exists); err != nil {
		return false, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return exists, nil
}

// ListByRecipientAndType returns all incoming shares of a specific type for recipientID.
func (r *ShareRepository) ListByRecipientAndType(ctx context.Context, recipientID string, targetType domain.ShareTargetType) ([]domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT id, owner_id, recipient_id, target_type, target_id, created_at FROM shares WHERE recipient_id = ? AND target_type = ?`
	rows, err := r.db.QueryContext(ctx, q, recipientID, targetType)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()
	return scanShareRows(rows)
}

// scanShareRows scans a *sql.Rows cursor into a slice of domain.Share.
func scanShareRows(rows *sql.Rows) ([]domain.Share, error) {
	shares := []domain.Share{}
	for rows.Next() {
		var (
			id          string
			ownerID     string
			recipientID string
			targetType  string
			targetID    string
			createdAt   string
		)
		if err := rows.Scan(&id, &ownerID, &recipientID, &targetType, &targetID, &createdAt); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		shares = append(shares, buildShareFromRow(id, ownerID, recipientID, domain.ShareTargetType(targetType), targetID, parsedCreatedAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return shares, nil
}

// scanShareRow scans a single *sql.Row into a domain.Share.
func scanShareRow(row *sql.Row) (domain.Share, error) {
	var (
		id          string
		ownerID     string
		recipientID string
		targetType  string
		targetID    string
		createdAt   string
	)
	if err := row.Scan(&id, &ownerID, &recipientID, &targetType, &targetID, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Share{}, domain.ErrNotFound
		}
		return domain.Share{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return domain.Share{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return buildShareFromRow(id, ownerID, recipientID, domain.ShareTargetType(targetType), targetID, parsedCreatedAt), nil
}

// buildShareFromRow constructs a domain.Share from scanned column values.
func buildShareFromRow(id, ownerID, recipientID string, targetType domain.ShareTargetType, targetID string, createdAt time.Time) domain.Share {
	var s domain.Share
	s.ID = id
	s.OwnerID = ownerID
	s.RecipientID = recipientID
	s.TargetType = targetType
	s.TargetID = targetID
	s.CreatedAt = createdAt
	return s
}
