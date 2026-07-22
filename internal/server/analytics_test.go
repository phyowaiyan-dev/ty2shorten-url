package server_test

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
)

func TestAnalyticsTracksPublicPageAndRedirect(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	seedShortLink(t, app.db, "download-1", "https://example.com/download?token=secret#fragment", true)

	recorder := app.do(t, http.MethodGet, "/?token=secret&utm_source=newsletter", nil, map[string]string{
		"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 Version/17.5 Mobile/15E148 Safari/604.1",
		"Referer":    "https://www.google.com/search?q=thai",
	})
	assertStatus(t, recorder, http.StatusOK)
	if cookie := app.cookies["ty2_analytics_session"]; cookie == nil || cookie.HttpOnly == false {
		t.Fatalf("analytics session cookie missing or not httpOnly: %+v", cookie)
	}

	recorder = app.do(t, http.MethodGet, "/r/download-1?utm_source=newsletter", nil, map[string]string{
		"User-Agent": "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/125.0 Mobile Safari/537.36",
	})
	assertStatus(t, recorder, http.StatusTemporaryRedirect)

	var sessions int64
	if err := app.db.Model(&models.VisitorSession{}).Count(&sessions).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessions != 1 {
		t.Fatalf("sessions = %d, want 1 reused analytics session", sessions)
	}

	var pageView models.PageView
	if err := app.db.First(&pageView).Error; err != nil {
		t.Fatalf("page view not recorded: %v", err)
	}
	if pageView.NormalizedPath != "/" || pageView.ReferrerType != "search" || pageView.UTMSource != "" {
		t.Fatalf("unexpected page view: %+v", pageView)
	}
	if strings.Contains(pageView.QueryMetadata, "token") {
		t.Fatalf("page view query metadata leaked sensitive data: %s", pageView.QueryMetadata)
	}

	var redirect models.RedirectEvent
	if err := app.db.First(&redirect).Error; err != nil {
		t.Fatalf("redirect event not recorded: %v", err)
	}
	if redirect.RouteType != "short_link" || redirect.ShortLinkSlug != "download-1" || redirect.DestinationHost != "example.com" || redirect.DeviceType != "mobile" {
		t.Fatalf("unexpected redirect event: %+v", redirect)
	}
	if strings.Contains(redirect.DestinationPath, "token") || strings.Contains(redirect.DestinationPath, "fragment") {
		t.Fatalf("redirect destination path leaked sensitive URL parts: %q", redirect.DestinationPath)
	}
}

