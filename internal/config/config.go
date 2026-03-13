// Package config handles environment-variable-driven configuration.
// Empty string and unset environment variables are treated equivalently:
// optional vars fall back to their defaults, required vars produce an error.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	ServerPort       int
	AppEnv           string
	StorageDataPath  string
	MediaStoragePath string
	LogLevel         slog.Level
}

// Load reads configuration from environment variables, applies defaults,
// and returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := loadFromEnv(cfg); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadFromEnv(cfg *Config) error {
	portStr := getEnv("SERVER_PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("SERVER_PORT must be a valid port number (1-65535), got %q", portStr)
	}
	cfg.ServerPort = port

	cfg.AppEnv = getEnv("APP_ENV", "dev")
	cfg.StorageDataPath = os.Getenv("STORAGE_DATA_PATH")
	cfg.MediaStoragePath = os.Getenv("MEDIA_STORAGE_PATH")

	lvl, err := parseLogLevel(getEnv("LOG_LEVEL", "info"))
	if err != nil {
		return err
	}
	cfg.LogLevel = lvl
	return nil
}

func parseLogLevel(s string) (slog.Level, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(s)); err != nil {
		return 0, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error, got %q", s)
	}
	return lvl, nil
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
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
