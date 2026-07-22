package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateDefaultSQLiteConfig(t *testing.T) {
	cfg := DefaultConfig("./storage/ty2shorten.db")

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if cfg.Database.Driver != DriverSQLite {
		t.Fatalf("driver = %q, want sqlite", cfg.Database.Driver)
	}
}

func TestValidateRejectsUnknownDriver(t *testing.T) {
	cfg := DefaultConfig("./storage/ty2shorten.db")
	cfg.Database.Driver = "postgres"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate returned nil, want unknown driver error")
	}
}

func TestValidateMySQLFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*BootstrapConfig)
	}{
		{name: "empty host", mutate: func(cfg *BootstrapConfig) { cfg.Database.MySQL.Host = "" }},
		{name: "url host", mutate: func(cfg *BootstrapConfig) { cfg.Database.MySQL.Host = "http://127.0.0.1" }},
		{name: "metadata ip", mutate: func(cfg *BootstrapConfig) { cfg.Database.MySQL.Host = "169.254.169.254" }},
		{name: "bad port", mutate: func(cfg *BootstrapConfig) { cfg.Database.MySQL.Port = 70000 }},
		{name: "missing name", mutate: func(cfg *BootstrapConfig) { cfg.Database.MySQL.Name = "" }},
		{name: "missing user", mutate: func(cfg *BootstrapConfig) { cfg.Database.MySQL.User = "" }},
		{name: "missing password", mutate: func(cfg *BootstrapConfig) { cfg.Database.MySQL.Password = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validMySQLConfig()
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate returned nil, want error")
			}
		})
	}
}

func TestMySQLDSNEscapesPassword(t *testing.T) {
	cfg := validMySQLConfig()
	cfg.Database.MySQL.Password = "p@ss:word/with?chars"

	dsn, err := MySQLDSN(cfg.Database.MySQL)
	if err != nil {
		t.Fatalf("MySQLDSN returned error: %v", err)
	}
	if !strings.Contains(dsn, "parseTime=true") || !strings.Contains(dsn, "charset=utf8mb4") {
		t.Fatalf("DSN missing expected parameters: %s", dsn)
	}
	if redacted := cfg.Database.MySQL.Redacted(); redacted.Password != "" {
		t.Fatal("redacted MySQL config still contains password")
	}
}

func TestSaveWritesAtomicallyWith0600Permissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	cfg := DefaultConfig(filepath.Join(t.TempDir(), "app.db"))

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 0600", got)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Database.SQLite.Path != cfg.Database.SQLite.Path {
		t.Fatalf("loaded path = %q, want %q", loaded.Database.SQLite.Path, cfg.Database.SQLite.Path)
	}
}

func TestProofValidation(t *testing.T) {
	manager := NewProofManager("test-secret", time.Minute)
	cfg := validMySQLConfig()
	hash := HashConfig(cfg.Database)

	proof, err := manager.Sign(hash)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}
	if err := manager.Verify(proof, hash); err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if err := manager.Verify(proof, HashConfig(DefaultConfig("./storage/app.db").Database)); err == nil {
		t.Fatal("Verify returned nil for changed config hash")
	}
}

func TestProofExpires(t *testing.T) {
	manager := NewProofManager("test-secret", time.Nanosecond)
	hash := HashConfig(validMySQLConfig().Database)

	proof, err := manager.Sign(hash)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}
	time.Sleep(time.Millisecond)
	if err := manager.Verify(proof, hash); err == nil {
		t.Fatal("Verify returned nil for expired proof")
	}
}

func TestConnectorRejectsUnknownDriver(t *testing.T) {
	connector := NewConnector()
	cfg := DefaultConfig("./storage/app.db")
	cfg.Database.Driver = "bad"

	if _, err := connector.Connect(context.Background(), cfg.Database); err == nil {
		t.Fatal("Connect returned nil error for unknown driver")
	}
}

func validMySQLConfig() BootstrapConfig {
	return BootstrapConfig{
		Version: 1,
		Database: DatabaseConfig{
			Driver: DriverMySQL,
			MySQL: MySQLConfig{
				Host:     "127.0.0.1",
				Port:     3306,
				Name:     "ty2shorten",
				User:     "ty2shorten",
				Password: "secret",
				Parameters: map[string]string{
					"charset":   "utf8mb4",
					"parseTime": "true",
				},
			},
		},
	}
}
