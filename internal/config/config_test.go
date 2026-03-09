package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/config"
)

func TestLoad_MissingRequired(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "")
	t.Setenv("MEDIA_STORAGE_PATH", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "STORAGE_DATA_PATH")
	assert.Contains(t, err.Error(), "MEDIA_STORAGE_PATH")
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "dev", cfg.AppEnv)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("APP_ENV", "production")
	t.Setenv("STORAGE_DATA_PATH", "/var/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/var/media")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.ServerPort)
	assert.Equal(t, "production", cfg.AppEnv)
	assert.Equal(t, "/var/data", cfg.StorageDataPath)
	assert.Equal(t, "/var/media", cfg.MediaStoragePath)
	assert.Equal(t, "debug", cfg.LogLevel)
}
