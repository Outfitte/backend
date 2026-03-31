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

// wearLogDB is the subset of *sql.DB methods used by WearLogRepository.
// Accepting this interface instead of *sql.DB allows test doubles to be injected.
type wearLogDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.WearLogRepository = (*WearLogRepository)(nil)

// WearLogRepository is a SQL-backed implementation of ports.WearLogRepository.
type WearLogRepository struct {
	db wearLogDB
}

// NewWearLogRepository creates a WearLogRepository backed by the given db.
func NewWearLogRepository(db wearLogDB) *WearLogRepository {
	return &WearLogRepository{db: db}
}

// Get retrieves a single wear log by ID.
// Returns domain.ErrNotFound if no wear log with that ID exists.
func (r *WearLogRepository) Get(ctx context.Context, id string) (domain.WearLog, error) {
	if err := ctx.Err(); err != nil {
		return domain.WearLog{}, err
	}
	const q = `SELECT id, item_id, owner_id, worn_on, notes, created_at FROM wear_logs WHERE id = ?`
	return scanWearLogRow(r.db.QueryRowContext(ctx, q, id))
}

// Save upserts the wear log row.
func (r *WearLogRepository) Save(ctx context.Context, log domain.WearLog) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			item_id  = excluded.item_id,
			owner_id = excluded.owner_id,
			worn_on  = excluded.worn_on,
			notes    = excluded.notes`
	_, err := r.db.ExecContext(ctx, q,
		log.ID, log.ItemID, log.OwnerID,
		log.WornOn.Format("2006-01-02"),
		log.Notes,
		log.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// Delete removes the wear log row identified by id.
// Returns domain.ErrNotFound if no wear log with that ID exists.
func (r *WearLogRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM wear_logs WHERE id = ?`
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

// ListByItem returns all wear logs for itemID, ordered by worn_on descending.
func (r *WearLogRepository) ListByItem(ctx context.Context, itemID string) ([]domain.WearLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `
		SELECT id, item_id, owner_id, worn_on, notes, created_at
		FROM wear_logs
		WHERE item_id = ?
		ORDER BY worn_on DESC, created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, itemID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()
	return scanWearLogRows(rows)
}

// LatestByItem returns the most recent wear log for itemID, or nil if none exist.
func (r *WearLogRepository) LatestByItem(ctx context.Context, itemID string) (*domain.WearLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `
		SELECT id, item_id, owner_id, worn_on, notes, created_at
		FROM wear_logs
		WHERE item_id = ?
		ORDER BY worn_on DESC, created_at DESC
		LIMIT 1`
	log, err := scanWearLogRow(r.db.QueryRowContext(ctx, q, itemID))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

// CountByItem returns the number of wear logs for itemID.
func (r *WearLogRepository) CountByItem(ctx context.Context, itemID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	const q = `SELECT COUNT(*) FROM wear_logs WHERE item_id = ?`
	var count int
	if err := r.db.QueryRowContext(ctx, q, itemID).Scan(&count); err != nil {
		return 0, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return count, nil
}

// scanWearLogRow scans a single *sql.Row into a domain.WearLog.
func scanWearLogRow(row *sql.Row) (domain.WearLog, error) {
	var (
		id        string
		itemID    string
		ownerID   string
		wornOn    string
		notes     sql.NullString
		createdAt string
	)
	if err := row.Scan(&id, &itemID, &ownerID, &wornOn, &notes, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.WearLog{}, domain.ErrNotFound
		}
		return domain.WearLog{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedWornOn, err := time.Parse("2006-01-02", wornOn)
	if err != nil {
		return domain.WearLog{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return domain.WearLog{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return buildWearLog(id, itemID, ownerID, notes, parsedWornOn, parsedCreatedAt), nil
}

// scanWearLogRows scans a *sql.Rows cursor into a slice of domain.WearLog.
func scanWearLogRows(rows *sql.Rows) ([]domain.WearLog, error) {
	logs := []domain.WearLog{}
	for rows.Next() {
		var (
			id        string
			itemID    string
			ownerID   string
			wornOn    string
			notes     sql.NullString
			createdAt string
		)
		if err := rows.Scan(&id, &itemID, &ownerID, &wornOn, &notes, &createdAt); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedWornOn, err := time.Parse("2006-01-02", wornOn)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		logs = append(logs, buildWearLog(id, itemID, ownerID, notes, parsedWornOn, parsedCreatedAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return logs, nil
}

// buildWearLog constructs a domain.WearLog from scanned column values.
func buildWearLog(id, itemID, ownerID string, notes sql.NullString, wornOn, createdAt time.Time) domain.WearLog {
	var log domain.WearLog
	log.ID = id
	log.ItemID = itemID
	log.OwnerID = ownerID
	if notes.Valid {
		log.Notes = &notes.String
	}
	log.WornOn = wornOn
	log.CreatedAt = createdAt
	return log
}
