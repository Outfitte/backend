package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// mockSessionStore is an in-memory StorageProvider[domain.Session] for tests.
type mockSessionStore struct {
	sessions  []domain.Session
	getErr    error
	saveErr   error
	deleteErr error
}

func (m *mockSessionStore) Get(_ context.Context, id string) (domain.Session, error) {
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

// makeTestSession creates a session pre-seeded with a known raw refresh token.
// The raw token format is "sessionID.rawRandom" and the hash covers only rawRandom.
func makeTestSession(t *testing.T, sessionID, userID string) (domain.Session, string) {
	t.Helper()
	rawRandom := "testrandompart123abc"
	rawToken := sessionID + "." + rawRandom
	hash, err := bcrypt.GenerateFromPassword([]byte(rawRandom), bcrypt.MinCost)
	require.NoError(t, err)
	var s domain.Session
	s.ID = sessionID
	s.UserID = userID
	s.TokenHash = string(hash)
	s.ExpiresAt = time.Now().UTC().Add(24 * time.Hour)
	s.CreatedAt = time.Now().UTC()
	return s, rawToken
}

func (m *mockSessionStore) List(_ context.Context) ([]domain.Session, error) {
	return m.sessions, nil
}

func (m *mockSessionStore) Save(_ context.Context, s domain.Session) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.sessions = append(m.sessions, s)
	return nil
}

