package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/config"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/database"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/server"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/validation"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var csrfPattern = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)

type routeTestApp struct {
	router  http.Handler
	db      *gorm.DB
	cookies map[string]*http.Cookie
}

func newRouteTestApp(t *testing.T) *routeTestApp {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "routes.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(db); err != nil {
			t.Fatalf("database.Close returned error: %v", err)
		}
	})
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("database.AutoMigrate returned error: %v", err)
	}

	router, err := server.NewRouter(server.Dependencies{
		Config: config.Config{
			AppEnv:        "test",
			AppHost:       "127.0.0.1",
			AppPort:       8722,
			DatabasePath:  dbPath,
			MediaPath:     filepath.Join(tempDir, "media"),
			SessionSecret: "test-secret-that-is-long-enough",
			BaseURL:       "http://localhost:8722",
		},
		DB:     db,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("server.NewRouter returned error: %v", err)
	}

	return &routeTestApp{
		router:  router,
		db:      db,
		cookies: map[string]*http.Cookie{},
	}
}

func TestSetupIncompleteRouteProtection(t *testing.T) {
	app := newRouteTestApp(t)

	recorder := app.do(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/setup")

	recorder = app.do(t, http.MethodGet, "/health", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
}

func TestSuccessfulSetup(t *testing.T) {
	app := newRouteTestApp(t)
	token := app.csrfToken(t, "/setup")

	recorder := app.do(t, http.MethodPost, "/setup", setupForm(token), nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin/login")

	var settings models.AppSetting
	if err := app.db.First(&settings).Error; err != nil {
		t.Fatalf("load app settings: %v", err)
	}
	if !settings.IsSetupCompleted {
		t.Fatal("setup was not marked complete")
	}

	var admin models.AdminUser
	if err := app.db.First(&admin).Error; err != nil {
		t.Fatalf("load admin: %v", err)
	}
	if admin.Email != "admin@example.com" {
		t.Fatalf("admin email = %q, want normalized email", admin.Email)
	}
}

func TestRepeatedSetupRejection(t *testing.T) {
	app := newRouteTestApp(t)
	token := app.csrfToken(t, "/setup")
	assertStatus(t, app.do(t, http.MethodPost, "/setup", setupForm(token), nil), http.StatusSeeOther)

	recorder := app.do(t, http.MethodPost, "/setup", setupForm(token), nil)
	assertStatus(t, recorder, http.StatusForbidden)
}

func TestInvalidSetupForm(t *testing.T) {
	app := newRouteTestApp(t)
	token := app.csrfToken(t, "/setup")

	form := setupForm(token)
	form.Set("admin_password", "short")
	form.Set("password_confirm", "different")
	form.Set("android_url", "ftp://example.com/app")

	recorder := app.do(t, http.MethodPost, "/setup", form, nil)
	assertStatus(t, recorder, http.StatusUnprocessableEntity)
	if !strings.Contains(recorder.Body.String(), "Use a password with at least 10 characters.") {
		t.Fatalf("expected friendly validation error, got: %s", recorder.Body.String())
	}
}

func TestSuccessfulLogin(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	token := app.csrfToken(t, "/admin/login")
	recorder := app.do(t, http.MethodPost, "/admin/login", loginForm(token, "admin@example.com", "strong-password"), nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin")

	recorder = app.do(t, http.MethodGet, "/admin", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), "Dashboard") {
		t.Fatalf("expected dashboard response, got: %s", recorder.Body.String())
	}
}

func TestAdminDashboardShowsSetupScoreAndMissingItems(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	recorder := app.do(t, http.MethodGet, "/admin", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		"Launch readiness",
		"Operational snapshot",
		"7 of 11 setup items complete",
		"64%",
		"Redirect activity",
		"Production checklist",
		"Missing setup",
		"Create short link",
		"Branding assets",
		"SEO metadata",
		"Short link inventory",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected dashboard marker %q, got: %s", want, body)
		}
	}
}

func TestAccountPageShowsLoggedInAdminProfile(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	recorder := app.do(t, http.MethodGet, "/admin/account", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		"Admin User",
		"admin@example.com",
		"Save profile",
		"Profile image",
		"Role",
		"Admin ID",
		"Created",
		"Last updated",
		"Change password",
		`data-password-toggle`,
		`data-password-rules`,
		"At least 10 characters",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected account profile marker %q, got: %s", want, body)
		}
	}
	if strings.Contains(body, `<span aria-hidden="true">U</span>Account`) {
		t.Fatalf("expected duplicate Account nav item to be removed, got: %s", body)
	}
	if strings.Count(body, `href="/admin/account"`) != 1 {
		t.Fatalf("expected account link only in drawer footer, got: %s", body)
	}
}

func TestAccountProfileUpdateStoresUserAndAvatar(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/account")
	recorder := app.doMultipart(t, http.MethodPost, "/admin/account/profile", map[string]string{
		"csrf_token": token,
		"name":       "Ty Admin",
		"email":      "TYADMIN@EXAMPLE.COM",
	}, map[string]uploadFile{
		"avatar_file": {filename: "avatar.png", content: tinyPNG},
	})
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin/account?saved=profile")

	var admin models.AdminUser
	if err := app.db.First(&admin).Error; err != nil {
		t.Fatalf("load admin: %v", err)
	}
	if admin.Name != "Ty Admin" || admin.Email != "tyadmin@example.com" {
		t.Fatalf("profile was not normalized/saved: %+v", admin)
	}
	if !strings.HasPrefix(admin.AvatarURL, "/media/avatar-") {
		t.Fatalf("AvatarURL = %q, want generated avatar media URL", admin.AvatarURL)
	}

	recorder = app.do(t, http.MethodGet, "/admin/account?saved=profile", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{"Ty Admin", "tyadmin@example.com", admin.AvatarURL, "Profile updated."} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected updated account marker %q, got: %s", want, body)
		}
	}
}

