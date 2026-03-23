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

func newLocationRepo(t *testing.T) (*sqlstore.LocationRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewLocationRepository(db), db
}

func seedUserForLocation(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (?, ?, 'hash', 'member', '2025-01-01T00:00:00Z')`,
		id, id+"@example.com")
	require.NoError(t, err)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestLocationRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newLocationRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "loc-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationRepositoryGetShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newLocationRepo(t)

	_, err := repo.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationRepositoryGetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewLocationRepository(db)
	db.Close()

	_, err := repo.Get(t.Context(), "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationRepositoryGetShouldReturnLocationWhenRowExists(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-get")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO locations (id, owner_id, label, created_at)
		VALUES ('loc-get-1', 'user-get', 'Closet', '2025-06-01T10:00:00Z')`)
	require.NoError(t, err)

	loc, err := repo.Get(t.Context(), "loc-get-1")
	require.NoError(t, err)
	require.Equal(t, "loc-get-1", loc.GetID())
	require.Equal(t, "user-get", loc.OwnerID)
	require.Equal(t, "Closet", loc.Label)
	require.Nil(t, loc.ParentID)
	require.Equal(t, time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC), loc.CreatedAt)
}

func TestLocationRepositoryGetShouldReturnLocationWithParentIDWhenSet(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-parent")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO locations (id, owner_id, label, created_at)
		VALUES ('loc-parent', 'user-parent', 'Room', '2025-06-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO locations (id, owner_id, parent_id, label, created_at)
		VALUES ('loc-child', 'user-parent', 'loc-parent', 'Shelf', '2025-06-02T00:00:00Z')`)
	require.NoError(t, err)

	loc, err := repo.Get(t.Context(), "loc-child")
	require.NoError(t, err)
	require.NotNil(t, loc.ParentID)
	require.Equal(t, "loc-parent", *loc.ParentID)
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestLocationRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newLocationRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var loc domain.Location
	loc.ID = "loc-1"
	err := repo.Save(ctx, loc)
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewLocationRepository(db)
	db.Close()

	var loc domain.Location
	loc.ID = "loc-1"
	err := repo.Save(t.Context(), loc)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationRepositorySaveShouldPersistNewLocation(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-save")

	var loc domain.Location
	loc.ID = "loc-save-1"
	loc.OwnerID = "user-save"
	loc.Label = "Wardrobe"
	loc.CreatedAt = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Save(t.Context(), loc))

	got, err := repo.Get(t.Context(), "loc-save-1")
	require.NoError(t, err)
	require.Equal(t, "loc-save-1", got.GetID())
	require.Equal(t, "Wardrobe", got.Label)
	require.Nil(t, got.ParentID)
}

func TestLocationRepositorySaveShouldUpdateExistingLocation(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-upd")

	var loc domain.Location
	loc.ID = "loc-upd-1"
	loc.OwnerID = "user-upd"
	loc.Label = "Old Label"
	loc.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Save(t.Context(), loc))

	loc.Label = "New Label"
	loc.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // different from seeded value
	require.NoError(t, repo.Save(t.Context(), loc))

	got, err := repo.Get(t.Context(), "loc-upd-1")
	require.NoError(t, err)
	require.Equal(t, "New Label", got.Label)
	require.Equal(t, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), got.CreatedAt) // must not change
}

func TestLocationRepositorySaveShouldNotCascadeToChildrenWithFKEnforced(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	repo := sqlstore.NewLocationRepository(db)
	seedUserForLocation(t, db, "user-fk")

	var parent domain.Location
	parent.ID = "loc-fk-parent"
	parent.OwnerID = "user-fk"
	parent.Label = "Room"
	parent.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), parent))

	parentID := "loc-fk-parent"
	var child domain.Location
	child.ID = "loc-fk-child"
	child.OwnerID = "user-fk"
	child.ParentID = &parentID
	child.Label = "Shelf"
	child.CreatedAt = time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), child))

	parent.Label = "Room Updated"
	require.NoError(t, repo.Save(t.Context(), parent))

	got, err := repo.Get(t.Context(), "loc-fk-parent")
	require.NoError(t, err)
	require.Equal(t, "Room Updated", got.Label)

	gotChild, err := repo.Get(t.Context(), "loc-fk-child")
	require.NoError(t, err)
	require.Equal(t, "loc-fk-child", gotChild.GetID(), "child must survive re-save of parent with FK enforcement on")
}

func TestLocationRepositorySaveShouldPersistParentID(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-pid")

	var parent domain.Location
	parent.ID = "loc-pid-parent"
	parent.OwnerID = "user-pid"
	parent.Label = "Room"
	parent.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), parent))

	parentID := "loc-pid-parent"
	var child domain.Location
	child.ID = "loc-pid-child"
	child.OwnerID = "user-pid"
	child.ParentID = &parentID
	child.Label = "Shelf"
	child.CreatedAt = time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), child))

	got, err := repo.Get(t.Context(), "loc-pid-child")
	require.NoError(t, err)
	require.NotNil(t, got.ParentID)
	require.Equal(t, "loc-pid-parent", *got.ParentID)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestLocationRepositoryDeleteShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newLocationRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.Delete(ctx, "loc-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationRepositoryDeleteShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newLocationRepo(t)

	err := repo.Delete(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationRepositoryDeleteShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewLocationRepository(db)
	db.Close()

	err := repo.Delete(t.Context(), "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationRepositoryDeleteShouldRemoveLocationWhenExists(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-del")

	var loc domain.Location
	loc.ID = "loc-del-1"
	loc.OwnerID = "user-del"
	loc.Label = "To Delete"
	loc.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), loc))

	require.NoError(t, repo.Delete(t.Context(), "loc-del-1"))

	_, err := repo.Get(t.Context(), "loc-del-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// ── ListByOwner ───────────────────────────────────────────────────────────────

func TestLocationRepositoryListByOwnerShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newLocationRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByOwner(ctx, "user-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationRepositoryListByOwnerShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewLocationRepository(db)
	db.Close()

	_, err := repo.ListByOwner(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationRepositoryListByOwnerShouldReturnEmptyWhenOwnerHasNoLocations(t *testing.T) {
	repo, _ := newLocationRepo(t)

	locs, err := repo.ListByOwner(t.Context(), "nonexistent-user")
	require.NoError(t, err)
	require.Empty(t, locs)
}

func TestLocationRepositoryListByOwnerShouldReturnOnlyOwnerLocations(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-a")
	seedUserForLocation(t, db, "user-b")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO locations (id, owner_id, label, created_at)
		VALUES
			('loc-a1', 'user-a', 'Closet A', '2025-01-01T00:00:00Z'),
			('loc-b1', 'user-b', 'Closet B', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	locs, err := repo.ListByOwner(t.Context(), "user-a")
	require.NoError(t, err)
	require.Len(t, locs, 1)
	require.Equal(t, "loc-a1", locs[0].GetID())
}

func TestLocationRepositoryListByOwnerShouldReturnAllOwnerLocations(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-list")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO locations (id, owner_id, label, created_at)
		VALUES
			('loc-list-1', 'user-list', 'Drawer', '2025-01-01T00:00:00Z'),
			('loc-list-2', 'user-list', 'Shelf',  '2025-01-02T00:00:00Z')`)
	require.NoError(t, err)

	locs, err := repo.ListByOwner(t.Context(), "user-list")
	require.NoError(t, err)
	require.Len(t, locs, 2)
}

