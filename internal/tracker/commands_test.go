package tracker

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
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
	s, ev := apply(t, s, cmd(t, "hero.update", map[string]any{"id": "hero-1", "body": 5, "equipment": "Broadsword, Chain mail"}))
	h := s.Heroes[0]
	if h.Body != 5 || h.Equipment != "Broadsword, Chain mail" || h.Mind != 2 {
		t.Fatalf("hero: %+v", h)
	}
	for _, want := range []string{"Grom", "body 8 → 5", "equipment"} {
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

	s, ev = apply(t, s, cmd(t, "monster.add", map[string]any{"type": "custom-ogre", "x": 1, "y": 1}))
	ogre := s.Monsters[len(s.Monsters)-1]
	if ogre.Width != 2 || ogre.Height != 2 || ogre.Color != "#aa3300" || ev.Summary != "Added Cave Ogre (monster-4) at (1,1)" {
		t.Fatalf("custom: %+v %q", ogre, ev.Summary)
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
	if len(s.Monsters) != 3 || ev.Summary != "Removed Orc (monster-3)" {
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

func TestDoorsAndGatesCanBeLockedAndUnlocked(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-3", "locked": false}))
	if s.Doors[2].Locked || ev.Summary != "Unlocked gate door-3" {
		t.Fatalf("unlock: %q", ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-3", "state": "open"}))
	if ev.Summary != "Opened gate door-3" {
		t.Fatalf("open gate: %q", ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-1", "locked": true, "state": "closed"}))
	if !s.Doors[0].Locked || ev.Summary != "Closed door-1; Locked door-1" {
		t.Fatalf("close and lock: %q", ev.Summary)
	}
	// Nothing is enforced: a locked door can still be opened.
	s, ev = apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-1", "state": "open"}))
	if s.Doors[0].State != "open" || !s.Doors[0].Locked || ev.Summary != "Opened door-1" {
		t.Fatalf("open while locked: %q", ev.Summary)
	}
}

func TestBlockedSquaresCanBeRemovedAndPutBack(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "blocked.set", map[string]any{"id": "blocked-1", "removed": true}))
	if !reflect.DeepEqual(s.RemovedBlocks, []string{"blocked-1"}) || ev.Summary != "Found the secret door at (3,1) (blocked-1 removed)" {
		t.Fatalf("hidden door: %v %q", s.RemovedBlocks, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "blocked.set", map[string]any{"id": "blocked-2", "removed": true}))
	if len(s.RemovedBlocks) != 2 || ev.Summary != "Removed blocked-2 at (4,4)" {
		t.Fatalf("plain block: %v %q", s.RemovedBlocks, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "blocked.set", map[string]any{"id": "blocked-1", "removed": false}))
	if !reflect.DeepEqual(s.RemovedBlocks, []string{"blocked-2"}) || ev.Summary != "Put back blocked-1 at (3,1)" {
		t.Fatalf("put back: %v %q", s.RemovedBlocks, ev.Summary)
	}
	if _, _, err := Apply(s, cmd(t, "blocked.set", map[string]any{"id": "blocked-9", "removed": true}), nil); err == nil {
		t.Fatal("unknown blocked square should be an error")
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

func TestTrapsCanBeRemovedAndBroughtBack(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "trap.set", map[string]any{"id": "trap-1", "state": "removed"}))
	if s.Traps[0].State != "removed" || ev.Summary != "Trap trap-1 (pit): hidden → removed" {
		t.Fatalf("remove: %q %+v", ev.Summary, s.Traps[0])
	}
	s, _ = apply(t, s, cmd(t, "trap.set", map[string]any{"id": "trap-1", "state": "disarmed"}))
	if s.Traps[0].State != "disarmed" {
		t.Fatal("a removed trap must come back with any other state")
	}
}

func TestMoveMovesATrap(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "move", map[string]any{"id": "trap-1", "x": 2, "y": 2}))
	if at := s.Traps[0].At; at == nil || *at != (maps.Tile{X: 2, Y: 2}) {
		t.Fatalf("trap position: %+v", s.Traps[0])
	}
	if ev.Summary != "Moved trap trap-1 (pit) from (4,3) to (2,2)" {
		t.Fatalf("summary: %q", ev.Summary)
	}
	_, ev = apply(t, s, cmd(t, "move", map[string]any{"id": "trap-1", "x": 3, "y": 2}))
	if ev.Summary != "Moved trap trap-1 (pit) from (2,2) to (3,2)" {
		t.Fatalf("second move starts from the live position: %q", ev.Summary)
	}
	if _, _, err := Apply(s, cmd(t, "move", map[string]any{"id": "trap-1", "x": 9, "y": 9}), nil); err == nil {
		t.Fatal("moving a trap off the board should be an error")
	}
}

func TestTrapSummariesUseCatalogNamesAndLabels(t *testing.T) {
	b, q, cat := fixture()
	q.Traps = append(q.Traps,
		maps.Trap{ID: "trap-2", Kind: "boulder", X: 3, Y: 3, State: maps.TrapHidden},
		maps.Trap{ID: "trap-3", Kind: "trigger", X: 3, Y: 2, State: maps.TrapHidden, Label: "1"},
	)
	cat.Traps = []content.TrapDef{{ID: "boulder", Name: "Boulder", Width: 1, Height: 1, Movable: true}}
	s, err := NewSession(b, q, "The Test", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	_, ev, err := Apply(s, cmd(t, "trap.set", map[string]any{"id": "trap-2", "state": "triggered"}), cat)
	if err != nil || ev.Summary != "Trap trap-2 (Boulder): hidden → triggered" {
		t.Fatalf("catalog name: %q %v", ev.Summary, err)
	}
	_, ev, err = Apply(s, cmd(t, "trap.set", map[string]any{"id": "trap-3", "state": "triggered"}), cat)
	if err != nil || ev.Summary != "Trap trap-3 (Trigger 1): hidden → triggered" {
		t.Fatalf("label: %q %v", ev.Summary, err)
	}
}

func TestRevealCanShowTheMonstersThere(t *testing.T) {
	s := newState(t)
	plain, ev := apply(t, s, cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1}))
	if plain.Monsters[0].Visibility != MonsterHidden || ev.Summary != "Revealed Lair" {
		t.Fatalf("a plain reveal leaves monsters alone: %q %+v", ev.Summary, plain.Monsters[0])
	}

	seen, ev := apply(t, s, cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1, "seen": true}))
	if seen.Monsters[0].Visibility != MonsterSeen || seen.Monsters[1].Visibility != MonsterSeen {
		t.Fatalf("both Lair monsters should be seen: %+v", seen.Monsters)
	}
	if ev.Summary != "Revealed Lair (seen: 2 monsters, 1 piece of furniture)" {
		t.Fatalf("summary: %q", ev.Summary)
	}
	if seen.Traps[0].State != maps.TrapHidden {
		t.Fatal("revealing never touches traps")
	}
}

