package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
)

const (
	AnalyticsSessionCookie        = "ty2_analytics_session"
	defaultAnalyticsRetentionDays = 90
	defaultAnalyticsCookieDays    = 30
	maxAnalyticsText              = 255
)

// ErrAnalyticsSessionNotFound is returned when an analytics session cannot be found.
var ErrAnalyticsSessionNotFound = errors.New("analytics session not found")

// AnalyticsSettingsForm contains editable privacy-aware analytics settings.
type AnalyticsSettingsForm struct {
	AnalyticsEnabled               bool
	PageViewTrackingEnabled        bool
	RedirectTrackingEnabled        bool
	BotTrackingEnabled             bool
	ExcludeBotsFromDashboard       bool
	UniqueVisitorEstimationEnabled bool
	IPHandlingMode                 string
	RawUserAgentStorageEnabled     bool
	ReferrerTrackingEnabled        bool
	UTMTrackingEnabled             bool
	ClientSideDeviceDetailsEnabled bool
	GeolocationEnrichmentEnabled   bool
	CookieConsentRequired          bool
	SessionCookieLifetimeDays      int
	DataRetentionDays              int
	AutomaticCleanupEnabled        bool
	AnalyticsExportEnabled         bool
	RespectDoNotTrack              bool
	RespectGlobalPrivacyControl    bool
	AdminIPExclusionList           string
	InternalTrafficExclusionCIDRs  string
	QueryParameterAllowlist        string
	QueryParameterDenylist         string
}

// AnalyticsOverview contains data rendered in the admin analytics dashboard.
type AnalyticsOverview struct {
	From              string
	To                string
	RangeLabel        string
	Summary           repositories.AnalyticsSummary
	DeviceRows        []repositories.AnalyticsBreakdownRow
	OSRows            []repositories.AnalyticsBreakdownRow
	BrowserRows       []repositories.AnalyticsBreakdownRow
	ReferrerRows      []repositories.AnalyticsBreakdownRow
	PlatformRows      []repositories.AnalyticsBreakdownRow
	RecentRedirects   []AnalyticsRedirectRow
	Settings          AnalyticsSettingsForm
	RetentionDays     int
	BotsExcluded      bool
	AnalyticsDisabled bool
}

// AnalyticsRedirectRow is a dashboard-safe redirect event row.
type AnalyticsRedirectRow struct {
	Time        string
	Route       string
	Platform    string
	SourcePath  string
	Slug        string
	IPNetwork   string
	IPHashShort string
	Device      string
	OS          string
	Browser     string
	Referrer    string
}

// AnalyticsSessionDetail contains privacy-safe session detail data for admins.
type AnalyticsSessionDetail struct {
	SessionIDShort  string
	FirstSeen       string
	LastSeen        string
	Duration        string
	PageViewCount   uint64
	RedirectCount   uint64
	DeviceType      string
	OperatingSystem string
	Browser         string
	Referrer        string
	BotStatus       string
	Timeline        []AnalyticsTimelineItem
}

// AnalyticsTimelineItem is one privacy-safe session timeline row.
type AnalyticsTimelineItem struct {
	Time  string
	Kind  string
	Label string
	Path  string
}

// AnalyticsTrackOptions describes one completed HTTP request.
type AnalyticsTrackOptions struct {
	RequestID           string
	StatusCode          int
	Duration            time.Duration
	RouteType           string
	Destination         string
	DestinationPlatform string
	ShortLinkID         *uint
	ShortLinkSlug       string
	PageTitle           string
}

// AnalyticsService owns first-party analytics tracking and reports.
type AnalyticsService struct {
	analytics    *repositories.AnalyticsRepository
	settings     *repositories.SettingsRepository
	isProduction bool
	secret       string
	logger       *slog.Logger
	clock        func() time.Time
}

// NewAnalyticsService constructs an AnalyticsService.
func NewAnalyticsService(analytics *repositories.AnalyticsRepository, settings *repositories.SettingsRepository, isProduction bool, secret string, logger *slog.Logger) *AnalyticsService {
	if strings.TrimSpace(secret) == "" {
		secret = "development-analytics-secret"
	}
	return &AnalyticsService{
		analytics:    analytics,
		settings:     settings,
		isProduction: isProduction,
		secret:       secret,
		logger:       logger,
		clock:        time.Now,
	}
}

// SettingsForm returns current analytics settings with privacy-conscious defaults.
func (s *AnalyticsService) SettingsForm() (AnalyticsSettingsForm, error) {
	settings, err := s.settings.Current()
	if err != nil {
		return AnalyticsSettingsForm{}, err
	}
	if settings == nil {
		return AnalyticsSettingsForm{}, fmt.Errorf("app settings are not available")
	}
	return analyticsSettingsFromModel(settings), nil
}

// UpdateSettings validates and stores analytics settings.
func (s *AnalyticsService) UpdateSettings(form AnalyticsSettingsForm) error {
	normalized := normalizeAnalyticsSettings(form)
	if err := ValidateAnalyticsSettings(normalized); err != nil {
		return err
	}
	current, err := s.settings.Current()
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("app settings are not available")
	}
	applyAnalyticsSettings(current, normalized)
	if err := s.settings.UpdateAnalytics(current); err != nil {
		return err
	}
	return nil
}

