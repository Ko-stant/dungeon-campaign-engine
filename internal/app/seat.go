package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// A player's seat online (docs/ONLINE_AND_RULES_PLAN.md, Phase 5): the game
// as the players see it (the player view, the feed), plus the player's own
// heroes' sheets and legal actions, and who else is here. The seat stream
// pushes all of it after every change, each player getting their own seat.

// SeatResponse is a player's game on load.
type SeatResponse struct {
	PlayerResponse
	Seat     tracker.SeatState `json:"seat"`
	Presence []Present         `json:"presence"`
}

// SeatUpdate is pushed to a seat: a change (player view and seat), or who
// is here.
type SeatUpdate struct {
	Player   *PlayerUpdate      `json:"player,omitempty"`
	Seat     *tracker.SeatState `json:"seat,omitempty"`
	Presence []Present          `json:"presence,omitempty"`
}

// Present is a player connected to a session, with the heroes they play.
type Present struct {
	Name   string   `json:"name"`
	Heroes []string `json:"heroes"`
}

// PresenceUpdate tells the GM's tracker who is connected.
type PresenceUpdate struct {
	Presence []Present `json:"presence"`
}

// seatConn is one connected seat.
type seatConn struct {
	conn   *websocket.Conn
	userID string
	name   string
	heroes []string
}

// seatHub holds each session's connected seats.
type seatHub struct {
	mu    sync.Mutex
	conns map[string]map[*seatConn]struct{}
}

func (h *seatHub) add(id string, c *seatConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns == nil {
		h.conns = map[string]map[*seatConn]struct{}{}
	}
	if h.conns[id] == nil {
		h.conns[id] = map[*seatConn]struct{}{}
	}
	h.conns[id][c] = struct{}{}
}

func (h *seatHub) remove(id string, c *seatConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns[id], c)
	if len(h.conns[id]) == 0 {
		delete(h.conns, id)
	}
}

func (h *seatHub) list(id string) []*seatConn {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*seatConn, 0, len(h.conns[id]))
	for c := range h.conns[id] {
		out = append(out, c)
	}
	return out
}

