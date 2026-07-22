package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	mydriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	DriverSQLite = "sqlite"
	DriverMySQL  = "mysql"

	configVersion = 1
)

var (
	ErrUnknownDriver = errors.New("unknown database driver")
	ErrInvalidConfig = errors.New("invalid bootstrap configuration")
)

// BootstrapConfig stores the database selection needed before the app database is opened.
type BootstrapConfig struct {
	Version  int            `json:"version"`
	Database DatabaseConfig `json:"database"`
}

// DatabaseConfig selects and configures the application database.
type DatabaseConfig struct {
	Driver string       `json:"driver"`
	SQLite SQLiteConfig `json:"sqlite,omitempty"`
	MySQL  MySQLConfig  `json:"mysql,omitempty"`
}

// SQLiteConfig contains SQLite database settings.
type SQLiteConfig struct {
	Path string `json:"path"`
}

// MySQLConfig contains MySQL database settings.
type MySQLConfig struct {
	Host       string            `json:"host"`
	Port       int               `json:"port"`
	Name       string            `json:"name"`
	User       string            `json:"user"`
	Password   string            `json:"password"`
	TLSMode    string            `json:"tls_mode,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// DefaultConfig returns a valid SQLite bootstrap config.
func DefaultConfig(sqlitePath string) BootstrapConfig {
	return BootstrapConfig{
		Version: configVersion,
		Database: DatabaseConfig{
			Driver: DriverSQLite,
			SQLite: SQLiteConfig{Path: sqlitePath},
			MySQL: MySQLConfig{
				Host: "127.0.0.1",
				Port: 3306,
				Parameters: map[string]string{
					"charset":   "utf8mb4",
					"parseTime": "true",
				},
			},
		},
	}
}

// Exists reports whether a bootstrap config file exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Load reads and validates a bootstrap config file.
func Load(path string) (BootstrapConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BootstrapConfig{}, fmt.Errorf("read bootstrap config: %w", err)
	}
	var cfg BootstrapConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return BootstrapConfig{}, fmt.Errorf("parse bootstrap config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return BootstrapConfig{}, err
	}
	return cfg, nil
}

// Save writes a bootstrap config atomically with 0600 permissions.
func Save(path string, cfg BootstrapConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create bootstrap config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal bootstrap config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temporary bootstrap config: %w", err)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return fmt.Errorf("chmod temporary bootstrap config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace bootstrap config: %w", err)
	}
	return nil
}

// Validate checks the bootstrap config without logging secrets.
func (c BootstrapConfig) Validate() error {
	if c.Version != configVersion {
		return fmt.Errorf("%w: unsupported version %d", ErrInvalidConfig, c.Version)
	}
	switch c.Database.Driver {
	case DriverSQLite:
		if strings.TrimSpace(c.Database.SQLite.Path) == "" {
			return fmt.Errorf("%w: sqlite path is required", ErrInvalidConfig)
		}
	case DriverMySQL:
		return c.Database.MySQL.Validate()
	default:
		return fmt.Errorf("%w: %q", ErrUnknownDriver, c.Database.Driver)
	}
	return nil
}

// Validate checks MySQL settings.
func (c MySQLConfig) Validate() error {
	host := strings.TrimSpace(c.Host)
	if host == "" {
		return fmt.Errorf("%w: mysql host is required", ErrInvalidConfig)
	}
	if strings.Contains(host, "://") {
		return fmt.Errorf("%w: mysql host must not include a URL scheme", ErrInvalidConfig)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("%w: mysql port is invalid", ErrInvalidConfig)
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("%w: mysql database name is required", ErrInvalidConfig)
	}
	if strings.TrimSpace(c.User) == "" {
		return fmt.Errorf("%w: mysql username is required", ErrInvalidConfig)
	}
	if c.Password == "" {
		return fmt.Errorf("%w: mysql password is required", ErrInvalidConfig)
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || isCloudMetadataIP(ip) {
			return fmt.Errorf("%w: mysql host is not allowed", ErrInvalidConfig)
		}
		return nil
	}
	if _, err := url.ParseRequestURI("mysql://" + host); err != nil || strings.ContainsAny(host, "/?#@") {
		return fmt.Errorf("%w: mysql host is malformed", ErrInvalidConfig)
	}
	return nil
}

// HashConfig returns a stable hash of the database config, including secrets.
func HashConfig(cfg DatabaseConfig) string {
	normalized := cfg
	if normalized.MySQL.Parameters == nil {
		normalized.MySQL.Parameters = map[string]string{}
	}
	data, _ := json.Marshal(normalized)
	sum := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// Redacted returns a copy safe for logs and templates.
func (c MySQLConfig) Redacted() MySQLConfig {
	c.Password = ""
	return c
}

// MySQLDSN builds a DSN with the official driver config helper.
func MySQLDSN(cfg MySQLConfig) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	params := map[string]string{
		"charset":           "utf8mb4",
		"parseTime":         "true",
		"timeout":           "7s",
		"loc":               "Local",
		"collation":         "utf8mb4_unicode_ci",
		"interpolateParams": "true",
	}
	for key, value := range cfg.Parameters {
		params[key] = value
	}
	mysqlCfg := mydriver.NewConfig()
	mysqlCfg.User = cfg.User
	mysqlCfg.Passwd = cfg.Password
	mysqlCfg.Net = "tcp"
	mysqlCfg.Addr = net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	mysqlCfg.DBName = cfg.Name
	mysqlCfg.Params = params
	if cfg.TLSMode != "" {
		mysqlCfg.TLSConfig = cfg.TLSMode
	}
	return mysqlCfg.FormatDSN(), nil
}

// Connector opens and tests configured databases.
type Connector struct{}

// NewConnector constructs a Connector.
func NewConnector() Connector {
	return Connector{}
}

// Connect opens the selected database.
func (c Connector) Connect(ctx context.Context, cfg DatabaseConfig) (*gorm.DB, error) {
	switch cfg.Driver {
	case DriverSQLite:
		return connectSQLite(cfg.SQLite.Path)
	case DriverMySQL:
		dsn, err := MySQLDSN(cfg.MySQL)
		if err != nil {
			return nil, err
		}
		db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
		if err != nil {
			return nil, errors.New("connect mysql database")
		}
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("get mysql sql handle: %w", err)
		}
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
		if err := pingContext(ctx, sqlDB.PingContext); err != nil {
			return nil, errors.New("ping mysql database")
		}
		return db, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownDriver, cfg.Driver)
	}
}

// Test verifies a database config using a short-lived connection.
func (c Connector) Test(ctx context.Context, cfg DatabaseConfig) error {
	db, err := c.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
	return err
}

func connectSQLite(path string) (*gorm.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%w: sqlite path is required", ErrInvalidConfig)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}
	dsn := fmt.Sprintf("%s?_busy_timeout=5000&_foreign_keys=on", path)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sqlite sql handle: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("enable sqlite WAL mode: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("set sqlite busy timeout: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	return db, nil
}

func isCloudMetadataIP(ip net.IP) bool {
	return ip.Equal(net.ParseIP("169.254.169.254"))
}

func pingContext(ctx context.Context, ping func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	return ping(ctx)
}

// ProofManager signs short-lived connection-test proofs.
type ProofManager struct {
	secret []byte
	ttl    time.Duration
}

// NewProofManager constructs a ProofManager.
func NewProofManager(secret string, ttl time.Duration) ProofManager {
	if strings.TrimSpace(secret) == "" {
		secret = "development-insecure-bootstrap-secret"
	}
	return ProofManager{secret: []byte(secret), ttl: ttl}
}

// Sign creates a proof tied to a database config hash.
func (m ProofManager) Sign(hash string) (string, error) {
	payload := map[string]any{
		"hash":    hash,
		"expires": time.Now().Add(m.ttl).UnixNano(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal proof: %w", err)
	}
	bodyPart := base64.RawURLEncoding.EncodeToString(body)
	return bodyPart + "." + m.sign(bodyPart), nil
}

// Verify validates a proof against a database config hash.
func (m ProofManager) Verify(proof string, hash string) error {
	bodyPart, sig, ok := strings.Cut(proof, ".")
	if !ok {
		return errors.New("invalid connection proof")
	}
	if subtle.ConstantTimeCompare([]byte(m.sign(bodyPart)), []byte(sig)) != 1 {
		return errors.New("invalid connection proof")
	}
	body, err := base64.RawURLEncoding.DecodeString(bodyPart)
	if err != nil {
		return errors.New("invalid connection proof")
	}
	var payload struct {
		Hash    string `json:"hash"`
		Expires int64  `json:"expires"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return errors.New("invalid connection proof")
	}
	if time.Now().UnixNano() > payload.Expires {
		return errors.New("connection proof expired")
	}
	if subtle.ConstantTimeCompare([]byte(payload.Hash), []byte(hash)) != 1 {
		return errors.New("connection proof does not match configuration")
	}
	return nil
}

func (m ProofManager) sign(body string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// CanonicalParameters returns a sorted copy of MySQL parameters.
func CanonicalParameters(params map[string]string) map[string]string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = params[key]
	}
	return out
}
