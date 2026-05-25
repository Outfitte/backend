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

func newItemTransferTransactor(t *testing.T) (*sqlstore.ItemTransferTransactor, *sqlstore.ItemTransferRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewItemTransferTransactor(db), sqlstore.NewItemTransferRepository(db), db
}

// ── Accept ────────────────────────────────────────────────────────────────────

func TestItemTransferTransactorAcceptShouldReturnErrWhenContextCancelled(t *testing.T) {
	tr, _, _ := newItemTransferTransactor(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := tr.Accept(ctx, "tr-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferTransactorAcceptShouldReturnErrNotFoundWhenTransferMissing(t *testing.T) {
	tr, _, _ := newItemTransferTransactor(t)

	_, err := tr.Accept(t.Context(), "nonexistent-tr")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferTransactorAcceptShouldReturnErrValidationWhenTransferNotPending(t *testing.T) {
	tr, repo, db := newItemTransferTransactor(t)
	seedUserForTransfer(t, db, "sender-vali")
	seedUserForTransfer(t, db, "recip-vali")
	seedItemForTransfer(t, db, "item-vali", "sender-vali")

	decided := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	accepted := buildTransfer("tr-vali-1", "item-vali", "sender-vali", "recip-vali", domain.TransferStatusAccepted, false)
	accepted.DecidedAt = &decided
	require.NoError(t, repo.Save(t.Context(), accepted))

	_, err := tr.Accept(t.Context(), "tr-vali-1")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemTransferTransactorAcceptShouldReturnErrNotFoundWhenItemMissing(t *testing.T) {
	tr, repo, db := newItemTransferTransactor(t)
	seedUserForTransfer(t, db, "sender-imiss")
	seedUserForTransfer(t, db, "recip-imiss")
	// No item is seeded; FK is not enforced in openMigratedDB so the insert succeeds.
	transfer := buildTransfer("tr-imiss-1", "nonexistent-item", "sender-imiss", "recip-imiss", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), transfer))

	_, err := tr.Accept(t.Context(), "tr-imiss-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferTransactorAcceptShouldReturnErrForbiddenWhenItemNotOwnedBySender(t *testing.T) {
	tr, repo, db := newItemTransferTransactor(t)
	seedUserForTransfer(t, db, "sender-forb")
	seedUserForTransfer(t, db, "recip-forb")
	seedUserForTransfer(t, db, "other-forb")
	// item is owned by "other-forb", not "sender-forb"
	seedItemForTransfer(t, db, "item-forb", "other-forb")

	transfer := buildTransfer("tr-forb-1", "item-forb", "sender-forb", "recip-forb", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), transfer))

	_, err := tr.Accept(t.Context(), "tr-forb-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemTransferTransactorAcceptShouldReturnErrValidationWhenItemArchived(t *testing.T) {
	tr, repo, db := newItemTransferTransactor(t)
	seedUserForTransfer(t, db, "sender-arch")
	seedUserForTransfer(t, db, "recip-arch")
	// Seed an archived item owned by sender
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata, archived_at)
		VALUES ('item-arch', 'sender-arch', 'Item', '2025-01-01T00:00:00Z', '{}', '2025-06-01T00:00:00Z')`)
	require.NoError(t, err)

	transfer := buildTransfer("tr-arch-1", "item-arch", "sender-arch", "recip-arch", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), transfer))

	_, err = tr.Accept(t.Context(), "tr-arch-1")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemTransferTransactorAcceptShouldTransferOwnershipWhenTransferHistoryTrue(t *testing.T) {
	tr, repo, db := newItemTransferTransactor(t)
	db.SetMaxOpenConns(1)
	_, err := db.ExecContext(t.Context(), `PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	seedUserForTransfer(t, db, "sender-hap")
	seedUserForTransfer(t, db, "recip-hap")
	seedItemForTransfer(t, db, "item-hap", "sender-hap")

	// seed an outfit that contains the item
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, name, created_at) VALUES ('outfit-hap', 'sender-hap', 'O', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `INSERT INTO outfit_items (outfit_id, item_id) VALUES ('outfit-hap', 'item-hap')`)
	require.NoError(t, err)

	// seed a share for the item
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO shares (id, owner_id, recipient_id, target_type, target_id, created_at)
		VALUES ('share-hap', 'sender-hap', 'recip-hap', 'item', 'item-hap', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	// seed a wear log for the item
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES ('wl-hap', 'item-hap', 'sender-hap', '2025-06-01', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	transferCreatedAt := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	transfer := buildTransfer("tr-hap-1", "item-hap", "sender-hap", "recip-hap", domain.TransferStatusPending, true)
	require.NoError(t, repo.Save(t.Context(), transfer))

	before := time.Now()
	got, err := tr.Accept(t.Context(), "tr-hap-1")
	after := time.Now()
	require.NoError(t, err)

	// Transfer must be accepted with decided_at set and original CreatedAt preserved
	require.Equal(t, domain.TransferStatusAccepted, got.Status)
	require.NotNil(t, got.DecidedAt)
	require.True(t, !got.DecidedAt.Before(before) && !got.DecidedAt.After(after))
	require.Equal(t, transferCreatedAt, got.CreatedAt)

	// Item owner must be updated and location_id cleared
	var ownerID string
	var locationID sql.NullString
	row := db.QueryRowContext(t.Context(), `SELECT owner_id, location_id FROM items WHERE id = 'item-hap'`)
	require.NoError(t, row.Scan(&ownerID, &locationID))
	require.Equal(t, "recip-hap", ownerID)
	require.False(t, locationID.Valid)

	// Wear log owner must be reassigned to recipient
	var wlOwner string
	row = db.QueryRowContext(t.Context(), `SELECT owner_id FROM wear_logs WHERE id = 'wl-hap'`)
	require.NoError(t, row.Scan(&wlOwner))
	require.Equal(t, "recip-hap", wlOwner)

	// outfit_items row must be deleted
	var count int
	row = db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM outfit_items WHERE item_id = 'item-hap'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count)

	// share must be deleted
	row = db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM shares WHERE target_id = 'item-hap'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count)

	// Persisted transfer must also reflect accepted status
	persisted, err := repo.Get(t.Context(), "tr-hap-1")
	require.NoError(t, err)
	require.Equal(t, domain.TransferStatusAccepted, persisted.Status)
	require.NotNil(t, persisted.DecidedAt)
}

