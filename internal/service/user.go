package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

type UserService struct {
	users    ports.StorageProvider[domain.User]
	settings ports.SingletonStore[domain.AppSettings]
	randRead func([]byte) (int, error)
}

func NewUserService(users ports.StorageProvider[domain.User], settings ports.SingletonStore[domain.AppSettings]) *UserService {
	return &UserService{users: users, settings: settings, randRead: rand.Read}
}

func (s *UserService) GetByID(ctx context.Context, id string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}
	return s.users.Get(ctx, id)
}

func (s *UserService) ListAll(ctx context.Context, callerID string) ([]domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	caller, err := s.users.Get(ctx, callerID)
	if err != nil {
		return nil, err
	}
	if caller.Role != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}
	users, err := s.users.List(ctx)
	if err != nil {
		return domain.User{}, err
	}
	for _, u := range users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (s *UserService) Register(ctx context.Context, email, password string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	if err := s.canRegister(ctx); err != nil {
		return domain.User{}, err
	}

	hash, err := hashPassword(password, s.randRead)
	if err != nil {
		return domain.User{}, err
	}

	role, err := s.defineRole(ctx, email)
	if err != nil {
		return domain.User{}, err
	}

	var user domain.User
	user.ID = uuid.NewString()
	user.Email = email
	user.PasswordHash = hash
	user.Role = role
	user.CreatedAt = time.Now().UTC()

	if err := s.users.Save(ctx, user); err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (s *UserService) GetSettings(ctx context.Context) (domain.AppSettings, error) {
	if err := ctx.Err(); err != nil {
		return domain.AppSettings{}, err
	}
	settings, err := s.settings.Load(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.AppSettings{}, nil
		}
		return domain.AppSettings{}, err
	}
	return settings, nil
}

func (s *UserService) UpdateRegistrationEnabled(ctx context.Context, callerID string, enabled bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	caller, err := s.users.Get(ctx, callerID)
	if err != nil {
		return err
	}
	if caller.Role != domain.RoleAdmin {
		return domain.ErrForbidden
	}
	settings, err := s.settings.Load(ctx)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		settings = domain.AppSettings{}
	}
	settings.RegistrationEnabled = enabled
	return s.settings.Save(ctx, settings)
}

// canRegister checks that registration is allowed.
// If no users exist yet, registration is always allowed (first-user bootstrap).
func (s *UserService) canRegister(ctx context.Context) error {
	users, err := s.users.List(ctx)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	settings, err := s.settings.Load(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrRegistrationDisabled
		}
		return err
	}
	if !settings.RegistrationEnabled {
		return domain.ErrRegistrationDisabled
	}
	return nil
}

// defineRole lists existing users to check for email conflicts and determine
// the role for the new user: RoleAdmin if this is the first user, RoleMember otherwise.
func (s *UserService) defineRole(ctx context.Context, email string) (domain.Role, error) {
	existing, err := s.users.List(ctx)
	if err != nil {
		return "", err
	}

	for _, u := range existing {
		if u.Email == email {
			return "", domain.ErrConflict
		}
	}

	if len(existing) == 0 {
		return domain.RoleAdmin, nil
	}
	return domain.RoleMember, nil
}

const (
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Time    = 3
	argon2Threads = 2
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

func hashPassword(password string, randRead func([]byte) (int, error)) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := randRead(salt); err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return encoded, nil
}

// verifyPassword checks password against a PHC-format argon2id hash produced by hashPassword.
// Returns domain.ErrUnauthorized if the password does not match or the hash is malformed.
// Malformed-hash failures are intentionally reported as ErrUnauthorized (not ErrIO) to avoid
// leaking structural information about the stored hash to callers.
func verifyPassword(password, hash string) error {
	// PHC format: $argon2id$v=19$m=...,t=...,p=...$<salt_b64>$<key_b64>
	// Split produces: ["", "argon2id", "v=19", "m=...", salt, key] → 6 parts.
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return domain.ErrUnauthorized
	}
	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return domain.ErrUnauthorized
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return domain.ErrUnauthorized
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return domain.ErrUnauthorized
	}
	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return domain.ErrUnauthorized
	}
	actualKey := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expectedKey)))
	if !bytes.Equal(actualKey, expectedKey) {
		return domain.ErrUnauthorized
	}
	return nil
}