// Prepare creates or refreshes the first-party analytics cookie before response headers are written.
func (s *AnalyticsService) Prepare(c *gin.Context) {
	settings, err := s.settings.Current()
	if err != nil || settings == nil {
		return
	}
	form := analyticsSettingsFromModel(settings)
	if !shouldTrackRequest(c.Request, form) {
		s.clearCookieIfDisabled(c, form)
		return
	}
	_ = s.ensureSessionCookie(c, form)
}

// Track records a public page view or redirect if the request is eligible.
func (s *AnalyticsService) Track(c *gin.Context, opts AnalyticsTrackOptions) {
	settings, err := s.settings.Current()
	if err != nil || settings == nil {
		s.safeLog("analytics settings unavailable", err)
		return
	}
	form := analyticsSettingsFromModel(settings)
	if !shouldTrackRequest(c.Request, form) {
		s.clearCookieIfDisabled(c, form)
		return
	}

	device := DeviceParser{}.Parse(c.GetHeader("User-Agent"), clientHints(c.Request))
	if device.IsBot && !form.BotTrackingEnabled {
		return
	}
	sessionID := s.ensureSessionCookie(c, form)
	if sessionID == "" {
		return
	}

	referrer := ReferrerInfo{Type: "direct"}
	if form.ReferrerTrackingEnabled {
		referrer = ClassifyReferrer(c.Request)
	}
	ipInfo := NewIPProcessor(s.secret, form.IPHandlingMode, parseCIDRs(form.InternalTrafficExclusionCIDRs)).Process(c.ClientIP())
	if ipInfo.Excluded {
		return
	}
	now := s.clock()
	visitorHash := ""
	if form.UniqueVisitorEstimationEnabled {
		visitorHash = hmacString(s.secret, strings.Join([]string{sessionID, ipInfo.Network, device.DeviceType, device.OperatingSystem, now.Local().Format("2006-01")}, "|"))
	}
	session := &models.VisitorSession{
		SessionID:              sessionID,
		VisitorIDHash:          visitorHash,
		FirstSeenAt:            now,
		LastSeenAt:             now,
		StartedAt:              now,
		IsBot:                  device.IsBot,
		BotName:                device.BotName,
		IPHash:                 ipInfo.Hash,
		IPNetwork:              ipInfo.Network,
		DeviceType:             device.DeviceType,
		DeviceBrand:            device.DeviceBrand,
		DeviceModel:            device.DeviceModel,
		OperatingSystem:        device.OperatingSystem,
		OperatingSystemVersion: device.OperatingSystemVersion,
		Browser:                device.Browser,
		BrowserVersion:         device.BrowserVersion,
		Engine:                 device.Engine,
		Language:               truncate(c.GetHeader("Accept-Language"), 80),
		ReferrerType:           referrer.Type,
		ReferrerName:           referrer.Name,
		ReferrerHost:           referrer.Host,
		LandingPath:            normalizePath(c.Request.URL.Path),
	}
	if err := s.analytics.UpsertSession(session); err != nil {
		s.safeLog("analytics session write failed", err)
		return
	}

	if isRedirectStatus(opts.StatusCode) && opts.RouteType != "" && form.RedirectTrackingEnabled {
		event := &models.RedirectEvent{
			SessionID:           sessionID,
			RequestID:           opts.RequestID,
			RouteType:           opts.RouteType,
			SourcePath:          normalizePath(c.Request.URL.Path),
			ShortLinkID:         opts.ShortLinkID,
			ShortLinkSlug:       truncate(opts.ShortLinkSlug, 160),
			DestinationPlatform: truncate(opts.DestinationPlatform, 80),
			RedirectStatus:      opts.StatusCode,
			IPHash:              ipInfo.Hash,
			IPNetwork:           ipInfo.Network,
			DeviceType:          device.DeviceType,
			OperatingSystem:     device.OperatingSystem,
			Browser:             device.Browser,
			ReferrerType:        referrer.Type,
			ReferrerName:        referrer.Name,
			ReferrerHost:        referrer.Host,
			IsBot:               device.IsBot,
			OccurredAt:          now,
		}
		event.DestinationHost, event.DestinationPath = safeURLParts(opts.Destination)
		if err := s.analytics.CreateRedirect(event); err != nil {
			s.safeLog("analytics redirect write failed", err)
			return
		}
		_ = s.analytics.IncrementSessionRedirects(sessionID)
		return
	}

	if eligiblePageView(c.Request, opts.StatusCode) && form.PageViewTrackingEnabled {
		view := &models.PageView{
			SessionID:      sessionID,
			RequestID:      opts.RequestID,
			Path:           normalizePath(c.Request.URL.Path),
			NormalizedPath: normalizePath(c.FullPath()),
			Method:         c.Request.Method,
			StatusCode:     opts.StatusCode,
			ReferrerType:   referrer.Type,
			ReferrerName:   referrer.Name,
			ReferrerHost:   referrer.Host,
			ReferrerPath:   referrer.Path,
			PageTitle:      truncate(opts.PageTitle, 160),
			DurationMS:     opts.Duration.Milliseconds(),
			IsBot:          device.IsBot,
			OccurredAt:     now,
		}
		if view.NormalizedPath == "" {
			view.NormalizedPath = view.Path
		}
		if err := s.analytics.CreatePageView(view); err != nil {
			s.safeLog("analytics page-view write failed", err)
			return
		}
		_ = s.analytics.IncrementSessionPageViews(sessionID)
	}
}

