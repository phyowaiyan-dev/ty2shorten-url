package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/session"
	"github.com/phyowaiyan-dev/ty2shorten-url/web"
)

// AdminHandler serves authenticated administrator pages.
type AdminHandler struct {
	dashboard *services.DashboardService
	settings  *services.SettingsService
	links     *services.LinkService
	account   *services.AccountService
	audit     *services.AuditService
	media     *services.MediaService
	analytics *services.AnalyticsService
	session   *session.Manager
}

type auditLogRow struct {
	ID            uint
	TimeLabel     string
	TimeTitle     string
	ActionLabel   string
	ActionRaw     string
	ResourceLabel string
	Summary       string
	RequestID     string
	RequestShort  string
}

// NewAdminHandler constructs an AdminHandler.
func NewAdminHandler(dashboard *services.DashboardService, settings *services.SettingsService, links *services.LinkService, account *services.AccountService, audit *services.AuditService, media *services.MediaService, analytics *services.AnalyticsService, sessions *session.Manager) *AdminHandler {
	return &AdminHandler{dashboard: dashboard, settings: settings, links: links, account: account, audit: audit, media: media, analytics: analytics, session: sessions}
}

// Dashboard renders the initial administrator dashboard.
func (h *AdminHandler) Dashboard(c *gin.Context) {
	data, err := h.dashboard.Data()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Dashboard unavailable", "Dashboard data could not be loaded."))
		return
	}

	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Dashboard unavailable", "A secure form token could not be created."))
		return
	}

	c.HTML(http.StatusOK, "public/admin.html", h.withAdmin(c, gin.H{
		"Title":     "Admin dashboard",
		"CSRFToken": csrfToken,
		"Dashboard": data,
	}))
}

// Settings renders editable application settings.
func (h *AdminHandler) Settings(c *gin.Context) {
	form, err := h.settings.CurrentForm()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Settings unavailable", "Settings could not be loaded."))
		return
	}
	h.renderSettings(c, http.StatusOK, form, nil, c.Query("saved") == "1")
}

// UpdateSettings validates and stores application settings.
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	form := services.SettingsForm{
		SiteName:        c.PostForm("site_name"),
		SiteDescription: c.PostForm("site_description"),
		AndroidURL:      c.PostForm("android_url"),
		AppleURL:        c.PostForm("apple_url"),
		DefaultURL:      c.PostForm("default_url"),
		PublicBaseURL:   c.PostForm("public_base_url"),
		SupportEmail:    c.PostForm("support_email"),
	}
	if err := h.settings.Update(form); err != nil {
		var validationErr services.SettingsValidationError
		if errors.As(err, &validationErr) {
			h.renderSettings(c, http.StatusUnprocessableEntity, form, validationErr.FieldErrors, false)
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Settings not saved", "Settings could not be saved."))
		return
	}
	_ = h.audit.Record(services.AuditEvent{
		Action:       "settings.updated",
		ResourceType: "settings",
		Summary:      "General settings updated",
		Metadata: map[string]any{
			"site_name": form.SiteName,
		},
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		RequestID: requestID(c),
	})
	c.Redirect(http.StatusSeeOther, "/admin/settings?saved=1")
}

// BrandingSettings renders editable branding and media settings.
func (h *AdminHandler) BrandingSettings(c *gin.Context) {
	form, err := h.settings.CurrentBrandingForm()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Branding settings unavailable", "Branding settings could not be loaded."))
		return
	}
	h.renderBrandingSettings(c, http.StatusOK, form, nil, c.Query("saved") == "1")
}

