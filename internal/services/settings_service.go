package services

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/validation"
)

// SettingsForm contains editable application settings.
type SettingsForm struct {
	SiteName        string
	SiteDescription string
	AndroidURL      string
	AppleURL        string
	DefaultURL      string
	PublicBaseURL   string
	SupportEmail    string
}

// SEOForm contains editable SEO settings.
type SEOForm struct {
	SEOEnabled                  bool
	SearchEngineIndexingEnabled bool
	RobotsEnabled               bool
	SitemapEnabled              bool
	DefaultMetaTitle            string
	DefaultMetaDescription      string
	DefaultKeywords             string
	CanonicalBaseURL            string
	DefaultRobotsDirective      string
	GoogleSiteVerification      string
	BingSiteVerification        string
	StructuredDataEnabled       bool
	OrganizationName            string
	OrganizationURL             string
	OrganizationLogoURL         string
	OpenGraphSiteName           string
	OpenGraphLocale             string
	OpenGraphDefaultImageURL    string
	TwitterCardType             string
	TwitterHandle               string
	TwitterDefaultImageURL      string
}

// BrandingForm contains editable branding/media settings.
type BrandingForm struct {
	SiteTagline           string
	LogoURL               string
	DarkLogoURL           string
	FaviconURL            string
	AppleTouchIconURL     string
	DefaultSocialImageURL string
}

// FooterForm contains editable public footer settings.
type FooterForm struct {
	FooterEnabled     bool
	FooterBrandText   string
	FooterDescription string
	CopyrightText     string
	SupportText       string
	FooterAddress     string
}

// LegalContentForm contains editable public legal page content.
type LegalContentForm struct {
	PrivacyPolicyMarkdown string
	TermsMarkdown         string
}

// SettingsValidationError contains friendly field errors.
type SettingsValidationError struct {
	FieldErrors map[string]string
}

func (e SettingsValidationError) Error() string {
	return "invalid settings form"
}

// SettingsService owns administrator settings updates.
type SettingsService struct {
	settings     *repositories.SettingsRepository
	isProduction bool
	logger       *slog.Logger
}

// NewSettingsService constructs a SettingsService.
func NewSettingsService(settings *repositories.SettingsRepository, isProduction bool, logger *slog.Logger) *SettingsService {
	return &SettingsService{settings: settings, isProduction: isProduction, logger: logger}
}

// CurrentForm returns settings formatted for administrator editing.
func (s *SettingsService) CurrentForm() (SettingsForm, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return SettingsForm{}, err
	}
	if settings == nil {
		return SettingsForm{}, fmt.Errorf("app settings are not available")
	}
	return SettingsForm{
		SiteName:        settings.SiteName,
		SiteDescription: settings.SiteDescription,
		AndroidURL:      settings.AndroidURL,
		AppleURL:        settings.AppleURL,
		DefaultURL:      settings.DefaultURL,
		PublicBaseURL:   settings.PublicBaseURL,
		SupportEmail:    settings.SupportEmail,
	}, nil
}

// Update validates and stores editable settings.
func (s *SettingsService) Update(form SettingsForm) error {
	normalized, err := s.validate(form)
	if err != nil {
		return err
	}
	current, err := s.settings.Current()
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("app settings are not available")
	}

	current.SiteName = normalized.SiteName
	current.SiteDescription = normalized.SiteDescription
	current.AndroidURL = normalized.AndroidURL
	current.AppleURL = normalized.AppleURL
	current.DefaultURL = normalized.DefaultURL
	current.PublicBaseURL = normalized.PublicBaseURL
	current.SupportEmail = normalized.SupportEmail

	if err := s.settings.Update(current); err != nil {
		return err
	}
	if s.logger != nil {
		s.logger.Info("settings updated")
	}
	return nil
}

