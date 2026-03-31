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

// userDB is the subset of *sql.DB methods used by UserRepository.
// Accepting this interface instead of *sql.DB allows test doubles to be injected.
type userDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.UserRepository = (*UserRepository)(nil)

// UserRepository is a SQL-backed implementation of ports.UserRepository.
type UserRepository struct {
	db userDB
}

// NewUserRepository creates a UserRepository backed by the given db.
func NewUserRepository(db userDB) *UserRepository {
	return &UserRepository{db: db}
}

// Get retrieves a single user by ID.
// Returns domain.ErrNotFound if no user with that ID exists.
func (r *UserRepository) Get(ctx context.Context, id string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}
	const q = `SELECT id, email, password_hash, role, created_at FROM users WHERE id = ?`
	return scanUserRow(r.db.QueryRowContext(ctx, q, id))
}

// Save upserts the user row.
// Returns domain.ErrConflict if the email is already in use by a different user.
func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email         = excluded.email,
			password_hash = excluded.password_hash,
			role          = excluded.role`
	_, err := r.db.ExecContext(ctx, q,
		user.ID, user.Email, user.PasswordHash, string(user.Role),
		user.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.email") {
			return domain.ErrConflict
		}
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// GetByEmail retrieves a user by email address.
// Returns domain.ErrNotFound if no user with that email exists.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}
	const q = `SELECT id, email, password_hash, role, created_at FROM users WHERE email = ?`
	return scanUserRow(r.db.QueryRowContext(ctx, q, email))
}

// Count returns the total number of users.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	const q = `SELECT COUNT(*) FROM users`
	var count int
	if err := r.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return 0, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return count, nil
}

// List returns all users.
func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT id, email, password_hash, role, created_at FROM users`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	users := []domain.User{}
	for rows.Next() {
		var (
			id           string
			email        string
			passwordHash string
			role         string
			createdAt    string
		)
		if err := rows.Scan(&id, &email, &passwordHash, &role, &createdAt); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		var u domain.User
		u.ID = id
		u.Email = email
		u.PasswordHash = passwordHash
		u.Role = domain.Role(role)
		u.CreatedAt = parsedCreatedAt
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return users, nil
}

// scanUserRow scans a single *sql.Row into a domain.User.
func scanUserRow(row *sql.Row) (domain.User, error) {
	var (
		id           string
		email        string
		passwordHash string
		role         string
		createdAt    string
	)
	if err := row.Scan(&id, &email, &passwordHash, &role, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	var u domain.User
	u.ID = id
	u.Email = email
	u.PasswordHash = passwordHash
	u.Role = domain.Role(role)
	u.CreatedAt = parsedCreatedAt
	return u, nil
}
