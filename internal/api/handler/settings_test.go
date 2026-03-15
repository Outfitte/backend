package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeSettingsGetter struct {
	getSettingsFn func(ctx context.Context) (domain.AppSettings, error)
}

func (f *fakeSettingsGetter) GetSettings(ctx context.Context) (domain.AppSettings, error) {
	return f.getSettingsFn(ctx)
}

// --- helpers ---

func getAdminSettings(t *testing.T, h *handler.SettingsHandler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/admin/settings", nil)
	w := httptest.NewRecorder()
	h.GetSettings(w, req)
	return w
}

// --- tests ---

func TestGetSettingsHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeSettingsGetter{
		getSettingsFn: func(_ context.Context) (domain.AppSettings, error) {
			return domain.AppSettings{}, domain.ErrIO
		},
	}
	h := handler.NewSettingsHandler(svc, slog.New(slog.DiscardHandler))

	w := getAdminSettings(t, h)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetSettingsHandlerShouldReturn200WithSettingsWhenServiceSucceeds(t *testing.T) {
	svc := &fakeSettingsGetter{
		getSettingsFn: func(_ context.Context) (domain.AppSettings, error) {
			return domain.AppSettings{RegistrationEnabled: true}, nil
		},
	}
	h := handler.NewSettingsHandler(svc, slog.New(slog.DiscardHandler))

	w := getAdminSettings(t, h)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, true, body["registration_enabled"])
}