// CurrentSEOForm returns SEO settings formatted for administrator editing.
func (s *SettingsService) CurrentSEOForm() (SEOForm, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return SEOForm{}, err
	}
	if settings == nil {
		return SEOForm{}, fmt.Errorf("app settings are not available")
	}
	return SEOForm{
		SEOEnabled:                  settings.SEOEnabled,
		SearchEngineIndexingEnabled: settings.SearchEngineIndexingEnabled,
		RobotsEnabled:               settings.RobotsEnabled,
		SitemapEnabled:              settings.SitemapEnabled,
		DefaultMetaTitle:            settings.DefaultMetaTitle,
		DefaultMetaDescription:      settings.DefaultMetaDescription,
		DefaultKeywords:             settings.DefaultKeywords,
		CanonicalBaseURL:            settings.CanonicalBaseURL,
		DefaultRobotsDirective:      settings.DefaultRobotsDirective,
		GoogleSiteVerification:      settings.GoogleSiteVerification,
		BingSiteVerification:        settings.BingSiteVerification,
		StructuredDataEnabled:       settings.StructuredDataEnabled,
		OrganizationName:            settings.OrganizationName,
		OrganizationURL:             settings.OrganizationURL,
		OrganizationLogoURL:         settings.OrganizationLogoURL,
		OpenGraphSiteName:           settings.OpenGraphSiteName,
		OpenGraphLocale:             settings.OpenGraphLocale,
		OpenGraphDefaultImageURL:    settings.OpenGraphDefaultImageURL,
		TwitterCardType:             settings.TwitterCardType,
		TwitterHandle:               settings.TwitterHandle,
		TwitterDefaultImageURL:      settings.TwitterDefaultImageURL,
	}, nil
}

// CurrentBrandingForm returns branding settings formatted for administrator editing.
func (s *SettingsService) CurrentBrandingForm() (BrandingForm, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return BrandingForm{}, err
	}
	if settings == nil {
		return BrandingForm{}, fmt.Errorf("app settings are not available")
	}
	return BrandingForm{
		SiteTagline:           settings.SiteTagline,
		LogoURL:               settings.LogoURL,
		DarkLogoURL:           settings.DarkLogoURL,
		FaviconURL:            settings.FaviconURL,
		AppleTouchIconURL:     settings.AppleTouchIconURL,
		DefaultSocialImageURL: settings.DefaultSocialImageURL,
	}, nil
}

// UpdateBranding validates and stores branding settings.
func (s *SettingsService) UpdateBranding(form BrandingForm) error {
	normalized := BrandingForm{
		SiteTagline:           strings.TrimSpace(form.SiteTagline),
		LogoURL:               strings.TrimSpace(form.LogoURL),
		DarkLogoURL:           strings.TrimSpace(form.DarkLogoURL),
		FaviconURL:            strings.TrimSpace(form.FaviconURL),
		AppleTouchIconURL:     strings.TrimSpace(form.AppleTouchIconURL),
		DefaultSocialImageURL: strings.TrimSpace(form.DefaultSocialImageURL),
	}

	fieldErrors := map[string]string{}
	taglineWords := wordCount(normalized.SiteTagline)
	if normalized.SiteTagline != "" && taglineWords < 3 {
		fieldErrors["site_tagline"] = "Use at least 3 words for the tagline."
	}
	if taglineWords > 12 {
		fieldErrors["site_tagline"] = "Keep the tagline under 12 words."
	}
	validateMediaOrHTTPURL(fieldErrors, "logo_url", normalized.LogoURL, s.isProduction)
	validateMediaOrHTTPURL(fieldErrors, "dark_logo_url", normalized.DarkLogoURL, s.isProduction)
	validateMediaOrHTTPURL(fieldErrors, "favicon_url", normalized.FaviconURL, s.isProduction)
	validateMediaOrHTTPURL(fieldErrors, "apple_touch_icon_url", normalized.AppleTouchIconURL, s.isProduction)
	validateMediaOrHTTPURL(fieldErrors, "default_social_image_url", normalized.DefaultSocialImageURL, s.isProduction)
	if len(fieldErrors) > 0 {
		return SettingsValidationError{FieldErrors: fieldErrors}
	}

	current, err := s.settings.Current()
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("app settings are not available")
	}
	current.SiteTagline = normalized.SiteTagline
	current.LogoURL = normalized.LogoURL
	current.DarkLogoURL = normalized.DarkLogoURL
	current.FaviconURL = normalized.FaviconURL
	current.AppleTouchIconURL = normalized.AppleTouchIconURL
	current.DefaultSocialImageURL = normalized.DefaultSocialImageURL
	if err := s.settings.UpdateBranding(current); err != nil {
		return err
	}
	if s.logger != nil {
		s.logger.Info("branding settings updated")
	}
	return nil
}

