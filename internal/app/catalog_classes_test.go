package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

// The base game's classes come from the database (imported from
// content/heroes); the content files only fill in a class not imported yet.
func TestBaseClassesComeFromTheDatabase(t *testing.T) {
	var elfRow string
	srv := testServerWith(t, func(s *Server) {
		ctx := context.Background()
		// The test catalog's files have an Elf with 6 Body; the database's has 7.
		for id, doc := range map[string]string{
			"elf":   `{"name":"Elf","body":7,"mind":4,"attack":2,"defense":2,"movementDice":2}`,
			"dwarf": `{"name":"Dwarf","body":7,"mind":3,"attack":2,"defense":2,"movementDice":2}`,
		} {
			name := map[string]string{"elf": "Elf", "dwarf": "Dwarf"}[id]
			if _, err := s.store.UpsertCatalogHeroClass(ctx, id, name, json.RawMessage(doc)); err != nil {
				t.Fatal(err)
			}
		}
		list, err := s.store.ListCustomHeroClasses(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, rec := range list {
			if rec.CatalogID == "elf" {
				elfRow = rec.ID
			}
		}
	})
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }

	heroes := func() map[string]content.HeroDef {
		t.Helper()
		_, data := c(http.MethodGet, "/api/catalog", nil)
		out := map[string]content.HeroDef{}
		for _, h := range decodeAny[content.Catalog](t, data).Heroes {
			if _, dup := out[h.ID]; dup {
				t.Errorf("class %s is in the catalog twice", h.ID)
			}
			out[h.ID] = h
		}
		return out
	}
	got := heroes()
	if elf := got["elf"]; elf.Body != 7 || elf.Custom || elf.Name != "Elf" || elf.Attack != 2 {
		t.Errorf("the database's Elf should win over the file's: %+v", elf)
	}
	if dwarf, ok := got["dwarf"]; !ok || dwarf.Body != 7 {
		t.Errorf("a class only in the database: %+v", dwarf)
	}

	// /classes lists them read-only, with Deactivate.
	_, page := get(t, client, srv.URL+"/classes")
	if !strings.Contains(page, "base game") || !strings.Contains(page, "Attack 2 dice") {
		t.Error("the classes page should list the base game's classes with their stats")
	}
	if !strings.Contains(page, `action="/classes/`+elfRow+`/deactivate"`) {
		t.Fatal("no Deactivate for a base class")
	}
	if regexp.MustCompile(`href="/classes/` + elfRow + `"`).MatchString(page) {
		t.Error("a base class has no edit page link")
	}
	if resp, _ := postForm(t, client, srv.URL+"/classes/"+elfRow, url.Values{"name": {"Brute"}}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("editing a base class: %d, want 400", resp.StatusCode)
	}
	if code, _ := get(t, client, srv.URL+"/classes/"+elfRow); code != http.StatusSeeOther {
		t.Errorf("a base class's page: %d, want a redirect to /classes", code)
	}

	// Deactivated, it leaves the add-hero picker, and the file's copy does not bring it back.
	resp, _ := postForm(t, client, srv.URL+"/campaigns", url.Values{"name": {"Winter"}})
	campaignURL := resp.Header.Get("Location")
	if _, page = get(t, client, srv.URL+campaignURL); !strings.Contains(page, `<option value="elf">`) {
		t.Fatal("the Elf should be offered for new heroes")
	}
	if resp, _ := postForm(t, client, srv.URL+"/classes/"+elfRow+"/deactivate", nil); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("deactivate: %d", resp.StatusCode)
	}
	if _, page = get(t, client, srv.URL+campaignURL); strings.Contains(page, `<option value="elf">`) {
		t.Error("a deactivated base class should not be offered")
	}
	if elf := heroes()["elf"]; !elf.Inactive || elf.Body != 7 {
		t.Errorf("the catalog keeps the deactivated Elf: %+v", elf)
	}
}
