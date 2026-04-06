package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// integrationJWTSecret is a 32+ char secret used by all integration tests.
const integrationJWTSecret = "integration-test-jwt-secret-32-chars-long!!"

func newIntegrationConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		DB:               config.DBConfig{Driver: "sqlite", DSN: ":memory:"},
		MediaStoragePath: t.TempDir(),
		ServerPort:       "8080",
		JWTSecret:        integrationJWTSecret,
	}
}

func startIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, cleanup, err := newServer(t.Context(), newIntegrationConfig(t), slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(func() {
		ts.Close()
		cleanup()
	})
	return ts
}

// --- helpers ---

func doJSON(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, r)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}

func registerUser(t *testing.T, srv *httptest.Server, username, password string) (accessToken, refreshToken string) {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/auth/register", map[string]string{
		"username": username,
		"password": password,
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.AccessToken)
	return result.AccessToken, result.RefreshToken
}

func createLocation(t *testing.T, srv *httptest.Server, token, label string) string {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/locations", map[string]any{"label": label}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.ID)
	return result.ID
}

func createItem(t *testing.T, srv *httptest.Server, token, name string, locationID *string) string {
	t.Helper()
	body := map[string]any{"name": name}
	if locationID != nil {
		body["location_id"] = *locationID
	}
	resp := doJSON(t, srv, http.MethodPost, "/items", body, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.ID)
	return result.ID
}

func uploadPhoto(t *testing.T, srv *httptest.Server, token, itemID string) string {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("photo", "dummy.jpg")
	require.NoError(t, err)
	_, err = fw.Write([]byte("fake image bytes"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/items/"+itemID+"/photos", &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Retrieve item to find the photo key.
	getResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		Photos []struct {
			MediaKey string `json:"media_key"`
		} `json:"photos"`
	}
	decodeJSON(t, getResp, &item)
	require.Len(t, item.Photos, 1)
	return item.Photos[0].MediaKey
}

func uploadOutfitPhoto(t *testing.T, srv *httptest.Server, token, outfitID string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("photo", "outfit.jpg")
	require.NoError(t, err)
	_, err = fw.Write([]byte("fake outfit image bytes"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/outfits/"+outfitID+"/photos", &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

type itemWearLogEntry struct {
	ID     string `json:"id"`
	WornOn string `json:"worn_on"`
}

func getItemWearLogs(t *testing.T, srv *httptest.Server, token, itemID string) []itemWearLogEntry {
	t.Helper()
	resp := doJSON(t, srv, http.MethodGet, "/items/"+itemID+"/wear-logs", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var logs []itemWearLogEntry
	decodeJSON(t, resp, &logs)
	return logs
}

// --- unauthorized cases ---

func TestIntegrationUnauthenticatedEndpoints(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"get items", http.MethodGet, "/items", nil},
		{"get locations", http.MethodGet, "/locations", nil},
		{"post items", http.MethodPost, "/items", map[string]any{"name": "jacket"}},
		{"post wear log", http.MethodPost, "/items/some-id/wear-logs", map[string]any{"worn_on": "2026-01-01"}},
		{"get wear logs", http.MethodGet, "/items/some-id/wear-logs", nil},
		{"delete wear log", http.MethodDelete, "/items/some-id/wear-logs/some-log-id", nil},
		{"archive item", http.MethodPost, "/items/some-id/archive", nil},
		{"unarchive item", http.MethodPost, "/items/some-id/unarchive", nil},
		{"dispose item", http.MethodPost, "/items/some-id/dispose", map[string]any{"reason": "donated"}},
		{"post outfit", http.MethodPost, "/outfits", map[string]any{}},
		{"get outfits", http.MethodGet, "/outfits", nil},
		{"get outfit by id", http.MethodGet, "/outfits/some-id", nil},
		{"delete outfit", http.MethodDelete, "/outfits/some-id", nil},
		{"add item to outfit", http.MethodPost, "/outfits/some-id/items", map[string]any{"item_id": "x"}},
		{"upload outfit photo", http.MethodPost, "/outfits/some-id/photos", nil},
		{"post outfit log", http.MethodPost, "/outfits/some-id/logs", map[string]any{"worn_on": "2026-01-01"}},
		{"get outfit logs", http.MethodGet, "/outfits/some-id/logs", nil},
		{"patch outfit log", http.MethodPatch, "/outfits/some-id/logs/some-log-id", map[string]any{"worn_on": "2026-01-01"}},
		{"delete outfit log", http.MethodDelete, "/outfits/some-id/logs/some-log-id", nil},
		{"get outfit logs by date range", http.MethodGet, "/outfit-logs?from=2026-01-01&to=2026-01-31", nil},
	}

	srv := startIntegrationServer(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, srv, tc.method, tc.path, tc.body, "")
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

// enableRegistration calls PATCH /admin/settings to allow new registrations.
func enableRegistration(t *testing.T, srv *httptest.Server, adminToken string) {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPatch, "/admin/settings",
		map[string]any{"registration_enabled": true}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- auth cycle ---

func TestIntegrationAuthCycleShouldLoginRefreshAndLogout(t *testing.T) {
	srv := startIntegrationServer(t)

	// Register creates the account and returns tokens.
	_, refreshToken := registerUser(t, srv, "authcycleuser", "password-authcycle-secure")

	// Login with credentials returns a fresh token pair.
	loginResp := doJSON(t, srv, http.MethodPost, "/auth/login",
		map[string]string{"username": "authcycleuser", "password": "password-authcycle-secure"}, "")
	require.Equal(t, http.StatusOK, loginResp.StatusCode)
	var loginResult struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	decodeJSON(t, loginResp, &loginResult)
	require.NotEmpty(t, loginResult.AccessToken)
	require.NotEmpty(t, loginResult.RefreshToken)

	// Refresh using the token obtained at registration (not the one from login above),
	// confirming both independent token lineages are valid.
	refreshResp := doJSON(t, srv, http.MethodPost, "/auth/refresh",
		map[string]string{"refresh_token": refreshToken}, "")
	require.Equal(t, http.StatusOK, refreshResp.StatusCode)
	var refreshResult struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	decodeJSON(t, refreshResp, &refreshResult)
	require.NotEmpty(t, refreshResult.AccessToken)
	require.NotEmpty(t, refreshResult.RefreshToken)

	// Logout invalidates the refresh token.
	logoutResp := doJSON(t, srv, http.MethodPost, "/auth/logout",
		map[string]string{"refresh_token": refreshResult.RefreshToken}, "")
	assert.Equal(t, http.StatusNoContent, logoutResp.StatusCode)

	// Using the invalidated refresh token must fail.
	staleResp := doJSON(t, srv, http.MethodPost, "/auth/refresh",
		map[string]string{"refresh_token": refreshResult.RefreshToken}, "")
	assert.Equal(t, http.StatusUnauthorized, staleResp.StatusCode)
}

// --- full cycle ---

func TestIntegrationFullCycle(t *testing.T) {
	srv := startIntegrationServer(t)

	// First user (admin), then enable registration for subsequent users.
	tokenA, _ := registerUser(t, srv, "alice", "password-alice-secure")
	enableRegistration(t, srv, tokenA)
	tokenB, _ := registerUser(t, srv, "bob", "password-bob-secure")

	// User A creates two locations.
	closetID := createLocation(t, srv, tokenA, "Closet")
	drawerID := createLocation(t, srv, tokenA, "Drawer")

	// User A creates an item assigned to the closet.
	itemID := createItem(t, srv, tokenA, "Blue Jacket", &closetID)

	// User A uploads a photo to the item.
	photoKey := uploadPhoto(t, srv, tokenA, itemID)
	require.NotEmpty(t, photoKey)

	// User A reassigns the item to the drawer.
	resp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID+"/location",
		map[string]any{"location_id": drawerID}, tokenA)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// User A verifies item shows updated location.
	getResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, tokenA)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		LocationID *string `json:"location_id"`
	}
	decodeJSON(t, getResp, &item)
	require.NotNil(t, item.LocationID)
	assert.Equal(t, drawerID, *item.LocationID)

	// User B cannot see User A's item.
	forbidResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, forbidResp.StatusCode)

	// User B has an empty item list.
	listResp := doJSON(t, srv, http.MethodGet, "/items", nil, tokenB)
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	var bItems []any
	decodeJSON(t, listResp, &bItems)
	assert.Empty(t, bItems)

	// User A deletes the photo.
	deletePhotoResp := doJSON(t, srv, http.MethodDelete, "/items/"+itemID+"/photos/"+photoKey, nil, tokenA)
	assert.Equal(t, http.StatusNoContent, deletePhotoResp.StatusCode)

	// User A deletes the item.
	deleteItemResp := doJSON(t, srv, http.MethodDelete, "/items/"+itemID, nil, tokenA)
	assert.Equal(t, http.StatusNoContent, deleteItemResp.StatusCode)

	// Item is gone.
	goneResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, tokenA)
	assert.Equal(t, http.StatusNotFound, goneResp.StatusCode)

	// User A deletes locations.
	deleteClosetResp := doJSON(t, srv, http.MethodDelete, "/locations/"+closetID, nil, tokenA)
	assert.Equal(t, http.StatusNoContent, deleteClosetResp.StatusCode)

	deleteDrawerResp := doJSON(t, srv, http.MethodDelete, "/locations/"+drawerID, nil, tokenA)
	assert.Equal(t, http.StatusNoContent, deleteDrawerResp.StatusCode)

	// Locations are gone.
	listLocsResp := doJSON(t, srv, http.MethodGet, "/locations", nil, tokenA)
	require.Equal(t, http.StatusOK, listLocsResp.StatusCode)
	var locs []any
	decodeJSON(t, listLocsResp, &locs)
	assert.Empty(t, locs)
}

