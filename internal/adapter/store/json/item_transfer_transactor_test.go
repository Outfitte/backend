package json_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/adapter/store/json"
	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

func newTransferTransactor(t *testing.T) (*json.ItemTransferTransactor, *json.ItemTransferRepository, *json.ItemRepository, *json.WearLogRepository, *json.OutfitRepository, *json.ShareRepository, *json.OutfitLogRepository) {
	t.Helper()
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	wearLogs := json.NewWearLogRepository(dir)
	outfits := json.NewOutfitRepository(dir)
	shares := json.NewShareRepository(dir)
	outfitLogs := json.NewOutfitLogRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, wearLogs, outfits, shares, outfitLogs, &sync.Mutex{})
	return tr, transfers, items, wearLogs, outfits, shares, outfitLogs
}

// --- Accept ---

func TestAcceptShouldReturnNotFoundWhenTransferDoesNotExist(t *testing.T) {
	tr, _, _, _, _, _, _ := newTransferTransactor(t)

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAcceptShouldReturnValidationErrorWhenTransferIsNotPending(t *testing.T) {
	tr, transfers, _, _, _, _, _ := newTransferTransactor(t)

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.Status = domain.TransferStatusAccepted
	require.NoError(t, transfers.Save(t.Context(), transfer))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestAcceptShouldReturnNotFoundWhenItemDoesNotExist(t *testing.T) {
	tr, transfers, _, _, _, _, _ := newTransferTransactor(t)

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAcceptShouldReturnForbiddenWhenSenderNoLongerOwnsItem(t *testing.T) {
	tr, transfers, items, _, _, _, _ := newTransferTransactor(t)

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "other-user"
	require.NoError(t, items.Save(t.Context(), item))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestAcceptShouldReturnValidationErrorWhenItemIsArchived(t *testing.T) {
	tr, transfers, items, _, _, _, _ := newTransferTransactor(t)

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	now := time.Now()
	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	item.ArchivedAt = &now
	require.NoError(t, items.Save(t.Context(), item))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestAcceptShouldReturnValidationErrorWhenItemIsDisposed(t *testing.T) {
	tr, transfers, items, _, _, _, _ := newTransferTransactor(t)

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	reason := domain.DisposalDonated
	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	item.DisposalReason = &reason
	require.NoError(t, items.Save(t.Context(), item))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestAcceptShouldTransferOwnershipAndReassignWearLogsWhenTransferHistoryIsTrue(t *testing.T) {
	tr, transfers, items, wearLogs, outfits, shares, _ := newTransferTransactor(t)

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.TransferHistory = true
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	loc := "loc1"
	item.LocationID = &loc
	require.NoError(t, items.Save(t.Context(), item))

	var wl1, wl2 domain.WearLog
	wl1.ID = "wl1"
	wl1.ItemID = "item1"
	wl1.OwnerID = "sender1"
	wl2.ID = "wl2"
	wl2.ItemID = "item1"
	wl2.OwnerID = "sender1"
	require.NoError(t, wearLogs.Save(t.Context(), wl1))
	require.NoError(t, wearLogs.Save(t.Context(), wl2))

	var outfit domain.Outfit
	outfit.ID = "outfit1"
	outfit.OwnerID = "sender1"
	outfit.Items = []domain.OutfitItem{{OutfitID: "outfit1", ItemID: "item1", Position: 0}}
	require.NoError(t, outfits.Save(t.Context(), outfit))

	var share domain.Share
	share.ID = "share1"
	share.TargetType = domain.ShareTargetItem
	share.TargetID = "item1"
	require.NoError(t, shares.Save(t.Context(), share))

	got, err := tr.Accept(t.Context(), "transfer1")
	require.NoError(t, err)
	require.Equal(t, domain.TransferStatusAccepted, got.Status)
	require.NotNil(t, got.DecidedAt)

	// Item owner updated, location cleared
	storedItem, err := items.Get(t.Context(), "item1")
	require.NoError(t, err)
	require.Equal(t, "recipient1", storedItem.OwnerID)
	require.Nil(t, storedItem.LocationID)

	// Wear logs re-assigned to recipient
	storedWL1, err := wearLogs.Get(t.Context(), "wl1")
	require.NoError(t, err)
	require.Equal(t, "recipient1", storedWL1.OwnerID)
	storedWL2, err := wearLogs.Get(t.Context(), "wl2")
	require.NoError(t, err)
	require.Equal(t, "recipient1", storedWL2.OwnerID)

	// Item removed from outfit
	storedOutfit, err := outfits.Get(t.Context(), "outfit1")
	require.NoError(t, err)
	require.Empty(t, storedOutfit.Items)

	// Share deleted
	_, err = shares.Get(t.Context(), "share1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAcceptShouldTransferOwnershipAndDeleteWearLogsWhenTransferHistoryIsFalse(t *testing.T) {
	tr, transfers, items, wearLogs, outfits, shares, _ := newTransferTransactor(t)

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.TransferHistory = false
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	var wl domain.WearLog
	wl.ID = "wl1"
	wl.ItemID = "item1"
	require.NoError(t, wearLogs.Save(t.Context(), wl))

	var outfit domain.Outfit
	outfit.ID = "outfit1"
	outfit.OwnerID = "sender1"
	outfit.Items = []domain.OutfitItem{{OutfitID: "outfit1", ItemID: "item1", Position: 0}}
	require.NoError(t, outfits.Save(t.Context(), outfit))

	var share domain.Share
	share.ID = "share1"
	share.TargetType = domain.ShareTargetItem
	share.TargetID = "item1"
	require.NoError(t, shares.Save(t.Context(), share))

	got, err := tr.Accept(t.Context(), "transfer1")
	require.NoError(t, err)
	require.Equal(t, domain.TransferStatusAccepted, got.Status)
	require.NotNil(t, got.DecidedAt)

	// Item owner updated
	storedItem, err := items.Get(t.Context(), "item1")
	require.NoError(t, err)
	require.Equal(t, "recipient1", storedItem.OwnerID)

	// Wear log deleted
	_, err = wearLogs.Get(t.Context(), "wl1")
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Item removed from outfit
	storedOutfit, err := outfits.Get(t.Context(), "outfit1")
	require.NoError(t, err)
	require.Empty(t, storedOutfit.Items)

	// Share deleted
	_, err = shares.Get(t.Context(), "share1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAcceptShouldSkipOutfitsThatDoNotContainTheItem(t *testing.T) {
	tr, transfers, items, _, outfits, _, _ := newTransferTransactor(t)

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	// Outfit with item1
	var outfitWith domain.Outfit
	outfitWith.ID = "outfit-with"
	outfitWith.OwnerID = "sender1"
	outfitWith.Items = []domain.OutfitItem{{OutfitID: "outfit-with", ItemID: "item1", Position: 0}}
	require.NoError(t, outfits.Save(t.Context(), outfitWith))

	// Outfit without item1 — exercises the outfitContainsItem return-false path
	var outfitWithout domain.Outfit
	outfitWithout.ID = "outfit-without"
	outfitWithout.OwnerID = "sender1"
	outfitWithout.Items = []domain.OutfitItem{{OutfitID: "outfit-without", ItemID: "other-item", Position: 0}}
	require.NoError(t, outfits.Save(t.Context(), outfitWithout))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.NoError(t, err)

	// outfit-with should have item removed; outfit-without should be unchanged
	storedWith, err := outfits.Get(t.Context(), "outfit-with")
	require.NoError(t, err)
	require.Empty(t, storedWith.Items)

	storedWithout, err := outfits.Get(t.Context(), "outfit-without")
	require.NoError(t, err)
	require.Len(t, storedWithout.Items, 1)
}

func TestAcceptShouldReturnIOErrorWhenWearLogStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, json.NewWearLogRepository(dir), json.NewOutfitRepository(dir), json.NewShareRepository(dir), json.NewOutfitLogRepository(dir), &sync.Mutex{})

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "wear_logs.json"), []byte("not json"), 0o644))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAcceptShouldReturnIOErrorWhenOutfitStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, json.NewWearLogRepository(dir), json.NewOutfitRepository(dir), json.NewShareRepository(dir), json.NewOutfitLogRepository(dir), &sync.Mutex{})

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "outfits.json"), []byte("not json"), 0o644))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAcceptShouldReturnIOErrorWhenShareStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, json.NewWearLogRepository(dir), json.NewOutfitRepository(dir), json.NewShareRepository(dir), json.NewOutfitLogRepository(dir), &sync.Mutex{})

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "shares.json"), []byte("not json"), 0o644))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAcceptShouldReturnIOErrorWhenItemStorageIsNotWritable(t *testing.T) {
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, json.NewWearLogRepository(dir), json.NewOutfitRepository(dir), json.NewShareRepository(dir), json.NewOutfitLogRepository(dir), &sync.Mutex{})

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	require.NoError(t, os.Chmod(filepath.Join(dir, "items.json"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "items.json"), 0o644) })

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAcceptShouldReturnIOErrorWhenTransferStorageIsNotWritable(t *testing.T) {
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, json.NewWearLogRepository(dir), json.NewOutfitRepository(dir), json.NewShareRepository(dir), json.NewOutfitLogRepository(dir), &sync.Mutex{})

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	require.NoError(t, os.Chmod(filepath.Join(dir, "item_transfers.json"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "item_transfers.json"), 0o644) })

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAcceptShouldReturnIOErrorWhenWearLogSaveFailsDuringHistoryTransfer(t *testing.T) {
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	wearLogs := json.NewWearLogRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, wearLogs, json.NewOutfitRepository(dir), json.NewShareRepository(dir), json.NewOutfitLogRepository(dir), &sync.Mutex{})

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.TransferHistory = true
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	var wl domain.WearLog
	wl.ID = "wl1"
	wl.ItemID = "item1"
	require.NoError(t, wearLogs.Save(t.Context(), wl))

	require.NoError(t, os.Chmod(filepath.Join(dir, "wear_logs.json"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "wear_logs.json"), 0o644) })

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAcceptShouldReturnIOErrorWhenWearLogDeleteFailsDuringNonHistoryTransfer(t *testing.T) {
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	wearLogs := json.NewWearLogRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, wearLogs, json.NewOutfitRepository(dir), json.NewShareRepository(dir), json.NewOutfitLogRepository(dir), &sync.Mutex{})

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.TransferHistory = false
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	var wl domain.WearLog
	wl.ID = "wl1"
	wl.ItemID = "item1"
	require.NoError(t, wearLogs.Save(t.Context(), wl))

	require.NoError(t, os.Chmod(filepath.Join(dir, "wear_logs.json"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "wear_logs.json"), 0o644) })

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAcceptShouldReturnIOErrorWhenOutfitDeleteItemFails(t *testing.T) {
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	outfits := json.NewOutfitRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, json.NewWearLogRepository(dir), outfits, json.NewShareRepository(dir), json.NewOutfitLogRepository(dir), &sync.Mutex{})

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	var outfit domain.Outfit
	outfit.ID = "outfit1"
	outfit.OwnerID = "sender1"
	outfit.Items = []domain.OutfitItem{{OutfitID: "outfit1", ItemID: "item1", Position: 0}}
	require.NoError(t, outfits.Save(t.Context(), outfit))

	require.NoError(t, os.Chmod(filepath.Join(dir, "outfits.json"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "outfits.json"), 0o644) })

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAcceptShouldRemoveWearLogIDFromOutfitLogWhenTransferHistoryIsFalse(t *testing.T) {
	tr, transfers, items, wearLogs, _, _, outfitLogs := newTransferTransactor(t)

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.TransferHistory = false
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	// Create a wear log for item1 and link it to an outfit log
	var wl domain.WearLog
	wl.ID = "wl1"
	wl.ItemID = "item1"
	require.NoError(t, wearLogs.Save(t.Context(), wl))

	var ol domain.OutfitLog
	ol.ID = "ol1"
	require.NoError(t, outfitLogs.Save(t.Context(), ol))
	require.NoError(t, outfitLogs.LinkWearLog(t.Context(), "ol1", "wl1"))

	_, err := tr.Accept(t.Context(), "transfer1")
	require.NoError(t, err)

	// Wear log deleted
	_, err = wearLogs.Get(t.Context(), "wl1")
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Outfit log still exists but wl1 link is removed
	storedOL, err := outfitLogs.Get(t.Context(), "ol1")
	require.NoError(t, err)
	require.Empty(t, storedOL.WearLogIDs)
}

