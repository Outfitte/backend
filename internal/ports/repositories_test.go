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

func (s *itemRepositoryStub) Get(_ context.Context, _ string) (domain.Item, error) {
	return domain.Item{}, nil
}
func (s *itemRepositoryStub) Save(_ context.Context, _ domain.Item) error          { return nil }
func (s *itemRepositoryStub) Delete(_ context.Context, _ string) error             { return nil }
func (s *itemRepositoryStub) ListByOwner(_ context.Context, _ string, _ ports.ItemListFilter) ([]domain.Item, error) {
	return nil, nil
}
func (s *itemRepositoryStub) CountByLocation(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *itemRepositoryStub) SavePhoto(_ context.Context, _, _, _ string, _ int) error { return nil }
func (s *itemRepositoryStub) DeletePhoto(_ context.Context, _, _ string) error         { return nil }
func (s *itemRepositoryStub) ListPhotoKeys(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

type userRepositoryStub struct{}

func (s *userRepositoryStub) Get(_ context.Context, _ string) (domain.User, error) {
	return domain.User{}, nil
}
func (s *userRepositoryStub) Save(_ context.Context, _ domain.User) error { return nil }
func (s *userRepositoryStub) GetByEmail(_ context.Context, _ string) (domain.User, error) {
	return domain.User{}, nil
}
func (s *userRepositoryStub) Count(_ context.Context) (int, error)            { return 0, nil }
func (s *userRepositoryStub) List(_ context.Context) ([]domain.User, error)  { return nil, nil }

type sessionRepositoryStub struct{}

func (s *sessionRepositoryStub) Get(_ context.Context, _ string) (domain.Session, error) {
	return domain.Session{}, nil
}
func (s *sessionRepositoryStub) Save(_ context.Context, _ domain.Session) error { return nil }
func (s *sessionRepositoryStub) Delete(_ context.Context, _ string) error       { return nil }
func (s *sessionRepositoryStub) FindByTokenHash(_ context.Context, _ string) (domain.Session, error) {
	return domain.Session{}, nil
}
func (s *sessionRepositoryStub) CountByUser(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *sessionRepositoryStub) DeleteOldestByUser(_ context.Context, _ string) error { return nil }
