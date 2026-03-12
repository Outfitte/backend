package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
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

func (s *UserService) Register(ctx context.Context, email, password string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	if err := s.canRegister(ctx); err != nil {
		return domain.User{}, err
	}

	hash, err := hashPassword(password, s.randRead)
	if err != nil {
		return domain.User{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
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

// canRegister checks that registration is enabled.
func (s *UserService) canRegister(ctx context.Context) error {
	settings, err := s.settings.Load(ctx)
	if err != nil {
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
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	encoded := base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
	return encoded, nil
}
