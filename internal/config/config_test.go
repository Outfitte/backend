package config_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/config"
)

func TestLoadShouldErrorWhenRequiredVarsAreMissing(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "")
	t.Setenv("MEDIA_STORAGE_PATH", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "STORAGE_DATA_PATH")
	assert.Contains(t, err.Error(), "MEDIA_STORAGE_PATH")
}

func TestLoadShouldUseDefaultsWhenOptionalVarsAreUnset(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.ServerPort)
	assert.Equal(t, "dev", cfg.AppEnv)
	assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
}

func TestLoadShouldErrorWhenOnlyOneRequiredVarIsMissing(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "STORAGE_DATA_PATH")
	assert.Contains(t, err.Error(), "MEDIA_STORAGE_PATH")
}

func TestLoadShouldErrorWhenServerPortIsInvalid(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("SERVER_PORT", "notaport")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SERVER_PORT")
}

func TestLoadShouldReadAllVarsWhenAllAreSet(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("APP_ENV", "production")
	t.Setenv("STORAGE_DATA_PATH", "/var/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/var/media")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.ServerPort)
	assert.Equal(t, "production", cfg.AppEnv)
	assert.Equal(t, "/var/data", cfg.StorageDataPath)
	assert.Equal(t, "/var/media", cfg.MediaStoragePath)
	assert.Equal(t, slog.LevelDebug, cfg.LogLevel)
}

func TestLoadShouldErrorWhenLogLevelIsUnrecognized(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("LOG_LEVEL", "verbose")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LOG_LEVEL")
}
