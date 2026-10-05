package tracker

import (
	"slices"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// trapState is a session with one trap of each kind that behaves differently when triggered.
func trapState(t *testing.T) (*State, *content.Catalog) {
	t.Helper()
	b, q, cat := fixture()
	q.Traps = append(q.Traps,
		maps.Trap{ID: "trap-2", Kind: "falling_block", X: 3, Y: 3, State: maps.TrapHidden},
		maps.Trap{ID: "trap-3", Kind: "spear", X: 4, Y: 1, State: maps.TrapHidden},
		maps.Trap{ID: "trap-4", Kind: "boulder", X: 3, Y: 4, State: maps.TrapHidden},
		maps.Trap{ID: "trap-5", Kind: "teleport", X: 1, Y: 1, State: maps.TrapHidden},
		maps.Trap{ID: "trap-6", Kind: "long_pit", X: 2, Y: 1, Rotation: 90, State: maps.TrapHidden},
	)
	cat.Traps = []content.TrapDef{
		{ID: "pit", Name: "Pit Trap", Width: 1, Height: 1},
		{ID: "long_pit", Name: "Long Pit", Width: 2, Height: 1},
		{ID: "falling_block", Name: "Falling Block Trap", Width: 1, Height: 1},
		{ID: "spear", Name: "Spear Trap", Width: 1, Height: 1},
		{ID: "boulder", Name: "Boulder", Width: 1, Height: 1, Movable: true},
	}
	s, err := NewSession(b, q, "Traps", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	return s, cat
}

func trapLive(s *State, id string) TrapState {
	for _, t := range s.Traps {
		if t.ID == id {
			return t
		}
	}
	return TrapState{}
}

func TestTrapTriggerFollowsTheTrapKind(t *testing.T) {
	s, cat := trapState(t)
	for _, tc := range []struct {
		id, state, summary string
		blocked            []maps.Rect
	}{
		{"trap-1", maps.TrapTriggered, "Pit Trap trap-1 triggered: it stays on the board", nil},
		{"trap-6", maps.TrapTriggered, "Long Pit trap-6 triggered: it stays on the board", nil},
		{"trap-2", maps.TrapRemoved, "Falling Block Trap trap-2 triggered: (3,3) is blocked now", []maps.Rect{{ID: "added-1", X: 3, Y: 3, W: 1, H: 1}}},
		{"trap-3", maps.TrapRemoved, "Spear Trap trap-3 triggered and is gone", nil},
		{"trap-4", maps.TrapTriggered, "Boulder trap-4 triggered: block the square where it stops", nil},
		{"trap-5", maps.TrapTriggered, "teleport trap-5 triggered", nil},
	} {
		next, ev := applyWith(t, s, cmd(t, "trap.trigger", map[string]any{"id": tc.id}), cat)
		if got := trapLive(next, tc.id).State; got != tc.state || ev.Summary != tc.summary {
			t.Errorf("%s: %s %q", tc.id, got, ev.Summary)
		}
		if !slices.Equal(next.AddedBlocks, tc.blocked) {
			t.Errorf("%s: added blocks %+v", tc.id, next.AddedBlocks)
		}
	}
	gone, _ := applyWith(t, s, cmd(t, "trap.trigger", map[string]any{"id": "trap-3"}), cat)
	if _, _, err := Apply(gone, cmd(t, "trap.trigger", map[string]any{"id": "trap-3"}), cat); err == nil {
		t.Error("a removed trap cannot be triggered")
	}
	if _, _, err := Apply(s, cmd(t, "trap.trigger", map[string]any{"id": "trap-9"}), cat); err == nil {
		t.Error("unknown trap")
	}
}

func TestTrapBlockTurnsATrapIntoBlockedSquares(t *testing.T) {
	s, cat := trapState(t)
	// The boulder rolls to (5,3) and stops there.
	s, _ = applyWith(t, s, cmd(t, "trap.trigger", map[string]any{"id": "trap-4"}), cat)
	s, _ = applyWith(t, s, cmd(t, "move", map[string]any{"id": "trap-4", "x": 5, "y": 3}), cat)
	s, ev := applyWith(t, s, cmd(t, "trap.block", map[string]any{"id": "trap-4"}), cat)
	if trapLive(s, "trap-4").State != maps.TrapRemoved || !slices.Equal(s.AddedBlocks, []maps.Rect{{ID: "added-1", X: 5, Y: 3, W: 1, H: 1}}) ||
		ev.Summary != "Boulder trap-4 turned into blocked squares at (5,3)" {
		t.Fatalf("boulder: %+v %q", s.AddedBlocks, ev.Summary)
	}
	// A rotated long pit blocks its rotated footprint.
	s, _ = applyWith(t, s, cmd(t, "trap.block", map[string]any{"id": "trap-6"}), cat)
	if got := s.AddedBlocks[1]; got != (maps.Rect{ID: "added-2", X: 2, Y: 1, W: 1, H: 2}) {
		t.Fatalf("long pit: %+v", got)
	}
	if _, _, err := Apply(s, cmd(t, "trap.block", map[string]any{"id": "trap-4"}), cat); err == nil {
		t.Error("a removed trap cannot be turned into blocked squares")
	}
}

func TestBlockAddAndRemove(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "block.add", map[string]any{"x": 2, "y": 2}))
	if !slices.Equal(s.AddedBlocks, []maps.Rect{{ID: "added-1", X: 2, Y: 2, W: 1, H: 1}}) || ev.Summary != "Blocked (2,2)" {
		t.Fatalf("add: %+v %q", s.AddedBlocks, ev.Summary)
	}
	s, _ = apply(t, s, cmd(t, "block.add", map[string]any{"x": 3, "y": 2}))
	s, ev = apply(t, s, cmd(t, "block.remove", map[string]any{"id": "added-1"}))
	if len(s.AddedBlocks) != 1 || s.AddedBlocks[0].ID != "added-2" || ev.Summary != "Cleared the blocked square at (2,2)" {
		t.Fatalf("remove: %+v %q", s.AddedBlocks, ev.Summary)
	}
	// New ids follow the highest one.
	s, _ = apply(t, s, cmd(t, "block.add", map[string]any{"x": 2, "y": 2}))
	if s.AddedBlocks[1].ID != "added-3" {
		t.Fatalf("next id: %+v", s.AddedBlocks)
	}
	for name, c := range map[string]Command{
		"off the board": cmd(t, "block.add", map[string]any{"x": 9, "y": 9}),
		"already there": cmd(t, "block.add", map[string]any{"x": 2, "y": 2}),
		"unknown block": cmd(t, "block.remove", map[string]any{"id": "added-9"}),
		"a quest block": cmd(t, "block.remove", map[string]any{"id": "blocked-2"}),
	} {
		if _, _, err := Apply(s, c, nil); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestTravelKeepsEachMapsAddedBlocks(t *testing.T) {
	_, _, cat := fixture()
	s := travelState(t)
	s, _ = apply(t, s, cmd(t, "block.add", map[string]any{"x": 2, "y": 2}))
	lb, lq := lowerMap()
	away, _, err := Travel(s, Destination{QuestID: "quest-lower", QuestName: "Lower Vaults", Board: lb, Quest: lq}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(away.AddedBlocks) != 0 {
		t.Fatalf("a new map has no added blocks: %+v", away.AddedBlocks)
	}
	back, _, _ := Travel(away, Destination{QuestID: "quest-upper"}, cat)
	if len(back.AddedBlocks) != 1 {
		t.Fatalf("returning restores them: %+v", back.AddedBlocks)
	}
}
