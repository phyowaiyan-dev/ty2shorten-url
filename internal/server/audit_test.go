package server_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
)

func TestSettingsUpdateCreatesAuditLog(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings")
	form := url.Values{
		"csrf_token":       {token},
		"site_name":        {"Audited Site"},
		"site_description": {"Audited redirect page for customers opening app links on mobile devices."},
		"android_url":      {"https://play.google.com/store/apps/details?id=com.audit"},
		"apple_url":        {"https://apps.apple.com/app/audit/id456"},
		"public_base_url":  {"http://localhost:8722"},
	}
	assertStatus(t, app.do(t, http.MethodPost, "/admin/settings", form, nil), http.StatusSeeOther)

	var log models.AuditLog
	if err := app.db.Where("action = ?", "settings.updated").First(&log).Error; err != nil {
		t.Fatalf("audit log not found: %v", err)
	}
	if log.Summary == "" || strings.Contains(log.MetadataJSON, "password") {
		t.Fatalf("audit log summary/metadata unsafe: %+v", log)
	}
}

func TestAuditLogUIReadOnly(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	recorder := app.do(t, http.MethodGet, "/admin/audit-logs", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{"Audit Logs", "Date range", "Activity timeline", `type="date"`, "Apply range"} {
		if !strings.Contains(body, want) {
			t.Fatalf("audit list missing marker %q: %s", want, body)
		}
	}
	for _, removed := range []string{"Summary search", "Resource type", `name="action"`, `name="resource_type"`, `name="q"`} {
		if strings.Contains(body, removed) {
			t.Fatalf("audit list still exposes old filter %q: %s", removed, body)
		}
	}

	recorder = app.do(t, http.MethodPost, "/admin/audit-logs", nil, nil)
	assertStatus(t, recorder, http.StatusNotFound)
}

func TestAuditLogFilteringAndDetail(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	seoLog := models.AuditLog{Action: "seo.updated", ResourceType: "settings", Summary: "SEO updated", MetadataJSON: `{"safe":"value"}`, CreatedAt: time.Date(2026, 7, 20, 10, 0, 0, 0, time.Local)}
	footerLog := models.AuditLog{Action: "footer.updated", ResourceType: "settings", Summary: "Footer updated", MetadataJSON: `{"secret":"[REDACTED]"}`, CreatedAt: time.Date(2026, 7, 22, 13, 55, 0, 0, time.Local)}
	if err := app.db.Create(&seoLog).Error; err != nil {
		t.Fatalf("seed seo log: %v", err)
	}
	if err := app.db.Create(&footerLog).Error; err != nil {
		t.Fatalf("seed footer log: %v", err)
	}

	recorder := app.do(t, http.MethodGet, "/admin/audit-logs?from=2026-07-22&to=2026-07-22", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	if !strings.Contains(body, "Footer updated") || strings.Contains(body, "SEO updated") {
		t.Fatalf("audit filter did not constrain date range: %s", body)
	}
	for _, want := range []string{"Jul 22, 2026 1:55 PM", "Footer updated", "Settings"} {
		if !strings.Contains(body, want) {
			t.Fatalf("audit list missing readable marker %q: %s", want, body)
		}
	}

	recorder = app.do(t, http.MethodGet, "/admin/audit-logs/"+strconv.FormatUint(uint64(footerLog.ID), 10), nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body = recorder.Body.String()
	if !strings.Contains(body, "Audit Log Detail") || !strings.Contains(body, "Footer updated") || !strings.Contains(body, "footer.updated") || !strings.Contains(body, "[REDACTED]") {
		t.Fatalf("audit detail did not render expected entry: %s", body)
	}
}