func TestAboutProjectPageRendersReadOnlyMarkdown(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	recorder := app.do(t, http.MethodGet, "/admin/about", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		"About Project",
		"Project information",
		"Read only",
		"ty2shorten-url is a self-hosted mobile redirect",
		"hello@phyowaiyan.com",
		"Commercial use requires a purchased commercial license",
		`class="admin-brand" href="/admin/about" aria-current="page"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected about page marker %q, got: %s", want, body)
		}
	}
	if strings.Contains(body, `<span aria-hidden="true">I</span>About Project`) {
		t.Fatalf("expected about project to be opened from the drawer brand only, got: %s", body)
	}
	if strings.Contains(body, `action="/admin/about"`) {
		t.Fatalf("expected about page to be read-only, got: %s", body)
	}
}

func TestLandingPageShowsQRCode(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	if !strings.Contains(body, "data:image/png;base64,") || !strings.Contains(body, `href="/get"`) {
		t.Fatalf("expected QR code and /get URL on landing page, got: %s", body)
	}
}

func TestLandingPageProductionLayout(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		`class="public-site"`,
		`class="public-appbar"`,
		`class="public-hero"`,
		`class="hero-qr-panel"`,
		`class="public-footer"`,
		`aria-label="Primary navigation"`,
		`aria-label="App download links"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected production landing marker %q, got: %s", want, body)
		}
	}
	if strings.Contains(body, "One link for every device") || strings.Contains(body, `class="download-section"`) {
		t.Fatalf("expected landing page to collapse download section into hero, got: %s", body)
	}
}

func TestLegalPagesUsePublicLayoutChrome(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	for _, path := range []string{"/privacy", "/terms"} {
		recorder := app.do(t, http.MethodGet, path, nil, nil)
		assertStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		for _, want := range []string{
			`class="public-appbar"`,
			`aria-label="Primary navigation"`,
			`href="/"`,
			`href="/get"`,
			`class="legal-page"`,
			`class="legal-markdown"`,
			`Back to home`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("%s missing public layout marker %q, got: %s", path, want, body)
			}
		}
	}
}

