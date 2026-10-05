package app

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// The lobby (docs/ONLINE_AND_RULES_PLAN.md, Phase 4): the GM opens a
// session to players; signed-in players see open games, pick a free hero
// or make their own, and join, even after the game has started. A player
// may play several heroes. Who plays a hero is the campaign hero's UserID;
// a player's seat in a session is every hero there that is theirs.

func (s *Server) registerLobby(mux routeMux) {
	mux.HandleFunc("GET /lobby", s.lobbyPage)
	mux.HandleFunc("GET /join/{id}", s.joinPage)
	mux.HandleFunc("POST /join/{id}/hero", s.joinWithNewHero)
	mux.HandleFunc("POST /join/{id}/claim/{heroId}", s.joinWithFreeHero)
	mux.HandleFunc("POST /play/{id}/open", s.openSessionForm)
	mux.HandleFunc("POST /play/{id}/start", s.startOnlineForm)
	mux.HandleFunc("GET /api/sessions/{id}/seat", s.getSeat)
	mux.HandleFunc("POST /api/sessions/{id}/seat-commands", s.seatCommand)
}

// signedInUser is the request's user. The lobby and seats exist only with
// sign-in on; without it (the table companion) they answer 404.
func (s *Server) signedInUser(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	if !s.auth.On() {
		http.NotFound(w, r)
		return store.User{}, false
	}
	u, _, ok := s.signedIn(r)
	if !ok || u.ID == "" {
		writeError(w, http.StatusUnauthorized, "sign in first")
		return store.User{}, false
	}
	return u, true
}

// seatedIn reports whether a user plays a hero in a session's campaign.
func (s *Server) seatedIn(ctx context.Context, sessionID, userID string) bool {
	ss, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return false
	}
	_, heroes, err := s.loadCampaign(ctx, ss.CampaignID)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(heroes, func(h tracker.CampaignHero) bool { return h.UserID == userID })
}

func (s *Server) lobbyPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedInUser(w, r); !ok {
		return
	}
	list, err := s.store.ListOpenSessions(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	games := make([]views.LobbyItem, 0, len(list))
	for _, o := range list {
		games = append(games, views.LobbyItem{SessionID: o.ID, Name: o.Name, CampaignName: o.CampaignName, GMName: o.GMName, Started: o.Started, UpdatedAt: o.UpdatedAt})
	}
	render(w, r, http.StatusOK, views.LobbyPage(games))
}

// openSession loads a session that is open to players and active.
func (s *Server) openSession(ctx context.Context, id string) (store.Session, error) {
	ss, err := s.store.GetSession(ctx, id)
	if err != nil {
		return store.Session{}, err
	}
	if !ss.Open || ss.Status != store.StatusActive {
		return store.Session{}, store.ErrNotFound
	}
	return ss, nil
}

func (s *Server) joinPage(w http.ResponseWriter, r *http.Request) {
	s.renderJoinPage(w, r, http.StatusOK, "")
}

