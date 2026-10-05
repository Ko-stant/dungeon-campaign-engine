package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/auth"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// browser is one person's browser: its own cookies, redirects not followed.
type browser struct {
	t      *testing.T
	base   string
	client *http.Client
}

func newBrowser(t *testing.T, base string) *browser {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &browser{t: t, base: base, client: &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (b *browser) do(method, path string, body io.Reader, contentType string) (*http.Response, string) {
	b.t.Helper()
	req, err := http.NewRequest(method, b.base+path, body)
	if err != nil {
		b.t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	return resp, string(data)
}

func (b *browser) get(path string) (*http.Response, string) {
	return b.do(http.MethodGet, path, nil, "")
}

func (b *browser) postForm(path string, form url.Values) (*http.Response, string) {
	return b.do(http.MethodPost, path, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
}

func (b *browser) sendJSON(method, path string, v any) (*http.Response, string) {
	data, err := json.Marshal(v)
	if err != nil {
		b.t.Fatal(err)
	}
	return b.do(method, path, strings.NewReader(string(data)), "application/json")
}

// devServer runs the app with dev sign-in; admins as in AUTH_ADMINS.
func devServer(t *testing.T, admins ...store.IdentityKey) (*httptest.Server, *Server) {
	t.Helper()
	var app *Server
	srv := testServerWith(t, func(s *Server) {
		s.SetAuth(auth.Config{Mode: auth.ModeDev, PublicURL: "http://localhost", Admins: admins})
		app = s
	})
	return srv, app
}

func (b *browser) signInAs(name string) {
	b.t.Helper()
	resp, _ := b.postForm("/auth/dev", url.Values{"name": {name}, "next": {"/maps"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/maps" {
		b.t.Fatalf("dev sign-in as %s: %d %s", name, resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestEveryRoutePatternHasAnAccessRule(t *testing.T) {
	for pattern, want := range map[string][]ownedParam{
		"GET /api/boards":                            nil,
		"PUT /api/boards/{id}":                       {{"id", store.OwnedBoard, false}},
		"GET /maps/{id}/edit":                        {{"id", store.OwnedBoard, false}},
		"POST /api/sessions/{id}/commands":           {{"id", store.OwnedSession, false}},
		"GET /play/{id}/players":                     {{"id", store.OwnedSession, true}},
		"POST /api/sessions/{id}/seat-commands":      {{"id", store.OwnedSession, true}},
		"GET /api/sessions/{id}/stream":              {{"id", store.OwnedSession, false}},
		"POST /join/{id}/claim/{heroId}":             nil,
		"GET /lobby":                                 nil,
		"GET /audio/{campaign}/{file}":               {{"campaign", store.OwnedCampaign, false}},
		"POST /campaigns/{id}/chapters/{questId}/up": {{"id", store.OwnedCampaign, false}, {"questId", store.OwnedQuest, false}},
		"POST /classes/{id}/delete":                  {{"id", store.OwnedClass, false}},
	} {
		got, ok := accessRule(pattern)
		if !ok || len(got) != len(want) {
			t.Errorf("%s: %v %v", pattern, got, ok)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: %v, want %v", pattern, got, want)
			}
		}
	}
	if _, ok := accessRule("GET /api/secrets"); ok {
		t.Error("an unknown route must have no rule")
	}
	// Registering every real route through the guard must not panic.
	testServer(t)
}

func TestSafeNextStaysOnThisSite(t *testing.T) {
	for in, want := range map[string]string{"/maps": "/maps", "/play/1?x=2": "/play/1?x=2", "//evil.example": "/", "https://evil.example": "/", "/\\evil": "/", "": "/"} {
		if got := safeNext(in); got != want {
			t.Errorf("safeNext(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSignInIsRequiredWhenOn(t *testing.T) {
	srv, _ := devServer(t)
	anon := newBrowser(t, srv.URL)
	if resp, _ := anon.get("/api/boards"); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("an API read: %d", resp.StatusCode)
	}
	if resp, _ := anon.sendJSON(http.MethodPost, "/api/boards", map[string]any{"name": "B", "width": 2, "height": 2}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("an API write: %d", resp.StatusCode)
	}
	if resp, _ := anon.get("/campaigns"); resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login?next=%2Fcampaigns" {
		t.Errorf("a page: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if resp, body := anon.get("/login?next=/maps"); resp.StatusCode != http.StatusOK || !strings.Contains(body, "Dev sign-in") || !strings.Contains(body, `value="/maps"`) {
		t.Errorf("the sign-in page: %d", resp.StatusCode)
	}
	if resp, _ := anon.get("/api/me"); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("/api/me: %d", resp.StatusCode)
	}
}

func TestEachUserSeesAndChangesOnlyTheirOwn(t *testing.T) {
	srv, _ := devServer(t)
	gm, friend := newBrowser(t, srv.URL), newBrowser(t, srv.URL)
	gm.signInAs("GM")
	friend.signInAs("Friend")

	_, body := gm.get("/api/me")
	var me MeResponse
	if err := json.Unmarshal([]byte(body), &me); err != nil || me.DisplayName != "GM" || me.Admin {
		t.Fatalf("me %s", body)
	}
	if _, page := gm.get("/campaigns"); !strings.Contains(page, "Sign out") {
		t.Error("pages offer signing out")
	}

	c := func(method, path string, v any) (int, []byte) {
		resp, data := gm.sendJSON(method, path, v)
		return resp.StatusCode, []byte(data)
	}
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodGet, "/api/quests/"+questID, nil)
	boardID := decodeAny[QuestResponse](t, data).BoardID

	for _, path := range []string{"/api/boards/" + boardID, "/api/quests/" + questID, "/maps/" + boardID + "/edit"} {
		if resp, _ := friend.get(path); resp.StatusCode != http.StatusForbidden {
			t.Errorf("friend GET %s: %d", path, resp.StatusCode)
		}
	}
	if resp, _ := friend.sendJSON(http.MethodPut, "/api/boards/"+boardID, map[string]any{"name": "Mine now"}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("friend PUT board: %d", resp.StatusCode)
	}
	if _, list := friend.get("/api/boards"); strings.Contains(list, boardID) {
		t.Errorf("friend's board list: %s", list)
	}
	if _, list := gm.get("/api/boards"); !strings.Contains(list, boardID) {
		t.Errorf("GM's board list: %s", list)
	}

	// The friend's own campaign can't start a session on the GM's quest.
	resp, data2 := friend.sendJSON(http.MethodPost, "/api/campaigns", map[string]any{"name": "Mine"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("friend campaign: %d %s", resp.StatusCode, data2)
	}
	camp := decodeAny[CampaignResponse](t, []byte(data2))
	if resp, body := friend.sendJSON(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID, "name": "Heist"}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("a session on someone else's quest: %d %s", resp.StatusCode, body)
	}
	if resp, _ := gm.get("/api/campaigns/" + camp.ID); resp.StatusCode != http.StatusForbidden {
		t.Errorf("the GM opening the friend's campaign: %d", resp.StatusCode)
	}

	// Signing out ends the session.
	if resp, _ := gm.postForm("/auth/logout", nil); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("sign out: %d", resp.StatusCode)
	}
	if resp, _ := gm.get("/api/me"); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("after signing out: %d", resp.StatusCode)
	}
}

func TestAdminsReachEverything(t *testing.T) {
	srv, app := devServer(t, store.IdentityKey{Provider: "dev", Subject: "gm"})
	legacy, err := app.store.CreateBoard(context.Background(), "From before sign-in", 2, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	gm, friend := newBrowser(t, srv.URL), newBrowser(t, srv.URL)
	gm.signInAs("GM")
	friend.signInAs("Friend")
	if resp, _ := gm.get("/api/boards/" + legacy.ID); resp.StatusCode != http.StatusOK {
		t.Errorf("the admin opening an unowned board: %d", resp.StatusCode)
	}
	if resp, _ := friend.get("/api/boards/" + legacy.ID); resp.StatusCode != http.StatusForbidden {
		t.Errorf("a friend opening an unowned board: %d", resp.StatusCode)
	}
	_, body := gm.get("/api/me")
	if !strings.Contains(body, `"admin":true`) {
		t.Errorf("me: %s", body)
	}
}

func TestDevSignInOnlyInDevMode(t *testing.T) {
	srv := testServerWith(t, func(s *Server) {
		s.SetAuth(auth.Config{Mode: auth.ModeDiscord, PublicURL: "http://localhost", Discord: auth.DefaultDiscord()})
	})
	if resp, _ := newBrowser(t, srv.URL).postForm("/auth/dev", url.Values{"name": {"GM"}}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("dev sign-in in discord mode: %d", resp.StatusCode)
	}
	table := testServer(t)
	if resp, _ := newBrowser(t, table.URL).get("/login"); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("the sign-in page with sign-in off: %d", resp.StatusCode)
	}
}

func TestSignInWithDiscord(t *testing.T) {
	fake := http.NewServeMux()
	fake.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		if r.PostFormValue("code") != "good-code" {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
	})
	fake.HandleFunc("GET /users/@me", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "1001", "username": "sam", "global_name": "Sam"})
	})
	discord := httptest.NewServer(fake)
	t.Cleanup(discord.Close)
	srv := testServerWith(t, func(s *Server) {
		d := auth.DefaultDiscord()
		d.ClientID, d.ClientSecret = "123", "shh"
		d.RedirectURL = "http://localhost/auth/discord/callback"
		d.AuthorizeURL, d.APIBase, d.HTTP = discord.URL+"/oauth2/authorize", discord.URL, discord.Client()
		s.SetAuth(auth.Config{Mode: auth.ModeDiscord, PublicURL: "http://localhost", Discord: d})
	})

	b := newBrowser(t, srv.URL)
	resp, _ := b.get("/auth/discord/login?next=/maps")
	loc, err := url.Parse(resp.Header.Get("Location"))
	if resp.StatusCode != http.StatusSeeOther || err != nil || !strings.HasPrefix(loc.String(), discord.URL+"/oauth2/authorize") {
		t.Fatalf("to Discord: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	state := loc.Query().Get("state")

	if resp, _ := newBrowser(t, srv.URL).get("/auth/discord/callback?code=good-code&state=" + state); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("a callback in another browser (no state cookie): %d", resp.StatusCode)
	}
	resp, _ = b.get("/auth/discord/callback?code=good-code&state=" + url.QueryEscape(state))
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/maps" {
		t.Fatalf("back from Discord: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if _, body := b.get("/api/me"); !strings.Contains(body, `"displayName":"Sam"`) {
		t.Errorf("me: %s", body)
	}

	// Canceling at Discord comes back without a code.
	b2 := newBrowser(t, srv.URL)
	resp, _ = b2.get("/auth/discord/login")
	state = func() string { u, _ := url.Parse(resp.Header.Get("Location")); return u.Query().Get("state") }()
	resp, _ = b2.get("/auth/discord/callback?error=access_denied&state=" + url.QueryEscape(state))
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/login?failed=1") {
		t.Errorf("canceled: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}
