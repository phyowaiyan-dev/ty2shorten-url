package services

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/validation"
)

// SEOView contains safe metadata for public templates.
type SEOView struct {
	Title                  string
	Description            string
	Keywords               string
	Robots                 string
	CanonicalURL           string
	OGType                 string
	OGTitle                string
	OGDescription          string
	OGURL                  string
	OGSiteName             string
	OGImage                string
	OGLocale               string
	TwitterCard            string
	TwitterTitle           string
	TwitterDescription     string
	TwitterImage           string
	TwitterSite            string
	GoogleSiteVerification string
	BingSiteVerification   string
	StructuredData         template.JS
	FaviconURL             string
	AppleTouchIconURL      string
}

// SEOService builds SEO metadata and crawler resources.
type SEOService struct {
	settings *repositories.SettingsRepository
}

// NewSEOService constructs an SEOService.
func NewSEOService(settings *repositories.SettingsRepository) *SEOService {
	return &SEOService{settings: settings}
}

// Home builds metadata for the public landing page.
func (s *SEOService) Home(settings *models.AppSetting) SEOView {
	base := normalizedBase(settings)
	title := firstNonEmpty(settings.DefaultMetaTitle, settings.SiteName, "Ty2Shorten URL")
	description := firstNonEmpty(settings.DefaultMetaDescription, settings.SiteDescription)
	image := absoluteURL(base, firstNonEmpty(settings.OpenGraphDefaultImageURL, settings.TwitterDefaultImageURL, settings.DefaultSocialImageURL))
	robots := firstNonEmpty(settings.DefaultRobotsDirective, "index, follow")
	if !settings.SEOEnabled || !settings.SearchEngineIndexingEnabled {
		robots = "noindex, nofollow"
	}

	view := SEOView{
		Title:                  title,
		Description:            description,
		Keywords:               settings.DefaultKeywords,
		Robots:                 robots,
		CanonicalURL:           validation.JoinURLPath(base, "/"),
		OGType:                 "website",
		OGTitle:                title,
		OGDescription:          description,
		OGURL:                  validation.JoinURLPath(base, "/"),
		OGSiteName:             firstNonEmpty(settings.OpenGraphSiteName, settings.SiteName),
		OGImage:                image,
		OGLocale:               firstNonEmpty(settings.OpenGraphLocale, "en_US"),
		TwitterCard:            firstNonEmpty(settings.TwitterCardType, "summary_large_image"),
		TwitterTitle:           title,
		TwitterDescription:     description,
		TwitterImage:           image,
		TwitterSite:            settings.TwitterHandle,
		GoogleSiteVerification: settings.GoogleSiteVerification,
		BingSiteVerification:   settings.BingSiteVerification,
		FaviconURL:             absoluteURL(base, settings.FaviconURL),
		AppleTouchIconURL:      absoluteURL(base, settings.AppleTouchIconURL),
	}
	if settings.StructuredDataEnabled {
		view.StructuredData = structuredData(settings, base)
	}
	return view
}

// NoIndex returns metadata for internal or error pages.
func NoIndex(title string) SEOView {
	return SEOView{Title: title, Robots: "noindex, nofollow"}
}

// Robots returns dynamic robots.txt content and status.
func (s *SEOService) Robots() (string, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return "", err
	}
	if settings == nil {
		return "User-agent: *\nDisallow: /\n", nil
	}
	if !settings.SearchEngineIndexingEnabled {
		return "User-agent: *\nDisallow: /\n", nil
	}
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	b.WriteString("Allow: /\n")
	b.WriteString("Disallow: /admin/\n")
	b.WriteString("Disallow: /setup/\n")
	if settings.SitemapEnabled {
		b.WriteString("Sitemap: ")
		b.WriteString(validation.JoinURLPath(normalizedBase(settings), "/sitemap.xml"))
		b.WriteString("\n")
	}
	return b.String(), nil
}

// Sitemap returns dynamic sitemap XML.
func (s *SEOService) Sitemap() ([]byte, int, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if settings == nil || !settings.SitemapEnabled {
		return []byte("sitemap disabled"), http.StatusNotFound, nil
	}
	type urlEntry struct {
		Loc string `xml:"loc"`
	}
	type urlSet struct {
		XMLName string     `xml:"urlset"`
		Xmlns   string     `xml:"xmlns,attr"`
		URLs    []urlEntry `xml:"url"`
	}
	body, err := xml.MarshalIndent(urlSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  []urlEntry{{Loc: validation.JoinURLPath(normalizedBase(settings), "/")}},
	}, "", "  ")
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("marshal sitemap XML: %w", err)
	}
	return append([]byte(xml.Header), body...), http.StatusOK, nil
}

func normalizedBase(settings *models.AppSetting) string {
	return strings.TrimRight(firstNonEmpty(settings.CanonicalBaseURL, settings.PublicBaseURL, "http://localhost:8722"), "/")
}

func absoluteURL(base, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return validation.JoinURLPath(base, value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func structuredData(settings *models.AppSetting, base string) template.JS {
	data := map[string]any{
		"@context":            "https://schema.org",
		"@type":               "SoftwareApplication",
		"name":                firstNonEmpty(settings.OrganizationName, settings.SiteName),
		"url":                 validation.JoinURLPath(base, "/"),
		"description":         settings.SiteDescription,
		"applicationCategory": "UtilitiesApplication",
		"operatingSystem":     "Android, iOS, Web",
	}
	if logo := absoluteURL(base, firstNonEmpty(settings.OrganizationLogoURL, settings.LogoURL)); logo != "" {
		data["logo"] = logo
	}
	if settings.AndroidURL != "" {
		data["downloadUrl"] = []string{settings.AndroidURL, settings.AppleURL}
	}
	body, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, body); err != nil {
		return template.JS(body)
	}
	return template.JS(compact.String())
}
