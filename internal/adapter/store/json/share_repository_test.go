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

func newShare(id, ownerID, recipientID string, targetType domain.ShareTargetType, targetID string) domain.Share {
	var s domain.Share
	s.ID = id
	s.OwnerID = ownerID
	s.RecipientID = recipientID
	s.TargetType = targetType
	s.TargetID = targetID
	return s
}

func TestNewShareRepositoryShouldImplementShareRepository(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	require.Implements(t, (*ports.ShareRepository)(nil), r)
}

// --- Get ---

func TestGetShouldReturnErrorWhenContextIsCancelledForShare(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Get(ctx, "1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetShouldReturnNotFoundWhenShareDoesNotExist(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())

	_, err := r.Get(t.Context(), "1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetShouldReturnShareWhenFound(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, r.Save(t.Context(), s))

	got, err := r.Get(t.Context(), "1")
	require.NoError(t, err)
	require.Equal(t, s, got)
}

// --- Save ---

func TestSaveShouldReturnErrorWhenContextIsCancelledForShare(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.Share{})
	require.ErrorIs(t, err, context.Canceled)
}

// --- Delete ---

func TestDeleteShouldReturnErrorWhenContextIsCancelledForShare(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Delete(ctx, "1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnNotFoundWhenShareDoesNotExist(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())

	err := r.Delete(t.Context(), "1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteShouldRemoveShareWhenFound(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, r.Save(t.Context(), s))

	require.NoError(t, r.Delete(t.Context(), "1"))

	_, err := r.Get(t.Context(), "1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// --- ListByOwner ---

func TestListByOwnerShouldReturnErrorWhenContextIsCancelledForShare(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListByOwner(ctx, "owner1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestListByOwnerShouldReturnIOErrorWhenStorageIsCorruptForShare(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shares.json"), []byte("not json"), 0o644))
	r := json.NewShareRepository(dir)

	_, err := r.ListByOwner(t.Context(), "owner1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListByOwnerShouldReturnEmptyWhenNoSharesExist(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())

	shares, err := r.ListByOwner(t.Context(), "owner1")
	require.NoError(t, err)
	require.Empty(t, shares)
}

func TestListByOwnerShouldReturnEmptyWhenNoSharesMatchOwner(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s := newShare("1", "owner2", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, r.Save(t.Context(), s))

	shares, err := r.ListByOwner(t.Context(), "owner1")
	require.NoError(t, err)
	require.Empty(t, shares)
}

func TestListByOwnerShouldReturnOnlySharesForOwner(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s1 := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	s2 := newShare("2", "owner1", "recip2", domain.ShareTargetItem, "item2")
	s3 := newShare("3", "owner2", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, r.Save(t.Context(), s1))
	require.NoError(t, r.Save(t.Context(), s2))
	require.NoError(t, r.Save(t.Context(), s3))

	shares, err := r.ListByOwner(t.Context(), "owner1")
	require.NoError(t, err)
	require.Len(t, shares, 2)
}

// --- ListByRecipient ---

func TestListByRecipientShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListByRecipient(ctx, "recip1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestListByRecipientShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shares.json"), []byte("not json"), 0o644))
	r := json.NewShareRepository(dir)

	_, err := r.ListByRecipient(t.Context(), "recip1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListByRecipientShouldReturnEmptyWhenNoSharesMatchRecipient(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())

	shares, err := r.ListByRecipient(t.Context(), "recip1")
	require.NoError(t, err)
	require.Empty(t, shares)
}

func TestListByRecipientShouldReturnOnlySharesForRecipient(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s1 := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	s2 := newShare("2", "owner1", "recip1", domain.ShareTargetOutfit, "outfit1")
	s3 := newShare("3", "owner1", "recip2", domain.ShareTargetItem, "item1")
	require.NoError(t, r.Save(t.Context(), s1))
	require.NoError(t, r.Save(t.Context(), s2))
	require.NoError(t, r.Save(t.Context(), s3))

	shares, err := r.ListByRecipient(t.Context(), "recip1")
	require.NoError(t, err)
	require.Len(t, shares, 2)
}

// --- FindByTarget ---

func TestFindByTargetShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.FindByTarget(ctx, "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestFindByTargetShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shares.json"), []byte("not json"), 0o644))
	r := json.NewShareRepository(dir)

	_, err := r.FindByTarget(t.Context(), "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestFindByTargetShouldReturnNilWhenNoMatchExists(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())

	got, err := r.FindByTarget(t.Context(), "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestFindByTargetShouldReturnNilWhenPartialMatchOnly(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, r.Save(t.Context(), s))

	got, err := r.FindByTarget(t.Context(), "owner1", "recip1", domain.ShareTargetItem, "item2")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestFindByTargetShouldReturnShareWhenAllFieldsMatch(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, r.Save(t.Context(), s))

	got, err := r.FindByTarget(t.Context(), "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, s, *got)
}

// --- DeleteByTarget ---

func TestDeleteByTargetShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.DeleteByTarget(ctx, domain.ShareTargetItem, "item1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteByTargetShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shares.json"), []byte("not json"), 0o644))
	r := json.NewShareRepository(dir)

	err := r.DeleteByTarget(t.Context(), domain.ShareTargetItem, "item1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDeleteByTargetShouldSucceedWhenNoMatchesExist(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())

	err := r.DeleteByTarget(t.Context(), domain.ShareTargetItem, "item1")
	require.NoError(t, err)
}