// Overview returns the admin analytics dashboard data.
func (s *AnalyticsService) Overview(from, to string) (AnalyticsOverview, error) {
	form, err := s.SettingsForm()
	if err != nil {
		return AnalyticsOverview{}, err
	}
	start, end := AnalyticsRange(from, to, s.clock())
	filter := repositories.AnalyticsRangeFilter{From: start, To: end, ExcludeBots: form.ExcludeBotsFromDashboard, Limit: 8}
	summary, err := s.analytics.Summary(filter)
	if err != nil {
		return AnalyticsOverview{}, err
	}
	devices, _ := s.analytics.Breakdown("redirect_events", "device_type", filter)
	oses, _ := s.analytics.Breakdown("redirect_events", "operating_system", filter)
	browsers, _ := s.analytics.Breakdown("redirect_events", "browser", filter)
	referrers, _ := s.analytics.Breakdown("page_views", "referrer_host", filter)
	platforms, _ := s.analytics.Breakdown("redirect_events", "destination_platform", filter)
	recent, _ := s.analytics.RecentRedirectEvents(repositories.AnalyticsRangeFilter{From: start, To: end, ExcludeBots: false, Limit: 25})
	rangeLabel := analyticsRangeLabel(start, end, from, to)
	return AnalyticsOverview{
		From:              start.Format("2006-01-02"),
		To:                end.AddDate(0, 0, -1).Format("2006-01-02"),
		RangeLabel:        rangeLabel,
		Summary:           summary,
		DeviceRows:        devices,
		OSRows:            oses,
		BrowserRows:       browsers,
		ReferrerRows:      referrers,
		PlatformRows:      platforms,
		RecentRedirects:   analyticsRedirectRows(recent),
		Settings:          form,
		RetentionDays:     form.DataRetentionDays,
		BotsExcluded:      form.ExcludeBotsFromDashboard,
		AnalyticsDisabled: !form.AnalyticsEnabled,
	}, nil
}

// ExportRedirectsCSV writes a safe redirect-events CSV export.
func (s *AnalyticsService) ExportRedirectsCSV(w io.Writer, from, to string) error {
	form, err := s.SettingsForm()
	if err != nil {
		return err
	}
	if !form.AnalyticsExportEnabled {
		return errors.New("analytics export is disabled")
	}
	start, end := AnalyticsRange(from, to, s.clock())
	events, err := s.analytics.RedirectEvents(repositories.AnalyticsRangeFilter{From: start, To: end, ExcludeBots: form.ExcludeBotsFromDashboard, Limit: 5000})
	if err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"occurred_at", "route_type", "source_path", "slug", "destination_host", "destination_path", "platform", "ip_network", "ip_hash", "device_type", "operating_system", "browser", "referrer_host"}); err != nil {
		return err
	}
	for _, event := range events {
		row := []string{
			event.OccurredAt.Local().Format(time.RFC3339),
			event.RouteType,
			event.SourcePath,
			event.ShortLinkSlug,
			event.DestinationHost,
			event.DestinationPath,
			event.DestinationPlatform,
			event.IPNetwork,
			event.IPHash,
			event.DeviceType,
			event.OperatingSystem,
			event.Browser,
			event.ReferrerHost,
		}
		for i := range row {
			row[i] = safeCSVCell(row[i])
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// SessionDetail returns a privacy-safe admin view of one analytics session.
func (s *AnalyticsService) SessionDetail(sessionID string) (AnalyticsSessionDetail, error) {
	if !validSessionID(sessionID) {
		return AnalyticsSessionDetail{}, ErrAnalyticsSessionNotFound
	}
	session, err := s.analytics.FindSession(sessionID)
	if err != nil {
		return AnalyticsSessionDetail{}, err
	}
	if session == nil {
		return AnalyticsSessionDetail{}, ErrAnalyticsSessionNotFound
	}
	timeline, err := s.analytics.SessionTimeline(sessionID, 50)
	if err != nil {
		return AnalyticsSessionDetail{}, err
	}
	items := make([]AnalyticsTimelineItem, 0, len(timeline))
	for _, row := range timeline {
		label := row.Label
		if label == "" {
			label = row.Path
		}
		items = append(items, AnalyticsTimelineItem{
			Time:  row.OccurredAt.Local().Format("Jan 2, 2006 3:04 PM"),
			Kind:  row.Kind,
			Label: truncate(label, 120),
			Path:  row.Path,
		})
	}
	referrer := "Direct"
	if session.ReferrerHost != "" {
		referrer = session.ReferrerHost
	}
	return AnalyticsSessionDetail{
		SessionIDShort:  shortenOpaqueID(session.SessionID),
		FirstSeen:       session.FirstSeenAt.Local().Format("Jan 2, 2006 3:04 PM"),
		LastSeen:        session.LastSeenAt.Local().Format("Jan 2, 2006 3:04 PM"),
		Duration:        friendlyDuration(session.LastSeenAt.Sub(session.FirstSeenAt)),
		PageViewCount:   session.PageViewCount,
		RedirectCount:   session.RedirectCount,
		DeviceType:      firstNonEmptyString(session.DeviceType, "unknown"),
		OperatingSystem: firstNonEmptyString(session.OperatingSystem, "unknown"),
		Browser:         firstNonEmptyString(session.Browser, "unknown"),
		Referrer:        referrer,
		BotStatus:       botStatus(session.IsBot, session.BotName),
		Timeline:        items,
	}, nil
}

// CleanupExpired deletes analytics rows older than the configured retention period.
func (s *AnalyticsService) CleanupExpired(dryRun bool) (repositories.AnalyticsCleanupSummary, error) {
	form, err := s.SettingsForm()
	if err != nil {
		return repositories.AnalyticsCleanupSummary{}, err
	}
	retentionDays := form.DataRetentionDays
	if retentionDays <= 0 {
		retentionDays = defaultAnalyticsRetentionDays
	}
	cutoff := s.clock().AddDate(0, 0, -retentionDays)
	return s.analytics.CleanupBefore(cutoff, dryRun)
}

func (s *AnalyticsService) ensureSessionCookie(c *gin.Context, form AnalyticsSettingsForm) string {
	if cookie, err := c.Request.Cookie(AnalyticsSessionCookie); err == nil && validSessionID(cookie.Value) {
		return cookie.Value
	}
	sessionID := randomToken(32)
	maxAge := form.SessionCookieLifetimeDays * 24 * 60 * 60
	if maxAge <= 0 {
		maxAge = defaultAnalyticsCookieDays * 24 * 60 * 60
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     AnalyticsSessionCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.isProduction,
		SameSite: http.SameSiteLaxMode,
	})
	c.Request.AddCookie(&http.Cookie{Name: AnalyticsSessionCookie, Value: sessionID})
	return sessionID
}

func (s *AnalyticsService) clearCookieIfDisabled(c *gin.Context, form AnalyticsSettingsForm) {
	if form.AnalyticsEnabled {
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: AnalyticsSessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.isProduction, SameSite: http.SameSiteLaxMode})
}

