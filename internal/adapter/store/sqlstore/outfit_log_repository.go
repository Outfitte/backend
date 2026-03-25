package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// outfitLogDB is the subset of *sql.DB methods used by OutfitLogRepository.
// Accepting this interface instead of *sql.DB allows test doubles to be injected.
type outfitLogDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.OutfitLogRepository = (*OutfitLogRepository)(nil)

// OutfitLogRepository is a SQL-backed implementation of ports.OutfitLogRepository.
type OutfitLogRepository struct {
	db outfitLogDB
}

// NewOutfitLogRepository creates an OutfitLogRepository backed by the given db.
func NewOutfitLogRepository(db outfitLogDB) *OutfitLogRepository {
	return &OutfitLogRepository{db: db}
}

// Get retrieves a single outfit log by ID, including its linked wear log IDs.
// Returns domain.ErrNotFound if no outfit log with that ID exists.
func (r *OutfitLogRepository) Get(ctx context.Context, id string) (domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return domain.OutfitLog{}, err
	}
	const q = `SELECT id, outfit_id, owner_id, worn_on, notes, created_at FROM outfit_logs WHERE id = ?`
	log, err := scanOutfitLogRow(r.db.QueryRowContext(ctx, q, id))
	if err != nil {
		return domain.OutfitLog{}, err
	}
	logs := []domain.OutfitLog{log}
	if err := r.batchLoadWearLogIDs(ctx, logs); err != nil {
		return domain.OutfitLog{}, err
	}
	return logs[0], nil
}

// Save persists the outfit log (not its wear log links).
func (r *OutfitLogRepository) Save(ctx context.Context, log domain.OutfitLog) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			outfit_id = excluded.outfit_id,
			owner_id  = excluded.owner_id,
			worn_on   = excluded.worn_on,
			notes     = excluded.notes`
	_, err := r.db.ExecContext(ctx, q,
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

// Delete removes the outfit log identified by id, including all wear log associations.
// Returns domain.ErrNotFound if no outfit log with that ID exists.
func (r *OutfitLogRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM outfit_logs WHERE id = ?`
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

// ListByOutfit returns all logs for outfitID, sorted by WornOn descending then created_at descending.
func (r *OutfitLogRepository) ListByOutfit(ctx context.Context, outfitID string) ([]domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `
		SELECT id, outfit_id, owner_id, worn_on, notes, created_at
		FROM outfit_logs
		WHERE outfit_id = ?
		ORDER BY worn_on DESC, created_at DESC`
	logs, err := r.queryOutfitLogs(ctx, q, outfitID)
	if err != nil {
		return nil, err
	}
	if err := r.batchLoadWearLogIDs(ctx, logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// ListByOwnerDateRange returns logs belonging to ownerID where WornOn is within [from, to],
// sorted by WornOn ascending then created_at ascending.
func (r *OutfitLogRepository) ListByOwnerDateRange(ctx context.Context, ownerID string, from, to time.Time) ([]domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `
		SELECT id, outfit_id, owner_id, worn_on, notes, created_at
		FROM outfit_logs
		WHERE owner_id = ? AND worn_on >= ? AND worn_on <= ?
		ORDER BY worn_on ASC, created_at ASC`
	logs, err := r.queryOutfitLogs(ctx, q, ownerID, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	if err := r.batchLoadWearLogIDs(ctx, logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// LinkWearLog associates a wear log with an outfit log.
func (r *OutfitLogRepository) LinkWearLog(ctx context.Context, outfitLogID, wearLogID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, q, outfitLogID, wearLogID)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// LinkedWearLogIDs returns the wear log IDs associated with the given outfit log.
func (r *OutfitLogRepository) LinkedWearLogIDs(ctx context.Context, outfitLogID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT wear_log_id FROM outfit_log_wear_logs WHERE outfit_log_id = ?`
	rows, err := r.db.QueryContext(ctx, q, outfitLogID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	ids := []string{}
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

// ── private helpers ───────────────────────────────────────────────────────────

func (r *OutfitLogRepository) queryOutfitLogs(ctx context.Context, query string, args ...any) ([]domain.OutfitLog, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	logs := []domain.OutfitLog{}
	for rows.Next() {
		var (
			id        string
			outfitID  string
			ownerID   string
			wornOn    string
			notes     sql.NullString
			createdAt string
		)
		if err := rows.Scan(&id, &outfitID, &ownerID, &wornOn, &notes, &createdAt); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		log, err := buildOutfitLog(id, outfitID, ownerID, notes, wornOn, createdAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return logs, nil
}

func scanOutfitLogRow(row *sql.Row) (domain.OutfitLog, error) {
	var (
		id        string
		outfitID  string
		ownerID   string
		wornOn    string
		notes     sql.NullString
		createdAt string
	)
	if err := row.Scan(&id, &outfitID, &ownerID, &wornOn, &notes, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.OutfitLog{}, domain.ErrNotFound
		}
		return domain.OutfitLog{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return buildOutfitLog(id, outfitID, ownerID, notes, wornOn, createdAt)
}

func buildOutfitLog(id, outfitID, ownerID string, notes sql.NullString, wornOnStr, createdAtStr string) (domain.OutfitLog, error) {
	parsedWornOn, err := time.Parse("2006-01-02", wornOnStr)
	if err != nil {
		return domain.OutfitLog{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return domain.OutfitLog{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	var log domain.OutfitLog
	log.ID = id
	log.OutfitID = outfitID
	log.OwnerID = ownerID
	log.WornOn = parsedWornOn
	if notes.Valid {
		log.Notes = &notes.String
	}
	log.WearLogIDs = []string{}
	log.CreatedAt = parsedCreatedAt
	return log, nil
}

func (r *OutfitLogRepository) batchLoadWearLogIDs(ctx context.Context, logs []domain.OutfitLog) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(logs) == 0 {
		return nil
	}
	ids := make([]string, len(logs))
	for i, l := range logs {
		ids[i] = l.GetID()
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	q := fmt.Sprintf(
		`SELECT outfit_log_id, wear_log_id FROM outfit_log_wear_logs WHERE outfit_log_id IN (%s)`,
		placeholders,
	)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	wearLogsByLogID := make(map[string][]string)
	for rows.Next() {
		var outfitLogID, wearLogID string
		if err := rows.Scan(&outfitLogID, &wearLogID); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		wearLogsByLogID[outfitLogID] = append(wearLogsByLogID[outfitLogID], wearLogID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	for i, l := range logs {
		if ids, ok := wearLogsByLogID[l.GetID()]; ok {
			logs[i].WearLogIDs = ids
		}
	}
	return nil
}