func TestIntegrationCrossUserItemAccessForbidden(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenA, _ := registerUser(t, srv, "forbid-alice", "password-alice-secure")
	enableRegistration(t, srv, tokenA)
	tokenB, _ := registerUser(t, srv, "forbid-bob", "password-bob-secure")

	locID := createLocation(t, srv, tokenA, "Wardrobe")
	itemID := createItem(t, srv, tokenA, "Red Dress", nil)

	logResp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/wear-logs",
		map[string]any{"worn_on": "2026-02-10"}, tokenA)
	require.Equal(t, http.StatusCreated, logResp.StatusCode)
	var wearEntry struct {
		ID string `json:"id"`
	}
	decodeJSON(t, logResp, &wearEntry)

	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"archive item", http.MethodPost, "/items/" + itemID + "/archive", nil},
		{"dispose item", http.MethodPost, "/items/" + itemID + "/dispose", map[string]any{"reason": "donated"}},
		{"log wear for item", http.MethodPost, "/items/" + itemID + "/wear-logs", map[string]any{"worn_on": "2026-01-15"}},
		{"delete wear log", http.MethodDelete, "/items/" + itemID + "/wear-logs/" + wearEntry.ID, nil},
		{"delete item", http.MethodDelete, "/items/" + itemID, nil},
		{"delete location", http.MethodDelete, "/locations/" + locID, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, srv, tc.method, tc.path, tc.body, tokenB)
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

func TestIntegrationWearLogCycle(t *testing.T) {
	srv := startIntegrationServer(t)

	token, _ := registerUser(t, srv, "wearuser", "password-wear-secure")
	itemID := createItem(t, srv, token, "Navy Suit", nil)

	// Log a wear entry.
	logResp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/wear-logs",
		map[string]any{"worn_on": "2026-03-01", "notes": "important meeting"}, token)
	require.Equal(t, http.StatusCreated, logResp.StatusCode)
	var entry struct {
		ID     string  `json:"id"`
		ItemID string  `json:"item_id"`
		WornOn string  `json:"worn_on"`
		Notes  *string `json:"notes"`
	}
	decodeJSON(t, logResp, &entry)
	require.NotEmpty(t, entry.ID)
	assert.Equal(t, itemID, entry.ItemID)
	assert.Equal(t, "2026-03-01", entry.WornOn)
	require.NotNil(t, entry.Notes)
	assert.Equal(t, "important meeting", *entry.Notes)

	// List wear logs — should contain the entry.
	listResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID+"/wear-logs", nil, token)
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	var logs []struct {
		ID string `json:"id"`
	}
	decodeJSON(t, listResp, &logs)
	require.Len(t, logs, 1)
	assert.Equal(t, entry.ID, logs[0].ID)

	// Delete the wear log entry.
	deleteResp := doJSON(t, srv, http.MethodDelete, "/items/"+itemID+"/wear-logs/"+entry.ID, nil, token)
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)

	// List again — should be empty.
	listResp2 := doJSON(t, srv, http.MethodGet, "/items/"+itemID+"/wear-logs", nil, token)
	require.Equal(t, http.StatusOK, listResp2.StatusCode)
	var logsAfter []any
	decodeJSON(t, listResp2, &logsAfter)
	assert.Empty(t, logsAfter)
}