func TestAnalyticsTracksWithoutVisitorConsentPrompt(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	if err := app.db.Model(&models.AppSetting{}).Where("1 = 1").Update("cookie_consent_required", true).Error; err != nil {
		t.Fatalf("enable consent requirement: %v", err)
	}

	assertStatus(t, app.do(t, http.MethodGet, "/", nil, nil), http.StatusOK)
	var pageViews int64
	if err := app.db.Model(&models.PageView{}).Count(&pageViews).Error; err != nil {
		t.Fatalf("count page views: %v", err)
	}
	if pageViews != 1 || app.cookies["ty2_analytics_session"] == nil {
		t.Fatalf("tracking should start immediately, pageViews=%d cookie=%+v", pageViews, app.cookies["ty2_analytics_session"])
	}
	recorder := app.do(t, http.MethodGet, "/privacy", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	for _, removed := range []string{"Analytics privacy controls", "Allow analytics", "Opt out", "Delete my analytics session"} {
		if strings.Contains(recorder.Body.String(), removed) {
			t.Fatalf("privacy page should not show visitor tracking prompt %q: %s", removed, recorder.Body.String())
		}
	}
}

func TestAnalyticsDNTStillTracksSiteTraffic(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	assertStatus(t, app.do(t, http.MethodGet, "/", nil, map[string]string{"DNT": "1", "Sec-GPC": "1"}), http.StatusOK)
	var pageViews int64
	if err := app.db.Model(&models.PageView{}).Count(&pageViews).Error; err != nil {
		t.Fatalf("count page views: %v", err)
	}
	if pageViews != 1 || app.cookies["ty2_analytics_session"] == nil {
		t.Fatalf("DNT/GPC should not stop first-party site analytics, pageViews=%d cookie=%+v", pageViews, app.cookies["ty2_analytics_session"])
	}
}

func TestAdminAnalyticsSessionDetailAndCleanup(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	assertStatus(t, app.do(t, http.MethodGet, "/", nil, nil), http.StatusOK)
	sessionID := app.cookies["ty2_analytics_session"].Value
	oldTime := time.Now().AddDate(0, 0, -120)
	oldSession := models.VisitorSession{SessionID: "oldsession1234567890abcdef1234567890", FirstSeenAt: oldTime, LastSeenAt: oldTime, StartedAt: oldTime}
	oldView := models.PageView{SessionID: oldSession.SessionID, Path: "/", NormalizedPath: "/", Method: "GET", StatusCode: 200, OccurredAt: oldTime, CreatedAt: oldTime}
	oldRedirect := models.RedirectEvent{SessionID: oldSession.SessionID, RouteType: "android", SourcePath: "/android", RedirectStatus: 307, OccurredAt: oldTime, CreatedAt: oldTime}
	if err := app.db.Create(&oldSession).Error; err != nil {
		t.Fatalf("seed old session: %v", err)
	}
	if err := app.db.Create(&oldView).Error; err != nil {
		t.Fatalf("seed old view: %v", err)
	}
	if err := app.db.Create(&oldRedirect).Error; err != nil {
		t.Fatalf("seed old redirect: %v", err)
	}

	app.login(t)
	recorder := app.do(t, http.MethodGet, "/admin/analytics/sessions/"+sessionID, nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{"Session detail", "Timeline", "Page views", "Redirects", "Raw IP is not shown"} {
		if !strings.Contains(body, want) {
			t.Fatalf("session detail missing marker %q: %s", want, body)
		}
	}
	if strings.Contains(body, sessionID) {
		t.Fatalf("session detail should not display full session id: %s", body)
	}

	token := app.csrfToken(t, "/admin/analytics")
	recorder = app.do(t, http.MethodPost, "/admin/analytics/cleanup", url.Values{"csrf_token": {token}}, nil)
	assertStatus(t, recorder, http.StatusSeeOther)

	var oldCount int64
	if err := app.db.Model(&models.VisitorSession{}).Where("session_id = ?", oldSession.SessionID).Count(&oldCount).Error; err != nil {
		t.Fatalf("count old session: %v", err)
	}
	if oldCount != 0 {
		t.Fatalf("old analytics session should be cleaned up")
	}
	var currentCount int64
	if err := app.db.Model(&models.VisitorSession{}).Where("session_id = ?", sessionID).Count(&currentCount).Error; err != nil {
		t.Fatalf("count current session: %v", err)
	}
	if currentCount != 1 {
		t.Fatalf("current analytics session should be retained")
	}
}

func TestAnalyticsExcludesAdmin(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	assertStatus(t, app.do(t, http.MethodGet, "/admin", nil, nil), http.StatusOK)

	var pageViews int64
	if err := app.db.Model(&models.PageView{}).Count(&pageViews).Error; err != nil {
		t.Fatalf("count page views: %v", err)
	}
	if pageViews != 0 {
		t.Fatalf("page views = %d, want admin traffic excluded", pageViews)
	}
}

func TestAdminAnalyticsDashboardAndExport(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	assertStatus(t, app.do(t, http.MethodGet, "/", nil, nil), http.StatusOK)
	assertStatus(t, app.do(t, http.MethodGet, "/android", nil, nil), http.StatusTemporaryRedirect)

	recorder := app.do(t, http.MethodGet, "/admin/analytics", nil, nil)
	assertStatus(t, recorder, http.StatusSeeOther)

	app.login(t)
	recorder = app.do(t, http.MethodGet, "/admin/analytics", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{"Analytics", "Traffic intelligence", "Page views", "Redirect events", "Device mix", "Export CSV", "Last 7 days", "Recent Android and iOS redirects", "IP network", "IP hash", "android"} {
		if !strings.Contains(body, want) {
			t.Fatalf("analytics dashboard missing marker %q: %s", want, body)
		}
	}

	recorder = app.do(t, http.MethodGet, "/admin/analytics/export?type=redirects&from=2026-01-01&to=2026-12-31", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/csv") {
		t.Fatalf("Content-Type = %q, want CSV", got)
	}
	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v\n%s", err, recorder.Body.String())
	}
	if len(records) < 2 || records[0][0] != "occurred_at" || records[0][1] != "route_type" {
		t.Fatalf("unexpected CSV records: %#v", records)
	}
}

func TestAnalyticsSettingsUpdate(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	recorder := app.do(t, http.MethodGet, "/admin/settings/analytics", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{"Analytics Settings", "Privacy controls", "IP handling mode", "Data retention"} {
		if !strings.Contains(body, want) {
			t.Fatalf("analytics settings missing marker %q: %s", want, body)
		}
	}

	token := app.csrfToken(t, "/admin/settings/analytics")
	recorder = app.do(t, http.MethodPost, "/admin/settings/analytics", url.Values{
		"csrf_token":                        {token},
		"analytics_enabled":                 {"1"},
		"page_view_tracking_enabled":        {"1"},
		"redirect_tracking_enabled":         {"1"},
		"bot_tracking_enabled":              {"1"},
		"exclude_bots_from_dashboard":       {"1"},
		"unique_visitor_estimation_enabled": {"1"},
		"ip_handling_mode":                  {"network_only"},
		"referrer_tracking_enabled":         {"1"},
		"session_cookie_lifetime_days":      {"14"},
		"data_retention_days":               {"30"},
		"automatic_cleanup_enabled":         {"1"},
		"analytics_export_enabled":          {"1"},
		"respect_do_not_track":              {"1"},
		"respect_global_privacy_control":    {"1"},
		"query_parameter_denylist":          {"token,password,code"},
	}, nil)
	assertStatus(t, recorder, http.StatusSeeOther)

	var settings models.AppSetting
	if err := app.db.First(&settings).Error; err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.IPHandlingMode != "network_only" || settings.DataRetentionDays != 30 || !settings.AnalyticsEnabled || settings.UTMTrackingEnabled {
		t.Fatalf("analytics settings not saved: %+v", settings)
	}
}
