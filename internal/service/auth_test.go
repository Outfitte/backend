package service

import (
	"context"
	"errors"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// mockSessionStore is an in-memory StorageProvider[domain.Session] for tests.
type mockSessionStore struct {
	sessions []domain.Session
	saveErr  error
}

func (m *mockSessionStore) Get(_ context.Context, id string) (domain.Session, error) {
	for _, s := range m.sessions {
		if s.GetID() == id {
			return s, nil
		}
	}
	return domain.Session{}, domain.ErrNotFound
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
	return errors.New("not implemented")
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
