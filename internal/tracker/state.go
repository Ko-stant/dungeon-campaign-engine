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

// StateVersion is the session state schema version. Version 2 uses the
// bottom-left, 1-based squares of maps.CurrentVersion 2.
const StateVersion = 2

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
	Equipment string `json:"equipment,omitempty"`
	Notes     string `json:"notes,omitempty"`
	// Items is the hero's inventory, carried between quests.
	Items []Item `json:"items,omitempty"`
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
	Equipment string `json:"equipment,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Status    string `json:"status"`
	Items     []Item `json:"items"`
	// Mana and abilities come from the hero's class when the session starts;
	// the session keeps its own copy so later class edits never change a game.
	// MaxMana is the class's; equipped items raise it (see ManaCap).
	Mana      int               `json:"mana,omitempty"`
	MaxMana   int               `json:"maxMana,omitempty"`
	Abilities []content.Ability `json:"abilities,omitempty"`
	// Cooldowns maps an ability id to the round it is ready again, for
	// abilities still cooling down.
	Cooldowns map[string]int `json:"cooldowns,omitempty"`
	// Combat is the class's combat stats, frozen like the abilities; nil for
	// built-in classes, which roll combat dice. Equipped items add to them
	// (see CombatTotals).
	Combat *Combat `json:"combat,omitempty"`
	// Effects are named conditions with optional countdowns (see fights.go).
	Effects []Effect `json:"effects,omitempty"`
}

// Combat is a hero's combat stats (see content.HeroDef and the Three Plagues rules).
type Combat struct {
	HitDice     string `json:"hitDice"`
	Accuracy    int    `json:"accuracy"`
	CritFrom    int    `json:"critFrom"`
	Damage      int    `json:"damage"`
	DefenseDice string `json:"defenseDice"`
	Avoidance   int    `json:"avoidance"`
	Mitigation  int    `json:"mitigation"`
	ManaRegen   int    `json:"manaRegen,omitempty"`
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
	// Size and color are copied from the monster type when it enters the
	// session, so later edits to a custom monster never change a game.
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Color  string `json:"color,omitempty"`
	// Combat is the campaign's combat stats for this monster type, frozen when
	// the monster is set up or added; nil when the campaign has none.
	Combat *content.MonsterCombat `json:"combat,omitempty"`
	// Effects are named conditions with optional countdowns (see fights.go).
	Effects []Effect `json:"effects,omitempty"`
}

// DoorState is the live state of a quest door.
type DoorState struct {
	ID    string `json:"id"`
	State string `json:"state"`
	// Found is false for a secret door the heroes have not discovered yet.
	Found  bool `json:"found"`
	Locked bool `json:"locked"`
}

// TrapState is the live state of a quest trap. At is set once the trap has
// been moved during play (a rolling boulder); until then it is where the quest
// put it.
type TrapState struct {
	ID    string     `json:"id"`
	State string     `json:"state"`
	At    *maps.Tile `json:"at,omitempty"`
}

// State is everything needed to resume a session. Board and Quest are frozen
// copies taken at session start, so later edits in the map creator never
// change a game in progress.
type State struct {
	Version int `json:"version"`
	// QuestID identifies the active map's quest (empty on sessions started
	// before travel between maps existed; the server fills it in).
	QuestID   string     `json:"questId,omitempty"`
	Board     maps.Board `json:"board"`
	Quest     maps.Quest `json:"quest"`
	QuestName string     `json:"questName"`
	Round     int        `json:"round"`
	// Fight is true while a fight is on (see fights.go).
	Fight bool `json:"fight,omitempty"`
	// Gold is the party's purse, taken from the campaign when the session
	// starts and saved back when it is completed.
	Gold     int         `json:"gold"`
	Heroes   []Hero      `json:"heroes"`
	Monsters []Monster   `json:"monsters"`
	Doors    []DoorState `json:"doors"`
	Traps    []TrapState `json:"traps"`
	// RemovedBlocks lists quest blocked squares taken off the board during
	// play (for example a found secret door).
	RemovedBlocks []string `json:"removedBlocks"`
	ConsumedNotes []string `json:"consumedNotes"`
	Discovered    []int    `json:"discovered"`
	// OtherMaps holds every map the session has left, exactly as it was left,
	// so traveling back restores it. The fields above are the active map.
	OtherMaps []MapState `json:"otherMaps,omitempty"`
	// ReadPassages lists the read-aloud passages (ids from the campaign's
	// script) already read at the table, across every map of the session.
	ReadPassages []string `json:"readPassages,omitempty"`
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
		RemovedBlocks: []string{},
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
			Equipment: ch.Equipment, Notes: ch.Notes, Status: HeroActive,
			Items: slices.Clone(ch.Items), MaxMana: class.Mana, Abilities: slices.Clone(class.Abilities),
			Combat: ClassCombat(class),
		}
		if h.Items == nil {
			h.Items = []Item{}
		}
		h.Mana = h.ManaCap()
		if i < len(quest.StartTiles) {
			h.X, h.Y, h.Placed = quest.StartTiles[i].X, quest.StartTiles[i].Y, true
		}
		s.Heroes = append(s.Heroes, h)
	}

	s.setUpMap(catalog)
	return s, nil
}

// ClassCombat is a class's combat stats as a session hero carries them (nil
// for classes without them: built-in classes roll combat dice).
func ClassCombat(class content.HeroDef) *Combat {
	if !class.Custom || class.AttackDice == "" {
		return nil
	}
	crit := class.CritFrom
	if crit == 0 {
		crit = 20
	}
	return &Combat{
		HitDice: class.AttackDice, Accuracy: class.Accuracy, CritFrom: crit, Damage: class.Damage,
		DefenseDice: class.DefenseDice, Avoidance: class.Avoidance, Mitigation: class.Mitigation, ManaRegen: class.ManaRegen,
	}
}

// combatCopy is a monster's own copy of its catalog combat stats (nil for none).
func combatCopy(c *content.MonsterCombat) *content.MonsterCombat {
	if c == nil {
		return nil
	}
	out := *c
	return &out
}

// setUpMap fills the active map's live state from its frozen quest: monsters
// hidden with catalog stats or quest overrides, doors and traps as placed,
// nothing removed or consumed, and the starting areas discovered.
func (s *State) setUpMap(catalog *content.Catalog) {
	s.Monsters, s.Doors, s.Traps = []Monster{}, []DoorState{}, []TrapState{}
	s.RemovedBlocks, s.ConsumedNotes = []string{}, []string{}
	for _, qm := range s.Quest.Monsters {
		m := Monster{ID: qm.ID, Type: qm.Type, Name: qm.Type, X: qm.X, Y: qm.Y, Visibility: MonsterHidden, Alive: true, Notes: qm.Notes, Width: 1, Height: 1}
		if def, ok := catalog.Monster(qm.Type); ok {
			m.Name, m.Body, m.Mind, m.Color = def.Name, def.Body, def.Mind, def.Color
			m.Width, m.Height = def.Size()
			m.Combat = combatCopy(def.Combat)
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

	for _, d := range s.Quest.Doors {
		s.Doors = append(s.Doors, DoorState{ID: d.ID, State: d.State, Found: d.Kind != maps.DoorSecret, Locked: d.Locked})
	}
	for _, t := range s.Quest.Traps {
		s.Traps = append(s.Traps, TrapState{ID: t.ID, State: t.State})
	}

	discovered := map[int]bool{}
	for _, st := range s.Quest.StartTiles {
		for _, i := range s.areaTiles(st.X, st.Y) {
			discovered[i] = true
		}
	}
	s.Discovered = sortedKeys(discovered)
}

// areaTiles returns the tile indexes of the room containing (x, y), or just
// that tile when it is corridor, rock or off the board.
func (s *State) areaTiles(x, y int) []int {
	b := &s.Board
	if !b.OnBoard(x, y) {
		return nil
	}
	region := b.RegionAt(x, y)
	if region <= maps.Corridor {
		return []int{b.Index(x, y)}
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
// this session: items, equipment and notes (the party's gold is State.Gold).
// Heroes not in the session are returned unchanged. Body and mind are not
// carried; heroes heal between quests.
func (s *State) CarryOver(campaign []CampaignHero) []CampaignHero {
	out := slices.Clone(campaign)
	for i := range out {
		for _, h := range s.Heroes {
			if h.ID == out[i].ID {
				out[i].Equipment = h.Equipment
				out[i].Notes = h.Notes
				out[i].Items = slices.Clone(h.Items)
			}
		}
	}
	return out
}
