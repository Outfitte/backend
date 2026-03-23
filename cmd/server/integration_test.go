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
			MediaKey string
		}
	}
	decodeJSON(t, getResp, &item)
	require.Len(t, item.Photos, 1)
	return item.Photos[0].MediaKey
}

// --- unauthorized cases ---

func TestIntegrationShouldReturn401WhenGetItemsCalledWithoutToken(t *testing.T) {
	srv := startIntegrationServer(t)
	resp := doJSON(t, srv, http.MethodGet, "/items", nil, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestIntegrationShouldReturn401WhenGetLocationsCalledWithoutToken(t *testing.T) {
	srv := startIntegrationServer(t)
	resp := doJSON(t, srv, http.MethodGet, "/locations", nil, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestIntegrationShouldReturn401WhenPostItemsCalledWithoutToken(t *testing.T) {
	srv := startIntegrationServer(t)
	resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{"name": "jacket"}, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// enableRegistration calls PATCH /admin/settings to allow new registrations.
func enableRegistration(t *testing.T, srv *httptest.Server, adminToken string) {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPatch, "/admin/settings",
		map[string]any{"registration_enabled": true}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
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
		LocationID *string `json:"LocationID"`
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

// User B cannot delete User A's location.
func TestIntegrationShouldReturn403WhenUserBDeletesUserALocation(t *testing.T) {
	srv := startIntegrationServer(t)

	tokenA, _ := registerUser(t, srv, "alice2", "password-alice-secure")
	enableRegistration(t, srv, tokenA)
	tokenB, _ := registerUser(t, srv, "bob2", "password-bob-secure")

	locID := createLocation(t, srv, tokenA, "Wardrobe")

	resp := doJSON(t, srv, http.MethodDelete, "/locations/"+locID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// User B cannot delete User A's item.
func TestIntegrationShouldReturn403WhenUserBDeletesUserAItem(t *testing.T) {
	srv := startIntegrationServer(t)

	tokenA, _ := registerUser(t, srv, "alice3", "password-alice-secure")
	enableRegistration(t, srv, tokenA)
	tokenB, _ := registerUser(t, srv, "bob3", "password-bob-secure")

	itemID := createItem(t, srv, tokenA, "Red Dress", nil)

	resp := doJSON(t, srv, http.MethodDelete, "/items/"+itemID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}
