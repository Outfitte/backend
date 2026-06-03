package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	localmedia "github.com/outfitte/backend/internal/adapter/media/local"
	"github.com/outfitte/backend/internal/adapter/store"
	"github.com/outfitte/backend/internal/api/server"
	"github.com/outfitte/backend/internal/config"
)

// ─── Server setup ────────────────────────────────────────────────────────────

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
	cfg := newIntegrationConfig(t)
	repos, closer, err := store.NewRepositories(t.Context(), *cfg)
	require.NoError(t, err)
	t.Cleanup(func() { closer.Close() }) //nolint:errcheck
	media := localmedia.NewProvider(cfg.MediaStoragePath)
	srv := server.New(cfg, slog.New(slog.DiscardHandler), repos, media)
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)
	return ts
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

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

// ─── Domain helpers ───────────────────────────────────────────────────────────

func registerUser(t *testing.T, srv *httptest.Server, username, password string) string {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/auth/register", map[string]string{
		"email":    username,
		"password": password,
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		AccessToken string `json:"access_token"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.AccessToken)
	return result.AccessToken
}

func enableRegistration(t *testing.T, srv *httptest.Server, adminToken string) {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPatch, "/admin/settings",
		map[string]any{"registration_enabled": true}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func createItem(t *testing.T, srv *httptest.Server, token, name string) string {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/items", map[string]any{"name": name}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.ID)
	return result.ID
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

func assignLocation(t *testing.T, srv *httptest.Server, token, itemID, locationID string) {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID+"/location",
		map[string]any{"location_id": locationID}, token)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func logWear(t *testing.T, srv *httptest.Server, token, itemID, wornOn string) string {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/wear-logs",
		map[string]any{"worn_on": wornOn}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.ID)
	return result.ID
}

func listWearLogs(t *testing.T, srv *httptest.Server, token, itemID string) []map[string]any {
	t.Helper()
	resp := doJSON(t, srv, http.MethodGet, "/items/"+itemID+"/wear-logs", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var logs []map[string]any
	decodeJSON(t, resp, &logs)
	return logs
}

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

type userEntry struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func listUsers(t *testing.T, srv *httptest.Server, token string) []userEntry {
	t.Helper()
	resp := doJSON(t, srv, http.MethodGet, "/users", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var users []userEntry
	decodeJSON(t, resp, &users)
	return users
}

func findUserID(t *testing.T, users []userEntry, email string) string {
	t.Helper()
	for _, u := range users {
		if u.Email == email {
			return u.ID
		}
	}
	t.Fatalf("user with email %q not found", email)
	return ""
}

func createTransfer(t *testing.T, srv *httptest.Server, token, itemID, recipientID string, transferHistory bool) string {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/transfers", map[string]any{
		"item_id":          itemID,
		"recipient_id":     recipientID,
		"transfer_history": transferHistory,
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.ID)
	return result.ID
}

func acceptTransfer(t *testing.T, srv *httptest.Server, token, transferID string) map[string]any {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/transfers/"+transferID+"/accept", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result map[string]any
	decodeJSON(t, resp, &result)
	return result
}

func rejectTransfer(t *testing.T, srv *httptest.Server, token, transferID string) map[string]any {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/transfers/"+transferID+"/reject", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result map[string]any
	decodeJSON(t, resp, &result)
	return result
}

func cancelTransfer(t *testing.T, srv *httptest.Server, token, transferID string) map[string]any {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/transfers/"+transferID+"/cancel", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result map[string]any
	decodeJSON(t, resp, &result)
	return result
}

func getTransfer(t *testing.T, srv *httptest.Server, token, transferID string) *http.Response {
	t.Helper()
	return doJSON(t, srv, http.MethodGet, "/transfers/"+transferID, nil, token)
}

func listIncomingTransfers(t *testing.T, srv *httptest.Server, token string) []map[string]any {
	t.Helper()
	resp := doJSON(t, srv, http.MethodGet, "/transfers/incoming", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var transfers []map[string]any
	decodeJSON(t, resp, &transfers)
	return transfers
}

func getItem(t *testing.T, srv *httptest.Server, token, itemID string) *http.Response {
	t.Helper()
	return doJSON(t, srv, http.MethodGet, "/items/"+itemID, nil, token)
}

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

func addItemToOutfit(t *testing.T, srv *httptest.Server, token, outfitID, itemID string) {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfitID+"/items",
		map[string]any{"item_id": itemID}, token)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func logOutfit(t *testing.T, srv *httptest.Server, token, outfitID, wornOn string) string {
	t.Helper()
	resp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfitID+"/logs",
		map[string]any{"worn_on": wornOn}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID string `json:"id"`
	}
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.ID)
	return result.ID
}

func getOutfit(t *testing.T, srv *httptest.Server, token, outfitID string) map[string]any {
	t.Helper()
	resp := doJSON(t, srv, http.MethodGet, "/outfits/"+outfitID, nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result map[string]any
	decodeJSON(t, resp, &result)
	return result
}

func listOutfitLogs(t *testing.T, srv *httptest.Server, token, outfitID string) []map[string]any {
	t.Helper()
	resp := doJSON(t, srv, http.MethodGet, "/outfits/"+outfitID+"/logs", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var logs []map[string]any
	decodeJSON(t, resp, &logs)
	return logs
}

// setupThreeUsers registers alice (admin), enables registration, then registers bob and carol.
// Returns (tokenAlice, tokenBob, tokenCarol, aliceID, bobID, carolID).
func setupThreeUsers(t *testing.T, srv *httptest.Server) (tokenAlice, tokenBob, tokenCarol, aliceID, bobID, carolID string) {
	t.Helper()
	tokenAlice = registerUser(t, srv, "alice", "password-alice-secure")
	enableRegistration(t, srv, tokenAlice)
	tokenBob = registerUser(t, srv, "bob", "password-bob-secure")
	tokenCarol = registerUser(t, srv, "carol", "password-carol-secure")

	users := listUsers(t, srv, tokenAlice)
	aliceID = findUserID(t, users, "alice")
	bobID = findUserID(t, users, "bob")
	carolID = findUserID(t, users, "carol")
	return
}

// ─── Tests ─────────────────────────────────────────────────────────────────────
//
// Error / rejection cases come first (scenarios 4–10), happy paths last (1–3).

// TestItemTransferShouldReturn422WhenSelfTransfer covers scenario 9:
// Alice attempts to transfer an item to herself → 422.
func TestItemTransferShouldReturn422WhenSelfTransfer(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, _, _, aliceID, _, _ := setupThreeUsers(t, srv)

	itemID := createItem(t, srv, tokenAlice, "Self-Transfer Item")

	resp := doJSON(t, srv, http.MethodPost, "/transfers", map[string]any{
		"item_id":      itemID,
		"recipient_id": aliceID,
	}, tokenAlice)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	var body map[string]string
	decodeJSON(t, resp, &body)
	assert.Equal(t, "cannot transfer to yourself", body["error"])
}

// TestItemTransferShouldReturn422WhenItemIsArchived covers scenario 10:
// Alice archives an item, then attempts to transfer it → 422.
func TestItemTransferShouldReturn422WhenItemIsArchived(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, _, _, _, bobID, _ := setupThreeUsers(t, srv)

	itemID := createItem(t, srv, tokenAlice, "Archived Item")

	// Archive the item.
	archiveResp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/archive", nil, tokenAlice)
	require.Equal(t, http.StatusNoContent, archiveResp.StatusCode)

	// Attempt transfer → 422.
	resp := doJSON(t, srv, http.MethodPost, "/transfers", map[string]any{
		"item_id":      itemID,
		"recipient_id": bobID,
	}, tokenAlice)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

// TestItemTransferShouldReturn409WhenSecondTransferInitiatedWhilePending covers scenario 7:
// Alice initiates a transfer; while it is pending she attempts a second transfer to Dave → 409.
func TestItemTransferShouldReturn409WhenSecondTransferInitiatedWhilePending(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, _, _, _, bobID, _ := setupThreeUsers(t, srv)

	// Register Dave (4th user) and resolve his ID.
	registerUser(t, srv, "dave", "password-dave-secure")
	daveID := findUserID(t, listUsers(t, srv, tokenAlice), "dave")

	itemID := createItem(t, srv, tokenAlice, "Contested Item")

	// First transfer to Bob.
	createTransfer(t, srv, tokenAlice, itemID, bobID, false)

	// Second transfer attempt to Dave — must return 409.
	resp := doJSON(t, srv, http.MethodPost, "/transfers", map[string]any{
		"item_id":      itemID,
		"recipient_id": daveID,
	}, tokenAlice)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	var body map[string]string
	decodeJSON(t, resp, &body)
	assert.Equal(t, "item has a pending transfer", body["error"])
}

// TestItemTransferShouldReturn403WhenUninvolvedPartyOrWrongRole covers scenario 8:
// - Carol (uninvolved) attempts GET /transfers/{id} → 403
// - Bob (recipient) attempts cancel (sender-only) → 403
// - Alice (sender) attempts accept (recipient-only) → 403
func TestItemTransferShouldReturn403WhenUninvolvedPartyOrWrongRole(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, tokenBob, tokenCarol, _, bobID, _ := setupThreeUsers(t, srv)

	itemID := createItem(t, srv, tokenAlice, "Role-Tested Item")
	transferID := createTransfer(t, srv, tokenAlice, itemID, bobID, false)

	t.Run("carol cannot get transfer", func(t *testing.T) {
		resp := getTransfer(t, srv, tokenCarol, transferID)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("bob cannot cancel (sender-only)", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodPost, "/transfers/"+transferID+"/cancel", nil, tokenBob)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("alice cannot accept (recipient-only)", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodPost, "/transfers/"+transferID+"/accept", nil, tokenAlice)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

// TestItemTransferShouldBlockMutationsWhenPending covers scenario 6:
// While a transfer is pending, all write operations on the item return 409 with
// "item has a pending transfer". Read operations continue to work for both Alice and Bob.
func TestItemTransferShouldBlockMutationsWhenPending(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, tokenBob, _, _, bobID, carolID := setupThreeUsers(t, srv)

	itemID := createItem(t, srv, tokenAlice, "Blocked Item")
	locID := createLocation(t, srv, tokenAlice, "Wardrobe")
	outfitID := createOutfit(t, srv, tokenAlice)

	// Share item with Carol before initiating transfer.
	createShare(t, srv, tokenAlice, carolID, "item", itemID)

	// Initiate transfer — item is now locked.
	createTransfer(t, srv, tokenAlice, itemID, bobID, false)

	const pendingErr = "item has a pending transfer"

	t.Run("update blocked", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID,
			map[string]any{"name": "New Name"}, tokenAlice)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		var body map[string]string
		decodeJSON(t, resp, &body)
		assert.Equal(t, pendingErr, body["error"])
	})

	t.Run("archive blocked", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/archive", nil, tokenAlice)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		var body map[string]string
		decodeJSON(t, resp, &body)
		assert.Equal(t, pendingErr, body["error"])
	})

	t.Run("assign location blocked", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID+"/location",
			map[string]any{"location_id": locID}, tokenAlice)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		var body map[string]string
		decodeJSON(t, resp, &body)
		assert.Equal(t, pendingErr, body["error"])
	})

	t.Run("delete blocked", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodDelete, "/items/"+itemID, nil, tokenAlice)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		var body map[string]string
		decodeJSON(t, resp, &body)
		assert.Equal(t, pendingErr, body["error"])
	})

	t.Run("add photo blocked", func(t *testing.T) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, err := mw.CreateFormFile("photo", "dummy.jpg")
		require.NoError(t, err)
		_, err = fw.Write([]byte("fake"))
		require.NoError(t, err)
		require.NoError(t, mw.Close())

		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
			srv.URL+"/items/"+itemID+"/photos", &buf)
		require.NoError(t, err)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+tokenAlice)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		var body map[string]string
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, pendingErr, body["error"])
	})

	t.Run("add to outfit blocked", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodPost, "/outfits/"+outfitID+"/items",
			map[string]any{"item_id": itemID}, tokenAlice)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		var body map[string]string
		decodeJSON(t, resp, &body)
		assert.Equal(t, pendingErr, body["error"])
	})

	t.Run("log wear blocked", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodPost, "/items/"+itemID+"/wear-logs",
			map[string]any{"worn_on": "2026-04-01"}, tokenAlice)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		var body map[string]string
		decodeJSON(t, resp, &body)
		assert.Equal(t, pendingErr, body["error"])
	})

	t.Run("share blocked", func(t *testing.T) {
		// Register Dave and resolve his ID for use as share recipient.
		registerUser(t, srv, "dave", "password-dave-secure")
		daveID := findUserID(t, listUsers(t, srv, tokenAlice), "dave")

		resp := doJSON(t, srv, http.MethodPost, "/shares", map[string]any{
			"recipient_id": daveID,
			"target_type":  "item",
			"target_id":    itemID,
		}, tokenAlice)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		var body map[string]string
		decodeJSON(t, resp, &body)
		assert.Equal(t, pendingErr, body["error"])
	})

	t.Run("read still works for Alice", func(t *testing.T) {
		resp := getItem(t, srv, tokenAlice, itemID)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Bob is the pending recipient; he cannot GET /items/{id} directly (no ownership/share),
	// but he can still read the pending transfer itself via GET /transfers/incoming.
	t.Run("incoming transfer still readable for Bob", func(t *testing.T) {
		pending := listIncomingTransfers(t, srv, tokenBob)
		require.Len(t, pending, 1)
		assert.Equal(t, "pending", pending[0]["status"])
	})
}

// TestItemTransferShouldUnlockItemWhenRejected covers scenario 4:
// Bob rejects the transfer. Item still belongs to Alice, status=rejected, decided_at set,
// and Alice can mutate the item again (item is unlocked).
func TestItemTransferShouldUnlockItemWhenRejected(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, tokenBob, _, _, bobID, _ := setupThreeUsers(t, srv)

	itemID := createItem(t, srv, tokenAlice, "Yellow Scarf")
	transferID := createTransfer(t, srv, tokenAlice, itemID, bobID, false)

	result := rejectTransfer(t, srv, tokenBob, transferID)
	assert.Equal(t, "rejected", result["status"])
	assert.NotNil(t, result["decided_at"])

	// Item still belongs to Alice.
	aliceGetResp := getItem(t, srv, tokenAlice, itemID)
	require.Equal(t, http.StatusOK, aliceGetResp.StatusCode)
	var itemBody map[string]any
	decodeJSON(t, aliceGetResp, &itemBody)
	assert.Equal(t, "active", itemBody["status"])

	// Alice can update the item again (lock was released).
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID,
		map[string]any{"name": "Yellow Scarf Updated"}, tokenAlice)
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
}

// TestItemTransferShouldUnlockItemWhenCancelled covers scenario 5:
// Alice cancels the transfer. Item still belongs to Alice, status=cancelled, decided_at set,
// and Alice can mutate the item again.
func TestItemTransferShouldUnlockItemWhenCancelled(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, _, _, _, bobID, _ := setupThreeUsers(t, srv)

	itemID := createItem(t, srv, tokenAlice, "Purple Hat")
	transferID := createTransfer(t, srv, tokenAlice, itemID, bobID, false)

	result := cancelTransfer(t, srv, tokenAlice, transferID)
	assert.Equal(t, "cancelled", result["status"])
	assert.NotNil(t, result["decided_at"])

	// Item still belongs to Alice.
	aliceGetResp := getItem(t, srv, tokenAlice, itemID)
	require.Equal(t, http.StatusOK, aliceGetResp.StatusCode)
	var itemBody map[string]any
	decodeJSON(t, aliceGetResp, &itemBody)
	assert.Equal(t, "active", itemBody["status"])

	// Alice can update the item again (lock was released).
	updateResp := doJSON(t, srv, http.MethodPatch, "/items/"+itemID,
		map[string]any{"name": "Purple Hat Updated"}, tokenAlice)
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
}

// ─── Happy path tests ─────────────────────────────────────────────────────────

// TestItemTransferShouldTransferOwnershipWithHistoryWhenAccepted covers scenario 1:
// Alice creates an item with a wear log, shares it with Carol, then transfers to Bob
// with transfer_history=true. After Bob accepts: Bob owns the item, location is null,
// wear log owner is Bob, Alice's share to Carol is gone, transfer is accepted with decided_at set.
func TestItemTransferShouldTransferOwnershipWithHistoryWhenAccepted(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, tokenBob, tokenCarol, _, bobID, carolID := setupThreeUsers(t, srv)

	// Alice creates an item with a location and logs a wear entry.
	locID := createLocation(t, srv, tokenAlice, "Wardrobe")
	itemID := createItem(t, srv, tokenAlice, "Blue Jacket")
	assignLocation(t, srv, tokenAlice, itemID, locID)
	logWear(t, srv, tokenAlice, itemID, "2026-01-10")

	// Alice shares the item with Carol.
	createShare(t, srv, tokenAlice, carolID, "item", itemID)

	// Carol can access the item.
	carolGetResp := getItem(t, srv, tokenCarol, itemID)
	require.Equal(t, http.StatusOK, carolGetResp.StatusCode)

	// Alice initiates transfer to Bob with history.
	transferID := createTransfer(t, srv, tokenAlice, itemID, bobID, true)

	// Bob lists incoming transfers — sees the pending one.
	incoming := listIncomingTransfers(t, srv, tokenBob)
	require.Len(t, incoming, 1)
	assert.Equal(t, transferID, incoming[0]["id"])
	assert.Equal(t, "pending", incoming[0]["status"])

	// Bob accepts.
	result := acceptTransfer(t, srv, tokenBob, transferID)
	assert.Equal(t, "accepted", result["status"])
	assert.NotNil(t, result["decided_at"])

	// Bob now owns the item.
	bobGetResp := getItem(t, srv, tokenBob, itemID)
	require.Equal(t, http.StatusOK, bobGetResp.StatusCode)
	var itemBody map[string]any
	decodeJSON(t, bobGetResp, &itemBody)
	assert.Equal(t, bobID, itemBody["owner_id"])
	assert.Nil(t, itemBody["location_id"])

	// Alice no longer owns the item — 403.
	aliceGetResp := getItem(t, srv, tokenAlice, itemID)
	assert.Equal(t, http.StatusForbidden, aliceGetResp.StatusCode)

	// Wear log owner is Bob.
	wearLogs := listWearLogs(t, srv, tokenBob, itemID)
	require.Len(t, wearLogs, 1)
	assert.Equal(t, bobID, wearLogs[0]["owner_id"])

	// Alice's share to Carol is gone — Carol can no longer access the item.
	carolAfterResp := getItem(t, srv, tokenCarol, itemID)
	assert.Equal(t, http.StatusForbidden, carolAfterResp.StatusCode)

	// Transfer has decided_at set and status=accepted.
	transferResp := getTransfer(t, srv, tokenBob, transferID)
	require.Equal(t, http.StatusOK, transferResp.StatusCode)
	var transfer map[string]any
	decodeJSON(t, transferResp, &transfer)
	assert.Equal(t, "accepted", transfer["status"])
	assert.NotNil(t, transfer["decided_at"])
}

// TestItemTransferShouldDeleteWearLogsWhenAcceptedWithoutHistory covers scenario 2:
// Same setup but transfer_history=false. After Bob accepts, wear logs are gone.
func TestItemTransferShouldDeleteWearLogsWhenAcceptedWithoutHistory(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, tokenBob, _, _, bobID, _ := setupThreeUsers(t, srv)

	itemID := createItem(t, srv, tokenAlice, "Red Shirt")
	logWear(t, srv, tokenAlice, itemID, "2026-02-15")

	transferID := createTransfer(t, srv, tokenAlice, itemID, bobID, false)
	acceptTransfer(t, srv, tokenBob, transferID)

	// Bob owns the item.
	bobGetResp := getItem(t, srv, tokenBob, itemID)
	require.Equal(t, http.StatusOK, bobGetResp.StatusCode)
	var itemBody map[string]any
	decodeJSON(t, bobGetResp, &itemBody)
	assert.Equal(t, bobID, itemBody["owner_id"])

	// Wear logs are gone.
	wearLogs := listWearLogs(t, srv, tokenBob, itemID)
	assert.Empty(t, wearLogs)
}

// TestItemTransferShouldDetachFromOutfitWhenAccepted covers scenario 3:
// Alice puts item in an outfit and logs it; transfers without history; Bob accepts.
// The item is removed from the outfit, its wear logs are gone, but the outfit log persists.
func TestItemTransferShouldDetachFromOutfitWhenAccepted(t *testing.T) {
	srv := startIntegrationServer(t)
	tokenAlice, tokenBob, _, _, bobID, _ := setupThreeUsers(t, srv)

	itemID := createItem(t, srv, tokenAlice, "Green Jacket")
	outfitID := createOutfit(t, srv, tokenAlice)
	addItemToOutfit(t, srv, tokenAlice, outfitID, itemID)
	logOutfit(t, srv, tokenAlice, outfitID, "2026-03-01")

	// Verify wear log was created via outfit logging.
	beforeLogs := listWearLogs(t, srv, tokenAlice, itemID)
	require.Len(t, beforeLogs, 1)

	// Transfer without history; Bob accepts.
	transferID := createTransfer(t, srv, tokenAlice, itemID, bobID, false)
	acceptTransfer(t, srv, tokenBob, transferID)

	// Bob owns the item.
	bobGetResp := getItem(t, srv, tokenBob, itemID)
	require.Equal(t, http.StatusOK, bobGetResp.StatusCode)
	var itemBody map[string]any
	decodeJSON(t, bobGetResp, &itemBody)
	assert.Equal(t, bobID, itemBody["owner_id"])

	// Item is detached from outfit — outfit shows no items.
	outfitData := getOutfit(t, srv, tokenAlice, outfitID)
	assert.Empty(t, outfitData["items"])

	// Wear logs for the item are gone (no history transfer).
	wearLogs := listWearLogs(t, srv, tokenBob, itemID)
	assert.Empty(t, wearLogs)

	// Outfit log itself still exists (history of wearing the outfit).
	outfitLogs := listOutfitLogs(t, srv, tokenAlice, outfitID)
	require.Len(t, outfitLogs, 1)
	assert.Equal(t, "2026-03-01", outfitLogs[0]["worn_on"])
}