// CurrentFooterForm returns footer settings formatted for administrator editing.
func (s *SettingsService) CurrentFooterForm() (FooterForm, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return FooterForm{}, err
	}
	if settings == nil {
		return FooterForm{}, fmt.Errorf("app settings are not available")
	}
	return FooterForm{
		FooterEnabled:     settings.FooterEnabled,
		FooterBrandText:   settings.FooterBrandText,
		FooterDescription: settings.FooterDescription,
		CopyrightText:     settings.CopyrightText,
		SupportText:       settings.SupportText,
		FooterAddress:     settings.FooterAddress,
	}, nil
}

// UpdateFooter validates and stores public footer settings.
func (s *SettingsService) UpdateFooter(form FooterForm) error {
	normalized := FooterForm{
		FooterEnabled:     form.FooterEnabled,
		FooterBrandText:   strings.TrimSpace(form.FooterBrandText),
		FooterDescription: strings.TrimSpace(form.FooterDescription),
		CopyrightText:     strings.TrimSpace(form.CopyrightText),
		SupportText:       strings.TrimSpace(form.SupportText),
		FooterAddress:     strings.TrimSpace(form.FooterAddress),
	}
	fieldErrors := map[string]string{}
	if len(normalized.FooterBrandText) > 120 {
		fieldErrors["footer_brand_text"] = "Keep the brand text under 120 characters."
	}
	if len(normalized.FooterDescription) > 300 {
		fieldErrors["footer_description"] = "Keep the description under 300 characters."
	}
	if len(fieldErrors) > 0 {
		return SettingsValidationError{FieldErrors: fieldErrors}
	}

	current, err := s.settings.Current()
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("app settings are not available")
	}
	current.FooterEnabled = normalized.FooterEnabled
	current.FooterBrandText = normalized.FooterBrandText
	current.FooterDescription = normalized.FooterDescription
	current.CopyrightText = normalized.CopyrightText
	current.SupportText = normalized.SupportText
	current.FooterAddress = normalized.FooterAddress
	if err := s.settings.UpdateFooter(current); err != nil {
		return err
	}
	if s.logger != nil {
		s.logger.Info("footer settings updated")
	}
	return nil
}

// CurrentLegalContentForm returns legal page Markdown formatted for administrator editing.
func (s *SettingsService) CurrentLegalContentForm() (LegalContentForm, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return LegalContentForm{}, err
	}
	if settings == nil {
		return LegalContentForm{}, fmt.Errorf("app settings are not available")
	}
	return LegalContentForm{
		PrivacyPolicyMarkdown: settings.PrivacyPolicyMarkdown,
		TermsMarkdown:         settings.TermsMarkdown,
	}, nil
}

// UpdateLegalContent validates and stores public legal page Markdown.
func (s *SettingsService) UpdateLegalContent(form LegalContentForm) error {
	normalized := LegalContentForm{
		PrivacyPolicyMarkdown: strings.TrimSpace(form.PrivacyPolicyMarkdown),
		TermsMarkdown:         strings.TrimSpace(form.TermsMarkdown),
	}
	fieldErrors := map[string]string{}
	requireText(fieldErrors, "privacy_policy_markdown", normalized.PrivacyPolicyMarkdown, "Enter privacy policy content.")
	requireText(fieldErrors, "terms_markdown", normalized.TermsMarkdown, "Enter terms of use content.")
	if len(normalized.PrivacyPolicyMarkdown) > 50000 {
		fieldErrors["privacy_policy_markdown"] = "Keep the privacy policy under 50,000 characters."
	}
	if len(normalized.TermsMarkdown) > 50000 {
		fieldErrors["terms_markdown"] = "Keep the terms of use under 50,000 characters."
	}
	if len(fieldErrors) > 0 {
		return SettingsValidationError{FieldErrors: fieldErrors}
	}

	current, err := s.settings.Current()
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("app settings are not available")
	}
	current.PrivacyPolicyMarkdown = normalized.PrivacyPolicyMarkdown
	current.TermsMarkdown = normalized.TermsMarkdown
	if err := s.settings.UpdateLegalContent(current); err != nil {
		return err
	}
	if s.logger != nil {
		s.logger.Info("legal content updated")
	}
	return nil
}

