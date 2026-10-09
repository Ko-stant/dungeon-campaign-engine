package app

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

// The whole catalog comes from the database (make import-content); the server
// has no content files (New takes none).
func TestCatalogComesFromTheDatabase(t *testing.T) {
	var orcRow string
	srv := testServerWith(t, func(s *Server) {
		list, err := s.store.ListCustomMonsters(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, rec := range list {
			if rec.CatalogID == "orc" {
				orcRow = rec.ID
			}
		}
	})
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	client := noRedirects()

	_, data := c(http.MethodGet, "/api/catalog", nil)
	cat := decodeAny[content.Catalog](t, data)
	if len(cat.Furniture) != 1 || cat.Furniture[0].ID != "table" || cat.Furniture[0].Width != 2 {
		t.Errorf("furniture: %+v", cat.Furniture)
	}
	if len(cat.Traps) != 1 || cat.Traps[0].ID != "long_pit" || cat.Traps[0].Height != 2 {
		t.Errorf("traps: %+v", cat.Traps)
	}
	if len(cat.Monsters) != 1 || cat.Monsters[0].ID != "orc" || cat.Monsters[0].Custom || cat.Monsters[0].Body != 1 {
		t.Errorf("monsters: %+v", cat.Monsters)
	}
	if len(cat.Heroes) != 1 || cat.Heroes[0].ID != "elf" {
		t.Errorf("heroes: %+v", cat.Heroes)
	}

	// The monsters page lists the base game's read-only; the forms refuse them.
	if orcRow == "" {
		t.Fatal("the orc was not imported")
	}
	_, page := get(t, client, srv.URL+"/monsters")
	if !strings.Contains(page, "Base game monsters") || !strings.Contains(page, "Orc") {
		t.Error("the monsters page should list the base game's monsters")
	}
	if strings.Contains(page, `action="/monsters/`+orcRow+`"`) || strings.Contains(page, "/monsters/"+orcRow+"/delete") {
		t.Error("a base monster has no edit or delete form")
	}
	for _, path := range []string{"/monsters/" + orcRow, "/monsters/" + orcRow + "/delete"} {
		if resp, _ := postForm(t, client, srv.URL+path, url.Values{"name": {"Big Orc"}, "width": {"1"}, "height": {"1"}}); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("POST %s on a base monster: %d, want 400", path, resp.StatusCode)
		}
	}
	if _, data = c(http.MethodGet, "/api/catalog", nil); decodeAny[content.Catalog](t, data).Monsters[0].Name != "Orc" {
		t.Error("the base monster should be unchanged")
	}
}
