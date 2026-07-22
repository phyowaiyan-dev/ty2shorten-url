package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/bootstrap"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/config"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/handlers"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/middleware"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/session"
	"gorm.io/gorm"
)

// RouteDependencies groups route-level dependencies.
type RouteDependencies struct {
	DB           *gorm.DB
	Logger       *slog.Logger
	Version      string
	Commit       string
	BuildTime    string
	SettingsRepo *repositories.SettingsRepository
	Setup        *services.SetupService
	Auth         *services.AuthService
	Settings     *services.SettingsService
	Links        *services.LinkService
	Account      *services.AccountService
	Audit        *services.AuditService
	Media        *services.MediaService
	Dashboard    *services.DashboardService
	Redirects    *services.RedirectService
	SEO          *services.SEOService
	Analytics    *services.AnalyticsService
	Sessions     *session.Manager
	CSRFHandler  gin.HandlerFunc
	AuthHandler  gin.HandlerFunc
}

// BootstrapRouteDependencies groups first-stage setup dependencies.
type BootstrapRouteDependencies struct {
	Config   config.Config
	Sessions *session.Manager
}

// RegisterBootstrapRoutes attaches database-bootstrap routes.
func RegisterBootstrapRoutes(router *gin.Engine, deps BootstrapRouteDependencies) {
	handler := handlers.NewDatabaseSetupHandler(
		deps.Config.AppConfigPath,
		bootstrap.NewConnector(),
		bootstrap.NewProofManager(deps.Config.SessionSecret, 10*time.Minute),
		deps.Sessions,
	)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "setup_required",
			"service":    "ty2shorten-url",
			"stage":      "database_bootstrap",
			"version":    deps.Config.Version,
			"commit":     deps.Config.Commit,
			"build_time": deps.Config.BuildTime,
		})
	})
	router.GET("/setup/database", handler.Show)
	router.POST("/setup/database/test", middleware.CSRFRequired(deps.Sessions), handler.Test)
	router.POST("/setup/database", middleware.CSRFRequired(deps.Sessions), handler.Store)
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Redirect(http.StatusSeeOther, "/setup/database")
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database setup is required"})
	})
}

// RegisterRoutes attaches application routes to the router.
func RegisterRoutes(router *gin.Engine, deps RouteDependencies) {
	publicHandler := handlers.NewPublicHandler(deps.DB, deps.SettingsRepo, deps.SEO, deps.Analytics, deps.Sessions, deps.Logger, deps.Version, deps.Commit, deps.BuildTime)
	setupHandler := handlers.NewSetupHandler(deps.Setup, deps.Sessions)
	authHandler := handlers.NewAuthHandler(deps.Auth, deps.SettingsRepo, deps.Sessions)
	adminHandler := handlers.NewAdminHandler(deps.Dashboard, deps.Settings, deps.Links, deps.Account, deps.Audit, deps.Media, deps.Analytics, deps.Sessions)
	redirectHandler := handlers.NewRedirectHandler(deps.Redirects)
	seoHandler := handlers.NewSEOHandler(deps.SEO)
	mediaHandler := handlers.NewMediaHandler(deps.Media)

	router.GET("/", publicHandler.Home)
	router.GET("/health", publicHandler.Health)
	router.GET("/media/:name", mediaHandler.Show)
	router.GET("/robots.txt", seoHandler.Robots)
	router.GET("/sitemap.xml", seoHandler.Sitemap)
	router.GET("/privacy", publicHandler.Placeholder)
	router.GET("/terms", publicHandler.Placeholder)
	router.GET("/setup", setupHandler.Show)
	router.POST("/setup", deps.CSRFHandler, setupHandler.Store)
	router.GET("/admin/login", authHandler.ShowLogin)
	router.POST("/admin/login", deps.CSRFHandler, authHandler.Login)
	router.POST("/admin/logout", deps.AuthHandler, deps.CSRFHandler, authHandler.Logout)
	router.GET("/admin", deps.AuthHandler, adminHandler.Dashboard)
	router.GET("/admin/settings", deps.AuthHandler, adminHandler.Settings)
	router.POST("/admin/settings", deps.AuthHandler, deps.CSRFHandler, adminHandler.UpdateSettings)
	router.GET("/admin/settings/branding", deps.AuthHandler, adminHandler.BrandingSettings)
	router.POST("/admin/settings/branding", deps.AuthHandler, deps.CSRFHandler, adminHandler.UpdateBrandingSettings)
	router.GET("/admin/settings/seo", deps.AuthHandler, adminHandler.SEOSettings)
	router.POST("/admin/settings/seo", deps.AuthHandler, deps.CSRFHandler, adminHandler.UpdateSEOSettings)
	router.GET("/admin/settings/footer", deps.AuthHandler, adminHandler.FooterSettings)
	router.POST("/admin/settings/footer", deps.AuthHandler, deps.CSRFHandler, adminHandler.UpdateFooterSettings)
	router.GET("/admin/settings/legal", deps.AuthHandler, adminHandler.LegalContentSettings)
	router.POST("/admin/settings/legal", deps.AuthHandler, deps.CSRFHandler, adminHandler.UpdateLegalContentSettings)
	router.GET("/admin/links", deps.AuthHandler, adminHandler.Links)
	router.GET("/admin/links/new", deps.AuthHandler, adminHandler.NewLink)
	router.POST("/admin/links", deps.AuthHandler, deps.CSRFHandler, adminHandler.CreateLink)
	router.GET("/admin/links/:id/edit", deps.AuthHandler, adminHandler.EditLink)
	router.POST("/admin/links/:id", deps.AuthHandler, deps.CSRFHandler, adminHandler.UpdateLink)
	router.POST("/admin/links/:id/delete", deps.AuthHandler, deps.CSRFHandler, adminHandler.DeleteLink)
	router.GET("/admin/account", deps.AuthHandler, adminHandler.Account)
	router.POST("/admin/account/profile", deps.AuthHandler, deps.CSRFHandler, adminHandler.UpdateAccountProfile)
	router.POST("/admin/account/password", deps.AuthHandler, deps.CSRFHandler, adminHandler.ChangePassword)
	router.GET("/admin/about", deps.AuthHandler, adminHandler.About)
	router.GET("/admin/audit-logs", deps.AuthHandler, adminHandler.AuditLogs)
	router.GET("/admin/audit-logs/:id", deps.AuthHandler, adminHandler.AuditLogDetail)
	router.GET("/admin/analytics", deps.AuthHandler, adminHandler.Analytics)
	router.GET("/admin/analytics/export", deps.AuthHandler, adminHandler.ExportAnalytics)
	router.GET("/admin/analytics/sessions/:session_id", deps.AuthHandler, adminHandler.AnalyticsSessionDetail)
	router.POST("/admin/analytics/cleanup", deps.AuthHandler, deps.CSRFHandler, adminHandler.CleanupAnalytics)
	router.GET("/admin/settings/analytics", deps.AuthHandler, adminHandler.AnalyticsSettings)
	router.POST("/admin/settings/analytics", deps.AuthHandler, deps.CSRFHandler, adminHandler.UpdateAnalyticsSettings)
	router.GET("/android", redirectHandler.Android)
	router.GET("/apple", redirectHandler.Apple)
	router.GET("/get", redirectHandler.Get)
	router.GET("/r/:slug", redirectHandler.ShortLink)
	router.NoRoute(publicHandler.NotFound)
}