func (s *Server) renderJoinPage(w http.ResponseWriter, r *http.Request, status int, problem string) {
	ctx := r.Context()
	user, ok := s.signedInUser(w, r)
	if !ok {
		return
	}
	ss, err := s.openSession(ctx, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	camp, heroes, err := s.loadCampaign(ctx, ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	cat, err := s.campaignCatalog(ctx, ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	d := views.JoinData{SessionID: ss.ID, SessionName: ss.Name, CampaignName: camp.Name, Error: problem}
	for _, h := range heroes {
		className := h.Class
		if def, ok := cat.Hero(h.Class); ok {
			className = def.Name
		}
		jh := views.JoinHero{ID: h.ID, Name: h.Name, ClassName: className, Player: h.Player}
		switch h.UserID {
		case user.ID:
			d.Yours = append(d.Yours, jh)
		case "":
			d.Free = append(d.Free, jh)
		default:
			d.Taken = append(d.Taken, jh)
		}
	}
	for _, c := range cat.Heroes {
		d.Classes = append(d.Classes, views.ClassChoice{ID: c.ID, Name: c.Name})
	}
	render(w, r, status, views.JoinPage(d))
}

// joinHero saves the campaign's heroes with this one played by the user,
// and brings the hero into the running session if not there yet.
func (s *Server) joinHero(ctx context.Context, ss store.Session, camp store.Campaign, heroes []tracker.CampaignHero, hero tracker.CampaignHero) error {
	if _, err := s.saveCampaignHeroes(ctx, camp.ID, camp.Name, heroes); err != nil {
		return err
	}
	_, state, err := s.loadSessionState(ctx, ss.ID)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(state.Heroes, func(h tracker.Hero) bool { return h.ID == hero.ID }) {
		return nil
	}
	payload, err := json.Marshal(map[string]any{"hero": hero})
	if err != nil {
		return err
	}
	_, err = s.applyCommand(ctx, ss.ID, tracker.Command{Type: "hero.join", Payload: payload})
	return err
}

func (s *Server) joinWithNewHero(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := s.signedInUser(w, r)
	if !ok {
		return
	}
	ss, err := s.openSession(ctx, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	camp, heroes, err := s.loadCampaign(ctx, ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	cat, err := s.campaignCatalog(ctx, ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	newHero := tracker.CampaignHero{Name: r.PostFormValue("name"), Class: r.PostFormValue("class"), Player: user.DisplayName, UserID: user.ID}
	all, err := normalizeHeroes(cat, append(heroes, newHero))
	if err != nil {
		s.renderJoinPage(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.joinHero(ctx, ss, camp, all, all[len(all)-1]); err != nil {
		writeCommandError(w, err)
		return
	}
	http.Redirect(w, r, "/join/"+ss.ID, http.StatusSeeOther)
}

func (s *Server) joinWithFreeHero(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := s.signedInUser(w, r)
	if !ok {
		return
	}
	ss, err := s.openSession(ctx, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	camp, heroes, err := s.loadCampaign(ctx, ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	i := slices.IndexFunc(heroes, func(h tracker.CampaignHero) bool { return h.ID == r.PathValue("heroId") })
	switch {
	case i < 0:
		http.NotFound(w, r)
		return
	case heroes[i].UserID != "" && heroes[i].UserID != user.ID:
		s.renderJoinPage(w, r, http.StatusConflict, heroes[i].Name+" is taken.")
		return
	}
	heroes[i].UserID, heroes[i].Player = user.ID, user.DisplayName
	if err := s.joinHero(ctx, ss, camp, heroes, heroes[i]); err != nil {
		writeCommandError(w, err)
		return
	}
	http.Redirect(w, r, "/join/"+ss.ID, http.StatusSeeOther)
}

// openSessionForm opens a session to players (open=1) or closes it.
func (s *Server) openSessionForm(w http.ResponseWriter, r *http.Request) {
	ss, err := s.store.GetSession(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.store.SetSessionOpen(r.Context(), ss.ID, r.PostFormValue("open") == "1"); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+ss.CampaignID, http.StatusSeeOther)
}

// startOnlineForm turns the rules on with a fresh random seed; starting a
// game already under the rules just goes back.
func (s *Server) startOnlineForm(w http.ResponseWriter, r *http.Request) {
	ss, state, err := s.loadSessionState(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if state.Rules == nil {
		var b [4]byte
		if _, err := rand.Read(b[:]); err != nil {
			writeStoreError(w, err)
			return
		}
		payload := fmt.Appendf(nil, `{"seed": %d}`, binary.BigEndian.Uint32(b[:]))
		if _, err := s.applyCommand(r.Context(), ss.ID, tracker.Command{Type: "rules.enable", Payload: payload}); err != nil {
			writeCommandError(w, err)
			return
		}
	}
	http.Redirect(w, r, "/campaigns/"+ss.CampaignID, http.StatusSeeOther)
}

// SeatHero is one of the player's heroes in a session, with what they may
// do now.
type SeatHero struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Placed  bool             `json:"placed"`
	Actions []tracker.Action `json:"actions"`
}

// SeatResponse is a player's seat: the game as the players see it and
// their heroes' legal actions. (Phase 5 adds each hero's own sheet.)
type SeatResponse struct {
	Round  int                 `json:"round"`
	Phase  string              `json:"phase,omitempty"`
	Heroes []SeatHero          `json:"heroes"`
	Player tracker.PlayerState `json:"player"`
}

// seat builds a user's seat in a session.
func (s *Server) seat(ctx context.Context, sessionID, userID string) (SeatResponse, error) {
	ss, state, err := s.loadSessionState(ctx, sessionID)
	if err != nil {
		return SeatResponse{}, err
	}
	_, heroes, err := s.loadCampaign(ctx, ss.CampaignID)
	if err != nil {
		return SeatResponse{}, err
	}
	cat, err := s.campaignCatalog(ctx, ss.CampaignID)
	if err != nil {
		return SeatResponse{}, err
	}
	resp := SeatResponse{Round: state.Round, Heroes: []SeatHero{}, Player: tracker.PlayerView(state)}
	if state.Rules != nil {
		resp.Phase = state.Rules.Phase
	}
	for _, h := range state.Heroes {
		if !slices.ContainsFunc(heroes, func(ch tracker.CampaignHero) bool { return ch.ID == h.ID && ch.UserID == userID }) {
			continue
		}
		actions := tracker.LegalActions(state, tracker.Actor{Kind: tracker.ActorSeat, HeroID: h.ID}, cat)
		if actions == nil {
			actions = []tracker.Action{}
		}
		resp.Heroes = append(resp.Heroes, SeatHero{ID: h.ID, Name: h.Name, Placed: h.Placed, Actions: actions})
	}
	return resp, nil
}

func (s *Server) getSeat(w http.ResponseWriter, r *http.Request) {
	user, ok := s.signedInUser(w, r)
	if !ok {
		return
	}
	resp, err := s.seat(r.Context(), r.PathValue("id"), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// seatCommand applies a player's command for one of their heroes; the
// actor is that hero's seat, whatever the client says.
func (s *Server) seatCommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hero    string          `json:"hero"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	user, ok := s.signedInUser(w, r)
	if !ok {
		return
	}
	if !readJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	ss, err := s.store.GetSession(ctx, r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	_, heroes, err := s.loadCampaign(ctx, ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if !slices.ContainsFunc(heroes, func(h tracker.CampaignHero) bool { return h.ID == req.Hero && h.UserID == user.ID }) {
		writeError(w, http.StatusForbidden, "you don't play that hero")
		return
	}
	cmd := tracker.Command{Type: strings.TrimSpace(req.Type), Payload: req.Payload, Actor: tracker.Actor{Kind: tracker.ActorSeat, HeroID: req.Hero}}
	if _, err := s.applyCommand(ctx, ss.ID, cmd); err != nil {
		writeCommandError(w, err)
		return
	}
	seat, err := s.seat(ctx, ss.ID, user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, seat)
}