func TestDeleteByTargetShouldRemoveMatchingSharesOnly(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s1 := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	s2 := newShare("2", "owner1", "recip2", domain.ShareTargetItem, "item1")
	s3 := newShare("3", "owner1", "recip1", domain.ShareTargetOutfit, "outfit1")
	require.NoError(t, r.Save(t.Context(), s1))
	require.NoError(t, r.Save(t.Context(), s2))
	require.NoError(t, r.Save(t.Context(), s3))

	err := r.DeleteByTarget(t.Context(), domain.ShareTargetItem, "item1")
	require.NoError(t, err)

	remaining, err := r.ListByOwner(t.Context(), "owner1")
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	require.Equal(t, "3", remaining[0].ID)
}

// --- HasDirectAccess ---

func TestHasDirectAccessShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.HasDirectAccess(ctx, "recip1", domain.ShareTargetItem, "item1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestHasDirectAccessShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shares.json"), []byte("not json"), 0o644))
	r := json.NewShareRepository(dir)

	_, err := r.HasDirectAccess(t.Context(), "recip1", domain.ShareTargetItem, "item1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestHasDirectAccessShouldReturnFalseWhenNoMatchExists(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())

	has, err := r.HasDirectAccess(t.Context(), "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, err)
	require.False(t, has)
}

func TestHasDirectAccessShouldReturnFalseWhenPartialMatchOnly(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, r.Save(t.Context(), s))

	has, err := r.HasDirectAccess(t.Context(), "recip1", domain.ShareTargetItem, "item2")
	require.NoError(t, err)
	require.False(t, has)
}

func TestHasDirectAccessShouldReturnTrueWhenMatchExists(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, r.Save(t.Context(), s))

	has, err := r.HasDirectAccess(t.Context(), "recip1", domain.ShareTargetItem, "item1")
	require.NoError(t, err)
	require.True(t, has)
}

// --- ListByRecipientAndType ---

func TestListByRecipientAndTypeShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListByRecipientAndType(ctx, "recip1", domain.ShareTargetItem)
	require.ErrorIs(t, err, context.Canceled)
}

func TestListByRecipientAndTypeShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shares.json"), []byte("not json"), 0o644))
	r := json.NewShareRepository(dir)

	_, err := r.ListByRecipientAndType(t.Context(), "recip1", domain.ShareTargetItem)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListByRecipientAndTypeShouldReturnEmptyWhenNoSharesMatchRecipientAndType(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())

	shares, err := r.ListByRecipientAndType(t.Context(), "recip1", domain.ShareTargetItem)
	require.NoError(t, err)
	require.Empty(t, shares)
}

func TestListByRecipientAndTypeShouldReturnOnlySharesMatchingBothFields(t *testing.T) {
	r := json.NewShareRepository(t.TempDir())
	s1 := newShare("1", "owner1", "recip1", domain.ShareTargetItem, "item1")
	s2 := newShare("2", "owner1", "recip1", domain.ShareTargetOutfit, "outfit1")
	s3 := newShare("3", "owner1", "recip2", domain.ShareTargetItem, "item2")
	require.NoError(t, r.Save(t.Context(), s1))
	require.NoError(t, r.Save(t.Context(), s2))
	require.NoError(t, r.Save(t.Context(), s3))

	shares, err := r.ListByRecipientAndType(t.Context(), "recip1", domain.ShareTargetItem)
	require.NoError(t, err)
	require.Len(t, shares, 1)
	require.Equal(t, "1", shares[0].ID)
}
