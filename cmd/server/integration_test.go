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

	"github.com/outfitte/outfitte/internal/config"
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
		{"delete location", http.MethodDelete, "/locations/" + locID, nil},
		{"delete item", http.MethodDelete, "/items/" + itemID, nil},
		{"archive item", http.MethodPost, "/items/" + itemID + "/archive", nil},
		{"dispose item", http.MethodPost, "/items/" + itemID + "/dispose", map[string]any{"reason": "donated"}},
		{"log wear for item", http.MethodPost, "/items/" + itemID + "/wear-logs", map[string]any{"worn_on": "2026-01-15"}},
		{"delete wear log", http.MethodDelete, "/items/" + itemID + "/wear-logs/" + wearEntry.ID, nil},
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
		{"delete outfit", http.MethodDelete, "/outfits/" + outfitID, nil},
		{"post outfit log", http.MethodPost, "/outfits/" + outfitID + "/logs", map[string]any{"worn_on": "2026-03-01"}},
		{"add item to outfit", http.MethodPost, "/outfits/" + outfitID + "/items", map[string]any{"item_id": itemID}},
		{"patch outfit log", http.MethodPatch, "/outfits/" + outfitID + "/logs/" + logID, map[string]any{"worn_on": "2026-03-05"}},
		{"delete outfit log", http.MethodDelete, "/outfits/" + outfitID + "/logs/" + logID, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, srv, tc.method, tc.path, tc.body, tokenB)
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}
