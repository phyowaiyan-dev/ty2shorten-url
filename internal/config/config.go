package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	defaultAppEnv       = "development"
	defaultAppHost      = "127.0.0.1"
	defaultAppPort      = 8722
	defaultDatabasePath = "./storage/ty2shorten.db"
	defaultMediaPath    = "./storage/media"
	defaultBaseURL      = "http://localhost:8722"
	defaultProxies      = "127.0.0.1,::1"
)

// Config contains application settings loaded from environment variables.
type Config struct {
	AppEnv         string
	AppHost        string
	AppPort        int
	DatabasePath   string
	AppConfigPath  string
	MediaPath      string
	SessionSecret  string
	BaseURL        string
	TrustedProxies []string
	Version        string
	Commit         string
	BuildTime      string
}

// Load reads configuration from the process environment and validates it.
func Load(logger *slog.Logger) (Config, error) {
	cfg := Config{
		AppEnv:         getEnv("APP_ENV", defaultAppEnv),
		AppHost:        getEnv("APP_HOST", defaultAppHost),
		AppPort:        defaultAppPort,
		DatabasePath:   getEnv("DATABASE_PATH", defaultDatabasePath),
		AppConfigPath:  getEnv("APP_CONFIG_PATH", "./storage/config.json"),
		MediaPath:      getEnv("MEDIA_STORAGE_PATH", defaultMediaPath),
		SessionSecret:  os.Getenv("SESSION_SECRET"),
		BaseURL:        getEnv("BASE_URL", defaultBaseURL),
		TrustedProxies: splitCSV(getEnv("TRUSTED_PROXIES", defaultProxies)),
		Version:        getEnv("APP_VERSION", "dev"),
		Commit:         getEnv("APP_COMMIT", "unknown"),
		BuildTime:      getEnv("APP_BUILD_TIME", "unknown"),
	}

	if portValue := strings.TrimSpace(os.Getenv("APP_PORT")); portValue != "" {
		port, err := strconv.Atoi(portValue)
		if err != nil {
			return Config{}, fmt.Errorf("parse APP_PORT: %w", err)
		}
		cfg.AppPort = port
	}

	if err := cfg.Validate(logger); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate checks configuration for startup safety.
func (c Config) Validate(logger *slog.Logger) error {
	var validationErrors []error

	if c.AppEnv == "" {
		validationErrors = append(validationErrors, errors.New("APP_ENV is required"))
	}
	if c.AppEnv != "development" && c.AppEnv != "production" && c.AppEnv != "test" {
		validationErrors = append(validationErrors, fmt.Errorf("APP_ENV must be one of development, production, or test; got %q", c.AppEnv))
	}
	if strings.TrimSpace(c.AppHost) == "" {
		validationErrors = append(validationErrors, errors.New("APP_HOST is required"))
	}
	if c.AppPort < 1 || c.AppPort > 65535 {
		validationErrors = append(validationErrors, fmt.Errorf("APP_PORT must be between 1 and 65535; got %d", c.AppPort))
	}
	if strings.TrimSpace(c.DatabasePath) == "" {
		validationErrors = append(validationErrors, errors.New("DATABASE_PATH is required"))
	}
	if strings.TrimSpace(c.MediaPath) == "" {
		validationErrors = append(validationErrors, errors.New("MEDIA_STORAGE_PATH is required"))
	}
	if _, err := url.ParseRequestURI(c.BaseURL); err != nil {
		validationErrors = append(validationErrors, fmt.Errorf("BASE_URL must be a valid absolute URL: %w", err))
	}
	for _, proxy := range c.TrustedProxies {
		if proxy == "*" || proxy == "0.0.0.0/0" || proxy == "::/0" {
			validationErrors = append(validationErrors, errors.New("TRUSTED_PROXIES must not trust every proxy"))
		}
	}
	if strings.TrimSpace(c.SessionSecret) == "" {
		if c.AppEnv == "production" {
			validationErrors = append(validationErrors, errors.New("SESSION_SECRET is required in production"))
		} else if logger != nil {
			logger.Warn("SESSION_SECRET is not set; generated sessions will not be secure for shared development environments")
		}
	}

	return errors.Join(validationErrors...)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

// Addr returns the HTTP listen address.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.AppHost, c.AppPort)
}

// IsProduction reports whether the app is running with production settings.
func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
