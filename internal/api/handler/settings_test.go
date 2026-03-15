package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/api/middleware"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeSettingsService struct {
	getSettingsFn                func(ctx context.Context) (domain.AppSettings, error)
	updateRegistrationEnabledFn  func(ctx context.Context, callerID string, enabled bool) error
}

func (f *fakeSettingsService) GetSettings(ctx context.Context) (domain.AppSettings, error) {
	return f.getSettingsFn(ctx)
}

func (f *fakeSettingsService) UpdateRegistrationEnabled(ctx context.Context, callerID string, enabled bool) error {
	return f.updateRegistrationEnabledFn(ctx, callerID, enabled)
}

// --- helpers ---

func getAdminSettings(t *testing.T, h *handler.SettingsHandler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/admin/settings", nil)
	w := httptest.NewRecorder()
	h.GetSettings(w, req)
	return w
}

func patchAdminSettings(t *testing.T, h *handler.SettingsHandler, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/admin/settings", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	return w
}

// --- tests ---

func TestGetSettingsHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeSettingsService{
		getSettingsFn: func(_ context.Context) (domain.AppSettings, error) {
			return domain.AppSettings{}, domain.ErrIO
		},
		updateRegistrationEnabledFn: func(_ context.Context, _ string, _ bool) error { return nil },
	}
	h := handler.NewSettingsHandler(svc, slog.New(slog.DiscardHandler))

	w := getAdminSettings(t, h)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetSettingsHandlerShouldReturn200WithSettingsWhenServiceSucceeds(t *testing.T) {
	svc := &fakeSettingsService{
		getSettingsFn: func(_ context.Context) (domain.AppSettings, error) {
			return domain.AppSettings{RegistrationEnabled: true}, nil
		},
		updateRegistrationEnabledFn: func(_ context.Context, _ string, _ bool) error { return nil },
	}
	h := handler.NewSettingsHandler(svc, slog.New(slog.DiscardHandler))

	w := getAdminSettings(t, h)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, true, body["registration_enabled"])
}

func TestUpdateSettingsHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	svc := &fakeSettingsService{
		getSettingsFn:               func(_ context.Context) (domain.AppSettings, error) { return domain.AppSettings{}, nil },
		updateRegistrationEnabledFn: func(_ context.Context, _ string, _ bool) error { return nil },
	}
	h := handler.NewSettingsHandler(svc, slog.New(slog.DiscardHandler))

	// No user ID injected into context — simulates missing auth middleware.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/admin/settings", strings.NewReader(`{"registration_enabled":true}`))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateSettingsHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	svc := &fakeSettingsService{
		getSettingsFn:               func(_ context.Context) (domain.AppSettings, error) { return domain.AppSettings{}, nil },
		updateRegistrationEnabledFn: func(_ context.Context, _ string, _ bool) error { return nil },
	}
	h := handler.NewSettingsHandler(svc, slog.New(slog.DiscardHandler))

	w := patchAdminSettings(t, h, "user-1", "not-json")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSettingsHandlerShouldReturn500WhenUpdateFails(t *testing.T) {
	svc := &fakeSettingsService{
		getSettingsFn: func(_ context.Context) (domain.AppSettings, error) { return domain.AppSettings{}, nil },
		updateRegistrationEnabledFn: func(_ context.Context, _ string, _ bool) error {
			return domain.ErrIO
		},
	}
	h := handler.NewSettingsHandler(svc, slog.New(slog.DiscardHandler))

	w := patchAdminSettings(t, h, "user-1", `{"registration_enabled":true}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateSettingsHandlerShouldReturn200WithUpdatedSettingsWhenSucceeds(t *testing.T) {
	svc := &fakeSettingsService{
		getSettingsFn: func(_ context.Context) (domain.AppSettings, error) {
			return domain.AppSettings{RegistrationEnabled: false}, nil
		},
		updateRegistrationEnabledFn: func(_ context.Context, _ string, _ bool) error { return nil },
	}
	h := handler.NewSettingsHandler(svc, slog.New(slog.DiscardHandler))

	w := patchAdminSettings(t, h, "user-1", `{"registration_enabled":false}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, false, body["registration_enabled"])
}

// statefulSettingsService is an in-memory settings service for cycle tests.
type statefulSettingsService struct {
	settings domain.AppSettings
}

func (s *statefulSettingsService) GetSettings(_ context.Context) (domain.AppSettings, error) {
	return s.settings, nil
}

func (s *statefulSettingsService) UpdateRegistrationEnabled(_ context.Context, _ string, enabled bool) error {
	s.settings.RegistrationEnabled = enabled
	return nil
}

func TestSettingsCycleShouldReflectUpdateWhenReadUpdateRead(t *testing.T) {
	svc := &statefulSettingsService{settings: domain.AppSettings{RegistrationEnabled: true}}
	h := handler.NewSettingsHandler(svc, slog.New(slog.DiscardHandler))

	// Step 1: Read initial settings — registration enabled.
	w := getAdminSettings(t, h)
	require.Equal(t, http.StatusOK, w.Code)
	var initial map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&initial))
	require.Equal(t, true, initial["registration_enabled"])

	// Step 2: Update — disable registration.
	w = patchAdminSettings(t, h, "admin-1", `{"registration_enabled":false}`)
	require.Equal(t, http.StatusOK, w.Code)

	// Step 3: Read again — registration should now be disabled.
	w = getAdminSettings(t, h)
	require.Equal(t, http.StatusOK, w.Code)
	var updated map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&updated))
	require.Equal(t, false, updated["registration_enabled"])
}
