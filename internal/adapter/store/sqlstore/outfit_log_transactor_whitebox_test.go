package sqlstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/outfitte/outfitte/internal/domain"
)

// ── CreateOutfitLog: insertOutfitLog error ────────────────────────────────────

func TestOutfitLogTransactorCreateOutfitLogShouldReturnErrIOWhenInsertOutfitLogFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-first-exec-fail")
	tr := &OutfitLogTransactor{db: db}

	_, err := tr.CreateOutfitLog(t.Context(), domain.OutfitLog{}, nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── CreateOutfitLog: insertOutfitLogWearLink error ────────────────────────────

func TestOutfitLogTransactorCreateOutfitLogShouldReturnErrIOWhenInsertWearLinkFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-exec-fail-after-2")
	tr := &OutfitLogTransactor{db: db}

	var wl domain.WearLog
	wl.ID = "wl-link-fail"
	_, err := tr.CreateOutfitLog(t.Context(), domain.OutfitLog{}, []domain.WearLog{wl})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── DeleteOutfitLog: deleteWearLogsByIDs error ────────────────────────────────

func TestOutfitLogTransactorDeleteOutfitLogShouldReturnErrIOWhenDeleteWearLogsFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-with-row-first-exec-fail")
	tr := &OutfitLogTransactor{db: db}

	err := tr.DeleteOutfitLog(t.Context(), "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── DeleteOutfitLog: deleteOutfitLogByID error ───────────────────────────────

func TestOutfitLogTransactorDeleteOutfitLogShouldReturnErrIOWhenDeleteOutfitLogFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-first-exec-fail")
	tr := &OutfitLogTransactor{db: db}

	err := tr.DeleteOutfitLog(t.Context(), "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── DeleteOutfitLog: commit error (no linked wear logs) ───────────────────────

func TestOutfitLogTransactorDeleteOutfitLogShouldReturnErrIOWhenCommitFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-exec-ok-commit-fail")
	tr := &OutfitLogTransactor{db: db}

	err := tr.DeleteOutfitLog(t.Context(), "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── UpdateOutfitLogDate: updateOutfitLogDate error ────────────────────────────

func TestOutfitLogTransactorUpdateOutfitLogDateShouldReturnErrIOWhenUpdateOutfitLogFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-first-exec-fail")
	tr := &OutfitLogTransactor{db: db}

	err := tr.UpdateOutfitLogDate(t.Context(), "ol-1", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── UpdateOutfitLogDate: updateWearLogsDates error ────────────────────────────

func TestOutfitLogTransactorUpdateOutfitLogDateShouldReturnErrIOWhenUpdateWearLogsFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-with-row-exec-fail-after-1")
	tr := &OutfitLogTransactor{db: db}

	err := tr.UpdateOutfitLogDate(t.Context(), "ol-1", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── UpdateOutfitLogDate: commit error (no linked wear logs) ───────────────────

func TestOutfitLogTransactorUpdateOutfitLogDateShouldReturnErrIOWhenCommitFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-exec-ok-commit-fail")
	tr := &OutfitLogTransactor{db: db}

	err := tr.UpdateOutfitLogDate(t.Context(), "ol-1", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── insertOutfitLog ───────────────────────────────────────────────────────────

func TestInsertOutfitLogShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	var ol domain.OutfitLog
	ol.ID = "ol-wb-tx"
	err = insertOutfitLog(t.Context(), tx, ol)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── insertWearLog ─────────────────────────────────────────────────────────────

func TestInsertWearLogShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	var wl domain.WearLog
	wl.ID = "wl-wb-tx"
	err = insertWearLog(t.Context(), tx, wl)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── insertOutfitLogWearLink ───────────────────────────────────────────────────

func TestInsertOutfitLogWearLinkShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = insertOutfitLogWearLink(t.Context(), tx, "ol-1", "wl-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── CreateOutfitLog: commit error ─────────────────────────────────────────────

func TestOutfitLogTransactorCreateOutfitLogShouldReturnErrIOWhenCommitFails(t *testing.T) {
	db := openFakeDB(t, "fake-commit-err")
	tr := &OutfitLogTransactor{db: db}

	var ol domain.OutfitLog
	ol.ID = "ol-commit-fail"
	_, err := tr.CreateOutfitLog(t.Context(), ol, nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── CreateOutfitLog: insertWearLog error propagation ─────────────────────────

func TestOutfitLogTransactorCreateOutfitLogShouldReturnErrIOWhenInsertWearLogFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-wl-fail', 'wl-fail@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at) VALUES ('outfit-wl-fail', 'user-wl-fail', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	tr := &OutfitLogTransactor{db: db}

	var ol domain.OutfitLog
	ol.ID = "ol-wl-fail"
	ol.OutfitID = "outfit-wl-fail"
	ol.OwnerID = "user-wl-fail"
	ol.WornOn = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	ol.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Insert a wear log referencing a non-existent item to cause FK violation
	// (FK enforcement must be ON for this to fail).
	db.SetMaxOpenConns(1)
	_, err = db.ExecContext(t.Context(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	var wl domain.WearLog
	wl.ID = "wl-fail-1"
	wl.ItemID = "nonexistent-item"
	wl.OwnerID = "user-wl-fail"
	wl.WornOn = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	wl.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err = tr.CreateOutfitLog(t.Context(), ol, []domain.WearLog{wl})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── selectLinkedWearLogIDs: QueryContext error ────────────────────────────────

func TestSelectLinkedWearLogIDsShouldReturnErrIOWhenQueryFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	_, err = selectLinkedWearLogIDs(t.Context(), tx, "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── selectLinkedWearLogIDs: rows.Err ─────────────────────────────────────────

func TestSelectLinkedWearLogIDsShouldReturnErrIOWhenRowsErrFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-rows-err")
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	_, err = selectLinkedWearLogIDs(t.Context(), tx, "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── selectLinkedWearLogIDs: scan error ────────────────────────────────────────

func TestSelectLinkedWearLogIDsShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-scan-err")
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	_, err = selectLinkedWearLogIDs(t.Context(), tx, "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── deleteWearLogsByIDs: ExecContext error ────────────────────────────────────

func TestDeleteWearLogsByIDsShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = deleteWearLogsByIDs(t.Context(), tx, []string{"wl-1"})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── deleteOutfitLogByID: ExecContext error ────────────────────────────────────

func TestDeleteOutfitLogByIDShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = deleteOutfitLogByID(t.Context(), tx, "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── DeleteOutfitLog: selectLinkedWearLogIDs error ────────────────────────────

func TestOutfitLogTransactorDeleteOutfitLogShouldReturnErrIOWhenSelectWearLogIDsFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-rows-err")
	tr := &OutfitLogTransactor{db: db}

	err := tr.DeleteOutfitLog(t.Context(), "ol-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── updateOutfitLogDate: ExecContext error ────────────────────────────────────

func TestUpdateOutfitLogDateShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = updateOutfitLogDate(t.Context(), tx, "ol-1", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── updateWearLogsDates: ExecContext error ────────────────────────────────────

func TestUpdateWearLogsDatessShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = updateWearLogsDates(t.Context(), tx, []string{"wl-1"}, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── UpdateOutfitLogDate: selectLinkedWearLogIDs error ─────────────────────────

func TestOutfitLogTransactorUpdateOutfitLogDateShouldReturnErrIOWhenSelectWearLogIDsFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-rows-err")
	tr := &OutfitLogTransactor{db: db}

	err := tr.UpdateOutfitLogDate(t.Context(), "ol-1", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}


// ── UpdateOutfitLogDate: updateOutfitLogDate error ───────────────────────────

func TestOutfitLogTransactorUpdateOutfitLogDateShouldReturnErrIOWhenUpdateOutfitLogDateFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-upd-fail', 'upd-fail@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at) VALUES ('outfit-upd-fail', 'user-upd-fail', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at)
		VALUES ('ol-upd-fail', 'outfit-upd-fail', 'user-upd-fail', '2025-06-01', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = updateOutfitLogDate(t.Context(), tx, "ol-upd-fail", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}