// UpdateSEO validates and stores SEO settings.
func (s *SettingsService) UpdateSEO(form SEOForm) error {
	normalized := SEOForm{
		SEOEnabled:                  form.SEOEnabled,
		SearchEngineIndexingEnabled: form.SearchEngineIndexingEnabled,
		RobotsEnabled:               form.RobotsEnabled,
		SitemapEnabled:              form.SitemapEnabled,
		DefaultMetaTitle:            strings.TrimSpace(form.DefaultMetaTitle),
		DefaultMetaDescription:      strings.TrimSpace(form.DefaultMetaDescription),
		DefaultKeywords:             strings.TrimSpace(form.DefaultKeywords),
		CanonicalBaseURL:            strings.TrimSpace(form.CanonicalBaseURL),
		DefaultRobotsDirective:      strings.TrimSpace(form.DefaultRobotsDirective),
		GoogleSiteVerification:      strings.TrimSpace(form.GoogleSiteVerification),
		BingSiteVerification:        strings.TrimSpace(form.BingSiteVerification),
		StructuredDataEnabled:       form.StructuredDataEnabled,
		OrganizationName:            strings.TrimSpace(form.OrganizationName),
		OrganizationURL:             strings.TrimSpace(form.OrganizationURL),
		OrganizationLogoURL:         strings.TrimSpace(form.OrganizationLogoURL),
		OpenGraphSiteName:           strings.TrimSpace(form.OpenGraphSiteName),
		OpenGraphLocale:             strings.TrimSpace(form.OpenGraphLocale),
		OpenGraphDefaultImageURL:    strings.TrimSpace(form.OpenGraphDefaultImageURL),
		TwitterCardType:             strings.TrimSpace(form.TwitterCardType),
		TwitterHandle:               strings.TrimSpace(form.TwitterHandle),
		TwitterDefaultImageURL:      strings.TrimSpace(form.TwitterDefaultImageURL),
	}
	fieldErrors := map[string]string{}
	validateOptionalURL(fieldErrors, "canonical_base_url", normalized.CanonicalBaseURL, s.isProduction)
	validateOptionalURL(fieldErrors, "organization_url", normalized.OrganizationURL, s.isProduction)
	validateMediaOrHTTPURL(fieldErrors, "organization_logo_url", normalized.OrganizationLogoURL, s.isProduction)
	validateMediaOrHTTPURL(fieldErrors, "open_graph_default_image_url", normalized.OpenGraphDefaultImageURL, s.isProduction)
	validateMediaOrHTTPURL(fieldErrors, "twitter_default_image_url", normalized.TwitterDefaultImageURL, s.isProduction)
	if len(fieldErrors) > 0 {
		return SettingsValidationError{FieldErrors: fieldErrors}
	}

	current, err := s.settings.Current()
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("app settings are not available")
	}
	current.SEOEnabled = normalized.SEOEnabled
	current.SearchEngineIndexingEnabled = normalized.SearchEngineIndexingEnabled
	current.RobotsEnabled = normalized.RobotsEnabled
	current.SitemapEnabled = normalized.SitemapEnabled
	current.DefaultMetaTitle = normalized.DefaultMetaTitle
	current.DefaultMetaDescription = normalized.DefaultMetaDescription
	current.DefaultKeywords = normalized.DefaultKeywords
	current.CanonicalBaseURL = normalized.CanonicalBaseURL
	current.DefaultRobotsDirective = normalized.DefaultRobotsDirective
	current.GoogleSiteVerification = normalized.GoogleSiteVerification
	current.BingSiteVerification = normalized.BingSiteVerification
	current.StructuredDataEnabled = normalized.StructuredDataEnabled
	current.OrganizationName = normalized.OrganizationName
	current.OrganizationURL = normalized.OrganizationURL
	current.OrganizationLogoURL = normalized.OrganizationLogoURL
	current.OpenGraphSiteName = normalized.OpenGraphSiteName
	current.OpenGraphLocale = normalized.OpenGraphLocale
	current.OpenGraphDefaultImageURL = normalized.OpenGraphDefaultImageURL
	current.TwitterCardType = normalized.TwitterCardType
	current.TwitterHandle = normalized.TwitterHandle
	current.TwitterDefaultImageURL = normalized.TwitterDefaultImageURL
	if current.DefaultRobotsDirective == "" {
		current.DefaultRobotsDirective = "index, follow"
	}
	if current.TwitterCardType == "" {
		current.TwitterCardType = "summary_large_image"
	}
	if err := s.settings.UpdateSEO(current); err != nil {
		return err
	}
	if s.logger != nil {
		s.logger.Info("SEO settings updated")
	}
	return nil
}

