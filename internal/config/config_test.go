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
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DSN", "/data/outfitte.db")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "STORAGE_DATA_PATH")
	assert.Contains(t, err.Error(), "MEDIA_STORAGE_PATH")
}

func TestLoadShouldErrorWhenOnlyOneRequiredVarIsMissing(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "")
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DSN", "/data/outfitte.db")

	_, err := config.Load()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "STORAGE_DATA_PATH")
	assert.Contains(t, err.Error(), "MEDIA_STORAGE_PATH")
}

func TestLoadShouldErrorWhenServerPortIsInvalid(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DSN", "/data/outfitte.db")
	t.Setenv("SERVER_PORT", "notaport")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SERVER_PORT")
}

func TestLoadShouldErrorWhenLogLevelIsUnrecognized(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DSN", "/data/outfitte.db")
	t.Setenv("LOG_LEVEL", "verbose")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LOG_LEVEL")
}

func TestLoadShouldErrorWhenJWTSecretIsTooShort(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("JWT_SECRET", "tooshort")
	t.Setenv("DB_DSN", "/data/outfitte.db")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoadShouldErrorWhenJWTSecretIsMissing(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DB_DSN", "/data/outfitte.db")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoadShouldErrorWhenDBDSNIsMissing(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DSN", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_DSN")
}

func TestLoadShouldUseDefaultsWhenOptionalVarsAreUnset(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DSN", "/data/outfitte.db")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "dev", cfg.AppEnv)
	assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
}

func TestLoadShouldDefaultToSQLiteDriverWhenDBDriverIsUnset(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DSN", "/data/outfitte.db")
	t.Setenv("DB_DRIVER", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "sqlite", cfg.DB.Driver)
}

func TestLoadShouldReadJWTSecretWhenSetAndLongEnough(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DSN", "/data/outfitte.db")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "a-secure-random-string-that-is-32-chars!!", cfg.JWTSecret)
}

func TestLoadShouldReadDBConfigWhenBothVarsAreSet(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/media")
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_DSN", "postgres://user:pass@host:5432/outfitte?sslmode=disable")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "postgres", cfg.DB.Driver)
	assert.Equal(t, "postgres://user:pass@host:5432/outfitte?sslmode=disable", cfg.DB.DSN)
}

func TestLoadShouldReadAllVarsWhenAllAreSet(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("APP_ENV", "production")
	t.Setenv("STORAGE_DATA_PATH", "/var/data")
	t.Setenv("MEDIA_STORAGE_PATH", "/var/media")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("JWT_SECRET", "a-secure-random-string-that-is-32-chars!!")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_DSN", "postgres://user:pass@host:5432/outfitte?sslmode=disable")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.ServerPort)
	assert.Equal(t, "production", cfg.AppEnv)
	assert.Equal(t, "/var/data", cfg.StorageDataPath)
	assert.Equal(t, "/var/media", cfg.MediaStoragePath)
	assert.Equal(t, slog.LevelDebug, cfg.LogLevel)
	assert.Equal(t, "a-secure-random-string-that-is-32-chars!!", cfg.JWTSecret)
	assert.Equal(t, "postgres", cfg.DB.Driver)
	assert.Equal(t, "postgres://user:pass@host:5432/outfitte?sslmode=disable", cfg.DB.DSN)
}