func TestLegalContentSettingsUpdate(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings/legal")
	recorder := app.do(t, http.MethodPost, "/admin/settings/legal", url.Values{
		"csrf_token":              {token},
		"privacy_policy_markdown": {"# Privacy Policy\n\nWe collect only the data needed to send users to the right app store."},
		"terms_markdown":          {"# Terms of Use\n\nUse this redirect service responsibly."},
	}, nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin/settings/legal?saved=1")

	var settings models.AppSetting
	if err := app.db.First(&settings).Error; err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if !strings.Contains(settings.PrivacyPolicyMarkdown, "We collect only") || !strings.Contains(settings.TermsMarkdown, "responsibly") {
		t.Fatalf("legal content was not saved: %+v", settings)
	}

	for path, want := range map[string]string{
		"/privacy": "We collect only the data needed",
		"/terms":   "Use this redirect service responsibly.",
	} {
		recorder = app.do(t, http.MethodGet, path, nil, nil)
		assertStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), want) {
			t.Fatalf("%s did not render legal content %q: %s", path, want, recorder.Body.String())
		}
	}
}

func TestSettingsUpdate(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings")
	form := url.Values{
		"csrf_token":       {token},
		"site_name":        {"Updated Site"},
		"site_description": {"Updated redirect page for customers opening app links on mobile devices."},
		"android_url":      {"https://play.google.com/store/apps/details?id=com.example.updated"},
		"apple_url":        {"https://apps.apple.com/app/updated/id456"},
		"default_url":      {"https://example.com/updated"},
		"public_base_url":  {"https://short.example.com"},
		"support_email":    {"HELP@EXAMPLE.COM"},
	}
	recorder := app.do(t, http.MethodPost, "/admin/settings", form, nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin/settings?saved=1")

	var settings models.AppSetting
	if err := app.db.First(&settings).Error; err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if !settings.IsSetupCompleted {
		t.Fatal("settings update reset setup completion")
	}
	if settings.SiteName != "Updated Site" || settings.SupportEmail != "help@example.com" {
		t.Fatalf("settings were not normalized/saved: %+v", settings)
	}
}

func TestSettingsPageShowsProductionGuidance(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	recorder := app.do(t, http.MethodGet, "/admin/settings", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		"Public identity",
		"Redirect routing",
		`placeholder="https://play.google.com/store/apps/details?id=com.example.app"`,
		`placeholder="https://apps.apple.com/app/example/id123456789"`,
		`data-count-target="site-name-count"`,
		`data-word-target="site-description-count"`,
		`/static/js/settings-form.js`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected settings guidance marker %q, got: %s", want, body)
		}
	}
}

func TestSettingsUpdateRejectsUnsafeURL(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings")
	form := url.Values{
		"csrf_token":       {token},
		"site_name":        {"Updated Site"},
		"site_description": {"Updated redirect page for customers opening app links on mobile devices."},
		"android_url":      {"javascript:alert(1)"},
		"apple_url":        {"https://apps.apple.com/app/updated/id456"},
		"public_base_url":  {"https://short.example.com"},
	}
	recorder := app.do(t, http.MethodPost, "/admin/settings", form, nil)
	assertStatus(t, recorder, http.StatusUnprocessableEntity)
	if !strings.Contains(recorder.Body.String(), "Only http and https URLs are allowed.") {
		t.Fatalf("expected safe URL validation message, got: %s", recorder.Body.String())
	}
}

func TestSettingsUpdateRejectsWrongStoreDomains(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings")
	form := url.Values{
		"csrf_token":       {token},
		"site_name":        {"Updated Site"},
		"site_description": {"Updated redirect page for customers opening app links on mobile devices."},
		"android_url":      {"https://example.com/android"},
		"apple_url":        {"https://example.com/ios"},
		"public_base_url":  {"https://short.example.com"},
	}
	recorder := app.do(t, http.MethodPost, "/admin/settings", form, nil)
	assertStatus(t, recorder, http.StatusUnprocessableEntity)
	body := recorder.Body.String()
	for _, want := range []string{
		"Use a URL like https://play.google.com/store/apps/details?id=com.example.app.",
		"Use a URL like https://apps.apple.com/app/example/id123456789.",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected store URL validation message %q, got: %s", want, body)
		}
	}
}

