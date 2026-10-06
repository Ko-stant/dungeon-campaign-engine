package app

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// The player screen's API: only tracker.PlayerView and player-safe event
// lines ever go out here, never the GM's state or log.

// playerFeedLength is how many recent player lines the screen gets on load.
const playerFeedLength = 30

// PlayerEventResponse is one line of the player screen's event feed.
type PlayerEventResponse struct {
	Seq       int64     `json:"seq"`
	Round     int       `json:"round"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"createdAt"`
}

// PlayerResponse is the player screen on load: the view, the latest lines
// and what the screen needs from the catalog to draw it.
type PlayerResponse struct {
	State    tracker.PlayerState   `json:"state"`
	Events   []PlayerEventResponse `json:"events"`
	EventSeq int64                 `json:"eventSeq"`
	Catalog  PlayerCatalog         `json:"catalog"`
}

// PlayerCatalog is the catalog as the player screen sees it: names, sizes,
// colors and artwork only (custom monsters' notes and class details stay out).
type PlayerCatalog struct {
	Furniture []content.FurnitureDef `json:"furniture"`
	Traps     []content.TrapDef      `json:"traps"`
	Monsters  []PlayerMonsterDef     `json:"monsters"`
	Heroes    []PlayerHeroDef        `json:"heroes"`
}

// PlayerMonsterDef is a monster type for the player screen.
type PlayerMonsterDef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Movement int    `json:"movement,omitempty"`
	Image    string `json:"image,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Color    string `json:"color,omitempty"`
}

// PlayerHeroDef is a hero class for the player screen.
type PlayerHeroDef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

func playerCatalog(cat *content.Catalog) PlayerCatalog {
	out := PlayerCatalog{
		Furniture: slices.Clone(cat.Furniture), Traps: slices.Clone(cat.Traps),
		Monsters: make([]PlayerMonsterDef, 0, len(cat.Monsters)), Heroes: make([]PlayerHeroDef, 0, len(cat.Heroes)),
	}
	if out.Furniture == nil {
		out.Furniture = []content.FurnitureDef{}
	}
	if out.Traps == nil {
		out.Traps = []content.TrapDef{}
	}
	for _, m := range cat.Monsters {
		out.Monsters = append(out.Monsters, PlayerMonsterDef{ID: m.ID, Name: m.Name, Movement: m.Movement, Image: m.Image, Width: m.Width, Height: m.Height, Color: m.Color})
	}
	for _, h := range cat.Heroes {
		out.Heroes = append(out.Heroes, PlayerHeroDef{ID: h.ID, Name: h.Name, Color: h.Color})
	}
	return out
}

// PlayerUpdate is pushed to player screens on every change; Event is nil
// when the players hear nothing about it. Feed (nil otherwise) replaces the
// screen's feed when earlier lines changed: a sighting taken back.
type PlayerUpdate struct {
	State    tracker.PlayerState   `json:"state"`
	Event    *PlayerEventResponse  `json:"event"`
	EventSeq int64                 `json:"eventSeq"`
	Feed     []PlayerEventResponse `json:"feed"`
}

// playerEventResponse is an event's player line: its sightings (monster ids
// stay on the server) and the rest.
func playerEventResponse(e store.Event) PlayerEventResponse {
	var spotted []tracker.SpottedMonster
	_ = json.Unmarshal(e.PlayerSpotted, &spotted) // a bad value just shows no sightings
	return PlayerEventResponse{Seq: e.Seq, Round: e.Round, Summary: tracker.PlayerLine(spotted, e.PlayerSummary), CreatedAt: e.CreatedAt}
}

// playerFeed is the screen's latest lines, oldest first.
func (s *Server) playerFeed(ctx context.Context, sessionID string) ([]PlayerEventResponse, error) {
	events, err := s.store.ListPlayerEvents(ctx, sessionID, playerFeedLength)
	if err != nil {
		return nil, err
	}
	out := make([]PlayerEventResponse, 0, len(events))
	for _, e := range events {
		if line := playerEventResponse(e); line.Summary != "" {
			out = append(out, line)
		}
	}
	return out, nil
}

// newEvent is the store's record of a tracker event.
func newEvent(ev tracker.Event) store.NewEvent {
	out := store.NewEvent{Round: ev.Round, Kind: ev.Kind, Summary: ev.Summary, Payload: ev.Payload, PlayerSummary: ev.PlayerSummary}
	if len(ev.PlayerSpotted) > 0 {
		out.PlayerSpotted, _ = json.Marshal(ev.PlayerSpotted) // plain strings: cannot fail
	}
	if r := ev.PlayerRetract; r != nil {
		out.RetractSpotted = &store.SpottedRef{ID: r.ID, Map: r.Map}
	}
	return out
}

func (s *Server) playerView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ss, state, err := s.loadSessionState(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	events, err := s.playerFeed(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	cat, err := s.campaignCatalog(r.Context(), ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, PlayerResponse{State: tracker.PlayerView(state), Events: events, EventSeq: ss.EventSeq, Catalog: playerCatalog(cat)})
}

func (s *Server) playerStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.store.GetSession(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	s.playerStreams.add(id, conn)
	defer s.playerStreams.remove(id, conn)
	s.listen(r.Context(), conn)
}
