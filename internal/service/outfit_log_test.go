package service

import (
	"context"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// mockOutfitLogRepo is an in-memory ports.OutfitLogRepository for tests.
type mockOutfitLogRepo struct {
	logs               []domain.OutfitLog
	getErr             error
	listByOutfitErr    error
	listByDateRangeErr error
}

func (m *mockOutfitLogRepo) Get(_ context.Context, id string) (domain.OutfitLog, error) {
	if m.getErr != nil {
		return domain.OutfitLog{}, m.getErr
	}
	for _, l := range m.logs {
		if l.GetID() == id {
			return l, nil
		}
	}
	return domain.OutfitLog{}, domain.ErrNotFound
}

func (m *mockOutfitLogRepo) Save(_ context.Context, log domain.OutfitLog) error {
	for i, existing := range m.logs {
		if existing.GetID() == log.GetID() {
			m.logs[i] = log
			return nil
		}
	}
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockOutfitLogRepo) Delete(_ context.Context, id string) error {
	for i, l := range m.logs {
		if l.GetID() == id {
			m.logs = append(m.logs[:i], m.logs[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockOutfitLogRepo) ListByOutfit(_ context.Context, outfitID string) ([]domain.OutfitLog, error) {
	if m.listByOutfitErr != nil {
		return nil, m.listByOutfitErr
	}
	var result []domain.OutfitLog
	for _, l := range m.logs {
		if l.OutfitID == outfitID {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *mockOutfitLogRepo) ListByOwnerDateRange(_ context.Context, ownerID string, from, to time.Time) ([]domain.OutfitLog, error) {
	if m.listByDateRangeErr != nil {
		return nil, m.listByDateRangeErr
	}
	var result []domain.OutfitLog
	for _, l := range m.logs {
		if l.OwnerID == ownerID && !l.WornOn.Before(from) && !l.WornOn.After(to) {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *mockOutfitLogRepo) LinkWearLog(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockOutfitLogRepo) LinkedWearLogIDs(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (m *mockOutfitLogRepo) RemoveWearLogLink(_ context.Context, _ string) error {
	return nil
}

// mockOutfitLogTransactor is an in-memory ports.OutfitLogTransactor for tests.
type mockOutfitLogTransactor struct {
	repo          *mockOutfitLogRepo // optional: when set, UpdateOutfitLogDate applies the change to repo.logs
	createErr     error
	deleteErr     error
	updateDateErr error
	createdLog    domain.OutfitLog
	deletedLogID  string
	updatedLogID  string
	updatedDate   time.Time
}

func (m *mockOutfitLogTransactor) CreateOutfitLog(_ context.Context, log domain.OutfitLog, _ []domain.WearLog) (domain.OutfitLog, error) {
	if m.createErr != nil {
		return domain.OutfitLog{}, m.createErr
	}
	m.createdLog = log
	return log, nil
}

func (m *mockOutfitLogTransactor) DeleteOutfitLog(_ context.Context, outfitLogID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedLogID = outfitLogID
	return nil
}

func (m *mockOutfitLogTransactor) UpdateOutfitLogDate(_ context.Context, outfitLogID string, newDate time.Time) error {
	if m.updateDateErr != nil {
		return m.updateDateErr
	}
	m.updatedLogID = outfitLogID
	m.updatedDate = newDate
	if m.repo != nil {
		for i, l := range m.repo.logs {
			if l.GetID() == outfitLogID {
				m.repo.logs[i].WornOn = newDate
				break
			}
		}
	}
	return nil
}

// helpers

func outfitLogWithOwner(id, outfitID, ownerID string, wornOn time.Time) domain.OutfitLog {
	var l domain.OutfitLog
	l.ID = id
	l.OutfitID = outfitID
	l.OwnerID = ownerID
	l.WornOn = wornOn
	l.CreatedAt = time.Now().UTC()
	return l
}

func newOutfitLogSvc(outfits *mockOutfitRepo, outfitLogs *mockOutfitLogRepo, transactor *mockOutfitLogTransactor, shares *mockShareAccessChecker) *OutfitLogService {
	return NewOutfitLogService(outfits, outfitLogs, transactor, shares)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestOutfitLogServiceDeleteShouldReturnContextErrorWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	err := svc.Delete(ctx, "owner-1", "log-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogServiceDeleteShouldReturnNotFoundWhenOutfitLogDoesNotExist(t *testing.T) {
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogServiceDeleteShouldReturnForbiddenWhenCallerIsNotOwner(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).UTC()
	log := outfitLogWithOwner("log-1", "outfit-1", "owner-2", past)
	logRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log}}
	svc := newOutfitLogSvc(&mockOutfitRepo{}, logRepo, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestOutfitLogServiceDeleteShouldReturnErrorWhenTransactorDeleteFails(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).UTC()
	log := outfitLogWithOwner("log-1", "outfit-1", "owner-1", past)
	logRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log}}
	transactor := &mockOutfitLogTransactor{deleteErr: domain.ErrIO}
	svc := newOutfitLogSvc(&mockOutfitRepo{}, logRepo, transactor, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogServiceDeleteShouldDeleteWhenCallerIsOwner(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).UTC()
	log := outfitLogWithOwner("log-1", "outfit-1", "owner-1", past)
	logRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log}}
	transactor := &mockOutfitLogTransactor{}
	svc := newOutfitLogSvc(&mockOutfitRepo{}, logRepo, transactor, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "log-1")
	require.NoError(t, err)
	require.Equal(t, "log-1", transactor.deletedLogID)
}

// ── UpdateDate ────────────────────────────────────────────────────────────────

func TestOutfitLogServiceUpdateDateShouldReturnContextErrorWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.UpdateDate(ctx, "owner-1", "log-1", time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogServiceUpdateDateShouldReturnErrFutureDateNotAllowedWhenNewDateIsInFuture(t *testing.T) {
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.UpdateDate(t.Context(), "owner-1", "log-1", time.Now().Add(24*time.Hour))
	require.ErrorIs(t, err, domain.ErrFutureDateNotAllowed)
}

func TestOutfitLogServiceUpdateDateShouldReturnNotFoundWhenOutfitLogDoesNotExist(t *testing.T) {
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.UpdateDate(t.Context(), "owner-1", "log-1", time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogServiceUpdateDateShouldReturnForbiddenWhenCallerIsNotOwner(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).UTC()
	log := outfitLogWithOwner("log-1", "outfit-1", "owner-2", past)
	logRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log}}
	svc := newOutfitLogSvc(&mockOutfitRepo{}, logRepo, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.UpdateDate(t.Context(), "owner-1", "log-1", time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestOutfitLogServiceUpdateDateShouldReturnErrorWhenTransactorUpdateFails(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).UTC()
	log := outfitLogWithOwner("log-1", "outfit-1", "owner-1", past)
	logRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log}}
	transactor := &mockOutfitLogTransactor{updateDateErr: domain.ErrIO}
	svc := newOutfitLogSvc(&mockOutfitRepo{}, logRepo, transactor, &mockShareAccessChecker{})

	_, err := svc.UpdateDate(t.Context(), "owner-1", "log-1", time.Now().Add(-time.Hour))
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogServiceUpdateDateShouldReturnUpdatedLogWhenSuccessful(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).UTC()
	newDate := time.Now().Add(-48 * time.Hour).UTC()
	log := outfitLogWithOwner("log-1", "outfit-1", "owner-1", past)
	logRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log}}
	transactor := &mockOutfitLogTransactor{repo: logRepo}
	svc := newOutfitLogSvc(&mockOutfitRepo{}, logRepo, transactor, &mockShareAccessChecker{})

	got, err := svc.UpdateDate(t.Context(), "owner-1", "log-1", newDate)
	require.NoError(t, err)
	require.Equal(t, "log-1", got.GetID())
	require.Equal(t, newDate, got.WornOn)
	require.Equal(t, "log-1", transactor.updatedLogID)
	require.Equal(t, newDate.UTC(), transactor.updatedDate)
}

// ── ListByDateRange ───────────────────────────────────────────────────────────

func TestOutfitLogServiceListByDateRangeShouldReturnValidationErrorWhenFromIsAfterTo(t *testing.T) {
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	from := time.Now().UTC()
	to := from.Add(-24 * time.Hour)
	_, err := svc.ListByDateRange(t.Context(), "owner-1", from, to)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestOutfitLogServiceListByDateRangeShouldReturnContextErrorWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	from := time.Now().Add(-7 * 24 * time.Hour)
	_, err := svc.ListByDateRange(ctx, "owner-1", from, time.Now())
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogServiceListByDateRangeShouldReturnErrorWhenRepoFails(t *testing.T) {
	logRepo := &mockOutfitLogRepo{listByDateRangeErr: domain.ErrIO}
	svc := newOutfitLogSvc(&mockOutfitRepo{}, logRepo, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	from := time.Now().Add(-7 * 24 * time.Hour)
	_, err := svc.ListByDateRange(t.Context(), "owner-1", from, time.Now())
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogServiceListByDateRangeShouldReturnLogsWithinRangeWhenSuccessful(t *testing.T) {
	base := time.Now().UTC()
	log1 := outfitLogWithOwner("log-1", "outfit-1", "owner-1", base.Add(-1*24*time.Hour))
	log2 := outfitLogWithOwner("log-2", "outfit-1", "owner-1", base.Add(-3*24*time.Hour))
	logRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log1, log2}}
	svc := newOutfitLogSvc(&mockOutfitRepo{}, logRepo, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	from := base.Add(-7 * 24 * time.Hour)
	got, err := svc.ListByDateRange(t.Context(), "owner-1", from, base)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

// ── ListByOutfit ──────────────────────────────────────────────────────────────

func TestOutfitLogServiceListByOutfitShouldReturnContextErrorWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.ListByOutfit(ctx, "owner-1", "outfit-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogServiceListByOutfitShouldReturnNotFoundWhenOutfitDoesNotExist(t *testing.T) {
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.ListByOutfit(t.Context(), "owner-1", "outfit-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogServiceListByOutfitShouldReturnErrorWhenShareCheckFails(t *testing.T) {
	outfit := outfitWithOwner("outfit-1", "owner-2")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitLogSvc(outfitRepo, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{err: domain.ErrIO})

	_, err := svc.ListByOutfit(t.Context(), "owner-1", "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogServiceListByOutfitShouldReturnForbiddenWhenCallerIsNotOutfitOwnerAndHasNoSharedAccess(t *testing.T) {
	outfit := outfitWithOwner("outfit-1", "owner-2")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitLogSvc(outfitRepo, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{hasAccess: false})

	_, err := svc.ListByOutfit(t.Context(), "owner-1", "outfit-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestOutfitLogServiceListByOutfitShouldReturnErrorWhenRepoListFails(t *testing.T) {
	outfit := outfitWithOwner("outfit-1", "owner-1")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	logRepo := &mockOutfitLogRepo{listByOutfitErr: domain.ErrIO}
	svc := newOutfitLogSvc(outfitRepo, logRepo, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.ListByOutfit(t.Context(), "owner-1", "outfit-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogServiceListByOutfitShouldReturnLogsWhenCallerHasSharedAccess(t *testing.T) {
	outfit := outfitWithOwner("outfit-1", "owner-2")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	past := time.Now().Add(-24 * time.Hour).UTC()
	log1 := outfitLogWithOwner("log-1", "outfit-1", "owner-2", past)
	log2 := outfitLogWithOwner("log-2", "outfit-1", "owner-2", past.Add(-time.Hour))
	logRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log1, log2}}
	svc := newOutfitLogSvc(outfitRepo, logRepo, &mockOutfitLogTransactor{}, &mockShareAccessChecker{hasAccess: true})

	got, err := svc.ListByOutfit(t.Context(), "owner-1", "outfit-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestOutfitLogServiceListByOutfitShouldReturnLogsWhenCallerIsOwner(t *testing.T) {
	outfit := outfitWithOwner("outfit-1", "owner-1")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	past := time.Now().Add(-24 * time.Hour).UTC()
	log1 := outfitLogWithOwner("log-1", "outfit-1", "owner-1", past)
	log2 := outfitLogWithOwner("log-2", "outfit-1", "owner-1", past.Add(-time.Hour))
	logRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log1, log2}}
	svc := newOutfitLogSvc(outfitRepo, logRepo, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	got, err := svc.ListByOutfit(t.Context(), "owner-1", "outfit-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

// ── LogWear ───────────────────────────────────────────────────────────────────

func TestOutfitLogServiceLogWearShouldReturnContextErrorWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.LogWear(ctx, "owner-1", "outfit-1", time.Now(), nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitLogServiceLogWearShouldReturnNotFoundWhenOutfitDoesNotExist(t *testing.T) {
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.LogWear(t.Context(), "owner-1", "outfit-1", time.Now(), nil)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitLogServiceLogWearShouldReturnForbiddenWhenCallerIsNotOutfitOwner(t *testing.T) {
	outfit := outfitWithOwner("outfit-1", "owner-2")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitLogSvc(outfitRepo, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	_, err := svc.LogWear(t.Context(), "owner-1", "outfit-1", time.Now(), nil)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestOutfitLogServiceLogWearShouldReturnErrFutureDateNotAllowedWhenWornOnIsInFuture(t *testing.T) {
	// Date guard fires before ownership lookup — no outfit needed in repo.
	svc := newOutfitLogSvc(&mockOutfitRepo{}, &mockOutfitLogRepo{}, &mockOutfitLogTransactor{}, &mockShareAccessChecker{})

	future := time.Now().Add(24 * time.Hour)
	_, err := svc.LogWear(t.Context(), "owner-1", "outfit-1", future, nil)
	require.ErrorIs(t, err, domain.ErrFutureDateNotAllowed)
}

func TestOutfitLogServiceLogWearShouldReturnErrorWhenTransactorCreateFails(t *testing.T) {
	outfit := outfitWithOwner("outfit-1", "owner-1")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	transactor := &mockOutfitLogTransactor{createErr: domain.ErrIO}
	svc := newOutfitLogSvc(outfitRepo, &mockOutfitLogRepo{}, transactor, &mockShareAccessChecker{})

	_, err := svc.LogWear(t.Context(), "owner-1", "outfit-1", time.Now().Add(-time.Hour), nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitLogServiceLogWearShouldCreateOutfitLogWithWearLogsPerItemWhenSuccessful(t *testing.T) {
	outfit := outfitWithOwner("outfit-1", "owner-1")
	outfit.Items = []domain.OutfitItem{
		{OutfitID: "outfit-1", ItemID: "item-1", Position: 0},
		{OutfitID: "outfit-1", ItemID: "item-2", Position: 1},
	}
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	transactor := &mockOutfitLogTransactor{}
	svc := newOutfitLogSvc(outfitRepo, &mockOutfitLogRepo{}, transactor, &mockShareAccessChecker{})

	note := "first wear"
	wornOn := time.Now().Add(-24 * time.Hour).UTC()
	got, err := svc.LogWear(t.Context(), "owner-1", "outfit-1", wornOn, &note)

	require.NoError(t, err)
	require.NotEmpty(t, got.GetID())
	require.Equal(t, "outfit-1", got.OutfitID)
	require.Equal(t, "owner-1", got.OwnerID)
	require.Equal(t, wornOn, got.WornOn)
	require.NotNil(t, got.Notes)
	require.Equal(t, note, *got.Notes)
	require.False(t, got.CreatedAt.IsZero())
	require.Equal(t, got.GetID(), transactor.createdLog.GetID())
}