// UpdateBrandingSettings stores branding settings and validated media uploads.
func (h *AdminHandler) UpdateBrandingSettings(c *gin.Context) {
	form := services.BrandingForm{
		SiteTagline:           c.PostForm("site_tagline"),
		LogoURL:               c.PostForm("logo_url"),
		DarkLogoURL:           c.PostForm("dark_logo_url"),
		FaviconURL:            c.PostForm("favicon_url"),
		AppleTouchIconURL:     c.PostForm("apple_touch_icon_url"),
		DefaultSocialImageURL: c.PostForm("default_social_image_url"),
	}
	fieldErrors := map[string]string{}
	uploads := []struct {
		field string
		kind  string
		set   func(string)
	}{
		{field: "logo_file", kind: "logo", set: func(value string) { form.LogoURL = value }},
		{field: "dark_logo_file", kind: "dark_logo", set: func(value string) { form.DarkLogoURL = value }},
		{field: "favicon_file", kind: "favicon", set: func(value string) { form.FaviconURL = value }},
		{field: "apple_touch_icon_file", kind: "apple_icon", set: func(value string) { form.AppleTouchIconURL = value }},
		{field: "default_social_image_file", kind: "social_image", set: func(value string) { form.DefaultSocialImageURL = value }},
	}
	if strings.HasPrefix(c.ContentType(), "multipart/") {
		for _, upload := range uploads {
			header, err := c.FormFile(upload.field)
			if err != nil {
				if errors.Is(err, http.ErrMissingFile) {
					continue
				}
				fieldErrors[upload.field] = "Upload could not be read."
				continue
			}
			url, err := h.media.Save(upload.kind, header)
			if err != nil {
				fieldErrors[upload.field] = mediaErrorMessage(err)
				continue
			}
			if url != "" {
				upload.set(url)
			}
		}
	}
	if len(fieldErrors) > 0 {
		h.renderBrandingSettings(c, http.StatusUnprocessableEntity, form, fieldErrors, false)
		return
	}
	if err := h.settings.UpdateBranding(form); err != nil {
		var validationErr services.SettingsValidationError
		if errors.As(err, &validationErr) {
			h.renderBrandingSettings(c, http.StatusUnprocessableEntity, form, validationErr.FieldErrors, false)
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Branding settings not saved", "Branding settings could not be saved."))
		return
	}
	_ = h.audit.Record(services.AuditEvent{Action: "branding.updated", ResourceType: "settings", Summary: "Branding settings updated", IPAddress: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"), RequestID: requestID(c)})
	c.Redirect(http.StatusSeeOther, "/admin/settings/branding?saved=1")
}

// FooterSettings renders editable public footer settings.
func (h *AdminHandler) FooterSettings(c *gin.Context) {
	form, err := h.settings.CurrentFooterForm()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Footer settings unavailable", "Footer settings could not be loaded."))
		return
	}
	h.renderFooterSettings(c, http.StatusOK, form, nil, c.Query("saved") == "1")
}

// UpdateFooterSettings stores public footer settings.
func (h *AdminHandler) UpdateFooterSettings(c *gin.Context) {
	form := services.FooterForm{
		FooterEnabled:     c.PostForm("footer_enabled") == "1",
		FooterBrandText:   c.PostForm("footer_brand_text"),
		FooterDescription: c.PostForm("footer_description"),
		CopyrightText:     c.PostForm("copyright_text"),
		SupportText:       c.PostForm("support_text"),
		FooterAddress:     c.PostForm("footer_address"),
	}
	if err := h.settings.UpdateFooter(form); err != nil {
		var validationErr services.SettingsValidationError
		if errors.As(err, &validationErr) {
			h.renderFooterSettings(c, http.StatusUnprocessableEntity, form, validationErr.FieldErrors, false)
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Footer settings not saved", "Footer settings could not be saved."))
		return
	}
	_ = h.audit.Record(services.AuditEvent{Action: "footer.updated", ResourceType: "settings", Summary: "Footer settings updated", IPAddress: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"), RequestID: requestID(c)})
	c.Redirect(http.StatusSeeOther, "/admin/settings/footer?saved=1")
}

// LegalContentSettings renders editable public legal page content.
func (h *AdminHandler) LegalContentSettings(c *gin.Context) {
	form, err := h.settings.CurrentLegalContentForm()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Legal content unavailable", "Legal content could not be loaded."))
		return
	}
	h.renderLegalContentSettings(c, http.StatusOK, form, nil, c.Query("saved") == "1")
}

// UpdateLegalContentSettings stores public legal page content.
func (h *AdminHandler) UpdateLegalContentSettings(c *gin.Context) {
	form := services.LegalContentForm{
		PrivacyPolicyMarkdown: c.PostForm("privacy_policy_markdown"),
		TermsMarkdown:         c.PostForm("terms_markdown"),
	}
	if err := h.settings.UpdateLegalContent(form); err != nil {
		var validationErr services.SettingsValidationError
		if errors.As(err, &validationErr) {
			h.renderLegalContentSettings(c, http.StatusUnprocessableEntity, form, validationErr.FieldErrors, false)
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Legal content not saved", "Legal content could not be saved."))
		return
	}
	_ = h.audit.Record(services.AuditEvent{Action: "legal_content.updated", ResourceType: "settings", Summary: "Legal content updated", IPAddress: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"), RequestID: requestID(c)})
	c.Redirect(http.StatusSeeOther, "/admin/settings/legal?saved=1")
}

// SEOSettings renders editable SEO settings.
func (h *AdminHandler) SEOSettings(c *gin.Context) {
	form, err := h.settings.CurrentSEOForm()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("SEO settings unavailable", "SEO settings could not be loaded."))
		return
	}
	h.renderSEOSettings(c, http.StatusOK, form, nil, c.Query("saved") == "1")
}

// UpdateSEOSettings stores SEO settings.
func (h *AdminHandler) UpdateSEOSettings(c *gin.Context) {
	form := services.SEOForm{
		SEOEnabled:                  c.PostForm("seo_enabled") == "1",
		SearchEngineIndexingEnabled: c.PostForm("search_engine_indexing_enabled") == "1",
		RobotsEnabled:               c.PostForm("robots_enabled") == "1",
		SitemapEnabled:              c.PostForm("sitemap_enabled") == "1",
		DefaultMetaTitle:            c.PostForm("default_meta_title"),
		DefaultMetaDescription:      c.PostForm("default_meta_description"),
		DefaultKeywords:             c.PostForm("default_keywords"),
		CanonicalBaseURL:            c.PostForm("canonical_base_url"),
		DefaultRobotsDirective:      c.PostForm("default_robots_directive"),
		GoogleSiteVerification:      c.PostForm("google_site_verification"),
		BingSiteVerification:        c.PostForm("bing_site_verification"),
		StructuredDataEnabled:       c.PostForm("structured_data_enabled") == "1",
		OrganizationName:            c.PostForm("organization_name"),
		OrganizationURL:             c.PostForm("organization_url"),
		OrganizationLogoURL:         c.PostForm("organization_logo_url"),
		OpenGraphSiteName:           c.PostForm("open_graph_site_name"),
		OpenGraphLocale:             c.PostForm("open_graph_locale"),
		OpenGraphDefaultImageURL:    c.PostForm("open_graph_default_image_url"),
		TwitterCardType:             c.PostForm("twitter_card_type"),
		TwitterHandle:               c.PostForm("twitter_handle"),
		TwitterDefaultImageURL:      c.PostForm("twitter_default_image_url"),
	}
	fieldErrors := map[string]string{}
	uploads := []struct {
		field string
		kind  string
		set   func(string)
	}{
		{field: "open_graph_default_image_file", kind: "open_graph_image", set: func(value string) { form.OpenGraphDefaultImageURL = value }},
		{field: "twitter_default_image_file", kind: "twitter_image", set: func(value string) { form.TwitterDefaultImageURL = value }},
		{field: "organization_logo_file", kind: "organization_logo", set: func(value string) { form.OrganizationLogoURL = value }},
	}
	if strings.HasPrefix(c.ContentType(), "multipart/") {
		for _, upload := range uploads {
			header, err := c.FormFile(upload.field)
			if err != nil {
				if errors.Is(err, http.ErrMissingFile) {
					continue
				}
				fieldErrors[upload.field] = "Upload could not be read."
				continue
			}
			url, err := h.media.Save(upload.kind, header)
			if err != nil {
				fieldErrors[upload.field] = mediaErrorMessage(err)
				continue
			}
			if url != "" {
				upload.set(url)
			}
		}
	}
	if len(fieldErrors) > 0 {
		h.renderSEOSettings(c, http.StatusUnprocessableEntity, form, fieldErrors, false)
		return
	}
	if err := h.settings.UpdateSEO(form); err != nil {
		var validationErr services.SettingsValidationError
		if errors.As(err, &validationErr) {
			h.renderSEOSettings(c, http.StatusUnprocessableEntity, form, validationErr.FieldErrors, false)
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("SEO settings not saved", "SEO settings could not be saved."))
		return
	}
	_ = h.audit.Record(services.AuditEvent{Action: "seo.updated", ResourceType: "settings", Summary: "SEO settings updated", IPAddress: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"), RequestID: requestID(c)})
	c.Redirect(http.StatusSeeOther, "/admin/settings/seo?saved=1")
}

// Links lists short links.
func (h *AdminHandler) Links(c *gin.Context) {
	links, err := h.links.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Links unavailable", "Short links could not be loaded."))
		return
	}
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Links unavailable", "A secure form token could not be created."))
		return
	}
	c.HTML(http.StatusOK, "public/links.html", h.withAdmin(c, gin.H{
		"Title":     "Short links",
		"Links":     links,
		"CSRFToken": csrfToken,
		"Saved":     c.Query("saved") == "1",
		"Deleted":   c.Query("deleted") == "1",
	}))
}

// NewLink renders the create-link form.
func (h *AdminHandler) NewLink(c *gin.Context) {
	h.renderLinkForm(c, http.StatusOK, "New short link", "/admin/links", services.LinkForm{IsActive: true}, nil)
}

// CreateLink validates and stores a new short link.
func (h *AdminHandler) CreateLink(c *gin.Context) {
	form := linkFormFromRequest(c)
	if err := h.links.Create(form); err != nil {
		var validationErr services.LinkValidationError
		if errors.As(err, &validationErr) {
			h.renderLinkForm(c, http.StatusUnprocessableEntity, "New short link", "/admin/links", form, validationErr.FieldErrors)
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Link not saved", "The short link could not be saved."))
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/links?saved=1")
}

// EditLink renders the edit-link form.
func (h *AdminHandler) EditLink(c *gin.Context) {
	id, err := services.ParseID(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusNotFound, "public/not_found.html", notFoundView("Link not found", "That short link could not be found."))
		return
	}
	form, err := h.links.FormForID(id)
	if err != nil {
		if errors.Is(err, services.ErrLinkNotFound) {
			c.HTML(http.StatusNotFound, "public/not_found.html", notFoundView("Link not found", "That short link could not be found."))
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Link unavailable", "The short link could not be loaded."))
		return
	}
	h.renderLinkForm(c, http.StatusOK, "Edit short link", "/admin/links/"+c.Param("id"), form, nil)
}

// UpdateLink validates and stores an existing short link.
func (h *AdminHandler) UpdateLink(c *gin.Context) {
	id, err := services.ParseID(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusNotFound, "public/not_found.html", notFoundView("Link not found", "That short link could not be found."))
		return
	}
	form := linkFormFromRequest(c)
	if err := h.links.Update(id, form); err != nil {
		var validationErr services.LinkValidationError
		if errors.As(err, &validationErr) {
			h.renderLinkForm(c, http.StatusUnprocessableEntity, "Edit short link", "/admin/links/"+c.Param("id"), form, validationErr.FieldErrors)
			return
		}
		if errors.Is(err, services.ErrLinkNotFound) {
			c.HTML(http.StatusNotFound, "public/not_found.html", notFoundView("Link not found", "That short link could not be found."))
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Link not saved", "The short link could not be saved."))
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/links?saved=1")
}

// DeleteLink removes a short link.
func (h *AdminHandler) DeleteLink(c *gin.Context) {
	id, err := services.ParseID(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusNotFound, "public/not_found.html", notFoundView("Link not found", "That short link could not be found."))
		return
	}
	if err := h.links.Delete(id); err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Link not deleted", "The short link could not be deleted."))
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/links?deleted=1")
}

// Account renders account security settings.
func (h *AdminHandler) Account(c *gin.Context) {
	h.renderAccount(c, http.StatusOK, nil, c.Query("saved"))
}

// About renders read-only project information.
func (h *AdminHandler) About(c *gin.Context) {
	body, err := web.FS.ReadFile("content/about_project.md")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("About unavailable", "Project information could not be loaded."))
		return
	}
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("About unavailable", "A secure form token could not be created."))
		return
	}

	c.HTML(http.StatusOK, "public/about.html", h.withAdmin(c, gin.H{
		"Title":     "About Project",
		"CSRFToken": csrfToken,
		"About":     renderAboutMarkdown(string(body)),
	}))
}

// UpdateAccountProfile validates and stores administrator profile details.
func (h *AdminHandler) UpdateAccountProfile(c *gin.Context) {
	adminID, ok := h.session.AdminID(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/admin/login")
		return
	}

	form := services.ProfileForm{
		Name:      c.PostForm("name"),
		Email:     c.PostForm("email"),
		AvatarURL: c.PostForm("avatar_url"),
	}
	header, err := c.FormFile("avatar_file")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		h.renderAccount(c, http.StatusUnprocessableEntity, map[string]string{"avatar_file": "Upload could not be read."}, "")
		return
	}
	if header != nil {
		url, err := h.media.Save("avatar", header)
		if err != nil {
			h.renderAccount(c, http.StatusUnprocessableEntity, map[string]string{"avatar_file": mediaErrorMessage(err)}, "")
			return
		}
		if url != "" {
			form.AvatarURL = url
		}
	}

	if err := h.account.UpdateProfile(adminID, form); err != nil {
		var validationErr services.ProfileValidationError
		if errors.As(err, &validationErr) {
			h.renderAccount(c, http.StatusUnprocessableEntity, validationErr.FieldErrors, "")
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Profile not saved", "Account profile could not be saved."))
		return
	}

	_ = h.audit.Record(services.AuditEvent{Action: "account.profile.updated", ResourceType: "admin_user", Summary: "Account profile updated", IPAddress: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"), RequestID: requestID(c)})
	c.Redirect(http.StatusSeeOther, "/admin/account?saved=profile")
}

// AuditLogs renders recent read-only audit events.
func (h *AdminHandler) AuditLogs(c *gin.Context) {
	logs, err := h.audit.Search(services.AuditListFilter{
		DateFrom: c.Query("from"),
		DateTo:   c.Query("to"),
		Limit:    50,
	})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Audit logs unavailable", "Audit logs could not be loaded."))
		return
	}
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Audit logs unavailable", "A secure form token could not be created."))
		return
	}
	c.HTML(http.StatusOK, "public/audit_logs.html", h.withAdmin(c, gin.H{
		"Title":     "Audit Logs",
		"Logs":      auditRows(logs),
		"CSRFToken": csrfToken,
		"Filters":   gin.H{"From": c.Query("from"), "To": c.Query("to")},
	}))
}

// AuditLogDetail renders one read-only audit event.
func (h *AdminHandler) AuditLogDetail(c *gin.Context) {
	id, err := services.ParseID(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusNotFound, "public/not_found.html", notFoundView("Audit log not found", "That audit log entry could not be found."))
		return
	}
	log, err := h.audit.Detail(id)
	if err != nil {
		if errors.Is(err, services.ErrAuditLogNotFound) {
			c.HTML(http.StatusNotFound, "public/not_found.html", notFoundView("Audit log not found", "That audit log entry could not be found."))
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Audit log unavailable", "Audit log details could not be loaded."))
		return
	}
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Audit log unavailable", "A secure form token could not be created."))
		return
	}
	c.HTML(http.StatusOK, "public/audit_log_detail.html", h.withAdmin(c, gin.H{
		"Title":         "Audit Log Detail",
		"Log":           log,
		"TimeLabel":     formatAuditTime(log.CreatedAt),
		"ActionLabel":   humanizeAuditAction(log.Action),
		"ResourceLabel": humanizeAuditResource(log.ResourceType),
		"CSRFToken":     csrfToken,
	}))
}

// Analytics renders the administrator analytics dashboard.
func (h *AdminHandler) Analytics(c *gin.Context) {
	overview, err := h.analytics.Overview(c.Query("from"), c.Query("to"))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Analytics unavailable", "Analytics data could not be loaded."))
		return
	}
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Analytics unavailable", "A secure form token could not be created."))
		return
	}
	c.HTML(http.StatusOK, "public/analytics.html", h.withAdmin(c, gin.H{
		"Title":     "Analytics",
		"Analytics": overview,
		"CSRFToken": csrfToken,
		"Cleanup":   c.Query("cleanup") == "1",
	}))
}