func TestIntegrationArchiveCycle(t *testing.T) {
	srv := startIntegrationServer(t)

	token, _ := registerUser(t, srv, "archiveuser", "password-archive-secure")
	itemID := createItem(t, srv, token, "Wool Trousers", nil)

	// Item appears in active list by default.
	activeResp := doJSON(t, srv, http.MethodGet, "/items", nil, token)
	require.Equal(t, http.StatusOK, activeResp.StatusCode)
	var activeItems []struct{ ID string `json:"id"` }
	decodeJSON(t, activeResp, &activeItems)
	require.Len(t, activeItems, 1)
	assert.Equal(t, itemID, activeItems[0].ID)

	// Archive the item.
	archiveResp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/archive", nil, token)
	assert.Equal(t, http.StatusNoContent, archiveResp.StatusCode)

	// Archived item no longer appears in default (active) list.
	activeAfterResp := doJSON(t, srv, http.MethodGet, "/items", nil, token)
	require.Equal(t, http.StatusOK, activeAfterResp.StatusCode)
	var activeAfter []any
	decodeJSON(t, activeAfterResp, &activeAfter)
	assert.Empty(t, activeAfter)

	// Archived item appears in archived list.
	archivedResp := doJSON(t, srv, http.MethodGet, "/items?status=archived", nil, token)
	require.Equal(t, http.StatusOK, archivedResp.StatusCode)
	var archivedItems []struct{ ID string `json:"id"` }
	decodeJSON(t, archivedResp, &archivedItems)
	require.Len(t, archivedItems, 1)
	assert.Equal(t, itemID, archivedItems[0].ID)

	// Unarchive the item.
	unarchiveResp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/unarchive", nil, token)
	assert.Equal(t, http.StatusNoContent, unarchiveResp.StatusCode)

	// Item is back in active list.
	backResp := doJSON(t, srv, http.MethodGet, "/items", nil, token)
	require.Equal(t, http.StatusOK, backResp.StatusCode)
	var backItems []struct{ ID string `json:"id"` }
	decodeJSON(t, backResp, &backItems)
	require.Len(t, backItems, 1)
	assert.Equal(t, itemID, backItems[0].ID)
}

func TestIntegrationDisposeCycle(t *testing.T) {
	srv := startIntegrationServer(t)

	token, _ := registerUser(t, srv, "disposeuser", "password-dispose-secure")
	itemID := createItem(t, srv, token, "Old Sneakers", nil)

	// Dispose the item.
	disposeResp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/dispose",
		map[string]any{"reason": "donated"}, token)
	assert.Equal(t, http.StatusNoContent, disposeResp.StatusCode)

	// Disposed item does not appear in active list.
	activeResp := doJSON(t, srv, http.MethodGet, "/items", nil, token)
	require.Equal(t, http.StatusOK, activeResp.StatusCode)
	var activeItems []any
	decodeJSON(t, activeResp, &activeItems)
	assert.Empty(t, activeItems)

	// Disposed item appears in archived list (disposed implies archived).
	archivedResp := doJSON(t, srv, http.MethodGet, "/items?status=archived", nil, token)
	require.Equal(t, http.StatusOK, archivedResp.StatusCode)
	var archivedItems []struct {
		ID string `json:"id"`
	}
	decodeJSON(t, archivedResp, &archivedItems)
	require.Len(t, archivedItems, 1)
	assert.Equal(t, itemID, archivedItems[0].ID)
}

func TestIntegrationOutfitLifecycleShouldCreateLogUpdateAndCascadeDelete(t *testing.T) {
	srv := startIntegrationServer(t)

	token, _ := registerUser(t, srv, "outfitlifecycle", "password-lifecycle-secure")
	item1ID := createItem(t, srv, token, "Blue Shirt", nil)
	item2ID := createItem(t, srv, token, "Dark Jeans", nil)

	// Step 1: Create outfit.
	createResp := doJSON(t, srv, http.MethodPost, "/outfits", map[string]any{"name": "Casual Look"}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var outfit struct {
		ID string `json:"id"`
	}
	decodeJSON(t, createResp, &outfit)
	require.NotEmpty(t, outfit.ID)
	outfitID := outfit.ID

	// Step 2: Add two items.
	addItem1Resp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfitID+"/items", map[string]any{"item_id": item1ID}, token)
	require.Equal(t, http.StatusNoContent, addItem1Resp.StatusCode)
	addItem2Resp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfitID+"/items", map[string]any{"item_id": item2ID}, token)
	require.Equal(t, http.StatusNoContent, addItem2Resp.StatusCode)

	// Step 3: Upload a photo; verify it is persisted on the outfit.
	uploadOutfitPhoto(t, srv, token, outfitID)
	getOutfitResp := doJSON(t, srv, http.MethodGet, "/outfits/"+outfitID, nil, token)
	require.Equal(t, http.StatusOK, getOutfitResp.StatusCode)
	var outfitWithPhoto struct {
		Photos []struct {
			MediaKey string `json:"media_key"`
		} `json:"photos"`
	}
	decodeJSON(t, getOutfitResp, &outfitWithPhoto)
	require.Len(t, outfitWithPhoto.Photos, 1)

	// Step 4: Log wearing the outfit; verify item wear logs are created.
	logResp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfitID+"/logs",
		map[string]any{"worn_on": "2026-03-01"}, token)
	require.Equal(t, http.StatusCreated, logResp.StatusCode)
	var outfitLog struct {
		ID     string `json:"id"`
		WornOn string `json:"worn_on"`
	}
	decodeJSON(t, logResp, &outfitLog)
	require.NotEmpty(t, outfitLog.ID)
	assert.Equal(t, "2026-03-01", outfitLog.WornOn)

	wearLogsItem1 := getItemWearLogs(t, srv, token, item1ID)
	require.Len(t, wearLogsItem1, 1)
	assert.Equal(t, "2026-03-01", wearLogsItem1[0].WornOn)

	wearLogsItem2 := getItemWearLogs(t, srv, token, item2ID)
	require.Len(t, wearLogsItem2, 1)
	assert.Equal(t, "2026-03-01", wearLogsItem2[0].WornOn)

	// Step 5: List outfit logs — entry is present.
	listLogsResp := doJSON(t, srv, http.MethodGet, "/outfits/"+outfitID+"/logs", nil, token)
	require.Equal(t, http.StatusOK, listLogsResp.StatusCode)
	var logList []struct {
		ID string `json:"id"`
	}
	decodeJSON(t, listLogsResp, &logList)
	require.Len(t, logList, 1)
	assert.Equal(t, outfitLog.ID, logList[0].ID)

	// Step 6: Update outfit log date; OutfitLogService propagates the new date to all
	// linked item WearLog rows, so we verify those are updated too.
	updateLogResp := doJSON(t, srv, http.MethodPatch, "/outfits/"+outfitID+"/logs/"+outfitLog.ID,
		map[string]any{"worn_on": "2026-03-02"}, token)
	require.Equal(t, http.StatusOK, updateLogResp.StatusCode)

	wearLogsItem1After := getItemWearLogs(t, srv, token, item1ID)
	require.Len(t, wearLogsItem1After, 1)
	assert.Equal(t, "2026-03-02", wearLogsItem1After[0].WornOn)

	wearLogsItem2After := getItemWearLogs(t, srv, token, item2ID)
	require.Len(t, wearLogsItem2After, 1)
	assert.Equal(t, "2026-03-02", wearLogsItem2After[0].WornOn)

	// Step 7: Delete outfit log; verify linked wear logs deleted.
	deleteLogResp := doJSON(t, srv, http.MethodDelete, "/outfits/"+outfitID+"/logs/"+outfitLog.ID, nil, token)
	require.Equal(t, http.StatusNoContent, deleteLogResp.StatusCode)

	assert.Empty(t, getItemWearLogs(t, srv, token, item1ID))
	assert.Empty(t, getItemWearLogs(t, srv, token, item2ID))

	// Step 8: Delete outfit; verify it is gone.
	deleteOutfitResp := doJSON(t, srv, http.MethodDelete, "/outfits/"+outfitID, nil, token)
	require.Equal(t, http.StatusNoContent, deleteOutfitResp.StatusCode)

	goneResp := doJSON(t, srv, http.MethodGet, "/outfits/"+outfitID, nil, token)
	assert.Equal(t, http.StatusNotFound, goneResp.StatusCode)
}