func (s *AnalyticsService) safeLog(message string, err error) {
	if s.logger == nil || err == nil {
		return
	}
	s.logger.Warn(message, slog.String("error", err.Error()))
}

// DeviceInfo contains safe inferred device information.
type DeviceInfo struct {
	DeviceType             string
	DeviceBrand            string
	DeviceModel            string
	OperatingSystem        string
	OperatingSystemVersion string
	Browser                string
	BrowserVersion         string
	Engine                 string
	IsBot                  bool
	BotName                string
}

// DeviceParser parses normal User-Agent and safe Client Hint values.
type DeviceParser struct{}

// Parse returns conservative device information. Unknown is used when unreliable.
func (DeviceParser) Parse(userAgent string, hints map[string]string) DeviceInfo {
	ua := strings.TrimSpace(userAgent)
	lower := strings.ToLower(ua)
	info := DeviceInfo{DeviceType: "unknown", OperatingSystem: "unknown", Browser: "unknown", Engine: "unknown"}
	if ua == "" {
		return info
	}
	if name := detectBot(lower); name != "" {
		info.DeviceType = "bot"
		info.Browser = "bot"
		info.IsBot = true
		info.BotName = name
		return info
	}
	switch {
	case strings.Contains(lower, "ipad"):
		info.DeviceType = "tablet"
		info.OperatingSystem = "iPadOS"
	case strings.Contains(lower, "iphone"), strings.Contains(lower, "ipod"):
		info.DeviceType = "mobile"
		info.OperatingSystem = "iOS"
	case strings.Contains(lower, "android"):
		if strings.Contains(lower, "mobile") {
			info.DeviceType = "mobile"
		} else {
			info.DeviceType = "tablet"
		}
		info.OperatingSystem = "Android"
	case strings.Contains(lower, "windows"):
		info.DeviceType = "desktop"
		info.OperatingSystem = "Windows"
	case strings.Contains(lower, "mac os x"), strings.Contains(lower, "macintosh"):
		info.DeviceType = "desktop"
		info.OperatingSystem = "macOS"
	case strings.Contains(lower, "cros"):
		info.DeviceType = "desktop"
		info.OperatingSystem = "ChromeOS"
	case strings.Contains(lower, "linux"):
		info.DeviceType = "desktop"
		info.OperatingSystem = "Linux"
	}
	if mobile, ok := hints["Sec-CH-UA-Mobile"]; ok && mobile == "?1" && info.DeviceType != "tablet" {
		info.DeviceType = "mobile"
	}
	if platform := cleanCH(hints["Sec-CH-UA-Platform"]); platform != "" {
		info.OperatingSystem = normalizePlatform(platform)
	}
	info.OperatingSystemVersion = firstVersion(lower, []string{"android ", "cpu iphone os ", "cpu os ", "mac os x ", "windows nt "})
	info.DeviceModel = truncate(cleanCH(hints["Sec-CH-UA-Model"]), 80)
	switch {
	case strings.Contains(lower, "samsungbrowser/"):
		info.Browser = "Samsung Internet"
		info.BrowserVersion = versionAfter(lower, "samsungbrowser/")
	case strings.Contains(lower, "edg/"):
		info.Browser = "Edge"
		info.BrowserVersion = versionAfter(lower, "edg/")
	case strings.Contains(lower, "firefox/"):
		info.Browser = "Firefox"
		info.BrowserVersion = versionAfter(lower, "firefox/")
	case strings.Contains(lower, "opr/"), strings.Contains(lower, "opera/"):
		info.Browser = "Opera"
		info.BrowserVersion = versionAfterAny(lower, []string{"opr/", "opera/"})
	case strings.Contains(lower, "chrome/"), strings.Contains(lower, "crios/"):
		info.Browser = "Chrome"
		info.BrowserVersion = versionAfterAny(lower, []string{"chrome/", "crios/"})
	case strings.Contains(lower, "safari/") && strings.Contains(lower, "version/"):
		info.Browser = "Safari"
		info.BrowserVersion = versionAfter(lower, "version/")
	}
	switch {
	case strings.Contains(lower, "applewebkit"):
		info.Engine = "WebKit"
	case strings.Contains(lower, "gecko"):
		info.Engine = "Gecko"
	}
	return info
}

