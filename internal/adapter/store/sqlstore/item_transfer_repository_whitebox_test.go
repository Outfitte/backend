package sqlstore

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/domain"
)

// ── Delete: RowsAffected error ─────────────────────────────────────────────────

func TestItemTransferRepositoryDeleteShouldReturnErrIOWhenRowsAffectedFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-aff-err")
	repo := &ItemTransferRepository{db: db}
	err := repo.Delete(t.Context(), "tr-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanItemTransferRows: rows.Err, scan and time.Parse errors ────────────────

func TestItemTransferRepositoryListBySenderShouldReturnErrIOWhenRowsErrFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &ItemTransferRepository{db: db}
	_, err := repo.ListBySender(t.Context(), "sender-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositoryListBySenderShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &ItemTransferRepository{db: db}
	_, err := repo.ListBySender(t.Context(), "sender-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositoryListBySenderShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	seedTransferUsers(t, db, "sender-bad-ts", "recip-bad-ts")
	seedTransferItem(t, db, "item-bad-ts", "sender-bad-ts")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO item_transfers (id, item_id, sender_id, recipient_id, status, transfer_history, created_at)
		VALUES ('tr-bad-ts', 'item-bad-ts', 'sender-bad-ts', 'recip-bad-ts', 'pending', 0, 'not-a-date')`)
	require.NoError(t, err)

	repo := &ItemTransferRepository{db: db}
	_, err = repo.ListBySender(t.Context(), "sender-bad-ts", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositoryListBySenderShouldReturnErrIOWhenDecidedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	seedTransferUsers(t, db, "sender-bad-da", "recip-bad-da")
	seedTransferItem(t, db, "item-bad-da", "sender-bad-da")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO item_transfers (id, item_id, sender_id, recipient_id, status, transfer_history, created_at, decided_at)
		VALUES ('tr-bad-da', 'item-bad-da', 'sender-bad-da', 'recip-bad-da', 'accepted', 0, '2025-06-01T00:00:00Z', 'not-a-date')`)
	require.NoError(t, err)

	repo := &ItemTransferRepository{db: db}
	_, err = repo.ListBySender(t.Context(), "sender-bad-da", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanItemTransferRow: scan and time.Parse errors ───────────────────────────

func TestScanItemTransferRowShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)
	// SELECT 1 returns a single integer column; scanning into 8 fields will fail.
	row := db.QueryRowContext(t.Context(), "SELECT 1")
	_, err := scanItemTransferRow(row)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestScanItemTransferRowShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	seedTransferUsers(t, db, "sender-bad-get-ts", "recip-bad-get-ts")
	seedTransferItem(t, db, "item-bad-get-ts", "sender-bad-get-ts")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO item_transfers (id, item_id, sender_id, recipient_id, status, transfer_history, created_at)
		VALUES ('tr-bad-get-ts', 'item-bad-get-ts', 'sender-bad-get-ts', 'recip-bad-get-ts', 'pending', 0, 'not-a-date')`)
	require.NoError(t, err)

	repo := &ItemTransferRepository{db: db}
	_, err = repo.Get(t.Context(), "tr-bad-get-ts")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestScanItemTransferRowShouldReturnErrIOWhenDecidedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	seedTransferUsers(t, db, "sender-bad-get-da", "recip-bad-get-da")
	seedTransferItem(t, db, "item-bad-get-da", "sender-bad-get-da")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO item_transfers (id, item_id, sender_id, recipient_id, status, transfer_history, created_at, decided_at)
		VALUES ('tr-bad-get-da', 'item-bad-get-da', 'sender-bad-get-da', 'recip-bad-get-da', 'accepted', 0, '2025-06-01T00:00:00Z', 'not-a-date')`)
	require.NoError(t, err)

	repo := &ItemTransferRepository{db: db}
	_, err = repo.Get(t.Context(), "tr-bad-get-da")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func seedTransferUsers(t *testing.T, db *sql.DB, senderID, recipientID string) {
	t.Helper()
	for _, id := range []string{senderID, recipientID} {
		_, err := db.ExecContext(t.Context(), `
			INSERT INTO users (id, email, password_hash, role, created_at)
			VALUES (?, ?, 'hash', 'member', '2025-01-01T00:00:00Z')`,
			id, id+"@example.com")
		require.NoError(t, err)
	}
}

func seedTransferItem(t *testing.T, db *sql.DB, itemID, ownerID string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES (?, ?, 'Item', '2025-01-01T00:00:00Z', '{}')`,
		itemID, ownerID)
	require.NoError(t, err)
}
