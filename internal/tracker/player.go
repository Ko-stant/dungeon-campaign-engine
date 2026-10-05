package tracker

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// The player screen sees a filtered copy of the session (PlayerView) and a
// player-safe line for each event (PlayerSummary). Everything here works
// from an allow-list: a field or command not named here never reaches the
// players. GM-only things that must stay out: quest notes, monster notes,
// hidden monsters, unseen pieces, hidden traps and trap labels, room names,
// the read-aloud script and the GM's own log notes.

// PlayerState is what the player screen shows.
type PlayerState struct {
	QuestName string `json:"questName"`
	// Goal is the quest's aim as the players hear it.
	Goal  string `json:"goal,omitempty"`
	Round int    `json:"round"`
	Fight bool   `json:"fight,omitempty"`
	// The map layout (rooms and corridors, walls); room names are left out.
	Width      int         `json:"width"`
	Height     int         `json:"height"`
	Regions    []int       `json:"regions"`
	DrawnWalls []maps.Edge `json:"drawnWalls,omitempty"`
	// Discovered squares are drawn in the "seen" color.
	Discovered []int            `json:"discovered"`
	Doors      []PlayerDoor     `json:"doors"`
	Furniture  []maps.Furniture `json:"furniture"`
	Blocks     []PlayerBlock    `json:"blocks"`
	Traps      []PlayerTrap     `json:"traps"`
	Monsters   []PlayerMonster  `json:"monsters"`
	Heroes     []PlayerHero     `json:"heroes"`
	// ShowMonsterBody is false when the GM keeps monsters' Body off the screen.
	ShowMonsterBody bool `json:"showMonsterBody"`
}

// PlayerDoor is a door the players were shown; a found secret door is a plain door.
type PlayerDoor struct {
	ID     string    `json:"id"`
	Edge   maps.Edge `json:"edge"`
	Span   int       `json:"span,omitempty"`
	Kind   string    `json:"kind"`
	State  string    `json:"state"`
	Locked bool      `json:"locked,omitempty"`
}

// PlayerBlock is a rectangle of blocked squares.
type PlayerBlock struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
	W  int    `json:"w"`
	H  int    `json:"h"`
}

// PlayerTrap is a revealed, triggered or disarmed trap, where it is now.
type PlayerTrap struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Rotation int    `json:"rotation,omitempty"`
	State    string `json:"state"`
}

// PlayerMonster is a living monster the players were shown. Body and MaxBody
// are nil while the GM hides monsters' Body.
type PlayerMonster struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Name    string                 `json:"name"`
	X       int                    `json:"x"`
	Y       int                    `json:"y"`
	Width   int                    `json:"width"`
	Height  int                    `json:"height"`
	Color   string                 `json:"color,omitempty"`
	Body    *int                   `json:"body,omitempty"`
	MaxBody *int                   `json:"maxBody,omitempty"`
	Wounded bool                   `json:"wounded,omitempty"`
	Combat  *content.MonsterCombat `json:"combat,omitempty"`
	Effects []Effect               `json:"effects,omitempty"`
}

// PlayerHero is a hero for the party panel and the board.
type PlayerHero struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Class   string   `json:"class"`
	X       int      `json:"x"`
	Y       int      `json:"y"`
	Placed  bool     `json:"placed"`
	Body    int      `json:"body"`
	MaxBody int      `json:"maxBody"`
	Mind    int      `json:"mind"`
	MaxMind int      `json:"maxMind"`
	Mana    int      `json:"mana,omitempty"`
	ManaCap int      `json:"manaCap,omitempty"`
	Status  string   `json:"status"`
	Effects []Effect `json:"effects,omitempty"`
	// Combat is the hero's totals (class plus equipped items).
	Combat *Combat `json:"combat,omitempty"`
	// Determination is the hero's Accuracy bonus from misses in a row.
	Determination int `json:"determination,omitempty"`
}

// wounded reports a living monster that is hurt and at or below a quarter of
// its Body (Faltering), which the players can see.
func wounded(m *Monster) bool {
	return m.Body > 0 && m.Body < m.MaxBody && m.Body <= max(1, m.MaxBody/4)
}

