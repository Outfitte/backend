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

func newItemTransferRepo(t *testing.T) (*sqlstore.ItemTransferRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewItemTransferRepository(db), db
}

func seedUserForTransfer(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (?, ?, 'hash', 'member', '2025-01-01T00:00:00Z')`,
		id, id+"@example.com")
	require.NoError(t, err)
}

func seedItemForTransfer(t *testing.T, db *sql.DB, itemID, ownerID string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES (?, ?, 'Item', '2025-01-01T00:00:00Z', '{}')`,
		itemID, ownerID)
	require.NoError(t, err)
}

func buildTransfer(id, itemID, senderID, recipientID string, status domain.TransferStatus, history bool) domain.ItemTransfer {
	var tr domain.ItemTransfer
	tr.ID = id
	tr.ItemID = itemID
	tr.SenderID = senderID
	tr.RecipientID = recipientID
	tr.Status = status
	tr.TransferHistory = history
	tr.CreatedAt = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	return tr
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestItemTransferRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemTransferRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "tr-1")
	require.ErrorIs(t, err, context.Canceled)
}
