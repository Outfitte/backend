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

// outfitDB is the subset of *sql.DB methods used by OutfitRepository.
// Accepting this interface instead of *sql.DB allows test doubles to be injected.
type outfitDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.OutfitRepository = (*OutfitRepository)(nil)

// OutfitRepository is a SQL-backed implementation of ports.OutfitRepository.
type OutfitRepository struct {
	db outfitDB
}

// NewOutfitRepository creates an OutfitRepository backed by the given db.
func NewOutfitRepository(db outfitDB) *OutfitRepository {
	return &OutfitRepository{db: db}
}

// Get retrieves a single outfit by ID, including its items and photos.
// Returns domain.ErrNotFound if no outfit with that ID exists.
func (r *OutfitRepository) Get(ctx context.Context, id string) (domain.Outfit, error) {
	if err := ctx.Err(); err != nil {
		return domain.Outfit{}, err
	}
	const q = `SELECT id, owner_id, name, notes, created_at FROM outfits WHERE id = ?`
	outfit, err := scanOutfitRow(r.db.QueryRowContext(ctx, q, id))
	if err != nil {
		return domain.Outfit{}, err
	}
	outfits := []domain.Outfit{outfit}
	if err := r.batchLoadItems(ctx, outfits); err != nil {
		return domain.Outfit{}, err
	}
	if err := r.batchLoadOutfitPhotos(ctx, outfits); err != nil {
		return domain.Outfit{}, err
	}
	return outfits[0], nil
}