func TestShortLinkCreateEditDelete(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/links/new")
	createForm := url.Values{
		"csrf_token":  {token},
		"title":       {"Launch"},
		"slug":        {"Launch_1"},
		"destination": {"https://example.com/launch"},
		"is_active":   {"1"},
	}
	recorder := app.do(t, http.MethodPost, "/admin/links", createForm, nil)
	assertStatus(t, recorder, http.StatusSeeOther)

	var link models.ShortLink
	if err := app.db.Where("slug = ?", "launch_1").First(&link).Error; err != nil {
		t.Fatalf("created link not found: %v", err)
	}

	token = app.csrfToken(t, "/admin/links/"+strconv.FormatUint(uint64(link.ID), 10)+"/edit")
	editForm := url.Values{
		"csrf_token":  {token},
		"title":       {"Launch edited"},
		"slug":        {"launch_edited"},
		"destination": {"https://example.com/edited"},
	}
	recorder = app.do(t, http.MethodPost, "/admin/links/"+strconv.FormatUint(uint64(link.ID), 10), editForm, nil)
	assertStatus(t, recorder, http.StatusSeeOther)

	if err := app.db.First(&link, link.ID).Error; err != nil {
		t.Fatalf("edited link not found: %v", err)
	}
	if link.IsActive {
		t.Fatal("expected unchecked active box to disable link")
	}

	token = app.csrfToken(t, "/admin/links")
	recorder = app.do(t, http.MethodPost, "/admin/links/"+strconv.FormatUint(uint64(link.ID), 10)+"/delete", url.Values{"csrf_token": {token}}, nil)
	assertStatus(t, recorder, http.StatusSeeOther)
}

func TestDuplicateShortLinkSlug(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	seedShortLink(t, app.db, "launch-1", "https://example.com/launch", true)
	app.login(t)

	token := app.csrfToken(t, "/admin/links/new")
	form := url.Values{
		"csrf_token":  {token},
		"title":       {"Duplicate"},
		"slug":        {"launch-1"},
		"destination": {"https://example.com/other"},
		"is_active":   {"1"},
	}
	recorder := app.do(t, http.MethodPost, "/admin/links", form, nil)
	assertStatus(t, recorder, http.StatusUnprocessableEntity)
	if !strings.Contains(recorder.Body.String(), "That slug is already in use.") {
		t.Fatalf("expected duplicate slug message, got: %s", recorder.Body.String())
	}
}

func TestPasswordChangeRotatesSession(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	before := app.cookies["ty2_session"].Value
	token := app.csrfToken(t, "/admin/account")
	form := url.Values{
		"csrf_token":       {token},
		"current_password": {"strong-password"},
		"new_password":     {"new-strong-password"},
		"confirm_password": {"new-strong-password"},
	}
	recorder := app.do(t, http.MethodPost, "/admin/account/password", form, nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	after := app.cookies["ty2_session"].Value
	if before == after {
		t.Fatal("session was not rotated after password change")
	}
}

func TestCSRFRejection(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	recorder := app.do(t, http.MethodPost, "/admin/links", url.Values{
		"title":       {"No token"},
		"slug":        {"no-token"},
		"destination": {"https://example.com/no-token"},
	}, nil)
	assertStatus(t, recorder, http.StatusBadRequest)
}

func TestMissingDestinationFallback(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	if err := app.db.Model(&models.AppSetting{}).Where("1 = 1").Update("android_url", "").Error; err != nil {
		t.Fatalf("clear android url: %v", err)
	}

	recorder := app.do(t, http.MethodGet, "/android", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), "Android app link is not configured yet.") {
		t.Fatalf("expected missing destination message, got: %s", recorder.Body.String())
	}
}

func TestFailedLogin(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	token := app.csrfToken(t, "/admin/login")
	recorder := app.do(t, http.MethodPost, "/admin/login", loginForm(token, "admin@example.com", "wrong-password"), nil)
	assertStatus(t, recorder, http.StatusUnauthorized)
	if !strings.Contains(recorder.Body.String(), "Invalid email or password.") {
		t.Fatalf("expected generic credential error, got: %s", recorder.Body.String())
	}
}

func TestProtectedAdminRoute(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/admin", nil, nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin/login")
}

