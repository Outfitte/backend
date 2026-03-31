package sqlstore

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/outfitte/backend/internal/domain"
)

// ── mockQueryFailAfterDB ──────────────────────────────────────────────────────
// Wraps a real outfitLogDB and causes QueryContext to fail after failAfter successes.
// QueryRowContext always delegates to the inner DB (used only for single-row lookups).

type mockQueryFailAfterDB struct {
	inner      outfitLogDB
	queryCount int
	failAfter  int
}

func (m *mockQueryFailAfterDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return m.inner.ExecContext(ctx, query, args...)
}

func (m *mockQueryFailAfterDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	m.queryCount++
	if m.queryCount > m.failAfter {
		return nil, errFakeDB
	}
	return m.inner.QueryContext(ctx, query, args...)
}

func (m *mockQueryFailAfterDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return m.inner.QueryRowContext(ctx, query, args...)
}

// ── Delete: RowsAffected error ────────────────────────────────────────────────

func TestOutfitLogRepositoryDeleteShouldReturnErrIOWhenRowsAffectedFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-aff-err")
	repo := &OutfitLogRepository{db: db}
	err := repo.Delete(t.Context(), "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanOutfitLogRow: scan error (non-ErrNoRows) ──────────────────────────────

func TestScanOutfitLogRowShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)
	row := db.QueryRowContext(t.Context(), "SELECT 1")
	_, err := scanOutfitLogRow(row)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── buildOutfitLog: worn_on parse error ───────────────────────────────────────

