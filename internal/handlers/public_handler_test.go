package handlers_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/config"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/database"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/server"
)

func TestHealthOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "health.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(db); err != nil {
			t.Fatalf("database.Close returned error: %v", err)
		}
	})

	router, err := server.NewRouter(server.Dependencies{
		Config: config.Config{
			AppEnv:        "test",
			AppHost:       "127.0.0.1",
			AppPort:       8722,
			DatabasePath:  dbPath,
			SessionSecret: "test-secret",
			BaseURL:       "http://localhost:8722",
			Version:       "test",
			Commit:        "abc123",
			BuildTime:     "2026-07-22T00:00:00Z",
		},
		DB:     db,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("server.NewRouter returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	want := `{"build_time":"2026-07-22T00:00:00Z","commit":"abc123","service":"ty2shorten-url","status":"ok","version":"test"}`
	if recorder.Body.String() != want {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), want)
	}
}
