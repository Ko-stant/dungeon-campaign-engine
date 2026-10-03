package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// MapState is a map the session has left: its frozen board and quest, its live
// pieces and discoveries, and where each hero stood.
type MapState struct {
	QuestID       string         `json:"questId"`
	QuestName     string         `json:"questName"`
	Board         maps.Board     `json:"board"`
	Quest         maps.Quest     `json:"quest"`
	Monsters      []Monster      `json:"monsters"`
	Doors         []DoorState    `json:"doors"`
	Traps         []TrapState    `json:"traps"`
	RemovedBlocks []string       `json:"removedBlocks"`
	ConsumedNotes []string       `json:"consumedNotes"`
	Discovered    []int          `json:"discovered"`
	HeroPositions []HeroPosition `json:"heroPositions"`
}

// HeroPosition is where a hero stood on a map.
type HeroPosition struct {
	ID     string `json:"id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Placed bool   `json:"placed"`
}

// Destination is the map to travel to. Board and Quest are needed only for a
// map the session has not visited yet.
type Destination struct {
	QuestID   string
	QuestName string
	Board     *maps.Board
	Quest     *maps.Quest
}

// Visited reports whether the session is on, or has been to, a quest's map.
func (s *State) Visited(questID string) bool {
	return s.QuestID == questID || slices.ContainsFunc(s.OtherMaps, func(m MapState) bool { return m.QuestID == questID })
}

// Travel moves the whole party to another map mid-game. The heroes keep their
// stats, gold, equipment and status, and the round carries on. The map being
// left is saved as it stands, so traveling back restores it; a new map starts
// fresh with the heroes on its start squares. Like Apply, it never modifies s.
func Travel(s *State, dest Destination, catalog *content.Catalog) (*State, Event, error) {
	if s.Version != StateVersion {
		return nil, Event{}, fmt.Errorf("session state version %d is not the current version %d; start a new session", s.Version, StateVersion)
	}
	if dest.QuestID == "" {
		return nil, Event{}, errors.New("no map to travel to")
	}
	if dest.QuestID == s.QuestID {
		return nil, Event{}, errors.New("the party is already on that map")
	}
	next, err := clone(s)
	if err != nil {
		return nil, Event{}, err
	}

	left := next.saveMap()
	i := slices.IndexFunc(next.OtherMaps, func(m MapState) bool { return m.QuestID == dest.QuestID })
	var summary string
	if i >= 0 {
		back := next.OtherMaps[i]
		next.OtherMaps = slices.Delete(next.OtherMaps, i, i+1)
		next.restoreMap(back)
		summary = "Returned to " + back.QuestName
	} else {
		if dest.Board == nil || dest.Quest == nil {
			return nil, Event{}, fmt.Errorf("map %q has not been visited; its board and quest are needed", dest.QuestID)
		}
		next.QuestID, next.QuestName = dest.QuestID, dest.QuestName
		next.Board, next.Quest = *dest.Board, *dest.Quest
		next.setUpMap(catalog)
		next.placeOnStartSquares()
		summary = "Traveled to " + dest.QuestName
	}
	next.OtherMaps = append(next.OtherMaps, left)

	payload, err := json.Marshal(map[string]any{"command": map[string]string{"questId": dest.QuestID}, "from": left.QuestID})
	if err != nil {
		return nil, Event{}, err
	}
	return next, Event{Round: next.Round, Kind: "map.travel", Summary: summary, Payload: payload}, nil
}

// saveMap captures the active map, including where each hero stands.
func (s *State) saveMap() MapState {
	m := MapState{
		QuestID: s.QuestID, QuestName: s.QuestName, Board: s.Board, Quest: s.Quest,
		Monsters: s.Monsters, Doors: s.Doors, Traps: s.Traps,
		RemovedBlocks: s.RemovedBlocks, ConsumedNotes: s.ConsumedNotes, Discovered: s.Discovered,
		HeroPositions: make([]HeroPosition, 0, len(s.Heroes)),
	}
	for _, h := range s.Heroes {
		m.HeroPositions = append(m.HeroPositions, HeroPosition{ID: h.ID, X: h.X, Y: h.Y, Placed: h.Placed})
	}
	return m
}

// restoreMap makes a saved map active again and puts the heroes back where
// they stood (heroes who were not there wait off the board).
func (s *State) restoreMap(m MapState) {
	s.QuestID, s.QuestName, s.Board, s.Quest = m.QuestID, m.QuestName, m.Board, m.Quest
	s.Monsters, s.Doors, s.Traps = m.Monsters, m.Doors, m.Traps
	s.RemovedBlocks, s.ConsumedNotes, s.Discovered = m.RemovedBlocks, m.ConsumedNotes, m.Discovered
	for i := range s.Heroes {
		h := &s.Heroes[i]
		h.X, h.Y, h.Placed = 0, 0, false
		for _, p := range m.HeroPositions {
			if p.ID == h.ID {
				h.X, h.Y, h.Placed = p.X, p.Y, p.Placed
			}
		}
	}
}

// placeOnStartSquares puts the heroes on the active map's start squares in
// order; heroes beyond the squares wait off the board for the GM to place.
func (s *State) placeOnStartSquares() {
	for i := range s.Heroes {
		h := &s.Heroes[i]
		h.X, h.Y, h.Placed = 0, 0, false
		if i < len(s.Quest.StartTiles) {
			h.X, h.Y, h.Placed = s.Quest.StartTiles[i].X, s.Quest.StartTiles[i].Y, true
		}
	}
}
