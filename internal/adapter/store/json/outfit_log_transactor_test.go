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

func newTransactor(t *testing.T) *json.OutfitLogTransactor {
	t.Helper()
	dir := t.TempDir()
	return json.NewOutfitLogTransactor(
		json.NewOutfitLogRepository(dir),
		json.NewWearLogRepository(dir),
	)
}

// --- CreateOutfitLog ---

func TestCreateOutfitLogShouldReturnErrorWhenContextCancelled(t *testing.T) {
	tr := newTransactor(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := tr.CreateOutfitLog(ctx, domain.OutfitLog{}, nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestCreateOutfitLogShouldReturnIOErrorWhenOutfitLogStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "outfit_logs.json"), []byte("not json"), 0o644))
	tr := json.NewOutfitLogTransactor(
		json.NewOutfitLogRepository(dir),
		json.NewWearLogRepository(dir),
	)
	var ol domain.OutfitLog
	ol.ID = "ol1"

	_, err := tr.CreateOutfitLog(t.Context(), ol, nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestCreateOutfitLogShouldReturnIOErrorWhenWearLogStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wear_logs.json"), []byte("not json"), 0o644))
	tr := json.NewOutfitLogTransactor(
		json.NewOutfitLogRepository(dir),
		json.NewWearLogRepository(dir),
	)
	var ol domain.OutfitLog
	ol.ID = "ol1"
	var wl domain.WearLog
	wl.ID = "wl1"

	_, err := tr.CreateOutfitLog(t.Context(), ol, []domain.WearLog{wl})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestCreateOutfitLogShouldSaveOutfitLogAndWearLogsAndReturnWithWearLogIDs(t *testing.T) {
	dir := t.TempDir()
	olRepo := json.NewOutfitLogRepository(dir)
	wlRepo := json.NewWearLogRepository(dir)
	tr := json.NewOutfitLogTransactor(olRepo, wlRepo)

	var ol domain.OutfitLog
	ol.ID = "ol1"
	ol.OutfitID = "outfit1"
	var wl1, wl2 domain.WearLog
	wl1.ID = "wl1"
	wl2.ID = "wl2"

	got, err := tr.CreateOutfitLog(t.Context(), ol, []domain.WearLog{wl1, wl2})
	require.NoError(t, err)
	require.Equal(t, "ol1", got.ID)
	require.Equal(t, []string{"wl1", "wl2"}, got.WearLogIDs)

	// Verify outfit log persisted
	storedOL, err := olRepo.Get(t.Context(), "ol1")
	require.NoError(t, err)
	require.Equal(t, []string{"wl1", "wl2"}, storedOL.WearLogIDs)

	// Verify wear logs persisted
	_, err = wlRepo.Get(t.Context(), "wl1")
	require.NoError(t, err)
	_, err = wlRepo.Get(t.Context(), "wl2")
	require.NoError(t, err)
}

// --- DeleteOutfitLog ---

func TestDeleteOutfitLogShouldReturnErrorWhenContextCancelled(t *testing.T) {
	tr := newTransactor(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := tr.DeleteOutfitLog(ctx, "ol1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteOutfitLogShouldReturnNotFoundWhenOutfitLogDoesNotExist(t *testing.T) {
	tr := newTransactor(t)

	err := tr.DeleteOutfitLog(t.Context(), "ol1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteOutfitLogShouldReturnNotFoundWhenLinkedWearLogDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	olRepo := json.NewOutfitLogRepository(dir)
	tr := json.NewOutfitLogTransactor(olRepo, json.NewWearLogRepository(dir))

	var ol domain.OutfitLog
	ol.ID = "ol1"
	require.NoError(t, olRepo.Save(t.Context(), ol))
	// Bypass the transactor to simulate a data-inconsistency state: the outfit
	// log references a wear log ID that has no matching wear log record.
	require.NoError(t, olRepo.LinkWearLog(t.Context(), "ol1", "ghost-wl"))

	err := tr.DeleteOutfitLog(t.Context(), "ol1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteOutfitLogShouldDeleteOutfitLogAndLinkedWearLogs(t *testing.T) {
	dir := t.TempDir()
	olRepo := json.NewOutfitLogRepository(dir)
	wlRepo := json.NewWearLogRepository(dir)
	tr := json.NewOutfitLogTransactor(olRepo, wlRepo)

	var ol domain.OutfitLog
	ol.ID = "ol1"
	var wl1, wl2 domain.WearLog
	wl1.ID = "wl1"
	wl2.ID = "wl2"
	_, err := tr.CreateOutfitLog(t.Context(), ol, []domain.WearLog{wl1, wl2})
	require.NoError(t, err)

	require.NoError(t, tr.DeleteOutfitLog(t.Context(), "ol1"))

	_, err = olRepo.Get(t.Context(), "ol1")
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = wlRepo.Get(t.Context(), "wl1")
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = wlRepo.Get(t.Context(), "wl2")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// --- UpdateOutfitLogDate ---

func TestUpdateOutfitLogDateShouldReturnErrorWhenContextCancelled(t *testing.T) {
	tr := newTransactor(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := tr.UpdateOutfitLogDate(ctx, "ol1", time.Now())
	require.ErrorIs(t, err, context.Canceled)
}

func TestUpdateOutfitLogDateShouldReturnNotFoundWhenOutfitLogDoesNotExist(t *testing.T) {
	tr := newTransactor(t)

	err := tr.UpdateOutfitLogDate(t.Context(), "ol1", time.Now())
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdateOutfitLogDateShouldReturnNotFoundWhenLinkedWearLogDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	olRepo := json.NewOutfitLogRepository(dir)
	tr := json.NewOutfitLogTransactor(olRepo, json.NewWearLogRepository(dir))

	var ol domain.OutfitLog
	ol.ID = "ol1"
	require.NoError(t, olRepo.Save(t.Context(), ol))
	// Bypass the transactor to simulate a data-inconsistency state: the outfit
	// log references a wear log ID that has no matching wear log record.
	require.NoError(t, olRepo.LinkWearLog(t.Context(), "ol1", "ghost-wl"))

	err := tr.UpdateOutfitLogDate(t.Context(), "ol1", time.Now())
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdateOutfitLogDateShouldReturnIOErrorWhenOutfitLogStorageIsNotWritable(t *testing.T) {
	dir := t.TempDir()
	olRepo := json.NewOutfitLogRepository(dir)
	tr := json.NewOutfitLogTransactor(olRepo, json.NewWearLogRepository(dir))

	var ol domain.OutfitLog
	ol.ID = "ol1"
	require.NoError(t, olRepo.Save(t.Context(), ol))
	require.NoError(t, os.Chmod(filepath.Join(dir, "outfit_logs.json"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "outfit_logs.json"), 0o644) })

	err := tr.UpdateOutfitLogDate(t.Context(), "ol1", time.Now())
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUpdateOutfitLogDateShouldUpdateOutfitLogAndLinkedWearLogDates(t *testing.T) {
	dir := t.TempDir()
	olRepo := json.NewOutfitLogRepository(dir)
	wlRepo := json.NewWearLogRepository(dir)
	tr := json.NewOutfitLogTransactor(olRepo, wlRepo)

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	var ol domain.OutfitLog
	ol.ID = "ol1"
	ol.WornOn = base
	var wl1, wl2 domain.WearLog
	wl1.ID = "wl1"
	wl1.WornOn = base
	wl2.ID = "wl2"
	wl2.WornOn = base
	_, err := tr.CreateOutfitLog(t.Context(), ol, []domain.WearLog{wl1, wl2})
	require.NoError(t, err)

	require.NoError(t, tr.UpdateOutfitLogDate(t.Context(), "ol1", newDate))

	storedOL, err := olRepo.Get(t.Context(), "ol1")
	require.NoError(t, err)
	require.Equal(t, newDate, storedOL.WornOn)

	storedWL1, err := wlRepo.Get(t.Context(), "wl1")
	require.NoError(t, err)
	require.Equal(t, newDate, storedWL1.WornOn)

	storedWL2, err := wlRepo.Get(t.Context(), "wl2")
	require.NoError(t, err)
	require.Equal(t, newDate, storedWL2.WornOn)
}
