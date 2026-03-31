package service

import (
	"testing"

	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestFindSessionByTokenShouldReturnErrorWhenFindByTokenHashFails(t *testing.T) {
	sessionRepo := &mockSessionRepo{findByTokenHashErr: domain.ErrIO}
	svc := NewAuthService(&mockUserStore{}, sessionRepo, []byte("secret"))

	_, err := svc.findSessionByToken(t.Context(), "any-token")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestFindSessionByTokenShouldReturnErrNotFoundWhenNoSessionsExist(t *testing.T) {
	svc := NewAuthService(&mockUserStore{}, &mockSessionRepo{}, []byte("secret"))

	_, err := svc.findSessionByToken(t.Context(), "any-token")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestFindSessionByTokenShouldReturnErrNotFoundWhenTokenDoesNotMatch(t *testing.T) {
	var sess domain.Session
	sess.ID = "session-1"
	sess.TokenHash = hashToken([]byte("secret"), "correct-token")
	sessionRepo := &mockSessionRepo{sessions: []domain.Session{sess}}
	svc := NewAuthService(&mockUserStore{}, sessionRepo, []byte("secret"))

	_, err := svc.findSessionByToken(t.Context(), "wrong-token")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestFindSessionByTokenShouldReturnSessionWhenTokenMatches(t *testing.T) {
	var sess domain.Session
	sess.ID = "session-42"
	sess.TokenHash = hashToken([]byte("secret"), "correct-token")
	sessionRepo := &mockSessionRepo{sessions: []domain.Session{sess}}
	svc := NewAuthService(&mockUserStore{}, sessionRepo, []byte("secret"))

	got, err := svc.findSessionByToken(t.Context(), "correct-token")
	require.NoError(t, err)
	require.Equal(t, "session-42", got.GetID())
}

func TestHashTokenShouldReturn64HexCharsWhenGivenInput(t *testing.T) {
	result := hashToken([]byte("secret"), "randompart")
	require.Len(t, result, 64) // HMAC-SHA256 = 32 bytes = 64 hex chars
}

func TestHashTokenShouldBeDeterministicWhenGivenSameInputs(t *testing.T) {
	h1 := hashToken([]byte("secret"), "randompart")
	h2 := hashToken([]byte("secret"), "randompart")
	require.Equal(t, h1, h2)
}

func TestHashTokenShouldDifferWhenSecretDiffers(t *testing.T) {
	h1 := hashToken([]byte("secret-a"), "randompart")
	h2 := hashToken([]byte("secret-b"), "randompart")
	require.NotEqual(t, h1, h2)
}

func TestHashTokenShouldDifferWhenRawRandomDiffers(t *testing.T) {
	secret := []byte("secret")
	h1 := hashToken(secret, "randompart-a")
	h2 := hashToken(secret, "randompart-b")
	require.NotEqual(t, h1, h2)
}