func TestAcceptShouldReturnIOErrorWhenOutfitLogStorageIsCorruptDuringWearLogLinkRemoval(t *testing.T) {
	dir := t.TempDir()
	transfers := json.NewItemTransferRepository(dir)
	items := json.NewItemRepository(dir)
	wearLogs := json.NewWearLogRepository(dir)
	outfitLogs := json.NewOutfitLogRepository(dir)
	tr := json.NewItemTransferTransactor(transfers, items, wearLogs, json.NewOutfitRepository(dir), json.NewShareRepository(dir), outfitLogs, &sync.Mutex{})

	var transfer domain.ItemTransfer
	transfer.ID = "transfer1"
	transfer.ItemID = "item1"
	transfer.SenderID = "sender1"
	transfer.RecipientID = "recipient1"
	transfer.TransferHistory = false
	transfer.Status = domain.TransferStatusPending
	require.NoError(t, transfers.Save(t.Context(), transfer))

	var item domain.Item
	item.ID = "item1"
	item.OwnerID = "sender1"
	require.NoError(t, items.Save(t.Context(), item))

	var wl domain.WearLog
	wl.ID = "wl1"
	wl.ItemID = "item1"
	require.NoError(t, wearLogs.Save(t.Context(), wl))

	// Create an outfit log linked to wl1 so RemoveWearLogLink will try to save
	var ol domain.OutfitLog
	ol.ID = "ol1"
	require.NoError(t, outfitLogs.Save(t.Context(), ol))
	require.NoError(t, outfitLogs.LinkWearLog(t.Context(), "ol1", "wl1"))

	// Make outfit_logs.json read-only so RemoveWearLogLink's Save fails
	require.NoError(t, os.Chmod(filepath.Join(dir, "outfit_logs.json"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "outfit_logs.json"), 0o644) })

	_, err := tr.Accept(t.Context(), "transfer1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAcceptShouldReturnErrorWhenContextCancelled(t *testing.T) {
	tr, _, _, _, _, _, _ := newTransferTransactor(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := tr.Accept(ctx, "transfer1")
	require.ErrorIs(t, err, context.Canceled)
}