func TestLogout(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	loginToken := app.csrfToken(t, "/admin/login")
	assertStatus(t, app.do(t, http.MethodPost, "/admin/login", loginForm(loginToken, "admin@example.com", "strong-password"), nil), http.StatusSeeOther)
	logoutToken := app.csrfToken(t, "/admin")

	recorder := app.do(t, http.MethodPost, "/admin/logout", url.Values{"csrf_token": {logoutToken}}, nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin/login")

	recorder = app.do(t, http.MethodGet, "/admin", nil, nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin/login")
}

func TestAndroidRedirect(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/android", nil, nil)
	assertStatus(t, recorder, http.StatusTemporaryRedirect)
	assertHeader(t, recorder, "Location", "https://play.google.com/store/apps/details?id=com.example.app")
}

func TestAppleRedirect(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/apple", nil, nil)
	assertStatus(t, recorder, http.StatusTemporaryRedirect)
	assertHeader(t, recorder, "Location", "https://apps.apple.com/app/example/id123")
}

func TestGetDeviceDetection(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{name: "android", userAgent: "Mozilla/5.0 (Linux; Android 14)", want: "/android"},
		{name: "ios", userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)", want: "/apple"},
		{name: "desktop", userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", want: "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newRouteTestApp(t)
			seedConfiguredApp(t, app.db)

			recorder := app.do(t, http.MethodGet, "/get", nil, map[string]string{"User-Agent": tt.userAgent})
			assertStatus(t, recorder, http.StatusTemporaryRedirect)
			assertHeader(t, recorder, "Location", tt.want)
		})
	}
}

func TestActiveCustomShortLinkRedirect(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	seedShortLink(t, app.db, "launch-1", "https://example.com/launch", true)

	recorder := app.do(t, http.MethodGet, "/r/launch-1", nil, nil)
	assertStatus(t, recorder, http.StatusTemporaryRedirect)
	assertHeader(t, recorder, "Location", "https://example.com/launch")

	var link models.ShortLink
	if err := app.db.Where("slug = ?", "launch-1").First(&link).Error; err != nil {
		t.Fatalf("load short link: %v", err)
	}
	if link.ClickCount != 1 {
		t.Fatalf("ClickCount = %d, want 1", link.ClickCount)
	}
}

func TestMissingShortLink(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/r/missing-1", nil, nil)
	assertStatus(t, recorder, http.StatusNotFound)
}

func TestInactiveShortLink(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	seedShortLink(t, app.db, "sleepy-1", "https://example.com/sleepy", false)

	recorder := app.do(t, http.MethodGet, "/r/sleepy-1", nil, nil)
	assertStatus(t, recorder, http.StatusNotFound)
}

func TestAnimatedNotFoundPageRedirectsHome(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/r/missing-1", nil, nil)
	assertStatus(t, recorder, http.StatusNotFound)
	body := recorder.Body.String()
	for _, want := range []string{`class="error-page error-page--404"`, `Redirecting home in`, `data-countdown="5"`, `http-equiv="refresh" content="5; url=/"`, `/static/js/error-countdown.js`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected animated 404 marker %q, got: %s", want, body)
		}
	}
}

func TestAnimatedForbiddenPageRedirectsHome(t *testing.T) {
	app := newRouteTestApp(t)
	token := app.csrfToken(t, "/setup")
	assertStatus(t, app.do(t, http.MethodPost, "/setup", setupForm(token), nil), http.StatusSeeOther)

	recorder := app.do(t, http.MethodPost, "/setup", setupForm(token), nil)
	assertStatus(t, recorder, http.StatusForbidden)
	body := recorder.Body.String()
	for _, want := range []string{`class="error-page error-page--403"`, `Access denied`, `data-countdown="5"`, `http-equiv="refresh" content="5; url=/"`, `/static/js/error-countdown.js`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected animated 403 marker %q, got: %s", want, body)
		}
	}
}

func TestReservedSlugValidation(t *testing.T) {
	if err := validation.ValidateSlug("admin"); err == nil {
		t.Fatal("ValidateSlug(admin) returned nil, want reserved error")
	}
	if err := validation.ValidateSlug("launch-1"); err != nil {
		t.Fatalf("ValidateSlug(launch-1) returned error: %v", err)
	}
}

