package app

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

func postForm(t *testing.T, client *http.Client, u string, form url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := client.PostForm(u, form)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

func TestCampaignPagesFlow(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	questID := setupQuest(t, urlServer{srv.URL}, func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) })

	code, body := get(t, client, srv.URL+"/campaigns")
	if code != http.StatusOK || !strings.Contains(body, `action="/campaigns"`) {
		t.Fatalf("campaigns page: %d", code)
	}

	resp, _ := postForm(t, client, srv.URL+"/campaigns", url.Values{"name": {"Winter Campaign"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/campaigns/") {
		t.Fatalf("create campaign: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	campaignURL := resp.Header.Get("Location")

	resp, _ = postForm(t, client, srv.URL+campaignURL+"/heroes", url.Values{"name": {"Faelyn"}, "player": {"Jo"}, "class": {"elf"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("add hero: %d", resp.StatusCode)
	}
	resp, body = postForm(t, client, srv.URL+campaignURL+"/heroes", url.Values{"name": {"Nope"}, "class": {"paladin"}})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "paladin") {
		t.Fatalf("bad hero: %d", resp.StatusCode)
	}

	code, body = get(t, client, srv.URL+campaignURL)
	for _, want := range []string{"Winter Campaign", "Faelyn", "Jo", "The Trial"} {
		if !strings.Contains(body, want) {
			t.Errorf("campaign page (%d) is missing %q", code, want)
		}
	}

	resp, _ = postForm(t, client, srv.URL+campaignURL+"/sessions", url.Values{"questId": {questID}, "name": {"Night one"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/play/") {
		t.Fatalf("start session: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	playURL := resp.Header.Get("Location")

	code, body = get(t, client, srv.URL+playURL)
	if code != http.StatusOK || !strings.Contains(body, `data-session-id="`) || !strings.Contains(body, "/static/dist/tracker.js") {
		t.Fatalf("play page: %d", code)
	}

	_, body = get(t, client, srv.URL+campaignURL)
	if !strings.Contains(body, playURL) {
		t.Fatal("campaign page should link to the active session")
	}
	_, body = get(t, client, srv.URL+"/campaigns")
	if !strings.Contains(body, playURL) {
		t.Fatal("campaigns page should list the active session")
	}

	// Removing a hero.
	_, body = get(t, client, srv.URL+campaignURL)
	start := strings.Index(body, campaignURL+"/heroes/")
	if start < 0 {
		t.Fatal("no remove-hero form found")
	}
	action := body[start : start+strings.Index(body[start:], `"`)]
	resp, _ = postForm(t, client, srv.URL+action, url.Values{})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("remove hero: %d", resp.StatusCode)
	}
	_, body = get(t, client, srv.URL+campaignURL)
	if strings.Contains(body, "Faelyn") {
		t.Fatal("hero should be removed from the campaign page")
	}
}

func TestPlayPageForUnknownSessionIs404(t *testing.T) {
	srv := testServer(t)
	if code, _ := get(t, srv.Client(), srv.URL+"/play/01900000-0000-7000-8000-000000000000"); code != http.StatusNotFound {
		t.Fatalf("status %d", code)
	}
	if code, _ := get(t, srv.Client(), srv.URL+"/campaigns/01900000-0000-7000-8000-000000000000"); code != http.StatusNotFound {
		t.Fatalf("status %d", code)
	}
}

func TestCampaignPageMarksCustomTypes(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	if resp, _ := postForm(t, client, srv.URL+"/classes", rogueForm()); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create class: %d", resp.StatusCode)
	}
	if resp, _ := postForm(t, client, srv.URL+"/monsters", ogreForm()); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create monster: %d", resp.StatusCode)
	}
	rogue := customClasses(t, c)[0].ID
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "C"})
	camp := decodeAny[CampaignResponse](t, data)
	if code, data := c(http.MethodPut, "/api/campaigns/"+camp.ID, map[string]any{"name": "C", "heroes": []tracker.CampaignHero{
		{Name: "Vex", Class: "elf", Equipment: "Old sword"},
		{Name: "Bram", Class: rogue, Items: []tracker.Item{
			{Name: "Dirk", Kind: "weapon", Equipped: true, ItemStats: tracker.ItemStats{Damage: 2}},
			{Name: "Sword", Kind: "weapon", Equipped: true, ItemStats: tracker.ItemStats{Damage: 3}},
			{Name: "Rope"},
		}},
	}}); code != http.StatusOK {
		t.Fatalf("put heroes: %d %s", code, data)
	}

	_, body := get(t, client, srv.URL+"/campaigns/"+camp.ID)
	// Setup pages tell custom classes and monsters from the base game's.
	for _, want := range []string{">Rogue (custom)</option>", ">Elf</option>", ">Cave Ogre (custom)</option>", ">Orc</option>", `title="custom class"`} {
		if !strings.Contains(body, want) {
			t.Errorf("campaign page is missing %q", want)
		}
	}
	// The heroes table lists equipped items (old free-text equipment when there are none).
	for _, want := range []string{`title="Dirk, Sword"`, `title="Old sword"`, "<th>Equipped</th>"} {
		if !strings.Contains(body, want) {
			t.Errorf("campaign page is missing %q", want)
		}
	}
	if strings.Contains(body, "Dirk, Sword, Rope") {
		t.Error("unequipped items are not listed as equipped")
	}
}