func TestIntegrationOutfitLogsByDateRangeShouldReturnCorrectLogs(t *testing.T) {
	srv := startIntegrationServer(t)

	token, _ := registerUser(t, srv, "calendaruser", "password-calendar-secure")
	itemID := createItem(t, srv, token, "Calendar Shirt", nil)

	// name is intentionally omitted to verify the field is optional.
	createResp := doJSON(t, srv, http.MethodPost, "/outfits", map[string]any{}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var outfit struct {
		ID string `json:"id"`
	}
	decodeJSON(t, createResp, &outfit)

	addResp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfit.ID+"/items", map[string]any{"item_id": itemID}, token)
	require.Equal(t, http.StatusNoContent, addResp.StatusCode)

	logResp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfit.ID+"/logs", map[string]any{"worn_on": "2026-03-10"}, token)
	require.Equal(t, http.StatusCreated, logResp.StatusCode)
	var outfitLog struct {
		ID string `json:"id"`
	}
	decodeJSON(t, logResp, &outfitLog)

	// Query within range — should return the entry.
	inRangeResp := doJSON(t, srv, http.MethodGet, "/outfit-logs?from=2026-03-01&to=2026-03-31", nil, token)
	require.Equal(t, http.StatusOK, inRangeResp.StatusCode)
	var inRange []struct {
		ID string `json:"id"`
	}
	decodeJSON(t, inRangeResp, &inRange)
	require.Len(t, inRange, 1)
	assert.Equal(t, outfitLog.ID, inRange[0].ID)

	// Query outside range — should return empty.
	outOfRangeResp := doJSON(t, srv, http.MethodGet, "/outfit-logs?from=2026-04-01&to=2026-04-30", nil, token)
	require.Equal(t, http.StatusOK, outOfRangeResp.StatusCode)
	var outOfRange []any
	decodeJSON(t, outOfRangeResp, &outOfRange)
	assert.Empty(t, outOfRange)
}

// --- purchase fields ---

func TestIntegrationItemShouldRejectWhenCreatedWithPurchasePriceButNoCurrency(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchaseval1", "password-purchase-secure")

	resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":           "Test Jacket",
		"purchase_price": "29.99",
	}, token)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestIntegrationItemShouldRejectWhenCreatedWithPurchaseCurrencyButNoPrice(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchaseval2", "password-purchase-secure")

	resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":              "Test Coat",
		"purchase_currency": "USD",
	}, token)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestIntegrationItemShouldRejectWhenCreatedWithNegativePurchasePrice(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchaseval3", "password-purchase-secure")

	resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":              "Test Shirt",
		"purchase_price":    "-5.00",
		"purchase_currency": "EUR",
	}, token)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestIntegrationItemShouldRejectWhenCreatedWithInvalidPurchaseCurrency(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchaseval4", "password-purchase-secure")

	tests := []struct {
		name     string
		currency string
	}{
		{"two letters", "US"},
		{"four letters", "USDD"},
		{"digits", "U5D"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
				"name":              "Test Trousers",
				"purchase_price":    "10.00",
				"purchase_currency": tc.currency,
			}, token)
			assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
		})
	}
}

func TestIntegrationItemShouldRejectWhenCreatedWithFuturePurchaseDate(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchaseval5", "password-purchase-secure")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":              "Future Boots",
		"purchase_price":    "99.00",
		"purchase_currency": "GBP",
		"purchase_date":     futureDate,
	}, token)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestIntegrationItemShouldRejectWhenUpdatedWithFuturePurchaseDate(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchaseval6", "password-purchase-secure")
	itemID := createItem(t, srv, token, "Leather Belt", nil)
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	resp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID, map[string]any{
		"name":              "Leather Belt",
		"purchase_price":    "25.00",
		"purchase_currency": "PLN",
		"purchase_date":     futureDate,
	}, token)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestIntegrationItemShouldPersistAllPurchaseFieldsWhenCreated(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchasehappy1", "password-purchase-secure")

	resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":              "Denim Jacket",
		"purchase_price":    "59.99",
		"purchase_currency": "USD",
		"purchase_date":     "2025-06-15",
		"seller_url":        "https://example.com/jacket",
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &created)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		PurchasePrice    *string `json:"purchase_price"`
		PurchaseCurrency *string `json:"purchase_currency"`
		PurchaseDate     *string `json:"purchase_date"`
		SellerURL        *string `json:"seller_url"`
	}
	decodeJSON(t, getResp, &item)
	require.NotNil(t, item.PurchasePrice)
	assert.Equal(t, "59.99", *item.PurchasePrice)
	require.NotNil(t, item.PurchaseCurrency)
	assert.Equal(t, "USD", *item.PurchaseCurrency)
	require.NotNil(t, item.PurchaseDate)
	assert.Equal(t, "2025-06-15", *item.PurchaseDate)
	require.NotNil(t, item.SellerURL)
	assert.Equal(t, "https://example.com/jacket", *item.SellerURL)
}

