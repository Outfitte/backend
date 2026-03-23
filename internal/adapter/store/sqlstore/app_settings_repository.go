package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// appSettingsDB is the subset of *sql.DB methods used by AppSettingsRepository.
type appSettingsDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.AppSettingsRepository = (*AppSettingsRepository)(nil)

// AppSettingsRepository is a SQL-backed implementation of ports.AppSettingsRepository.
type AppSettingsRepository struct {
	db appSettingsDB
}

// NewAppSettingsRepository creates an AppSettingsRepository backed by the given db.
func NewAppSettingsRepository(db appSettingsDB) *AppSettingsRepository {
	return &AppSettingsRepository{db: db}
}

// Load retrieves the singleton application settings.
// Returns domain.ErrNotFound if no settings have been saved yet.
func (r *AppSettingsRepository) Load(ctx context.Context) (domain.AppSettings, error) {
	if err := ctx.Err(); err != nil {
		return domain.AppSettings{}, err
	}
	const q = `SELECT registration_enabled FROM app_settings WHERE id = 'app_settings'`
	var registrationEnabled bool
	if err := r.db.QueryRowContext(ctx, q).Scan(&registrationEnabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AppSettings{}, domain.ErrNotFound
		}
		return domain.AppSettings{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return domain.AppSettings{RegistrationEnabled: registrationEnabled}, nil
}

// Save upserts the singleton application settings.
func (r *AppSettingsRepository) Save(ctx context.Context, settings domain.AppSettings) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO app_settings (id, registration_enabled)
		VALUES ('app_settings', ?)
		ON CONFLICT(id) DO UPDATE SET
			registration_enabled = excluded.registration_enabled`
	if _, err := r.db.ExecContext(ctx, q, settings.RegistrationEnabled); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}
