package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/adapter/store/sqlstore"
	"github.com/outfitte/backend/internal/domain"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newOutfitLogRepo(t *testing.T) (*sqlstore.OutfitLogRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewOutfitLogRepository(db), db
}

func seedOutfitForLog(t *testing.T, db *sql.DB, outfitID, ownerID string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at) VALUES (?, ?, '2025-01-01T00:00:00Z')`,
		outfitID, ownerID)
	require.NoError(t, err)
}

func seedWearLogForOutfitLog(t *testing.T, db *sql.DB, wearLogID, itemID, ownerID string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES (?, ?, ?, '2025-06-01', '2025-01-01T00:00:00Z')`,
		wearLogID, itemID, ownerID)
	require.NoError(t, err)
}

func seedOutfitLog(t *testing.T, db *sql.DB, id, outfitID, ownerID, wornOn string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at)
		VALUES (?, ?, ?, ?, '2025-01-01T00:00:00Z')`,
		id, outfitID, ownerID, wornOn)
	require.NoError(t, err)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestOutfitLogRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "log-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogRepositoryGetShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)

	_, err := repo.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogRepositoryGetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitLogRepository(db)
	db.Close()

	_, err := repo.Get(t.Context(), "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogRepositoryGetShouldReturnLogWithWearLogIDsWhenRowExists(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-get-log")
	seedItemForOutfit(t, db, "item-get-log", "user-get-log")
	seedOutfitForLog(t, db, "outfit-get-log", "user-get-log")
	seedWearLogForOutfitLog(t, db, "wl-get-log-1", "item-get-log", "user-get-log")
	seedOutfitLog(t, db, "ol-get-1", "outfit-get-log", "user-get-log", "2025-06-15")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES ('ol-get-1', 'wl-get-log-1')`)
	require.NoError(t, err)

	got, err := repo.Get(t.Context(), "ol-get-1")
	require.NoError(t, err)
	require.Equal(t, "ol-get-1", got.GetID())
	require.Equal(t, "outfit-get-log", got.OutfitID)
	require.Equal(t, "user-get-log", got.OwnerID)
	require.Equal(t, "2025-06-15", got.WornOn.Format("2006-01-02"))
	require.Nil(t, got.Notes)
	require.Equal(t, []string{"wl-get-log-1"}, got.WearLogIDs)
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestOutfitLogRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var l domain.OutfitLog
	l.ID = "ol-save-1"
	err := repo.Save(ctx, l)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitLogRepository(db)
	db.Close()

	var l domain.OutfitLog
	l.ID = "ol-save-1"
	err := repo.Save(t.Context(), l)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogRepositorySaveShouldPersistNewLog(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-save-log")
	seedOutfitForLog(t, db, "outfit-save-log", "user-save-log")

	notes := "nice outfit"
	var l domain.OutfitLog
	l.ID = "ol-save-new"
	l.OutfitID = "outfit-save-log"
	l.OwnerID = "user-save-log"
	l.WornOn = time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	l.Notes = &notes
	l.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Save(t.Context(), l))

	got, err := repo.Get(t.Context(), "ol-save-new")
	require.NoError(t, err)
	require.Equal(t, "ol-save-new", got.GetID())
	require.Equal(t, "outfit-save-log", got.OutfitID)
	require.Equal(t, "user-save-log", got.OwnerID)
	require.Equal(t, "2025-06-15", got.WornOn.Format("2006-01-02"))
	require.NotNil(t, got.Notes)
	require.Equal(t, "nice outfit", *got.Notes)
	require.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), got.CreatedAt)
}

