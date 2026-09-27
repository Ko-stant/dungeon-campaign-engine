package app

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func get(t *testing.T, client *http.Client, u string) (int, string) {
	t.Helper()
	resp, err := client.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func noRedirects() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func TestMapsPageListsBoards(t *testing.T) {
	srv := testServer(t)
	call(t, srv, http.MethodPost, "/api/boards", map[string]any{"name": "Winter Keep", "width": 24, "height": 30})

	code, body := get(t, srv.Client(), srv.URL+"/maps")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	for _, want := range []string{"Winter Keep", "24 × 30", `action="/maps"`} {
		if !strings.Contains(body, want) {
			t.Errorf("page is missing %q", want)
		}
	}
}

func TestCreateBoardFormRedirectsToEditor(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()

	resp, err := client.PostForm(srv.URL+"/maps", url.Values{"name": {"Crypt"}, "width": {"12"}, "height": {"9"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status %d, want 303", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if !strings.HasPrefix(location, "/maps/") || !strings.HasSuffix(location, "/edit") {
		t.Fatalf("redirect to %q", location)
	}

	code, body := get(t, client, srv.URL+location)
	if code != http.StatusOK || !strings.Contains(body, `data-board-id="`) || !strings.Contains(body, "/static/dist/mapEditor.js") {
		t.Fatalf("editor page: %d", code)
	}
}

func TestCreateBoardFormShowsErrors(t *testing.T) {
	srv := testServer(t)
	resp, err := noRedirects().PostForm(srv.URL+"/maps", url.Values{"name": {"Too big"}, "width": {"999"}, "height": {"9"}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "board size") {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
}

func TestEditorPageForUnknownBoardIs404(t *testing.T) {
	srv := testServer(t)
	if code, _ := get(t, srv.Client(), srv.URL+"/maps/01900000-0000-7000-8000-000000000000/edit"); code != http.StatusNotFound {
		t.Fatalf("status %d", code)
	}
}