// Save creates or updates the outfit. It does NOT touch item or photo entries.
func (r *OutfitRepository) Save(ctx context.Context, outfit domain.Outfit) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO outfits (id, owner_id, name, notes, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			owner_id = excluded.owner_id,
			name     = excluded.name,
			notes    = excluded.notes`
	_, err := r.db.ExecContext(ctx, q,
		outfit.ID, outfit.OwnerID,
		nullableString(outfit.Name),
		nullableString(outfit.Notes),
		outfit.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// Delete removes the outfit identified by id, including all its associated items and photos.
// Returns domain.ErrNotFound if no outfit with that ID exists.
func (r *OutfitRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM outfits WHERE id = ?`
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

// ListByOwner returns all outfits belonging to ownerID, each including its items and photos.
func (r *OutfitRepository) ListByOwner(ctx context.Context, ownerID string) ([]domain.Outfit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	outfits, err := r.queryOutfitsByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if err := r.batchLoadItems(ctx, outfits); err != nil {
		return nil, err
	}
	if err := r.batchLoadOutfitPhotos(ctx, outfits); err != nil {
		return nil, err
	}
	return outfits, nil
}

// SaveItem creates or updates the association between outfitID and itemID.
func (r *OutfitRepository) SaveItem(ctx context.Context, outfitID, itemID string, position int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO outfit_items (outfit_id, item_id, position)
		VALUES (?, ?, ?)
		ON CONFLICT(outfit_id, item_id) DO UPDATE SET position = excluded.position`
	_, err := r.db.ExecContext(ctx, q, outfitID, itemID, position)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// DeleteItem removes the association between outfitID and itemID.
// This operation is idempotent: deleting a non-existent association is not an error.
func (r *OutfitRepository) DeleteItem(ctx context.Context, outfitID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM outfit_items WHERE outfit_id = ? AND item_id = ?`
	_, err := r.db.ExecContext(ctx, q, outfitID, itemID)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// ListItemIDs returns the ordered item IDs for outfitID.
func (r *OutfitRepository) ListItemIDs(ctx context.Context, outfitID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT item_id FROM outfit_items WHERE outfit_id = ? ORDER BY position ASC`
	rows, err := r.db.QueryContext(ctx, q, outfitID)
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

// SavePhoto saves a photo entry linking photoID and mediaKey to outfitID at the given position.
func (r *OutfitRepository) SavePhoto(ctx context.Context, outfitID, photoID, mediaKey string, position int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO outfit_photos (id, outfit_id, media_key, position, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET position = excluded.position`
	_, err := r.db.ExecContext(ctx, q,
		photoID, outfitID, mediaKey, position,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// DeletePhoto removes the photo entry identified by outfitID + mediaKey.
func (r *OutfitRepository) DeletePhoto(ctx context.Context, outfitID, mediaKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM outfit_photos WHERE outfit_id = ? AND media_key = ?`
	_, err := r.db.ExecContext(ctx, q, outfitID, mediaKey)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// ── private helpers ───────────────────────────────────────────────────────────

func nullableString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func scanOutfitRow(row *sql.Row) (domain.Outfit, error) {
	var (
		id        string
		ownerID   string
		name      sql.NullString
		notes     sql.NullString
		createdAt string
	)
	if err := row.Scan(&id, &ownerID, &name, &notes, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Outfit{}, domain.ErrNotFound
		}
		return domain.Outfit{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domain.Outfit{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return buildOutfit(id, ownerID, name, notes, parsedCreatedAt), nil
}

func buildOutfit(id, ownerID string, name, notes sql.NullString, createdAt time.Time) domain.Outfit {
	var o domain.Outfit
	o.ID = id
	o.OwnerID = ownerID
	if name.Valid {
		o.Name = &name.String
	}
	if notes.Valid {
		o.Notes = &notes.String
	}
	o.CreatedAt = createdAt
	o.Items = []domain.OutfitItem{}
	o.Photos = []domain.OutfitPhoto{}
	return o
}

func (r *OutfitRepository) batchLoadItems(ctx context.Context, outfits []domain.Outfit) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(outfits) == 0 {
		return nil
	}
	ids := make([]string, len(outfits))
	for i, o := range outfits {
		ids[i] = o.ID
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	q := fmt.Sprintf(`
		SELECT outfit_id, item_id, position
		FROM outfit_items
		WHERE outfit_id IN (%s)
		ORDER BY position ASC`, placeholders)

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	itemsByOutfitID := make(map[string][]domain.OutfitItem)
	for rows.Next() {
		var outfitID, itemID string
		var position int
		if err := rows.Scan(&outfitID, &itemID, &position); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		itemsByOutfitID[outfitID] = append(itemsByOutfitID[outfitID], domain.OutfitItem{
			OutfitID: outfitID,
			ItemID:   itemID,
			Position: position,
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	for i, o := range outfits {
		if items, ok := itemsByOutfitID[o.ID]; ok {
			outfits[i].Items = items
		}
	}
	return nil
}

func (r *OutfitRepository) batchLoadOutfitPhotos(ctx context.Context, outfits []domain.Outfit) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(outfits) == 0 {
		return nil
	}
	ids := make([]string, len(outfits))
	for i, o := range outfits {
		ids[i] = o.ID
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	q := fmt.Sprintf(`
		SELECT outfit_id, id, media_key, position, created_at
		FROM outfit_photos
		WHERE outfit_id IN (%s)
		ORDER BY position ASC`, placeholders)

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	photosByOutfitID := make(map[string][]domain.OutfitPhoto)
	for rows.Next() {
		var outfitID, id, mediaKey string
		var position int
		var createdAtStr string
		if err := rows.Scan(&outfitID, &id, &mediaKey, &position, &createdAtStr); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
		if err != nil {
			return fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		photosByOutfitID[outfitID] = append(photosByOutfitID[outfitID], domain.OutfitPhoto{
			ID:        id,
			MediaKey:  mediaKey,
			Position:  position,
			CreatedAt: parsedCreatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	for i, o := range outfits {
		if photos, ok := photosByOutfitID[o.ID]; ok {
			outfits[i].Photos = photos
		}
	}
	return nil
}

func (r *OutfitRepository) queryOutfitsByOwner(ctx context.Context, ownerID string) ([]domain.Outfit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `
		SELECT id, owner_id, name, notes, created_at
		FROM outfits
		WHERE owner_id = ?
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, ownerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	outfits := []domain.Outfit{}
	for rows.Next() {
		var (
			id        string
			ownerID   string
			name      sql.NullString
			notes     sql.NullString
			createdAt string
		)
		if err := rows.Scan(&id, &ownerID, &name, &notes, &createdAt); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		outfits = append(outfits, buildOutfit(id, ownerID, name, notes, parsedCreatedAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return outfits, nil
}
