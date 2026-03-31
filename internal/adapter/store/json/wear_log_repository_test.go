package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/adapter/store/json"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewWearLogRepositoryShouldImplementWearLogRepository(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	require.Implements(t, (*ports.WearLogRepository)(nil), r)
}

func TestGetShouldReturnNotFoundWhenWearLogDoesNotExist(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())

	_, err := r.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetShouldReturnErrorWhenContextIsCancelledForWearLog(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Get(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnErrorWhenContextIsCancelledForWearLog(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Delete(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnNotFoundWhenWearLogDoesNotExist(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())

	err := r.Delete(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteShouldRemoveWearLogWhenFound(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	var wl domain.WearLog
	wl.ID = "42"
	require.NoError(t, r.Save(t.Context(), wl))

	require.NoError(t, r.Delete(t.Context(), "42"))

	_, err := r.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestListByItemShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListByItem(ctx, "item1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestListByItemShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wear_logs.json"), []byte("not json"), 0o644))
	r := json.NewWearLogRepository(dir)

	_, err := r.ListByItem(t.Context(), "item1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListByItemShouldReturnEmptyWhenNoLogsExist(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())

	logs, err := r.ListByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.Empty(t, logs)
}

func TestListByItemShouldReturnOnlyLogsForItem(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	var wl1, wl2, wl3 domain.WearLog
	wl1.ID = "1"
	wl1.ItemID = "item1"
	wl2.ID = "2"
	wl2.ItemID = "item1"
	wl3.ID = "3"
	wl3.ItemID = "item2"
	require.NoError(t, r.Save(t.Context(), wl1))
	require.NoError(t, r.Save(t.Context(), wl2))
	require.NoError(t, r.Save(t.Context(), wl3))

	logs, err := r.ListByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.Len(t, logs, 2)
}

func TestListByItemShouldReturnLogsSortedByWornOnDesc(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var wl1, wl2, wl3 domain.WearLog
	wl1.ID = "1"
	wl1.ItemID = "item1"
	wl1.WornOn = base
	wl2.ID = "2"
	wl2.ItemID = "item1"
	wl2.WornOn = base.Add(48 * time.Hour)
	wl3.ID = "3"
	wl3.ItemID = "item1"
	wl3.WornOn = base.Add(24 * time.Hour)
	require.NoError(t, r.Save(t.Context(), wl1))
	require.NoError(t, r.Save(t.Context(), wl2))
	require.NoError(t, r.Save(t.Context(), wl3))

	logs, err := r.ListByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.Len(t, logs, 3)
	require.Equal(t, "2", logs[0].ID)
	require.Equal(t, "3", logs[1].ID)
	require.Equal(t, "1", logs[2].ID)
}

func TestLatestByItemShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.LatestByItem(ctx, "item1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLatestByItemShouldReturnNilWhenNoLogsExist(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())

	got, err := r.LatestByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestLatestByItemShouldReturnMostRecentLog(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var wl1, wl2 domain.WearLog
	wl1.ID = "1"
	wl1.ItemID = "item1"
	wl1.WornOn = base
	wl2.ID = "2"
	wl2.ItemID = "item1"
	wl2.WornOn = base.Add(24 * time.Hour)
	require.NoError(t, r.Save(t.Context(), wl1))
	require.NoError(t, r.Save(t.Context(), wl2))

	got, err := r.LatestByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "2", got.ID)
}

func TestCountByItemShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.CountByItem(ctx, "item1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestCountByItemShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wear_logs.json"), []byte("not json"), 0o644))
	r := json.NewWearLogRepository(dir)

	_, err := r.CountByItem(t.Context(), "item1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestCountByItemShouldReturnZeroWhenNoLogsExist(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())

	count, err := r.CountByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestCountByItemShouldReturnCountForItem(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	var wl1, wl2, wl3 domain.WearLog
	wl1.ID = "1"
	wl1.ItemID = "item1"
	wl2.ID = "2"
	wl2.ItemID = "item1"
	wl3.ID = "3"
	wl3.ItemID = "item2"
	require.NoError(t, r.Save(t.Context(), wl1))
	require.NoError(t, r.Save(t.Context(), wl2))
	require.NoError(t, r.Save(t.Context(), wl3))

	count, err := r.CountByItem(t.Context(), "item1")
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestSaveShouldReturnErrorWhenContextIsCancelledForWearLog(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.WearLog{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetShouldReturnWearLogWhenFound(t *testing.T) {
	r := json.NewWearLogRepository(t.TempDir())
	var wl domain.WearLog
	wl.ID = "42"
	wl.ItemID = "item1"
	wl.OwnerID = "owner1"
	require.NoError(t, r.Save(t.Context(), wl))

	got, err := r.Get(t.Context(), "42")
	require.NoError(t, err)
	require.Equal(t, wl, got)
}
