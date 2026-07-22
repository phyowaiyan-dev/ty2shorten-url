package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAnalyticsDeviceParser(t *testing.T) {
	parser := DeviceParser{}
	tests := []struct {
		name        string
		userAgent   string
		wantDevice  string
		wantOS      string
		wantBrowser string
		wantBot     bool
	}{
		{name: "android chrome", userAgent: "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0 Mobile Safari/537.36", wantDevice: "mobile", wantOS: "Android", wantBrowser: "Chrome"},
		{name: "samsung internet", userAgent: "Mozilla/5.0 (Linux; Android 13; SAMSUNG SM-S918B) AppleWebKit/537.36 SamsungBrowser/24.0 Chrome/117.0 Mobile Safari/537.36", wantDevice: "mobile", wantOS: "Android", wantBrowser: "Samsung Internet"},
		{name: "iphone safari", userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1", wantDevice: "mobile", wantOS: "iOS", wantBrowser: "Safari"},
		{name: "windows edge", userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/125.0 Safari/537.36 Edg/125.0", wantDevice: "desktop", wantOS: "Windows", wantBrowser: "Edge"},
		{name: "linux firefox", userAgent: "Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0", wantDevice: "desktop", wantOS: "Linux", wantBrowser: "Firefox"},
		{name: "googlebot", userAgent: "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; Googlebot/2.1; +http://www.google.com/bot.html)", wantDevice: "bot", wantOS: "unknown", wantBrowser: "bot", wantBot: true},
		{name: "empty", userAgent: "", wantDevice: "unknown", wantOS: "unknown", wantBrowser: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.Parse(tt.userAgent, nil)
			if got.DeviceType != tt.wantDevice || got.OperatingSystem != tt.wantOS || got.Browser != tt.wantBrowser || got.IsBot != tt.wantBot {
				t.Fatalf("Parse() = %+v", got)
			}
		})
	}
}

func TestAnalyticsPrivacyHelpers(t *testing.T) {
	processor := NewIPProcessor("analytics-secret", "hashed", nil)
	info := processor.Process("203.0.113.42")
	if info.Hash == "" || strings.Contains(info.Hash, "203.0.113.42") {
		t.Fatalf("expected HMAC hash without raw IP, got %+v", info)
	}
	if info.Network != "203.0.113.0/24" {
		t.Fatalf("network = %q, want /24 network", info.Network)
	}

	second := NewIPProcessor("analytics-secret", "hashed", nil).Process("203.0.113.42")
	if info.Hash != second.Hash {
		t.Fatal("same secret and IP should produce deterministic hash")
	}
	rotated := NewIPProcessor("rotated-secret", "hashed", nil).Process("203.0.113.42")
	if rotated.Hash == info.Hash {
		t.Fatal("rotated secret should produce a different hash")
	}
}

func TestAnalyticsSettingsValidation(t *testing.T) {
	form := AnalyticsSettingsForm{
		AnalyticsEnabled:               true,
		PageViewTrackingEnabled:        true,
		RedirectTrackingEnabled:        true,
		BotTrackingEnabled:             true,
		ExcludeBotsFromDashboard:       true,
		UniqueVisitorEstimationEnabled: true,
		IPHandlingMode:                 "hashed",
		SessionCookieLifetimeDays:      30,
		DataRetentionDays:              90,
		AutomaticCleanupEnabled:        true,
		RespectDoNotTrack:              true,
		RespectGlobalPrivacyControl:    true,
		QueryParameterDenylist:         "token,password,code",
	}
	if err := ValidateAnalyticsSettings(form); err != nil {
		t.Fatalf("valid settings returned error: %v", err)
	}
	form.IPHandlingMode = "raw"
	if err := ValidateAnalyticsSettings(form); err == nil {
		t.Fatal("raw IP mode should be rejected")
	}
}

func TestReferrerClassification(t *testing.T) {
	tests := []struct {
		name     string
		referrer string
		baseHost string
		wantType string
		wantName string
		wantHost string
	}{
		{name: "direct", baseHost: "app.example.com", wantType: "direct"},
		{name: "google", referrer: "https://www.google.com/search?q=test", baseHost: "app.example.com", wantType: "search", wantName: "Google", wantHost: "www.google.com"},
		{name: "facebook", referrer: "https://facebook.com/story.php?id=1", baseHost: "app.example.com", wantType: "social", wantName: "Facebook", wantHost: "facebook.com"},
		{name: "internal", referrer: "https://app.example.com/privacy?token=x", baseHost: "app.example.com", wantType: "internal", wantHost: "app.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "https://"+tt.baseHost+"/get", nil)
			if tt.referrer != "" {
				request.Header.Set("Referer", tt.referrer)
			}
			got := ClassifyReferrer(request)
			if got.Type != tt.wantType || got.Name != tt.wantName || got.Host != tt.wantHost {
				t.Fatalf("ClassifyReferrer() = %+v", got)
			}
			if strings.Contains(got.Path, "token") {
				t.Fatalf("referrer path leaked query: %+v", got)
			}
		})
	}
}

func TestAnalyticsRangeDefaults(t *testing.T) {
	start, end := AnalyticsRange("", "", time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC))
	if !start.Equal(time.Date(2026, 7, 16, 0, 0, 0, 0, time.Local)) || !end.Equal(time.Date(2026, 7, 23, 0, 0, 0, 0, time.Local)) {
		t.Fatalf("default range = %s - %s", start, end)
	}
}