func TestIntegrationItemShouldPersistPurchaseFieldsWhenUpdated(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchasehappy2", "password-purchase-secure")
	itemID := createItem(t, srv, token, "Wool Coat", nil)

	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID, map[string]any{
		"name":              "Wool Coat",
		"purchase_price":    "120.00",
		"purchase_currency": "EUR",
		"purchase_date":     "2024-12-01",
		"seller_url":        "https://example.com/coat",
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		PurchasePrice    *string `json:"purchase_price"`
		PurchaseCurrency *string `json:"purchase_currency"`
		PurchaseDate     *string `json:"purchase_date"`
		SellerURL        *string `json:"seller_url"`
	}
	decodeJSON(t, getResp, &item)
	require.NotNil(t, item.PurchasePrice)
	assert.Equal(t, "120.00", *item.PurchasePrice)
	require.NotNil(t, item.PurchaseCurrency)
	assert.Equal(t, "EUR", *item.PurchaseCurrency)
	require.NotNil(t, item.PurchaseDate)
	assert.Equal(t, "2024-12-01", *item.PurchaseDate)
	require.NotNil(t, item.SellerURL)
	assert.Equal(t, "https://example.com/coat", *item.SellerURL)
}

func TestIntegrationItemShouldClearPurchaseFieldsWhenUpdatedWithNulls(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchasehappy3", "password-purchase-secure")

	createResp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":              "Silk Blouse",
		"purchase_price":    "45.00",
		"purchase_currency": "GBP",
		"purchase_date":     "2025-03-10",
		"seller_url":        "https://example.com/blouse",
	}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, createResp, &created)

	// Update with explicit nulls to clear the fields.
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+created.ID, map[string]any{
		"name":              "Silk Blouse",
		"purchase_price":    nil,
		"purchase_currency": nil,
		"purchase_date":     nil,
		"seller_url":        nil,
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		PurchasePrice    *string `json:"purchase_price"`
		PurchaseCurrency *string `json:"purchase_currency"`
		PurchaseDate     *string `json:"purchase_date"`
		SellerURL        *string `json:"seller_url"`
	}
	decodeJSON(t, getResp, &item)
	assert.Nil(t, item.PurchasePrice)
	assert.Nil(t, item.PurchaseCurrency)
	assert.Nil(t, item.PurchaseDate)
	assert.Nil(t, item.SellerURL)
}

func TestIntegrationItemShouldNormaliseCurrencyToUppercaseWhenCreated(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchasehappy4", "password-purchase-secure")

	resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":              "Cotton Tee",
		"purchase_price":    "15.00",
		"purchase_currency": "usd",
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &created)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		PurchaseCurrency *string `json:"purchase_currency"`
	}
	decodeJSON(t, getResp, &item)
	require.NotNil(t, item.PurchaseCurrency)
	assert.Equal(t, "USD", *item.PurchaseCurrency)
}

func TestIntegrationItemShouldAllowSellerURLWithoutPurchaseFields(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchasehappy5", "password-purchase-secure")

	resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":       "Canvas Sneakers",
		"seller_url": "https://example.com/sneakers",
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &created)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		PurchasePrice    *string `json:"purchase_price"`
		PurchaseCurrency *string `json:"purchase_currency"`
		SellerURL        *string `json:"seller_url"`
	}
	decodeJSON(t, getResp, &item)
	assert.Nil(t, item.PurchasePrice)
	assert.Nil(t, item.PurchaseCurrency)
	require.NotNil(t, item.SellerURL)
	assert.Equal(t, "https://example.com/sneakers", *item.SellerURL)
}

func TestIntegrationItemShouldClearSellerURLIndependentlyOfPurchaseFields(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "purchasehappy6", "password-purchase-secure")

	createResp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":              "Oxford Shoes",
		"purchase_price":    "80.00",
		"purchase_currency": "CHF",
		"seller_url":        "https://example.com/shoes",
	}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, createResp, &created)

	// Clear only seller_url while keeping purchase fields intact.
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+created.ID, map[string]any{
		"name":              "Oxford Shoes",
		"purchase_price":    "80.00",
		"purchase_currency": "CHF",
		"seller_url":        nil,
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		PurchasePrice    *string `json:"purchase_price"`
		PurchaseCurrency *string `json:"purchase_currency"`
		SellerURL        *string `json:"seller_url"`
	}
	decodeJSON(t, getResp, &item)
	require.NotNil(t, item.PurchasePrice)
	assert.Equal(t, "80.00", *item.PurchasePrice)
	require.NotNil(t, item.PurchaseCurrency)
	assert.Equal(t, "CHF", *item.PurchaseCurrency)
	assert.Nil(t, item.SellerURL)
}

func TestIntegrationItemPatchPreservesAbsentFields(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "patchpreserve1", "password-patchpreserve-secure")

	createResp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":  "Sneaker",
		"brand": "Nike",
		"color": "Red",
	}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created struct{ ID string `json:"id"` }
	decodeJSON(t, createResp, &created)

	// PATCH with only color — brand should be preserved.
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+created.ID, map[string]any{
		"color": "Blue",
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		Brand *string `json:"brand"`
		Color *string `json:"color"`
	}
	decodeJSON(t, getResp, &item)
	require.NotNil(t, item.Brand)
	assert.Equal(t, "Nike", *item.Brand)
	require.NotNil(t, item.Color)
	assert.Equal(t, "Blue", *item.Color)
}

func TestIntegrationItemPatchClearsNullableField(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "patchclear1", "password-patchclear-secure")

	createResp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":  "Hoodie",
		"brand": "Nike",
	}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created struct{ ID string `json:"id"` }
	decodeJSON(t, createResp, &created)

	// PATCH brand with explicit null — should clear.
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+created.ID, map[string]any{
		"brand": nil,
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		Brand *string `json:"brand"`
	}
	decodeJSON(t, getResp, &item)
	assert.Nil(t, item.Brand)
}

