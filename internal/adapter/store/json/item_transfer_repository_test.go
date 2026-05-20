package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/outfitte/backend/internal/adapter/store/json"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
	"github.com/stretchr/testify/require"
)

func newTransfer(id, itemID, senderID, recipientID string, status domain.TransferStatus) domain.ItemTransfer {
	var t domain.ItemTransfer
	t.ID = id
	t.ItemID = itemID
	t.SenderID = senderID
	t.RecipientID = recipientID
	t.Status = status
	return t
}

func TestNewItemTransferRepositoryShouldImplementItemTransferRepository(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	require.Implements(t, (*ports.ItemTransferRepository)(nil), r)
}

// --- Get ---

func TestGetShouldReturnErrorWhenContextIsCancelledForItemTransfer(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Get(ctx, "1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetShouldReturnNotFoundWhenTransferDoesNotExist(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())

	_, err := r.Get(t.Context(), "1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetShouldReturnTransferWhenFound(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr := newTransfer("1", "item1", "sender1", "recip1", domain.TransferStatusPending)
	require.NoError(t, r.Save(t.Context(), tr))

	got, err := r.Get(t.Context(), "1")
	require.NoError(t, err)
	require.Equal(t, tr, got)
}

// --- Save ---

func TestSaveShouldReturnErrorWhenContextIsCancelledForItemTransfer(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.ItemTransfer{})
	require.ErrorIs(t, err, context.Canceled)
}

// --- Delete ---

func TestDeleteShouldReturnErrorWhenContextIsCancelledForItemTransfer(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Delete(ctx, "1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnNotFoundWhenTransferDoesNotExist(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())

	err := r.Delete(t.Context(), "1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteShouldRemoveTransferWhenFound(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr := newTransfer("1", "item1", "sender1", "recip1", domain.TransferStatusPending)
	require.NoError(t, r.Save(t.Context(), tr))

	require.NoError(t, r.Delete(t.Context(), "1"))

	_, err := r.Get(t.Context(), "1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// --- ListBySender ---

func TestListBySenderShouldReturnErrorWhenContextIsCancelledForItemTransfer(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListBySender(ctx, "sender1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestListBySenderShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "item_transfers.json"), []byte("not json"), 0o644))
	r := json.NewItemTransferRepository(dir)

	_, err := r.ListBySender(t.Context(), "sender1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListBySenderShouldReturnEmptyWhenNoTransfersExist(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())

	transfers, err := r.ListBySender(t.Context(), "sender1", nil)
	require.NoError(t, err)
	require.Empty(t, transfers)
}

func TestListBySenderShouldReturnEmptyWhenNoTransfersMatchSender(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr := newTransfer("1", "item1", "sender2", "recip1", domain.TransferStatusPending)
	require.NoError(t, r.Save(t.Context(), tr))

	transfers, err := r.ListBySender(t.Context(), "sender1", nil)
	require.NoError(t, err)
	require.Empty(t, transfers)
}

func TestListBySenderShouldReturnOnlyTransfersForSender(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr1 := newTransfer("1", "item1", "sender1", "recip1", domain.TransferStatusPending)
	tr2 := newTransfer("2", "item2", "sender1", "recip1", domain.TransferStatusAccepted)
	tr3 := newTransfer("3", "item3", "sender2", "recip1", domain.TransferStatusPending)
	require.NoError(t, r.Save(t.Context(), tr1))
	require.NoError(t, r.Save(t.Context(), tr2))
	require.NoError(t, r.Save(t.Context(), tr3))

	transfers, err := r.ListBySender(t.Context(), "sender1", nil)
	require.NoError(t, err)
	require.Len(t, transfers, 2)
}

func TestListBySenderShouldFilterByStatusWhenStatusFilterIsSet(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr1 := newTransfer("1", "item1", "sender1", "recip1", domain.TransferStatusPending)
	tr2 := newTransfer("2", "item2", "sender1", "recip1", domain.TransferStatusAccepted)
	require.NoError(t, r.Save(t.Context(), tr1))
	require.NoError(t, r.Save(t.Context(), tr2))

	status := domain.TransferStatusPending
	transfers, err := r.ListBySender(t.Context(), "sender1", &status)
	require.NoError(t, err)
	require.Len(t, transfers, 1)
	require.Equal(t, "1", transfers[0].ID)
}

// --- ListByRecipient ---

func TestListByRecipientShouldReturnErrorWhenContextIsCancelledForItemTransfer(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListByRecipient(ctx, "recip1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestListByRecipientShouldReturnIOErrorWhenStorageIsCorruptForItemTransfer(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "item_transfers.json"), []byte("not json"), 0o644))
	r := json.NewItemTransferRepository(dir)

	_, err := r.ListByRecipient(t.Context(), "recip1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListByRecipientShouldReturnEmptyWhenNoTransfersMatchRecipient(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())

	transfers, err := r.ListByRecipient(t.Context(), "recip1", nil)
	require.NoError(t, err)
	require.Empty(t, transfers)
}

func TestListByRecipientShouldReturnOnlyTransfersForRecipient(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr1 := newTransfer("1", "item1", "sender1", "recip1", domain.TransferStatusPending)
	tr2 := newTransfer("2", "item2", "sender1", "recip1", domain.TransferStatusAccepted)
	tr3 := newTransfer("3", "item3", "sender1", "recip2", domain.TransferStatusPending)
	require.NoError(t, r.Save(t.Context(), tr1))
	require.NoError(t, r.Save(t.Context(), tr2))
	require.NoError(t, r.Save(t.Context(), tr3))

	transfers, err := r.ListByRecipient(t.Context(), "recip1", nil)
	require.NoError(t, err)
	require.Len(t, transfers, 2)
}

func TestListByRecipientShouldFilterByStatusWhenStatusFilterIsSet(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr1 := newTransfer("1", "item1", "sender1", "recip1", domain.TransferStatusPending)
	tr2 := newTransfer("2", "item2", "sender1", "recip1", domain.TransferStatusRejected)
	require.NoError(t, r.Save(t.Context(), tr1))
	require.NoError(t, r.Save(t.Context(), tr2))

	status := domain.TransferStatusRejected
	transfers, err := r.ListByRecipient(t.Context(), "recip1", &status)
	require.NoError(t, err)
	require.Len(t, transfers, 1)
	require.Equal(t, "2", transfers[0].ID)
}

// --- FindPendingByItem ---

func TestFindPendingByItemShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.FindPendingByItem(ctx, "item1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestFindPendingByItemShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "item_transfers.json"), []byte("not json"), 0o644))
	r := json.NewItemTransferRepository(dir)

	_, err := r.FindPendingByItem(t.Context(), "item1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestFindPendingByItemShouldReturnNilWhenNoPendingTransferExists(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())

	got, err := r.FindPendingByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestFindPendingByItemShouldReturnNilWhenTransferExistsButNotPending(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr := newTransfer("1", "item1", "sender1", "recip1", domain.TransferStatusAccepted)
	require.NoError(t, r.Save(t.Context(), tr))

	got, err := r.FindPendingByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestFindPendingByItemShouldReturnPendingTransferWhenFound(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr := newTransfer("1", "item1", "sender1", "recip1", domain.TransferStatusPending)
	require.NoError(t, r.Save(t.Context(), tr))

	got, err := r.FindPendingByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, tr, *got)
}

// --- HasPending ---

func TestHasPendingShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.HasPending(ctx, "item1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestHasPendingShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "item_transfers.json"), []byte("not json"), 0o644))
	r := json.NewItemTransferRepository(dir)

	_, err := r.HasPending(t.Context(), "item1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestHasPendingShouldReturnFalseWhenNoPendingTransferExists(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())

	has, err := r.HasPending(t.Context(), "item1")
	require.NoError(t, err)
	require.False(t, has)
}

func TestHasPendingShouldReturnTrueWhenPendingTransferExists(t *testing.T) {
	r := json.NewItemTransferRepository(t.TempDir())
	tr := newTransfer("1", "item1", "sender1", "recip1", domain.TransferStatusPending)
	require.NoError(t, r.Save(t.Context(), tr))

	has, err := r.HasPending(t.Context(), "item1")
	require.NoError(t, err)
	require.True(t, has)
}