// ExportAnalytics streams a privacy-aware CSV export.
func (h *AdminHandler) ExportAnalytics(c *gin.Context) {
	if c.Query("type") != "" && c.Query("type") != "redirects" {
		c.HTML(http.StatusBadRequest, "public/error.html", errorView("Export unavailable", "Only redirect analytics export is available."))
		return
	}
	filename := fmt.Sprintf("ty2-redirect-analytics-%s.csv", time.Now().Local().Format("20060102-150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	if err := h.analytics.ExportRedirectsCSV(c.Writer, c.Query("from"), c.Query("to")); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	_ = h.audit.Record(services.AuditEvent{
		Action:       "analytics.exported",
		ResourceType: "analytics",
		Summary:      "Analytics CSV export generated",
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		RequestID:    requestID(c),
	})
}

// AnalyticsSessionDetail renders one privacy-safe analytics session detail.
func (h *AdminHandler) AnalyticsSessionDetail(c *gin.Context) {
	detail, err := h.analytics.SessionDetail(c.Param("session_id"))
	if err != nil {
		if errors.Is(err, services.ErrAnalyticsSessionNotFound) {
			c.HTML(http.StatusNotFound, "public/not_found.html", notFoundView("Session not found", "That analytics session could not be found."))
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Analytics session unavailable", "Analytics session details could not be loaded."))
		return
	}
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Analytics session unavailable", "A secure form token could not be created."))
		return
	}
	c.HTML(http.StatusOK, "public/analytics_session.html", h.withAdmin(c, gin.H{
		"Title":     "Analytics Session",
		"Session":   detail,
		"CSRFToken": csrfToken,
	}))
}

