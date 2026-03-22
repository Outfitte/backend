package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

const (
	refreshTokenBytes = 48
	refreshTokenTTL   = 7 * 24 * time.Hour
	accessTokenTTL    = 15 * time.Minute
)

type AuthService struct {
	users      ports.UserRepository
	sessions   ports.SessionRepository
	secret     []byte
	randRead   func([]byte) (int, error)
	now        func() time.Time
	issueToken func(domain.User, time.Time, []byte) (string, error)
}

func NewAuthService(
	users ports.UserRepository,
	sessions ports.SessionRepository,
	secret []byte,
) *AuthService {
	return &AuthService{
		users:      users,
		sessions:   sessions,
		secret:     secret,
		randRead:   rand.Read,
		now:        func() time.Time { return time.Now().UTC() },
		issueToken: issueAccessToken,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}

	user, err := s.findAndVerifyUser(ctx, email, password)
	if err != nil {
		return "", "", err
	}

	rawToken, err := s.createSession(ctx, user.GetID())
	if err != nil {
		return "", "", err
	}

	signed, err := s.issueToken(user, s.now(), s.secret)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return signed, rawToken, nil
}

func (s *AuthService) findAndVerifyUser(ctx context.Context, email, password string) (domain.User, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, domain.ErrUnauthorized
		}
		return domain.User{}, err
	}
	if verifyPassword(password, u.PasswordHash) != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	return u, nil
}

func (s *AuthService) createSession(ctx context.Context, userID string) (string, error) {
	buf := make([]byte, refreshTokenBytes)
	if _, err := s.randRead(buf); err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	rawToken := base64.RawURLEncoding.EncodeToString(buf)

	now := s.now()
	var session domain.Session
	session.ID = uuid.NewString()
	session.UserID = userID
	session.TokenHash = hashToken(s.secret, rawToken)
	session.ExpiresAt = now.Add(refreshTokenTTL)
	session.CreatedAt = now
	if err := s.sessions.Save(ctx, session); err != nil {
		return "", err
	}

	if err := s.evictOldestIfOverLimit(ctx, userID); err != nil {
		return "", err
	}

	return rawToken, nil
}

const maxSessionsPerUser = 10

func (s *AuthService) evictOldestIfOverLimit(ctx context.Context, userID string) error {
	count, err := s.sessions.CountByUser(ctx, userID)
	if err != nil {
		return err
	}
	if count <= maxSessionsPerUser {
		return nil
	}
	return s.sessions.DeleteOldestByUser(ctx, userID)
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (accessToken, refreshToken string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}

	session, err := s.retrieveSession(ctx, rawRefreshToken)
	if err != nil {
		return "", "", err
	}

	return s.refreshSession(ctx, session)
}

func (s *AuthService) retrieveSession(ctx context.Context, rawToken string) (domain.Session, error) {
	session, err := s.findSessionByToken(ctx, rawToken)
	if err != nil {
		return domain.Session{}, err
	}

	if s.now().After(session.ExpiresAt) {
		return domain.Session{}, domain.ErrSessionExpired
	}

	return session, nil
}

func (s *AuthService) refreshSession(ctx context.Context, session domain.Session) (accessToken, refreshToken string, err error) {
	user, err := s.users.Get(ctx, session.UserID)
	if err != nil {
		return "", "", err
	}

	if err := s.sessions.Delete(ctx, session.GetID()); err != nil {
		return "", "", err
	}

	newRawToken, err := s.createSession(ctx, user.GetID())
	if err != nil {
		return "", "", err
	}

	signed, err := s.issueToken(user, s.now(), s.secret)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return signed, newRawToken, nil
}

func (s *AuthService) findSessionByToken(ctx context.Context, rawToken string) (domain.Session, error) {
	hash := hashToken(s.secret, rawToken)
	return s.sessions.FindByTokenHash(ctx, hash)
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	session, err := s.findSessionByToken(ctx, rawRefreshToken)
	if err != nil {
		return err
	}
	return s.sessions.Delete(ctx, session.GetID())
}

type accessTokenClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

func issueAccessToken(user domain.User, now time.Time, secret []byte) (string, error) {
	claims := accessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.GetID(),
			Issuer:    "outfitte",
			Audience:  jwt.ClaimStrings{"outfitte-api"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			ID:        uuid.NewString(),
		},
		Role: string(user.Role),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func hashToken(secret []byte, rawRandom string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(rawRandom))
	return hex.EncodeToString(mac.Sum(nil))
}
