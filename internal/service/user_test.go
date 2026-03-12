package service

import (
	"context"
	"errors"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// mockUserStore is an in-memory StorageProvider[domain.User] for tests.
type mockUserStore struct {
	users   []domain.User
	listErr error
	saveErr error
}

func (m *mockUserStore) Get(_ context.Context, id string) (domain.User, error) {
	for _, u := range m.users {
		if u.GetID() == id {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (m *mockUserStore) List(_ context.Context) ([]domain.User, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.users, nil
}

func (m *mockUserStore) Save(_ context.Context, u domain.User) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	for i, existing := range m.users {
		if existing.GetID() == u.GetID() {
			m.users[i] = u
			return nil
		}
	}
	m.users = append(m.users, u)
	return nil
}

func (m *mockUserStore) Delete(_ context.Context, id string) error {
	return errors.New("not implemented")
}

// mockSettingsStore is an in-memory SingletonStore[domain.AppSettings] for tests.
type mockSettingsStore struct {
	settings domain.AppSettings
	err      error
	notFound bool
}

func (m *mockSettingsStore) Load(_ context.Context) (domain.AppSettings, error) {
	if m.err != nil {
		return domain.AppSettings{}, m.err
	}
	if m.notFound {
		return domain.AppSettings{}, domain.ErrNotFound
	}
	return m.settings, nil
}

func (m *mockSettingsStore) Save(_ context.Context, s domain.AppSettings) error {
	if m.err != nil {
		return m.err
	}
	m.settings = s
	return nil
}

func TestGetByIDShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetByID(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetByIDShouldReturnErrNotFoundWhenUserDoesNotExist(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})

	_, err := svc.GetByID(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetByIDShouldReturnUserWhenFound(t *testing.T) {
	var u domain.User
	u.ID = "42"
	u.Email = "alice@example.com"

	store := &mockUserStore{users: []domain.User{u}}
	svc := NewUserService(store, &mockSettingsStore{})

	got, err := svc.GetByID(t.Context(), "42")
	require.NoError(t, err)
	require.Equal(t, u, got)
}

func TestGetByEmailShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetByEmail(ctx, "alice@example.com")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetByEmailShouldReturnErrNotFoundWhenEmailDoesNotExist(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})

	_, err := svc.GetByEmail(t.Context(), "alice@example.com")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetByEmailShouldReturnErrorWhenListFails(t *testing.T) {
	store := &mockUserStore{listErr: domain.ErrIO}
	svc := NewUserService(store, &mockSettingsStore{})

	_, err := svc.GetByEmail(t.Context(), "alice@example.com")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetByEmailShouldReturnUserWhenFound(t *testing.T) {
	var u domain.User
	u.ID = "42"
	u.Email = "alice@example.com"

	store := &mockUserStore{users: []domain.User{u}}
	svc := NewUserService(store, &mockSettingsStore{})

	got, err := svc.GetByEmail(t.Context(), "alice@example.com")
	require.NoError(t, err)
	require.Equal(t, u, got)
}

func TestRegisterShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Register(ctx, "alice@example.com", "password")
	require.ErrorIs(t, err, context.Canceled)
}

func TestRegisterShouldReturnErrRegistrationDisabledWhenRegistrationIsDisabled(t *testing.T) {
	store := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: false}}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrRegistrationDisabled)
}

func TestRegisterShouldReturnErrConflictWhenEmailAlreadyExists(t *testing.T) {
	var existingUser domain.User
	existingUser.ID = "1"
	existingUser.Email = "alice@example.com"

	store := &mockUserStore{users: []domain.User{existingUser}}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestRegisterShouldReturnErrorWhenStoreListFails(t *testing.T) {
	store := &mockUserStore{listErr: domain.ErrIO}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRegisterShouldReturnErrorWhenSettingsLoadFails(t *testing.T) {
	store := &mockUserStore{}
	settings := &mockSettingsStore{err: domain.ErrIO}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRegisterShouldReturnErrIOWhenRandFails(t *testing.T) {
	store := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	svc := NewUserService(store, settings)
	svc.randRead = func(b []byte) (int, error) {
		return 0, errors.New("entropy failure")
	}

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRegisterShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	store := &mockUserStore{saveErr: domain.ErrIO}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRegisterShouldCreateAdminWhenFirstUser(t *testing.T) {
	store := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	svc := NewUserService(store, settings)

	user, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)
	require.Equal(t, domain.RoleAdmin, user.Role)
	require.Equal(t, "alice@example.com", user.Email)
	require.NotEmpty(t, user.GetID())
	require.NotEmpty(t, user.PasswordHash)
}

func TestRegisterShouldCreateMemberWhenRegistrationIsEnabled(t *testing.T) {
	var existingUser domain.User
	existingUser.ID = "1"
	existingUser.Email = "bob@example.com"

	store := &mockUserStore{users: []domain.User{existingUser}}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	svc := NewUserService(store, settings)

	user, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)
	require.Equal(t, domain.RoleMember, user.Role)
	require.Equal(t, "alice@example.com", user.Email)
	require.NotEmpty(t, user.GetID())
	require.NotEmpty(t, user.PasswordHash)
	require.False(t, user.CreatedAt.IsZero())
}