// CleanupAnalytics deletes expired analytics rows according to retention settings.
func (h *AdminHandler) CleanupAnalytics(c *gin.Context) {
	summary, err := h.analytics.CleanupExpired(false)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Cleanup failed", "Expired analytics data could not be cleaned up."))
		return
	}
	_ = h.audit.Record(services.AuditEvent{
		Action:       "analytics.cleanup.run",
		ResourceType: "analytics",
		Summary:      "Analytics retention cleanup run",
		Metadata: map[string]any{
			"page_views":      summary.PageViews,
			"redirect_events": summary.RedirectEvents,
			"sessions":        summary.Sessions,
		},
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		RequestID: requestID(c),
	})
	c.Redirect(http.StatusSeeOther, "/admin/analytics?cleanup=1")
}

// AnalyticsSettings renders editable analytics settings.
func (h *AdminHandler) AnalyticsSettings(c *gin.Context) {
	form, err := h.analytics.SettingsForm()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Analytics settings unavailable", "Analytics settings could not be loaded."))
		return
	}
	h.renderAnalyticsSettings(c, http.StatusOK, form, nil, c.Query("saved") == "1")
}

// UpdateAnalyticsSettings validates and stores analytics settings.
func (h *AdminHandler) UpdateAnalyticsSettings(c *gin.Context) {
	form := services.AnalyticsSettingsForm{
		AnalyticsEnabled:               c.PostForm("analytics_enabled") == "1",
		PageViewTrackingEnabled:        c.PostForm("page_view_tracking_enabled") == "1",
		RedirectTrackingEnabled:        c.PostForm("redirect_tracking_enabled") == "1",
		BotTrackingEnabled:             c.PostForm("bot_tracking_enabled") == "1",
		ExcludeBotsFromDashboard:       c.PostForm("exclude_bots_from_dashboard") == "1",
		UniqueVisitorEstimationEnabled: c.PostForm("unique_visitor_estimation_enabled") == "1",
		IPHandlingMode:                 c.PostForm("ip_handling_mode"),
		RawUserAgentStorageEnabled:     c.PostForm("raw_user_agent_storage_enabled") == "1",
		ReferrerTrackingEnabled:        c.PostForm("referrer_tracking_enabled") == "1",
		UTMTrackingEnabled:             false,
		ClientSideDeviceDetailsEnabled: c.PostForm("client_side_device_details_enabled") == "1",
		GeolocationEnrichmentEnabled:   c.PostForm("geolocation_enrichment_enabled") == "1",
		CookieConsentRequired:          c.PostForm("cookie_consent_required") == "1",
		SessionCookieLifetimeDays:      atoiDefault(c.PostForm("session_cookie_lifetime_days"), 30),
		DataRetentionDays:              atoiDefault(c.PostForm("data_retention_days"), 90),
		AutomaticCleanupEnabled:        c.PostForm("automatic_cleanup_enabled") == "1",
		AnalyticsExportEnabled:         c.PostForm("analytics_export_enabled") == "1",
		RespectDoNotTrack:              c.PostForm("respect_do_not_track") == "1",
		RespectGlobalPrivacyControl:    c.PostForm("respect_global_privacy_control") == "1",
		AdminIPExclusionList:           c.PostForm("admin_ip_exclusion_list"),
		InternalTrafficExclusionCIDRs:  c.PostForm("internal_traffic_exclusion_cidrs"),
		QueryParameterAllowlist:        c.PostForm("query_parameter_allowlist"),
		QueryParameterDenylist:         c.PostForm("query_parameter_denylist"),
	}
	if err := h.analytics.UpdateSettings(form); err != nil {
		var validationErr services.SettingsValidationError
		if errors.As(err, &validationErr) {
			h.renderAnalyticsSettings(c, http.StatusUnprocessableEntity, form, validationErr.FieldErrors, false)
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Analytics settings not saved", "Analytics settings could not be saved."))
		return
	}
	_ = h.audit.Record(services.AuditEvent{
		Action:       "analytics.settings.updated",
		ResourceType: "settings",
		Summary:      "Analytics settings updated",
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		RequestID:    requestID(c),
	})
	c.Redirect(http.StatusSeeOther, "/admin/settings/analytics?saved=1")
}

// ChangePassword validates and stores a new administrator password.
func (h *AdminHandler) ChangePassword(c *gin.Context) {
	adminID, ok := h.session.AdminID(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/admin/login")
		return
	}
	form := services.PasswordForm{
		CurrentPassword: c.PostForm("current_password"),
		NewPassword:     c.PostForm("new_password"),
		ConfirmPassword: c.PostForm("confirm_password"),
	}
	if err := h.account.ChangePassword(adminID, form); err != nil {
		var validationErr services.PasswordValidationError
		if errors.As(err, &validationErr) {
			h.renderAccount(c, http.StatusUnprocessableEntity, validationErr.FieldErrors, "")
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Password not changed", "The password could not be changed."))
		return
	}
	if err := h.session.SetAdmin(c, adminID); err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Password changed", "Your password changed, but the session could not be refreshed. Please sign in again."))
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/account?saved=password")
}

func (h *AdminHandler) renderSettings(c *gin.Context, status int, form services.SettingsForm, fieldErrors map[string]string, saved bool) {
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Settings unavailable", "A secure form token could not be created."))
		return
	}
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}
	c.HTML(status, "public/settings.html", h.withAdmin(c, gin.H{"Title": "Settings", "Form": form, "FieldErrors": fieldErrors, "CSRFToken": csrfToken, "Saved": saved}))
}