func TestIntegrationItemPatchClearsLocation(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "patchclearloc1", "password-patchclearloc-secure")

	locResp := doJSON(t, srv, http.MethodPost, "/locations", map[string]any{"label": "Wardrobe"}, token)
	require.Equal(t, http.StatusCreated, locResp.StatusCode)
	var loc struct{ ID string `json:"id"` }
	decodeJSON(t, locResp, &loc)

	itemID := createItem(t, srv, token, "Jeans", &loc.ID)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var before struct{ LocationID *string `json:"location_id"` }
	decodeJSON(t, getResp, &before)
	require.NotNil(t, before.LocationID)

	// PATCH with location_id: null — should clear.
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID, map[string]any{
		"location_id": nil,
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp2 := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, token)
	require.Equal(t, http.StatusOK, getResp2.StatusCode)
	var after struct{ LocationID *string `json:"location_id"` }
	decodeJSON(t, getResp2, &after)
	assert.Nil(t, after.LocationID)
}

func TestIntegrationItemPatchClearsCategory(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "patchclearcat1", "password-patchclearcat-secure")

	// Get a valid category ID.
	listResp := doJSON(t, srv, http.MethodGet, "/categories", nil, token)
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	var categories []struct{ ID string `json:"id"` }
	decodeJSON(t, listResp, &categories)
	require.NotEmpty(t, categories)
	catID := categories[0].ID

	createResp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":        "Blazer",
		"category_id": catID,
	}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created struct{ ID string `json:"id"` }
	decodeJSON(t, createResp, &created)

	// PATCH with category_id: null — should clear.
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+created.ID, map[string]any{
		"category_id": nil,
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct{ CategoryID *string `json:"category_id"` }
	decodeJSON(t, getResp, &item)
	assert.Nil(t, item.CategoryID)
}

func TestIntegrationItemPatchClearsPurchaseFields(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "patchclearpurchase1", "password-patchclearpurchase-secure")

	createResp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":              "Cardigan",
		"purchase_price":    "10.00",
		"purchase_currency": "USD",
	}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created struct{ ID string `json:"id"` }
	decodeJSON(t, createResp, &created)

	// PATCH with both purchase fields as null — should clear both.
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+created.ID, map[string]any{
		"purchase_price":    nil,
		"purchase_currency": nil,
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp := doJSON(t, srv, http.MethodGet, "/items/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var item struct {
		PurchasePrice    *string `json:"purchase_price"`
		PurchaseCurrency *string `json:"purchase_currency"`
	}
	decodeJSON(t, getResp, &item)
	assert.Nil(t, item.PurchasePrice)
	assert.Nil(t, item.PurchaseCurrency)
}

func TestIntegrationItemPatchRejectsPurchasePairViolationOnPreservedField(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "patchpurchasepair1", "password-patchpurchasepair-secure")

	createResp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{
		"name":              "Trousers",
		"purchase_price":    "10.00",
		"purchase_currency": "USD",
	}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created struct{ ID string `json:"id"` }
	decodeJSON(t, createResp, &created)

	// PATCH with only currency=null: price is preserved from existing, resulting state
	// has price without currency — must be 422.
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+created.ID, map[string]any{
		"purchase_currency": nil,
	}, token)
	assert.Equal(t, http.StatusUnprocessableEntity, updateResp.StatusCode)
}

