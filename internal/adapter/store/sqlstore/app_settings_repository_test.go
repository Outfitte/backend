package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/adapter/store/sqlstore"
	"github.com/outfitte/outfitte/internal/domain"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newAppSettingsRepo(t *testing.T) (*sqlstore.AppSettingsRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewAppSettingsRepository(db), db
}

// ── Load ──────────────────────────────────────────────────────────────────────

func TestAppSettingsRepositoryLoadShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newAppSettingsRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Load(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestAppSettingsRepositoryLoadShouldReturnErrNotFoundWhenNoRowExists(t *testing.T) {
	repo, _ := newAppSettingsRepo(t)

	_, err := repo.Load(t.Context())
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAppSettingsRepositoryLoadShouldReturnSettingsWhenRowExists(t *testing.T) {
	repo, db := newAppSettingsRepo(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO app_settings (id, registration_enabled) VALUES ('app_settings', 1)`)
	require.NoError(t, err)

	got, err := repo.Load(t.Context())
	require.NoError(t, err)
	require.True(t, got.RegistrationEnabled)
}

func TestAppSettingsRepositoryLoadShouldReturnRegistrationDisabledWhenStoredAsFalse(t *testing.T) {
	repo, db := newAppSettingsRepo(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO app_settings (id, registration_enabled) VALUES ('app_settings', 0)`)
	require.NoError(t, err)

	got, err := repo.Load(t.Context())
	require.NoError(t, err)
	require.False(t, got.RegistrationEnabled)
}

func TestAppSettingsRepositoryLoadShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewAppSettingsRepository(db)
	db.Close()

	_, err := repo.Load(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestAppSettingsRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newAppSettingsRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.Save(ctx, domain.AppSettings{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestAppSettingsRepositorySaveShouldInsertNewSettings(t *testing.T) {
	repo, _ := newAppSettingsRepo(t)

	err := repo.Save(t.Context(), domain.AppSettings{RegistrationEnabled: true})
	require.NoError(t, err)

	got, err := repo.Load(t.Context())
	require.NoError(t, err)
	require.True(t, got.RegistrationEnabled)
}

func TestAppSettingsRepositorySaveShouldUpdateExistingSettings(t *testing.T) {
	repo, db := newAppSettingsRepo(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO app_settings (id, registration_enabled) VALUES ('app_settings', 1)`)
	require.NoError(t, err)

	err = repo.Save(t.Context(), domain.AppSettings{RegistrationEnabled: false})
	require.NoError(t, err)

	got, err := repo.Load(t.Context())
	require.NoError(t, err)
	require.False(t, got.RegistrationEnabled)
}

func TestAppSettingsRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewAppSettingsRepository(db)
	db.Close()

	err := repo.Save(t.Context(), domain.AppSettings{})
	require.ErrorIs(t, err, domain.ErrIO)
}
