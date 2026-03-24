package sqlstore

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/outfitte/outfitte/internal/domain"
)

// ── batchLoadItems ────────────────────────────────────────────────────────────

func TestBatchLoadItemsShouldReturnErrIOWhenQueryFails(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.Close()

	repo := &OutfitRepository{db: db}
	var o domain.Outfit
	o.ID = "outfit-1"
	err = repo.batchLoadItems(t.Context(), []domain.Outfit{o})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestBatchLoadItemsShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)

	// Insert an outfit_item with non-integer position to cause scan failure.
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_items (outfit_id, item_id, position)
		VALUES ('outfit-scan', 'item-scan', 'not-an-int')`)
	require.NoError(t, err)

	repo := &OutfitRepository{db: db}
	var o domain.Outfit
	o.ID = "outfit-scan"
	err = repo.batchLoadItems(t.Context(), []domain.Outfit{o})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestBatchLoadItemsRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &OutfitRepository{db: db}
	var o domain.Outfit
	o.ID = "outfit-1"
	err := repo.batchLoadItems(t.Context(), []domain.Outfit{o})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── batchLoadOutfitPhotos ─────────────────────────────────────────────────────

func TestBatchLoadOutfitPhotosShouldReturnErrIOWhenQueryFails(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.Close()

	repo := &OutfitRepository{db: db}
	var o domain.Outfit
	o.ID = "outfit-1"
	err = repo.batchLoadOutfitPhotos(t.Context(), []domain.Outfit{o})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestBatchLoadOutfitPhotosShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_photos (id, outfit_id, media_key, position, created_at)
		VALUES ('ph-scan', 'outfit-scan', 'k.jpg', 'not-an-int', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	repo := &OutfitRepository{db: db}
	var o domain.Outfit
	o.ID = "outfit-scan"
	err = repo.batchLoadOutfitPhotos(t.Context(), []domain.Outfit{o})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestBatchLoadOutfitPhotosRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &OutfitRepository{db: db}
	var o domain.Outfit
	o.ID = "outfit-1"
	err := repo.batchLoadOutfitPhotos(t.Context(), []domain.Outfit{o})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── queryOutfitsByOwner ───────────────────────────────────────────────────────

func TestQueryOutfitsByOwnerRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &OutfitRepository{db: db}
	_, err := repo.queryOutfitsByOwner(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestQueryOutfitsByOwnerShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &OutfitRepository{db: db}
	_, err := repo.queryOutfitsByOwner(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── OutfitRepository.Delete RowsAffected error ────────────────────────────────

func TestOutfitRepositoryDeleteShouldReturnErrIOWhenRowsAffectedFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-aff-err")
	repo := &OutfitRepository{db: db}
	err := repo.Delete(t.Context(), "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── ListItemIDs scan error ────────────────────────────────────────────────────

func TestListItemIDsShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &OutfitRepository{db: db}
	_, err := repo.ListItemIDs(t.Context(), "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListItemIDsRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &OutfitRepository{db: db}
	_, err := repo.ListItemIDs(t.Context(), "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanOutfitRow time.Parse error ────────────────────────────────────────────

func TestGetShouldReturnErrIOWhenOutfitCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at)
		VALUES ('outfit-bad-ts', 'owner-x', 'not-a-timestamp')`)
	require.NoError(t, err)

	repo := &OutfitRepository{db: db}
	_, err = repo.Get(t.Context(), "outfit-bad-ts")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── Get batchLoadItems error path ─────────────────────────────────────────────

func TestGetShouldReturnErrIOWhenBatchLoadItemsFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at)
		VALUES ('outfit-items-fail', 'owner-x', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	// Insert outfit_item with invalid position to make batchLoadItems scan fail.
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_items (outfit_id, item_id, position)
		VALUES ('outfit-items-fail', 'item-x', 'not-an-int')`)
	require.NoError(t, err)

	repo := &OutfitRepository{db: db}
	_, err = repo.Get(t.Context(), "outfit-items-fail")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── Get batchLoadOutfitPhotos error path ──────────────────────────────────────

func TestGetShouldReturnErrIOWhenBatchLoadPhotosFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at)
		VALUES ('outfit-photos-fail', 'owner-x', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	// Insert outfit_photo with invalid position to make batchLoadOutfitPhotos scan fail.
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_photos (id, outfit_id, media_key, position, created_at)
		VALUES ('ph-fail', 'outfit-photos-fail', 'k.jpg', 'not-an-int', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	repo := &OutfitRepository{db: db}
	_, err = repo.Get(t.Context(), "outfit-photos-fail")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── batchLoadOutfitPhotos time.Parse error ────────────────────────────────────

func TestBatchLoadOutfitPhotosShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfit_photos (id, outfit_id, media_key, position, created_at)
		VALUES ('ph-bad-ts', 'outfit-x', 'k.jpg', 0, 'not-a-timestamp')`)
	require.NoError(t, err)

	repo := &OutfitRepository{db: db}
	var o domain.Outfit
	o.ID = "outfit-x"
	err = repo.batchLoadOutfitPhotos(t.Context(), []domain.Outfit{o})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── queryOutfitsByOwner time.Parse error ─────────────────────────────────────

func TestQueryOutfitsByOwnerShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at)
		VALUES ('outfit-bad-ts2', 'owner-y', 'not-a-timestamp')`)
	require.NoError(t, err)

	repo := &OutfitRepository{db: db}
	_, err = repo.queryOutfitsByOwner(t.Context(), "owner-y")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── ListByOwner batchLoadItems error path ─────────────────────────────────────

func TestListByOwnerShouldReturnErrIOWhenBatchLoadItemsFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at)
		VALUES ('outfit-listby-items', 'owner-lb', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_items (outfit_id, item_id, position)
		VALUES ('outfit-listby-items', 'item-lb', 'not-an-int')`)
	require.NoError(t, err)

	repo := &OutfitRepository{db: db}
	_, err = repo.ListByOwner(t.Context(), "owner-lb")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── ListByOwner batchLoadOutfitPhotos error path ──────────────────────────────

func TestListByOwnerShouldReturnErrIOWhenBatchLoadPhotosFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO outfits (id, owner_id, created_at)
		VALUES ('outfit-listby-photos', 'owner-lbp', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO outfit_photos (id, outfit_id, media_key, position, created_at)
		VALUES ('ph-lb', 'outfit-listby-photos', 'k.jpg', 'not-an-int', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	repo := &OutfitRepository{db: db}
	_, err = repo.ListByOwner(t.Context(), "owner-lbp")
	require.ErrorIs(t, err, domain.ErrIO)
}