// PlayerView filters a session down to what the players may see.
func PlayerView(s *State) PlayerState {
	pv := PlayerState{
		QuestName: s.QuestName, Goal: s.Quest.Goal, Round: s.Round, Fight: s.Fight,
		Width: s.Board.Width, Height: s.Board.Height, Regions: slices.Clone(s.Board.Regions), DrawnWalls: slices.Clone(s.Board.DrawnWalls),
		Discovered: slices.Clone(s.Discovered),
		Doors:      []PlayerDoor{}, Furniture: []maps.Furniture{}, Blocks: []PlayerBlock{}, Traps: []PlayerTrap{},
		Monsters: []PlayerMonster{}, Heroes: []PlayerHero{},
		ShowMonsterBody: !s.Players.HideMonsterBody,
	}
	if pv.Discovered == nil {
		pv.Discovered = []int{}
	}
	for _, qd := range s.Quest.Doors {
		for _, d := range s.Doors {
			if d.ID != qd.ID || !d.Seen || !d.Found {
				continue
			}
			kind := qd.Kind
			if kind == maps.DoorSecret {
				kind = maps.DoorNormal
			}
			pv.Doors = append(pv.Doors, PlayerDoor{ID: d.ID, Edge: qd.Edge, Span: qd.Span, Kind: kind, State: d.State, Locked: d.Locked})
		}
	}
	for _, f := range s.Quest.Furniture {
		if slices.Contains(s.SeenFurniture, f.ID) {
			pv.Furniture = append(pv.Furniture, f)
		}
	}
	for _, r := range s.Quest.BlockedSquares {
		if slices.Contains(s.SeenBlocks, r.ID) && !slices.Contains(s.RemovedBlocks, r.ID) {
			pv.Blocks = append(pv.Blocks, PlayerBlock{ID: r.ID, X: r.X, Y: r.Y, W: r.W, H: r.H})
		}
	}
	for _, r := range s.AddedBlocks {
		pv.Blocks = append(pv.Blocks, PlayerBlock{ID: r.ID, X: r.X, Y: r.Y, W: r.W, H: r.H})
	}
	for i := range s.Traps {
		t := &s.Traps[i]
		switch t.State {
		case maps.TrapRevealed, maps.TrapTriggered, maps.TrapDisarmed:
		default:
			continue
		}
		for _, qt := range s.Quest.Traps {
			if qt.ID == t.ID && qt.Kind != maps.TrapTrigger {
				at := trapAt(t, qt)
				pv.Traps = append(pv.Traps, PlayerTrap{ID: t.ID, Kind: qt.Kind, X: at.X, Y: at.Y, Rotation: qt.Rotation, State: t.State})
			}
		}
	}
	for i := range s.Monsters {
		m := &s.Monsters[i]
		if !m.Alive || m.Visibility != MonsterSeen {
			continue
		}
		pm := PlayerMonster{
			ID: m.ID, Type: m.Type, Name: m.Name, X: m.X, Y: m.Y, Width: max(m.Width, 1), Height: max(m.Height, 1), Color: m.Color,
			Wounded: wounded(m), Combat: combatCopy(m.Combat), Effects: slices.Clone(m.Effects),
		}
		if !s.Players.HideMonsterBody {
			body, maxBody := m.Body, m.MaxBody
			pm.Body, pm.MaxBody = &body, &maxBody
		}
		pv.Monsters = append(pv.Monsters, pm)
	}
	for i := range s.Heroes {
		h := &s.Heroes[i]
		pv.Heroes = append(pv.Heroes, PlayerHero{
			ID: h.ID, Name: h.Name, Class: h.Class, X: h.X, Y: h.Y, Placed: h.Placed,
			Body: h.Body, MaxBody: h.MaxBody, Mind: h.Mind, MaxMind: h.MaxMind, Mana: h.Mana, ManaCap: h.ManaCap(),
			Status: h.Status, Effects: slices.Clone(h.Effects), Combat: h.CombatTotals(), Determination: h.Determination,
		})
	}
	return pv
}

