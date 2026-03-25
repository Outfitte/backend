package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/adapter/store/sqlstore"
	"github.com/outfitte/outfitte/internal/domain"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newOutfitRepo(t *testing.T) (*sqlstore.OutfitRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewOutfitRepository(db), db
}

func seedUserForOutfit(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (?, ?, 'hash', 'member', '2025-01-01T00:00:00Z')`,
		id, id+"@example.com")
	require.NoError(t, err)
}

func seedItemForOutfit(t *testing.T, db *sql.DB, itemID, ownerID string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES (?, ?, 'Test Item', '2025-01-01T00:00:00Z', '{}')`,
		itemID, ownerID)
	require.NoError(t, err)
}

func seedOutfit(t *testing.T, db *sql.DB, id, ownerID string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at)
		VALUES (?, ?, '2025-01-01T00:00:00Z')`,
		id, ownerID)
	require.NoError(t, err)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestOutfitRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "outfit-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitRepositoryGetShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newOutfitRepo(t)

	_, err := repo.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitRepositoryGetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitRepository(db)
	db.Close()

	_, err := repo.Get(t.Context(), "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitRepositoryGetShouldReturnOutfitWithItemsAndPhotosWhenRowExists(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-get")
	seedItemForOutfit(t, db, "item-get-1", "user-get")
	seedOutfit(t, db, "outfit-get-1", "user-get")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_items (outfit_id, item_id, position) VALUES ('outfit-get-1', 'item-get-1', 0)`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_photos (id, outfit_id, media_key, position, created_at)
		VALUES ('photo-get-1', 'outfit-get-1', 'key.jpg', 0, '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	outfit, err := repo.Get(t.Context(), "outfit-get-1")
	require.NoError(t, err)
	require.Equal(t, "outfit-get-1", outfit.GetID())
	require.Equal(t, "user-get", outfit.OwnerID)
	require.Len(t, outfit.Items, 1)
	require.Equal(t, "item-get-1", outfit.Items[0].ItemID)
	require.Len(t, outfit.Photos, 1)
	require.Equal(t, "key.jpg", outfit.Photos[0].MediaKey)
}

func TestOutfitRepositoryGetShouldReturnNilNameAndNotesWhenNotSet(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-nil")
	seedOutfit(t, db, "outfit-nil-1", "user-nil")

	outfit, err := repo.Get(t.Context(), "outfit-nil-1")
	require.NoError(t, err)
	require.Nil(t, outfit.Name)
	require.Nil(t, outfit.Notes)
}

func TestOutfitRepositoryGetShouldReturnNameAndNotesWhenSet(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-named")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, name, notes, created_at)
		VALUES ('outfit-named-1', 'user-named', 'Summer', 'Cool outfit', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	outfit, err := repo.Get(t.Context(), "outfit-named-1")
	require.NoError(t, err)
	require.NotNil(t, outfit.Name)
	require.Equal(t, "Summer", *outfit.Name)
	require.NotNil(t, outfit.Notes)
	require.Equal(t, "Cool outfit", *outfit.Notes)
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestOutfitRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var o domain.Outfit
	o.ID = "outfit-1"
	err := repo.Save(ctx, o)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitRepository(db)
	db.Close()

	var o domain.Outfit
	o.ID = "outfit-1"
	err := repo.Save(t.Context(), o)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitRepositorySaveShouldPersistNewOutfit(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-save")

	name := "Casual"
	notes := "For weekends"
	var o domain.Outfit
	o.ID = "outfit-save-1"
	o.OwnerID = "user-save"
	o.Name = &name
	o.Notes = &notes
	o.CreatedAt = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Save(t.Context(), o))

	got, err := repo.Get(t.Context(), "outfit-save-1")
	require.NoError(t, err)
	require.Equal(t, "outfit-save-1", got.GetID())
	require.Equal(t, "user-save", got.OwnerID)
	require.NotNil(t, got.Name)
	require.Equal(t, "Casual", *got.Name)
	require.NotNil(t, got.Notes)
	require.Equal(t, "For weekends", *got.Notes)
	require.Equal(t, time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC), got.CreatedAt)
}

