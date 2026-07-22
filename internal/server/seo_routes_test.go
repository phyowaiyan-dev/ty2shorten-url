package server_test

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
)

func TestRobotsIndexingEnabledWithSitemap(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/robots.txt", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	if got := recorder.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := recorder.Body.String()
	for _, want := range []string{"User-agent: *", "Allow: /", "Disallow: /admin/", "Disallow: /setup/", "Sitemap: http://localhost:8722/sitemap.xml"} {
		if !strings.Contains(body, want) {
			t.Fatalf("robots.txt missing %q: %s", want, body)
		}
	}
}

func TestRobotsIndexingDisabled(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	if err := app.db.Model(&models.AppSetting{}).Where("1 = 1").Updates(map[string]any{
		"search_engine_indexing_enabled": false,
	}).Error; err != nil {
		t.Fatalf("disable indexing: %v", err)
	}

	recorder := app.do(t, http.MethodGet, "/robots.txt", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	if body := recorder.Body.String(); !strings.Contains(body, "Disallow: /") || strings.Contains(body, "Sitemap:") {
		t.Fatalf("unexpected robots body: %s", body)
	}
}

func TestSitemapXML(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/sitemap.xml", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	if got := recorder.Header().Get("Content-Type"); got != "application/xml; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}

	var parsed struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(recorder.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("sitemap XML did not parse: %v\n%s", err, recorder.Body.String())
	}
	if len(parsed.URLs) != 1 || parsed.URLs[0].Loc != "http://localhost:8722/" {
		t.Fatalf("unexpected sitemap URLs: %+v", parsed.URLs)
	}
}

func TestHomeMetadata(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)

	recorder := app.do(t, http.MethodGet, "/", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		`<meta name="description" content="Short links and app redirects.">`,
		`<link rel="canonical" href="http://localhost:8722/">`,
		`<meta property="og:title" content="TY2 Shorten">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`application/ld+json`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("home page missing metadata %q:\n%s", want, body)
		}
	}
}

func TestSEOSettingsPageUsesImageUploadCards(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)
	if err := app.db.Model(&models.AppSetting{}).Where("1 = 1").Updates(map[string]any{
		"open_graph_default_image_url": "/media/open-graph.png",
		"twitter_default_image_url":    "/media/twitter.png",
		"organization_logo_url":        "/media/org-logo.png",
	}).Error; err != nil {
		t.Fatalf("update SEO image URLs: %v", err)
	}

	recorder := app.do(t, http.MethodGet, "/admin/settings/seo", nil, nil)
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{
		"Search visibility",
		"Uploaded SEO images",
		"Reupload Open Graph image",
		"Reupload Twitter/X image",
		"Reupload organization logo",
		`<img src="/media/open-graph.png"`,
		`name="open_graph_default_image_url" value="/media/open-graph.png"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected SEO UI marker %q, got: %s", want, body)
		}
	}
	for _, removed := range []string{"Open Graph image URL", "Twitter/X image URL", "Organization logo URL"} {
		if strings.Contains(body, removed) {
			t.Fatalf("expected visible URL field %q to be removed, got: %s", removed, body)
		}
	}
}

func TestSEOImageUploadsAreSaved(t *testing.T) {
	app := newRouteTestApp(t)
	seedConfiguredApp(t, app.db)
	app.login(t)

	token := app.csrfToken(t, "/admin/settings/seo")
	recorder := app.doMultipart(t, http.MethodPost, "/admin/settings/seo", map[string]string{
		"csrf_token": token,
	}, map[string]uploadFile{
		"open_graph_default_image_file": {filename: "open-graph.png", content: tinyPNG},
		"twitter_default_image_file":    {filename: "twitter.png", content: tinyPNG},
		"organization_logo_file":        {filename: "organization.png", content: tinyPNG},
	})
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/admin/settings/seo?saved=1")

	var settings models.AppSetting
	if err := app.db.First(&settings).Error; err != nil {
		t.Fatalf("load settings: %v", err)
	}
	for name, value := range map[string]string{
		"OpenGraphDefaultImageURL": settings.OpenGraphDefaultImageURL,
		"TwitterDefaultImageURL":   settings.TwitterDefaultImageURL,
		"OrganizationLogoURL":      settings.OrganizationLogoURL,
	} {
		if !strings.HasPrefix(value, "/media/") || !strings.HasSuffix(value, ".png") {
			t.Fatalf("%s = %q, want uploaded media URL", name, value)
		}
	}
}