// IPInfo contains privacy-preserving IP processing output.
type IPInfo struct {
	Hash     string
	Network  string
	Excluded bool
}

// IPProcessor normalizes, hashes, and truncates IP data.
type IPProcessor struct {
	secret     string
	mode       string
	exclusions []*net.IPNet
}

// NewIPProcessor constructs an IPProcessor.
func NewIPProcessor(secret, mode string, exclusions []*net.IPNet) IPProcessor {
	if strings.TrimSpace(secret) == "" {
		secret = "development-analytics-secret"
	}
	if mode == "" {
		mode = "hashed"
	}
	return IPProcessor{secret: secret, mode: mode, exclusions: exclusions}
}

// Process returns network-only and optional HMAC-hashed IP information.
func (p IPProcessor) Process(value string) IPInfo {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return IPInfo{}
	}
	for _, cidr := range p.exclusions {
		if cidr.Contains(ip) {
			return IPInfo{Excluded: true}
		}
	}
	network := ipNetwork(ip)
	info := IPInfo{Network: network}
	if p.mode == "hashed" {
		info.Hash = hmacString(p.secret, ip.String())
	}
	return info
}

// ReferrerInfo contains safe referrer classification data.
type ReferrerInfo struct {
	Type string
	Name string
	Host string
	Path string
}

// ClassifyReferrer classifies a request referrer without storing external query strings.
func ClassifyReferrer(r *http.Request) ReferrerInfo {
	raw := strings.TrimSpace(r.Header.Get("Referer"))
	if raw == "" {
		return ReferrerInfo{Type: "direct"}
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return ReferrerInfo{Type: "unknown"}
	}
	host := strings.ToLower(parsed.Hostname())
	baseHost := strings.ToLower(r.Host)
	refPath := normalizePath(parsed.EscapedPath())
	if strings.EqualFold(host, baseHost) {
		return ReferrerInfo{Type: "internal", Host: host, Path: refPath}
	}
	if name := searchEngineName(host); name != "" {
		return ReferrerInfo{Type: "search", Name: name, Host: host, Path: refPath}
	}
	if name := socialName(host); name != "" {
		return ReferrerInfo{Type: "social", Name: name, Host: host, Path: refPath}
	}
	return ReferrerInfo{Type: "referral", Host: host, Path: refPath}
}

// AnalyticsRange parses date inputs into an inclusive local date range.
func AnalyticsRange(from, to string, now time.Time) (time.Time, time.Time) {
	defaultEnd := time.Date(now.Local().Year(), now.Local().Month(), now.Local().Day()+1, 0, 0, 0, 0, time.Local)
	defaultStart := defaultEnd.AddDate(0, 0, -7)
	start := parseDateOnly(from)
	end := parseDateOnly(to)
	if start.IsZero() {
		start = defaultStart
	}
	if end.IsZero() {
		end = defaultEnd
	} else {
		end = end.AddDate(0, 0, 1)
	}
	if !start.Before(end) {
		return defaultStart, defaultEnd
	}
	if end.Sub(start) > 366*24*time.Hour {
		return end.AddDate(-1, 0, 0), end
	}
	return start, end
}

func analyticsRangeLabel(start, end time.Time, rawFrom, rawTo string) string {
	if strings.TrimSpace(rawFrom) == "" && strings.TrimSpace(rawTo) == "" {
		return "Last 7 days"
	}
	endInclusive := end.AddDate(0, 0, -1)
	if start.Year() == endInclusive.Year() {
		return start.Local().Format("Jan 2") + " to " + endInclusive.Local().Format("Jan 2, 2006")
	}
	return start.Local().Format("Jan 2, 2006") + " to " + endInclusive.Local().Format("Jan 2, 2006")
}

