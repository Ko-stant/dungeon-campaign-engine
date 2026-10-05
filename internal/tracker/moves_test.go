package tracker

import (
	"slices"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// hallBoard is room 1 (columns 1-2), a corridor (column 3) and room 2
// (columns 4-5), with closed doors on the left (d1) and right (d2) of the
// corridor's middle square; a hidden pit in the corridor at (3,3); in room 2
// an orc at (5,3), a tall bookcase at (5,2) and a chest at (5,1). Grom
// stands at (1,2), Ilsa at (1,1).
//
//	y3: 1 1 . 2 2
//	y2: 1 1 . 2 2
//	y1: 1 1 . 2 2
func hallBoard() (*maps.Board, *maps.Quest, *content.Catalog) {
	b := &maps.Board{
		Version: maps.CurrentVersion, Width: 5, Height: 3,
		Regions: []int{
			1, 1, 0, 2, 2,
			1, 1, 0, 2, 2,
			1, 1, 0, 2, 2,
		},
		Rooms: []maps.Room{{ID: 1, Name: "Start"}, {ID: 2, Name: "Lair"}},
	}
	q := maps.NewQuest(b)
	q.Doors = []maps.Door{
		{ID: "d1", Edge: maps.Edge{X: 3, Y: 2, Orientation: maps.Vertical}, Kind: maps.DoorNormal, State: maps.DoorClosed},
		{ID: "d2", Edge: maps.Edge{X: 4, Y: 2, Orientation: maps.Vertical}, Kind: maps.DoorNormal, State: maps.DoorClosed},
	}
	q.Traps = []maps.Trap{{ID: "pit", Kind: "pit", X: 3, Y: 3, State: maps.TrapHidden}}
	q.Monsters = []maps.Monster{{ID: "orc", Type: "orc", X: 5, Y: 3}}
	q.Furniture = []maps.Furniture{
		{ID: "shelf", Type: "bookcase", X: 5, Y: 2},
		{ID: "chest", Type: "chest", X: 5, Y: 1},
	}
	q.StartTiles = []maps.Tile{{X: 1, Y: 2}, {X: 1, Y: 1}}
	_, _, cat := fixture()
	cat.Furniture = append(cat.Furniture, content.FurnitureDef{ID: "bookcase", Name: "Bookcase", Width: 1, Height: 1, BlocksMovement: true, BlocksLineOfSight: true})
	cat.Traps = []content.TrapDef{{ID: "pit", Name: "Pit Trap", Width: 1, Height: 1}}
	return b, q, cat
}

func hallApply(t *testing.T, s *State, c Command) (*State, Event) {
	t.Helper()
	_, _, cat := hallBoard()
	next, ev, err := Apply(s, c, cat)
	if err != nil {
		t.Fatalf("%s: %v", c.Type, err)
	}
	return next, ev
}

func hallErr(t *testing.T, s *State, c Command) string {
	t.Helper()
	_, _, cat := hallBoard()
	_, _, err := Apply(s, c, cat)
	if err == nil {
		t.Fatalf("%s: expected an error", c.Type)
	}
	return err.Error()
}

// hallTurn starts Grom's turn in rules mode with a movement roll of rolled
// (two d6 given by the GM).
func hallTurn(t *testing.T, d1, d2 int) *State {
	t.Helper()
	b, q, cat := hallBoard()
	s, err := NewSession(b, q, "Hall", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	s, _ = hallApply(t, s, cmd(t, "rules.enable", map[string]any{"seed": 7}))
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"})))
	roll := cmd(t, "turn.roll-move", map[string]any{})
	roll.Dice = []int{d1, d2}
	s, _ = hallApply(t, s, roll)
	return s
}

func moveTo(t *testing.T, x, y int) Command {
	return as(seat("hero-1"), cmd(t, "turn.move", map[string]any{"to": map[string]int{"x": x, "y": y}}))
}

func openDoor(t *testing.T, id string) Command {
	return as(seat("hero-1"), cmd(t, "turn.door", map[string]any{"door": id}))
}

func discovered(s *State, x, y int) bool {
	return slices.Contains(s.Discovered, s.Board.Index(x, y))
}

func TestMovementNeedsARoll(t *testing.T) {
	b, q, cat := hallBoard()
	s, _ := NewSession(b, q, "Hall", party(), cat)
	s, _ = hallApply(t, s, cmd(t, "rules.enable", map[string]any{"seed": 7}))
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"})))
	if msg := hallErr(t, s, moveTo(t, 2, 2)); !strings.Contains(msg, "roll") {
		t.Errorf("moving before rolling: %q", msg)
	}
}

func TestAMoveSpendsSquaresAlongTheShortestPath(t *testing.T) {
	s := hallTurn(t, 1, 2)
	s, ev := hallApply(t, s, moveTo(t, 2, 3))
	h := s.Heroes[0]
	if h.X != 2 || h.Y != 3 || s.Rules.Turn.MoveLeft != 1 {
		t.Fatalf("hero at (%d,%d), %d left", h.X, h.Y, s.Rules.Turn.MoveLeft)
	}
	if ev.Summary != "Grom (Barbarian) moves 2 squares to (2,3)" {
		t.Errorf("summary %q", ev.Summary)
	}
	if msg := hallErr(t, s, moveTo(t, 1, 2)); !strings.Contains(msg, "1 square") {
		t.Errorf("too far: %q", msg)
	}
	s, _ = hallApply(t, s, moveTo(t, 1, 3))
	if s.Rules.Turn.MoveLeft != 0 {
		t.Errorf("moved the last square: %d left", s.Rules.Turn.MoveLeft)
	}
}

func TestClosedDoorsWallsAndOccupiedSquaresStopAMove(t *testing.T) {
	s := hallTurn(t, 6, 6)
	if msg := hallErr(t, s, moveTo(t, 3, 2)); !strings.Contains(msg, "no way") {
		t.Errorf("through a closed door: %q", msg)
	}
	if msg := hallErr(t, s, moveTo(t, 1, 1)); !strings.Contains(msg, "Ilsa") {
		t.Errorf("onto an ally: %q", msg)
	}
	if msg := hallErr(t, s, moveTo(t, 9, 9)); !strings.Contains(msg, "off the") {
		t.Errorf("off the board: %q", msg)
	}
}

func TestHeroesPassThroughAllies(t *testing.T) {
	b, q, cat := hallBoard()
	q.StartTiles = []maps.Tile{{X: 1, Y: 1}, {X: 1, Y: 2}}
	s, _ := NewSession(b, q, "Hall", party(), cat)
	s, _ = hallApply(t, s, cmd(t, "rules.enable", map[string]any{"seed": 7}))
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"})))
	roll := cmd(t, "turn.roll-move", map[string]any{})
	roll.Dice = []int{1, 1}
	s, _ = hallApply(t, s, roll)
	s, _ = hallApply(t, s, moveTo(t, 1, 3))
	if h := s.Heroes[0]; h.X != 1 || h.Y != 3 {
		t.Fatalf("Grom should pass Ilsa: at (%d,%d)", h.X, h.Y)
	}
}

func TestAGivenPathIsChecked(t *testing.T) {
	s := hallTurn(t, 3, 3)
	path := func(tiles ...[2]int) Command {
		var p []map[string]int
		for _, tl := range tiles {
			p = append(p, map[string]int{"x": tl[0], "y": tl[1]})
		}
		return as(seat("hero-1"), cmd(t, "turn.move", map[string]any{"path": p}))
	}
	if msg := hallErr(t, s, path([2]int{2, 2}, [2]int{2, 1}, [2]int{3, 1})); !strings.Contains(msg, "(2,1) to (3,1)") {
		t.Errorf("a step through a wall: %q", msg)
	}
	if msg := hallErr(t, s, path([2]int{2, 3})); !strings.Contains(msg, "(1,2) to (2,3)") {
		t.Errorf("a diagonal step: %q", msg)
	}
	s, ev := hallApply(t, s, path([2]int{1, 3}, [2]int{2, 3}, [2]int{2, 2}))
	if h := s.Heroes[0]; h.X != 2 || h.Y != 2 || s.Rules.Turn.MoveLeft != 3 || ev.Summary != "Grom (Barbarian) moves 3 squares to (2,2)" {
		t.Fatalf("hero (%d,%d), %d left, %q", h.X, h.Y, s.Rules.Turn.MoveLeft, ev.Summary)
	}
}

func TestOpeningADoorRevealsWhatTheHeroCanSee(t *testing.T) {
	s := hallTurn(t, 3, 3)
	if msg := hallErr(t, s, openDoor(t, "d1")); !strings.Contains(msg, "beside") {
		t.Errorf("a door out of reach: %q", msg)
	}
	s, _ = hallApply(t, s, moveTo(t, 2, 2))
	if discovered(s, 3, 2) {
		t.Fatal("the corridor should not be seen through a closed door")
	}
	s, ev := hallApply(t, s, openDoor(t, "d1"))
	if s.Doors[0].State != maps.DoorOpen || !s.Doors[0].Seen {
		t.Fatalf("door: %+v", s.Doors[0])
	}
	for y := 1; y <= 3; y++ {
		if !discovered(s, 3, y) {
			t.Errorf("corridor square (3,%d) should be in view", y)
		}
	}
	if discovered(s, 4, 2) || s.Monsters[0].Visibility != MonsterHidden {
		t.Error("room 2 is still behind its closed door")
	}
	if ev.Summary != "Grom (Barbarian) opens d1 (seen: 1 door)" {
		t.Errorf("summary %q", ev.Summary)
	}
	if s.Rules.Turn.MoveLeft != 5 {
		t.Errorf("opening a door is free: %d left", s.Rules.Turn.MoveLeft)
	}
	if msg := hallErr(t, s, openDoor(t, "d1")); !strings.Contains(msg, "open") {
		t.Errorf("an open door: %q", msg)
	}
}

func TestSpottingAMonsterStartsAFight(t *testing.T) {
	s := hallTurn(t, 3, 3)
	s, _ = hallApply(t, s, moveTo(t, 2, 2))
	s, _ = hallApply(t, s, openDoor(t, "d1"))
	s, _ = hallApply(t, s, moveTo(t, 3, 2))
	s, ev := hallApply(t, s, openDoor(t, "d2"))
	if s.Monsters[0].Visibility != MonsterSeen || !s.Fight {
		t.Fatalf("orc %+v, fight %v", s.Monsters[0], s.Fight)
	}
	if !slices.Contains(s.SeenFurniture, "chest") {
		t.Error("the chest is in view")
	}
	if !strings.Contains(ev.Summary, "1 monster") || !strings.HasSuffix(ev.Summary, "; fight started") {
		t.Errorf("summary %q", ev.Summary)
	}
	if len(ev.PlayerSpotted) != 1 || ev.PlayerSpotted[0].ID != "orc" {
		t.Errorf("the player screen hears about the orc: %+v", ev.PlayerSpotted)
	}
}

func TestTallFurnitureHidesWhatIsBehindIt(t *testing.T) {
	b, q, cat := hallBoard()
	// Grom stands in room 2 with the bookcase between him and the orc.
	q.Monsters[0].X, q.Monsters[0].Y = 4, 3
	q.Furniture[0].X, q.Furniture[0].Y = 4, 2
	q.StartTiles = []maps.Tile{{X: 4, Y: 1}, {X: 1, Y: 1}}
	s, _ := NewSession(b, q, "Hall", party(), cat)
	s, _ = hallApply(t, s, cmd(t, "rules.enable", map[string]any{"seed": 7}))
	if !discovered(s, 5, 1) || !discovered(s, 5, 2) {
		t.Fatal("the squares beside Grom are in view")
	}
	// (The whole starting room counts as discovered; its pieces show only
	// when in view.)
	if s.Monsters[0].Visibility == MonsterSeen {
		t.Error("the orc at (4,3) is behind the bookcase")
	}
}

func TestAHiddenTrapStopsTheMove(t *testing.T) {
	s := hallTurn(t, 6, 6)
	s, _ = hallApply(t, s, moveTo(t, 2, 2))
	s, _ = hallApply(t, s, openDoor(t, "d1"))
	s, ev := hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.move", map[string]any{"path": []map[string]int{{"x": 3, "y": 2}, {"x": 3, "y": 3}, {"x": 3, "y": 2}}})))
	h := s.Heroes[0]
	if h.X != 3 || h.Y != 3 {
		t.Fatalf("the move should stop on the pit: (%d,%d)", h.X, h.Y)
	}
	if s.Traps[0].State != maps.TrapTriggered || s.Rules.Turn.MoveLeft != 0 || !s.Rules.Turn.MoveDone {
		t.Fatalf("trap %+v, turn %+v", s.Traps[0], s.Rules.Turn)
	}
	if ev.Summary != "Grom (Barbarian) moves 2 squares to (3,3); Pit Trap pit triggered: it stays on the board; the move ends" {
		t.Errorf("summary %q", ev.Summary)
	}
	if msg := hallErr(t, s, moveTo(t, 3, 2)); !strings.Contains(msg, "over") {
		t.Errorf("moving after a trap: %q", msg)
	}
}

func TestAPathAvoidsKnownTrapsWhenItCan(t *testing.T) {
	s := hallTurn(t, 6, 6)
	s.Heroes[0].X, s.Heroes[0].Y = 3, 2
	s.Doors[0].State = maps.DoorOpen
	s.Traps[0].State = maps.TrapRevealed
	// Only one way to (3,3) runs over the pit, so the pit is walked into.
	s, _ = hallApply(t, s, moveTo(t, 3, 3))
	if s.Traps[0].State != maps.TrapTriggered {
		t.Errorf("walking onto a revealed trap still sets it off: %+v", s.Traps[0])
	}
}

func TestRulesEnableRevealsWhatTheHeroesSee(t *testing.T) {
	b, q, cat := hallBoard()
	s, _ := NewSession(b, q, "Hall", party(), cat)
	s, ev := hallApply(t, s, cmd(t, "rules.enable", map[string]any{"seed": 7}))
	if !s.Doors[0].Seen || ev.Summary != "Rules on (three-plagues/1): the heroes' turns (seen: 1 door)" {
		t.Errorf("door %+v, summary %q", s.Doors[0], ev.Summary)
	}
}
