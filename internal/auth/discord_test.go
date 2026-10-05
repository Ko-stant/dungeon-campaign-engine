package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// fakeDiscord answers the token and user endpoints like Discord does.
func fakeDiscord(t *testing.T, user map[string]any) (*httptest.Server, *Discord) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		id, secret, ok := r.BasicAuth()
		if !ok || id != "123" || secret != "shh" {
			http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
			return
		}
		if r.PostForm.Get("grant_type") != "authorization_code" || r.PostForm.Get("redirect_uri") != "http://localhost:8090/auth/discord/callback" {
			http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
			return
		}
		if r.PostForm.Get("code") != "good-code" {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "token_type": "Bearer", "scope": "identify"})
	})
	mux.HandleFunc("GET /users/@me", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(user)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	d := &Discord{
		ClientID: "123", ClientSecret: "shh", RedirectURL: "http://localhost:8090/auth/discord/callback",
		AuthorizeURL: srv.URL + "/oauth2/authorize", APIBase: srv.URL, HTTP: srv.Client(),
	}
	return srv, d
}

func TestTheAuthorizeURLAsksOnlyWhoYouAre(t *testing.T) {
	_, d := fakeDiscord(t, nil)
	u, err := url.Parse(d.AuthCodeURL("state-1"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("client_id") != "123" || q.Get("response_type") != "code" || q.Get("scope") != "identify" ||
		q.Get("state") != "state-1" || q.Get("redirect_uri") != d.RedirectURL || q.Get("prompt") != "none" {
		t.Errorf("query %v", q)
	}
}

func TestExchangeReturnsTheDiscordIdentity(t *testing.T) {
	_, d := fakeDiscord(t, map[string]any{"id": "1001", "username": "sam", "global_name": "Sam", "avatar": "abc"})
	id, err := d.Exchange(context.Background(), "good-code")
	if err != nil {
		t.Fatal(err)
	}
	want := store.Identity{Provider: "discord", Subject: "1001", DisplayName: "Sam", AvatarURL: "https://cdn.discordapp.com/avatars/1001/abc.png"}
	if id != want {
		t.Errorf("identity %+v, want %+v", id, want)
	}
}

func TestExchangeFallsBackToTheUsername(t *testing.T) {
	_, d := fakeDiscord(t, map[string]any{"id": "1002", "username": "jo", "global_name": nil, "avatar": nil})
	id, err := d.Exchange(context.Background(), "good-code")
	if err != nil || id.DisplayName != "jo" || id.AvatarURL != "" {
		t.Fatalf("identity %+v, %v", id, err)
	}
}

func TestExchangeErrors(t *testing.T) {
	_, d := fakeDiscord(t, map[string]any{"id": "1001", "username": "sam"})
	if _, err := d.Exchange(context.Background(), "bad-code"); err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("a bad code: %v", err)
	}
	d.ClientSecret = "wrong"
	if _, err := d.Exchange(context.Background(), "good-code"); err == nil || strings.Contains(err.Error(), "wrong") {
		t.Errorf("a bad secret (never shown): %v", err)
	}
	_, empty := fakeDiscord(t, map[string]any{"username": "nobody"})
	if _, err := empty.Exchange(context.Background(), "good-code"); err == nil {
		t.Error("a user without an id")
	}
}
