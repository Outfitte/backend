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

// sessionDB is the subset of *sql.DB methods used by SessionRepository.
// Accepting this interface instead of *sql.DB allows test doubles to be injected.
type sessionDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.SessionRepository = (*SessionRepository)(nil)

// SessionRepository is a SQL-backed implementation of ports.SessionRepository.
type SessionRepository struct {
	db sessionDB
}

// NewSessionRepository creates a SessionRepository backed by the given db.
func NewSessionRepository(db sessionDB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Get retrieves a single session by ID.
// Returns domain.ErrNotFound if no session with that ID exists.
func (r *SessionRepository) Get(ctx context.Context, id string) (domain.Session, error) {
	if err := ctx.Err(); err != nil {
		return domain.Session{}, err
	}
	const q = `SELECT id, user_id, token_hash, expires_at, created_at FROM sessions WHERE id = ?`
	return scanSessionRow(r.db.QueryRowContext(ctx, q, id))
}

// Save upserts the session row.
func (r *SessionRepository) Save(ctx context.Context, session domain.Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			user_id    = excluded.user_id,
			token_hash = excluded.token_hash,
			expires_at = excluded.expires_at`
	_, err := r.db.ExecContext(ctx, q,
		session.ID, session.UserID, session.TokenHash,
		session.ExpiresAt.UTC().Format(time.RFC3339),
		session.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// Delete removes the session row identified by id.
// Returns domain.ErrNotFound if no session with that ID exists.
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM sessions WHERE id = ?`
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

// FindByTokenHash retrieves a session by its token hash.
// Returns domain.ErrNotFound if no session with that hash exists.
func (r *SessionRepository) FindByTokenHash(ctx context.Context, hash string) (domain.Session, error) {
	if err := ctx.Err(); err != nil {
		return domain.Session{}, err
	}
	const q = `SELECT id, user_id, token_hash, expires_at, created_at FROM sessions WHERE token_hash = ?`
	return scanSessionRow(r.db.QueryRowContext(ctx, q, hash))
}

// CountByUser returns the total number of sessions for the given userID.
// Returns 0, nil if the user has no sessions or does not exist.
func (r *SessionRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	const q = `SELECT COUNT(*) FROM sessions WHERE user_id = ?`
	var count int
	if err := r.db.QueryRowContext(ctx, q, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return count, nil
}

// DeleteOldestByUser removes the oldest session belonging to userID.
// Returns domain.ErrNotFound if the user has no sessions.
func (r *SessionRepository) DeleteOldestByUser(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM sessions WHERE id = (SELECT id FROM sessions WHERE user_id = ? ORDER BY created_at ASC LIMIT 1)`
	result, err := r.db.ExecContext(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if n == 0 {
		return fmt.Errorf("%w: no sessions for user %s", domain.ErrNotFound, userID)
	}
	return nil
}

// scanSessionRow scans a single *sql.Row into a domain.Session.
func scanSessionRow(row *sql.Row) (domain.Session, error) {
	var (
		id        string
		userID    string
		tokenHash string
		expiresAt string
		createdAt string
	)
	if err := row.Scan(&id, &userID, &tokenHash, &expiresAt, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Session{}, domain.ErrNotFound
		}
		return domain.Session{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedExpiresAt, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return domain.Session{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return domain.Session{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	var s domain.Session
	s.ID = id
	s.UserID = userID
	s.TokenHash = tokenHash
	s.ExpiresAt = parsedExpiresAt
	s.CreatedAt = parsedCreatedAt
	return s, nil
}