// ValidateAnalyticsSettings validates analytics settings without executable values.
func ValidateAnalyticsSettings(form AnalyticsSettingsForm) error {
	fieldErrors := map[string]string{}
	switch form.IPHandlingMode {
	case "none", "network_only", "hashed":
	default:
		fieldErrors["ip_handling_mode"] = "Use none, network_only, or hashed."
	}
	if form.RawUserAgentStorageEnabled {
		fieldErrors["raw_user_agent_storage_enabled"] = "Raw User-Agent storage is not supported in the standard dashboard."
	}
	if form.SessionCookieLifetimeDays < 1 || form.SessionCookieLifetimeDays > 365 {
		fieldErrors["session_cookie_lifetime_days"] = "Use a cookie lifetime between 1 and 365 days."
	}
	if form.DataRetentionDays < 7 || form.DataRetentionDays > 3650 {
		fieldErrors["data_retention_days"] = "Use a retention period between 7 and 3650 days."
	}
	if invalidSettingList(form.QueryParameterAllowlist) {
		fieldErrors["query_parameter_allowlist"] = "Use only comma-separated parameter names."
	}
	if invalidSettingList(form.QueryParameterDenylist) {
		fieldErrors["query_parameter_denylist"] = "Use only comma-separated parameter names."
	}
	if invalidCIDRList(form.InternalTrafficExclusionCIDRs) {
		fieldErrors["internal_traffic_exclusion_cidrs"] = "Use comma-separated CIDR values."
	}
	if len(fieldErrors) > 0 {
		return SettingsValidationError{FieldErrors: fieldErrors}
	}
	return nil
}