func TestItemTransferTransactorAcceptShouldTransferOwnershipWhenTransferHistoryFalse(t *testing.T) {
	tr, repo, db := newItemTransferTransactor(t)
	db.SetMaxOpenConns(1)
	_, err := db.ExecContext(t.Context(), `PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	seedUserForTransfer(t, db, "sender-noh")
	seedUserForTransfer(t, db, "recip-noh")
	seedItemForTransfer(t, db, "item-noh", "sender-noh")

	// seed outfit containing item
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, name, created_at) VALUES ('outfit-noh', 'sender-noh', 'O', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `INSERT INTO outfit_items (outfit_id, item_id) VALUES ('outfit-noh', 'item-noh')`)
	require.NoError(t, err)

	// seed outfit_log and wear_log linked together
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at)
		VALUES ('ol-noh', 'outfit-noh', 'sender-noh', '2025-06-01', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES ('wl-noh', 'item-noh', 'sender-noh', '2025-06-01', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES ('ol-noh', 'wl-noh')`)
	require.NoError(t, err)

	transfer := buildTransfer("tr-noh-1", "item-noh", "sender-noh", "recip-noh", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), transfer))

	got, err := tr.Accept(t.Context(), "tr-noh-1")
	require.NoError(t, err)
	require.Equal(t, domain.TransferStatusAccepted, got.Status)
	require.NotNil(t, got.DecidedAt)

	// Wear log must be deleted
	var count int
	row := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM wear_logs WHERE id = 'wl-noh'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count)

	// outfit_log_wear_logs row must be cascade-deleted
	row = db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM outfit_log_wear_logs WHERE wear_log_id = 'wl-noh'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count)

	// outfit_items row must be deleted
	row = db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM outfit_items WHERE item_id = 'item-noh'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count)

	// outfit_log must remain
	row = db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM outfit_logs WHERE id = 'ol-noh'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 1, count)
}
