package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/bootstrap"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/database"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/session"
)

// DatabaseSetupHandler serves first-stage database bootstrap setup.
type DatabaseSetupHandler struct {
	configPath string
	connector  bootstrap.Connector
	proofs     bootstrap.ProofManager
	sessions   *session.Manager
}

// NewDatabaseSetupHandler constructs a DatabaseSetupHandler.
func NewDatabaseSetupHandler(configPath string, connector bootstrap.Connector, proofs bootstrap.ProofManager, sessions *session.Manager) *DatabaseSetupHandler {
	return &DatabaseSetupHandler{configPath: configPath, connector: connector, proofs: proofs, sessions: sessions}
}

// Show renders the database bootstrap form.
func (h *DatabaseSetupHandler) Show(c *gin.Context) {
	h.render(c, http.StatusOK, bootstrap.DefaultConfig("./storage/ty2shorten.db"), "", "", nil)
}

// Test validates and tests a database config without saving it.
func (h *DatabaseSetupHandler) Test(c *gin.Context) {
	cfg := h.formConfig(c)
	if err := cfg.Validate(); err != nil {
		h.render(c, http.StatusUnprocessableEntity, cfg, "", "Check the database fields and try again.", map[string]string{"database": "Invalid database configuration."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 7*time.Second)
	defer cancel()
	if err := h.connector.Test(ctx, cfg.Database); err != nil {
		h.render(c, http.StatusBadGateway, cfg, "", "The database connection could not be verified.", map[string]string{"database": "Connection test failed."})
		return
	}

	hash := bootstrap.HashConfig(cfg.Database)
	proof, err := h.proofs.Sign(hash)
	if err != nil {
		h.render(c, http.StatusInternalServerError, cfg, "", "A connection proof could not be created.", nil)
		return
	}
	h.render(c, http.StatusOK, cfg, proof, "Connection test passed. You can save this database configuration.", nil)
}

// Store verifies the proof, connects, migrates, and saves the bootstrap config.
func (h *DatabaseSetupHandler) Store(c *gin.Context) {
	if bootstrap.Exists(h.configPath) {
		c.HTML(http.StatusForbidden, "public/forbidden.html", forbiddenView("Access denied", "Database setup has already been completed."))
		return
	}
	cfg := h.formConfig(c)
	if err := cfg.Validate(); err != nil {
		h.render(c, http.StatusUnprocessableEntity, cfg, "", "Check the database fields and try again.", map[string]string{"database": "Invalid database configuration."})
		return
	}
	proof := c.PostForm("connection_proof")
	if err := h.proofs.Verify(proof, bootstrap.HashConfig(cfg.Database)); err != nil {
		h.render(c, http.StatusForbidden, cfg, "", "Test this exact database configuration before saving.", map[string]string{"database": "Connection proof is missing, expired, or does not match."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 7*time.Second)
	defer cancel()
	db, err := h.connector.Connect(ctx, cfg.Database)
	if err != nil {
		h.render(c, http.StatusBadGateway, cfg, "", "The database connection could not be opened.", map[string]string{"database": "Connection failed."})
		return
	}
	if err := database.AutoMigrate(db); err != nil {
		_ = database.Close(db)
		h.render(c, http.StatusInternalServerError, cfg, "", "Database migrations could not be applied.", nil)
		return
	}
	if err := database.Close(db); err != nil {
		h.render(c, http.StatusInternalServerError, cfg, "", "Database setup could not finish cleanly.", nil)
		return
	}
	if err := bootstrap.Save(h.configPath, cfg); err != nil {
		h.render(c, http.StatusInternalServerError, cfg, "", "Database configuration could not be saved.", nil)
		return
	}

	c.HTML(http.StatusOK, "public/database_setup_done.html", gin.H{
		"Title": "Database setup complete",
	})
}

func (h *DatabaseSetupHandler) render(c *gin.Context, status int, cfg bootstrap.BootstrapConfig, proof string, message string, fieldErrors map[string]string) {
	csrfToken, err := h.sessions.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Database setup unavailable", "A secure form token could not be created."))
		return
	}
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}
	redacted := cfg
	redacted.Database.MySQL = redacted.Database.MySQL.Redacted()
	c.HTML(status, "public/database_setup.html", gin.H{
		"Title":       "Database setup",
		"CSRFToken":   csrfToken,
		"Config":      redacted,
		"Proof":       proof,
		"Message":     message,
		"FieldErrors": fieldErrors,
	})
}

func (h *DatabaseSetupHandler) formConfig(c *gin.Context) bootstrap.BootstrapConfig {
	driver := c.PostForm("driver")
	if driver == "" {
		driver = bootstrap.DriverSQLite
	}
	cfg := bootstrap.BootstrapConfig{
		Version: 1,
		Database: bootstrap.DatabaseConfig{
			Driver: driver,
			SQLite: bootstrap.SQLiteConfig{
				Path: c.PostForm("sqlite_path"),
			},
			MySQL: bootstrap.MySQLConfig{
				Host:     c.PostForm("mysql_host"),
				Port:     parsePort(c.PostForm("mysql_port")),
				Name:     c.PostForm("mysql_name"),
				User:     c.PostForm("mysql_user"),
				Password: c.PostForm("mysql_password"),
				TLSMode:  c.PostForm("mysql_tls_mode"),
				Parameters: map[string]string{
					"charset":   "utf8mb4",
					"parseTime": "true",
				},
			},
		},
	}
	if cfg.Database.SQLite.Path == "" {
		cfg.Database.SQLite.Path = "./storage/ty2shorten.db"
	}
	if cfg.Database.MySQL.Host == "" {
		cfg.Database.MySQL.Host = "127.0.0.1"
	}
	if cfg.Database.MySQL.Port == 0 {
		cfg.Database.MySQL.Port = 3306
	}
	return cfg
}

func parsePort(value string) int {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return port
}
