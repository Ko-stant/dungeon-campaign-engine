package auth

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestConfigDefaultsToNoSignIn(t *testing.T) {
	cfg, err := ConfigFromEnv(env(nil))
	if err != nil || cfg.Mode != ModeNone || cfg.On() {
		t.Fatalf("cfg %+v, %v", cfg, err)
	}
	if (Config{}).On() {
		t.Error("an empty config means sign-in off")
	}
}

func TestDiscordConfig(t *testing.T) {
	cfg, err := ConfigFromEnv(env(map[string]string{
		"AUTH_MODE": "discord", "APP_PORT": "8090",
		"DISCORD_CLIENT_ID": "123", "DISCORD_CLIENT_SECRET": "shh",
		"AUTH_ADMINS": "discord:1001, dev:gm",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != ModeDiscord || cfg.PublicURL != "http://localhost:8090" || cfg.SecureCookies {
		t.Errorf("cfg %+v", cfg)
	}
	if got := cfg.Discord.RedirectURL; got != "http://localhost:8090/auth/discord/callback" {
		t.Errorf("redirect %q", got)
	}
	if !cfg.IsAdmin([]store.IdentityKey{{Provider: "discord", Subject: "1001"}}) || cfg.IsAdmin([]store.IdentityKey{{Provider: "discord", Subject: "1002"}}) {
		t.Errorf("admins %+v", cfg.Admins)
	}
}

func TestAnHTTPSPublicURLMakesSecureCookies(t *testing.T) {
	cfg, err := ConfigFromEnv(env(map[string]string{
		"AUTH_MODE": "discord", "PUBLIC_URL": "https://dce.example.com/",
		"DISCORD_CLIENT_ID": "123", "DISCORD_CLIENT_SECRET": "shh",
	}))
	if err != nil || !cfg.SecureCookies || cfg.Discord.RedirectURL != "https://dce.example.com/auth/discord/callback" {
		t.Fatalf("cfg %+v, %v", cfg, err)
	}
}

func TestConfigErrors(t *testing.T) {
	for name, vars := range map[string]map[string]string{
		"unknown mode":   {"AUTH_MODE": "magic"},
		"no client id":   {"AUTH_MODE": "discord", "DISCORD_CLIENT_SECRET": "shh"},
		"no secret":      {"AUTH_MODE": "discord", "DISCORD_CLIENT_ID": "123"},
		"bad admin":      {"AUTH_MODE": "dev", "AUTH_ADMINS": "gm"},
		"bad public URL": {"AUTH_MODE": "dev", "PUBLIC_URL": "dce.example.com"},
	} {
		if _, err := ConfigFromEnv(env(vars)); err == nil {
			t.Errorf("%s: expected an error", name)
		} else if strings.Contains(err.Error(), "shh") {
			t.Errorf("%s: the error shows the secret: %v", name, err)
		}
	}
}

func TestMembersAreApprovedUnlessOpen(t *testing.T) {
	cfg, err := ConfigFromEnv(env(map[string]string{"AUTH_MODE": "dev"}))
	if err != nil || cfg.OpenMembership {
		t.Fatalf("by default the GM approves members: %+v %v", cfg, err)
	}
	cfg, err = ConfigFromEnv(env(map[string]string{"AUTH_MODE": "dev", "AUTH_MEMBERS": "open"}))
	if err != nil || !cfg.OpenMembership {
		t.Fatalf("open: %+v %v", cfg, err)
	}
	if _, err := ConfigFromEnv(env(map[string]string{"AUTH_MODE": "dev", "AUTH_MEMBERS": "everyone"})); err == nil || !strings.Contains(err.Error(), "AUTH_MEMBERS") {
		t.Errorf("an unknown setting: %v", err)
	}
}
