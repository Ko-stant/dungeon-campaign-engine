package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// These tests run from cmd/server, so point the static roots at the repo.
func testRoutes(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("DATABASE_URL", "") // no database: app routes explain how to start it
	mux, cleanup := newMux(routesConfig{staticDir: "../../internal/web/static", assetsDir: "../../assets", contentDir: "../../content"})
	t.Cleanup(cleanup)
	return mux
}

func TestRootRedirectsToCampaigns(t *testing.T) {
	rec := httptest.NewRecorder()
	testRoutes(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/campaigns" {
		t.Fatalf("GET / = %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestUnknownPathsAre404(t *testing.T) {
	rec := httptest.NewRecorder()
	testRoutes(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/lobby", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /lobby = %d, want 404 now that the multiplayer lobby is gone", rec.Code)
	}
}

func TestAppRoutesExplainMissingDatabase(t *testing.T) {
	for _, path := range []string{"/campaigns", "/maps", "/play/x", "/api/catalog"} {
		rec := httptest.NewRecorder()
		testRoutes(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("GET %s = %d, want 503 without a database", path, rec.Code)
		}
	}
}

func TestStaticFilesAreServedWithoutCaching(t *testing.T) {
	rec := httptest.NewRecorder()
	testRoutes(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/styles/index.css", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("static: %d cache=%q", rec.Code, rec.Header().Get("Cache-Control"))
	}
}
