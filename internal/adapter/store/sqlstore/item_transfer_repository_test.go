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

// ── Save ──────────────────────────────────────────────────────────────────────

func TestItemTransferRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemTransferRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	tr := buildTransfer("tr-1", "item-1", "sender-1", "recip-1", domain.TransferStatusPending, false)
	err := repo.Save(ctx, tr)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemTransferRepository(db)
	db.Close()

	tr := buildTransfer("tr-1", "item-1", "sender-1", "recip-1", domain.TransferStatusPending, false)
	err := repo.Save(t.Context(), tr)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositorySaveShouldReturnErrConflictWhenDuplicatePendingForSameItem(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-dup")
	seedUserForTransfer(t, db, "recip-dup")
	seedItemForTransfer(t, db, "item-dup", "sender-dup")

	tr1 := buildTransfer("tr-dup-1", "item-dup", "sender-dup", "recip-dup", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), tr1))

	tr2 := buildTransfer("tr-dup-2", "item-dup", "sender-dup", "recip-dup", domain.TransferStatusPending, false)
	err := repo.Save(t.Context(), tr2)
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestItemTransferRepositorySaveShouldPersistNewTransfer(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-sv")
	seedUserForTransfer(t, db, "recip-sv")
	seedItemForTransfer(t, db, "item-sv", "sender-sv")

	tr := buildTransfer("tr-sv-1", "item-sv", "sender-sv", "recip-sv", domain.TransferStatusPending, true)
	require.NoError(t, repo.Save(t.Context(), tr))

	got, err := repo.Get(t.Context(), "tr-sv-1")
	require.NoError(t, err)
	require.Equal(t, "tr-sv-1", got.GetID())
	require.Equal(t, domain.TransferStatusPending, got.Status)
	require.True(t, got.TransferHistory)
	require.Nil(t, got.DecidedAt)
}

func TestItemTransferRepositorySaveShouldNotChangeImmutableFieldsOnUpdate(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-imm")
	seedUserForTransfer(t, db, "recip-imm")
	seedUserForTransfer(t, db, "other-imm")
	seedItemForTransfer(t, db, "item-imm", "sender-imm")
	seedItemForTransfer(t, db, "item-imm-other", "other-imm")

	tr := buildTransfer("tr-imm-1", "item-imm", "sender-imm", "recip-imm", domain.TransferStatusPending, true)
	require.NoError(t, repo.Save(t.Context(), tr))

	// Attempt to mutate immutable fields on the second Save.
	decided := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	tr.ItemID = "item-imm-other"
	tr.SenderID = "other-imm"
	tr.RecipientID = "other-imm"
	tr.TransferHistory = false
	tr.Status = domain.TransferStatusAccepted
	tr.DecidedAt = &decided
	require.NoError(t, repo.Save(t.Context(), tr))

	got, err := repo.Get(t.Context(), "tr-imm-1")
	require.NoError(t, err)
	require.Equal(t, "item-imm", got.ItemID)
	require.Equal(t, "sender-imm", got.SenderID)
	require.Equal(t, "recip-imm", got.RecipientID)
	require.True(t, got.TransferHistory)
	require.Equal(t, domain.TransferStatusAccepted, got.Status)
	require.NotNil(t, got.DecidedAt)
}

