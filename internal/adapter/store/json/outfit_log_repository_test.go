package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewOutfitLogRepositoryShouldImplementOutfitLogRepository(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	require.NotNil(t, r)
}

func TestOutfitLogGetShouldReturnNotFoundWhenDoesNotExist(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())

	_, err := r.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogGetShouldReturnErrorWhenContextCancelled(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Get(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogSaveShouldReturnErrorWhenContextCancelled(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.OutfitLog{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogDeleteShouldReturnErrorWhenContextCancelled(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Delete(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogDeleteShouldReturnNotFoundWhenDoesNotExist(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())

	err := r.Delete(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogDeleteShouldRemoveLog(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	var ol domain.OutfitLog
	ol.ID = "42"
	require.NoError(t, r.Save(t.Context(), ol))

	require.NoError(t, r.Delete(t.Context(), "42"))

	_, err := r.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogListByOutfitShouldReturnErrorWhenContextCancelled(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListByOutfit(ctx, "outfit1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogListByOutfitShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "outfit_logs.json"), []byte("not json"), 0o644))
	r := json.NewOutfitLogRepository(dir)

	_, err := r.ListByOutfit(t.Context(), "outfit1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogListByOutfitShouldReturnEmptyWhenNoneExist(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())

	logs, err := r.ListByOutfit(t.Context(), "outfit1")
	require.NoError(t, err)
	require.Empty(t, logs)
}

func TestOutfitLogListByOutfitShouldReturnOnlyLogsForOutfit(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ol1 := domain.OutfitLog{}
	ol1.ID = "1"
	ol1.OutfitID = "outfit1"
	ol2 := domain.OutfitLog{}
	ol2.ID = "2"
	ol2.OutfitID = "outfit1"
	ol3 := domain.OutfitLog{}
	ol3.ID = "3"
	ol3.OutfitID = "outfit2"
	require.NoError(t, r.Save(t.Context(), ol1))
	require.NoError(t, r.Save(t.Context(), ol2))
	require.NoError(t, r.Save(t.Context(), ol3))

	logs, err := r.ListByOutfit(t.Context(), "outfit1")
	require.NoError(t, err)
	require.Len(t, logs, 2)
}

func TestOutfitLogListByOutfitShouldReturnLogsSortedByWornOnDesc(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ol1 := domain.OutfitLog{}
	ol1.ID = "1"
	ol1.OutfitID = "outfit1"
	ol1.WornOn = base
	ol2 := domain.OutfitLog{}
	ol2.ID = "2"
	ol2.OutfitID = "outfit1"
	ol2.WornOn = base.Add(48 * time.Hour)
	ol3 := domain.OutfitLog{}
	ol3.ID = "3"
	ol3.OutfitID = "outfit1"
	ol3.WornOn = base.Add(24 * time.Hour)
	require.NoError(t, r.Save(t.Context(), ol1))
	require.NoError(t, r.Save(t.Context(), ol2))
	require.NoError(t, r.Save(t.Context(), ol3))

	logs, err := r.ListByOutfit(t.Context(), "outfit1")
	require.NoError(t, err)
	require.Len(t, logs, 3)
	require.Equal(t, "2", logs[0].ID)
	require.Equal(t, "3", logs[1].ID)
	require.Equal(t, "1", logs[2].ID)
}

func TestOutfitLogListByOwnerDateRangeShouldReturnErrorWhenContextCancelled(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := r.ListByOwnerDateRange(ctx, "owner1", base, base.Add(24*time.Hour))
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogListByOwnerDateRangeShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "outfit_logs.json"), []byte("not json"), 0o644))
	r := json.NewOutfitLogRepository(dir)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := r.ListByOwnerDateRange(t.Context(), "owner1", base, base.Add(24*time.Hour))
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogListByOwnerDateRangeShouldReturnEmptyWhenNoneExist(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	logs, err := r.ListByOwnerDateRange(t.Context(), "owner1", base, base.Add(24*time.Hour))
	require.NoError(t, err)
	require.Empty(t, logs)
}

func TestOutfitLogListByOwnerDateRangeShouldFilterByOwnerAndDateRange(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	base := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	// in range, correct owner
	ol1 := domain.OutfitLog{}
	ol1.ID = "1"
	ol1.OwnerID = "owner1"
	ol1.WornOn = base
	// in range, wrong owner
	ol2 := domain.OutfitLog{}
	ol2.ID = "2"
	ol2.OwnerID = "owner2"
	ol2.WornOn = base
	// out of range, correct owner
	ol3 := domain.OutfitLog{}
	ol3.ID = "3"
	ol3.OwnerID = "owner1"
	ol3.WornOn = base.Add(-48 * time.Hour)
	// in range, correct owner
	ol4 := domain.OutfitLog{}
	ol4.ID = "4"
	ol4.OwnerID = "owner1"
	ol4.WornOn = base.Add(24 * time.Hour)
	require.NoError(t, r.Save(t.Context(), ol1))
	require.NoError(t, r.Save(t.Context(), ol2))
	require.NoError(t, r.Save(t.Context(), ol3))
	require.NoError(t, r.Save(t.Context(), ol4))

	logs, err := r.ListByOwnerDateRange(t.Context(), "owner1", base, base.Add(48*time.Hour))
	require.NoError(t, err)
	require.Len(t, logs, 2)
}

func TestOutfitLogListByOwnerDateRangeShouldReturnLogsSortedByWornOnAsc(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	base := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	ol1 := domain.OutfitLog{}
	ol1.ID = "1"
	ol1.OwnerID = "owner1"
	ol1.WornOn = base
	ol2 := domain.OutfitLog{}
	ol2.ID = "2"
	ol2.OwnerID = "owner1"
	ol2.WornOn = base.Add(48 * time.Hour)
	ol3 := domain.OutfitLog{}
	ol3.ID = "3"
	ol3.OwnerID = "owner1"
	ol3.WornOn = base.Add(24 * time.Hour)
	require.NoError(t, r.Save(t.Context(), ol1))
	require.NoError(t, r.Save(t.Context(), ol2))
	require.NoError(t, r.Save(t.Context(), ol3))

	logs, err := r.ListByOwnerDateRange(t.Context(), "owner1", base, base.Add(72*time.Hour))
	require.NoError(t, err)
	require.Len(t, logs, 3)
	require.Equal(t, "1", logs[0].ID)
	require.Equal(t, "3", logs[1].ID)
	require.Equal(t, "2", logs[2].ID)
}

func TestOutfitLogListByOwnerDateRangeShouldIncludeBoundaryDates(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	base := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	ol1 := domain.OutfitLog{}
	ol1.ID = "1"
	ol1.OwnerID = "owner1"
	ol1.WornOn = base // exactly at from
	ol2 := domain.OutfitLog{}
	ol2.ID = "2"
	ol2.OwnerID = "owner1"
	ol2.WornOn = base.Add(48 * time.Hour) // exactly at to
	require.NoError(t, r.Save(t.Context(), ol1))
	require.NoError(t, r.Save(t.Context(), ol2))

	logs, err := r.ListByOwnerDateRange(t.Context(), "owner1", base, base.Add(48*time.Hour))
	require.NoError(t, err)
	require.Len(t, logs, 2)
}

func TestOutfitLogLinkWearLogShouldReturnErrorWhenContextCancelled(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.LinkWearLog(ctx, "ol1", "wl1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogLinkWearLogShouldReturnNotFoundWhenOutfitLogDoesNotExist(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())

	err := r.LinkWearLog(t.Context(), "ol1", "wl1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogLinkWearLogShouldAppendWearLogID(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ol := domain.OutfitLog{}
	ol.ID = "ol1"
	require.NoError(t, r.Save(t.Context(), ol))

	require.NoError(t, r.LinkWearLog(t.Context(), "ol1", "wl1"))

	got, err := r.Get(t.Context(), "ol1")
	require.NoError(t, err)
	require.Equal(t, []string{"wl1"}, got.WearLogIDs)
}

func TestOutfitLogLinkWearLogShouldAppendMultipleWearLogIDs(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ol := domain.OutfitLog{}
	ol.ID = "ol1"
	require.NoError(t, r.Save(t.Context(), ol))

	require.NoError(t, r.LinkWearLog(t.Context(), "ol1", "wl1"))
	require.NoError(t, r.LinkWearLog(t.Context(), "ol1", "wl2"))

	got, err := r.Get(t.Context(), "ol1")
	require.NoError(t, err)
	require.Equal(t, []string{"wl1", "wl2"}, got.WearLogIDs)
}

func TestOutfitLogLinkedWearLogIDsShouldReturnErrorWhenContextCancelled(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.LinkedWearLogIDs(ctx, "ol1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogLinkedWearLogIDsShouldReturnNotFoundWhenOutfitLogDoesNotExist(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())

	_, err := r.LinkedWearLogIDs(t.Context(), "ol1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogLinkedWearLogIDsShouldReturnEmptyWhenNoLinksExist(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ol := domain.OutfitLog{}
	ol.ID = "ol1"
	require.NoError(t, r.Save(t.Context(), ol))

	ids, err := r.LinkedWearLogIDs(t.Context(), "ol1")
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestOutfitLogLinkedWearLogIDsShouldReturnLinkedIDs(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	ol := domain.OutfitLog{}
	ol.ID = "ol1"
	require.NoError(t, r.Save(t.Context(), ol))
	require.NoError(t, r.LinkWearLog(t.Context(), "ol1", "wl1"))
	require.NoError(t, r.LinkWearLog(t.Context(), "ol1", "wl2"))

	ids, err := r.LinkedWearLogIDs(t.Context(), "ol1")
	require.NoError(t, err)
	require.Equal(t, []string{"wl1", "wl2"}, ids)
}

func TestOutfitLogGetShouldReturnSavedLog(t *testing.T) {
	r := json.NewOutfitLogRepository(t.TempDir())
	var ol domain.OutfitLog
	ol.ID = "42"
	ol.OutfitID = "outfit1"
	ol.OwnerID = "owner1"
	require.NoError(t, r.Save(t.Context(), ol))

	got, err := r.Get(t.Context(), "42")
	require.NoError(t, err)
	require.Equal(t, ol, got)
}
