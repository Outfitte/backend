package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/adapter/store/sqlstore"
	"github.com/outfitte/outfitte/internal/domain"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newWearLogRepo(t *testing.T) (*sqlstore.WearLogRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewWearLogRepository(db), db
}

func seedUserForWearLog(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (?, ?, 'hash', 'member', '2025-01-01T00:00:00Z')`,
		id, id+"@example.com")
	require.NoError(t, err)
}

func seedItemForWearLog(t *testing.T, db *sql.DB, itemID, ownerID string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES (?, ?, 'Test Item', '2025-01-01T00:00:00Z', '{}')`,
		itemID, ownerID)
	require.NoError(t, err)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestWearLogRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newWearLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "log-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogRepositoryGetShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newWearLogRepo(t)

	_, err := repo.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestWearLogRepositoryGetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewWearLogRepository(db)
	db.Close()

	_, err := repo.Get(t.Context(), "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogRepositoryGetShouldReturnWearLogWhenRowExists(t *testing.T) {
	repo, db := newWearLogRepo(t)
	seedUserForWearLog(t, db, "user-get")
	seedItemForWearLog(t, db, "item-get", "user-get")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, notes, created_at)
		VALUES ('log-get-1', 'item-get', 'user-get', '2025-06-01', 'Nice day', '2025-06-01T10:00:00Z')`)
	require.NoError(t, err)

	log, err := repo.Get(t.Context(), "log-get-1")
	require.NoError(t, err)
	require.Equal(t, "log-get-1", log.GetID())
	require.Equal(t, "item-get", log.ItemID)
	require.Equal(t, "user-get", log.OwnerID)
	require.Equal(t, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), log.WornOn)
	require.NotNil(t, log.Notes)
	require.Equal(t, "Nice day", *log.Notes)
	require.Equal(t, time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC), log.CreatedAt)
}

func TestWearLogRepositoryGetShouldReturnNilNotesWhenNotSet(t *testing.T) {
	repo, db := newWearLogRepo(t)
	seedUserForWearLog(t, db, "user-nil-notes")
	seedItemForWearLog(t, db, "item-nil-notes", "user-nil-notes")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, notes, created_at)
		VALUES ('log-nil-notes', 'item-nil-notes', 'user-nil-notes', '2025-06-01', NULL, '2025-06-01T10:00:00Z')`)
	require.NoError(t, err)

	log, err := repo.Get(t.Context(), "log-nil-notes")
	require.NoError(t, err)
	require.Nil(t, log.Notes)
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestWearLogRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newWearLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var log domain.WearLog
	log.ID = "log-1"
	err := repo.Save(ctx, log)
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewWearLogRepository(db)
	db.Close()

	var log domain.WearLog
	log.ID = "log-1"
	err := repo.Save(t.Context(), log)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogRepositorySaveShouldPersistNewWearLog(t *testing.T) {
	repo, db := newWearLogRepo(t)
	seedUserForWearLog(t, db, "user-save")
	seedItemForWearLog(t, db, "item-save", "user-save")

	note := "First wear"
	var log domain.WearLog
	log.ID = "log-save-1"
	log.ItemID = "item-save"
	log.OwnerID = "user-save"
	log.WornOn = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	log.Notes = &note
	log.CreatedAt = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Save(t.Context(), log))

	got, err := repo.Get(t.Context(), "log-save-1")
	require.NoError(t, err)
	require.Equal(t, "log-save-1", got.GetID())
	require.Equal(t, "First wear", *got.Notes)
}

