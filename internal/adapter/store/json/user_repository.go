package json

import (
	"context"
	"fmt"

	"github.com/outfitte/outfitte/internal/domain"
)

// UserRepository is a JSON file-backed implementation of ports.UserRepository.
type UserRepository struct {
	provider *Provider[domain.User]
}

// NewUserRepository creates a UserRepository that stores users in root/users.json.
func NewUserRepository(root string) *UserRepository {
	return &UserRepository{
		provider: NewProvider[domain.User](root, "users.json"),
	}
}

func (r *UserRepository) Get(ctx context.Context, id string) (domain.User, error) {
	return r.provider.Get(ctx, id)
}

func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	users, err := r.provider.List(ctx)
	if err != nil {
		return err
	}
	for _, u := range users {
		if u.Email == user.Email && u.ID != user.ID {
			return fmt.Errorf("%w: email already in use", domain.ErrConflict)
		}
	}
	return r.provider.Save(ctx, user)
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	return r.provider.List(ctx)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}
	users, err := r.provider.List(ctx)
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

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	users, err := r.provider.List(ctx)
	if err != nil {
		return 0, err
	}
	return len(users), nil
}
