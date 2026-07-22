package config

import (
	"io"
	"log/slog"
	"testing"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_HOST", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("BASE_URL", "")

	cfg, err := Load(testLogger())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.AppEnv != defaultAppEnv {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, defaultAppEnv)
	}
	if cfg.AppHost != defaultAppHost {
		t.Fatalf("AppHost = %q, want %q", cfg.AppHost, defaultAppHost)
	}
	if cfg.AppPort != defaultAppPort {
		t.Fatalf("AppPort = %d, want %d", cfg.AppPort, defaultAppPort)
	}
	if cfg.DatabasePath != defaultDatabasePath {
		t.Fatalf("DatabasePath = %q, want %q", cfg.DatabasePath, defaultDatabasePath)
	}
	if cfg.BaseURL != defaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, defaultBaseURL)
	}
}

func TestValidateRequiresSessionSecretInProduction(t *testing.T) {
	cfg := Config{
		AppEnv:       "production",
		AppHost:      "127.0.0.1",
		AppPort:      defaultAppPort,
		DatabasePath: "./storage/test.db",
		BaseURL:      "https://example.com",
	}

	if err := cfg.Validate(testLogger()); err == nil {
		t.Fatal("Validate returned nil, want production SESSION_SECRET error")
	}
}

func TestValidateRejectsInvalidPort(t *testing.T) {
	cfg := Config{
		AppEnv:        "development",
		AppHost:       "127.0.0.1",
		AppPort:       70000,
		DatabasePath:  "./storage/test.db",
		SessionSecret: "secret",
		BaseURL:       defaultBaseURL,
	}

	if err := cfg.Validate(testLogger()); err == nil {
		t.Fatal("Validate returned nil, want invalid port error")
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
