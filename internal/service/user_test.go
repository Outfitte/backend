package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// mockUserStore is an in-memory UserRepository for tests.
type mockUserStore struct {
	users             []domain.User
	listErr           error
	countErr          error
	countErrOnNthCall int // if > 0, return countErr only on this call number (1-indexed)
	countCallCount    int
	getByEmailErr     error
	saveErr           error
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

func (m *mockUserStore) GetByEmail(_ context.Context, email string) (domain.User, error) {
	if m.getByEmailErr != nil {
		return domain.User{}, m.getByEmailErr
	}
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (m *mockUserStore) Count(_ context.Context) (int, error) {
	m.countCallCount++
	if m.countErr != nil && (m.countErrOnNthCall == 0 || m.countCallCount == m.countErrOnNthCall) {
		return 0, m.countErr
	}
	return len(m.users), nil
}

func (m *mockUserStore) Delete(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

// mockSettingsStore is an in-memory SingletonStore[domain.AppSettings] for tests.
type mockSettingsStore struct {
	settings domain.AppSettings
	err      error
	saveErr  error
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
	if m.saveErr != nil {
		return m.saveErr
	}
	m.settings = s
	return nil
}

var _ ports.UserRepository = &mockUserStore{}
var _ ports.AppSettingsRepository = &mockSettingsStore{}

func TestRegisterShouldReturnErrorWhenCanRegisterCountFails(t *testing.T) {
	store := &mockUserStore{countErr: domain.ErrIO}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUpdateRegistrationEnabledShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.UpdateRegistrationEnabled(ctx, "42", true)
	require.ErrorIs(t, err, context.Canceled)
}

func TestUpdateRegistrationEnabledShouldReturnErrNotFoundWhenCallerDoesNotExist(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})

	err := svc.UpdateRegistrationEnabled(t.Context(), "42", true)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdateRegistrationEnabledShouldReturnErrForbiddenWhenCallerIsNotAdmin(t *testing.T) {
	var member domain.User
	member.ID = "42"
	member.Role = domain.RoleMember

	store := &mockUserStore{users: []domain.User{member}}
	svc := NewUserService(store, &mockSettingsStore{})

	err := svc.UpdateRegistrationEnabled(t.Context(), "42", true)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestUpdateRegistrationEnabledShouldReturnErrorWhenSettingsLoadFails(t *testing.T) {
	var admin domain.User
	admin.ID = "42"
	admin.Role = domain.RoleAdmin

	store := &mockUserStore{users: []domain.User{admin}}
	settings := &mockSettingsStore{err: domain.ErrIO}
	svc := NewUserService(store, settings)

	err := svc.UpdateRegistrationEnabled(t.Context(), "42", true)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUpdateRegistrationEnabledShouldReturnErrorWhenSettingsSaveFails(t *testing.T) {
	var admin domain.User
	admin.ID = "42"
	admin.Role = domain.RoleAdmin

	store := &mockUserStore{users: []domain.User{admin}}
	settings := &mockSettingsStore{saveErr: domain.ErrIO}
	svc := NewUserService(store, settings)

	err := svc.UpdateRegistrationEnabled(t.Context(), "42", true)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUpdateRegistrationEnabledShouldSucceedWhenSettingsNotFound(t *testing.T) {
	var admin domain.User
	admin.ID = "42"
	admin.Role = domain.RoleAdmin

	store := &mockUserStore{users: []domain.User{admin}}
	settings := &mockSettingsStore{notFound: true}
	svc := NewUserService(store, settings)

	err := svc.UpdateRegistrationEnabled(t.Context(), "42", true)
	require.NoError(t, err)
	require.True(t, settings.settings.RegistrationEnabled)
}

func TestUpdateRegistrationEnabledShouldUpdateSettingsWhenCallerIsAdmin(t *testing.T) {
	var admin domain.User
	admin.ID = "42"
	admin.Role = domain.RoleAdmin

	store := &mockUserStore{users: []domain.User{admin}}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: false}}
	svc := NewUserService(store, settings)

	err := svc.UpdateRegistrationEnabled(t.Context(), "42", true)
	require.NoError(t, err)
	require.True(t, settings.settings.RegistrationEnabled)
}

func TestListShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.List(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestListShouldReturnErrorWhenRepositoryFails(t *testing.T) {
	store := &mockUserStore{listErr: domain.ErrIO}
	svc := NewUserService(store, &mockSettingsStore{})

	_, err := svc.List(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListShouldReturnAllUsers(t *testing.T) {
	var u1, u2 domain.User
	u1.ID = "1"
	u1.Role = domain.RoleAdmin
	u2.ID = "2"
	u2.Role = domain.RoleMember

	store := &mockUserStore{users: []domain.User{u1, u2}}
	svc := NewUserService(store, &mockSettingsStore{})

	got, err := svc.List(t.Context())
	require.NoError(t, err)
	require.Equal(t, []domain.User{u1, u2}, got)
}

func TestListAllShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListAll(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestListAllShouldReturnErrNotFoundWhenCallerDoesNotExist(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})

	_, err := svc.ListAll(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestListAllShouldReturnErrForbiddenWhenCallerIsNotAdmin(t *testing.T) {
	var member domain.User
	member.ID = "42"
	member.Role = domain.RoleMember

	store := &mockUserStore{users: []domain.User{member}}
	svc := NewUserService(store, &mockSettingsStore{})

	_, err := svc.ListAll(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestListAllShouldReturnErrorWhenListFails(t *testing.T) {
	var admin domain.User
	admin.ID = "42"
	admin.Role = domain.RoleAdmin

	store := &mockUserStore{users: []domain.User{admin}, listErr: domain.ErrIO}
	svc := NewUserService(store, &mockSettingsStore{})

	_, err := svc.ListAll(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListAllShouldReturnAllUsersWhenCallerIsAdmin(t *testing.T) {
	var admin domain.User
	admin.ID = "1"
	admin.Role = domain.RoleAdmin

	var member domain.User
	member.ID = "2"
	member.Role = domain.RoleMember

	store := &mockUserStore{users: []domain.User{admin, member}}
	svc := NewUserService(store, &mockSettingsStore{})

	got, err := svc.ListAll(t.Context(), "1")
	require.NoError(t, err)
	require.Equal(t, []domain.User{admin, member}, got)
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

func TestGetByEmailShouldReturnErrorWhenGetByEmailFails(t *testing.T) {
	store := &mockUserStore{getByEmailErr: domain.ErrIO}
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

func TestRegisterShouldReturnErrRegistrationDisabledWhenSettingsNotFoundAndUsersExist(t *testing.T) {
	var existingUser domain.User
	existingUser.ID = "1"
	existingUser.Email = "bob@example.com"

	store := &mockUserStore{users: []domain.User{existingUser}}
	settings := &mockSettingsStore{notFound: true}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrRegistrationDisabled)
}

func TestRegisterShouldReturnErrRegistrationDisabledWhenRegistrationIsDisabled(t *testing.T) {
	var existingUser domain.User
	existingUser.ID = "1"
	existingUser.Email = "bob@example.com"

	store := &mockUserStore{users: []domain.User{existingUser}}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: false}}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrRegistrationDisabled)
}

func TestRegisterShouldSucceedWithAdminRoleWhenFirstUserAndRegistrationIsDisabled(t *testing.T) {
	store := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: false}}
	svc := NewUserService(store, settings)

	user, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.NoError(t, err)
	require.Equal(t, domain.RoleAdmin, user.Role)
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

func TestRegisterShouldReturnErrorWhenDefineRoleCountFails(t *testing.T) {
	// Store is empty: canRegister → Count call 1 returns 0 (bootstrap, no error).
	// defineRole → GetByEmail returns ErrNotFound → Count call 2 fails.
	store := &mockUserStore{countErr: domain.ErrIO, countErrOnNthCall: 2}
	settings := &mockSettingsStore{}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRegisterShouldReturnErrorWhenDefineRoleGetByEmailFails(t *testing.T) {
	// canRegister → Count returns 0 (bootstrap). defineRole → GetByEmail fails with unexpected error.
	store := &mockUserStore{getByEmailErr: domain.ErrIO}
	settings := &mockSettingsStore{}
	svc := NewUserService(store, settings)

	_, err := svc.Register(t.Context(), "alice@example.com", "password")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRegisterShouldReturnErrorWhenSettingsLoadFails(t *testing.T) {
	var existingUser domain.User
	existingUser.ID = "1"
	existingUser.Email = "bob@example.com"

	store := &mockUserStore{users: []domain.User{existingUser}}
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

func TestGetSettingsShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetSettings(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetSettingsShouldReturnDefaultSettingsWhenNotFound(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{notFound: true})

	got, err := svc.GetSettings(t.Context())
	require.NoError(t, err)
	require.Equal(t, domain.AppSettings{}, got)
}

func TestGetSettingsShouldReturnErrorWhenStoreFails(t *testing.T) {
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{err: domain.ErrIO})

	_, err := svc.GetSettings(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetSettingsShouldReturnSettingsWhenFound(t *testing.T) {
	settings := domain.AppSettings{RegistrationEnabled: true}
	svc := NewUserService(&mockUserStore{}, &mockSettingsStore{settings: settings})

	got, err := svc.GetSettings(t.Context())
	require.NoError(t, err)
	require.Equal(t, settings, got)
}

func TestUserServiceShouldCompleteFullCycleWhenOperationsAreValid(t *testing.T) {
	store := &mockUserStore{}
	settings := &mockSettingsStore{settings: domain.AppSettings{RegistrationEnabled: true}}
	svc := NewUserService(store, settings)

	// Register first user → must become admin.
	admin, err := svc.Register(t.Context(), "alice@example.com", "s3cr3t")
	require.NoError(t, err)
	require.Equal(t, domain.RoleAdmin, admin.Role)
	require.Equal(t, "alice@example.com", admin.Email)
	require.NotEmpty(t, admin.ID)
	require.False(t, admin.CreatedAt.IsZero())

	// Register second user → must become member.
	member, err := svc.Register(t.Context(), "bob@example.com", "p4ssw0rd")
	require.NoError(t, err)
	require.Equal(t, domain.RoleMember, member.Role)
	require.Equal(t, "bob@example.com", member.Email)
	require.NotEmpty(t, member.ID)

	// GetByID must return the stored admin.
	got, err := svc.GetByID(t.Context(), admin.ID)
	require.NoError(t, err)
	require.Equal(t, admin, got)

	// GetByEmail must return the stored member.
	got, err = svc.GetByEmail(t.Context(), "bob@example.com")
	require.NoError(t, err)
	require.Equal(t, member, got)

	// ListAll as admin must return both users.
	all, err := svc.ListAll(t.Context(), admin.ID)
	require.NoError(t, err)
	require.Len(t, all, 2)

	// UpdateRegistrationEnabled must persist the change.
	err = svc.UpdateRegistrationEnabled(t.Context(), admin.ID, false)
	require.NoError(t, err)
	require.False(t, settings.settings.RegistrationEnabled)

	// A new registration must now be rejected.
	_, err = svc.Register(t.Context(), "carol@example.com", "pw")
	require.ErrorIs(t, err, domain.ErrRegistrationDisabled)
}
