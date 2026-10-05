package app

import (
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
// when the players hear nothing about it.
type PlayerUpdate struct {
	State    tracker.PlayerState  `json:"state"`
	Event    *PlayerEventResponse `json:"event"`
	EventSeq int64                `json:"eventSeq"`
}

func playerEventResponse(e store.Event) PlayerEventResponse {
	return PlayerEventResponse{Seq: e.Seq, Round: e.Round, Summary: e.PlayerSummary, CreatedAt: e.CreatedAt}
}

func (s *Server) playerView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ss, state, err := s.loadSessionState(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	events, err := s.store.ListPlayerEvents(r.Context(), id, playerFeedLength)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	cat, err := s.campaignCatalog(r.Context(), ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	resp := PlayerResponse{State: tracker.PlayerView(state), Events: make([]PlayerEventResponse, 0, len(events)), EventSeq: ss.EventSeq, Catalog: playerCatalog(cat)}
	for _, e := range events {
		resp.Events = append(resp.Events, playerEventResponse(e))
	}
	writeJSON(w, http.StatusOK, resp)
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
	// The screen only listens; read until it goes away so close frames are handled.
	for {
		if _, _, err := conn.Read(r.Context()); err != nil {
			return
		}
	}
}
