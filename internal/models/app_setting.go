package models

import "time"

// AppSetting stores site metadata and app redirect targets.
type AppSetting struct {
	ID                             uint `gorm:"primaryKey"`
	SiteName                       string
	SiteDescription                string
	AndroidURL                     string
	AppleURL                       string
	DefaultURL                     string
	PublicBaseURL                  string
	SupportEmail                   string
	SiteTagline                    string
	FrontendThemeColor             string
	AdminThemeColor                string
	LogoURL                        string
	DarkLogoURL                    string
	FaviconURL                     string
	AppleTouchIconURL              string
	DefaultSocialImageURL          string
	SEOEnabled                     bool `gorm:"not null;default:true"`
	SearchEngineIndexingEnabled    bool `gorm:"not null;default:true"`
	RobotsEnabled                  bool `gorm:"not null;default:true"`
	SitemapEnabled                 bool `gorm:"not null;default:true"`
	DefaultMetaTitle               string
	DefaultMetaDescription         string
	DefaultKeywords                string
	CanonicalBaseURL               string
	DefaultRobotsDirective         string
	GoogleSiteVerification         string
	BingSiteVerification           string
	StructuredDataEnabled          bool `gorm:"not null;default:true"`
	OrganizationName               string
	OrganizationURL                string
	OrganizationLogoURL            string
	OpenGraphSiteName              string
	OpenGraphLocale                string
	OpenGraphDefaultImageURL       string
	TwitterCardType                string
	TwitterHandle                  string
	TwitterDefaultImageURL         string
	FooterEnabled                  bool `gorm:"not null;default:true"`
	FooterBrandText                string
	FooterDescription              string
	CopyrightText                  string
	SupportText                    string
	FooterAddress                  string
	PrivacyPolicyURL               string
	TermsURL                       string
	PrivacyPolicyMarkdown          string `gorm:"type:text"`
	TermsMarkdown                  string `gorm:"type:text"`
	PoweredByText                  string
	AnalyticsEnabled               bool `gorm:"not null;default:true"`
	PageViewTrackingEnabled        bool `gorm:"not null;default:true"`
	RedirectTrackingEnabled        bool `gorm:"not null;default:true"`
	BotTrackingEnabled             bool `gorm:"not null;default:true"`
	ExcludeBotsFromDashboard       bool `gorm:"not null;default:true"`
	UniqueVisitorEstimationEnabled bool `gorm:"not null;default:true"`
	IPHandlingMode                 string
	RawUserAgentStorageEnabled     bool `gorm:"not null;default:false"`
	ReferrerTrackingEnabled        bool `gorm:"not null;default:true"`
	UTMTrackingEnabled             bool `gorm:"not null;default:true"`
	ClientSideDeviceDetailsEnabled bool `gorm:"not null;default:false"`
	GeolocationEnrichmentEnabled   bool `gorm:"not null;default:false"`
	CookieConsentRequired          bool `gorm:"not null;default:false"`
	SessionCookieLifetimeDays      int  `gorm:"not null;default:30"`
	DataRetentionDays              int  `gorm:"not null;default:90"`
	AutomaticCleanupEnabled        bool `gorm:"not null;default:true"`
	AnalyticsExportEnabled         bool `gorm:"not null;default:true"`
	RespectDoNotTrack              bool `gorm:"not null;default:true"`
	RespectGlobalPrivacyControl    bool `gorm:"not null;default:true"`
	AdminIPExclusionList           string
	InternalTrafficExclusionCIDRs  string
	QueryParameterAllowlist        string
	QueryParameterDenylist         string
	IsSetupCompleted               bool `gorm:"not null;default:false"`
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}
