package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// Discord is Sign in with Discord: the OAuth2 authorization code flow with
// only the "identify" scope (who you are; no email, no servers).
type Discord struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	// AuthorizeURL is Discord's consent page; APIBase serves the token and
	// user endpoints. Tests point them at a fake.
	AuthorizeURL string
	APIBase      string
	HTTP         *http.Client
}

// DefaultDiscord is Discord's real endpoints.
func DefaultDiscord() Discord {
	return Discord{
		AuthorizeURL: "https://discord.com/oauth2/authorize",
		APIBase:      "https://discord.com/api",
		HTTP:         &http.Client{Timeout: 10 * time.Second},
	}
}

// AuthCodeURL is where to send someone to sign in; state comes back with
// them and must match.
func (d *Discord) AuthCodeURL(state string) string {
	q := url.Values{
		"client_id":     {d.ClientID},
		"redirect_uri":  {d.RedirectURL},
		"response_type": {"code"},
		"scope":         {"identify"},
		"state":         {state},
		"prompt":        {"none"},
	}
	return d.AuthorizeURL + "?" + q.Encode()
}

// Exchange turns the code Discord sent back into the person's identity.
func (d *Discord) Exchange(ctx context.Context, code string) (store.Identity, error) {
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {d.RedirectURL}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.APIBase+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return store.Identity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(d.ClientID, d.ClientSecret)
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := d.do(req, &tok); err != nil {
		return store.Identity{}, fmt.Errorf("discord token: %w", err)
	}
	if tok.AccessToken == "" {
		return store.Identity{}, errors.New("discord token: no access token")
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, d.APIBase+"/users/@me", nil)
	if err != nil {
		return store.Identity{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	var u struct {
		ID         string  `json:"id"`
		Username   string  `json:"username"`
		GlobalName *string `json:"global_name"`
		Avatar     *string `json:"avatar"`
	}
	if err := d.do(req, &u); err != nil {
		return store.Identity{}, fmt.Errorf("discord user: %w", err)
	}
	if u.ID == "" {
		return store.Identity{}, errors.New("discord user: no id")
	}
	id := store.Identity{Provider: ModeDiscord, Subject: u.ID, DisplayName: u.Username}
	if u.GlobalName != nil && *u.GlobalName != "" {
		id.DisplayName = *u.GlobalName
	}
	if u.Avatar != nil && *u.Avatar != "" {
		id.AvatarURL = "https://cdn.discordapp.com/avatars/" + url.PathEscape(u.ID) + "/" + url.PathEscape(*u.Avatar) + ".png"
	}
	return id, nil
}

// do sends a request and decodes a JSON answer. An error answer is reported
// by Discord's error code only, so nothing sensitive is echoed.
func (d *Discord) do(req *http.Request, into any) error {
	client := d.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &e)
		if e.Error == "" {
			e.Error = "unexpected answer"
		}
		return fmt.Errorf("%s (HTTP %d)", e.Error, resp.StatusCode)
	}
	return json.Unmarshal(body, into)
}