// ── HasChildren ───────────────────────────────────────────────────────────────

func TestLocationRepositoryHasChildrenShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newLocationRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.HasChildren(ctx, "loc-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationRepositoryHasChildrenShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewLocationRepository(db)
	db.Close()

	_, err := repo.HasChildren(t.Context(), "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationRepositoryHasChildrenShouldReturnFalseWhenNoChildren(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-hc")

	var loc domain.Location
	loc.ID = "loc-hc-1"
	loc.OwnerID = "user-hc"
	loc.Label = "Alone"
	loc.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), loc))

	has, err := repo.HasChildren(t.Context(), "loc-hc-1")
	require.NoError(t, err)
	require.False(t, has)
}

func TestLocationRepositoryHasChildrenShouldReturnTrueWhenChildrenExist(t *testing.T) {
	repo, db := newLocationRepo(t)
	seedUserForLocation(t, db, "user-hc2")

	var parent domain.Location
	parent.ID = "loc-hc2-parent"
	parent.OwnerID = "user-hc2"
	parent.Label = "Parent"
	parent.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), parent))

	parentID := "loc-hc2-parent"
	var child domain.Location
	child.ID = "loc-hc2-child"
	child.OwnerID = "user-hc2"
	child.ParentID = &parentID
	child.Label = "Child"
	child.CreatedAt = time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(t.Context(), child))

	has, err := repo.HasChildren(t.Context(), "loc-hc2-parent")
	require.NoError(t, err)
	require.True(t, has)
}