func TestItemTransferRepositorySaveShouldUpdateStatusAndDecidedAt(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-upd")
	seedUserForTransfer(t, db, "recip-upd")
	seedItemForTransfer(t, db, "item-upd", "sender-upd")

	tr := buildTransfer("tr-upd-1", "item-upd", "sender-upd", "recip-upd", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), tr))

	decided := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	tr.Status = domain.TransferStatusAccepted
	tr.DecidedAt = &decided
	require.NoError(t, repo.Save(t.Context(), tr))

	got, err := repo.Get(t.Context(), "tr-upd-1")
	require.NoError(t, err)
	require.Equal(t, domain.TransferStatusAccepted, got.Status)
	require.NotNil(t, got.DecidedAt)
	require.Equal(t, decided, *got.DecidedAt)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestItemTransferRepositoryDeleteShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemTransferRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.Delete(ctx, "tr-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferRepositoryDeleteShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newItemTransferRepo(t)

	err := repo.Delete(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferRepositoryDeleteShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemTransferRepository(db)
	db.Close()

	err := repo.Delete(t.Context(), "tr-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositoryDeleteShouldRemoveTransferWhenExists(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-del")
	seedUserForTransfer(t, db, "recip-del")
	seedItemForTransfer(t, db, "item-del", "sender-del")

	tr := buildTransfer("tr-del-1", "item-del", "sender-del", "recip-del", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), tr))

	require.NoError(t, repo.Delete(t.Context(), "tr-del-1"))

	_, err := repo.Get(t.Context(), "tr-del-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// ── ListBySender ──────────────────────────────────────────────────────────────

func TestItemTransferRepositoryListBySenderShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemTransferRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListBySender(ctx, "sender-1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferRepositoryListBySenderShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemTransferRepository(db)
	db.Close()

	_, err := repo.ListBySender(t.Context(), "sender-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositoryListBySenderShouldReturnEmptyWhenNoTransfers(t *testing.T) {
	repo, _ := newItemTransferRepo(t)

	transfers, err := repo.ListBySender(t.Context(), "nonexistent-sender", nil)
	require.NoError(t, err)
	require.Empty(t, transfers)
}

func TestItemTransferRepositoryListBySenderShouldReturnOnlySenderTransfers(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-ls-a")
	seedUserForTransfer(t, db, "sender-ls-b")
	seedUserForTransfer(t, db, "recip-ls")
	seedItemForTransfer(t, db, "item-ls-a", "sender-ls-a")
	seedItemForTransfer(t, db, "item-ls-b", "sender-ls-b")

	require.NoError(t, repo.Save(t.Context(), buildTransfer("tr-ls-a", "item-ls-a", "sender-ls-a", "recip-ls", domain.TransferStatusPending, false)))
	require.NoError(t, repo.Save(t.Context(), buildTransfer("tr-ls-b", "item-ls-b", "sender-ls-b", "recip-ls", domain.TransferStatusPending, false)))

	transfers, err := repo.ListBySender(t.Context(), "sender-ls-a", nil)
	require.NoError(t, err)
	require.Len(t, transfers, 1)
	require.Equal(t, "tr-ls-a", transfers[0].GetID())
}

func TestItemTransferRepositoryListBySenderShouldFilterByStatus(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-lsf")
	seedUserForTransfer(t, db, "recip-lsf")
	seedItemForTransfer(t, db, "item-lsf-1", "sender-lsf")
	seedItemForTransfer(t, db, "item-lsf-2", "sender-lsf")

	tr1 := buildTransfer("tr-lsf-1", "item-lsf-1", "sender-lsf", "recip-lsf", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), tr1))

	decided := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	tr2 := buildTransfer("tr-lsf-2", "item-lsf-2", "sender-lsf", "recip-lsf", domain.TransferStatusAccepted, false)
	tr2.DecidedAt = &decided
	require.NoError(t, repo.Save(t.Context(), tr2))

	status := domain.TransferStatusPending
	transfers, err := repo.ListBySender(t.Context(), "sender-lsf", &status)
	require.NoError(t, err)
	require.Len(t, transfers, 1)
	require.Equal(t, "tr-lsf-1", transfers[0].GetID())
}

// ── ListByRecipient ───────────────────────────────────────────────────────────

func TestItemTransferRepositoryListByRecipientShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemTransferRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByRecipient(ctx, "recip-1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferRepositoryListByRecipientShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemTransferRepository(db)
	db.Close()

	_, err := repo.ListByRecipient(t.Context(), "recip-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositoryListByRecipientShouldReturnEmptyWhenNoTransfers(t *testing.T) {
	repo, _ := newItemTransferRepo(t)

	transfers, err := repo.ListByRecipient(t.Context(), "nonexistent-recip", nil)
	require.NoError(t, err)
	require.Empty(t, transfers)
}

func TestItemTransferRepositoryListByRecipientShouldReturnOnlyRecipientTransfers(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-lr")
	seedUserForTransfer(t, db, "recip-lr-a")
	seedUserForTransfer(t, db, "recip-lr-b")
	seedItemForTransfer(t, db, "item-lr-a", "sender-lr")
	seedItemForTransfer(t, db, "item-lr-b", "sender-lr")

	require.NoError(t, repo.Save(t.Context(), buildTransfer("tr-lr-a", "item-lr-a", "sender-lr", "recip-lr-a", domain.TransferStatusPending, false)))
	require.NoError(t, repo.Save(t.Context(), buildTransfer("tr-lr-b", "item-lr-b", "sender-lr", "recip-lr-b", domain.TransferStatusPending, false)))

	transfers, err := repo.ListByRecipient(t.Context(), "recip-lr-a", nil)
	require.NoError(t, err)
	require.Len(t, transfers, 1)
	require.Equal(t, "tr-lr-a", transfers[0].GetID())
}

func TestItemTransferRepositoryListByRecipientShouldFilterByStatus(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-lrf")
	seedUserForTransfer(t, db, "recip-lrf")
	seedItemForTransfer(t, db, "item-lrf-1", "sender-lrf")
	seedItemForTransfer(t, db, "item-lrf-2", "sender-lrf")

	tr1 := buildTransfer("tr-lrf-1", "item-lrf-1", "sender-lrf", "recip-lrf", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), tr1))

	decided := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	tr2 := buildTransfer("tr-lrf-2", "item-lrf-2", "sender-lrf", "recip-lrf", domain.TransferStatusRejected, false)
	tr2.DecidedAt = &decided
	require.NoError(t, repo.Save(t.Context(), tr2))

	status := domain.TransferStatusRejected
	transfers, err := repo.ListByRecipient(t.Context(), "recip-lrf", &status)
	require.NoError(t, err)
	require.Len(t, transfers, 1)
	require.Equal(t, "tr-lrf-2", transfers[0].GetID())
}