// PlayerSummary is the player-safe line for a change (empty: the players
// hear nothing). It compares the state before and after the command;
// gmSummary is reused only for commands that concern the heroes alone.
func PlayerSummary(before, after *State, c Command, gmSummary string) string {
	switch c.Type {
	case "item.add", "item.remove", "item.give", "item.update", "item.equip", "item.use", "gold.set", "ability.use", "ability.reset":
		return gmSummary
	case "round.advance", "round.set":
		return fmt.Sprintf("Round %d", after.Round)
	case "fight.start":
		return "Fight!"
	case "fight.end":
		return "Fight over"
	case "hero.update":
		return heroChanges(before, after)
	case "door.set":
		return doorChanges(before, after)
	case "monster.update", "monster.add", "seen.set", "area.reveal", "tiles.reveal":
		return monsterChanges(before, after)
	case "effect.add", "effect.remove":
		return effectChanges(before, after)
	case "rules.enable":
		return "The game begins: the heroes' turns"
	case "phase.monsters":
		return "The monsters' turn"
	case "phase.end", "hero.join", "quest.end":
		return playerSafe(before, after, gmSummary)
	case "monster.move", "monster.attack":
		// A monster the heroes can't see acts unheard.
		var p struct {
			Monster string `json:"monster"`
		}
		_ = json.Unmarshal(c.Payload, &p) // the command was applied, so it decodes
		if m := findMonster(after, p.Monster); m == nil || m.Visibility != MonsterSeen {
			return ""
		}
		return playerSafe(before, after, gmSummary)
	}
	if strings.HasPrefix(c.Type, "turn.") {
		return playerSafe(before, after, gmSummary)
	}
	return ""
}

// notesMentioned matches the GM's reminder that a quest note is on a
// searched piece ("; note A is here").
var notesMentioned = regexp.MustCompile(`; note [^;]+ is here`)

// playerSafe is a rules command's GM line as the players may hear it: no
// quest notes, and monsters by name without their ids.
func playerSafe(before, after *State, line string) string {
	line = notesMentioned.ReplaceAllString(line, "")
	for _, st := range []*State{after, before} {
		for _, m := range st.Monsters {
			line = strings.ReplaceAll(line, monsterLabel(&m), m.Name)
		}
	}
	return line
}

func heroChanges(before, after *State) string {
	var parts []string
	for i := range after.Heroes {
		a := &after.Heroes[i]
		b := findHero(before, a.ID)
		if b == nil {
			continue
		}
		var changes []string
		stat := func(label string, from, to int) {
			if from != to {
				changes = append(changes, fmt.Sprintf("%s %d → %d", label, from, to))
			}
		}
		stat("Body", b.Body, a.Body)
		stat("Mind", b.Mind, a.Mind)
		stat("Mana", b.Mana, a.Mana)
		if len(changes) > 0 {
			parts = append(parts, a.Name+": "+strings.Join(changes, ", "))
		}
		if b.Status != a.Status {
			switch a.Status {
			case HeroDead:
				parts = append(parts, a.Name+" has fallen")
			case HeroEscaped:
				parts = append(parts, a.Name+" escaped")
			case HeroActive:
				parts = append(parts, a.Name+" is back in the fight")
			}
		}
	}
	return strings.Join(parts, "; ")
}

func findHero(s *State, id string) *Hero {
	for i := range s.Heroes {
		if s.Heroes[i].ID == id {
			return &s.Heroes[i]
		}
	}
	return nil
}

func findMonster(s *State, id string) *Monster {
	for i := range s.Monsters {
		if s.Monsters[i].ID == id {
			return &s.Monsters[i]
		}
	}
	return nil
}

func doorChanges(before, after *State) string {
	var parts []string
	for _, a := range after.Doors {
		if !a.Seen || !a.Found {
			continue
		}
		for _, b := range before.Doors {
			if b.ID != a.ID || b.State == a.State {
				continue
			}
			noun := "door"
			for _, qd := range after.Quest.Doors {
				if qd.ID == a.ID && qd.Kind == maps.DoorGate {
					noun = "gate"
				}
			}
			verb := "opened"
			if a.State == maps.DoorClosed {
				verb = "closed"
			}
			parts = append(parts, fmt.Sprintf("A %s %s", noun, verb))
		}
	}
	return strings.Join(parts, "; ")
}

