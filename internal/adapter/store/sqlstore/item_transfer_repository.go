package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// itemTransferDB is the subset of *sql.DB methods used by ItemTransferRepository.
type itemTransferDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.ItemTransferRepository = (*ItemTransferRepository)(nil)

// ItemTransferRepository is a SQL-backed implementation of ports.ItemTransferRepository.
type ItemTransferRepository struct {
	db itemTransferDB
}

// NewItemTransferRepository creates an ItemTransferRepository backed by the given db.
func NewItemTransferRepository(db itemTransferDB) *ItemTransferRepository {
	return &ItemTransferRepository{db: db}
}

// Get retrieves a single transfer by ID.
func (r *ItemTransferRepository) Get(ctx context.Context, id string) (domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return domain.ItemTransfer{}, err
	}
	const q = `SELECT id, item_id, sender_id, recipient_id, status, transfer_history, created_at, decided_at FROM item_transfers WHERE id = ?`
	return scanItemTransferRow(r.db.QueryRowContext(ctx, q, id))
}

// Save creates or updates a transfer entry.
func (r *ItemTransferRepository) Save(ctx context.Context, transfer domain.ItemTransfer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Only status and decided_at are mutable after creation; all other columns are frozen.
	const q = `
		INSERT INTO item_transfers (id, item_id, sender_id, recipient_id, status, transfer_history, created_at, decided_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status     = excluded.status,
			decided_at = excluded.decided_at`
	var decidedAt any
	if transfer.DecidedAt != nil {
		decidedAt = transfer.DecidedAt.UTC().Format(time.RFC3339)
	}
	_, err := r.db.ExecContext(ctx, q,
		transfer.ID, transfer.ItemID, transfer.SenderID, transfer.RecipientID,
		string(transfer.Status), boolToInt(transfer.TransferHistory),
		transfer.CreatedAt.UTC().Format(time.RFC3339), decidedAt,
	)
	if err != nil {
		if isDuplicatePendingTransferError(err) {
			return domain.ErrConflict
		}
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Delete removes the transfer identified by id.
func (r *ItemTransferRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM item_transfers WHERE id = ?`
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

// ListBySender returns outgoing transfers created by senderID.
func (r *ItemTransferRepository) ListBySender(ctx context.Context, senderID string, statusFilter *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.listByField(ctx, "sender_id", senderID, statusFilter)
}

// ListByRecipient returns incoming transfers for recipientID.
func (r *ItemTransferRepository) ListByRecipient(ctx context.Context, recipientID string, statusFilter *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.listByField(ctx, "recipient_id", recipientID, statusFilter)
}

// listByField queries item_transfers filtering by a single column value and optional status.
func (r *ItemTransferRepository) listByField(ctx context.Context, column, value string, statusFilter *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	const base = `SELECT id, item_id, sender_id, recipient_id, status, transfer_history, created_at, decided_at FROM item_transfers WHERE `
	var (
		q    string
		args []any
	)
	if statusFilter != nil {
		q = base + column + ` = ? AND status = ? ORDER BY created_at DESC`
		args = []any{value, string(*statusFilter)}
	} else {
		q = base + column + ` = ? ORDER BY created_at DESC`
		args = []any{value}
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()
	return scanItemTransferRows(rows)
}

// FindPendingByItem returns the pending transfer for itemID, or nil if none exists.
func (r *ItemTransferRepository) FindPendingByItem(ctx context.Context, itemID string) (*domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT id, item_id, sender_id, recipient_id, status, transfer_history, created_at, decided_at FROM item_transfers WHERE item_id = ? AND status = 'pending' LIMIT 1`
	tr, err := scanItemTransferRow(r.db.QueryRowContext(ctx, q, itemID))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tr, nil
}

// HasPending reports whether a pending transfer exists for itemID.
func (r *ItemTransferRepository) HasPending(ctx context.Context, itemID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	const q = `SELECT 1 FROM item_transfers WHERE item_id = ? AND status = 'pending' LIMIT 1`
	var exists int
	if err := r.db.QueryRowContext(ctx, q, itemID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return true, nil
}

// scanItemTransferRows scans a *sql.Rows cursor into a slice of domain.ItemTransfer.
func scanItemTransferRows(rows *sql.Rows) ([]domain.ItemTransfer, error) {
	transfers := []domain.ItemTransfer{}
	for rows.Next() {
		var (
			id              string
			itemID          string
			senderID        string
			recipientID     string
			status          string
			transferHistory int
			createdAt       string
			decidedAtStr    sql.NullString
		)
		if err := rows.Scan(&id, &itemID, &senderID, &recipientID, &status, &transferHistory, &createdAt, &decidedAtStr); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		decidedAt, err := parseOptionalTime(decidedAtStr)
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, buildItemTransferFromRow(id, itemID, senderID, recipientID, domain.TransferStatus(status), transferHistory != 0, parsedCreatedAt, decidedAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return transfers, nil
}

// scanItemTransferRow scans a single *sql.Row into a domain.ItemTransfer.
func scanItemTransferRow(row *sql.Row) (domain.ItemTransfer, error) {
	var (
		id              string
		itemID          string
		senderID        string
		recipientID     string
		status          string
		transferHistory int
		createdAt       string
		decidedAtStr    sql.NullString
	)
	if err := row.Scan(&id, &itemID, &senderID, &recipientID, &status, &transferHistory, &createdAt, &decidedAtStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ItemTransfer{}, domain.ErrNotFound
		}
		return domain.ItemTransfer{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return domain.ItemTransfer{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	decidedAt, err := parseOptionalTime(decidedAtStr)
	if err != nil {
		return domain.ItemTransfer{}, err
	}
	return buildItemTransferFromRow(id, itemID, senderID, recipientID, domain.TransferStatus(status), transferHistory != 0, parsedCreatedAt, decidedAt), nil
}

// parseOptionalTime parses a nullable RFC3339 timestamp into a *time.Time.
func parseOptionalTime(s sql.NullString) (*time.Time, error) {
	if !s.Valid {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s.String)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return &t, nil
}

// buildItemTransferFromRow constructs a domain.ItemTransfer from scanned column values.
func buildItemTransferFromRow(id, itemID, senderID, recipientID string, status domain.TransferStatus, history bool, createdAt time.Time, decidedAt *time.Time) domain.ItemTransfer {
	var tr domain.ItemTransfer
	tr.ID = id
	tr.ItemID = itemID
	tr.SenderID = senderID
	tr.RecipientID = recipientID
	tr.Status = status
	tr.TransferHistory = history
	tr.CreatedAt = createdAt
	tr.DecidedAt = decidedAt
	return tr
}

// isDuplicatePendingTransferError reports whether err is a SQLite violation of the
// partial unique index that enforces at most one pending transfer per item.
func isDuplicatePendingTransferError(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed: item_transfers.item_id")
}
