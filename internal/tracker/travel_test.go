package tracker

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// lowerMap is a 4x3 all-corridor map with two start squares on the left, a
// door and one orc.
func lowerMap() (*maps.Board, *maps.Quest) {
	b := &maps.Board{Version: maps.CurrentVersion, Width: 4, Height: 3, Regions: make([]int, 12)}
	q := maps.NewQuest(b)
	q.StartTiles = []maps.Tile{{X: 1, Y: 1}, {X: 1, Y: 2}}
	q.Doors = []maps.Door{{ID: "door-1", Edge: maps.Edge{X: 3, Y: 2, Orientation: maps.Vertical}, Kind: maps.DoorNormal, State: maps.DoorClosed}}
	q.Monsters = []maps.Monster{{ID: "monster-1", Type: "orc", X: 4, Y: 3}}
	return b, q
}

func travelState(t *testing.T) *State {
	t.Helper()
	s := newState(t)
	s.QuestID = "quest-upper"
	return s
}

func TestTravelToANewMapKeepsTheHeroesAndSavesTheOldMap(t *testing.T) {
	_, _, cat := fixture()
	s := travelState(t)
	s, _ = apply(t, s, cmd(t, "hero.update", map[string]any{"id": "hero-1", "body": 3, "gold": 40}))
	s, _ = apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-1", "state": "open"}))
	s, _ = apply(t, s, cmd(t, "move", map[string]any{"id": "hero-1", "x": 3, "y": 3}))
	s, _ = apply(t, s, cmd(t, "round.advance", nil))
	before, _ := json.Marshal(s)

	lb, lq := lowerMap()
	next, ev, err := Travel(s, Destination{QuestID: "quest-lower", QuestName: "Lower Vaults", Board: lb, Quest: lq}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if after, _ := json.Marshal(s); string(after) != string(before) {
		t.Fatal("Travel modified its input")
	}

	if ev.Kind != "map.travel" || ev.Summary != "Travelled to Lower Vaults" || ev.Round != 2 {
		t.Fatalf("event: %+v", ev)
	}
	if next.QuestID != "quest-lower" || next.QuestName != "Lower Vaults" || next.Board.Width != 4 || next.Round != 2 {
		t.Fatalf("active map: %s %s %dx%d round %d", next.QuestID, next.QuestName, next.Board.Width, next.Board.Height, next.Round)
	}
	grom := next.Heroes[0]
	if grom.Body != 3 || grom.Gold != 40 || grom.X != 1 || grom.Y != 1 || !grom.Placed {
		t.Fatalf("heroes keep their stats and start on the new map's start squares: %+v", grom)
	}
	if next.Heroes[1].X != 1 || next.Heroes[1].Y != 2 {
		t.Fatalf("second hero: %+v", next.Heroes[1])
	}
	if len(next.Monsters) != 1 || next.Monsters[0].Name != "Orc" || next.Monsters[0].Visibility != MonsterHidden {
		t.Fatalf("new map monsters: %+v", next.Monsters)
	}
	if len(next.Doors) != 1 || next.Doors[0].State != maps.DoorClosed {
		t.Fatalf("new map doors: %+v", next.Doors)
	}
	// The new map's start squares are discovered (corridor squares reveal only themselves).
	if !reflect.DeepEqual(next.Discovered, []int{0, 4}) {
		t.Fatalf("discovered: %v", next.Discovered)
	}

	if len(next.OtherMaps) != 1 {
		t.Fatalf("other maps: %+v", next.OtherMaps)
	}
	upper := next.OtherMaps[0]
	if upper.QuestID != "quest-upper" || upper.QuestName != "The Test" || upper.Doors[0].State != maps.DoorOpen || len(upper.Monsters) != 2 {
		t.Fatalf("saved map: %+v", upper)
	}
	if p := upper.HeroPositions[0]; p.ID != "hero-1" || p.X != 3 || p.Y != 3 || !p.Placed {
		t.Fatalf("saved hero position: %+v", upper.HeroPositions)
	}
}

func TestTravelBackRestoresTheMapAsItWasLeft(t *testing.T) {
	_, _, cat := fixture()
	s := travelState(t)
	s, _ = apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-1", "state": "open"}))
	s, _ = apply(t, s, cmd(t, "move", map[string]any{"id": "hero-1", "x": 3, "y": 3}))
	lb, lq := lowerMap()
	s, _, err := Travel(s, Destination{QuestID: "quest-lower", QuestName: "Lower Vaults", Board: lb, Quest: lq}, cat)
	if err != nil {
		t.Fatal(err)
	}
	s, _ = apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-1", "state": "open"})) // lower map's door
	s, _ = apply(t, s, cmd(t, "hero.update", map[string]any{"id": "hero-1", "body": 2}))

	back, ev, err := Travel(s, Destination{QuestID: "quest-upper"}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Summary != "Returned to The Test" {
		t.Fatalf("summary: %q", ev.Summary)
	}
	if back.QuestID != "quest-upper" || back.Board.Width != 6 || len(back.Monsters) != 2 || back.Doors[0].State != maps.DoorOpen {
		t.Fatalf("restored map: %s %dx%d %+v", back.QuestID, back.Board.Width, back.Board.Height, back.Doors)
	}
	if g := back.Heroes[0]; g.X != 3 || g.Y != 3 || g.Body != 2 {
		t.Fatalf("hero back where they left, with current stats: %+v", g)
	}
	if len(back.OtherMaps) != 1 || back.OtherMaps[0].QuestID != "quest-lower" || back.OtherMaps[0].Doors[0].State != maps.DoorOpen {
		t.Fatalf("the lower map is saved in turn: %+v", back.OtherMaps)
	}
	if !back.Visited("quest-lower") || !back.Visited("quest-upper") || back.Visited("quest-nowhere") {
		t.Fatal("Visited should know every map of the session")
	}
}

func TestTravelErrors(t *testing.T) {
	_, _, cat := fixture()
	s := travelState(t)
	if _, _, err := Travel(s, Destination{QuestID: "quest-upper"}, cat); err == nil {
		t.Fatal("travelling to the current map should be an error")
	}
	if _, _, err := Travel(s, Destination{QuestID: "quest-new"}, cat); err == nil {
		t.Fatal("a map not visited yet needs its board and quest")
	}
	old := newState(t)
	old.Version = 1
	lb, lq := lowerMap()
	if _, _, err := Travel(old, Destination{QuestID: "q", Board: lb, Quest: lq}, cat); err == nil {
		t.Fatal("old states are refused")
	}
}
