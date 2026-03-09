// Package config handles environment-variable-driven configuration.
// Empty string and unset environment variables are treated equivalently:
// optional vars fall back to their defaults, required vars produce an error.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	ServerPort       string
	AppEnv           string
	StorageDataPath  string
	MediaStoragePath string
	LogLevel         string
}

// Load reads configuration from environment variables, applies defaults,
// and returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{}
	loadFromEnv(cfg)
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadFromEnv(cfg *Config) {
	cfg.ServerPort = getEnv("SERVER_PORT", "8080")
	cfg.AppEnv = getEnv("APP_ENV", "dev")
	cfg.StorageDataPath = os.Getenv("STORAGE_DATA_PATH")
	cfg.MediaStoragePath = os.Getenv("MEDIA_STORAGE_PATH")
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
}

func (c *Config) validate() error {
	var missing []string
	if c.StorageDataPath == "" {
		missing = append(missing, "STORAGE_DATA_PATH")
	}
	if c.MediaStoragePath == "" {
		missing = append(missing, "MEDIA_STORAGE_PATH")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	port, err := strconv.Atoi(c.ServerPort)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("SERVER_PORT must be a valid port number (1-65535), got %q", c.ServerPort)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
