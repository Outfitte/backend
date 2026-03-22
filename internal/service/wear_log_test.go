package service

import (
	"context"
	"testing"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// mockWearLogRepo is an in-memory ports.WearLogRepository for tests.
type mockWearLogRepo struct {
	logs    []domain.WearLog
	getErr  error
	saveErr error
	delErr  error

	listByItemErr    error
	latestByItemErr  error
	latestByItemResult *domain.WearLog
	countByItemErr   error
	countByItemResult int
}

func (m *mockWearLogRepo) Get(_ context.Context, id string) (domain.WearLog, error) {
	if m.getErr != nil {
		return domain.WearLog{}, m.getErr
	}
	for _, l := range m.logs {
		if l.GetID() == id {
			return l, nil
		}
	}
	return domain.WearLog{}, domain.ErrNotFound
}

func (m *mockWearLogRepo) Save(_ context.Context, log domain.WearLog) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	for i, existing := range m.logs {
		if existing.GetID() == log.GetID() {
			m.logs[i] = log
			return nil
		}
	}
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockWearLogRepo) Delete(_ context.Context, id string) error {
	if m.delErr != nil {
		return m.delErr
	}
	for i, l := range m.logs {
		if l.GetID() == id {
			m.logs = append(m.logs[:i], m.logs[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockWearLogRepo) ListByItem(_ context.Context, itemID string) ([]domain.WearLog, error) {
	if m.listByItemErr != nil {
		return nil, m.listByItemErr
	}
	var result []domain.WearLog
	for _, l := range m.logs {
		if l.ItemID == itemID {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *mockWearLogRepo) LatestByItem(_ context.Context, _ string) (*domain.WearLog, error) {
	if m.latestByItemErr != nil {
		return nil, m.latestByItemErr
	}
	return m.latestByItemResult, nil
}

func (m *mockWearLogRepo) CountByItem(_ context.Context, _ string) (int, error) {
	if m.countByItemErr != nil {
		return 0, m.countByItemErr
	}
	return m.countByItemResult, nil
}

// ── DeleteWearLog ─────────────────────────────────────────────────────────────

func TestWearLogServiceDeleteWearLogShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.DeleteWearLog(ctx, "owner-1", "log-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrNotFoundWhenLogDoesNotExist(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{})

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-2"

	svc := NewWearLogService(&mockWearLogRepo{logs: []domain.WearLog{log}}, &mockItemRepo{})

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrorWhenDeleteFails(t *testing.T) {
	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log}, delErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, &mockItemRepo{})

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrorWhenLatestByItemFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log}, latestByItemErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}})

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrorWhenCountByItemFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log}, countByItemErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}})

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrorWhenItemGetFails(t *testing.T) {
	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log}}
	itemRepo := &mockItemRepo{getErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, itemRepo)

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrorWhenItemSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.WearCount = 1

	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log}}
	itemRepo := &mockItemRepo{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, itemRepo)

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceDeleteWearLogShouldSetLastWornAtNilAndWearCountZeroWhenNoLogsRemain(t *testing.T) {
	wornOn := time.Now().Add(-48 * time.Hour)
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.WearCount = 1
	item.LastWornAt = &wornOn

	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log}}
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewWearLogService(logRepo, itemRepo)

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.NoError(t, err)
	require.Empty(t, logRepo.logs)
	require.Equal(t, 0, itemRepo.items[0].WearCount)
	require.Nil(t, itemRepo.items[0].LastWornAt)
}

func TestWearLogServiceDeleteWearLogShouldRecomputeStatsFromRemainingLogsWhenLogsExist(t *testing.T) {
	wornOn1 := time.Now().Add(-24 * time.Hour)
	wornOn2 := time.Now().Add(-48 * time.Hour)

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.WearCount = 2
	item.LastWornAt = &wornOn1

	var log1 domain.WearLog
	log1.ID = "log-1"
	log1.ItemID = "item-1"
	log1.OwnerID = "owner-1"
	log1.WornOn = wornOn1

	var log2 domain.WearLog
	log2.ID = "log-2"
	log2.ItemID = "item-1"
	log2.OwnerID = "owner-1"
	log2.WornOn = wornOn2

	latestRemaining := log2
	logRepo := &mockWearLogRepo{
		logs:               []domain.WearLog{log1, log2},
		latestByItemResult: &latestRemaining,
		countByItemResult:  1,
	}
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewWearLogService(logRepo, itemRepo)

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.NoError(t, err)
	require.Equal(t, 1, itemRepo.items[0].WearCount)
	require.NotNil(t, itemRepo.items[0].LastWornAt)
	require.Equal(t, wornOn2, *itemRepo.items[0].LastWornAt)
}

