package tracker

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func newState(t *testing.T) *State {
	t.Helper()
	b, q, cat := fixture()
	s, err := NewSession(b, q, "The Test", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func cmd(t *testing.T, kind string, payload any) Command {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return Command{Type: kind, Payload: data}
}

func apply(t *testing.T, s *State, c Command) (*State, Event) {
	t.Helper()
	_, _, cat := fixture()
	next, ev, err := Apply(s, c, cat)
	if err != nil {
		t.Fatalf("%s: %v", c.Type, err)
	}
	return next, ev
}

func TestApplyNeverMutatesItsInput(t *testing.T) {
	s := newState(t)
	before, _ := json.Marshal(s)
	apply(t, s, cmd(t, "move", map[string]any{"id": "hero-1", "x": 4, "y": 1}))
	apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-1", "state": "open"}))
	apply(t, s, cmd(t, "tiles.reveal", map[string]any{"tiles": []map[string]int{{"x": 6, "y": 4}}}))
	after, _ := json.Marshal(s)
	if string(before) != string(after) {
		t.Fatal("Apply modified the input state")
	}
}

func TestMoveHeroAndMonster(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "move", map[string]any{"id": "hero-1", "x": 4, "y": 2}))
	if s.Heroes[0].X != 4 || s.Heroes[0].Y != 2 {
		t.Fatalf("hero at %d,%d", s.Heroes[0].X, s.Heroes[0].Y)
	}
	if ev.Kind != "move" || ev.Summary != "Moved Grom (Barbarian) from (1,4) to (4,2)" || ev.Round != 1 {
		t.Fatalf("event: %+v", ev)
	}

	s, ev = apply(t, s, cmd(t, "move", map[string]any{"id": "monster-1", "x": 4, "y": 1}))
	if s.Monsters[0].X != 4 || ev.Summary != "Moved Orc (monster-1) from (5,2) to (4,1)" {
		t.Fatalf("monster move: %+v %q", s.Monsters[0], ev.Summary)
	}
}

func TestPlacingAnUnplacedHero(t *testing.T) {
	b, q, cat := fixture()
	q.StartTiles = nil
	s, err := NewSession(b, q, "Q", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	s, ev := apply(t, s, cmd(t, "move", map[string]any{"id": "hero-2", "x": 2, "y": 3}))
	if !s.Heroes[1].Placed || ev.Summary != "Placed Ilsa (Wizard) at (2,3)" {
		t.Fatalf("%+v %q", s.Heroes[1], ev.Summary)
	}
}

func TestHeroUpdateDescribesEveryChange(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "hero.update", map[string]any{"id": "hero-1", "body": 5, "gold": 114, "equipment": "Broadsword, Chain mail"}))
	h := s.Heroes[0]
	if h.Body != 5 || h.Gold != 114 || h.Equipment != "Broadsword, Chain mail" || h.Mind != 2 {
		t.Fatalf("hero: %+v", h)
	}
	for _, want := range []string{"Grom", "body 8 → 5", "gold 30 → 114", "equipment"} {
		if !strings.Contains(ev.Summary, want) {
			t.Errorf("summary %q is missing %q", ev.Summary, want)
		}
	}

	s, ev = apply(t, s, cmd(t, "hero.update", map[string]any{"id": "hero-1", "status": "dead"}))
	if s.Heroes[0].Status != HeroDead || !strings.Contains(ev.Summary, "active → dead") {
		t.Fatalf("status: %+v %q", s.Heroes[0], ev.Summary)
	}
}

func TestMonsterLifecycle(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "monster.add", map[string]any{"type": "orc", "x": 3, "y": 2}))
	added := s.Monsters[len(s.Monsters)-1]
	if added.ID != "monster-3" || added.Body != 1 || added.Visibility != MonsterSeen || !added.Alive || ev.Summary != "Added Orc (monster-3) at (3,2)" {
		t.Fatalf("added: %+v %q", added, ev.Summary)
	}

	s, ev = apply(t, s, cmd(t, "monster.update", map[string]any{"id": "monster-1", "visibility": "seen"}))
	if s.Monsters[0].Visibility != MonsterSeen || ev.Summary != "Orc (monster-1): now seen" {
		t.Fatalf("seen: %q", ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "monster.update", map[string]any{"id": "monster-1", "body": 0, "alive": false}))
	if s.Monsters[0].Alive || !strings.Contains(ev.Summary, "body 1 → 0") || !strings.Contains(ev.Summary, "killed") {
		t.Fatalf("killed: %q", ev.Summary)
	}
	// The GM may bring it back.
	s, ev = apply(t, s, cmd(t, "monster.update", map[string]any{"id": "monster-1", "alive": true}))
	if !s.Monsters[0].Alive || !strings.Contains(ev.Summary, "revived") {
		t.Fatalf("revived: %q", ev.Summary)
	}

	s, ev = apply(t, s, cmd(t, "monster.remove", map[string]any{"id": "monster-3"}))
	if len(s.Monsters) != 2 || ev.Summary != "Removed Orc (monster-3)" {
		t.Fatalf("remove: %d %q", len(s.Monsters), ev.Summary)
	}
}