func TestBuildOutfitLogShouldReturnErrIOWhenWornOnIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('ol-wb-owner', 'ol-wb@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at) VALUES ('ol-wb-outfit', 'ol-wb-owner', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at)
		VALUES ('ol-wb-bad-worn', 'ol-wb-outfit', 'ol-wb-owner', 'not-a-date', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	repo := &OutfitLogRepository{db: db}
	_, err = repo.Get(t.Context(), "ol-wb-bad-worn")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── buildOutfitLog: created_at parse error ────────────────────────────────────

func TestBuildOutfitLogShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('ol-wb-ca-owner', 'ol-wb-ca@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at) VALUES ('ol-wb-ca-outfit', 'ol-wb-ca-owner', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at)
		VALUES ('ol-wb-bad-ca', 'ol-wb-ca-outfit', 'ol-wb-ca-owner', '2025-06-01', 'not-a-timestamp')`)
	require.NoError(t, err)

	repo := &OutfitLogRepository{db: db}
	_, err = repo.Get(t.Context(), "ol-wb-bad-ca")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── Get: batchLoadWearLogIDs error path ───────────────────────────────────────

func TestGetShouldReturnErrIOWhenBatchLoadWearLogIDsFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('ol-wb-bl-owner', 'ol-wb-bl@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at) VALUES ('ol-wb-bl-outfit', 'ol-wb-bl-owner', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at)
		VALUES ('ol-wb-bl-1', 'ol-wb-bl-outfit', 'ol-wb-bl-owner', '2025-06-01', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	// Get uses QueryRowContext for the row lookup and QueryContext inside batchLoadWearLogIDs.
	// failAfter=0 means the first QueryContext call (in batchLoadWearLogIDs) fails.
	mock := &mockQueryFailAfterDB{inner: db, failAfter: 0}
	repo := &OutfitLogRepository{db: mock}
	_, err = repo.Get(t.Context(), "ol-wb-bl-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── queryOutfitLogs: rows.Err error ───────────────────────────────────────────

func TestQueryOutfitLogsShouldReturnErrIOWhenRowsErrFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &OutfitLogRepository{db: db}
	_, err := repo.queryOutfitLogs(t.Context(), "SELECT 1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── queryOutfitLogs: scan error ───────────────────────────────────────────────

func TestQueryOutfitLogsShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &OutfitLogRepository{db: db}
	_, err := repo.queryOutfitLogs(t.Context(), "SELECT 1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── queryOutfitLogs: buildOutfitLog error propagation ────────────────────────

func TestQueryOutfitLogsShouldReturnErrIOWhenWornOnIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('ol-wb-q-owner', 'ol-wb-q@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at) VALUES ('ol-wb-q-outfit', 'ol-wb-q-owner', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at)
		VALUES ('ol-wb-q-bad', 'ol-wb-q-outfit', 'ol-wb-q-owner', 'bad-date', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	repo := &OutfitLogRepository{db: db}
	_, err = repo.ListByOutfit(t.Context(), "ol-wb-q-outfit")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── ListByOutfit: batchLoadWearLogIDs error path ─────────────────────────────

func TestListByOutfitShouldReturnErrIOWhenBatchLoadWearLogIDsFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('ol-wb-lb-owner', 'ol-wb-lb@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at) VALUES ('ol-wb-lb-outfit', 'ol-wb-lb-owner', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at)
		VALUES ('ol-wb-lb-1', 'ol-wb-lb-outfit', 'ol-wb-lb-owner', '2025-06-01', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	// ListByOutfit uses QueryContext for the row scan and again inside batchLoadWearLogIDs.
	// failAfter=1 means the first QueryContext (queryOutfitLogs) succeeds, the second fails.
	mock := &mockQueryFailAfterDB{inner: db, failAfter: 1}
	repo := &OutfitLogRepository{db: mock}
	_, err = repo.ListByOutfit(t.Context(), "ol-wb-lb-outfit")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── ListByOwnerDateRange: batchLoadWearLogIDs error path ─────────────────────

func TestListByOwnerDateRangeShouldReturnErrIOWhenBatchLoadWearLogIDsFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('ol-wb-dr-owner', 'ol-wb-dr@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at) VALUES ('ol-wb-dr-outfit', 'ol-wb-dr-owner', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at)
		VALUES ('ol-wb-dr-1', 'ol-wb-dr-outfit', 'ol-wb-dr-owner', '2025-06-01', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	// Same pattern as ListByOutfit: first QueryContext succeeds, second fails.
	mock := &mockQueryFailAfterDB{inner: db, failAfter: 1}
	repo := &OutfitLogRepository{db: mock}
	_, err = repo.ListByOwnerDateRange(t.Context(), "ol-wb-dr-owner",
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── batchLoadWearLogIDs: QueryContext error ───────────────────────────────────

func TestBatchLoadWearLogIDsShouldReturnErrIOWhenQueryFails(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.Close()

	repo := &OutfitLogRepository{db: db}
	var l domain.OutfitLog
	l.ID = "ol-1"
	err = repo.batchLoadWearLogIDs(t.Context(), []domain.OutfitLog{l})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── batchLoadWearLogIDs: rows.Err error ───────────────────────────────────────

func TestBatchLoadWearLogIDsShouldReturnErrIOWhenRowsErrFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &OutfitLogRepository{db: db}
	var l domain.OutfitLog
	l.ID = "ol-1"
	err := repo.batchLoadWearLogIDs(t.Context(), []domain.OutfitLog{l})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── batchLoadWearLogIDs: scan error ───────────────────────────────────────────

func TestBatchLoadWearLogIDsShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &OutfitLogRepository{db: db}
	var l domain.OutfitLog
	l.ID = "ol-1"
	err := repo.batchLoadWearLogIDs(t.Context(), []domain.OutfitLog{l})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── batchLoadWearLogIDs: context cancelled ────────────────────────────────────

func TestBatchLoadWearLogIDsShouldReturnErrWhenContextCancelled(t *testing.T) {
	db := openTestDB(t)
	repo := &OutfitLogRepository{db: db}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var l domain.OutfitLog
	l.ID = "ol-1"
	err := repo.batchLoadWearLogIDs(ctx, []domain.OutfitLog{l})
	require.ErrorIs(t, err, context.Canceled)
}

// ── LinkedWearLogIDs: rows.Err error ─────────────────────────────────────────

func TestLinkedWearLogIDsShouldReturnErrIOWhenRowsErrFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &OutfitLogRepository{db: db}
	_, err := repo.LinkedWearLogIDs(t.Context(), "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── LinkedWearLogIDs: scan error ─────────────────────────────────────────────

func TestLinkedWearLogIDsShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &OutfitLogRepository{db: db}
	_, err := repo.LinkedWearLogIDs(t.Context(), "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}
