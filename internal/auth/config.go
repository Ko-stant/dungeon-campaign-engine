// Package auth is sign-in for online play (docs/ONLINE_AND_RULES_PLAN.md,
// Phase 3): its configuration, Sign in with Discord, and the random tokens
// behind login cookies. Who may do what is decided in internal/app.
package auth

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// Modes.
const (
	// ModeNone is the table companion: no sign-in, everything open.
	ModeNone = "none"
	// ModeDiscord signs people in with Discord.
	ModeDiscord = "discord"
	// ModeDev signs anyone in by name, from this machine only: for local
	// testing and bots.
	ModeDev = "dev"
)

// Config is how sign-in is set up, read from the environment.
type Config struct {
	Mode string
	// PublicURL is where the app is reached, without a trailing slash.
	PublicURL string
	// SecureCookies is set when PublicURL is https.
	SecureCookies bool
	Discord       Discord
	// Admins may open and change everything, including things made before
	// sign-in existed (which have no owner).
	Admins []store.IdentityKey
}

// ConfigFromEnv reads AUTH_MODE (none, discord or dev), PUBLIC_URL (default
// http://localhost:APP_PORT), DISCORD_CLIENT_ID, DISCORD_CLIENT_SECRET and
// AUTH_ADMINS ("discord:1234, dev:gm"). Errors never include the secret.
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := Config{Mode: strings.TrimSpace(getenv("AUTH_MODE"))}
	if cfg.Mode == "" {
		cfg.Mode = ModeNone
	}
	if cfg.Mode != ModeNone && cfg.Mode != ModeDiscord && cfg.Mode != ModeDev {
		return Config{}, fmt.Errorf("AUTH_MODE must be none, discord or dev, got %q", cfg.Mode)
	}

	public := strings.TrimRight(strings.TrimSpace(getenv("PUBLIC_URL")), "/")
	if public == "" {
		port := getenv("APP_PORT")
		if port == "" {
			port = "8080"
		}
		public = "http://localhost:" + port
	}
	u, err := url.Parse(public)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Config{}, fmt.Errorf("PUBLIC_URL must be an http or https address, got %q", public)
	}
	cfg.PublicURL, cfg.SecureCookies = public, u.Scheme == "https"

	cfg.Discord = DefaultDiscord()
	cfg.Discord.ClientID = strings.TrimSpace(getenv("DISCORD_CLIENT_ID"))
	cfg.Discord.ClientSecret = strings.TrimSpace(getenv("DISCORD_CLIENT_SECRET"))
	cfg.Discord.RedirectURL = public + "/auth/discord/callback"
	if cfg.Mode == ModeDiscord && (cfg.Discord.ClientID == "" || cfg.Discord.ClientSecret == "") {
		return Config{}, errors.New("AUTH_MODE=discord needs DISCORD_CLIENT_ID and DISCORD_CLIENT_SECRET")
	}

	for entry := range strings.SplitSeq(getenv("AUTH_ADMINS"), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		provider, subject, ok := strings.Cut(entry, ":")
		if !ok || provider == "" || subject == "" {
			return Config{}, fmt.Errorf("AUTH_ADMINS entries look like discord:1234 or dev:gm, got %q", entry)
		}
		cfg.Admins = append(cfg.Admins, store.IdentityKey{Provider: provider, Subject: subject})
	}
	return cfg, nil
}

// On reports whether sign-in is required (any mode but none; an empty mode
// is none).
func (c Config) On() bool {
	return c.Mode != "" && c.Mode != ModeNone
}

// IsAdmin reports whether any of a user's identities is an admin.
func (c Config) IsAdmin(ids []store.IdentityKey) bool {
	return slices.ContainsFunc(ids, func(k store.IdentityKey) bool { return slices.Contains(c.Admins, k) })
}
