package ports_test

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// Compile-time assertion: Repositories must hold all six repository interfaces.
var _ = ports.Repositories{
	Items:       (*itemRepositoryStub)(nil),
	Users:       (*userRepositoryStub)(nil),
	Sessions:    (*sessionRepositoryStub)(nil),
	Locations:   (*locationRepositoryStub)(nil),
	WearLogs:    (*wearLogRepositoryStub)(nil),
	AppSettings: (*appSettingsRepositoryStub)(nil),
}

// Compile-time assertions: stubs must satisfy their interfaces.
var _ ports.ItemRepository = (*itemRepositoryStub)(nil)
var _ ports.UserRepository = (*userRepositoryStub)(nil)
var _ ports.SessionRepository = (*sessionRepositoryStub)(nil)

type itemRepositoryStub struct{}

func (s *itemRepositoryStub) Get(ctx context.Context, _ string) (domain.Item, error) {
	return domain.Item{}, nil
}
func (s *itemRepositoryStub) Save(ctx context.Context, _ domain.Item) error          { return nil }
func (s *itemRepositoryStub) Delete(ctx context.Context, _ string) error             { return nil }
func (s *itemRepositoryStub) ListByOwner(ctx context.Context, _ string, _ ports.ItemListFilter) ([]domain.Item, error) {
	return nil, nil
}
func (s *itemRepositoryStub) CountByLocation(ctx context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *itemRepositoryStub) SavePhoto(ctx context.Context, _, _, _ string, _ int) error { return nil }
func (s *itemRepositoryStub) DeletePhoto(ctx context.Context, _, _ string) error         { return nil }
func (s *itemRepositoryStub) ListPhotoKeys(ctx context.Context, _ string) ([]string, error) {
	return nil, nil
}

type userRepositoryStub struct{}

func (s *userRepositoryStub) Get(ctx context.Context, _ string) (domain.User, error) {
	return domain.User{}, nil
}
func (s *userRepositoryStub) Save(ctx context.Context, _ domain.User) error { return nil }
func (s *userRepositoryStub) GetByEmail(ctx context.Context, _ string) (domain.User, error) {
	return domain.User{}, nil
}
func (s *userRepositoryStub) Count(ctx context.Context) (int, error)           { return 0, nil }
func (s *userRepositoryStub) List(ctx context.Context) ([]domain.User, error) { return nil, nil }

type sessionRepositoryStub struct{}

func (s *sessionRepositoryStub) Get(ctx context.Context, _ string) (domain.Session, error) {
	return domain.Session{}, nil
}
func (s *sessionRepositoryStub) Save(ctx context.Context, _ domain.Session) error { return nil }
func (s *sessionRepositoryStub) Delete(ctx context.Context, _ string) error       { return nil }
func (s *sessionRepositoryStub) FindByTokenHash(ctx context.Context, _ string) (domain.Session, error) {
	return domain.Session{}, nil
}
func (s *sessionRepositoryStub) CountByUser(ctx context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *sessionRepositoryStub) DeleteOldestByUser(ctx context.Context, _ string) error { return nil }