func (h *AdminHandler) renderBrandingSettings(c *gin.Context, status int, form services.BrandingForm, fieldErrors map[string]string, saved bool) {
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Branding settings unavailable", "A secure form token could not be created."))
		return
	}
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}
	c.HTML(status, "public/branding_settings.html", h.withAdmin(c, gin.H{"Title": "Branding Settings", "Form": form, "FieldErrors": fieldErrors, "CSRFToken": csrfToken, "Saved": saved}))
}

func (h *AdminHandler) renderFooterSettings(c *gin.Context, status int, form services.FooterForm, fieldErrors map[string]string, saved bool) {
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Footer settings unavailable", "A secure form token could not be created."))
		return
	}
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}
	c.HTML(status, "public/footer_settings.html", h.withAdmin(c, gin.H{"Title": "Footer Settings", "Form": form, "FieldErrors": fieldErrors, "CSRFToken": csrfToken, "Saved": saved}))
}

func (h *AdminHandler) renderLegalContentSettings(c *gin.Context, status int, form services.LegalContentForm, fieldErrors map[string]string, saved bool) {
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Legal content unavailable", "A secure form token could not be created."))
		return
	}
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}
	c.HTML(status, "public/legal_settings.html", h.withAdmin(c, gin.H{"Title": "Legal Content", "Form": form, "FieldErrors": fieldErrors, "CSRFToken": csrfToken, "Saved": saved}))
}

