package tracker

import (
	"slices"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// searchState: hallBoard without monsters, Grom at (4,1) beside the chest
// (5,1) in room 2, Ilsa in room 1; the rules on and Grom's turn under way.
func searchState(t *testing.T, change func(q *maps.Quest)) *State {
	t.Helper()
	b, q, cat := hallBoard()
	q.Monsters = nil
	q.StartTiles = []maps.Tile{{X: 4, Y: 1}, {X: 1, Y: 1}}
	if change != nil {
		change(q)
	}
	s, err := NewSession(b, q, "Hall", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	s, _ = hallApply(t, s, cmd(t, "rules.enable", map[string]any{"seed": 5}))
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"})))
	return s
}

func search(t *testing.T, hero, kind, furniture string) Command {
	p := map[string]any{"kind": kind}
	if furniture != "" {
		p["furniture"] = furniture
	}
	return as(seat(hero), cmd(t, "turn.search", p))
}

func TestNoSearchingWhileAMonsterIsRevealed(t *testing.T) {
	s := searchState(t, func(q *maps.Quest) {
		q.Monsters = []maps.Monster{{ID: "orc", Type: "orc", X: 4, Y: 3}}
	})
	if !s.Fight {
		t.Fatal("the orc is in view")
	}
	for _, kind := range []string{"treasure", "traps", "doors"} {
		if msg := hallErr(t, s, search(t, "hero-1", kind, "chest")); !strings.Contains(msg, "combat") {
			t.Errorf("%s: %q", kind, msg)
		}
	}
}

func TestTreasureIsSearchedOncePerPiecePerParty(t *testing.T) {
	s := searchState(t, nil)
	s, ev := hallApply(t, s, search(t, "hero-1", "treasure", "chest"))
	if !slices.Contains(s.Rules.Searched, "chest") || !s.Rules.Turn.Acted {
		t.Fatalf("rules %+v", s.Rules)
	}
	if ev.Summary != "Grom (Barbarian) searches the Chest (chest) for treasure" {
		t.Errorf("summary %q", ev.Summary)
	}
	if msg := hallErr(t, s, search(t, "hero-1", "traps", "")); !strings.Contains(msg, "already acted") {
		t.Errorf("a second search in one turn: %q", msg)
	}
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	s.Heroes[1].X, s.Heroes[1].Y = 5, 2
	s, _ = hallApply(t, s, as(seat("hero-2"), cmd(t, "turn.start", map[string]any{"hero": "hero-2"})))
	if msg := hallErr(t, s, search(t, "hero-2", "treasure", "chest")); !strings.Contains(msg, "already been searched") {
		t.Errorf("the party searched it already: %q", msg)
	}
}

func TestTreasureSearchNeedsTheHeroBesideThePiece(t *testing.T) {
	s := searchState(t, nil)
	s.Heroes[0].X, s.Heroes[0].Y = 4, 3
	if msg := hallErr(t, s, search(t, "hero-1", "treasure", "chest")); !strings.Contains(msg, "beside") {
		t.Errorf("from across the room: %q", msg)
	}
	if msg := hallErr(t, s, search(t, "hero-1", "treasure", "")); !strings.Contains(msg, "which") {
		t.Errorf("no piece named: %q", msg)
	}
	if msg := hallErr(t, s, search(t, "hero-1", "gold", "chest")); !strings.Contains(msg, "treasure, traps or doors") {
		t.Errorf("an unknown kind: %q", msg)
	}
}

func TestATrappedChestGoesOffWhenSearched(t *testing.T) {
	s := searchState(t, func(q *maps.Quest) {
		q.Traps = append(q.Traps, maps.Trap{ID: "needle", Kind: "pit", X: 5, Y: 1, FurnitureID: "chest", State: maps.TrapHidden})
		q.Notes = []maps.Note{{ID: "note-A", Label: "A", X: 5, Y: 1, Text: "50 gold"}}
	})
	s, ev := hallApply(t, s, search(t, "hero-1", "treasure", "chest"))
	i := slices.IndexFunc(s.Traps, func(t TrapState) bool { return t.ID == "needle" })
	if s.Traps[i].State != maps.TrapTriggered {
		t.Fatalf("trap %+v", s.Traps[i])
	}
	want := "Grom (Barbarian) searches the Chest (chest) for treasure; Pit Trap needle triggered: it stays on the board; note A is here"
	if ev.Summary != want {
		t.Errorf("summary\n %q\nwant\n %q", ev.Summary, want)
	}
}

func TestSearchingForTrapsFindsThoseInView(t *testing.T) {
	s := searchState(t, func(q *maps.Quest) {
		q.Traps = append(q.Traps,
			maps.Trap{ID: "room-pit", Kind: "pit", X: 4, Y: 3, State: maps.TrapHidden},
			maps.Trap{ID: "marker", Kind: maps.TrapTrigger, X: 4, Y: 2, State: maps.TrapHidden},
		)
	})
	s, ev := hallApply(t, s, search(t, "hero-1", "traps", ""))
	state := func(id string) string {
		return s.Traps[slices.IndexFunc(s.Traps, func(t TrapState) bool { return t.ID == id })].State
	}
	if state("room-pit") != maps.TrapRevealed || state("pit") != maps.TrapHidden || state("marker") != maps.TrapHidden {
		t.Fatalf("traps %+v", s.Traps)
	}
	if ev.Summary != "Grom (Barbarian) searches for traps: found Pit Trap room-pit" {
		t.Errorf("summary %q", ev.Summary)
	}
	s = searchState(t, nil)
	if _, ev := hallApply(t, s, search(t, "hero-1", "traps", "")); ev.Summary != "Grom (Barbarian) searches for traps: found none" {
		t.Errorf("summary %q", ev.Summary)
	}
}

func TestSearchingForSecretDoors(t *testing.T) {
	s := searchState(t, func(q *maps.Quest) {
		q.Doors[1].Kind = maps.DoorSecret
		q.StartTiles[0] = maps.Tile{X: 4, Y: 2}
	})
	if s.Doors[1].Found {
		t.Fatal("the secret door starts hidden")
	}
	s, ev := hallApply(t, s, search(t, "hero-1", "doors", ""))
	if !s.Doors[1].Found || !s.Doors[1].Seen {
		t.Fatalf("door %+v", s.Doors[1])
	}
	if ev.Summary != "Grom (Barbarian) searches for secret doors: found 1" {
		t.Errorf("summary %q", ev.Summary)
	}
}

// rogueState puts Ilsa (a Rogue here) beside a revealed pit, on her turn.
func rogueState(t *testing.T) *State {
	t.Helper()
	s := searchState(t, func(q *maps.Quest) {
		q.Traps[0].State = maps.TrapRevealed
	})
	s.Heroes[1].Class = "rogue"
	s.Heroes[1].X, s.Heroes[1].Y = 3, 2
	s.Doors[0].State = maps.DoorOpen
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	s, _ = hallApply(t, s, as(seat("hero-2"), cmd(t, "turn.start", map[string]any{"hero": "hero-2"})))
	return s
}

func disarm(t *testing.T, die int) Command {
	c := cmd(t, "turn.disarm", map[string]any{"trap": "pit"})
	c.Dice = []int{die}
	return c
}

func TestARogueDisarmsATrap(t *testing.T) {
	s, ev := hallApply(t, rogueState(t), disarm(t, 5))
	if s.Traps[0].State != maps.TrapDisarmed || !s.Rules.Turn.Acted {
		t.Fatalf("trap %+v", s.Traps[0])
	}
	if ev.Summary != "Ilsa (Rogue) disarms Pit Trap pit (1d8: 5)" {
		t.Errorf("summary %q", ev.Summary)
	}
}

func TestAFailedDisarmSetsTheTrapOff(t *testing.T) {
	s, ev := hallApply(t, rogueState(t), disarm(t, 1))
	if s.Traps[0].State != maps.TrapTriggered {
		t.Fatalf("trap %+v", s.Traps[0])
	}
	if ev.Summary != "Ilsa (Rogue) fails to disarm Pit Trap pit (1d8: 1); Pit Trap pit triggered: it stays on the board" {
		t.Errorf("summary %q", ev.Summary)
	}
}

func TestOnlyADisarmingClassDisarmsAKnownTrapBeside(t *testing.T) {
	s := rogueState(t)
	s.Heroes[1].Class = "wizard"
	if msg := hallErr(t, s, disarm(t, 5)); !strings.Contains(msg, "disarm") {
		t.Errorf("a class without disarm: %q", msg)
	}
	s = rogueState(t)
	s.Traps[0].State = maps.TrapHidden
	if msg := hallErr(t, s, disarm(t, 5)); !strings.Contains(msg, "no known trap") {
		t.Errorf("a hidden trap: %q", msg)
	}
	s = rogueState(t)
	s.Heroes[1].X, s.Heroes[1].Y = 3, 1
	if msg := hallErr(t, s, disarm(t, 5)); !strings.Contains(msg, "beside") {
		t.Errorf("two squares away: %q", msg)
	}
}
