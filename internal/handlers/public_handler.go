package handlers

import (
	"encoding/base64"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/database"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/session"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/validation"
	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

const defaultPublicLogoURL = "/assets/web-app-manifest-192x192.png"

// PublicHandler serves unauthenticated public routes.
type PublicHandler struct {
	db        *gorm.DB
	settings  *repositories.SettingsRepository
	seo       *services.SEOService
	logger    *slog.Logger
	analytics *services.AnalyticsService
	sessions  *session.Manager
	version   string
	commit    string
	buildTime string
}

// NewPublicHandler constructs a PublicHandler.
func NewPublicHandler(db *gorm.DB, settings *repositories.SettingsRepository, seo *services.SEOService, analytics *services.AnalyticsService, sessions *session.Manager, logger *slog.Logger, version string, commit string, buildTime string) *PublicHandler {
	return &PublicHandler{db: db, settings: settings, seo: seo, analytics: analytics, sessions: sessions, logger: logger, version: version, commit: commit, buildTime: buildTime}
}

// Home renders the public landing page.
func (h *PublicHandler) Home(c *gin.Context) {
	settings, err := h.settings.Current()
	if err != nil || settings == nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Page unavailable", "The public page could not be loaded."))
		return
	}

	baseURL := strings.TrimSpace(settings.PublicBaseURL)
	if baseURL == "" {
		baseURL = "http://" + c.Request.Host
	}
	getURL := validation.JoinURLPath(baseURL, "/get")
	qrDataURI := ""
	if png, err := qrcode.Encode(getURL, qrcode.Medium, 220); err == nil {
		qrDataURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	} else if h.logger != nil {
		h.logger.Error("generate landing QR code", slog.String("error", err.Error()))
	}

	c.HTML(http.StatusOK, "public/home.html", gin.H{
		"Title":           settings.SiteName,
		"SiteName":        settings.SiteName,
		"SiteTagline":     settings.SiteTagline,
		"SiteDescription": settings.SiteDescription,
		"LogoURL":         publicLogoURL(settings),
		"AndroidURL":      settings.AndroidURL,
		"AppleURL":        settings.AppleURL,
		"DefaultURL":      settings.DefaultURL,
		"SupportEmail":    settings.SupportEmail,
		"GetURL":          getURL,
		"QRCode":          template.URL(qrDataURI),
		"SEO":             h.seo.Home(settings),
		"Theme":           services.FrontendTheme(settings),
		"PublicChrome":    publicChrome(settings),
		"Footer":          publicFooter(settings),
	})
}

// Health reports service and database readiness.
func (h *PublicHandler) Health(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"service": "ty2shorten-url",
			"error":   "database is not configured",
		})
		return
	}

	if err := database.Ping(h.db); err != nil {
		if h.logger != nil {
			h.logger.Error("health database ping failed", slog.String("error", err.Error()))
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"service": "ty2shorten-url",
			"error":   "database unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"service":    "ty2shorten-url",
		"version":    h.version,
		"commit":     h.commit,
		"build_time": h.buildTime,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstInitial(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "T"
	}
	return strings.ToUpper(string([]rune(trimmed)[0]))
}

func publicChrome(settings *models.AppSetting) gin.H {
	siteName := firstNonEmpty(settings.SiteName, "ty2shorten-url")
	return gin.H{
		"Enabled":    true,
		"SiteName":   siteName,
		"Initial":    firstInitial(siteName),
		"LogoURL":    publicLogoURL(settings),
		"PrivacyURL": "/privacy",
		"TermsURL":   "/terms",
	}
}

func publicLogoURL(settings *models.AppSetting) string {
	if settings == nil {
		return defaultPublicLogoURL
	}
	return firstNonEmpty(settings.LogoURL, defaultPublicLogoURL)
}

func publicFooter(settings *models.AppSetting) gin.H {
	return gin.H{
		"Enabled":      settings.FooterEnabled,
		"BrandText":    firstNonEmpty(settings.FooterBrandText, settings.SiteName),
		"Description":  settings.FooterDescription,
		"Copyright":    settings.CopyrightText,
		"SupportText":  settings.SupportText,
		"SupportEmail": settings.SupportEmail,
		"Address":      settings.FooterAddress,
		"PrivacyURL":   "/privacy",
		"TermsURL":     "/terms",
	}
}

// Placeholder renders public legal pages managed by administrator Markdown.
func (h *PublicHandler) Placeholder(c *gin.Context) {
	title := "Privacy Policy"
	markdownField := func(settings *models.AppSetting) string {
		return settings.PrivacyPolicyMarkdown
	}
	if c.FullPath() == "/terms" {
		title = "Terms of Use"
		markdownField = func(settings *models.AppSetting) string {
			return settings.TermsMarkdown
		}
	}
	settings, err := h.settings.Current()
	if err != nil || settings == nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Page unavailable", "The public page could not be loaded."))
		return
	}

	markdown := markdownField(settings)
	configured := strings.TrimSpace(markdown) != ""
	message := ""
	if !configured {
		message = "This page has not been configured yet."
	}
	c.HTML(http.StatusOK, "public/legal.html", gin.H{
		"Title":        title,
		"Heading":      title,
		"Message":      message,
		"Configured":   configured,
		"Content":      renderAboutMarkdown(markdown),
		"SEO":          services.NoIndexWithBrand(title, settings),
		"Theme":        services.FrontendTheme(settings),
		"PublicChrome": publicChrome(settings),
		"CSRFToken":    csrfTokenOrEmpty(h.sessions, c),
	})
}

// NotFound renders a custom public 404 page.
func (h *PublicHandler) NotFound(c *gin.Context) {
	data := notFoundView("Page not found", "The page you requested does not exist or has moved.")
	if settings, err := h.settings.Current(); err == nil && settings != nil {
		data["PublicChrome"] = publicChrome(settings)
		data["SEO"] = services.NoIndexWithBrand("Page not found", settings)
		data["Theme"] = services.FrontendTheme(settings)
	}
	c.HTML(http.StatusNotFound, "public/not_found.html", data)
}

func csrfTokenOrEmpty(sessions *session.Manager, c *gin.Context) string {
	if sessions == nil {
		return ""
	}
	token, err := sessions.CSRFToken(c)
	if err != nil {
		return ""
	}
	return token
}
