package tracker

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

func TestRulesCommandsTellThePlayersWhatHappened(t *testing.T) {
	s, ev := hallApply(t, duelState(t), attack(t, "hero-1", 15, 5))
	if ev.PlayerSummary != "Grom (Barbarian) attacks Orc: 18 vs 10 (1d20: 15, Accuracy +3; crit die 5): hit for 3 (Orc 9/12)" {
		t.Errorf("an attack, without the monster's id: %q", ev.PlayerSummary)
	}
	_, ev = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	if ev.PlayerSummary != "Grom (Barbarian) ends their turn" {
		t.Errorf("turn end: %q", ev.PlayerSummary)
	}
}

func TestTheGMsNotesStayOutOfThePlayersLines(t *testing.T) {
	s := searchState(t, func(q *maps.Quest) {
		q.Notes = []maps.Note{{ID: "note-A", Label: "A", X: 5, Y: 1, Text: "50 gold"}}
	})
	_, ev := hallApply(t, s, search(t, "hero-1", "treasure", "chest"))
	if !strings.Contains(ev.Summary, "note A") || strings.Contains(ev.PlayerSummary, "note") {
		t.Errorf("GM %q, players %q", ev.Summary, ev.PlayerSummary)
	}
}

func TestAHiddenMonsterMovesUnheard(t *testing.T) {
	s := monstersPhase(t)
	s.Monsters[0].Movement = 2
	s.Monsters[0].Visibility = MonsterHidden
	s.Heroes[0].X, s.Heroes[1].X = 1, 1 // nobody in room 2 to see it
	s.Heroes[0].Y, s.Heroes[1].Y = 2, 1
	_, ev := hallApply(t, s, cmd(t, "monster.move", map[string]any{"monster": "orc", "to": map[string]int{"x": 5, "y": 3}}))
	if ev.PlayerSummary != "" {
		t.Errorf("a hidden monster's move: %q", ev.PlayerSummary)
	}
	s = monstersPhase(t)
	_, ev = hallApply(t, s, monsterAttack(t, "hero-1", 6, 5, 3, 1, 1))
	if !strings.HasPrefix(ev.PlayerSummary, "Orc attacks Grom (Barbarian)") {
		t.Errorf("a seen monster's attack: %q", ev.PlayerSummary)
	}
	_, ev = hallApply(t, s, cmd(t, "phase.end", map[string]any{}))
	if !strings.HasPrefix(ev.PlayerSummary, "Round 2 begins") {
		t.Errorf("a new round: %q", ev.PlayerSummary)
	}
}

func TestThePlayersLinesNameNoPiecesByID(t *testing.T) {
	s := hallTurn(t, 3, 3)
	s, _ = hallApply(t, s, moveTo(t, 2, 2))
	if _, ev := hallApply(t, s, openDoor(t, "d1")); ev.PlayerSummary != "Grom (Barbarian) opens a door" {
		t.Errorf("a door: %q", ev.PlayerSummary)
	}
	s = searchState(t, nil)
	if _, ev := hallApply(t, s, search(t, "hero-1", "treasure", "chest")); ev.PlayerSummary != "Grom (Barbarian) searches the Chest for treasure" {
		t.Errorf("treasure: %q", ev.PlayerSummary)
	}
	s = searchState(t, func(q *maps.Quest) {
		q.Traps = append(q.Traps, maps.Trap{ID: "room-pit", Kind: "pit", X: 4, Y: 3, State: maps.TrapHidden})
	})
	if _, ev := hallApply(t, s, search(t, "hero-1", "traps", "")); ev.PlayerSummary != "Grom (Barbarian) searches for traps: found Pit Trap" {
		t.Errorf("traps: %q", ev.PlayerSummary)
	}
	if _, ev := hallApply(t, rogueState(t), disarm(t, 1)); ev.PlayerSummary != "Ilsa (Rogue) steps onto (3,3) and fails to disarm Pit Trap (1d8: 1); Pit Trap triggered: it stays on the board" {
		t.Errorf("a failed disarm: %q", ev.PlayerSummary)
	}
	s = searchState(t, func(q *maps.Quest) {
		q.Doors[1].Kind, q.Doors[1].Locked, q.Doors[1].Key = maps.DoorGate, true, "Iron Key"
		q.StartTiles[0] = maps.Tile{X: 4, Y: 2}
	})
	s.Heroes[0].Items = []Item{{Name: "Iron Key", Quantity: 1}}
	if _, ev := hallApply(t, s, openDoor(t, "d2")); ev.PlayerSummary != "Grom (Barbarian) unlocks a gate with the Iron Key and opens it" {
		t.Errorf("a gate: %q", ev.PlayerSummary)
	}
}