func TestIntegrationOutfitPatchPreservesNotes(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "outfitpatchnotes1", "password-outfitpatch-secure")

	createResp := doJSON(t, srv, http.MethodPost, "/outfits", map[string]any{
		"notes": "casual",
	}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created struct{ ID string `json:"id"` }
	decodeJSON(t, createResp, &created)

	// PATCH with only name — notes should be preserved.
	updateResp := doJSON(t, srv, http.MethodPatch, "/outfits/"+created.ID, map[string]any{
		"name": "Summer",
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp := doJSON(t, srv, http.MethodGet, "/outfits/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var outfit struct{ Notes *string `json:"notes"` }
	decodeJSON(t, getResp, &outfit)
	require.NotNil(t, outfit.Notes)
	assert.Equal(t, "casual", *outfit.Notes)
}

func TestIntegrationOutfitPatchClearsNotes(t *testing.T) {
	srv := startIntegrationServer(t)
	token, _ := registerUser(t, srv, "outfitpatchnotes2", "password-outfitpatch-secure")

	createResp := doJSON(t, srv, http.MethodPost, "/outfits", map[string]any{
		"notes": "casual",
	}, token)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created struct{ ID string `json:"id"` }
	decodeJSON(t, createResp, &created)

	// PATCH notes with explicit null — should clear.
	updateResp := doJSON(t, srv, http.MethodPatch, "/outfits/"+created.ID, map[string]any{
		"notes": nil,
	}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp := doJSON(t, srv, http.MethodGet, "/outfits/"+created.ID, nil, token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var outfit struct{ Notes *string `json:"notes"` }
	decodeJSON(t, getResp, &outfit)
	assert.Nil(t, outfit.Notes)
}

// createOutfit creates an outfit and returns its ID.
func createOutfit(t *testing.T, srv *httptest.Server, token string) string {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/outfits", map[string]any{}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.ID)
	return result.ID
}

// createShare creates a share and returns its ID.
func createShare(t *testing.T, srv *httptest.Server, token, recipientID, targetType, targetID string) string {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/shares", map[string]any{
		"recipient_id": recipientID,
		"target_type":  targetType,
		"target_id":    targetID,
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.ID)
	return result.ID
}

// getUserID registers a user and returns their ID by decoding the register response.
func getUserID(t *testing.T, srv *httptest.Server, token string) string {
	t.Helper()
	resp := doJSON(t, srv, http.MethodGet, "/users", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var users []struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	decodeJSON(t, resp, &users)
	require.NotEmpty(t, users)
	return users[len(users)-1].ID
}

func TestIntegrationSharingLifecycle(t *testing.T) {
	srv := startIntegrationServer(t)

	// Step 1: Register two users. User A is admin (first registrant), User B is member.
	tokenA, _ := registerUser(t, srv, "share-alice", "password-alice-secure")
	enableRegistration(t, srv, tokenA)
	tokenB, _ := registerUser(t, srv, "share-bob", "password-bob-secure")

	// Resolve User IDs via GET /users (as User B to verify the endpoint works).
	listUsersResp := doJSON(t, srv, http.MethodGet, "/users", nil, tokenB)
	require.Equal(t, http.StatusOK, listUsersResp.StatusCode)
	var allUsers []struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	decodeJSON(t, listUsersResp, &allUsers)

	// Step 3: User B calls GET /users — both users are listed.
	require.Len(t, allUsers, 2)
	var userAID, userBID string
	for _, u := range allUsers {
		if u.Email == "share-alice" {
			userAID = u.ID
		} else if u.Email == "share-bob" {
			userBID = u.ID
		}
	}
	require.NotEmpty(t, userAID, "User A ID not found in users list")
	require.NotEmpty(t, userBID, "User B ID not found in users list")

	// Step 2: User A creates an item, an outfit, and a location with items assigned.
	parentLocID := createLocation(t, srv, tokenA, "Wardrobe")
	childLocID := createLocation(t, srv, tokenA, "Wardrobe Shelf")
	// Assign child to parent.
	assignResp := doJSON(t, srv, http.MethodPatch, "/locations/"+childLocID+"/move",
		map[string]any{"parent_id": parentLocID}, tokenA)
	require.Equal(t, http.StatusOK, assignResp.StatusCode)

	// Items: one in the parent location, one in the child location.
	itemID := createItem(t, srv, tokenA, "Blue Jacket", &parentLocID)
	childItemID := createItem(t, srv, tokenA, "Scarf", &childLocID)
	outfitID := createOutfit(t, srv, tokenA)

	// User A logs a wear entry for the item.
	wearLogResp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/wear-logs",
		map[string]any{"worn_on": "2026-03-01"}, tokenA)
	require.Equal(t, http.StatusCreated, wearLogResp.StatusCode)

	// User A logs a wear entry for the outfit.
	outfitLogResp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfitID+"/logs",
		map[string]any{"worn_on": "2026-03-05"}, tokenA)
	require.Equal(t, http.StatusCreated, outfitLogResp.StatusCode)

	// Step 4: User B cannot access User A's item — 403.
	forbidItemResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, tokenB)
	require.Equal(t, http.StatusForbidden, forbidItemResp.StatusCode)

	// Step 5: User A shares the item with User B.
	itemShareID := createShare(t, srv, tokenA, userBID, "item", itemID)

	// Step 6: User B can now access User A's item — 200.
	getItemResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, tokenB)
	require.Equal(t, http.StatusOK, getItemResp.StatusCode)

	// Step 7: User B can view wear logs for the shared item.
	wearLogs := getItemWearLogs(t, srv, tokenB, itemID)
	require.Len(t, wearLogs, 1)
	assert.Equal(t, "2026-03-01", wearLogs[0].WornOn)

	// Step 8: User B cannot update the shared item — 403.
	updateItemResp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID,
		map[string]any{"name": "Red Jacket"}, tokenB)
	assert.Equal(t, http.StatusForbidden, updateItemResp.StatusCode)

	// Step 9: User B cannot delete the shared item — 403.
	deleteItemResp := doJSON(t, srv, http.MethodDelete, "/items/"+itemID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, deleteItemResp.StatusCode)

	// Step 10: User B calls GET /shares/with-me — the item appears with shared_by populated.
	sharedWithMeResp := doJSON(t, srv, http.MethodGet, "/shares/with-me", nil, tokenB)
	require.Equal(t, http.StatusOK, sharedWithMeResp.StatusCode)
	var sharedWithMe struct {
		Items []struct {
			ID       string `json:"id"`
			SharedBy struct {
				ID string `json:"id"`
			} `json:"shared_by"`
		} `json:"items"`
		Outfits   []any `json:"outfits"`
		Locations []any `json:"locations"`
	}
	decodeJSON(t, sharedWithMeResp, &sharedWithMe)
	require.Len(t, sharedWithMe.Items, 1)
	assert.Equal(t, itemID, sharedWithMe.Items[0].ID)
	assert.Equal(t, userAID, sharedWithMe.Items[0].SharedBy.ID)

	// Step 11: User A shares the outfit with User B.
	createShare(t, srv, tokenA, userBID, "outfit", outfitID)

	// Step 12: User B can access the outfit and its logs.
	getOutfitResp := doJSON(t, srv, http.MethodGet, "/outfits/"+outfitID, nil, tokenB)
	require.Equal(t, http.StatusOK, getOutfitResp.StatusCode)

	outfitLogsResp := doJSON(t, srv, http.MethodGet, "/outfits/"+outfitID+"/logs", nil, tokenB)
	require.Equal(t, http.StatusOK, outfitLogsResp.StatusCode)
	var outfitLogs []struct {
		ID     string `json:"id"`
		WornOn string `json:"worn_on"`
	}
	decodeJSON(t, outfitLogsResp, &outfitLogs)
	require.Len(t, outfitLogs, 1)
	assert.Equal(t, "2026-03-05", outfitLogs[0].WornOn)

	// Step 13: User A shares the parent location with User B.
	createShare(t, srv, tokenA, userBID, "location", parentLocID)

	// Step 14: User B can access the location and all items assigned to it.
	getLocResp := doJSON(t, srv, http.MethodGet, "/locations/"+parentLocID, nil, tokenB)
	require.Equal(t, http.StatusOK, getLocResp.StatusCode)

	// The item in the parent location is accessible via the location share.
	// (Already accessible via item share, but verifying location path works.)
	getItemViaLocResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, tokenB)
	assert.Equal(t, http.StatusOK, getItemViaLocResp.StatusCode)

	// Step 15: User B can access a child location (inherited access) and its items.
	getChildLocResp := doJSON(t, srv, http.MethodGet, "/locations/"+childLocID, nil, tokenB)
	assert.Equal(t, http.StatusOK, getChildLocResp.StatusCode)

	getChildItemResp := doJSON(t, srv, http.MethodGet, "/items/"+childItemID, nil, tokenB)
	assert.Equal(t, http.StatusOK, getChildItemResp.StatusCode)

	// Step 16: User A revokes the item share — User B can no longer access it directly.
	// (User B still has access via location share, so revoke location share first to test item-only access.)
	// Revoke only the item share to verify that access falls through to location share.
	revokeItemShareResp := doJSON(t, srv, http.MethodDelete, "/shares/"+itemShareID, nil, tokenA)
	require.Equal(t, http.StatusNoContent, revokeItemShareResp.StatusCode)

	// Item is still accessible via location share.
	getItemAfterRevokeResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, tokenB)
	assert.Equal(t, http.StatusOK, getItemAfterRevokeResp.StatusCode)

	// Now revoke the location share — item should no longer be accessible.
	locShares := listOutgoingShares(t, srv, tokenA)
	var locShareID string
	for _, s := range locShares {
		if s.TargetID == parentLocID {
			locShareID = s.ID
			break
		}
	}
	require.NotEmpty(t, locShareID)

	revokeLocShareResp := doJSON(t, srv, http.MethodDelete, "/shares/"+locShareID, nil, tokenA)
	require.Equal(t, http.StatusNoContent, revokeLocShareResp.StatusCode)

	// Now User B cannot access the item.
	getItemGoneResp := doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, getItemGoneResp.StatusCode)

	// Step 17: User A deletes the shared outfit — shares are cleaned up (no orphaned shares).
	deleteOutfitResp := doJSON(t, srv, http.MethodDelete, "/outfits/"+outfitID, nil, tokenA)
	require.Equal(t, http.StatusNoContent, deleteOutfitResp.StatusCode)

	// Outfit no longer appears in shared-with-me.
	sharedAfterDeleteResp := doJSON(t, srv, http.MethodGet, "/shares/with-me", nil, tokenB)
	require.Equal(t, http.StatusOK, sharedAfterDeleteResp.StatusCode)
	var sharedAfterDelete struct {
		Outfits []any `json:"outfits"`
	}
	decodeJSON(t, sharedAfterDeleteResp, &sharedAfterDelete)
	assert.Empty(t, sharedAfterDelete.Outfits)

	// Step 18: User A lists outgoing shares — sees remaining shares (none, all revoked/deleted).
	remainingShares := listOutgoingShares(t, srv, tokenA)
	assert.Empty(t, remainingShares)
}

