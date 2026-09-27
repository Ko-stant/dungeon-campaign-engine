package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNoCacheSetsRevalidateHeader(t *testing.T) {
	h := NoCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/app.css", nil))

	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want %q", got, "no-cache")
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want the wrapped handler's output", rec.Body.String())
	}
}

func TestNoCacheKeepsConditionalRequestsCheap(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := NoCache(http.FileServer(http.Dir(dir)))

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/app.css", nil))
	lastModified := first.Header().Get("Last-Modified")
	if first.Code != http.StatusOK || lastModified == "" {
		t.Fatalf("first request: code %d, Last-Modified %q", first.Code, lastModified)
	}

	req := httptest.NewRequest(http.MethodGet, "/app.css", nil)
	req.Header.Set("If-Modified-Since", lastModified)
	second := httptest.NewRecorder()
	h.ServeHTTP(second, req)
	if second.Code != http.StatusNotModified {
		t.Fatalf("revalidation code = %d, want 304", second.Code)
	}
}
