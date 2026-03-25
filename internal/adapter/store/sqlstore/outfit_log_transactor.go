package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// outfitLogTxDB is the subset of *sql.DB used by OutfitLogTransactor.
// Accepting this interface instead of *sql.DB allows test doubles to be injected.
type outfitLogTxDB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

var _ ports.OutfitLogTransactor = (*OutfitLogTransactor)(nil)

// OutfitLogTransactor is a SQL-backed implementation of ports.OutfitLogTransactor.
type OutfitLogTransactor struct {
	db outfitLogTxDB
}

// NewOutfitLogTransactor creates an OutfitLogTransactor backed by the given db.
func NewOutfitLogTransactor(db outfitLogTxDB) *OutfitLogTransactor {
	return &OutfitLogTransactor{db: db}
}

// CreateOutfitLog atomically creates the outfit log, creates one wear log
// per item in wearLogs, creates the associations linking them, and returns
// the persisted outfit log with its WearLogIDs populated.
func (t *OutfitLogTransactor) CreateOutfitLog(ctx context.Context, log domain.OutfitLog, wearLogs []domain.WearLog) (domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return domain.OutfitLog{}, err
	}
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.OutfitLog{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := insertOutfitLog(ctx, tx, log); err != nil {
		return domain.OutfitLog{}, err
	}

	wearLogIDs := make([]string, 0, len(wearLogs))
	for _, wl := range wearLogs {
		if err := insertWearLog(ctx, tx, wl); err != nil {
			return domain.OutfitLog{}, err
		}
		if err := insertOutfitLogWearLink(ctx, tx, log.GetID(), wl.ID); err != nil {
			return domain.OutfitLog{}, err
		}
		wearLogIDs = append(wearLogIDs, wl.ID)
	}

	if err := tx.Commit(); err != nil {
		return domain.OutfitLog{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	log.WearLogIDs = wearLogIDs
	return log, nil
}

func insertOutfitLog(ctx context.Context, tx *sql.Tx, log domain.OutfitLog) error {
	const q = `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, q,
		log.GetID(), log.OutfitID, log.OwnerID,
		log.WornOn.Format("2006-01-02"),
		nullableString(log.Notes),
		log.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

func insertWearLog(ctx context.Context, tx *sql.Tx, wl domain.WearLog) error {
	const q = `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, q,
		wl.ID, wl.ItemID, wl.OwnerID,
		wl.WornOn.Format("2006-01-02"),
		wl.Notes,
		wl.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

func insertOutfitLogWearLink(ctx context.Context, tx *sql.Tx, outfitLogID, wearLogID string) error {
	const q = `INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES (?, ?)`
	_, err := tx.ExecContext(ctx, q, outfitLogID, wearLogID)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// DeleteOutfitLog atomically deletes the outfit log, all of its wear log
// associations, and all linked wear logs.
func (t *OutfitLogTransactor) DeleteOutfitLog(ctx context.Context, outfitLogID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer tx.Rollback() //nolint:errcheck

	wearLogIDs, err := selectLinkedWearLogIDs(ctx, tx, outfitLogID)
	if err != nil {
		return err
	}

	if len(wearLogIDs) > 0 {
		if err := deleteWearLogsByIDs(ctx, tx, wearLogIDs); err != nil {
			return err
		}
	}

	if err := deleteOutfitLogByID(ctx, tx, outfitLogID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

func selectLinkedWearLogIDs(ctx context.Context, tx *sql.Tx, outfitLogID string) ([]string, error) {
	const q = `SELECT wear_log_id FROM outfit_log_wear_logs WHERE outfit_log_id = ?`
	rows, err := tx.QueryContext(ctx, q, outfitLogID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return ids, nil
}

func deleteWearLogsByIDs(ctx context.Context, tx *sql.Tx, ids []string) error {
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	q := fmt.Sprintf("DELETE FROM wear_logs WHERE id IN (%s)", placeholders)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

func deleteOutfitLogByID(ctx context.Context, tx *sql.Tx, outfitLogID string) error {
	const q = `DELETE FROM outfit_logs WHERE id = ?`
	result, err := tx.ExecContext(ctx, q, outfitLogID)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if n == 0 {
		return fmt.Errorf("%w: id %s", domain.ErrNotFound, outfitLogID)
	}
	return nil
}

// UpdateOutfitLogDate atomically updates the outfit log's worn_on date
// and propagates the new date to all linked wear logs.
func (t *OutfitLogTransactor) UpdateOutfitLogDate(ctx context.Context, outfitLogID string, newDate time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer tx.Rollback() //nolint:errcheck

	wearLogIDs, err := selectLinkedWearLogIDs(ctx, tx, outfitLogID)
	if err != nil {
		return err
	}

	if err := updateOutfitLogDate(ctx, tx, outfitLogID, newDate); err != nil {
		return err
	}

	if len(wearLogIDs) > 0 {
		if err := updateWearLogsDates(ctx, tx, wearLogIDs, newDate); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

func updateOutfitLogDate(ctx context.Context, tx *sql.Tx, outfitLogID string, newDate time.Time) error {
	const q = `UPDATE outfit_logs SET worn_on = ? WHERE id = ?`
	if _, err := tx.ExecContext(ctx, q, newDate.Format("2006-01-02"), outfitLogID); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

func updateWearLogsDates(ctx context.Context, tx *sql.Tx, ids []string, newDate time.Time) error {
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	q := fmt.Sprintf("UPDATE wear_logs SET worn_on = ? WHERE id IN (%s)", placeholders)
	args := make([]any, 0, 1+len(ids))
	args = append(args, newDate.Format("2006-01-02"))
	for _, id := range ids {
		args = append(args, id)
	}
	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}