func TestRevealSquaresCanShowMonstersOnThem(t *testing.T) {
	s := newState(t)
	// A 2x2 custom monster on (3,1)-(4,2): revealing any one of its squares shows it.
	s, _ = apply(t, s, cmd(t, "monster.add", map[string]any{"type": "custom-ogre", "x": 3, "y": 1, "visibility": "hidden"}))
	s, ev := apply(t, s, cmd(t, "tiles.reveal", map[string]any{"tiles": []map[string]int{{"x": 5, "y": 2}, {"x": 4, "y": 2}}, "seen": true}))
	byID := map[string]string{}
	for _, m := range s.Monsters {
		byID[m.ID] = m.Visibility
	}
	if byID["monster-1"] != MonsterSeen || byID["monster-2"] != MonsterHidden || byID["monster-3"] != MonsterSeen {
		t.Fatalf("visibility: %+v", byID)
	}
	if ev.Summary != "Revealed 2 squares (seen: 2 monsters, 1 door)" {
		t.Fatalf("summary: %q", ev.Summary)
	}
}

func TestRevealingAgainChangesNothing(t *testing.T) {
	s := newState(t)
	s, _ = apply(t, s, cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1}))
	// The same room again, and squares already revealed: no event, so no log line.
	for _, c := range []Command{
		cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1}),
		cmd(t, "tiles.reveal", map[string]any{"tiles": []map[string]int{{"x": 5, "y": 1}, {"x": 6, "y": 2}}}),
		cmd(t, "tiles.hide", map[string]any{"tiles": []map[string]int{{"x": 3, "y": 4}}}),
	} {
		if _, _, err := Apply(s, c, nil); err == nil || !strings.Contains(err.Error(), "already") {
			t.Fatalf("%s: want an 'already' error, got %v", c.Type, err)
		}
	}
	// Revealing it again with "show contents" still shows monsters not yet seen.
	s, ev := apply(t, s, cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1, "seen": true}))
	if ev.Summary != "Revealed Lair (seen: 2 monsters, 1 piece of furniture)" || s.Monsters[0].Visibility != MonsterSeen {
		t.Fatalf("contents shown on a second reveal: %q", ev.Summary)
	}
	if _, _, err := Apply(s, cmd(t, "area.reveal", map[string]any{"x": 6, "y": 1, "seen": true}), nil); err == nil {
		t.Fatal("nothing left to reveal or show: no event")
	}
	// Partly new squares still reveal (and count only the new ones).
	_, ev = apply(t, s, cmd(t, "tiles.reveal", map[string]any{"tiles": []map[string]int{{"x": 5, "y": 1}, {"x": 3, "y": 4}}}))
	if ev.Summary != "Revealed 1 square" {
		t.Fatalf("partly new: %q", ev.Summary)
	}
}

func TestMonstersCarryTheirMovement(t *testing.T) {
	s := newState(t)
	for _, m := range s.Monsters {
		if m.Type == "orc" && m.Movement != 8 {
			t.Fatalf("a set-up orc keeps the catalog's movement: %+v", m)
		}
	}
	s, _ = apply(t, s, cmd(t, "monster.add", map[string]any{"type": "orc", "x": 3, "y": 2}))
	if added := s.Monsters[len(s.Monsters)-1]; added.Movement != 8 {
		t.Fatalf("an added orc keeps the catalog's movement: %+v", added)
	}
}
