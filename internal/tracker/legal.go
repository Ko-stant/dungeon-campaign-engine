package tracker

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// Action is a command an actor may send right now, with a label for a
// button. Applying it succeeds (the property test in legal_test.go holds
// LegalActions and Apply to that): both use the same checks.
type Action struct {
	Label   string  `json:"label"`
	Command Command `json:"command"`
}

// LegalActions lists what an actor may do now in rules mode: a player the
// turn commands of their hero (starting a turn, rolling, every square they
// can move to, doors, attacks, searches, disarming, leaving, ending the
// turn); the GM or AI GM the phase changes and, in the monsters' phase,
// every monster's moves and attacks. GM overrides (the table commands,
// quest.end) are never listed. It never changes the state.
func LegalActions(s *State, actor Actor, catalog *content.Catalog) []Action {
	if s.Rules == nil || s.Rules.Phase == PhaseOver {
		return nil
	}
	l := legal{a: &applier{s: s, catalog: catalog, actor: actor}, actor: actor}
	if actor.Player() {
		l.hero()
	} else {
		l.gm()
	}
	return l.out
}

type legal struct {
	a     *applier
	actor Actor
	out   []Action
}

func (l *legal) add(label, kind string, payload map[string]any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	l.out = append(l.out, Action{Label: label, Command: Command{Type: kind, Payload: data, Actor: l.actor}})
}

// sortedTiles orders squares from the bottom row up, left to right.
func sortedTiles(set map[maps.Tile]int) []maps.Tile {
	out := make([]maps.Tile, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b maps.Tile) int {
		if a.Y != b.Y {
			return a.Y - b.Y
		}
		return a.X - b.X
	})
	return out
}

func tileJSON(t maps.Tile) map[string]int { return map[string]int{"x": t.X, "y": t.Y} }

func (l *legal) hero() {
	a, s, r := l.a, l.a.s, l.a.s.Rules
	h, err := a.hero(l.actor.HeroID)
	if err != nil {
		return
	}
	if r.Turn == nil {
		if r.Phase == PhaseHeroes && h.Status == HeroActive && h.Placed && !slices.Contains(r.Acted, h.ID) {
			l.add("Start "+h.Name+"'s turn", "turn.start", map[string]any{"hero": h.ID})
		}
		return
	}
	if r.Turn.HeroID != h.ID {
		return
	}
	turn := r.Turn
	if !turn.MoveDone && turn.MoveRoll == 0 {
		l.add("Roll for movement", "turn.roll-move", map[string]any{})
	}
	if !turn.MoveDone && turn.MoveRoll > 0 && turn.MoveLeft > 0 {
		start := maps.Tile{X: h.X, Y: h.Y}
		for _, t := range sortedTiles(maps.Reachable(a.terrain(), a.heroMove(h, turn.MoveLeft, nil))) {
			if t != start {
				l.add(fmt.Sprintf("Move to (%d,%d)", t.X, t.Y), "turn.move", map[string]any{"to": tileJSON(t)})
			}
		}
	}
	if !turn.MoveDone {
		for _, d := range s.Doors {
			if door, key, err := a.checkDoor(h, d.ID); err == nil {
				label := "Open " + a.doorLabel(door.ID)
				if key != nil {
					label = fmt.Sprintf("Unlock %s with the %s", a.doorLabel(door.ID), key.Name)
				}
				l.add(label, "turn.door", map[string]any{"door": door.ID})
			}
		}
	}
	if !turn.Acted {
		for i := range s.Monsters {
			m := &s.Monsters[i]
			if m.Alive && m.Visibility == MonsterSeen && a.checkAttack(h, m) == nil {
				l.add("Attack "+monsterLabel(m), "turn.attack", map[string]any{"target": m.ID})
			}
		}
		if !s.revealedMonster() {
			l.add("Search for traps", "turn.search", map[string]any{"kind": SearchTraps})
			l.add("Search for secret doors", "turn.search", map[string]any{"kind": SearchDoors})
			for _, f := range s.Quest.Furniture {
				if _, _, err := a.checkTreasure(h, f.ID); err == nil {
					l.add("Search the "+a.furnitureLabel(f)+" for treasure", "turn.search", map[string]any{"kind": SearchTreasure, "furniture": f.ID})
				}
			}
		}
		for _, t := range s.Traps {
			if _, qt, _, err := a.checkDisarm(h, t.ID); err == nil {
				l.add("Disarm "+a.trapKindLabel(qt)+" "+t.ID, "turn.disarm", map[string]any{"trap": t.ID})
			}
		}
	}
	if slices.Contains(s.Quest.ExitTiles, maps.Tile{X: h.X, Y: h.Y}) {
		l.add("Leave by the exit", "turn.exit", map[string]any{})
	}
	l.add("End the turn", "turn.end", map[string]any{})
}

func (l *legal) gm() {
	a, s, r := l.a, l.a.s, l.a.s.Rules
	if r.Turn != nil {
		return
	}
	if r.Phase == PhaseHeroes {
		l.add("The monsters' turn", "phase.monsters", map[string]any{})
		l.add("End the round", "phase.end", map[string]any{})
		return
	}
	for i := range s.Monsters {
		m := &s.Monsters[i]
		if !m.Alive {
			continue
		}
		if !slices.Contains(r.MonstersMoved, m.ID) && m.Movement > 0 {
			start := maps.Tile{X: m.X, Y: m.Y}
			for _, t := range sortedTiles(maps.Reachable(a.terrain(), a.monsterMoveSpec(m, m.Movement))) {
				if t != start {
					l.add(fmt.Sprintf("%s: move to (%d,%d)", monsterLabel(m), t.X, t.Y), "monster.move", map[string]any{"monster": m.ID, "to": tileJSON(t)})
				}
			}
		}
		if !slices.Contains(r.MonstersActed, m.ID) {
			for j := range s.Heroes {
				if h := &s.Heroes[j]; a.checkMonsterAttack(m, h) == nil {
					l.add(fmt.Sprintf("%s: attack %s", monsterLabel(m), h.Name), "monster.attack", map[string]any{"monster": m.ID, "target": h.ID})
				}
			}
		}
	}
	l.add("End the round", "phase.end", map[string]any{})
}
