package json_test

import (
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewRepositoriesShouldReturnPopulatedRepositories(t *testing.T) {
	repos := json.NewRepositories(t.TempDir())

	require.NotNil(t, repos.Items)
	require.NotNil(t, repos.Users)
	require.NotNil(t, repos.Sessions)
	require.NotNil(t, repos.Locations)
	require.NotNil(t, repos.WearLogs)
	require.NotNil(t, repos.AppSettings)
}

func TestNewRepositoriesShouldReturnCorrectInterfaceTypes(t *testing.T) {
	repos := json.NewRepositories(t.TempDir())

	require.Implements(t, (*ports.ItemRepository)(nil), repos.Items)
	require.Implements(t, (*ports.UserRepository)(nil), repos.Users)
	require.Implements(t, (*ports.SessionRepository)(nil), repos.Sessions)
	require.Implements(t, (*ports.LocationRepository)(nil), repos.Locations)
	require.Implements(t, (*ports.WearLogRepository)(nil), repos.WearLogs)
	require.Implements(t, (*ports.AppSettingsRepository)(nil), repos.AppSettings)
}