func (m *mockSessionStore) Delete(_ context.Context, id string) error {
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

func TestLogoutShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionStore{}, []byte("secret"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Logout(ctx, "session-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLogoutShouldReturnErrNotFoundWhenSessionDoesNotExist(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionStore{}, []byte("secret"))

	err := svc.Logout(t.Context(), "nonexistent-session")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLogoutShouldReturnErrorWhenDeleteFails(t *testing.T) {
	sessionStore := &mockSessionStore{deleteErr: domain.ErrIO}
	svc := NewAuthService(&mockUserStore{}, sessionStore, []byte("secret"))

	err := svc.Logout(t.Context(), "session-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLogoutShouldSucceedWhenSessionExists(t *testing.T) {
	var session domain.Session
	session.ID = "session-42"
	session.UserID = "user-1"
	sessionStore := &mockSessionStore{sessions: []domain.Session{session}}
	svc := NewAuthService(&mockUserStore{}, sessionStore, []byte("secret"))

	err := svc.Logout(t.Context(), "session-42")
	require.NoError(t, err)
	require.Empty(t, sessionStore.sessions)
}

func TestRefreshShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionStore{}, []byte("secret"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := svc.Refresh(ctx, "session-1.token")
	require.ErrorIs(t, err, context.Canceled)
}

func TestRefreshShouldReturnErrUnauthorizedWhenTokenFormatIsInvalid(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionStore{}, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), "nodotintoken")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestRefreshShouldReturnErrUnauthorizedWhenTokenHashDoesNotMatch(t *testing.T) {
	session, _ := makeTestSession(t, "session-42", "user-1")
	sessionStore := &mockSessionStore{sessions: []domain.Session{session}}
	svc := NewAuthService(&mockUserStore{}, sessionStore, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), "session-42.wrongrandompart")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestRefreshShouldReturnErrSessionExpiredWhenSessionIsExpired(t *testing.T) {
	session, rawToken := makeTestSession(t, "session-42", "user-1")
	session.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour)
	sessionStore := &mockSessionStore{sessions: []domain.Session{session}}
	svc := NewAuthService(&mockUserStore{}, sessionStore, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrSessionExpired)
}

func TestRefreshShouldReturnErrorWhenUserGetFails(t *testing.T) {
	userStore := &mockUserStore{} // user-1 not in store
	session, rawToken := makeTestSession(t, "session-42", "user-1")
	sessionStore := &mockSessionStore{sessions: []domain.Session{session}}
	svc := NewAuthService(userStore, sessionStore, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRefreshShouldReturnErrorWhenDeleteFails(t *testing.T) {
	var user domain.User
	user.ID = "user-1"
	userStore := &mockUserStore{users: []domain.User{user}}
	session, rawToken := makeTestSession(t, "session-42", "user-1")
	sessionStore := &mockSessionStore{
		sessions:  []domain.Session{session},
		deleteErr: domain.ErrIO,
	}
	svc := NewAuthService(userStore, sessionStore, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRefreshShouldReturnErrorWhenNewSessionSaveFails(t *testing.T) {
	var user domain.User
	user.ID = "user-1"
	userStore := &mockUserStore{users: []domain.User{user}}
	session, rawToken := makeTestSession(t, "session-42", "user-1")
	sessionStore := &mockSessionStore{
		sessions: []domain.Session{session},
		saveErr:  domain.ErrIO,
	}
	svc := NewAuthService(userStore, sessionStore, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), rawToken)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRefreshShouldReturnErrIOWhenRandFails(t *testing.T) {
	var user domain.User
	user.ID = "user-1"
	userStore := &mockUserStore{users: []domain.User{user}}
	session, rawToken := makeTestSession(t, "session-42", "user-1")
	sessionStore := &mockSessionStore{sessions: []domain.Session{session}}
	svc := NewAuthService(userStore, sessionStore, []byte("secret"))
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

	sessionStore := &mockSessionStore{}
	svc := NewAuthService(userStore, sessionStore, []byte("jwt-secret"))

	_, refreshToken, err := svc.Login(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)
	require.Len(t, sessionStore.sessions, 1)

	accessToken2, refreshToken2, err := svc.Refresh(t.Context(), refreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken2)
	require.NotEmpty(t, refreshToken2)
	require.NotEqual(t, refreshToken, refreshToken2)

	// Old session replaced by new one.
	require.Len(t, sessionStore.sessions, 1)
	require.NotEqual(t, sessionStore.sessions[0].ID, strings.SplitN(refreshToken, ".", 2)[0])
}

func TestRefreshShouldReturnErrNotFoundWhenSessionDoesNotExist(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionStore{}, []byte("secret"))

	_, _, err := svc.Refresh(t.Context(), "nonexistent-session.sometoken")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRefreshShouldReturnErrIOWhenIssueTokenFails(t *testing.T) {
	var user domain.User
	user.ID = "user-1"
	userStore := &mockUserStore{users: []domain.User{user}}
	session, rawToken := makeTestSession(t, "session-42", "user-1")
	sessionStore := &mockSessionStore{sessions: []domain.Session{session}}
	svc := NewAuthService(userStore, sessionStore, []byte("secret"))
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
	require.NotNil(t, claims["iat"])
	require.Equal(t, "42", claims["sub"])
	require.Equal(t, string(domain.RoleMember), claims["role"])
}

func TestLoginShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionStore{}, []byte("secret"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := svc.Login(ctx, "alice@example.com", "password")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLoginShouldReturnErrUnauthorizedWhenEmailNotFound(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionStore{}, []byte("secret"))

	_, _, err := svc.Login(t.Context(), "unknown@example.com", "password")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestLoginShouldReturnErrorWhenUserListFails(t *testing.T) {
	store := &mockUserStore{listErr: domain.ErrIO}
	svc := NewAuthService(store, &mockSessionStore{}, []byte("secret"))

	_, _, err := svc.Login(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLoginShouldReturnErrUnauthorizedWhenPasswordIsWrong(t *testing.T) {
	var u domain.User
	u.ID = "42"
	u.Email = "alice@example.com"
	// hashPassword using known salt+key produced via Register
	store := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(store, settings)
	registered, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	authSvc := NewAuthService(store, &mockSessionStore{}, []byte("secret"))

	_, _, err = authSvc.Login(t.Context(), registered.Email, "wrong-password")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestLoginShouldReturnErrorWhenSessionSaveFails(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	sessionStore := &mockSessionStore{saveErr: domain.ErrIO}
	svc := NewAuthService(userStore, sessionStore, []byte("secret"))

	_, _, err = svc.Login(t.Context(), "alice@example.com", "correct-password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLoginShouldReturnErrIOWhenRandFails(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	svc := NewAuthService(userStore, &mockSessionStore{}, []byte("secret"))
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

	svc := NewAuthService(userStore, &mockSessionStore{}, []byte("secret"))
	svc.issueToken = func(_ domain.User, _ time.Time, _ []byte) (string, error) {
		return "", errors.New("signing failure")
	}

	_, _, err = svc.Login(t.Context(), "alice@example.com", "correct-password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestCreateSessionShouldReturnErrIOWhenGenerateHashFails(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	_, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	svc := NewAuthService(userStore, &mockSessionStore{}, []byte("secret"))
	svc.generateHash = func(_ []byte, _ int) ([]byte, error) {
		return nil, errors.New("bcrypt failure")
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

	sessionStore := &mockSessionStore{}
	svc := NewAuthService(userStore, sessionStore, []byte("jwt-secret"))

	// Login.
	accessToken1, refreshToken1, err := svc.Login(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)
	require.NotEmpty(t, accessToken1)
	require.NotEmpty(t, refreshToken1)
	require.Len(t, sessionStore.sessions, 1)

	// Refresh: old session is replaced by a new one.
	accessToken2, refreshToken2, err := svc.Refresh(t.Context(), refreshToken1)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken2)
	require.NotEmpty(t, refreshToken2)
	require.NotEqual(t, refreshToken1, refreshToken2)
	require.Len(t, sessionStore.sessions, 1)

	// Logout: delete the session created by the last refresh.
	sessionID2 := strings.SplitN(refreshToken2, ".", 2)[0]
	err = svc.Logout(t.Context(), sessionID2)
	require.NoError(t, err)
	require.Empty(t, sessionStore.sessions)

	// Refreshing with the old token must now fail.
	_, _, err = svc.Refresh(t.Context(), refreshToken1)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLoginShouldReturnTokensWhenCredentialsAreValid(t *testing.T) {
	userStore := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	userSvc := NewUserService(userStore, settings)
	registered, err := userSvc.Register(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)

	sessionStore := &mockSessionStore{}
	svc := NewAuthService(userStore, sessionStore, []byte("jwt-secret"))

	accessToken, refreshToken, err := svc.Login(t.Context(), "alice@example.com", "correct-password")
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// Session must be persisted with a bcrypt hash of the raw refresh token.
	require.Len(t, sessionStore.sessions, 1)
	session := sessionStore.sessions[0]
	require.Equal(t, registered.GetID(), session.UserID)
	require.NotEmpty(t, session.ID)
	require.False(t, session.CreatedAt.IsZero())
	require.False(t, session.ExpiresAt.IsZero())
}