// ── FindPendingByItem ─────────────────────────────────────────────────────────

func TestItemTransferRepositoryFindPendingByItemShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemTransferRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.FindPendingByItem(ctx, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferRepositoryFindPendingByItemShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemTransferRepository(db)
	db.Close()

	_, err := repo.FindPendingByItem(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositoryFindPendingByItemShouldReturnNilWhenNoPendingExists(t *testing.T) {
	repo, _ := newItemTransferRepo(t)

	got, err := repo.FindPendingByItem(t.Context(), "nonexistent-item")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestItemTransferRepositoryFindPendingByItemShouldReturnTransferWhenPendingExists(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-fp")
	seedUserForTransfer(t, db, "recip-fp")
	seedItemForTransfer(t, db, "item-fp", "sender-fp")

	tr := buildTransfer("tr-fp-1", "item-fp", "sender-fp", "recip-fp", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), tr))

	got, err := repo.FindPendingByItem(t.Context(), "item-fp")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "tr-fp-1", got.GetID())
	require.Equal(t, domain.TransferStatusPending, got.Status)
}

func TestItemTransferRepositoryFindPendingByItemShouldReturnNilWhenOnlyNonPendingExists(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-fpn")
	seedUserForTransfer(t, db, "recip-fpn")
	seedItemForTransfer(t, db, "item-fpn", "sender-fpn")

	decided := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	tr := buildTransfer("tr-fpn-1", "item-fpn", "sender-fpn", "recip-fpn", domain.TransferStatusAccepted, false)
	tr.DecidedAt = &decided
	require.NoError(t, repo.Save(t.Context(), tr))

	got, err := repo.FindPendingByItem(t.Context(), "item-fpn")
	require.NoError(t, err)
	require.Nil(t, got)
}

// ── HasPending ────────────────────────────────────────────────────────────────

func TestItemTransferRepositoryHasPendingShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemTransferRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.HasPending(ctx, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferRepositoryHasPendingShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemTransferRepository(db)
	db.Close()

	_, err := repo.HasPending(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositoryHasPendingShouldReturnFalseWhenNoPendingExists(t *testing.T) {
	repo, _ := newItemTransferRepo(t)

	has, err := repo.HasPending(t.Context(), "nonexistent-item")
	require.NoError(t, err)
	require.False(t, has)
}

func TestItemTransferRepositoryHasPendingShouldReturnTrueWhenPendingExists(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-hp")
	seedUserForTransfer(t, db, "recip-hp")
	seedItemForTransfer(t, db, "item-hp", "sender-hp")

	tr := buildTransfer("tr-hp-1", "item-hp", "sender-hp", "recip-hp", domain.TransferStatusPending, false)
	require.NoError(t, repo.Save(t.Context(), tr))

	has, err := repo.HasPending(t.Context(), "item-hp")
	require.NoError(t, err)
	require.True(t, has)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestItemTransferRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemTransferRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "tr-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemTransferRepositoryGetShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newItemTransferRepo(t)

	_, err := repo.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemTransferRepositoryGetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemTransferRepository(db)
	db.Close()

	_, err := repo.Get(t.Context(), "tr-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemTransferRepositoryGetShouldReturnTransferWhenRowExists(t *testing.T) {
	repo, db := newItemTransferRepo(t)
	seedUserForTransfer(t, db, "sender-get")
	seedUserForTransfer(t, db, "recip-get")
	seedItemForTransfer(t, db, "item-get", "sender-get")

	tr := buildTransfer("tr-get-1", "item-get", "sender-get", "recip-get", domain.TransferStatusPending, true)
	require.NoError(t, repo.Save(t.Context(), tr))

	got, err := repo.Get(t.Context(), "tr-get-1")
	require.NoError(t, err)
	require.Equal(t, "tr-get-1", got.GetID())
	require.Equal(t, "item-get", got.ItemID)
	require.Equal(t, "sender-get", got.SenderID)
	require.Equal(t, "recip-get", got.RecipientID)
	require.Equal(t, domain.TransferStatusPending, got.Status)
	require.True(t, got.TransferHistory)
	require.Equal(t, time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC), got.CreatedAt)
	require.Nil(t, got.DecidedAt)
}
