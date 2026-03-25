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

func newOutfitLogTransactor(t *testing.T) (*sqlstore.OutfitLogTransactor, *sqlstore.OutfitLogRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewOutfitLogTransactor(db), sqlstore.NewOutfitLogRepository(db), db
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newOutfitLogTransactorWithDB(t *testing.T, db *sql.DB) *sqlstore.OutfitLogTransactor {
	t.Helper()
	return sqlstore.NewOutfitLogTransactor(db)
}

// ── CreateOutfitLog ───────────────────────────────────────────────────────────

func TestOutfitLogTransactorCreateOutfitLogShouldReturnErrWhenContextCancelled(t *testing.T) {
	tr, _, _ := newOutfitLogTransactor(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := tr.CreateOutfitLog(ctx, domain.OutfitLog{}, nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogTransactorCreateOutfitLogShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	tr := sqlstore.NewOutfitLogTransactor(db)
	db.Close()

	_, err := tr.CreateOutfitLog(t.Context(), domain.OutfitLog{}, nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── DeleteOutfitLog ───────────────────────────────────────────────────────────

func TestOutfitLogTransactorDeleteOutfitLogShouldReturnErrWhenContextCancelled(t *testing.T) {
	tr, _, _ := newOutfitLogTransactor(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := tr.DeleteOutfitLog(ctx, "ol-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogTransactorDeleteOutfitLogShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	tr := sqlstore.NewOutfitLogTransactor(db)
	db.Close()

	err := tr.DeleteOutfitLog(t.Context(), "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogTransactorDeleteOutfitLogShouldRemoveOutfitLogAndLinkedWearLogs(t *testing.T) {
	tr, repo, db := newOutfitLogTransactor(t)
	seedUserForOutfit(t, db, "user-del-tx")
	seedItemForOutfit(t, db, "item-del-tx", "user-del-tx")
	seedOutfitForLog(t, db, "outfit-del-tx", "user-del-tx")
	seedWearLogForOutfitLog(t, db, "wl-del-tx-1", "item-del-tx", "user-del-tx")
	seedOutfitLog(t, db, "ol-del-tx", "outfit-del-tx", "user-del-tx", "2025-06-01")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES ('ol-del-tx', 'wl-del-tx-1')`)
	require.NoError(t, err)

	require.NoError(t, tr.DeleteOutfitLog(t.Context(), "ol-del-tx"))

	// outfit log must be gone
	_, err = repo.Get(t.Context(), "ol-del-tx")
	require.ErrorIs(t, err, domain.ErrNotFound)

	// wear log must be gone
	var count int
	row := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM wear_logs WHERE id = 'wl-del-tx-1'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count)
}

// ── UpdateOutfitLogDate ───────────────────────────────────────────────────────

func TestOutfitLogTransactorUpdateOutfitLogDateShouldReturnErrWhenContextCancelled(t *testing.T) {
	tr, _, _ := newOutfitLogTransactor(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := tr.UpdateOutfitLogDate(ctx, "ol-1", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogTransactorUpdateOutfitLogDateShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	tr := sqlstore.NewOutfitLogTransactor(db)
	db.Close()

	err := tr.UpdateOutfitLogDate(t.Context(), "ol-1", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogTransactorUpdateOutfitLogDateShouldUpdateDateOnOutfitLogAndWearLogs(t *testing.T) {
	tr, repo, db := newOutfitLogTransactor(t)
	seedUserForOutfit(t, db, "user-upd-tx")
	seedItemForOutfit(t, db, "item-upd-tx", "user-upd-tx")
	seedOutfitForLog(t, db, "outfit-upd-tx", "user-upd-tx")
	seedWearLogForOutfitLog(t, db, "wl-upd-tx-1", "item-upd-tx", "user-upd-tx")
	seedOutfitLog(t, db, "ol-upd-tx", "outfit-upd-tx", "user-upd-tx", "2025-06-01")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES ('ol-upd-tx', 'wl-upd-tx-1')`)
	require.NoError(t, err)

	newDate := time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, tr.UpdateOutfitLogDate(t.Context(), "ol-upd-tx", newDate))

	// Outfit log date must be updated
	fetched, err := repo.Get(t.Context(), "ol-upd-tx")
	require.NoError(t, err)
	require.Equal(t, "2025-07-15", fetched.WornOn.Format("2006-01-02"))

	// Wear log date must also be updated
	var wornOn string
	row := db.QueryRowContext(t.Context(), `SELECT worn_on FROM wear_logs WHERE id = 'wl-upd-tx-1'`)
	require.NoError(t, row.Scan(&wornOn))
	require.Equal(t, "2025-07-15", wornOn)
}

func TestOutfitLogTransactorCreateOutfitLogShouldPersistOutfitLogAndWearLogsWithLinks(t *testing.T) {
	tr, repo, db := newOutfitLogTransactor(t)
	seedUserForOutfit(t, db, "user-create-tx")
	seedItemForOutfit(t, db, "item-create-tx-1", "user-create-tx")
	seedItemForOutfit(t, db, "item-create-tx-2", "user-create-tx")
	seedOutfitForLog(t, db, "outfit-create-tx", "user-create-tx")

	var ol domain.OutfitLog
	ol.ID = "ol-create-tx"
	ol.OutfitID = "outfit-create-tx"
	ol.OwnerID = "user-create-tx"
	ol.WornOn = time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	ol.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	var wl1, wl2 domain.WearLog
	wl1.ID = "wl-create-tx-1"
	wl1.ItemID = "item-create-tx-1"
	wl1.OwnerID = "user-create-tx"
	wl1.WornOn = time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	wl1.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	wl2.ID = "wl-create-tx-2"
	wl2.ItemID = "item-create-tx-2"
	wl2.OwnerID = "user-create-tx"
	wl2.WornOn = time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	wl2.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	got, err := tr.CreateOutfitLog(t.Context(), ol, []domain.WearLog{wl1, wl2})
	require.NoError(t, err)

	// Returned log must have WearLogIDs populated
	require.Equal(t, "ol-create-tx", got.GetID())
	require.ElementsMatch(t, []string{"wl-create-tx-1", "wl-create-tx-2"}, got.WearLogIDs)

	// Persisted log must be fetchable
	fetched, err := repo.Get(t.Context(), "ol-create-tx")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"wl-create-tx-1", "wl-create-tx-2"}, fetched.WearLogIDs)
}