func shouldTrackRequest(r *http.Request, form AnalyticsSettingsForm) bool {
	if !form.AnalyticsEnabled {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	p := r.URL.Path
	if p == "/health" || p == "/robots.txt" || p == "/sitemap.xml" ||
		strings.HasPrefix(p, "/admin") || strings.HasPrefix(p, "/setup") ||
		strings.HasPrefix(p, "/static/") || strings.HasPrefix(p, "/media/") ||
		strings.HasPrefix(p, "/analytics/") || strings.HasPrefix(p, "/privacy/analytics/") {
		return false
	}
	return true
}

func eligiblePageView(r *http.Request, status int) bool {
	return status >= 200 && status < 300 && !isRedirectStatus(status) && r.Method == http.MethodGet
}

func isRedirectStatus(status int) bool {
	return status >= 300 && status < 400
}

func analyticsSettingsFromModel(settings *models.AppSetting) AnalyticsSettingsForm {
	if legacyAnalyticsSettings(settings) {
		return normalizeAnalyticsSettings(AnalyticsSettingsForm{
			AnalyticsEnabled:               true,
			PageViewTrackingEnabled:        true,
			RedirectTrackingEnabled:        true,
			BotTrackingEnabled:             true,
			ExcludeBotsFromDashboard:       true,
			UniqueVisitorEstimationEnabled: true,
			IPHandlingMode:                 "hashed",
			ReferrerTrackingEnabled:        true,
			UTMTrackingEnabled:             false,
			SessionCookieLifetimeDays:      defaultAnalyticsCookieDays,
			DataRetentionDays:              defaultAnalyticsRetentionDays,
			AutomaticCleanupEnabled:        true,
			AnalyticsExportEnabled:         true,
			RespectDoNotTrack:              true,
			RespectGlobalPrivacyControl:    true,
		})
	}
	form := AnalyticsSettingsForm{
		AnalyticsEnabled:               settings.AnalyticsEnabled,
		PageViewTrackingEnabled:        settings.PageViewTrackingEnabled,
		RedirectTrackingEnabled:        settings.RedirectTrackingEnabled,
		BotTrackingEnabled:             settings.BotTrackingEnabled,
		ExcludeBotsFromDashboard:       settings.ExcludeBotsFromDashboard,
		UniqueVisitorEstimationEnabled: settings.UniqueVisitorEstimationEnabled,
		IPHandlingMode:                 settings.IPHandlingMode,
		RawUserAgentStorageEnabled:     settings.RawUserAgentStorageEnabled,
		ReferrerTrackingEnabled:        settings.ReferrerTrackingEnabled,
		UTMTrackingEnabled:             settings.UTMTrackingEnabled,
		ClientSideDeviceDetailsEnabled: settings.ClientSideDeviceDetailsEnabled,
		GeolocationEnrichmentEnabled:   settings.GeolocationEnrichmentEnabled,
		CookieConsentRequired:          settings.CookieConsentRequired,
		SessionCookieLifetimeDays:      settings.SessionCookieLifetimeDays,
		DataRetentionDays:              settings.DataRetentionDays,
		AutomaticCleanupEnabled:        settings.AutomaticCleanupEnabled,
		AnalyticsExportEnabled:         settings.AnalyticsExportEnabled,
		RespectDoNotTrack:              settings.RespectDoNotTrack,
		RespectGlobalPrivacyControl:    settings.RespectGlobalPrivacyControl,
		AdminIPExclusionList:           settings.AdminIPExclusionList,
		InternalTrafficExclusionCIDRs:  settings.InternalTrafficExclusionCIDRs,
		QueryParameterAllowlist:        settings.QueryParameterAllowlist,
		QueryParameterDenylist:         settings.QueryParameterDenylist,
	}
	return normalizeAnalyticsSettings(form)
}

func legacyAnalyticsSettings(settings *models.AppSetting) bool {
	return !settings.AnalyticsEnabled &&
		!settings.PageViewTrackingEnabled &&
		!settings.RedirectTrackingEnabled &&
		!settings.BotTrackingEnabled &&
		!settings.ExcludeBotsFromDashboard &&
		!settings.UniqueVisitorEstimationEnabled &&
		settings.IPHandlingMode == "" &&
		settings.SessionCookieLifetimeDays == 0 &&
		settings.DataRetentionDays == 0
}

func normalizeAnalyticsSettings(form AnalyticsSettingsForm) AnalyticsSettingsForm {
	if form.IPHandlingMode == "" {
		form.IPHandlingMode = "hashed"
	}
	if form.SessionCookieLifetimeDays == 0 {
		form.SessionCookieLifetimeDays = defaultAnalyticsCookieDays
	}
	if form.DataRetentionDays == 0 {
		form.DataRetentionDays = defaultAnalyticsRetentionDays
	}
	if strings.TrimSpace(form.QueryParameterDenylist) == "" {
		form.QueryParameterDenylist = "token,password,code,session,csrf,authorization,access_token,refresh_token"
	}
	form.IPHandlingMode = strings.TrimSpace(form.IPHandlingMode)
	form.QueryParameterAllowlist = normalizeCSVNames(form.QueryParameterAllowlist)
	form.QueryParameterDenylist = normalizeCSVNames(form.QueryParameterDenylist)
	form.InternalTrafficExclusionCIDRs = normalizeCSVNames(form.InternalTrafficExclusionCIDRs)
	form.UTMTrackingEnabled = false
	return form
}

func applyAnalyticsSettings(settings *models.AppSetting, form AnalyticsSettingsForm) {
	settings.AnalyticsEnabled = form.AnalyticsEnabled
	settings.PageViewTrackingEnabled = form.PageViewTrackingEnabled
	settings.RedirectTrackingEnabled = form.RedirectTrackingEnabled
	settings.BotTrackingEnabled = form.BotTrackingEnabled
	settings.ExcludeBotsFromDashboard = form.ExcludeBotsFromDashboard
	settings.UniqueVisitorEstimationEnabled = form.UniqueVisitorEstimationEnabled
	settings.IPHandlingMode = form.IPHandlingMode
	settings.RawUserAgentStorageEnabled = form.RawUserAgentStorageEnabled
	settings.ReferrerTrackingEnabled = form.ReferrerTrackingEnabled
	settings.UTMTrackingEnabled = false
	settings.ClientSideDeviceDetailsEnabled = form.ClientSideDeviceDetailsEnabled
	settings.GeolocationEnrichmentEnabled = form.GeolocationEnrichmentEnabled
	settings.CookieConsentRequired = form.CookieConsentRequired
	settings.SessionCookieLifetimeDays = form.SessionCookieLifetimeDays
	settings.DataRetentionDays = form.DataRetentionDays
	settings.AutomaticCleanupEnabled = form.AutomaticCleanupEnabled
	settings.AnalyticsExportEnabled = form.AnalyticsExportEnabled
	settings.RespectDoNotTrack = form.RespectDoNotTrack
	settings.RespectGlobalPrivacyControl = form.RespectGlobalPrivacyControl
	settings.AdminIPExclusionList = form.AdminIPExclusionList
	settings.InternalTrafficExclusionCIDRs = form.InternalTrafficExclusionCIDRs
	settings.QueryParameterAllowlist = form.QueryParameterAllowlist
	settings.QueryParameterDenylist = form.QueryParameterDenylist
}

func randomToken(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(bytes)
}

func validSessionID(value string) bool {
	if len(value) < 32 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'f') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func hmacString(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func ipNetwork(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2])
	}
	ip = ip.To16()
	if ip == nil {
		return ""
	}
	return fmt.Sprintf("%x:%x:%x::/48", uint16(ip[0])<<8|uint16(ip[1]), uint16(ip[2])<<8|uint16(ip[3]), uint16(ip[4])<<8|uint16(ip[5]))
}

func parseCIDRs(value string) []*net.IPNet {
	parts := strings.Split(value, ",")
	out := make([]*net.IPNet, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		_, cidr, err := net.ParseCIDR(trimmed)
		if err == nil {
			out = append(out, cidr)
		}
	}
	return out
}

func normalizePath(value string) string {
	if value == "" {
		return "/"
	}
	cleaned := path.Clean("/" + strings.TrimSpace(value))
	if cleaned == "." {
		return "/"
	}
	return truncate(cleaned, maxAnalyticsText)
}

func safeURLParts(raw string) (string, string) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", ""
	}
	return truncate(strings.ToLower(parsed.Hostname()), maxAnalyticsText), normalizePath(parsed.EscapedPath())
}