func (h *AdminHandler) renderSEOSettings(c *gin.Context, status int, form services.SEOForm, fieldErrors map[string]string, saved bool) {
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("SEO settings unavailable", "A secure form token could not be created."))
		return
	}
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}
	c.HTML(status, "public/seo_settings.html", h.withAdmin(c, gin.H{"Title": "SEO Settings", "Form": form, "FieldErrors": fieldErrors, "CSRFToken": csrfToken, "Saved": saved}))
}

func (h *AdminHandler) renderAnalyticsSettings(c *gin.Context, status int, form services.AnalyticsSettingsForm, fieldErrors map[string]string, saved bool) {
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Analytics settings unavailable", "A secure form token could not be created."))
		return
	}
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}
	c.HTML(status, "public/analytics_settings.html", h.withAdmin(c, gin.H{"Title": "Analytics Settings", "Form": form, "FieldErrors": fieldErrors, "CSRFToken": csrfToken, "Saved": saved}))
}

func (h *AdminHandler) renderLinkForm(c *gin.Context, status int, title, action string, form services.LinkForm, fieldErrors map[string]string) {
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Link unavailable", "A secure form token could not be created."))
		return
	}
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}
	c.HTML(status, "public/link_form.html", h.withAdmin(c, gin.H{"Title": title, "Heading": title, "Action": action, "Form": form, "FieldErrors": fieldErrors, "CSRFToken": csrfToken}))
}

