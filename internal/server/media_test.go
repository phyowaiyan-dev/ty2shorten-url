package server_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
)

var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func TestBrandingUploadStoresAndServesLogo(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings/branding")
	recorder := app.doMultipart(t, http.MethodPost, "/admin/settings/branding", map[string]string{
		"csrf_token": token,
	}, map[string]uploadFile{
		"logo_file": {filename: "logo.png", content: tinyPNG},
	})
	assertStatus(t, recorder, http.StatusSeeOther)

	var settings models.AppSetting
	if err := app.db.First(&settings).Error; err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if !strings.HasPrefix(settings.LogoURL, "/media/") || !strings.HasSuffix(settings.LogoURL, ".png") {
		t.Fatalf("LogoURL = %q, want safe public media URL", settings.LogoURL)
	}

	media := app.do(t, http.MethodGet, settings.LogoURL, nil, nil)
	assertStatus(t, media, http.StatusOK)
	if got := media.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}
}

func TestBrandingUploadRejectsFakeImage(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings/branding")
	recorder := app.doMultipart(t, http.MethodPost, "/admin/settings/branding", map[string]string{
		"csrf_token": token,
	}, map[string]uploadFile{
		"logo_file": {filename: "logo.png", content: []byte("not really a png")},
	})
	assertStatus(t, recorder, http.StatusUnprocessableEntity)
	if !strings.Contains(recorder.Body.String(), "Upload a PNG, JPEG, WebP, or ICO file.") {
		t.Fatalf("expected MIME validation error, got: %s", recorder.Body.String())
	}
}

func TestBrandingUploadRejectsOversizedFavicon(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	tooLarge := append([]byte{}, tinyPNG...)
	tooLarge = append(tooLarge, bytes.Repeat([]byte{0}, 512*1024)...)
	token := app.csrfToken(t, "/admin/settings/branding")
	recorder := app.doMultipart(t, http.MethodPost, "/admin/settings/branding", map[string]string{
		"csrf_token": token,
	}, map[string]uploadFile{
		"favicon_file": {filename: "favicon.png", content: tooLarge},
	})
	assertStatus(t, recorder, http.StatusUnprocessableEntity)
	if !strings.Contains(recorder.Body.String(), "favicon file is too large") {
		t.Fatalf("expected size validation error, got: %s", recorder.Body.String())
	}
}

func TestBrandingSettingsPageShowsUploadCards(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)
	if err := app.db.Model(&models.AppSetting{}).Where("1 = 1").Updates(map[string]any{
		"logo_url":    "/media/logo.png",
		"favicon_url": "/media/favicon.png",
	}).Error; err != nil {
		t.Fatalf("update branding URLs: %v", err)
	}

	recorder := app.do(t, http.MethodGet, "/admin/settings/branding", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		"Brand voice",
		"Uploaded brand files",
		"Reupload logo",
		"Reupload favicon",
		`<img src="/media/logo.png"`,
		`name="logo_url" value="/media/logo.png"`,
		`/static/js/settings-form.js`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected branding UI marker %q, got: %s", want, body)
		}
	}
	for _, removed := range []string{"Logo URL", "Dark logo URL", "Favicon URL", "Apple touch icon URL", "Social image URL"} {
		if strings.Contains(body, removed) {
			t.Fatalf("expected visible URL field %q to be removed, got: %s", removed, body)
		}
	}
}

func TestAccountAvatarUploadOverMediaLimitShowsUploadError(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	tooLarge := append([]byte{}, tinyPNG...)
	tooLarge = append(tooLarge, bytes.Repeat([]byte{0}, 7*1024*1024)...)
	token := app.csrfToken(t, "/admin/account")
	recorder := app.doMultipart(t, http.MethodPost, "/admin/account/profile", map[string]string{
		"csrf_token": token,
		"name":       "Admin User",
		"email":      "admin@example.com",
	}, map[string]uploadFile{
		"avatar_file": {filename: "avatar.png", content: tooLarge},
	})
	assertStatus(t, recorder, http.StatusUnprocessableEntity)
	if strings.Contains(recorder.Body.String(), "Invalid form token") {
		t.Fatalf("expected upload validation instead of CSRF error, got: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "avatar file is too large") {
		t.Fatalf("expected avatar size validation error, got: %s", recorder.Body.String())
	}
}

func TestBrandingURLsRenderOnPublicPages(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	if err := app.db.Model(&models.AppSetting{}).Where("1 = 1").Updates(map[string]any{
		"logo_url":                 "/media/logo.png",
		"favicon_url":              "/media/favicon.png",
		"default_social_image_url": "/media/social.png",
	}).Error; err != nil {
		t.Fatalf("update branding URLs: %v", err)
	}

	recorder := app.do(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{`<link rel="icon" href="http://localhost:8722/media/favicon.png">`, `<img class="brand-logo" src="/media/logo.png"`, `property="og:image" content="http://localhost:8722/media/social.png"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in public page, got: %s", want, body)
		}
	}
}

func TestMissingMediaReturnsNotFound(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/media/missing.png", nil, nil)
	assertStatus(t, recorder, http.StatusNotFound)
}

type uploadFile struct {
	filename string
	content  []byte
}

func (a *routeTestApp) doMultipart(t *testing.T, method, target string, fields map[string]string, files map[string]uploadFile) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	for field, file := range files {
		part, err := writer.CreateFormFile(field, file.filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(file.content); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(method, target, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	for _, cookie := range a.cookies {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	a.router.ServeHTTP(recorder, request)
	a.storeCookies(recorder.Result().Cookies())
	return recorder
}

func TestAdminShellNavigationIsResponsive(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	recorder := app.do(t, http.MethodGet, "/admin", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		`data-sidebar-provider`,
		`data-sidebar="sidebar"`,
		`data-sidebar="inset"`,
		`data-sidebar="trigger"`,
		`data-sidebar-group-label`,
		`aria-current="page"`,
		`Platform`,
		`Settings`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected responsive admin shell marker %q, got: %s", want, body)
		}
	}
}
