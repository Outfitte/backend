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

func newShareRepo(t *testing.T) (*sqlstore.ShareRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewShareRepository(db), db
}

func seedUserForShare(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (?, ?, 'hash', 'member', '2025-01-01T00:00:00Z')`,
		id, id+"@example.com")
	require.NoError(t, err)
}

func buildShare(id, ownerID, recipientID string, targetType domain.ShareTargetType, targetID string) domain.Share {
	var s domain.Share
	s.ID = id
	s.OwnerID = ownerID
	s.RecipientID = recipientID
	s.TargetType = targetType
	s.TargetID = targetID
	s.CreatedAt = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	return s
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestShareRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newShareRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "share-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareRepositoryGetShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newShareRepo(t)

	_, err := repo.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareRepositoryGetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewShareRepository(db)
	db.Close()

	_, err := repo.Get(t.Context(), "share-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositoryGetShouldReturnShareWhenRowExists(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-get")
	seedUserForShare(t, db, "recip-get")

	share := buildShare("share-get-1", "owner-get", "recip-get", domain.ShareTargetItem, "item-1")
	require.NoError(t, repo.Save(t.Context(), share))

	got, err := repo.Get(t.Context(), "share-get-1")
	require.NoError(t, err)
	require.Equal(t, "share-get-1", got.GetID())
	require.Equal(t, "owner-get", got.OwnerID)
	require.Equal(t, "recip-get", got.RecipientID)
	require.Equal(t, domain.ShareTargetItem, got.TargetType)
	require.Equal(t, "item-1", got.TargetID)
	require.Equal(t, time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC), got.CreatedAt)
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestShareRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newShareRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	share := buildShare("share-1", "owner-1", "recip-1", domain.ShareTargetItem, "item-1")
	err := repo.Save(ctx, share)
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewShareRepository(db)
	db.Close()

	share := buildShare("share-1", "owner-1", "recip-1", domain.ShareTargetItem, "item-1")
	err := repo.Save(t.Context(), share)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositorySaveShouldPersistNewShare(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-save")
	seedUserForShare(t, db, "recip-save")

	share := buildShare("share-save-1", "owner-save", "recip-save", domain.ShareTargetOutfit, "outfit-1")
	require.NoError(t, repo.Save(t.Context(), share))

	got, err := repo.Get(t.Context(), "share-save-1")
	require.NoError(t, err)
	require.Equal(t, "share-save-1", got.GetID())
	require.Equal(t, domain.ShareTargetOutfit, got.TargetType)
}

func TestShareRepositorySaveShouldUpdateExistingShare(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-upd")
	seedUserForShare(t, db, "recip-upd")
	seedUserForShare(t, db, "recip-upd2")

	share := buildShare("share-upd-1", "owner-upd", "recip-upd", domain.ShareTargetItem, "item-1")
	require.NoError(t, repo.Save(t.Context(), share))

	share.RecipientID = "recip-upd2"
	require.NoError(t, repo.Save(t.Context(), share))

	got, err := repo.Get(t.Context(), "share-upd-1")
	require.NoError(t, err)
	require.Equal(t, "recip-upd2", got.RecipientID)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestShareRepositoryDeleteShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newShareRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.Delete(ctx, "share-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareRepositoryDeleteShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newShareRepo(t)

	err := repo.Delete(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestShareRepositoryDeleteShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewShareRepository(db)
	db.Close()

	err := repo.Delete(t.Context(), "share-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositoryDeleteShouldRemoveShareWhenExists(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-del")
	seedUserForShare(t, db, "recip-del")

	share := buildShare("share-del-1", "owner-del", "recip-del", domain.ShareTargetItem, "item-1")
	require.NoError(t, repo.Save(t.Context(), share))

	require.NoError(t, repo.Delete(t.Context(), "share-del-1"))

	_, err := repo.Get(t.Context(), "share-del-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// ── ListByOwner ───────────────────────────────────────────────────────────────

func TestShareRepositoryListByOwnerShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newShareRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByOwner(ctx, "owner-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareRepositoryListByOwnerShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewShareRepository(db)
	db.Close()

	_, err := repo.ListByOwner(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositoryListByOwnerShouldReturnEmptyWhenOwnerHasNoShares(t *testing.T) {
	repo, _ := newShareRepo(t)

	shares, err := repo.ListByOwner(t.Context(), "nonexistent-owner")
	require.NoError(t, err)
	require.Empty(t, shares)
}

func TestShareRepositoryListByOwnerShouldReturnOnlyOwnerShares(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-a")
	seedUserForShare(t, db, "owner-b")
	seedUserForShare(t, db, "recip-ab")

	require.NoError(t, repo.Save(t.Context(), buildShare("share-a1", "owner-a", "recip-ab", domain.ShareTargetItem, "item-1")))
	require.NoError(t, repo.Save(t.Context(), buildShare("share-b1", "owner-b", "recip-ab", domain.ShareTargetItem, "item-2")))

	shares, err := repo.ListByOwner(t.Context(), "owner-a")
	require.NoError(t, err)
	require.Len(t, shares, 1)
	require.Equal(t, "share-a1", shares[0].GetID())
}

// ── ListByRecipient ───────────────────────────────────────────────────────────

func TestShareRepositoryListByRecipientShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newShareRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByRecipient(ctx, "recip-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareRepositoryListByRecipientShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewShareRepository(db)
	db.Close()

	_, err := repo.ListByRecipient(t.Context(), "recip-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositoryListByRecipientShouldReturnEmptyWhenRecipientHasNoShares(t *testing.T) {
	repo, _ := newShareRepo(t)

	shares, err := repo.ListByRecipient(t.Context(), "nonexistent-recip")
	require.NoError(t, err)
	require.Empty(t, shares)
}

func TestShareRepositoryListByRecipientShouldReturnOnlyRecipientShares(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-lr")
	seedUserForShare(t, db, "recip-lr1")
	seedUserForShare(t, db, "recip-lr2")

	require.NoError(t, repo.Save(t.Context(), buildShare("share-lr1", "owner-lr", "recip-lr1", domain.ShareTargetItem, "item-1")))
	require.NoError(t, repo.Save(t.Context(), buildShare("share-lr2", "owner-lr", "recip-lr2", domain.ShareTargetItem, "item-2")))

	shares, err := repo.ListByRecipient(t.Context(), "recip-lr1")
	require.NoError(t, err)
	require.Len(t, shares, 1)
	require.Equal(t, "share-lr1", shares[0].GetID())
}

// ── FindByTarget ──────────────────────────────────────────────────────────────

func TestShareRepositoryFindByTargetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newShareRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.FindByTarget(ctx, "owner-1", "recip-1", domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareRepositoryFindByTargetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewShareRepository(db)
	db.Close()

	_, err := repo.FindByTarget(t.Context(), "owner-1", "recip-1", domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositoryFindByTargetShouldReturnNilWhenNoMatch(t *testing.T) {
	repo, _ := newShareRepo(t)

	got, err := repo.FindByTarget(t.Context(), "owner-1", "recip-1", domain.ShareTargetItem, "item-1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestShareRepositoryFindByTargetShouldReturnShareWhenMatchExists(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-ft")
	seedUserForShare(t, db, "recip-ft")

	share := buildShare("share-ft-1", "owner-ft", "recip-ft", domain.ShareTargetLocation, "loc-1")
	require.NoError(t, repo.Save(t.Context(), share))

	got, err := repo.FindByTarget(t.Context(), "owner-ft", "recip-ft", domain.ShareTargetLocation, "loc-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "share-ft-1", got.GetID())
}

// ── DeleteByTarget ────────────────────────────────────────────────────────────

func TestShareRepositoryDeleteByTargetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newShareRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.DeleteByTarget(ctx, domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareRepositoryDeleteByTargetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewShareRepository(db)
	db.Close()

	err := repo.DeleteByTarget(t.Context(), domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositoryDeleteByTargetShouldReturnNoErrorWhenNoRowsMatch(t *testing.T) {
	repo, _ := newShareRepo(t)

	err := repo.DeleteByTarget(t.Context(), domain.ShareTargetItem, "nonexistent-item")
	require.NoError(t, err)
}

func TestShareRepositoryDeleteByTargetShouldRemoveAllMatchingShares(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-dbt")
	seedUserForShare(t, db, "recip-dbt1")
	seedUserForShare(t, db, "recip-dbt2")

	require.NoError(t, repo.Save(t.Context(), buildShare("share-dbt1", "owner-dbt", "recip-dbt1", domain.ShareTargetItem, "item-dbt")))
	require.NoError(t, repo.Save(t.Context(), buildShare("share-dbt2", "owner-dbt", "recip-dbt2", domain.ShareTargetItem, "item-dbt")))

	require.NoError(t, repo.DeleteByTarget(t.Context(), domain.ShareTargetItem, "item-dbt"))

	shares, err := repo.ListByOwner(t.Context(), "owner-dbt")
	require.NoError(t, err)
	require.Empty(t, shares)
}

func TestShareRepositoryDeleteByTargetShouldOnlyRemoveSharingForMatchingTarget(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-dbt2")
	seedUserForShare(t, db, "recip-dbt3")

	require.NoError(t, repo.Save(t.Context(), buildShare("share-dbt3", "owner-dbt2", "recip-dbt3", domain.ShareTargetItem, "item-keep")))
	require.NoError(t, repo.Save(t.Context(), buildShare("share-dbt4", "owner-dbt2", "recip-dbt3", domain.ShareTargetOutfit, "outfit-delete")))

	require.NoError(t, repo.DeleteByTarget(t.Context(), domain.ShareTargetOutfit, "outfit-delete"))

	shares, err := repo.ListByOwner(t.Context(), "owner-dbt2")
	require.NoError(t, err)
	require.Len(t, shares, 1)
	require.Equal(t, "share-dbt3", shares[0].GetID())
}

// ── HasDirectAccess ───────────────────────────────────────────────────────────

func TestShareRepositoryHasDirectAccessShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newShareRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.HasDirectAccess(ctx, "recip-1", domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareRepositoryHasDirectAccessShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewShareRepository(db)
	db.Close()

	_, err := repo.HasDirectAccess(t.Context(), "recip-1", domain.ShareTargetItem, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositoryHasDirectAccessShouldReturnFalseWhenNoShareExists(t *testing.T) {
	repo, _ := newShareRepo(t)

	has, err := repo.HasDirectAccess(t.Context(), "recip-1", domain.ShareTargetItem, "item-1")
	require.NoError(t, err)
	require.False(t, has)
}

func TestShareRepositoryHasDirectAccessShouldReturnTrueWhenShareExists(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-hda")
	seedUserForShare(t, db, "recip-hda")

	share := buildShare("share-hda-1", "owner-hda", "recip-hda", domain.ShareTargetItem, "item-hda")
	require.NoError(t, repo.Save(t.Context(), share))

	has, err := repo.HasDirectAccess(t.Context(), "recip-hda", domain.ShareTargetItem, "item-hda")
	require.NoError(t, err)
	require.True(t, has)
}

// ── ListByRecipientAndType ────────────────────────────────────────────────────

func TestShareRepositoryListByRecipientAndTypeShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newShareRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByRecipientAndType(ctx, "recip-1", domain.ShareTargetItem)
	require.ErrorIs(t, err, context.Canceled)
}

func TestShareRepositoryListByRecipientAndTypeShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewShareRepository(db)
	db.Close()

	_, err := repo.ListByRecipientAndType(t.Context(), "recip-1", domain.ShareTargetItem)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositoryListByRecipientAndTypeShouldReturnEmptyWhenNoMatches(t *testing.T) {
	repo, _ := newShareRepo(t)

	shares, err := repo.ListByRecipientAndType(t.Context(), "recip-1", domain.ShareTargetItem)
	require.NoError(t, err)
	require.Empty(t, shares)
}

func TestShareRepositoryListByRecipientAndTypeShouldFilterByType(t *testing.T) {
	repo, db := newShareRepo(t)
	seedUserForShare(t, db, "owner-lbrt")
	seedUserForShare(t, db, "recip-lbrt")

	require.NoError(t, repo.Save(t.Context(), buildShare("share-lbrt1", "owner-lbrt", "recip-lbrt", domain.ShareTargetItem, "item-1")))
	require.NoError(t, repo.Save(t.Context(), buildShare("share-lbrt2", "owner-lbrt", "recip-lbrt", domain.ShareTargetOutfit, "outfit-1")))

	shares, err := repo.ListByRecipientAndType(t.Context(), "recip-lbrt", domain.ShareTargetItem)
	require.NoError(t, err)
	require.Len(t, shares, 1)
	require.Equal(t, "share-lbrt1", shares[0].GetID())
}
