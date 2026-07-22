package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
)

func TestFooterSettingsUpdate(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings/footer")
	form := url.Values{
		"csrf_token":         {token},
		"footer_enabled":     {"1"},
		"footer_brand_text":  {"TY2 Footer"},
		"footer_description": {"Public trust links."},
		"copyright_text":     {"Copyright 2026 TY2"},
		"support_text":       {"Need help?"},
		"footer_address":     {"Yangon"},
		"powered_by_text":    {"Should not be saved"},
	}
	recorder := app.do(t, http.MethodPost, "/admin/settings/footer", form, nil)
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin/settings/footer?saved=1")

	var settings models.AppSetting
	if err := app.db.First(&settings).Error; err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.FooterBrandText != "TY2 Footer" || settings.PoweredByText == "Should not be saved" {
		t.Fatalf("footer settings were not saved: %+v", settings)
	}

	home := app.do(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, home, http.StatusOK)
	if !strings.Contains(home.Body.String(), "TY2 Footer") || !strings.Contains(home.Body.String(), `href="/terms"`) || !strings.Contains(home.Body.String(), `href="https://phyowaiyan.com/"`) {
		t.Fatalf("updated footer not rendered: %s", home.Body.String())
	}
}

func TestFooterSettingsRemovesProtectedFields(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	recorder := app.do(t, http.MethodGet, "/admin/settings/footer", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		"Footer visibility",
		"Public text content",
		"Visitor help details",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("footer settings missing marker %q: %s", want, body)
		}
	}
	for _, removed := range []string{
		"Privacy Policy URL",
		"Terms URL",
		"privacy_policy_url",
		"terms_url",
		"Powered-by text",
		"powered_by_text",
		"Protected credit",
		"Powered-by link",
		"Powered-by Phyo Wai Yan",
	} {
		if strings.Contains(body, removed) {
			t.Fatalf("footer settings still expose protected field %q: %s", removed, body)
		}
	}
}

func TestFooterSettingsCanDisableFooter(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings/footer")
	recorder := app.do(t, http.MethodPost, "/admin/settings/footer", url.Values{
		"csrf_token":        {token},
		"footer_brand_text": {"Hidden footer"},
	}, nil)
	assertStatus(t, recorder, http.StatusSeeOther)

	home := app.do(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, home, http.StatusOK)
	if strings.Contains(home.Body.String(), `class="public-footer"`) {
		t.Fatalf("footer rendered after disable: %s", home.Body.String())
	}
}
