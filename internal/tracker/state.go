// Package tracker is the table-companion game model: the complete state of a
// quest being played at the table, and the GM commands that change it.
//
// Nothing here enforces HeroQuest rules. The GM may move anything anywhere,
// re-close doors, un-trigger traps or revive monsters; the tracker only records
// the outcome and describes each change in a readable event.
package tracker

import (
	"fmt"
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// StateVersion is the session state schema version.
const StateVersion = 1

// Hero statuses.
const (
	HeroActive  = "active"
	HeroDead    = "dead"
	HeroEscaped = "escaped"
)

// Monster visibility: hidden monsters are known to the GM only.
const (
	MonsterHidden = "hidden"
	MonsterSeen   = "seen"
)

// CampaignHero is a hero as kept on the campaign between quests.
type CampaignHero struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Player    string `json:"player,omitempty"`
	Class     string `json:"class"`
	Gold      int    `json:"gold"`
	Equipment string `json:"equipment,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

// Hero is a hero during a session.
type Hero struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Player    string `json:"player,omitempty"`
	Class     string `json:"class"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Placed    bool   `json:"placed"`
	Body      int    `json:"body"`
	MaxBody   int    `json:"maxBody"`
	Mind      int    `json:"mind"`
	MaxMind   int    `json:"maxMind"`
	Gold      int    `json:"gold"`
	Equipment string `json:"equipment,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Status    string `json:"status"`
}

// Monster is a monster during a session.
type Monster struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Body       int    `json:"body"`
	MaxBody    int    `json:"maxBody"`
	Mind       int    `json:"mind"`
	Visibility string `json:"visibility"`
	Alive      bool   `json:"alive"`
	Notes      string `json:"notes,omitempty"`
}

// DoorState is the live state of a quest door.
type DoorState struct {
	ID    string `json:"id"`
	State string `json:"state"`
	// Found is false for a secret door the heroes have not discovered yet.
	Found bool `json:"found"`
}

// TrapState is the live state of a quest trap.
type TrapState struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

// State is everything needed to resume a session. Board and Quest are frozen
// copies taken at session start, so later edits in the map creator never
// change a game in progress.
type State struct {
	Version       int         `json:"version"`
	Board         maps.Board  `json:"board"`
	Quest         maps.Quest  `json:"quest"`
	QuestName     string      `json:"questName"`
	Round         int         `json:"round"`
	Heroes        []Hero      `json:"heroes"`
	Monsters      []Monster   `json:"monsters"`
	Doors         []DoorState `json:"doors"`
	Traps         []TrapState `json:"traps"`
	ConsumedNotes []string    `json:"consumedNotes"`
	Discovered    []int       `json:"discovered"`
}

// NewSession sets up round 1 of a quest: heroes on the start squares in
// order (extra heroes wait off the board), monsters hidden with catalog stats
// or quest overrides, doors and traps as placed, and the starting areas
// discovered.
func NewSession(board *maps.Board, quest *maps.Quest, questName string, party []CampaignHero, catalog *content.Catalog) (*State, error) {
	s := &State{
		Version:       StateVersion,
		Board:         *board,
		Quest:         *quest,
		QuestName:     questName,
		Round:         1,
		Heroes:        []Hero{},
		Monsters:      []Monster{},
		Doors:         []DoorState{},
		Traps:         []TrapState{},
		ConsumedNotes: []string{},
		Discovered:    []int{},
	}

	for i, ch := range party {
		class, ok := catalog.Hero(ch.Class)
		if !ok {
			return nil, fmt.Errorf("hero %q has unknown class %q", ch.Name, ch.Class)
		}
		h := Hero{
			ID: ch.ID, Name: ch.Name, Player: ch.Player, Class: ch.Class,
			Body: class.Body, MaxBody: class.Body, Mind: class.Mind, MaxMind: class.Mind,
			Gold: ch.Gold, Equipment: ch.Equipment, Notes: ch.Notes, Status: HeroActive,
		}
		if i < len(quest.StartTiles) {
			h.X, h.Y, h.Placed = quest.StartTiles[i].X, quest.StartTiles[i].Y, true
		}
		s.Heroes = append(s.Heroes, h)
	}

	for _, qm := range quest.Monsters {
		m := Monster{ID: qm.ID, Type: qm.Type, Name: qm.Type, X: qm.X, Y: qm.Y, Visibility: MonsterHidden, Alive: true, Notes: qm.Notes}
		if def, ok := catalog.Monster(qm.Type); ok {
			m.Name, m.Body, m.Mind = def.Name, def.Body, def.Mind
		}
		if qm.Body != nil {
			m.Body = *qm.Body
		}
		if qm.Mind != nil {
			m.Mind = *qm.Mind
		}
		m.MaxBody = m.Body
		s.Monsters = append(s.Monsters, m)
	}

	for _, d := range quest.Doors {
		s.Doors = append(s.Doors, DoorState{ID: d.ID, State: d.State, Found: d.Kind != maps.DoorSecret})
	}
	for _, t := range quest.Traps {
		s.Traps = append(s.Traps, TrapState{ID: t.ID, State: t.State})
	}

	discovered := map[int]bool{}
	for _, st := range quest.StartTiles {
		for _, i := range s.areaTiles(st.X, st.Y) {
			discovered[i] = true
		}
	}
	s.Discovered = sortedKeys(discovered)
	return s, nil
}

// areaTiles returns the tile indexes of the room containing (x, y), or just
// that tile when it is corridor, rock or off the board.
func (s *State) areaTiles(x, y int) []int {
	b := &s.Board
	if x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return nil
	}
	region := b.RegionAt(x, y)
	if region <= maps.Corridor {
		return []int{y*b.Width + x}
	}
	var out []int
	for i, r := range b.Regions {
		if r == region {
			out = append(out, i)
		}
	}
	return out
}

func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// CarryOver returns the campaign's heroes updated with what they carry out of
// this session: gold, equipment and notes. Heroes not in the session are
// returned unchanged. Body and mind are not carried; heroes heal between quests.
func (s *State) CarryOver(campaign []CampaignHero) []CampaignHero {
	out := slices.Clone(campaign)
	for i := range out {
		for _, h := range s.Heroes {
			if h.ID == out[i].ID {
				out[i].Gold = h.Gold
				out[i].Equipment = h.Equipment
				out[i].Notes = h.Notes
			}
		}
	}
	return out
}