func TestOutfitRepositorySaveShouldUpdateExistingOutfit(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-upd")

	var o domain.Outfit
	o.ID = "outfit-upd-1"
	o.OwnerID = "user-upd"
	o.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), o))

	newName := "Updated Name"
	o.Name = &newName
	require.NoError(t, repo.Save(t.Context(), o))

	got, err := repo.Get(t.Context(), "outfit-upd-1")
	require.NoError(t, err)
	require.NotNil(t, got.Name)
	require.Equal(t, "Updated Name", *got.Name)
	// created_at must not change
	require.Equal(t, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), got.CreatedAt)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestOutfitRepositoryDeleteShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.Delete(ctx, "outfit-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitRepositoryDeleteShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newOutfitRepo(t)

	err := repo.Delete(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitRepositoryDeleteShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitRepository(db)
	db.Close()

	err := repo.Delete(t.Context(), "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitRepositoryDeleteShouldRemoveOutfitWhenExists(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-del")
	seedOutfit(t, db, "outfit-del-1", "user-del")

	require.NoError(t, repo.Delete(t.Context(), "outfit-del-1"))

	_, err := repo.Get(t.Context(), "outfit-del-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitRepositoryDeleteShouldCascadeToItemsAndPhotos(t *testing.T) {
	_, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-cascade")
	seedItemForOutfit(t, db, "item-cascade-1", "user-cascade")
	seedOutfit(t, db, "outfit-cascade-1", "user-cascade")

	// Enable FK so cascade is enforced
	_, err := db.ExecContext(t.Context(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	repo := sqlstore.NewOutfitRepository(db)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_items (outfit_id, item_id, position) VALUES ('outfit-cascade-1', 'item-cascade-1', 0)`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_photos (id, outfit_id, media_key, position, created_at)
		VALUES ('photo-cascade-1', 'outfit-cascade-1', 'key.jpg', 0, '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(t.Context(), "outfit-cascade-1"))

	var count int
	row := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM outfit_items WHERE outfit_id = 'outfit-cascade-1'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count)

	row = db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM outfit_photos WHERE outfit_id = 'outfit-cascade-1'`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count)
}

// ── ListByOwner ───────────────────────────────────────────────────────────────

func TestOutfitRepositoryListByOwnerShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByOwner(ctx, "user-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitRepositoryListByOwnerShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitRepository(db)
	db.Close()

	_, err := repo.ListByOwner(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitRepositoryListByOwnerShouldReturnEmptyWhenOwnerHasNoOutfits(t *testing.T) {
	repo, _ := newOutfitRepo(t)

	outfits, err := repo.ListByOwner(t.Context(), "nonexistent-owner")
	require.NoError(t, err)
	require.Empty(t, outfits)
}

func TestOutfitRepositoryListByOwnerShouldReturnOnlyOwnerOutfits(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-list-a")
	seedUserForOutfit(t, db, "user-list-b")
	seedOutfit(t, db, "outfit-list-a1", "user-list-a")
	seedOutfit(t, db, "outfit-list-b1", "user-list-b")

	outfits, err := repo.ListByOwner(t.Context(), "user-list-a")
	require.NoError(t, err)
	require.Len(t, outfits, 1)
	require.Equal(t, "outfit-list-a1", outfits[0].GetID())
}

func TestOutfitRepositoryListByOwnerShouldBatchLoadItemsAndPhotos(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-batch")
	seedItemForOutfit(t, db, "item-batch-1", "user-batch")
	seedOutfit(t, db, "outfit-batch-1", "user-batch")
	seedOutfit(t, db, "outfit-batch-2", "user-batch")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_items (outfit_id, item_id, position) VALUES ('outfit-batch-1', 'item-batch-1', 0)`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_photos (id, outfit_id, media_key, position, created_at)
		VALUES ('photo-batch-1', 'outfit-batch-1', 'k.jpg', 0, '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	outfits, err := repo.ListByOwner(t.Context(), "user-batch")
	require.NoError(t, err)
	require.Len(t, outfits, 2)

	// find outfit-batch-1 in results
	var found domain.Outfit
	for _, o := range outfits {
		if o.GetID() == "outfit-batch-1" {
			found = o
		}
	}
	require.Equal(t, "outfit-batch-1", found.GetID())
	require.Len(t, found.Items, 1)
	require.Len(t, found.Photos, 1)
}

// ── SaveItem ──────────────────────────────────────────────────────────────────

func TestOutfitRepositorySaveItemShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.SaveItem(ctx, "outfit-1", "item-1", 0)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitRepositorySaveItemShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitRepository(db)
	db.Close()

	err := repo.SaveItem(t.Context(), "outfit-1", "item-1", 0)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitRepositorySaveItemShouldPersistAssociation(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-saveitem")
	seedItemForOutfit(t, db, "item-saveitem-1", "user-saveitem")
	seedOutfit(t, db, "outfit-saveitem-1", "user-saveitem")

	require.NoError(t, repo.SaveItem(t.Context(), "outfit-saveitem-1", "item-saveitem-1", 2))

	ids, err := repo.ListItemIDs(t.Context(), "outfit-saveitem-1")
	require.NoError(t, err)
	require.Equal(t, []string{"item-saveitem-1"}, ids)
}

func TestOutfitRepositorySaveItemShouldUpdatePositionWhenAssociationExists(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-saveitem-upd")
	seedItemForOutfit(t, db, "item-saveitem-upd", "user-saveitem-upd")
	seedOutfit(t, db, "outfit-saveitem-upd", "user-saveitem-upd")

	require.NoError(t, repo.SaveItem(t.Context(), "outfit-saveitem-upd", "item-saveitem-upd", 0))
	require.NoError(t, repo.SaveItem(t.Context(), "outfit-saveitem-upd", "item-saveitem-upd", 5))

	outfit, err := repo.Get(t.Context(), "outfit-saveitem-upd")
	require.NoError(t, err)
	require.Len(t, outfit.Items, 1)
	require.Equal(t, 5, outfit.Items[0].Position)
}

// ── DeleteItem ────────────────────────────────────────────────────────────────

func TestOutfitRepositoryDeleteItemShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.DeleteItem(ctx, "outfit-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitRepositoryDeleteItemShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitRepository(db)
	db.Close()

	err := repo.DeleteItem(t.Context(), "outfit-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitRepositoryDeleteItemShouldRemoveAssociation(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-delitem")
	seedItemForOutfit(t, db, "item-delitem-1", "user-delitem")
	seedOutfit(t, db, "outfit-delitem-1", "user-delitem")
	require.NoError(t, repo.SaveItem(t.Context(), "outfit-delitem-1", "item-delitem-1", 0))

	require.NoError(t, repo.DeleteItem(t.Context(), "outfit-delitem-1", "item-delitem-1"))

	ids, err := repo.ListItemIDs(t.Context(), "outfit-delitem-1")
	require.NoError(t, err)
	require.Empty(t, ids)
}

// ── ListItemIDs ───────────────────────────────────────────────────────────────

func TestOutfitRepositoryListItemIDsShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListItemIDs(ctx, "outfit-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitRepositoryListItemIDsShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitRepository(db)
	db.Close()

	_, err := repo.ListItemIDs(t.Context(), "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitRepositoryListItemIDsShouldReturnEmptyWhenNoItems(t *testing.T) {
	repo, _ := newOutfitRepo(t)

	ids, err := repo.ListItemIDs(t.Context(), "nonexistent-outfit")
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestOutfitRepositoryListItemIDsShouldReturnOrderedIDs(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-listids")
	seedItemForOutfit(t, db, "item-listids-a", "user-listids")
	seedItemForOutfit(t, db, "item-listids-b", "user-listids")
	seedItemForOutfit(t, db, "item-listids-c", "user-listids")
	seedOutfit(t, db, "outfit-listids-1", "user-listids")

	require.NoError(t, repo.SaveItem(t.Context(), "outfit-listids-1", "item-listids-b", 1))
	require.NoError(t, repo.SaveItem(t.Context(), "outfit-listids-1", "item-listids-a", 0))
	require.NoError(t, repo.SaveItem(t.Context(), "outfit-listids-1", "item-listids-c", 2))

	ids, err := repo.ListItemIDs(t.Context(), "outfit-listids-1")
	require.NoError(t, err)
	require.Equal(t, []string{"item-listids-a", "item-listids-b", "item-listids-c"}, ids)
}

// ── SavePhoto ─────────────────────────────────────────────────────────────────

func TestOutfitRepositorySavePhotoShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.SavePhoto(ctx, "outfit-1", "photo-1", "key.jpg", 0)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitRepositorySavePhotoShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitRepository(db)
	db.Close()

	err := repo.SavePhoto(t.Context(), "outfit-1", "photo-1", "key.jpg", 0)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitRepositorySavePhotoShouldPersistPhoto(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-savephoto")
	seedOutfit(t, db, "outfit-savephoto-1", "user-savephoto")

	require.NoError(t, repo.SavePhoto(t.Context(), "outfit-savephoto-1", "photo-id-1", "key.jpg", 0))

	outfit, err := repo.Get(t.Context(), "outfit-savephoto-1")
	require.NoError(t, err)
	require.Len(t, outfit.Photos, 1)
	require.Equal(t, "photo-id-1", outfit.Photos[0].ID)
	require.Equal(t, "key.jpg", outfit.Photos[0].MediaKey)
	require.Equal(t, 0, outfit.Photos[0].Position)
}

func TestOutfitRepositorySavePhotoShouldUpdatePositionWhenPhotoExists(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-savephoto-upd")
	seedOutfit(t, db, "outfit-savephoto-upd", "user-savephoto-upd")

	require.NoError(t, repo.SavePhoto(t.Context(), "outfit-savephoto-upd", "photo-upd-1", "key.jpg", 0))
	require.NoError(t, repo.SavePhoto(t.Context(), "outfit-savephoto-upd", "photo-upd-1", "key.jpg", 3))

	outfit, err := repo.Get(t.Context(), "outfit-savephoto-upd")
	require.NoError(t, err)
	require.Len(t, outfit.Photos, 1)
	require.Equal(t, 3, outfit.Photos[0].Position)
}

// ── DeletePhoto ───────────────────────────────────────────────────────────────

func TestOutfitRepositoryDeletePhotoShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newOutfitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.DeletePhoto(ctx, "outfit-1", "key.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitRepositoryDeletePhotoShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewOutfitRepository(db)
	db.Close()

	err := repo.DeletePhoto(t.Context(), "outfit-1", "key.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitRepositoryDeletePhotoShouldRemovePhoto(t *testing.T) {
	repo, db := newOutfitRepo(t)
	seedUserForOutfit(t, db, "user-delphoto")
	seedOutfit(t, db, "outfit-delphoto-1", "user-delphoto")
	require.NoError(t, repo.SavePhoto(t.Context(), "outfit-delphoto-1", "photo-del-1", "key.jpg", 0))

	require.NoError(t, repo.DeletePhoto(t.Context(), "outfit-delphoto-1", "key.jpg"))

	outfit, err := repo.Get(t.Context(), "outfit-delphoto-1")
	require.NoError(t, err)
	require.Empty(t, outfit.Photos)
}
