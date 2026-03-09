package config

import (
	"fmt"
	"os"
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
	cfg := &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		AppEnv:           getEnv("APP_ENV", "dev"),
		StorageDataPath:  os.Getenv("STORAGE_DATA_PATH"),
		MediaStoragePath: os.Getenv("MEDIA_STORAGE_PATH"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
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
