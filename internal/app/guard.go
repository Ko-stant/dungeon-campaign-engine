package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views/components"
)

// Who may use which route when sign-in is on (docs/ONLINE_AND_RULES_PLAN.md,
// Phase 3). Every app route is registered through a guard that reads what
// the route touches from its own pattern: a board, quest, campaign, session,
// custom monster or class named in the path is open to its owner and to
// admins; routes naming nothing (lists, creating) need only a signed-in
// user, and the store keeps lists to the viewer's own. With sign-in off
// (AUTH_MODE=none, the table companion) the guard does nothing.

// routeMux is what the register functions add routes to.
type routeMux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// ownedParam is one thing a route touches: a path parameter and its kind.
// Seated also lets in players with a hero in the session (player-safe
// routes only).
type ownedParam struct {
	param  string
	kind   store.OwnedKind
	seated bool
}

// accessRule lists what a route touches, from its pattern; ok is false for a
// pattern with no rule (registering it panics, so no route goes unguarded).
func accessRule(pattern string) (owned []ownedParam, ok bool) {
	_, path, _ := strings.Cut(pattern, " ")
	if path == "" {
		path = pattern
	}
	byPrefix := []struct {
		prefix string
		owned  []ownedParam
	}{
		// Player-safe session routes: the player view and the seat.
		{"/api/sessions/{id}/seat", []ownedParam{{"id", store.OwnedSession, true}}},
		{"/api/sessions/{id}/seat-commands", []ownedParam{{"id", store.OwnedSession, true}}},
		{"/api/sessions/{id}/player", []ownedParam{{"id", store.OwnedSession, true}}},
		{"/api/sessions/{id}/player-stream", []ownedParam{{"id", store.OwnedSession, true}}},
		{"/play/{id}/players", []ownedParam{{"id", store.OwnedSession, true}}},
		// Joining checks the session is open itself.
		{"/join/{id}", nil},
		{"/api/boards/{id}", []ownedParam{{"id", store.OwnedBoard, false}}},
		{"/maps/{id}", []ownedParam{{"id", store.OwnedBoard, false}}},
		{"/api/quests/{id}", []ownedParam{{"id", store.OwnedQuest, false}}},
		{"/api/campaigns/{id}", []ownedParam{{"id", store.OwnedCampaign, false}}},
		{"/campaigns/{id}/chapters/{questId}", []ownedParam{{"id", store.OwnedCampaign, false}, {"questId", store.OwnedQuest, false}}},
		{"/campaigns/{id}", []ownedParam{{"id", store.OwnedCampaign, false}}},
		{"/audio/{campaign}", []ownedParam{{"campaign", store.OwnedCampaign, false}}},
		{"/api/sessions/{id}", []ownedParam{{"id", store.OwnedSession, false}}},
		{"/play/{id}", []ownedParam{{"id", store.OwnedSession, false}}},
		{"/monsters/{id}", []ownedParam{{"id", store.OwnedMonster, false}}},
		{"/classes/{id}", []ownedParam{{"id", store.OwnedClass, false}}},
	}
	for _, r := range byPrefix {
		if path == r.prefix || strings.HasPrefix(path, r.prefix+"/") {
			return r.owned, true
		}
	}
	switch path {
	case "/api/catalog", "/api/boards", "/api/campaigns", "/maps", "/campaigns", "/monsters", "/classes", "/classes/new", "/lobby":
		return nil, true
	}
	return nil, false
}

// guarded registers routes through the guard.
type guarded struct {
	s   *Server
	mux *http.ServeMux
}

func (g guarded) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	owned, ok := accessRule(pattern)
	if !ok {
		panic(fmt.Sprintf("app: route %q has no access rule (see guard.go)", pattern))
	}
	g.mux.HandleFunc(pattern, g.s.guard(owned, h))
}

// guard checks the signed-in user may use what the route touches, and runs
// the handler acting for them.
func (s *Server) guard(owned []ownedParam, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.On() {
			h(w, r)
			return
		}
		user, viewer, ok := s.signedIn(r)
		if !ok {
			s.notSignedIn(w, r)
			return
		}
		ctx := components.WithSignedIn(store.WithViewer(r.Context(), viewer), user.DisplayName)
		for _, o := range owned {
			if err := s.mayUse(ctx, o.kind, r.PathValue(o.param)); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					break // the handler answers 404
				}
				if errors.Is(err, errNotYours) && o.seated && s.seatedIn(ctx, r.PathValue(o.param), user.ID) {
					continue
				}
				if errors.Is(err, errNotYours) {
					s.forbidden(w, r)
					return
				}
				writeStoreError(w, err)
				return
			}
		}
		h(w, r.WithContext(ctx))
	}
}

var errNotYours = errors.New("this belongs to someone else")

// mayUse checks the context's viewer may use something: it is theirs, or
// they are an admin. Without a viewer (sign-in off) everything may be used.
func (s *Server) mayUse(ctx context.Context, kind store.OwnedKind, id string) error {
	v, ok := store.ViewerFrom(ctx)
	if !ok || v.Admin {
		return nil
	}
	owner, err := s.store.OwnerOf(ctx, kind, id)
	if err != nil {
		return err
	}
	if owner != v.UserID {
		return errNotYours
	}
	return nil
}

// mayUseFromRequest is mayUse for an id a handler read from a request body:
// someone else's thing is answered as if it did not exist.
func (s *Server) mayUseFromRequest(ctx context.Context, kind store.OwnedKind, id string) error {
	if err := s.mayUse(ctx, kind, id); errors.Is(err, errNotYours) {
		return store.ErrNotFound
	} else if err != nil {
		return err
	}
	return nil
}

// notSignedIn sends pages to the sign-in page and answers anything else 401.
func (s *Server) notSignedIn(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/") {
		http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
		return
	}
	writeError(w, http.StatusUnauthorized, "sign in first")
}

func (s *Server) forbidden(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusForbidden, "this belongs to someone else")
		return
	}
	http.Error(w, "This belongs to someone else.", http.StatusForbidden)
}