func TestOutfitLogRepositorySaveShouldUpdateExistingLog(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-save-upd")
	seedOutfitForLog(t, db, "outfit-save-upd", "user-save-upd")
	seedOutfitLog(t, db, "ol-save-upd", "outfit-save-upd", "user-save-upd", "2025-06-15")

	notes := "updated notes"
	var l domain.OutfitLog
	l.ID = "ol-save-upd"
	l.OutfitID = "outfit-save-upd"
	l.OwnerID = "user-save-upd"
	l.WornOn = time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	l.Notes = &notes
	l.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Save(t.Context(), l))

	got, err := repo.Get(t.Context(), "ol-save-upd")
	require.NoError(t, err)
	require.Equal(t, "2025-07-01", got.WornOn.Format("2006-01-02"))
	require.NotNil(t, got.Notes)
	require.Equal(t, "updated notes", *got.Notes)
	// created_at must not change
	require.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), got.CreatedAt)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestOutfitLogRepositoryDeleteShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.Delete(ctx, "ol-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogRepositoryDeleteShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)

	err := repo.Delete(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogRepositoryDeleteShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitLogRepository(db)
	db.Close()

	err := repo.Delete(t.Context(), "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogRepositoryDeleteShouldRemoveLogWhenExists(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-del-log")
	seedOutfitForLog(t, db, "outfit-del-log", "user-del-log")
	seedOutfitLog(t, db, "ol-del-1", "outfit-del-log", "user-del-log", "2025-06-01")

	require.NoError(t, repo.Delete(t.Context(), "ol-del-1"))

	_, err := repo.Get(t.Context(), "ol-del-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogRepositoryDeleteShouldCascadeToWearLogLinks(t *testing.T) {
	_, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-del-cascade")
	seedItemForOutfit(t, db, "item-del-cascade", "user-del-cascade")
	seedOutfitForLog(t, db, "outfit-del-cascade", "user-del-cascade")
	seedWearLogForOutfitLog(t, db, "wl-del-cascade-1", "item-del-cascade", "user-del-cascade")
	seedOutfitLog(t, db, "ol-del-cascade", "outfit-del-cascade", "user-del-cascade", "2025-06-01")

	_, err := db.ExecContext(t.Context(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	repo := sqlstore.NewOutfitLogRepository(db)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES ('ol-del-cascade', 'wl-del-cascade-1')`)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(t.Context(), "ol-del-cascade"))

	var count int
	row := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM outfit_log_wear_logs WHERE outfit_log_id = 'ol-del-cascade'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count)
}

// ── ListByOutfit ──────────────────────────────────────────────────────────────

func TestOutfitLogRepositoryListByOutfitShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByOutfit(ctx, "outfit-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogRepositoryListByOutfitShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitLogRepository(db)
	db.Close()

	_, err := repo.ListByOutfit(t.Context(), "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogRepositoryListByOutfitShouldReturnEmptyWhenNoLogs(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)

	logs, err := repo.ListByOutfit(t.Context(), "nonexistent-outfit")
	require.NoError(t, err)
	require.Empty(t, logs)
}

func TestOutfitLogRepositoryListByOutfitShouldReturnLogsOrderedByWornOnDesc(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-list-outfit")
	seedOutfitForLog(t, db, "outfit-list-1", "user-list-outfit")
	seedOutfitLog(t, db, "ol-list-b", "outfit-list-1", "user-list-outfit", "2025-06-01")
	seedOutfitLog(t, db, "ol-list-a", "outfit-list-1", "user-list-outfit", "2025-07-01")

	logs, err := repo.ListByOutfit(t.Context(), "outfit-list-1")
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Equal(t, "ol-list-a", logs[0].GetID()) // 2025-07-01 first (desc)
	require.Equal(t, "ol-list-b", logs[1].GetID())
}

func TestOutfitLogRepositoryListByOutfitShouldBatchLoadWearLogIDs(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-list-wear")
	seedItemForOutfit(t, db, "item-list-wear", "user-list-wear")
	seedOutfitForLog(t, db, "outfit-list-wear", "user-list-wear")
	seedWearLogForOutfitLog(t, db, "wl-list-1", "item-list-wear", "user-list-wear")
	seedOutfitLog(t, db, "ol-list-wear-1", "outfit-list-wear", "user-list-wear", "2025-06-01")
	seedOutfitLog(t, db, "ol-list-wear-2", "outfit-list-wear", "user-list-wear", "2025-06-02")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES ('ol-list-wear-1', 'wl-list-1')`)
	require.NoError(t, err)

	logs, err := repo.ListByOutfit(t.Context(), "outfit-list-wear")
	require.NoError(t, err)
	require.Len(t, logs, 2)

	var logWithWear domain.OutfitLog
	for _, l := range logs {
		if l.GetID() == "ol-list-wear-1" {
			logWithWear = l
		}
	}
	require.Equal(t, []string{"wl-list-1"}, logWithWear.WearLogIDs)
}

// ── ListByOwnerDateRange ───────────────────────────────────────────────────────

func TestOutfitLogRepositoryListByOwnerDateRangeShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByOwnerDateRange(ctx, "user-1", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogRepositoryListByOwnerDateRangeShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitLogRepository(db)
	db.Close()

	_, err := repo.ListByOwnerDateRange(t.Context(), "user-1", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogRepositoryListByOwnerDateRangeShouldReturnEmptyWhenNoLogsInRange(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)

	logs, err := repo.ListByOwnerDateRange(t.Context(), "user-1",
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Empty(t, logs)
}

func TestOutfitLogRepositoryListByOwnerDateRangeShouldReturnLogsInRangeOrderedByWornOnAsc(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-range")
	seedOutfitForLog(t, db, "outfit-range", "user-range")
	seedOutfitLog(t, db, "ol-range-out", "outfit-range", "user-range", "2025-01-01")
	seedOutfitLog(t, db, "ol-range-b", "outfit-range", "user-range", "2025-06-15")
	seedOutfitLog(t, db, "ol-range-a", "outfit-range", "user-range", "2025-06-01")

	logs, err := repo.ListByOwnerDateRange(t.Context(), "user-range",
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Equal(t, "ol-range-a", logs[0].GetID()) // 2025-06-01 first (asc)
	require.Equal(t, "ol-range-b", logs[1].GetID())
}

func TestOutfitLogRepositoryListByOwnerDateRangeShouldOnlyReturnOwnerLogs(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-range-a")
	seedUserForOutfit(t, db, "user-range-b")
	seedOutfitForLog(t, db, "outfit-range-a", "user-range-a")
	seedOutfitForLog(t, db, "outfit-range-b", "user-range-b")
	seedOutfitLog(t, db, "ol-range-owner-a", "outfit-range-a", "user-range-a", "2025-06-01")
	seedOutfitLog(t, db, "ol-range-owner-b", "outfit-range-b", "user-range-b", "2025-06-01")

	logs, err := repo.ListByOwnerDateRange(t.Context(), "user-range-a",
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, "ol-range-owner-a", logs[0].GetID())
}

func TestOutfitLogRepositoryListByOwnerDateRangeShouldBatchLoadWearLogIDs(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-range-wl")
	seedItemForOutfit(t, db, "item-range-wl", "user-range-wl")
	seedOutfitForLog(t, db, "outfit-range-wl", "user-range-wl")
	seedWearLogForOutfitLog(t, db, "wl-range-1", "item-range-wl", "user-range-wl")
	seedOutfitLog(t, db, "ol-range-wl-1", "outfit-range-wl", "user-range-wl", "2025-06-01")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES ('ol-range-wl-1', 'wl-range-1')`)
	require.NoError(t, err)

	logs, err := repo.ListByOwnerDateRange(t.Context(), "user-range-wl",
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, []string{"wl-range-1"}, logs[0].WearLogIDs)
}

// ── LinkWearLog ───────────────────────────────────────────────────────────────

func TestOutfitLogRepositoryLinkWearLogShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.LinkWearLog(ctx, "ol-1", "wl-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogRepositoryLinkWearLogShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitLogRepository(db)
	db.Close()

	err := repo.LinkWearLog(t.Context(), "ol-1", "wl-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogRepositoryLinkWearLogShouldPersistAssociation(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-link")
	seedItemForOutfit(t, db, "item-link", "user-link")
	seedOutfitForLog(t, db, "outfit-link", "user-link")
	seedWearLogForOutfitLog(t, db, "wl-link-1", "item-link", "user-link")
	seedOutfitLog(t, db, "ol-link-1", "outfit-link", "user-link", "2025-06-01")

	require.NoError(t, repo.LinkWearLog(t.Context(), "ol-link-1", "wl-link-1"))

	ids, err := repo.LinkedWearLogIDs(t.Context(), "ol-link-1")
	require.NoError(t, err)
	require.Equal(t, []string{"wl-link-1"}, ids)
}

// ── LinkedWearLogIDs ──────────────────────────────────────────────────────────

func TestOutfitLogRepositoryLinkedWearLogIDsShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.LinkedWearLogIDs(ctx, "ol-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogRepositoryLinkedWearLogIDsShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitLogRepository(db)
	db.Close()

	_, err := repo.LinkedWearLogIDs(t.Context(), "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogRepositoryLinkedWearLogIDsShouldReturnEmptyWhenNoLinks(t *testing.T) {
	repo, _ := newOutfitLogRepo(t)

	ids, err := repo.LinkedWearLogIDs(t.Context(), "nonexistent-log")
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestOutfitLogRepositoryLinkedWearLogIDsShouldReturnAllLinkedIDs(t *testing.T) {
	repo, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-linked-ids")
	seedItemForOutfit(t, db, "item-linked-ids", "user-linked-ids")
	seedOutfitForLog(t, db, "outfit-linked-ids", "user-linked-ids")
	seedWearLogForOutfitLog(t, db, "wl-linked-1", "item-linked-ids", "user-linked-ids")
	seedWearLogForOutfitLog(t, db, "wl-linked-2", "item-linked-ids", "user-linked-ids")
	seedOutfitLog(t, db, "ol-linked-1", "outfit-linked-ids", "user-linked-ids", "2025-06-01")

	require.NoError(t, repo.LinkWearLog(t.Context(), "ol-linked-1", "wl-linked-1"))
	require.NoError(t, repo.LinkWearLog(t.Context(), "ol-linked-1", "wl-linked-2"))

	ids, err := repo.LinkedWearLogIDs(t.Context(), "ol-linked-1")
	require.NoError(t, err)
	require.Len(t, ids, 2)
	require.Contains(t, ids, "wl-linked-1")
	require.Contains(t, ids, "wl-linked-2")
}

func TestOutfitLogRepositoryGetShouldReturnNotesWhenSet(t *testing.T) {
	_, db := newOutfitLogRepo(t)
	seedUserForOutfit(t, db, "user-get-notes")
	seedOutfitForLog(t, db, "outfit-get-notes", "user-get-notes")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, notes, created_at)
		VALUES ('ol-notes-1', 'outfit-get-notes', 'user-get-notes', '2025-06-15', 'great day', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	repo := sqlstore.NewOutfitLogRepository(db)
	got, err := repo.Get(t.Context(), "ol-notes-1")
	require.NoError(t, err)
	require.NotNil(t, got.Notes)
	require.Equal(t, "great day", *got.Notes)
}