// monsterChanges describes monsters the players can see: newly spotted ones,
// Body changes (without numbers while Body is hidden) and deaths.
func monsterChanges(before, after *State) string {
	var parts []string
	for i := range after.Monsters {
		a := &after.Monsters[i]
		b := findMonster(before, a.ID)
		seenNow := a.Visibility == MonsterSeen
		seenBefore := b != nil && b.Visibility == MonsterSeen
		if seenNow && a.Alive && !seenBefore {
			continue // spotted: see spottedMonsters
		}
		if !seenNow || b == nil || !seenBefore {
			continue
		}
		if b.Alive && !a.Alive {
			parts = append(parts, a.Name+" slain")
			continue
		}
		if b.Body != a.Body && a.Alive {
			line := fmt.Sprintf("%s: Body %d → %d", a.Name, b.Body, a.Body)
			if after.Players.HideMonsterBody {
				line = a.Name + " took damage"
				if a.Body > b.Body {
					line = a.Name + " recovered"
				}
			}
			if wounded(a) && !wounded(b) {
				line += " (Wounded)"
			}
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "; ")
}

// SpottedMonster is a monster the players just saw. Events keep sightings
// apart from the rest of the player line, so removing a monster added by
// mistake can take its sighting back (see PlayerRetract); Map is the quest of
// the map it was on, since monster ids repeat between maps.
type SpottedMonster struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Map  string `json:"map,omitempty"`
}

// PlayerLine is the player screen's line for an event: its sightings, then
// the rest ("Spotted: Goblin, Orc; Orc slain").
func PlayerLine(spotted []SpottedMonster, rest string) string {
	var parts []string
	if len(spotted) > 0 {
		names := make([]string, len(spotted))
		for i, m := range spotted {
			names[i] = m.Name
		}
		parts = append(parts, "Spotted: "+strings.Join(names, ", "))
	}
	if rest != "" {
		parts = append(parts, rest)
	}
	return strings.Join(parts, "; ")
}

// PlayerSpotted lists the living monsters that a command shows the players.
func PlayerSpotted(before, after *State, c Command) []SpottedMonster {
	switch c.Type {
	case "monster.update", "monster.add", "seen.set", "area.reveal", "tiles.reveal", "rules.enable", "turn.move", "turn.door", "monster.move":
	default:
		return nil
	}
	var out []SpottedMonster
	for i := range after.Monsters {
		a := &after.Monsters[i]
		b := findMonster(before, a.ID)
		if a.Visibility == MonsterSeen && a.Alive && (b == nil || b.Visibility != MonsterSeen) {
			out = append(out, SpottedMonster{ID: a.ID, Name: a.Name, Map: after.QuestID})
		}
	}
	return out
}

// PlayerRetract is the sighting a command takes back: removing a living
// monster means it was never there. A killed monster's sighting stays, even
// when its body is cleared away.
func PlayerRetract(before *State, c Command) *SpottedMonster {
	if c.Type != "monster.remove" {
		return nil
	}
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(c.Payload, &p); err != nil {
		return nil
	}
	m := findMonster(before, p.ID)
	if m == nil || !m.Alive {
		return nil
	}
	return &SpottedMonster{ID: m.ID, Name: m.Name, Map: before.QuestID}
}

// effectChanges names effects added to or removed from heroes and seen monsters.
func effectChanges(before, after *State) string {
	var parts []string
	describe := func(name string, was, now []Effect) {
		for _, e := range now {
			if !slices.ContainsFunc(was, func(w Effect) bool { return w.ID == e.ID }) {
				if e.Rounds > 0 {
					parts = append(parts, fmt.Sprintf("%s: %s (%d rounds)", name, e.Name, e.Rounds))
				} else {
					parts = append(parts, fmt.Sprintf("%s: %s", name, e.Name))
				}
			}
		}
		for _, e := range was {
			if !slices.ContainsFunc(now, func(n Effect) bool { return n.ID == e.ID }) {
				parts = append(parts, fmt.Sprintf("%s: %s ended", name, e.Name))
			}
		}
	}
	for i := range after.Heroes {
		if b := findHero(before, after.Heroes[i].ID); b != nil {
			describe(after.Heroes[i].Name, b.Effects, after.Heroes[i].Effects)
		}
	}
	for i := range after.Monsters {
		a := &after.Monsters[i]
		if b := findMonster(before, a.ID); b != nil && a.Visibility == MonsterSeen && b.Visibility == MonsterSeen {
			describe(a.Name, b.Effects, a.Effects)
		}
	}
	return strings.Join(parts, "; ")
}
