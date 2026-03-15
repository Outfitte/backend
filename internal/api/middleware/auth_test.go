package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/outfitte/outfitte/internal/api/middleware"
)

const testSecret = "test-secret-that-is-long-enough-for-tests-32chars"

// makeToken generates a signed HS256 JWT with the given claims overrides.
func makeToken(secret []byte, sub, role string, iss, aud string, exp time.Time) string {
	claims := jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"iss":  iss,
		"aud":  []string{aud},
		"exp":  exp.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		panic(err)
	}
	return signed
}

func validToken() string {
	return makeToken([]byte(testSecret), "user-1", "member", "outfitte", "outfitte-api", time.Now().Add(time.Hour))
}

// okHandler is a simple next handler that records it was called.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// captureHandler records the context values injected by middleware.
type captureHandler struct {
	userID string
	role   string
}

func (h *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIDFromContext(r.Context())
	role, _ := middleware.RoleFromContext(r.Context())
	h.userID = id
	h.role = string(role)
	w.WriteHeader(http.StatusOK)
}

func newMiddleware() *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware([]byte(testSecret))
}

func newRequest(t *testing.T, method, target string) *http.Request {
	t.Helper()
	return httptest.NewRequest(method, target, nil).WithContext(t.Context())
}

// --- Authenticate failure cases ---

func TestAuthenticateShouldReturn401WhenAuthorizationHeaderIsMissing(t *testing.T) {
	m := newMiddleware()
	req := newRequest(t, http.MethodGet, "/")
	rr := httptest.NewRecorder()
	m.Authenticate(okHandler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateShouldReturn401WhenAuthorizationSchemeIsNotBearer(t *testing.T) {
	m := newMiddleware()
	req := newRequest(t, http.MethodGet, "/")
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()
	m.Authenticate(okHandler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateShouldReturn401WhenTokenIsExpired(t *testing.T) {
	m := newMiddleware()
	token := makeToken([]byte(testSecret), "user-1", "member", "outfitte", "outfitte-api", time.Now().Add(-time.Hour))
	req := newRequest(t, http.MethodGet, "/")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	m.Authenticate(okHandler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateShouldReturn401WhenIssuerIsWrong(t *testing.T) {
	m := newMiddleware()
	token := makeToken([]byte(testSecret), "user-1", "member", "wrong-issuer", "outfitte-api", time.Now().Add(time.Hour))
	req := newRequest(t, http.MethodGet, "/")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	m.Authenticate(okHandler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateShouldReturn401WhenAudienceIsWrong(t *testing.T) {
	m := newMiddleware()
	token := makeToken([]byte(testSecret), "user-1", "member", "outfitte", "wrong-audience", time.Now().Add(time.Hour))
	req := newRequest(t, http.MethodGet, "/")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	m.Authenticate(okHandler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateShouldReturn401WhenAlgorithmIsNotHS256(t *testing.T) {
	m := newMiddleware()
	// Sign with none algorithm by crafting a token manually (simulate alg confusion)
	claims := jwt.MapClaims{
		"sub":  "user-1",
		"role": "member",
		"iss":  "outfitte",
		"aud":  []string{"outfitte-api"},
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	// RS256 requires a real key — use HS384 as a valid but unexpected HMAC variant
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	req := newRequest(t, http.MethodGet, "/")
	req.Header.Set("Authorization", "Bearer "+signed)
	rr := httptest.NewRecorder()
	m.Authenticate(okHandler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateShouldReturn401WhenTokenSignatureIsInvalid(t *testing.T) {
	m := newMiddleware()
	wrongSecret := []byte("wrong-secret-also-long-enough-for-jwt-32chars!!")
	token := makeToken(wrongSecret, "user-1", "member", "outfitte", "outfitte-api", time.Now().Add(time.Hour))
	req := newRequest(t, http.MethodGet, "/")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	m.Authenticate(okHandler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// --- Authenticate happy path ---

func TestAuthenticateShouldInjectClaimsWhenTokenIsValid(t *testing.T) {
	m := newMiddleware()
	token := makeToken([]byte(testSecret), "user-42", "admin", "outfitte", "outfitte-api", time.Now().Add(time.Hour))
	req := newRequest(t, http.MethodGet, "/")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	capture := &captureHandler{}
	m.Authenticate(capture).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "user-42", capture.userID)
	assert.Equal(t, "admin", capture.role)
}

// --- WithUserID ---

func TestWithUserIDShouldStoreUserIDInContext(t *testing.T) {
	ctx := middleware.WithUserID(t.Context(), "user-42")
	id, ok := middleware.UserIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "user-42", id)
}

// --- RequireAdmin failure cases ---

func TestRequireAdminShouldReturn403WhenRoleIsMissingFromContext(t *testing.T) {
	req := newRequest(t, http.MethodGet, "/")
	rr := httptest.NewRecorder()
	middleware.RequireAdmin(okHandler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireAdminShouldReturn403WhenRoleIsMember(t *testing.T) {
	m := newMiddleware()
	memberToken := makeToken([]byte(testSecret), "user-1", "member", "outfitte", "outfitte-api", time.Now().Add(time.Hour))
	req := newRequest(t, http.MethodGet, "/")
	req.Header.Set("Authorization", "Bearer "+memberToken)
	rr := httptest.NewRecorder()
	m.Authenticate(middleware.RequireAdmin(okHandler)).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// --- RequireAdmin happy path ---

func TestRequireAdminShouldCallNextWhenRoleIsAdmin(t *testing.T) {
	m := newMiddleware()
	adminToken := makeToken([]byte(testSecret), "user-42", "admin", "outfitte", "outfitte-api", time.Now().Add(time.Hour))
	req := newRequest(t, http.MethodGet, "/")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr := httptest.NewRecorder()
	m.Authenticate(middleware.RequireAdmin(okHandler)).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
