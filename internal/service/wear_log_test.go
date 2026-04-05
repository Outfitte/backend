package service

import (
	"context"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// mockWearLogRepo is an in-memory ports.WearLogRepository for tests.
type mockWearLogRepo struct {
	logs    []domain.WearLog
	getErr  error
	saveErr error
	delErr  error

	listByItemErr error
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

func (m *mockWearLogRepo) LatestByItem(_ context.Context, itemID string) (*domain.WearLog, error) {
	for _, l := range m.logs {
		if l.ItemID == itemID {
			return &l, nil
		}
	}
	return nil, nil
}

func (m *mockWearLogRepo) CountByItem(_ context.Context, itemID string) (int, error) {
	count := 0
	for _, l := range m.logs {
		if l.ItemID == itemID {
			count++
		}
	}
	return count, nil
}

// ── DeleteWearLog ─────────────────────────────────────────────────────────────

func TestWearLogServiceDeleteWearLogShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.DeleteWearLog(ctx, "owner-1", "log-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrNotFoundWhenLogDoesNotExist(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-2"

	svc := NewWearLogService(&mockWearLogRepo{logs: []domain.WearLog{log}}, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestWearLogServiceDeleteWearLogShouldReturnErrorWhenDeleteFails(t *testing.T) {
	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log}, delErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceDeleteWearLogShouldDeleteLogWhenSuccessful(t *testing.T) {
	var log domain.WearLog
	log.ID = "log-1"
	log.ItemID = "item-1"
	log.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log}}
	svc := NewWearLogService(logRepo, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.DeleteWearLog(t.Context(), "owner-1", "log-1")
	require.NoError(t, err)
	require.Empty(t, logRepo.logs)
}

// ── ListByItem ────────────────────────────────────────────────────────────────

func TestWearLogServiceListByItemShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListByItem(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogServiceListByItemShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.ListByItem(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestWearLogServiceListByItemShouldReturnErrForbiddenWhenCallerHasNoSharedAccess(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-2"

	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{items: []domain.Item{item}}, &mockShareAccessChecker{hasAccess: false})

	_, err := svc.ListByItem(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestWearLogServiceListByItemShouldReturnLogsWhenCallerHasSharedAccess(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-2"

	var log1 domain.WearLog
	log1.ID = "log-1"
	log1.ItemID = "item-1"
	log1.OwnerID = "owner-2"

	logRepo := &mockWearLogRepo{logs: []domain.WearLog{log1}}
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}}, &mockShareAccessChecker{hasAccess: true})

	got, err := svc.ListByItem(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestWearLogServiceListByItemShouldReturnErrorWhenShareCheckFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-2"

	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{items: []domain.Item{item}}, &mockShareAccessChecker{err: domain.ErrIO})

	_, err := svc.ListByItem(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceListByItemShouldReturnErrorWhenRepoListByItemFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{listByItemErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}}, &mockShareAccessChecker{})

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
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}}, &mockShareAccessChecker{})

	got, err := svc.ListByItem(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

// ── LogWear ───────────────────────────────────────────────────────────────────

func TestWearLogServiceLogWearShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.LogWear(ctx, "owner-1", "item-1", time.Now(), nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestWearLogServiceLogWearShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", time.Now(), nil)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestWearLogServiceLogWearShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-2"

	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{items: []domain.Item{item}}, &mockShareAccessChecker{})

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", time.Now(), nil)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestWearLogServiceLogWearShouldReturnErrFutureDateNotAllowedWhenWornOnIsInFuture(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	future := time.Now().Add(24 * time.Hour)
	svc := NewWearLogService(&mockWearLogRepo{}, &mockItemRepo{items: []domain.Item{item}}, &mockShareAccessChecker{})

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", future, nil)
	require.ErrorIs(t, err, domain.ErrFutureDateNotAllowed)
}

func TestWearLogServiceLogWearShouldReturnErrorWhenWearLogSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{saveErr: domain.ErrIO}
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}}, &mockShareAccessChecker{})

	_, err := svc.LogWear(t.Context(), "owner-1", "item-1", time.Now(), nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestWearLogServiceLogWearShouldCreateLogWhenSuccessful(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	logRepo := &mockWearLogRepo{}
	svc := NewWearLogService(logRepo, &mockItemRepo{items: []domain.Item{item}}, &mockShareAccessChecker{})

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
}
