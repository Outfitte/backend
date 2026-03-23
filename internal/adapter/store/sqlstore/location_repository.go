package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// locationDB is the subset of *sql.DB methods used by LocationRepository.
// Accepting this interface instead of *sql.DB allows test doubles to be injected.
type locationDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.LocationRepository = (*LocationRepository)(nil)

// LocationRepository is a SQL-backed implementation of ports.LocationRepository.
type LocationRepository struct {
	db locationDB
}

// NewLocationRepository creates a LocationRepository backed by the given db.
func NewLocationRepository(db locationDB) *LocationRepository {
	return &LocationRepository{db: db}
}

// Get retrieves a single location by ID.
// Returns domain.ErrNotFound if no location with that ID exists.
func (r *LocationRepository) Get(ctx context.Context, id string) (domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return domain.Location{}, err
	}
	const q = `SELECT id, owner_id, parent_id, label, created_at FROM locations WHERE id = ?`
	return scanLocationRow(r.db.QueryRowContext(ctx, q, id))
}

// Save upserts the location row.
func (r *LocationRepository) Save(ctx context.Context, location domain.Location) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO locations (id, owner_id, parent_id, label, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			owner_id  = excluded.owner_id,
			parent_id = excluded.parent_id,
			label     = excluded.label`
	_, err := r.db.ExecContext(ctx, q,
		location.ID, location.OwnerID, location.ParentID,
		location.Label,
		location.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// Delete removes the location row identified by id.
// Returns domain.ErrNotFound if no location with that ID exists.
func (r *LocationRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM locations WHERE id = ?`
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

// ListByOwner returns all locations belonging to ownerID.
func (r *LocationRepository) ListByOwner(ctx context.Context, ownerID string) ([]domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT id, owner_id, parent_id, label, created_at FROM locations WHERE owner_id = ?`
	rows, err := r.db.QueryContext(ctx, q, ownerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()
	return scanLocationRows(rows)
}

// HasChildren reports whether locationID has any child locations.
func (r *LocationRepository) HasChildren(ctx context.Context, locationID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	const q = `SELECT EXISTS(SELECT 1 FROM locations WHERE parent_id = ?)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, q, locationID).Scan(&exists); err != nil {
		return false, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return exists, nil
}

// scanLocationRow scans a single *sql.Row into a domain.Location.
func scanLocationRow(row *sql.Row) (domain.Location, error) {
	var (
		id        string
		ownerID   string
		parentID  sql.NullString
		label     string
		createdAt string
	)
	if err := row.Scan(&id, &ownerID, &parentID, &label, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Location{}, domain.ErrNotFound
		}
		return domain.Location{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return domain.Location{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return buildLocation(id, ownerID, parentID, label, parsedCreatedAt), nil
}

// scanLocationRows scans a *sql.Rows cursor into a slice of domain.Location.
func scanLocationRows(rows *sql.Rows) ([]domain.Location, error) {
	locations := []domain.Location{}
	for rows.Next() {
		var (
			id        string
			ownerID   string
			parentID  sql.NullString
			label     string
			createdAt string
		)
		if err := rows.Scan(&id, &ownerID, &parentID, &label, &createdAt); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		locations = append(locations, buildLocation(id, ownerID, parentID, label, parsedCreatedAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return locations, nil
}

// buildLocation constructs a domain.Location from scanned column values.
func buildLocation(id, ownerID string, parentID sql.NullString, label string, createdAt time.Time) domain.Location {
	var loc domain.Location
	loc.ID = id
	loc.OwnerID = ownerID
	if parentID.Valid {
		loc.ParentID = &parentID.String
	}
	loc.Label = label
	loc.CreatedAt = createdAt
	return loc
}