func TestDoorsCanBeOpenedAndReclosed(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-1", "state": "open"}))
	if s.Doors[0].State != "open" || ev.Summary != "Opened door-1" {
		t.Fatalf("open: %q", ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-1", "state": "closed"}))
	if s.Doors[0].State != "closed" || ev.Summary != "Closed door-1" {
		t.Fatalf("reclose: %q", ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-2", "found": true}))
	if !s.Doors[1].Found || ev.Summary != "Found secret door door-2" {
		t.Fatalf("found: %q", ev.Summary)
	}
}

func TestTrapsMoveBetweenAnyStates(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "trap.set", map[string]any{"id": "trap-1", "state": "triggered"}))
	if s.Traps[0].State != "triggered" || ev.Summary != "Trap trap-1 (pit): hidden → triggered" {
		t.Fatalf("trigger: %q", ev.Summary)
	}
	s, _ = apply(t, s, cmd(t, "trap.set", map[string]any{"id": "trap-1", "state": "hidden"}))
	if s.Traps[0].State != "hidden" {
		t.Fatal("un-triggering a trap must be allowed")
	}
}

func TestRevealAndHide(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1}))
	// Room 2 (Lair) = (5,1) (6,1) (5,2) (6,2) = 4 5 10 11, plus the start room 12 13 18 19.
	if !reflect.DeepEqual(s.Discovered, []int{4, 5, 10, 11, 12, 13, 18, 19}) || ev.Summary != "Revealed Lair" {
		t.Fatalf("discovered %v %q", s.Discovered, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "tiles.reveal", map[string]any{"tiles": []map[string]int{{"x": 3, "y": 4}, {"x": 4, "y": 4}, {"x": 99, "y": 4}, {"x": 0, "y": 4}}}))
	if !reflect.DeepEqual(s.Discovered, []int{4, 5, 10, 11, 12, 13, 18, 19, 20, 21}) || ev.Summary != "Revealed 2 squares" {
		t.Fatalf("tiles %v %q", s.Discovered, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "tiles.hide", map[string]any{"tiles": []map[string]int{{"x": 3, "y": 4}}}))
	if slices.Contains(s.Discovered, 20) || ev.Summary != "Hid 1 square" {
		t.Fatalf("hide %v %q", s.Discovered, ev.Summary)
	}
}

func TestNotesRoundsAndLog(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "note.consume", map[string]any{"id": "note-A", "consumed": true}))
	if !reflect.DeepEqual(s.ConsumedNotes, []string{"note-A"}) || ev.Summary != "Used note A" {
		t.Fatalf("consume: %v %q", s.ConsumedNotes, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "note.consume", map[string]any{"id": "note-A", "consumed": false}))
	if len(s.ConsumedNotes) != 0 || ev.Summary != "Restored note A" {
		t.Fatalf("restore: %v %q", s.ConsumedNotes, ev.Summary)
	}

	s, ev = apply(t, s, cmd(t, "round.advance", map[string]any{}))
	if s.Round != 2 || ev.Summary != "Round 2 begins" || ev.Round != 2 {
		t.Fatalf("round: %d %+v", s.Round, ev)
	}

	next, ev := apply(t, s, cmd(t, "log.note", map[string]any{"text": "  Grom bargains with the gargoyle.  "}))
	if ev.Kind != "log.note" || ev.Summary != "Grom bargains with the gargoyle." {
		t.Fatalf("log: %+v", ev)
	}
	a, _ := json.Marshal(s)
	b, _ := json.Marshal(next)
	if string(a) != string(b) {
		t.Fatal("a log note must not change the state")
	}
}

func TestEventPayloadRecordsTheCommand(t *testing.T) {
	s := newState(t)
	_, ev := apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-1", "state": "open"}))
	var payload struct {
		Command map[string]any `json:"command"`
	}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Command["id"] != "door-1" || payload.Command["state"] != "open" {
		t.Fatalf("payload: %s", ev.Payload)
	}
}

func TestApplyRejectsMalformedCommands(t *testing.T) {
	s := newState(t)
	_, _, cat := fixture()
	bad := []Command{
		{Type: "teleport", Payload: json.RawMessage(`{}`)},
		cmd(t, "move", map[string]any{"id": "nobody", "x": 1, "y": 1}),
		cmd(t, "move", map[string]any{"id": "hero-1", "x": 7, "y": 4}),
		cmd(t, "move", map[string]any{"id": "hero-1", "x": 0, "y": 1}),
		cmd(t, "move", map[string]any{"id": "hero-1", "x": 1, "y": 0}),
		cmd(t, "hero.update", map[string]any{"id": "hero-1", "body": -1}),
		cmd(t, "hero.update", map[string]any{"id": "hero-1", "status": "sleepy"}),
		cmd(t, "monster.add", map[string]any{"type": "dragon", "x": 1, "y": 1}),
		cmd(t, "monster.update", map[string]any{"id": "monster-1", "visibility": "invisible"}),
		cmd(t, "door.set", map[string]any{"id": "door-9", "state": "open"}),
		cmd(t, "door.set", map[string]any{"id": "door-1", "state": "ajar"}),
		cmd(t, "trap.set", map[string]any{"id": "trap-1", "state": "armed"}),
		cmd(t, "note.consume", map[string]any{"id": "note-Z", "consumed": true}),
		cmd(t, "log.note", map[string]any{"text": "   "}),
		{Type: "move", Payload: json.RawMessage(`{"id":`)},
	}
	for _, c := range bad {
		if _, _, err := Apply(s, c, cat); err == nil {
			t.Errorf("%s %s: expected an error", c.Type, c.Payload)
		}
	}
}

func TestApplyRefusesTopLeftStates(t *testing.T) {
	s := newState(t)
	s.Version = 1 // saved before squares counted from the bottom-left
	_, _, cat := fixture()
	if _, _, err := Apply(s, cmd(t, "round.advance", nil), cat); err == nil {
		t.Fatal("expected an error for a version 1 state")
	}
}
