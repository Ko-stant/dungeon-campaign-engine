package app

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/auth"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// Signing in and out (see internal/auth). A login session is a random
// token in an HttpOnly cookie; the database keeps only its hash.

const (
	sessionCookie = "dce_session"
	stateCookie   = "dce_oauth"
	sessionLength = 30 * 24 * time.Hour
	stateLength   = 10 * time.Minute
	maxDevName    = 40
)

// SetAuth switches sign-in on (or off, with auth.ModeNone, the default).
func (s *Server) SetAuth(cfg auth.Config) {
	s.auth = cfg
}

func (s *Server) registerSignIn(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("GET /auth/discord/login", s.discordLogin)
	mux.HandleFunc("GET /auth/discord/callback", s.discordCallback)
	mux.HandleFunc("POST /auth/dev", s.devLogin)
	mux.HandleFunc("POST /auth/logout", s.logout)
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("GET /waiting", s.waitingPage)
}

// signedIn returns the user of the request's login cookie and the viewer
// the store acts for.
func (s *Server) signedIn(r *http.Request) (store.User, store.Viewer, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return store.User{}, store.Viewer{}, false
	}
	u, err := s.store.LoginSessionUser(r.Context(), auth.HashToken(c.Value))
	if err != nil {
		return store.User{}, store.Viewer{}, false
	}
	ids, err := s.store.UserIdentities(r.Context(), u.ID)
	if err != nil {
		log.Printf("app: identities of %s: %v", u.ID, err)
	}
	return u, store.Viewer{UserID: u.ID, Admin: s.auth.IsAdmin(ids)}, true
}

func (s *Server) cookie(name, value string, maxAge time.Duration) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: value, Path: "/", MaxAge: int(maxAge.Seconds()),
		HttpOnly: true, Secure: s.auth.SecureCookies, SameSite: http.SameSiteLaxMode,
	}
}

// beginLogin signs the browser in as u and goes on to next. Admins are
// members from their first sign-in, so nobody waits on them.
func (s *Server) beginLogin(w http.ResponseWriter, r *http.Request, u store.User, next string) {
	if u.Status != store.MemberApproved {
		if ids, err := s.store.UserIdentities(r.Context(), u.ID); err == nil && s.auth.IsAdmin(ids) {
			if err := s.store.SetMemberStatus(r.Context(), u.ID, store.MemberApproved); err != nil {
				log.Printf("app: approving admin %s: %v", u.ID, err)
			}
		}
	}
	token, hash, err := auth.NewToken()
	if err == nil {
		err = s.store.CreateLoginSession(r.Context(), u.ID, hash, time.Now().Add(sessionLength))
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	http.SetCookie(w, s.cookie(sessionCookie, token, sessionLength))
	http.Redirect(w, r, safeNext(next), http.StatusSeeOther)
}

// safeNext keeps a redirect on this site: a local path, or the home page.
func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
		return "/"
	}
	return next
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if !s.auth.On() {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	next := safeNext(r.URL.Query().Get("next"))
	render(w, r, http.StatusOK, views.LoginPage(views.LoginData{Mode: s.auth.Mode, Next: next, Failed: r.URL.Query().Get("failed") != ""}))
}

func randomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// discordLogin sends the browser to Discord, remembering a random state
// (checked when Discord sends them back) and where to go afterward.
func (s *Server) discordLogin(w http.ResponseWriter, r *http.Request) {
	if s.auth.Mode != auth.ModeDiscord {
		http.NotFound(w, r)
		return
	}
	state, err := randomState()
	if err != nil {
		writeStoreError(w, err)
		return
	}
	next := safeNext(r.URL.Query().Get("next"))
	http.SetCookie(w, s.cookie(stateCookie, state+"|"+next, stateLength))
	http.Redirect(w, r, s.auth.Discord.AuthCodeURL(state), http.StatusSeeOther)
}

func (s *Server) discordCallback(w http.ResponseWriter, r *http.Request) {
	if s.auth.Mode != auth.ModeDiscord {
		http.NotFound(w, r)
		return
	}
	c, err := r.Cookie(stateCookie)
	state, next, _ := strings.Cut(func() string {
		if err != nil {
			return ""
		}
		return c.Value
	}(), "|")
	http.SetCookie(w, s.cookie(stateCookie, "", -time.Second))
	got := r.URL.Query().Get("state")
	if state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(got)) != 1 {
		http.Error(w, "The sign-in link is stale or was not started here; try signing in again.", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		// Discord sends error=access_denied when someone cancels.
		http.Redirect(w, r, "/login?failed=1&next="+safeNext(next), http.StatusSeeOther)
		return
	}
	id, err := s.auth.Discord.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("app: discord sign-in: %v", err)
		http.Redirect(w, r, "/login?failed=1&next="+safeNext(next), http.StatusSeeOther)
		return
	}
	u, err := s.store.SignIn(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.beginLogin(w, r, u, next)
}

var devSlug = regexp.MustCompile(`[^a-z0-9]+`)

// devLogin signs anyone in by name, in dev mode and from this machine only.
func (s *Server) devLogin(w http.ResponseWriter, r *http.Request) {
	if s.auth.Mode != auth.ModeDev {
		http.NotFound(w, r)
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		http.Error(w, "Dev sign-in works only from this machine.", http.StatusForbidden)
		return
	}
	name := strings.TrimSpace(r.PostFormValue("name"))
	subject := strings.Trim(devSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if subject == "" || len(name) > maxDevName {
		http.Error(w, "Give a name of up to 40 characters.", http.StatusBadRequest)
		return
	}
	u, err := s.store.SignIn(r.Context(), store.Identity{Provider: auth.ModeDev, Subject: subject, DisplayName: name})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.beginLogin(w, r, u, r.PostFormValue("next"))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if err := s.store.DeleteLoginSession(r.Context(), auth.HashToken(c.Value)); err != nil {
			log.Printf("app: sign out: %v", err)
		}
	}
	http.SetCookie(w, s.cookie(sessionCookie, "", -time.Second))
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// MeResponse is the signed-in user.
type MeResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	Admin       bool   `json:"admin"`
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, v, ok := s.signedIn(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "sign in first")
		return
	}
	writeJSON(w, http.StatusOK, MeResponse{ID: u.ID, DisplayName: u.DisplayName, AvatarURL: u.AvatarURL, Admin: v.Admin})
}
