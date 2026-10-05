package tracker

import (
	"slices"
	"testing"
)

func TestSeenSetShowsAndHidesPieces(t *testing.T) {
	s := newState(t)
	for _, tc := range []struct {
		id, shown, hidden string
		check             func(*State) bool
	}{
		{"furniture-1", "Shown to the players: Chest (furniture-1)", "Hidden from the players: Chest (furniture-1)",
			func(s *State) bool { return slices.Contains(s.SeenFurniture, "furniture-1") }},
		{"blocked-2", "Shown to the players: blocked squares at (4,4)", "Hidden from the players: blocked squares at (4,4)",
			func(s *State) bool { return slices.Contains(s.SeenBlocks, "blocked-2") }},
		{"door-1", "Shown to the players: door-1", "Hidden from the players: door-1",
			func(s *State) bool { return s.Doors[0].Seen }},
		{"door-3", "Shown to the players: gate door-3", "Hidden from the players: gate door-3",
			func(s *State) bool { return s.Doors[2].Seen }},
		{"monster-1", "Shown to the players: Orc (monster-1)", "Hidden from the players: Orc (monster-1)",
			func(s *State) bool { return s.Monsters[0].Visibility == MonsterSeen }},
	} {
		shown, ev := apply(t, s, cmd(t, "seen.set", map[string]any{"id": tc.id, "seen": true}))
		if !tc.check(shown) || ev.Summary != tc.shown {
			t.Errorf("%s shown: %q", tc.id, ev.Summary)
		}
		if _, _, err := Apply(shown, cmd(t, "seen.set", map[string]any{"id": tc.id, "seen": true}), nil); err == nil {
			t.Errorf("%s: showing it again should say nothing changed", tc.id)
		}
		hidden, ev := apply(t, shown, cmd(t, "seen.set", map[string]any{"id": tc.id, "seen": false}))
		if tc.check(hidden) || ev.Summary != tc.hidden {
			t.Errorf("%s hidden: %q", tc.id, ev.Summary)
		}
	}

	// Showing an unfound secret door finds it too (the GM is never blocked).
	s, ev := apply(t, s, cmd(t, "seen.set", map[string]any{"id": "door-2", "seen": true}))
	if !s.Doors[1].Seen || !s.Doors[1].Found || ev.Summary != "Shown to the players: door-2 (secret door found)" {
		t.Fatalf("secret door: %+v %q", s.Doors[1], ev.Summary)
	}

	for _, id := range []string{"trap-1", "note-A", "nope"} {
		if _, _, err := Apply(s, cmd(t, "seen.set", map[string]any{"id": id, "seen": true}), nil); err == nil {
			t.Errorf("seen.set %s should fail", id)
		}
	}
}

func TestRevealCanShowARoomsContents(t *testing.T) {
	s := newState(t)
	// The start room: door-1 is on its edge.
	start, ev := apply(t, s, cmd(t, "area.reveal", map[string]any{"x": 1, "y": 4, "seen": true}))
	if !start.Doors[0].Seen || ev.Summary != "Revealed Start (seen: 1 door)" {
		t.Fatalf("start room: %+v %q", start.Doors[0], ev.Summary)
	}

	// The Lair: both orcs and the chest; the secret door on its edge stays unseen until found.
	lair, ev := apply(t, s, cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1, "seen": true}))
	if lair.Doors[1].Seen || !slices.Equal(lair.SeenFurniture, []string{"furniture-1"}) || ev.Summary != "Revealed Lair (seen: 2 monsters, 1 piece of furniture)" {
		t.Fatalf("lair: %+v %v %q", lair.Doors[1], lair.SeenFurniture, ev.Summary)
	}
	found, _ := apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-2", "found": true}))
	found, _ = apply(t, found, cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1, "seen": true}))
	if !found.Doors[1].Seen {
		t.Fatal("a found secret door on the room's edge is seen")
	}

	// Corridor squares: the gate between (4,2) and (4,3), the rotated table on (3,2)-(3,3) and the blocked squares on (4,4).
	s, ev = apply(t, s, cmd(t, "tiles.reveal", map[string]any{"tiles": []map[string]int{{"x": 4, "y": 3}, {"x": 4, "y": 4}, {"x": 3, "y": 3}}, "seen": true}))
	if !s.Doors[2].Seen || !slices.Equal(s.SeenFurniture, []string{"furniture-2"}) || !slices.Equal(s.SeenBlocks, []string{"blocked-2"}) {
		t.Fatalf("corridor: %+v %v %v", s.Doors[2], s.SeenFurniture, s.SeenBlocks)
	}
	if ev.Summary != "Revealed 3 squares (seen: 1 piece of furniture, 1 door, 1 blocked square)" {
		t.Fatalf("corridor summary: %q", ev.Summary)
	}
	if s.Traps[0].State != "hidden" {
		t.Fatal("revealing never touches traps")
	}
	if pv := PlayerView(s); len(pv.Traps) != 0 {
		t.Fatalf("the players see a trap on a revealed square: %+v", pv.Traps)
	}

	// A plain reveal shows nothing but the squares.
	plain, _ := apply(t, newState(t), cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1}))
	if len(plain.SeenFurniture) != 0 || plain.Monsters[0].Visibility != MonsterHidden {
		t.Fatalf("plain reveal: %v", plain.SeenFurniture)
	}
}

func TestPlayersSetHidesMonsterBody(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "players.set", map[string]any{"hideMonsterBody": true}))
	if !s.Players.HideMonsterBody || ev.Summary != "Player screen: monster Body hidden" {
		t.Fatalf("hide: %+v %q", s.Players, ev.Summary)
	}
	if _, _, err := Apply(s, cmd(t, "players.set", map[string]any{"hideMonsterBody": true}), nil); err == nil {
		t.Fatal("nothing changed")
	}
	s, ev = apply(t, s, cmd(t, "players.set", map[string]any{"hideMonsterBody": false}))
	if s.Players.HideMonsterBody || ev.Summary != "Player screen: monster Body shown" {
		t.Fatalf("show: %q", ev.Summary)
	}
}

func TestTravelKeepsEachMapsSeenPieces(t *testing.T) {
	_, _, cat := fixture()
	s := travelState(t)
	s, _ = apply(t, s, cmd(t, "seen.set", map[string]any{"id": "furniture-1", "seen": true}))
	s, _ = apply(t, s, cmd(t, "seen.set", map[string]any{"id": "blocked-2", "seen": true}))
	lb, lq := lowerMap()
	away, _, err := Travel(s, Destination{QuestID: "quest-lower", QuestName: "Lower Vaults", Board: lb, Quest: lq}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(away.SeenFurniture) != 0 || len(away.SeenBlocks) != 0 {
		t.Fatalf("a new map starts with nothing seen: %v %v", away.SeenFurniture, away.SeenBlocks)
	}
	back, _, err := Travel(away, Destination{QuestID: "quest-upper"}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(back.SeenFurniture, []string{"furniture-1"}) || !slices.Equal(back.SeenBlocks, []string{"blocked-2"}) {
		t.Fatalf("returning restores what was seen: %v %v", back.SeenFurniture, back.SeenBlocks)
	}
}