func safeCSVCell(value string) string {
	if value == "" {
		return ""
	}
	switch value[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func shortenOpaqueID(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:8] + "..." + value[len(value)-4:]
}

func friendlyDuration(value time.Duration) string {
	if value < time.Minute {
		return "under 1 minute"
	}
	if value < time.Hour {
		return fmt.Sprintf("%d minutes", int(value.Minutes()))
	}
	return fmt.Sprintf("%d hours %d minutes", int(value.Hours()), int(value.Minutes())%60)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func botStatus(isBot bool, name string) string {
	if !isBot {
		return "Human traffic"
	}
	if name == "" {
		return "Bot traffic"
	}
	return "Bot traffic: " + name
}

func analyticsRedirectRows(events []models.RedirectEvent) []AnalyticsRedirectRow {
	rows := make([]AnalyticsRedirectRow, 0, len(events))
	for _, event := range events {
		rows = append(rows, AnalyticsRedirectRow{
			Time:        event.OccurredAt.Local().Format("Jan 2, 2006 3:04 PM"),
			Route:       firstNonEmptyString(event.RouteType, "unknown"),
			Platform:    firstNonEmptyString(event.DestinationPlatform, "unknown"),
			SourcePath:  event.SourcePath,
			Slug:        event.ShortLinkSlug,
			IPNetwork:   firstNonEmptyString(event.IPNetwork, "not stored"),
			IPHashShort: shortenOpaqueID(event.IPHash),
			Device:      firstNonEmptyString(event.DeviceType, "unknown"),
			OS:          firstNonEmptyString(event.OperatingSystem, "unknown"),
			Browser:     firstNonEmptyString(event.Browser, "unknown"),
			Referrer:    firstNonEmptyString(event.ReferrerHost, "direct"),
		})
	}
	return rows
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func detectBot(lower string) string {
	bots := []struct{ key, name string }{
		{"googlebot", "Googlebot"},
		{"bingbot", "Bingbot"},
		{"slurp", "Yahoo Slurp"},
		{"duckduckbot", "DuckDuckBot"},
		{"baiduspider", "Baidu Spider"},
		{"yandexbot", "YandexBot"},
		{"facebookexternalhit", "Facebook Preview"},
		{"twitterbot", "X/Twitter Preview"},
		{"linkedinbot", "LinkedIn Preview"},
		{"slackbot", "Slackbot"},
		{"uptimerobot", "UptimeRobot"},
	}
	for _, bot := range bots {
		if strings.Contains(lower, bot.key) {
			return bot.name
		}
	}
	if strings.Contains(lower, "bot") || strings.Contains(lower, "crawler") || strings.Contains(lower, "spider") {
		return "Unknown bot"
	}
	return ""
}

func searchEngineName(host string) string {
	switch {
	case strings.Contains(host, "google."):
		return "Google"
	case strings.Contains(host, "bing."):
		return "Bing"
	case strings.Contains(host, "duckduckgo."):
		return "DuckDuckGo"
	case strings.Contains(host, "yahoo."):
		return "Yahoo"
	default:
		return ""
	}
}

func socialName(host string) string {
	switch {
	case strings.Contains(host, "facebook."), strings.Contains(host, "fb."):
		return "Facebook"
	case strings.Contains(host, "instagram."):
		return "Instagram"
	case strings.Contains(host, "twitter."), strings.Contains(host, "x.com"), strings.Contains(host, "t.co"):
		return "X/Twitter"
	case strings.Contains(host, "linkedin."):
		return "LinkedIn"
	default:
		return ""
	}
}

func clientHints(r *http.Request) map[string]string {
	keys := []string{"Sec-CH-UA", "Sec-CH-UA-Mobile", "Sec-CH-UA-Platform", "Sec-CH-UA-Platform-Version", "Sec-CH-UA-Model"}
	hints := map[string]string{}
	for _, key := range keys {
		if value := truncate(r.Header.Get(key), 120); value != "" {
			hints[key] = value
		}
	}
	return hints
}

func cleanCH(value string) string {
	value = strings.Trim(value, `" `)
	value = strings.ReplaceAll(value, "\\", "")
	return truncate(value, 120)
}

func normalizePlatform(value string) string {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "android"):
		return "Android"
	case strings.Contains(lower, "ios"):
		return "iOS"
	case strings.Contains(lower, "mac"):
		return "macOS"
	case strings.Contains(lower, "windows"):
		return "Windows"
	case strings.Contains(lower, "chrome"):
		return "ChromeOS"
	case strings.Contains(lower, "linux"):
		return "Linux"
	default:
		return truncate(value, 80)
	}
}

func firstVersion(lower string, prefixes []string) string {
	for _, prefix := range prefixes {
		if version := versionAfter(lower, prefix); version != "" {
			return strings.ReplaceAll(version, "_", ".")
		}
	}
	return ""
}

func versionAfterAny(lower string, prefixes []string) string {
	for _, prefix := range prefixes {
		if version := versionAfter(lower, prefix); version != "" {
			return version
		}
	}
	return ""
}

func versionAfter(lower, prefix string) string {
	idx := strings.Index(lower, prefix)
	if idx < 0 {
		return ""
	}
	rest := lower[idx+len(prefix):]
	end := strings.IndexAny(rest, " ;)(")
	if end >= 0 {
		rest = rest[:end]
	}
	return truncate(rest, 40)
}

func parseDateOnly(value string) time.Time {
	parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), time.Local)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func normalizeCSVNames(value string) string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return strings.Join(out, ",")
}

func invalidSettingList(value string) bool {
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		for _, r := range trimmed {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
				return true
			}
		}
	}
	return false
}

func invalidCIDRList(value string) bool {
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(trimmed); err != nil {
			return true
		}
	}
	return false
}
