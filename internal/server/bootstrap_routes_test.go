package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/config"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/server"
)

func TestBootstrapModeRoutes(t *testing.T) {
	router := newBootstrapRouter(t)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	assertStatus(t, recorder, http.StatusSeeOther)
	assertHeader(t, recorder, "Location", "/setup/database")

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/setup/database", nil))
	assertStatus(t, recorder, http.StatusOK)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	assertStatus(t, recorder, http.StatusOK)
}

func TestDatabaseSetupUIDriverSections(t *testing.T) {
	router := newBootstrapRouter(t)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/setup/database", nil))
	assertStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{`<select name="driver"`, `value="sqlite" selected`, `data-driver-section="sqlite"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected SQLite setup UI marker %q, got: %s", want, body)
		}
	}
	if !strings.Contains(body, `data-driver-section="mysql" hidden`) {
		t.Fatalf("MySQL section should be hidden for default SQLite setup, got: %s", body)
	}

	csrf := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindStringSubmatch(body)
	if len(csrf) != 2 {
		t.Fatalf("csrf token not found in database setup page: %s", body)
	}
	form := url.Values{
		"csrf_token": {csrf[1]},
		"driver":     {"mysql"},
	}
	request := httptest.NewRequest(http.MethodPost, "/setup/database/test", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range recorder.Result().Cookies() {
		request.AddCookie(cookie)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assertStatus(t, recorder, http.StatusUnprocessableEntity)
	body = recorder.Body.String()
	if !strings.Contains(body, `data-driver-section="mysql"`) || strings.Contains(body, `data-driver-section="mysql" hidden`) {
		t.Fatalf("MySQL section should be visible after selecting MySQL, got: %s", body)
	}
	if !strings.Contains(body, "Use MySQL when you already manage a database server") {
		t.Fatalf("expected MySQL guidance text, got: %s", body)
	}
}

func newBootstrapRouter(t *testing.T) http.Handler {
	t.Helper()
	router, err := server.NewRouter(server.Dependencies{
		Config: config.Config{
			AppEnv:         "test",
			AppHost:        "127.0.0.1",
			AppPort:        8722,
			DatabasePath:   filepath.Join(t.TempDir(), "app.db"),
			AppConfigPath:  filepath.Join(t.TempDir(), "config.json"),
			SessionSecret:  "test-secret",
			BaseURL:        "http://localhost:8722",
			TrustedProxies: []string{"127.0.0.1"},
			Version:        "test",
			Commit:         "abc",
			BuildTime:      "now",
		},
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		BootstrapMode: true,
	})
	if err != nil {
		t.Fatalf("NewRouter returned error: %v", err)
	}
	return router
}
