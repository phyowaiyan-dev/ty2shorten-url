package server

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	builtInAssets "github.com/phyowaiyan-dev/ty2shorten-url/assets"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/config"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/middleware"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/session"
	"github.com/phyowaiyan-dev/ty2shorten-url/web"
	"gorm.io/gorm"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

// Dependencies groups services required to build the HTTP server.
type Dependencies struct {
	Config        config.Config
	DB            *gorm.DB
	Logger        *slog.Logger
	BootstrapMode bool
}

// NewHTTPServer constructs the configured http.Server.
func NewHTTPServer(deps Dependencies) (*http.Server, error) {
	router, err := NewRouter(deps)
	if err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:              deps.Config.Addr(),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}, nil
}

// NewRouter builds the Gin engine with middleware, templates, assets, and routes.
func NewRouter(deps Dependencies) (*gin.Engine, error) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	switch {
	case deps.Config.IsProduction():
		gin.SetMode(gin.ReleaseMode)
	case deps.Config.AppEnv == "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	if err := router.SetTrustedProxies(deps.Config.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}

	templates, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	router.SetHTMLTemplate(templates)

	registerCoreMiddleware(router, deps.Logger)

	staticFiles, err := fs.Sub(web.FS, "static")
	if err != nil {
		return nil, fmt.Errorf("load embedded static assets: %w", err)
	}
	router.StaticFS("/static", http.FS(staticFiles))
	router.StaticFS("/assets", http.FS(builtInAssets.FS))

	sessions := session.NewManager(deps.Config.SessionSecret, deps.Config.IsProduction())
	if deps.BootstrapMode || deps.DB == nil {
		RegisterBootstrapRoutes(router, BootstrapRouteDependencies{
			Config:   deps.Config,
			Sessions: sessions,
		})
		return router, nil
	}

	admins := repositories.NewAdminRepository(deps.DB)
	settings := repositories.NewSettingsRepository(deps.DB)
	shortLinks := repositories.NewShortLinkRepository(deps.DB)
	audits := repositories.NewAuditRepository(deps.DB)
	analyticsRepo := repositories.NewAnalyticsRepository(deps.DB)
	setupService := services.NewSetupService(deps.DB, admins, settings, deps.Config.IsProduction(), deps.Config.BaseURL)
	authService := services.NewAuthService(admins, services.NewMemoryLoginLimiter(5, 15*time.Minute), deps.Logger)
	settingsService := services.NewSettingsService(settings, deps.Config.IsProduction(), deps.Logger)
	linkService := services.NewLinkService(shortLinks, settings, deps.Config.IsProduction(), deps.Logger)
	accountService := services.NewAccountService(admins)
	auditService := services.NewAuditService(audits)
	mediaService := services.NewMediaService(deps.Config.MediaPath)
	dashboardService := services.NewDashboardService(settings, shortLinks, deps.Config.AppEnv)
	redirectService := services.NewRedirectService(settings, shortLinks, deps.Logger)
	seoService := services.NewSEOService(settings)
	analyticsService := services.NewAnalyticsService(analyticsRepo, settings, deps.Config.IsProduction(), deps.Config.SessionSecret, deps.Logger)

	router.Use(middleware.SetupGate(setupService))
	router.Use(middleware.Analytics(analyticsService))
	RegisterRoutes(router, RouteDependencies{
		DB:           deps.DB,
		Logger:       deps.Logger,
		Version:      deps.Config.Version,
		Commit:       deps.Config.Commit,
		BuildTime:    deps.Config.BuildTime,
		SettingsRepo: settings,
		Setup:        setupService,
		Auth:         authService,
		Settings:     settingsService,
		Links:        linkService,
		Account:      accountService,
		Audit:        auditService,
		Media:        mediaService,
		Dashboard:    dashboardService,
		Redirects:    redirectService,
		SEO:          seoService,
		Analytics:    analyticsService,
		Sessions:     sessions,
		CSRFHandler:  middleware.CSRFRequired(sessions),
		AuthHandler:  middleware.AuthRequired(sessions),
	})

	return router, nil
}

// Shutdown gracefully stops the HTTP server.
func Shutdown(ctx context.Context, srv *http.Server) error {
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}

func parseTemplates() (*template.Template, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires key/value pairs")
			}
			out := map[string]any{}
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				out[key] = values[i+1]
			}
			return out, nil
		},
	}).ParseFS(web.FS, "templates/layouts/*.html", "templates/public/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse embedded templates: %w", err)
	}

	return tmpl, nil
}
