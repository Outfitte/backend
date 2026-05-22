package sqlstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/outfitte/backend/internal/domain"
)

// ── Accept: BeginTx failure ───────────────────────────────────────────────────

func TestItemTransferTransactorAcceptShouldReturnErrIOWhenBeginTxFails(t *testing.T) {
	db := openTestDB(t)
	tr := &ItemTransferTransactor{db: db}
	db.Close()

	_, err := tr.Accept(t.Context(), "tr-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── Accept: helper error propagation ─────────────────────────────────────────

func TestItemTransferTransactorAcceptShouldReturnErrIOWhenApplyWearLogTransferFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-accept-exec-fail-at-1")
	tr := &ItemTransferTransactor{db: db}

	_, err := tr.Accept(t.Context(), "tr-fake")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferTransactorAcceptShouldReturnErrIOWhenDeleteOutfitItemsFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-accept-exec-fail-at-2")
	tr := &ItemTransferTransactor{db: db}

	_, err := tr.Accept(t.Context(), "tr-fake")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferTransactorAcceptShouldReturnErrIOWhenDeleteSharesFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-accept-exec-fail-at-3")
	tr := &ItemTransferTransactor{db: db}

	_, err := tr.Accept(t.Context(), "tr-fake")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferTransactorAcceptShouldReturnErrIOWhenUpdateItemOwnerFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-accept-exec-fail-at-4")
	tr := &ItemTransferTransactor{db: db}

	_, err := tr.Accept(t.Context(), "tr-fake")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferTransactorAcceptShouldReturnErrIOWhenAcceptTransferFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-accept-exec-fail-at-5")
	tr := &ItemTransferTransactor{db: db}

	_, err := tr.Accept(t.Context(), "tr-fake")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferTransactorAcceptShouldReturnErrIOWhenCommitFails(t *testing.T) {
	db := openFakeDB(t, "fake-tx-accept-commit-fail")
	tr := &ItemTransferTransactor{db: db}

	_, err := tr.Accept(t.Context(), "tr-fake")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── readAndValidateTransfer: scan and parse errors ───────────────────────────

func TestReadAndValidateTransferShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	_, err = readAndValidateTransfer(t.Context(), tx, "tr-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestReadAndValidateTransferShouldReturnErrIOWhenCreatedAtIsInvalidFormat(t *testing.T) {
	db := openTestDB(t)
	// Insert a transfer row with an unparseable created_at.
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('u-parse', 'u-parse@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('i-parse', 'u-parse', 'Item', '2025-01-01T00:00:00Z', '{}')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO item_transfers (id, item_id, sender_id, recipient_id, status, transfer_history, created_at)
		VALUES ('tr-parse', 'i-parse', 'u-parse', 'u-parse', 'pending', 0, 'not-a-timestamp')`)
	require.NoError(t, err)

	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	_, err = readAndValidateTransfer(t.Context(), tx, "tr-parse")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── validateItemForTransfer: scan errors ─────────────────────────────────────

func TestValidateItemForTransferShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	err = validateItemForTransfer(t.Context(), tx, "nonexistent-item", "sender-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestValidateItemForTransferShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = validateItemForTransfer(t.Context(), tx, "item-1", "sender-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── applyWearLogTransfer: exec errors ─────────────────────────────────────────

func TestApplyWearLogTransferShouldReturnErrIOWhenUpdateExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = applyWearLogTransfer(t.Context(), tx, "item-1", "recip-1", true)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestApplyWearLogTransferShouldReturnErrIOWhenDeleteExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = applyWearLogTransfer(t.Context(), tx, "item-1", "recip-1", false)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── deleteOutfitItemsForItem: exec error ──────────────────────────────────────

func TestDeleteOutfitItemsForItemShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = deleteOutfitItemsForItem(t.Context(), tx, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── deleteSharesForItem: exec error ───────────────────────────────────────────

func TestDeleteSharesForItemShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = deleteSharesForItem(t.Context(), tx, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── updateItemOwner: exec error ───────────────────────────────────────────────

func TestUpdateItemOwnerShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = updateItemOwner(t.Context(), tx, "item-1", "recip-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── acceptTransfer: exec error ────────────────────────────────────────────────

func TestAcceptTransferShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	err = acceptTransfer(t.Context(), tx, "tr-1", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, domain.ErrIO)
}