// outgoingShareEntry is used to decode entries from GET /shares.
type outgoingShareEntry struct {
	ID         string `json:"id"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

// listOutgoingShares calls GET /shares and returns the decoded list.
func listOutgoingShares(t *testing.T, srv *httptest.Server, token string) []outgoingShareEntry {
	t.Helper()
	resp := doJSON(t, srv, http.MethodGet, "/shares", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var shares []outgoingShareEntry
	decodeJSON(t, resp, &shares)
	return shares
}

func TestIntegrationShareEdgeCases(t *testing.T) {
	srv := startIntegrationServer(t)

	tokenA, _ := registerUser(t, srv, "share-edge-alice", "password-alice-secure")
	enableRegistration(t, srv, tokenA)
	tokenB, _ := registerUser(t, srv, "share-edge-bob", "password-bob-secure")

	// Resolve User B's ID.
	users := listUsersAs(t, srv, tokenA)
	var userAID, userBID string
	for _, u := range users {
		if u.Email == "share-edge-alice" {
			userAID = u.ID
		} else if u.Email == "share-edge-bob" {
			userBID = u.ID
		}
	}
	require.NotEmpty(t, userAID)
	require.NotEmpty(t, userBID)

	itemID := createItem(t, srv, tokenA, "Vintage Hat", nil)

	// Step 19: Duplicate share creation returns 409.
	createShare(t, srv, tokenA, userBID, "item", itemID)
	dupResp := doJSON(t, srv, http.MethodPost, "/shares", map[string]any{
		"recipient_id": userBID,
		"target_type":  "item",
		"target_id":    itemID,
	}, tokenA)
	assert.Equal(t, http.StatusConflict, dupResp.StatusCode)

	// Step 20: Self-share returns 422.
	selfResp := doJSON(t, srv, http.MethodPost, "/shares", map[string]any{
		"recipient_id": userAID,
		"target_type":  "item",
		"target_id":    itemID,
	}, tokenA)
	assert.Equal(t, http.StatusUnprocessableEntity, selfResp.StatusCode)

	// Step 21: Share with non-existent recipient returns 404.
	nonExistentUserResp := doJSON(t, srv, http.MethodPost, "/shares", map[string]any{
		"recipient_id": "00000000-0000-0000-0000-000000000000",
		"target_type":  "item",
		"target_id":    itemID,
	}, tokenA)
	assert.Equal(t, http.StatusNotFound, nonExistentUserResp.StatusCode)

	// Step 22: Share with non-existent target returns 404.
	nonExistentTargetResp := doJSON(t, srv, http.MethodPost, "/shares", map[string]any{
		"recipient_id": userBID,
		"target_type":  "item",
		"target_id":    "00000000-0000-0000-0000-000000000000",
	}, tokenA)
	assert.Equal(t, http.StatusNotFound, nonExistentTargetResp.StatusCode)

	// Step 23: Non-owner cannot revoke a share — returns 403.
	shares := listOutgoingShares(t, srv, tokenA)
	require.NotEmpty(t, shares)
	shareID := shares[0].ID
	revokeResp := doJSON(t, srv, http.MethodDelete, "/shares/"+shareID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, revokeResp.StatusCode)
}

type userEntry struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func listUsersAs(t *testing.T, srv *httptest.Server, token string) []userEntry {
	t.Helper()
	resp := doJSON(t, srv, http.MethodGet, "/users", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var users []userEntry
	decodeJSON(t, resp, &users)
	return users
}

func TestIntegrationCrossUserOutfitAccessForbidden(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenA, _ := registerUser(t, srv, "outfit-forbid-alice", "password-alice-secure")
	enableRegistration(t, srv, tokenA)
	tokenB, _ := registerUser(t, srv, "outfit-forbid-bob", "password-bob-secure")

	createResp := doJSON(t, srv, http.MethodPost, "/outfits", map[string]any{}, tokenA)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var outfit struct {
		ID string `json:"id"`
	}
	decodeJSON(t, createResp, &outfit)
	outfitID := outfit.ID

	itemID := createItem(t, srv, tokenA, "Sneakers", nil)

	logResp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfitID+"/logs",
		map[string]any{"worn_on": "2026-03-01"}, tokenA)
	require.Equal(t, http.StatusCreated, logResp.StatusCode)
	var outfitLog struct {
		ID string `json:"id"`
	}
	decodeJSON(t, logResp, &outfitLog)
	logID := outfitLog.ID

	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"get outfit", http.MethodGet, "/outfits/" + outfitID, nil},
		{"post outfit log", http.MethodPost, "/outfits/" + outfitID + "/logs", map[string]any{"worn_on": "2026-03-01"}},
		{"add item to outfit", http.MethodPost, "/outfits/" + outfitID + "/items", map[string]any{"item_id": itemID}},
		{"patch outfit log", http.MethodPatch, "/outfits/" + outfitID + "/logs/" + logID, map[string]any{"worn_on": "2026-03-05"}},
		{"delete outfit log", http.MethodDelete, "/outfits/" + outfitID + "/logs/" + logID, nil},
		{"delete outfit", http.MethodDelete, "/outfits/" + outfitID, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, srv, tc.method, tc.path, tc.body, tokenB)
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}