func (h *AdminHandler) renderAccount(c *gin.Context, status int, fieldErrors map[string]string, saved string) {
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Account unavailable", "A secure form token could not be created."))
		return
	}
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}
	profile, err := h.currentAdminProfile(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Account unavailable", "Account details could not be loaded."))
		return
	}
	c.HTML(status, "public/account.html", h.withAdmin(c, gin.H{"Title": "Account", "CSRFToken": csrfToken, "FieldErrors": fieldErrors, "Saved": saved, "Account": profile, "CurrentAdmin": profile}))
}

func (h *AdminHandler) withAdmin(c *gin.Context, data gin.H) gin.H {
	if data == nil {
		data = gin.H{}
	}
	data["AdminChrome"] = true
	if _, exists := data["Theme"]; !exists {
		if theme, err := h.settings.CurrentAdminTheme(); err == nil {
			data["Theme"] = theme
		} else {
			data["Theme"] = services.AdminTheme(nil)
		}
	}
	if _, exists := data["CurrentAdmin"]; exists {
		return data
	}

	profile, err := h.currentAdminProfile(c)
	if err != nil {
		data["CurrentAdmin"] = services.AccountProfile{
			Name:     "Administrator",
			Initials: "A",
			Role:     "Administrator",
		}
		return data
	}

	data["CurrentAdmin"] = profile
	return data
}