func TestWearLogRepositorySaveShouldUpdateExistingWearLog(t *testing.T) {
	repo, db := newWearLogRepo(t)
	seedUserForWearLog(t, db, "user-upd")
	seedItemForWearLog(t, db, "item-upd", "user-upd")

	var log domain.WearLog
	log.ID = "log-upd-1"
	log.ItemID = "item-upd"
	log.OwnerID = "user-upd"
	log.WornOn = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	log.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), log))

	newNote := "Updated note"
	log.WornOn = time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	log.Notes = &newNote
	log.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), log))

	got, err := repo.Get(t.Context(), "log-upd-1")
	require.NoError(t, err)
	require.Equal(t, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), got.WornOn)
	require.Equal(t, "Updated note", *got.Notes)
	require.Equal(t, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), got.CreatedAt) // must not change
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestWearLogRepositoryDeleteShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newWearLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.Delete(ctx, "log-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogRepositoryDeleteShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newWearLogRepo(t)

	err := repo.Delete(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestWearLogRepositoryDeleteShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewWearLogRepository(db)
	db.Close()

	err := repo.Delete(t.Context(), "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogRepositoryDeleteShouldRemoveWearLogWhenExists(t *testing.T) {
	repo, db := newWearLogRepo(t)
	seedUserForWearLog(t, db, "user-del")
	seedItemForWearLog(t, db, "item-del", "user-del")

	var log domain.WearLog
	log.ID = "log-del-1"
	log.ItemID = "item-del"
	log.OwnerID = "user-del"
	log.WornOn = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	log.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), log))

	require.NoError(t, repo.Delete(t.Context(), "log-del-1"))

	_, err := repo.Get(t.Context(), "log-del-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// ── ListByItem ────────────────────────────────────────────────────────────────

func TestWearLogRepositoryListByItemShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newWearLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByItem(ctx, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogRepositoryListByItemShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewWearLogRepository(db)
	db.Close()

	_, err := repo.ListByItem(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogRepositoryListByItemShouldReturnEmptyWhenItemHasNoLogs(t *testing.T) {
	repo, _ := newWearLogRepo(t)

	logs, err := repo.ListByItem(t.Context(), "nonexistent-item")
	require.NoError(t, err)
	require.Empty(t, logs)
}

func TestWearLogRepositoryListByItemShouldReturnOnlyItemLogs(t *testing.T) {
	repo, db := newWearLogRepo(t)
	seedUserForWearLog(t, db, "user-list-only")
	seedItemForWearLog(t, db, "item-list-a", "user-list-only")
	seedItemForWearLog(t, db, "item-list-b", "user-list-only")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES
			('log-a1', 'item-list-a', 'user-list-only', '2025-01-01', '2025-01-01T00:00:00Z'),
			('log-b1', 'item-list-b', 'user-list-only', '2025-01-02', '2025-01-02T00:00:00Z')`)
	require.NoError(t, err)

	logs, err := repo.ListByItem(t.Context(), "item-list-a")
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, "log-a1", logs[0].GetID())
}

func TestWearLogRepositoryListByItemShouldReturnLogsOrderedByWornOnDesc(t *testing.T) {
	repo, db := newWearLogRepo(t)
	seedUserForWearLog(t, db, "user-order")
	seedItemForWearLog(t, db, "item-order", "user-order")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES
			('log-order-1', 'item-order', 'user-order', '2025-01-01', '2025-01-01T00:00:00Z'),
			('log-order-2', 'item-order', 'user-order', '2025-03-01', '2025-03-01T00:00:00Z'),
			('log-order-3', 'item-order', 'user-order', '2025-02-01', '2025-02-01T00:00:00Z')`)
	require.NoError(t, err)

	logs, err := repo.ListByItem(t.Context(), "item-order")
	require.NoError(t, err)
	require.Len(t, logs, 3)
	require.Equal(t, "log-order-2", logs[0].GetID()) // 2025-03-01 is latest
	require.Equal(t, "log-order-3", logs[1].GetID()) // 2025-02-01
	require.Equal(t, "log-order-1", logs[2].GetID()) // 2025-01-01
}

// ── LatestByItem ──────────────────────────────────────────────────────────────

func TestWearLogRepositoryLatestByItemShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newWearLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.LatestByItem(ctx, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogRepositoryLatestByItemShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewWearLogRepository(db)
	db.Close()

	_, err := repo.LatestByItem(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogRepositoryLatestByItemShouldReturnNilWhenNoLogs(t *testing.T) {
	repo, _ := newWearLogRepo(t)

	got, err := repo.LatestByItem(t.Context(), "nonexistent-item")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestWearLogRepositoryLatestByItemShouldReturnMostRecentLog(t *testing.T) {
	repo, db := newWearLogRepo(t)
	seedUserForWearLog(t, db, "user-latest")
	seedItemForWearLog(t, db, "item-latest", "user-latest")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES
			('log-latest-old', 'item-latest', 'user-latest', '2025-01-01', '2025-01-01T00:00:00Z'),
			('log-latest-new', 'item-latest', 'user-latest', '2025-06-01', '2025-06-01T00:00:00Z')`)
	require.NoError(t, err)

	got, err := repo.LatestByItem(t.Context(), "item-latest")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "log-latest-new", got.GetID())
}

// ── CountByItem ───────────────────────────────────────────────────────────────

func TestWearLogRepositoryCountByItemShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newWearLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.CountByItem(ctx, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogRepositoryCountByItemShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewWearLogRepository(db)
	db.Close()

	_, err := repo.CountByItem(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogRepositoryCountByItemShouldReturnZeroWhenNoLogs(t *testing.T) {
	repo, _ := newWearLogRepo(t)

	count, err := repo.CountByItem(t.Context(), "nonexistent-item")
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestWearLogRepositoryCountByItemShouldReturnCorrectCount(t *testing.T) {
	repo, db := newWearLogRepo(t)
	seedUserForWearLog(t, db, "user-count")
	seedItemForWearLog(t, db, "item-count", "user-count")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES
			('log-count-1', 'item-count', 'user-count', '2025-01-01', '2025-01-01T00:00:00Z'),
			('log-count-2', 'item-count', 'user-count', '2025-02-01', '2025-02-01T00:00:00Z'),
			('log-count-3', 'item-count', 'user-count', '2025-03-01', '2025-03-01T00:00:00Z')`)
	require.NoError(t, err)

	count, err := repo.CountByItem(t.Context(), "item-count")
	require.NoError(t, err)
	require.Equal(t, 3, count)
}
