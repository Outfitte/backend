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
}

func NewUserService(users ports.StorageProvider[domain.User], settings ports.SingletonStore[domain.AppSettings]) *UserService {
	return &UserService{users: users, settings: settings}
}

func (s *UserService) Register(ctx context.Context, email, password string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	firstUser, err := s.canRegister(ctx, email)
	if err != nil {
		return domain.User{}, err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	role := domain.RoleMember
	if firstUser {
		role = domain.RoleAdmin
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

// canRegister checks that registration is enabled and the email is not taken.
// It returns whether this will be the first (admin) user.
func (s *UserService) canRegister(ctx context.Context, email string) (firstUser bool, err error) {
	settings, err := s.settings.Load(ctx)
	if err != nil {
		return false, err
	}
	if !settings.RegistrationEnabled {
		return false, domain.ErrRegistrationDisabled
	}

	existing, err := s.users.List(ctx)
	if err != nil {
		return false, err
	}

	for _, u := range existing {
		if u.Email == email {
			return false, domain.ErrConflict
		}
	}

	return len(existing) == 0, nil
}

const (
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Time    = 3
	argon2Threads = 2
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

func hashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	encoded := base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
	return encoded, nil
}
