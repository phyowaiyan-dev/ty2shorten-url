package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
)

func TestPublicFooterEnabled(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{"Audit-ready links", "Privacy Policy", "Terms", "Powered-by Phyo Wai Yan", `href="https://phyowaiyan.com/"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("footer missing %q:\n%s", want, body)
		}
	}
}

func TestPublicFooterDisabled(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	if err := app.db.Model(&models.AppSetting{}).Where("1 = 1").Update("footer_enabled", false).Error; err != nil {
		t.Fatalf("disable footer: %v", err)
	}

	recorder := app.do(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	if strings.Contains(recorder.Body.String(), "public-footer") {
		t.Fatalf("footer rendered when disabled:\n%s", recorder.Body.String())
	}
}
