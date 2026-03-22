package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

// mockSessionRepo is an in-memory ports.SessionRepository for tests.
type mockSessionRepo struct {
	sessions              []domain.Session
	getErr                error
	saveErr               error
	deleteErr             error
	findByTokenHashErr    error
	countByUserErr        error
	deleteOldestByUserErr error
}

var _ ports.SessionRepository = &mockSessionRepo{}

func (m *mockSessionRepo) Get(ctx context.Context, id string) (domain.Session, error) {
	if m.getErr != nil {
		return domain.Session{}, m.getErr
	}
	for _, s := range m.sessions {
		if s.GetID() == id {
			return s, nil
		}
	}
	return domain.Session{}, domain.ErrNotFound
}

func (m *mockSessionRepo) Save(ctx context.Context, s domain.Session) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.sessions = append(m.sessions, s)
	return nil
}

func (m *mockSessionRepo) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, s := range m.sessions {
		if s.GetID() == id {
			m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockSessionRepo) FindByTokenHash(ctx context.Context, hash string) (domain.Session, error) {
	if m.findByTokenHashErr != nil {
		return domain.Session{}, m.findByTokenHashErr
	}
	for _, s := range m.sessions {
		if s.TokenHash == hash {
			return s, nil
		}
	}
	return domain.Session{}, domain.ErrNotFound
}

func (m *mockSessionRepo) CountByUser(ctx context.Context, userID string) (int, error) {
	if m.countByUserErr != nil {
		return 0, m.countByUserErr
	}
	count := 0
	for _, s := range m.sessions {
		if s.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockSessionRepo) DeleteOldestByUser(ctx context.Context, userID string) error {
	if m.deleteOldestByUserErr != nil {
		return m.deleteOldestByUserErr
	}
	oldestIdx := -1
	for i, s := range m.sessions {
		if s.UserID == userID {
			if oldestIdx == -1 || s.CreatedAt.Before(m.sessions[oldestIdx].CreatedAt) {
				oldestIdx = i
			}
		}
	}
	if oldestIdx == -1 {
		return domain.ErrNotFound
	}
	m.sessions = append(m.sessions[:oldestIdx], m.sessions[oldestIdx+1:]...)
	return nil
}

// makeTestSession creates a session pre-seeded with a known opaque raw refresh token.
// The token is a single random-looking blob; the hash covers the full token.
func makeTestSession(t *testing.T, sessionID, userID string, secret []byte) (domain.Session, string) {
	t.Helper()
	rawToken := "opaquetesttoken48bytesbase64urlpadding123456789012"
	var s domain.Session
	s.ID = sessionID
	s.UserID = userID
	s.TokenHash = hashToken(secret, rawToken)
	s.ExpiresAt = time.Now().UTC().Add(24 * time.Hour)
	s.CreatedAt = time.Now().UTC()
	return s, rawToken
}

func TestLogoutShouldReturnErrorWhenFindByTokenHashFails(t *testing.T) {
	sessionRepo := &mockSessionRepo{findByTokenHashErr: domain.ErrIO}
	svc := NewAuthService(&mockUserStore{}, sessionRepo, []byte("secret"))

	err := svc.Logout(t.Context(), "any-opaque-token")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLogoutShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionRepo{}, []byte("secret"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Logout(ctx, "any-opaque-token")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLogoutShouldReturnErrNotFoundWhenSessionDoesNotExist(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionRepo{}, []byte("secret"))

	err := svc.Logout(t.Context(), "nonexistent-opaque-token")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLogoutShouldReturnErrorWhenDeleteFails(t *testing.T) {
	session, rawToken := makeTestSession(t, "session-1", "user-1", []byte("secret"))
	sessionRepo := &mockSessionRepo{sessions: []domain.Session{session}, deleteErr: domain.ErrIO}
	svc := NewAuthService(&mockUserStore{}, sessionRepo, []byte("secret"))

	err := svc.Logout(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLogoutShouldSucceedWhenSessionExists(t *testing.T) {
	session, rawToken := makeTestSession(t, "session-42", "user-1", []byte("secret"))
	sessionRepo := &mockSessionRepo{sessions: []domain.Session{session}}
	svc := NewAuthService(&mockUserStore{}, sessionRepo, []byte("secret"))

	err := svc.Logout(t.Context(), rawToken)
	require.NoError(t, err)
	require.Empty(t, sessionRepo.sessions)
}

func TestRefreshShouldReturnErrorWhenFindByTokenHashFails(t *testing.T) {
	sessionRepo := &mockSessionRepo{findByTokenHashErr: domain.ErrIO}
	svc := NewAuthService(&mockUserStore{}, sessionRepo, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), "any-opaque-token")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRefreshShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionRepo{}, []byte("secret"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := svc.Refresh(ctx, "session-1.token")
	require.ErrorIs(t, err, context.Canceled)
}

func TestRefreshShouldReturnErrNotFoundWhenTokenDoesNotMatchAnySession(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionRepo{}, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), "unknownOpaqueToken")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRefreshShouldReturnErrNotFoundWhenTokenHashDoesNotMatch(t *testing.T) {
	session, _ := makeTestSession(t, "session-42", "user-1", []byte("secret"))
	sessionRepo := &mockSessionRepo{sessions: []domain.Session{session}}
	svc := NewAuthService(&mockUserStore{}, sessionRepo, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), "wrong-opaque-token")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRefreshShouldReturnErrSessionExpiredWhenSessionIsExpired(t *testing.T) {
	session, rawToken := makeTestSession(t, "session-42", "user-1", []byte("secret"))
	session.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour)
	sessionRepo := &mockSessionRepo{sessions: []domain.Session{session}}
	svc := NewAuthService(&mockUserStore{}, sessionRepo, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrSessionExpired)
}

func TestRefreshShouldReturnErrorWhenUserGetFails(t *testing.T) {
	userStore := &mockUserStore{} // user-1 not in store
	session, rawToken := makeTestSession(t, "session-42", "user-1", []byte("secret"))
	sessionRepo := &mockSessionRepo{sessions: []domain.Session{session}}
	svc := NewAuthService(userStore, sessionRepo, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRefreshShouldReturnErrorWhenDeleteFails(t *testing.T) {
	var user domain.User
	user.ID = "user-1"
	userStore := &mockUserStore{users: []domain.User{user}}
	session, rawToken := makeTestSession(t, "session-42", "user-1", []byte("secret"))
	sessionRepo := &mockSessionRepo{
		sessions:  []domain.Session{session},
		deleteErr: domain.ErrIO,
	}
	svc := NewAuthService(userStore, sessionRepo, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRefreshShouldReturnErrorWhenNewSessionSaveFails(t *testing.T) {
	var user domain.User
	user.ID = "user-1"
	userStore := &mockUserStore{users: []domain.User{user}}
	session, rawToken := makeTestSession(t, "session-42", "user-1", []byte("secret"))
	sessionRepo := &mockSessionRepo{
		sessions: []domain.Session{session},
		saveErr:  domain.ErrIO,
	}
	svc := NewAuthService(userStore, sessionRepo, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRefreshShouldReturnErrIOWhenRandFails(t *testing.T) {
	var user domain.User
	user.ID = "user-1"
	userStore := &mockUserStore{users: []domain.User{user}}
	session, rawToken := makeTestSession(t, "session-42", "user-1", []byte("secret"))
	sessionRepo := &mockSessionRepo{sessions: []domain.Session{session}}
	svc := NewAuthService(userStore, sessionRepo, []byte("secret"))
	svc.randRead = func(b []byte) (int, error) {
		return 0, errors.New("entropy failure")
	}

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRefreshShouldReturnNewTokensWhenRefreshTokenIsValid(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)

	sessionRepo := &mockSessionRepo{}
	svc := NewAuthService(userStore, sessionRepo, []byte("jwt-secret"))

	_, refreshToken, err := svc.Login(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)
	require.Len(t, sessionRepo.sessions, 1)

	accessToken2, refreshToken2, err := svc.Refresh(t.Context(), refreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken2)
	require.NotEmpty(t, refreshToken2)
	require.NotEqual(t, refreshToken, refreshToken2)

	// Old session replaced by new one.
	require.Len(t, sessionRepo.sessions, 1)
}

func TestRefreshShouldReturnErrNotFoundWhenSessionDoesNotExist(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionRepo{}, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), "nonexistent-opaque-token")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRefreshShouldReturnErrIOWhenIssueTokenFails(t *testing.T) {
	var user domain.User
	user.ID = "user-1"
	userStore := &mockUserStore{users: []domain.User{user}}
	session, rawToken := makeTestSession(t, "session-42", "user-1", []byte("secret"))
	sessionRepo := &mockSessionRepo{sessions: []domain.Session{session}}
	svc := NewAuthService(userStore, sessionRepo, []byte("secret"))
	svc.issueToken = func(_ domain.User, _ time.Time, _ []byte) (string, error) {
		return "", errors.New("signing failure")
	}

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestIssueAccessTokenShouldReturnSignedTokenWhenUserIsValid(t *testing.T) {
	var user domain.User
	user.ID = "42"
	user.Role = domain.RoleMember

	now := time.Now().UTC()
	signed, err := issueAccessToken(user, now, []byte("secret"))
	require.NoError(t, err)
	require.NotEmpty(t, signed)
}

func TestIssueAccessTokenShouldIncludeRegisteredClaimsWhenUserIsValid(t *testing.T) {
	var user domain.User
	user.ID = "42"
	user.Role = domain.RoleMember

	now := time.Now().UTC().Truncate(time.Second)
	signed, err := issueAccessToken(user, now, []byte("secret"))
	require.NoError(t, err)

	token, _, err := new(jwt.Parser).ParseUnverified(signed, jwt.MapClaims{})
	require.NoError(t, err)

	claims := token.Claims.(jwt.MapClaims)
	require.Equal(t, "outfitte", claims["iss"])
	aud, ok := claims["aud"].([]interface{})
	require.True(t, ok)
	require.Contains(t, aud, "outfitte-api")
	require.NotEmpty(t, claims["jti"])
	require.Equal(t, float64(now.Unix()), claims["iat"])
	require.Equal(t, "42", claims["sub"])
	require.Equal(t, string(domain.RoleMember), claims["role"])
}

func TestLoginShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionRepo{}, []byte("secret"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := svc.Login(ctx, "alice@example.com", "password")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLoginShouldReturnErrUnauthorizedWhenEmailNotFound(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionRepo{}, []byte("secret"))

	_, _, err := svc.Login(t.Context(), "unknown@example.com", "password")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestLoginShouldReturnErrorWhenGetByEmailFails(t *testing.T) {
	store := &mockUserStore{getByEmailErr: domain.ErrIO}
	svc := NewAuthService(store, &mockSessionRepo{}, []byte("secret"))

	_, _, err := svc.Login(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLoginShouldReturnErrUnauthorizedWhenPasswordIsWrong(t *testing.T) {
	store := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(store, settings)
	registered, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	authSvc := NewAuthService(store, &mockSessionRepo{}, []byte("secret"))

	_, _, err = authSvc.Login(t.Context(), registered.Email, "wrong-password")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestLoginShouldReturnErrorWhenSessionSaveFails(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	sessionRepo := &mockSessionRepo{saveErr: domain.ErrIO}
	svc := NewAuthService(userStore, sessionRepo, []byte("secret"))

	_, _, err = svc.Login(t.Context(), "alice@example.com", "correct-password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLoginShouldReturnErrIOWhenRandFails(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	svc := NewAuthService(userStore, &mockSessionRepo{}, []byte("secret"))
	svc.randRead = func(b []byte) (int, error) {
		return 0, errors.New("entropy failure")
	}

	_, _, err = svc.Login(t.Context(), "alice@example.com", "correct-password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLoginShouldReturnErrIOWhenIssueTokenFails(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	svc := NewAuthService(userStore, &mockSessionRepo{}, []byte("secret"))
	svc.issueToken = func(_ domain.User, _ time.Time, _ []byte) (string, error) {
		return "", errors.New("signing failure")
	}

	_, _, err = svc.Login(t.Context(), "alice@example.com", "correct-password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestFullSessionCycleShouldSucceedWhenCredentialsAreValid(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)

	sessionRepo := &mockSessionRepo{}
	svc := NewAuthService(userStore, sessionRepo, []byte("jwt-secret"))

	// Login.
	accessToken1, refreshToken1, err := svc.Login(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)
	require.NotEmpty(t, accessToken1)
	require.NotEmpty(t, refreshToken1)
	require.Len(t, sessionRepo.sessions, 1)

	// Refresh: old session is replaced by a new one.
	accessToken2, refreshToken2, err := svc.Refresh(t.Context(), refreshToken1)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken2)
	require.NotEmpty(t, refreshToken2)
	require.NotEqual(t, refreshToken1, refreshToken2)
	require.Len(t, sessionRepo.sessions, 1)

	// Logout: delete the session created by the last refresh.
	err = svc.Logout(t.Context(), refreshToken2)
	require.NoError(t, err)
	require.Empty(t, sessionRepo.sessions)

	// Refreshing with the old token must now fail.
	_, _, err = svc.Refresh(t.Context(), refreshToken1)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCreateSessionShouldEvictOldestWhenUserExceeds10Sessions(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)

	userID := userStore.users[0].GetID()

	// Pre-seed 10 sessions with distinct CreatedAt values so eviction order is deterministic.
	sessions := make([]domain.Session, 10)
	for i := range sessions {
		sessions[i].ID = fmt.Sprintf("existing-session-%d", i)
		sessions[i].UserID = userID
		sessions[i].CreatedAt = time.Now().UTC().Add(-time.Duration(10-i) * time.Hour)
	}
	oldestID := sessions[0].ID
	sessionRepo := &mockSessionRepo{sessions: sessions}
	svc := NewAuthService(userStore, sessionRepo, []byte("secret"))

	// Login creates the 11th session, triggering eviction of the oldest.
	_, _, err = svc.Login(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)
	require.Len(t, sessionRepo.sessions, 10)
	for _, s := range sessionRepo.sessions {
		require.NotEqual(t, oldestID, s.ID, "oldest session must have been evicted")
	}
}

func TestCreateSessionShouldReturnErrorWhenEvictOldestDeleteFails(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)

	// Pre-seed 10 sessions so the next login triggers eviction.
	sessions := make([]domain.Session, 10)
	for i := range sessions {
		sessions[i].ID = fmt.Sprintf("existing-session-%d", i)
		sessions[i].UserID = userStore.users[0].GetID()
		sessions[i].CreatedAt = time.Now().UTC().Add(-time.Duration(10-i) * time.Hour)
	}
	sessionRepo := &mockSessionRepo{sessions: sessions, deleteOldestByUserErr: domain.ErrIO}
	svc := NewAuthService(userStore, sessionRepo, []byte("secret"))

	_, _, err = svc.Login(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestCreateSessionShouldReturnErrorWhenCountByUserFails(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)

	sessionRepo := &mockSessionRepo{countByUserErr: domain.ErrIO}
	svc := NewAuthService(userStore, sessionRepo, []byte("secret"))

	_, _, err = svc.Login(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLoginShouldReturnTokensWhenCredentialsAreValid(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	registered, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	sessionRepo := &mockSessionRepo{}
	svc := NewAuthService(userStore, sessionRepo, []byte("jwt-secret"))

	accessToken, refreshToken, err := svc.Login(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// Session must be persisted with an HMAC-SHA256 hash of the raw refresh token.
	require.Len(t, sessionRepo.sessions, 1)
	session := sessionRepo.sessions[0]
	require.Equal(t, registered.GetID(), session.UserID)
	require.NotEmpty(t, session.ID)
	require.False(t, session.CreatedAt.IsZero())
	require.False(t, session.ExpiresAt.IsZero())
}
