package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

const (
	refreshTokenBytes = 32
	bcryptCost        = 10
	refreshTokenTTL   = 7 * 24 * time.Hour
	accessTokenTTL    = 15 * time.Minute
)

type AuthService struct {
	users    ports.StorageProvider[domain.User]
	sessions ports.StorageProvider[domain.Session]
	secret   []byte
	randRead func([]byte) (int, error)
	now      func() time.Time
}

func NewAuthService(
	users ports.StorageProvider[domain.User],
	sessions ports.StorageProvider[domain.Session],
	secret []byte,
) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
		secret:   secret,
		randRead: rand.Read,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}

	users, err := s.users.List(ctx)
	if err != nil {
		return "", "", err
	}
	var user domain.User
	found := false
	for _, u := range users {
		if u.Email == email {
			user = u
			found = true
			break
		}
	}
	if !found {
		return "", "", domain.ErrUnauthorized
	}

	if err := verifyPassword(password, user.PasswordHash); err != nil {
		return "", "", domain.ErrUnauthorized
	}

	rawToken, err := generateRefreshToken(s.randRead)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	tokenHash, err := bcrypt.GenerateFromPassword([]byte(rawToken), bcryptCost)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	now := s.now()
	var session domain.Session
	session.ID = uuid.NewString()
	session.UserID = user.GetID()
	session.TokenHash = string(tokenHash)
	session.ExpiresAt = now.Add(refreshTokenTTL)
	session.CreatedAt = now

	if err := s.sessions.Save(ctx, session); err != nil {
		return "", "", err
	}

	signed, err := issueAccessToken(user, now, s.secret)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return signed, rawToken, nil
}

func generateRefreshToken(randRead func([]byte) (int, error)) (string, error) {
	buf := make([]byte, refreshTokenBytes)
	if _, err := randRead(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func issueAccessToken(user domain.User, now time.Time, secret []byte) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.GetID(),
		"role": string(user.Role),
		"exp":  now.Add(accessTokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