// presence lists who is connected to a session, by name, once each.
func (h *seatHub) presence(id string) []Present {
	byUser := map[string]*Present{}
	for _, c := range h.list(id) {
		if p, ok := byUser[c.userID]; ok {
			for _, hero := range c.heroes {
				if !slices.Contains(p.Heroes, hero) {
					p.Heroes = append(p.Heroes, hero)
				}
			}
			continue
		}
		byUser[c.userID] = &Present{Name: c.name, Heroes: slices.Clone(c.heroes)}
	}
	out := make([]Present, 0, len(byUser))
	for _, p := range byUser {
		sort.Strings(p.Heroes)
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

func (h *seatHub) send(id string, c *seatConn, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("app: seat encode: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
		h.remove(id, c)
		_ = c.conn.CloseNow()
	}
}

// heroesOf returns the ids and names of a user's heroes in a campaign.
func (s *Server) heroesOf(ctx context.Context, campaignID, userID string) ([]string, []string, error) {
	_, heroes, err := s.loadCampaign(ctx, campaignID)
	if err != nil {
		return nil, nil, err
	}
	var ids, names []string
	for _, h := range heroes {
		if h.UserID == userID {
			ids, names = append(ids, h.ID), append(names, h.Name)
		}
	}
	return ids, names, nil
}

// seatResponse builds a user's seat in a session.
func (s *Server) seatResponse(ctx context.Context, sessionID, userID string) (SeatResponse, error) {
	ss, state, err := s.loadSessionState(ctx, sessionID)
	if err != nil {
		return SeatResponse{}, err
	}
	events, err := s.playerFeed(ctx, sessionID)
	if err != nil {
		return SeatResponse{}, err
	}
	cat, err := s.campaignCatalog(ctx, ss.CampaignID)
	if err != nil {
		return SeatResponse{}, err
	}
	ids, _, err := s.heroesOf(ctx, ss.CampaignID, userID)
	if err != nil {
		return SeatResponse{}, err
	}
	return SeatResponse{
		PlayerResponse: PlayerResponse{State: tracker.PlayerView(state), Events: events, EventSeq: ss.EventSeq, Catalog: playerCatalog(cat)},
		Seat:           tracker.SeatView(state, ids, cat),
		Presence:       s.seats.presence(sessionID),
	}, nil
}

// seatPage is the shell of a player's game screen.
func (s *Server) seatPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedInUser(w, r); !ok {
		return
	}
	ss, err := s.store.GetSession(r.Context(), r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, r, http.StatusOK, views.SeatPage(ss.ID, ss.Name))
}

func (s *Server) getSeat(w http.ResponseWriter, r *http.Request) {
	user, ok := s.signedInUser(w, r)
	if !ok {
		return
	}
	resp, err := s.seatResponse(r.Context(), r.PathValue("id"), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// seatCommand applies a player's command for one of their heroes; the
// actor is that hero's seat, whatever the client says. It answers with the
// seat after the change.
func (s *Server) seatCommand(w http.ResponseWriter, r *http.Request) {
	user, ok := s.signedInUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Hero    string          `json:"hero"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
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
	ids, _, err := s.heroesOf(ctx, ss.CampaignID, user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if !slices.Contains(ids, req.Hero) {
		writeError(w, http.StatusForbidden, "you don't play that hero")
		return
	}
	cmd := tracker.Command{Type: strings.TrimSpace(req.Type), Payload: req.Payload, Actor: tracker.Actor{Kind: tracker.ActorSeat, HeroID: req.Hero}}
	if _, err := s.applyCommand(ctx, ss.ID, cmd); err != nil {
		writeCommandError(w, err)
		return
	}
	resp, err := s.seatResponse(ctx, ss.ID, user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// seatStream is a player's live connection: their seat after every change,
// and who is here whenever someone comes or goes.
func (s *Server) seatStream(w http.ResponseWriter, r *http.Request) {
	user, ok := s.signedInUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	ss, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	_, names, err := s.heroesOf(r.Context(), ss.CampaignID, user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	c := &seatConn{conn: conn, userID: user.ID, name: user.DisplayName, heroes: names}
	s.seats.add(id, c)
	s.announcePresence(id)
	defer func() {
		s.seats.remove(id, c)
		s.announcePresence(id)
	}()
	s.listen(r.Context(), conn)
}

// announcePresence tells every seat and the GM's tracker who is here.
func (s *Server) announcePresence(id string) {
	present := s.seats.presence(id)
	for _, c := range s.seats.list(id) {
		s.seats.send(id, c, SeatUpdate{Presence: present})
	}
	s.streams.broadcast(id, PresenceUpdate{Presence: present})
}

func (s *Server) getPresence(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, PresenceUpdate{Presence: s.seats.presence(r.PathValue("id"))})
}

// pushSeats sends every connected seat its seat after a change.
func (s *Server) pushSeats(ctx context.Context, sessionID string, state *tracker.State, update PlayerUpdate) {
	conns := s.seats.list(sessionID)
	if len(conns) == 0 {
		return
	}
	ss, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		log.Printf("app: seats of %s: %v", sessionID, err)
		return
	}
	_, heroes, err := s.loadCampaign(ctx, ss.CampaignID)
	if err != nil {
		log.Printf("app: seats of %s: %v", sessionID, err)
		return
	}
	cat, err := s.campaignCatalog(ctx, ss.CampaignID)
	if err != nil {
		log.Printf("app: seats of %s: %v", sessionID, err)
		return
	}
	for _, c := range conns {
		var ids []string
		for _, h := range heroes {
			if h.UserID == c.userID {
				ids = append(ids, h.ID)
			}
		}
		seat := tracker.SeatView(state, ids, cat)
		s.seats.send(sessionID, c, SeatUpdate{Player: &update, Seat: &seat})
	}
}
