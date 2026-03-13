package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

const (
	refreshTokenBytes = 32
	refreshTokenTTL   = 7 * 24 * time.Hour
	accessTokenTTL    = 15 * time.Minute
)

type AuthService struct {
	users      ports.StorageProvider[domain.User]
	sessions   ports.StorageProvider[domain.Session]
	secret     []byte
	randRead   func([]byte) (int, error)
	now        func() time.Time
	issueToken func(domain.User, time.Time, []byte) (string, error)
}

func NewAuthService(
	users ports.StorageProvider[domain.User],
	sessions ports.StorageProvider[domain.Session],
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
	users, err := s.users.List(ctx)
	if err != nil {
		return domain.User{}, err
	}
	for _, u := range users {
		if u.Email == email {
			if verifyPassword(password, u.PasswordHash) == nil {
				return u, nil
			}
			return domain.User{}, domain.ErrUnauthorized
		}
	}
	return domain.User{}, domain.ErrUnauthorized
}

func (s *AuthService) createSession(ctx context.Context, userID string) (string, error) {
	buf := make([]byte, refreshTokenBytes)
	if _, err := s.randRead(buf); err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	rawRandom := base64.RawURLEncoding.EncodeToString(buf)

	sessionID := uuid.NewString()
	rawToken := sessionID + "." + rawRandom

	now := s.now()
	var session domain.Session
	session.ID = sessionID
	session.UserID = userID
	session.TokenHash = hashToken(s.secret, rawRandom)
	session.ExpiresAt = now.Add(refreshTokenTTL)
	session.CreatedAt = now
	return rawToken, s.sessions.Save(ctx, session)
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (accessToken, refreshToken string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}

	sessionID, rawRandom, ok := strings.Cut(rawRefreshToken, ".")
	if !ok {
		return "", "", domain.ErrUnauthorized
	}

	session, err := s.retrieveSession(ctx, sessionID, rawRandom)
	if err != nil {
		return "", "", err
	}

	return s.refreshSession(ctx, session)
}

func (s *AuthService) retrieveSession(ctx context.Context, sessionID, rawRandom string) (domain.Session, error) {
	session, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return domain.Session{}, err
	}

	if s.now().After(session.ExpiresAt) {
		return domain.Session{}, domain.ErrSessionExpired
	}

	if !hmac.Equal([]byte(session.TokenHash), []byte(hashToken(s.secret, rawRandom))) {
		return domain.Session{}, domain.ErrUnauthorized
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

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.sessions.Delete(ctx, sessionID)
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