func (s *SettingsService) validate(form SettingsForm) (SettingsForm, error) {
	normalized := SettingsForm{
		SiteName:        strings.TrimSpace(form.SiteName),
		SiteDescription: strings.TrimSpace(form.SiteDescription),
		AndroidURL:      strings.TrimSpace(form.AndroidURL),
		AppleURL:        strings.TrimSpace(form.AppleURL),
		DefaultURL:      strings.TrimSpace(form.DefaultURL),
		PublicBaseURL:   strings.TrimSpace(form.PublicBaseURL),
		SupportEmail:    normalizeEmail(form.SupportEmail),
	}

	fieldErrors := map[string]string{}
	requireText(fieldErrors, "site_name", normalized.SiteName, "Enter a site name.")
	requireText(fieldErrors, "site_description", normalized.SiteDescription, "Enter a site description.")
	if len([]rune(normalized.SiteName)) > 80 {
		fieldErrors["site_name"] = "Keep the site name under 80 characters."
	}
	descriptionWords := wordCount(normalized.SiteDescription)
	if normalized.SiteDescription != "" && descriptionWords < 8 {
		fieldErrors["site_description"] = "Use at least 8 words for the site description."
	}
	if descriptionWords > 50 {
		fieldErrors["site_description"] = "Keep the site description under 50 words."
	}
	validateStoreURL(fieldErrors, "android_url", normalized.AndroidURL, "https://play.google.com/store/apps/details?id=com.example.app", isGooglePlayURL, s.isProduction)
	validateStoreURL(fieldErrors, "apple_url", normalized.AppleURL, "https://apps.apple.com/app/example/id123456789", isAppleStoreURL, s.isProduction)
	validateOptionalURL(fieldErrors, "default_url", normalized.DefaultURL, s.isProduction)
	validateRequiredURL(fieldErrors, "public_base_url", normalized.PublicBaseURL, s.isProduction)
	if normalized.SupportEmail != "" && !validEmail(normalized.SupportEmail) {
		fieldErrors["support_email"] = "Enter a valid support email."
	}

	if len(fieldErrors) > 0 {
		return normalized, SettingsValidationError{FieldErrors: fieldErrors}
	}
	return normalized, nil
}

func wordCount(value string) int {
	return len(strings.Fields(value))
}

func validateStoreURL(fieldErrors map[string]string, field, value, example string, matcher func(*url.URL) bool, requireHTTPS bool) {
	validateRequiredURL(fieldErrors, field, value, requireHTTPS)
	if _, exists := fieldErrors[field]; exists || strings.TrimSpace(value) == "" {
		return
	}
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || !matcher(parsed) {
		fieldErrors[field] = "Use a URL like " + example + "."
	}
}

func isGooglePlayURL(parsed *url.URL) bool {
	return parsed.Scheme == "https" && strings.EqualFold(parsed.Hostname(), "play.google.com") && strings.HasPrefix(parsed.EscapedPath(), "/store/apps/")
}

func isAppleStoreURL(parsed *url.URL) bool {
	return parsed.Scheme == "https" && strings.EqualFold(parsed.Hostname(), "apps.apple.com") && strings.Contains(parsed.EscapedPath(), "/app/")
}

func urlValidationMessage(err error) string {
	switch {
	case errors.Is(err, validation.ErrInvalidURL):
		return "Enter a valid URL."
	case errors.Is(err, validation.ErrUnsafeURLScheme):
		return "Only http and https URLs are allowed."
	case errors.Is(err, validation.ErrHTTPSRequired):
		return "Use an HTTPS URL in production."
	default:
		return ""
	}
}

func validateMediaOrHTTPURL(fieldErrors map[string]string, field, value string, requireHTTPS bool) {
	if value == "" {
		return
	}
	if strings.HasPrefix(value, "/media/") && !strings.Contains(value, "..") {
		return
	}
	validateOptionalURL(fieldErrors, field, value, requireHTTPS)
}