func (h *AdminHandler) currentAdminProfile(c *gin.Context) (services.AccountProfile, error) {
	adminID, ok := h.session.AdminID(c)
	if !ok {
		return services.AccountProfile{}, services.ErrInvalidCredentials
	}
	return h.account.Profile(adminID)
}

func linkFormFromRequest(c *gin.Context) services.LinkForm {
	return services.LinkForm{
		Title:       c.PostForm("title"),
		Slug:        c.PostForm("slug"),
		Destination: c.PostForm("destination"),
		IsActive:    c.PostForm("is_active") == "1",
	}
}

func requestID(c *gin.Context) string {
	value, ok := c.Get("request_id")
	if !ok {
		return ""
	}
	id, _ := value.(string)
	return id
}

func mediaErrorMessage(err error) string {
	switch {
	case errors.Is(err, services.ErrMediaTooLarge):
		return err.Error()
	case errors.Is(err, services.ErrMediaInvalidType):
		return "Upload a PNG, JPEG, WebP, or ICO file."
	default:
		return "Upload could not be saved."
	}
}

func auditRows(logs []models.AuditLog) []auditLogRow {
	rows := make([]auditLogRow, 0, len(logs))
	for _, log := range logs {
		rows = append(rows, auditLogRow{
			ID:            log.ID,
			TimeLabel:     formatAuditTime(log.CreatedAt),
			TimeTitle:     log.CreatedAt.Format(time.RFC3339),
			ActionLabel:   humanizeAuditAction(log.Action),
			ActionRaw:     log.Action,
			ResourceLabel: humanizeAuditResource(log.ResourceType),
			Summary:       log.Summary,
			RequestID:     log.RequestID,
			RequestShort:  shortenAuditRequestID(log.RequestID),
		})
	}
	return rows
}

func formatAuditTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Local().Format("Jan 2, 2006 3:04 PM")
}

func humanizeAuditAction(value string) string {
	switch value {
	case "settings.updated":
		return "General settings updated"
	case "branding.updated":
		return "Branding updated"
	case "seo.updated":
		return "SEO settings updated"
	case "footer.updated":
		return "Footer updated"
	case "legal_content.updated":
		return "Legal content updated"
	case "account.profile.updated":
		return "Account profile updated"
	default:
		return titleFromKey(value)
	}
}

func humanizeAuditResource(value string) string {
	switch value {
	case "admin_user":
		return "Admin user"
	case "settings":
		return "Settings"
	default:
		return titleFromKey(value)
	}
}

func titleFromKey(value string) string {
	parts := strings.Fields(strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(value))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func shortenAuditRequestID(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

func atoiDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}
