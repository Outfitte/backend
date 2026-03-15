// Package middleware provides HTTP middleware for the Outfitte API.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/outfitte/outfitte/internal/domain"
)

type contextKeyUserIDType struct{}
type contextKeyRoleType struct{}

var (
	contextKeyUserID = contextKeyUserIDType{}
	contextKeyRole   = contextKeyRoleType{}
)

// AuthMiddleware validates Bearer JWTs on incoming requests.
type AuthMiddleware struct {
	secret []byte
}

// NewAuthMiddleware creates an AuthMiddleware using the given HMAC secret.
func NewAuthMiddleware(secret []byte) *AuthMiddleware {
	return &AuthMiddleware{secret: secret}
}

// Authenticate is an http.Handler middleware that validates the Bearer JWT,
// injects userID and role into the request context, and calls next on success.
// On failure it writes a 401 JSON response.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeUnauthorized(w)
			return
		}
		tokenStr, ok := strings.CutPrefix(authHeader, "Bearer ")
		if !ok {
			writeUnauthorized(w)
			return
		}
		claims, err := parseToken(tokenStr, m.secret)
		if err != nil {
			writeUnauthorized(w)
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, contextKeyUserID, claims.Subject)
		ctx = context.WithValue(ctx, contextKeyRole, domain.Role(claims.Role))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext returns the user ID injected by Authenticate.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(contextKeyUserID).(string)
	return v, ok
}

// RoleFromContext returns the role injected by Authenticate.
func RoleFromContext(ctx context.Context) (domain.Role, bool) {
	v, ok := ctx.Value(contextKeyRole).(domain.Role)
	return v, ok
}

// RequireAdmin is middleware that allows only admin-role requests through.
// It must be used after Authenticate. Returns 403 if the role is not admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := RoleFromContext(r.Context())
		if !ok || role != domain.RoleAdmin {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}

type accessTokenClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

func parseToken(tokenStr string, secret []byte) (*accessTokenClaims, error) {
	var claims accessTokenClaims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer("outfitte"),
		jwt.WithAudience("outfitte-api"),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	return &claims, nil
}