func (a *routeTestApp) csrfToken(t *testing.T, path string) string {
	t.Helper()
	recorder := a.do(t, http.MethodGet, path, nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	matches := csrfPattern.FindStringSubmatch(recorder.Body.String())
	if len(matches) != 2 {
		t.Fatalf("csrf token not found in %s response: %s", path, recorder.Body.String())
	}

	return matches[1]
}

func (a *routeTestApp) login(t *testing.T) {
	t.Helper()
	token := a.csrfToken(t, "/admin/login")
	recorder := a.do(t, http.MethodPost, "/admin/login", loginForm(token, "admin@example.com", "strong-password"), nil)
	assertStatus(t, recorder, http.StatusSeeOther)
}

func (a *routeTestApp) do(t *testing.T, method, target string, form url.Values, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	request := httptest.NewRequest(method, target, body)
	if form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	for _, cookie := range a.cookies {
		request.AddCookie(cookie)
	}

	recorder := httptest.NewRecorder()
	a.router.ServeHTTP(recorder, request)
	a.storeCookies(recorder.Result().Cookies())
	return recorder
}

func (a *routeTestApp) storeCookies(cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie.MaxAge < 0 {
			delete(a.cookies, cookie.Name)
			continue
		}
		a.cookies[cookie.Name] = cookie
	}
}

func setupForm(csrfToken string) url.Values {
	return url.Values{
		"csrf_token":       {csrfToken},
		"site_name":        {"TY2 Shorten"},
		"site_description": {"Short links and app redirects."},
		"admin_name":       {"Admin User"},
		"admin_email":      {"ADMIN@EXAMPLE.COM"},
		"admin_password":   {"strong-password"},
		"password_confirm": {"strong-password"},
		"android_url":      {"https://play.google.com/store/apps/details?id=com.example.app"},
		"apple_url":        {"https://apps.apple.com/app/example/id123"},
		"default_url":      {"https://example.com"},
		"support_email":    {"support@example.com"},
		"public_base_url":  {"http://localhost:8722"},
	}
}

func loginForm(csrfToken, email, password string) url.Values {
	return url.Values{
		"csrf_token": {csrfToken},
		"email":      {email},
		"password":   {password},
	}
}

func seedConfiguredApp(t *testing.T, db *gorm.DB) {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("strong-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword returned error: %v", err)
	}

	if err := db.Create(&models.AdminUser{
		Name:         "Admin User",
		Email:        "admin@example.com",
		PasswordHash: string(passwordHash),
	}).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	if err := db.Create(&models.AppSetting{
		SiteName:              "TY2 Shorten",
		SiteDescription:       "Short links and app redirects.",
		AndroidURL:            "https://play.google.com/store/apps/details?id=com.example.app",
		AppleURL:              "https://apps.apple.com/app/example/id123",
		DefaultURL:            "https://example.com",
		PublicBaseURL:         "http://localhost:8722",
		SupportEmail:          "support@example.com",
		FooterEnabled:         true,
		FooterBrandText:       "Audit-ready links",
		FooterDescription:     "A small public footer for the landing page.",
		CopyrightText:         "Copyright 2026",
		PrivacyPolicyMarkdown: "# Privacy Policy\n\nDefault privacy content.",
		TermsMarkdown:         "# Terms of Use\n\nDefault terms content.",
		PoweredByText:         "Powered by Ty2Shorten",
		IsSetupCompleted:      true,
	}).Error; err != nil {
		t.Fatalf("seed settings: %v", err)
	}
}

func seedShortLink(t *testing.T, db *gorm.DB, slug, destination string, active bool) {
	t.Helper()

	if err := db.Create(&models.ShortLink{
		Slug:        slug,
		Title:       slug,
		Destination: destination,
		IsActive:    active,
	}).Error; err != nil {
		t.Fatalf("seed short link: %v", err)
	}
}

func assertStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, want, recorder.Body.String())
	}
}

func assertHeader(t *testing.T, recorder *httptest.ResponseRecorder, header, want string) {
	t.Helper()
	if got := recorder.Header().Get(header); got != want {
		t.Fatalf("%s = %q, want %q", header, got, want)
	}
}
