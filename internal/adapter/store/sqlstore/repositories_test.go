package sqlstore_test

import (
	"testing"

	"github.com/outfitte/backend/internal/adapter/store/sqlstore"
	"github.com/outfitte/backend/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewRepositoriesShouldReturnPopulatedRepositories(t *testing.T) {
	db := openMigratedDB(t)

	repos := sqlstore.NewRepositories(db)

	require.NotNil(t, repos.Items)
	require.NotNil(t, repos.Users)
	require.NotNil(t, repos.Sessions)
	require.NotNil(t, repos.Locations)
	require.NotNil(t, repos.WearLogs)
	require.NotNil(t, repos.AppSettings)
	require.NotNil(t, repos.Outfits)
	require.NotNil(t, repos.OutfitLogs)
	require.NotNil(t, repos.OutfitLogTransactor)
	require.NotNil(t, repos.Shares)
	require.NotNil(t, repos.ItemTransfers)
	require.NotNil(t, repos.ItemTransferTransactor)
}

func TestNewRepositoriesShouldReturnCorrectInterfaceTypes(t *testing.T) {
	db := openMigratedDB(t)

	repos := sqlstore.NewRepositories(db)

	require.Implements(t, (*ports.ItemRepository)(nil), repos.Items)
	require.Implements(t, (*ports.UserRepository)(nil), repos.Users)
	require.Implements(t, (*ports.SessionRepository)(nil), repos.Sessions)
	require.Implements(t, (*ports.LocationRepository)(nil), repos.Locations)
	require.Implements(t, (*ports.WearLogRepository)(nil), repos.WearLogs)
	require.Implements(t, (*ports.AppSettingsRepository)(nil), repos.AppSettings)
	require.Implements(t, (*ports.OutfitRepository)(nil), repos.Outfits)
	require.Implements(t, (*ports.OutfitLogRepository)(nil), repos.OutfitLogs)
	require.Implements(t, (*ports.OutfitLogTransactor)(nil), repos.OutfitLogTransactor)
	require.Implements(t, (*ports.ShareRepository)(nil), repos.Shares)
	require.Implements(t, (*ports.ItemTransferRepository)(nil), repos.ItemTransfers)
	require.Implements(t, (*ports.ItemTransferTransactor)(nil), repos.ItemTransferTransactor)
}