// ── ListByItem ────────────────────────────────────────────────────────────────

func TestWearLogServiceListByItemShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListByItem(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogServiceListByItemShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{})

	_, err := svc.ListByItem(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestWearLogServiceListByItemShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-2"

	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{items: []domain.Item{item}})

	_, err := svc.ListByItem(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestWearLogServiceListByItemShouldReturnErrorWhenRepoListByItemFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{listByItemErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}})

	_, err := svc.ListByItem(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceListByItemShouldReturnLogsWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	var log1, log2 domain.WearLog
	log1.ID = "log-1"
	log1.ItemID = "item-1"
	log1.OwnerID = "owner-1"
	log2.ID = "log-2"
	log2.ItemID = "item-1"
	log2.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log1, log2}}
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}})

	got, err := svc.ListByItem(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

// ── LogWear ───────────────────────────────────────────────────────────────────

func TestWearLogServiceLogWearShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.LogWear(ctx, "owner-1", "item-1", time.Now(), nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogServiceLogWearShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{})

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", time.Now(), nil)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestWearLogServiceLogWearShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-2"

	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{items: []domain.Item{item}})

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", time.Now(), nil)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestWearLogServiceLogWearShouldReturnErrFutureDateNotAllowedWhenWornOnIsInFuture(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	future := time.Now().Add(24 * time.Hour)
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{items: []domain.Item{item}})

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", future, nil)
	require.ErrorIs(t, err, domain.ErrFutureDateNotAllowed)
}

func TestWearLogServiceLogWearShouldReturnErrorWhenWearLogSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{saveErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}})

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", time.Now(), nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceLogWearShouldReturnErrorWhenItemSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	itemRepo := &mockItemRepo{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewWearLogService(&mockWearLogRepo{}, itemRepo)

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", time.Now(), nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceLogWearShouldNotUpdateLastWornAtWhenNewDateIsOlderThanExisting(t *testing.T) {
	recent := time.Now().Add(-24 * time.Hour).UTC()
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.WearCount = 1
	item.LastWornAt = &recent

	older := time.Now().Add(-48 * time.Hour).UTC()
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewWearLogService(&mockWearLogRepo{}, itemRepo)

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", older, nil)
	require.NoError(t, err)

	// LastWornAt must remain the more recent date, not be regressed to the older one.
	require.Equal(t, 2, itemRepo.items[0].WearCount)
	require.Equal(t, recent, *itemRepo.items[0].LastWornAt)
}

func TestWearLogServiceLogWearShouldCreateLogAndUpdateItemStatsWhenSuccessful(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.WearCount = 2

	logRepo := &mockWearLogRepo{}
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewWearLogService(logRepo, itemRepo)

	note := "first test wear"
	wornOn := time.Now().Add(-24 * time.Hour).UTC()
	got, err := svc.LogWear(t.Context(), "owner-1", "item-1", wornOn, &note)

	require.NoError(t, err)
	require.NotEmpty(t, got.GetID())
	require.Equal(t, "item-1", got.ItemID)
	require.Equal(t, "owner-1", got.OwnerID)
	require.Equal(t, wornOn, got.WornOn)
	require.NotNil(t, got.Notes)
	require.Equal(t, note, *got.Notes)
	require.False(t, got.CreatedAt.IsZero())

	require.Len(t, logRepo.logs, 1)
	require.Equal(t, 3, itemRepo.items[0].WearCount)
	require.NotNil(t, itemRepo.items[0].LastWornAt)
	require.Equal(t, wornOn, *itemRepo.items[0].LastWornAt)
}
