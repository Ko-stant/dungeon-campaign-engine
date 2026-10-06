package app

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/auth"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// approvalServer runs the app as hosted: dev sign-in, members approved by
// the GM (an admin, dev:gm), and board art in an assets folder.
func approvalServer(t *testing.T) string {
	t.Helper()
	assets := t.TempDir()
	if err := os.WriteFile(filepath.Join(assets, "tile.png"), []byte("art"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := testServerWith(t, func(s *Server) {
		s.SetAuth(auth.Config{Mode: auth.ModeDev, PublicURL: "http://localhost", Admins: []store.IdentityKey{{Provider: "dev", Subject: "gm"}}})
		s.SetAssetsDir(assets)
	})
	return srv.URL
}

func TestMembersWaitForTheGM(t *testing.T) {
	base := approvalServer(t)
	gm, sam, anon := newBrowser(t, base), newBrowser(t, base), newBrowser(t, base)
	gm.signInAs("GM")
	sam.signInAs("Sam")

	// Nobody signed out, and nobody waiting, sees anything: the board art included.
	if resp, _ := anon.get("/assets/tile.png"); resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/login") {
		t.Errorf("art when signed out: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	for _, path := range []string{"/campaigns", "/lobby", "/assets/tile.png"} {
		if resp, _ := sam.get(path); resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/waiting" {
			t.Errorf("waiting, %s: %d %s", path, resp.StatusCode, resp.Header.Get("Location"))
		}
	}
	if resp, body := sam.get("/api/boards"); resp.StatusCode != http.StatusForbidden || !strings.Contains(body, "GM") {
		t.Errorf("waiting, an API call: %d %s", resp.StatusCode, body)
	}
	if resp, body := sam.get("/waiting"); resp.StatusCode != http.StatusOK || !strings.Contains(body, "Waiting for the GM") || !strings.Contains(body, "Sam") {
		t.Errorf("the waiting page: %d %s", resp.StatusCode, body)
	}
	if resp, _ := sam.get("/members"); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("the members page while waiting: %d", resp.StatusCode)
	}

	// The GM (an admin, always a member) sees Sam waiting and lets them in.
	resp, page := gm.get("/members")
	if resp.StatusCode != http.StatusOK || !strings.Contains(page, "Sam") || !strings.Contains(page, "Waiting") {
		t.Fatalf("members page: %d %s", resp.StatusCode, page)
	}
	if _, nav := gm.get("/campaigns"); !strings.Contains(nav, `href="/members"`) || !strings.Contains(nav, "(1)") {
		t.Errorf("the GM's nav shows who waits: %s", nav)
	}
	samID := userIDIn(t, page, "Sam")
	if resp, _ := gm.postForm("/members/"+samID, url.Values{"status": {"member"}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("approve: %d", resp.StatusCode)
	}
	if resp, _ := sam.get("/campaigns"); resp.StatusCode != http.StatusOK {
		t.Errorf("a member: %d", resp.StatusCode)
	}
	if resp, body := sam.get("/assets/tile.png"); resp.StatusCode != http.StatusOK || body != "art" {
		t.Errorf("a member sees the art: %d %q", resp.StatusCode, body)
	}
	if resp, _ := sam.get("/waiting"); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("a member has nothing to wait for: %d", resp.StatusCode)
	}
	// Members are not admins.
	if resp, _ := sam.get("/members"); resp.StatusCode != http.StatusForbidden {
		t.Errorf("a member on the members page: %d", resp.StatusCode)
	}
	if resp, _ := sam.postForm("/members/"+samID, url.Values{"status": {"member"}}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("a member approving: %d", resp.StatusCode)
	}

	// Refused: back outside, told so.
	if resp, _ := gm.postForm("/members/"+samID, url.Values{"status": {"refused"}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("refuse: %d", resp.StatusCode)
	}
	if resp, _ := sam.get("/campaigns"); resp.Header.Get("Location") != "/waiting" {
		t.Errorf("refused: %d", resp.StatusCode)
	}
	if _, body := sam.get("/waiting"); !strings.Contains(body, "not let this account in") {
		t.Errorf("the refused page: %s", body)
	}
	if resp, _ := gm.postForm("/members/"+samID, url.Values{"status": {"boss"}}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("an unknown status: %d", resp.StatusCode)
	}
}

func TestTheArtNeedsNoSignInAtTheTable(t *testing.T) {
	assets := t.TempDir()
	if err := os.WriteFile(filepath.Join(assets, "tile.png"), []byte("art"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := testServerWith(t, func(s *Server) { s.SetAssetsDir(assets) })
	if resp, body := newBrowser(t, srv.URL).get("/assets/tile.png"); resp.StatusCode != http.StatusOK || body != "art" {
		t.Errorf("table mode: %d %q", resp.StatusCode, body)
	}
}

// userIDIn finds the id in the members page's form for the named user.
func userIDIn(t *testing.T, page, name string) string {
	t.Helper()
	i := strings.Index(page, name)
	j := strings.Index(page[i:], `action="/members/`)
	if i < 0 || j < 0 {
		t.Fatalf("no form for %s in %s", name, page)
	}
	rest := page[i+j+len(`action="/members/`):]
	return rest[:strings.Index(rest, `"`)]
}
