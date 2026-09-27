package tracker

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// fixture: a 6x4 board with room 1 in the top-left 2x2, room 2 in the
// bottom-right 2x2, corridor elsewhere; a quest with a door, a secret door, a
// trap, two monsters, a note and three start squares in room 1.
func fixture() (*maps.Board, *maps.Quest, *content.Catalog) {
	b := &maps.Board{
		Version: maps.CurrentVersion, Width: 6, Height: 4,
		Regions: []int{
			1, 1, 0, 0, 0, 0,
			1, 1, 0, 0, 0, 0,
			0, 0, 0, 0, 2, 2,
			0, 0, 0, 0, 2, 2,
		},
		Rooms: []maps.Room{{ID: 1, Name: "Start"}, {ID: 2, Name: "Lair"}},
	}
	body := 5
	q := maps.NewQuest(b)
	q.Doors = []maps.Door{
		{ID: "door-1", Edge: maps.Edge{X: 2, Y: 0, Orientation: maps.Vertical}, Kind: maps.DoorNormal, State: maps.DoorClosed},
		{ID: "door-2", Edge: maps.Edge{X: 4, Y: 2, Orientation: maps.Vertical}, Kind: maps.DoorSecret, State: maps.DoorClosed},
	}
	q.Traps = []maps.Trap{{ID: "trap-1", Kind: "pit", X: 3, Y: 1, State: maps.TrapHidden}}
	q.Monsters = []maps.Monster{
		{ID: "monster-1", Type: "orc", X: 4, Y: 2},
		{ID: "monster-2", Type: "orc", X: 5, Y: 3, Body: &body},
	}
	q.Notes = []maps.Note{{ID: "note-A", Label: "A", X: 5, Y: 2, Text: "84 gold"}}
	q.StartTiles = []maps.Tile{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}}
	cat := &content.Catalog{
		Monsters: []content.MonsterDef{{ID: "orc", Name: "Orc", Body: 1, Mind: 2, Attack: 3, Defense: 2, Movement: 8}},
		Heroes: []content.HeroDef{
			{ID: "barbarian", Name: "Barbarian", Body: 8, Mind: 2},
			{ID: "wizard", Name: "Wizard", Body: 4, Mind: 6},
		},
	}
	return b, q, cat
}

func party() []CampaignHero {
	return []CampaignHero{
		{ID: "hero-1", Name: "Grom", Player: "Sam", Class: "barbarian", Gold: 30, Equipment: "Broadsword"},
		{ID: "hero-2", Name: "Ilsa", Player: "Jo", Class: "wizard"},
	}
}

func TestNewSessionSetsUpFromQuest(t *testing.T) {
	b, q, cat := fixture()
	s, err := NewSession(b, q, "The Test", party(), cat)
	if err != nil {
		t.Fatal(err)
	}

	if s.Round != 1 || s.QuestName != "The Test" || s.Board.Width != 6 {
		t.Fatalf("header: round %d name %q", s.Round, s.QuestName)
	}

	wantHeroes := []Hero{
		{ID: "hero-1", Name: "Grom", Player: "Sam", Class: "barbarian", X: 0, Y: 0, Placed: true, Body: 8, MaxBody: 8, Mind: 2, MaxMind: 2, Gold: 30, Equipment: "Broadsword", Status: HeroActive},
		{ID: "hero-2", Name: "Ilsa", Player: "Jo", Class: "wizard", X: 1, Y: 0, Placed: true, Body: 4, MaxBody: 4, Mind: 6, MaxMind: 6, Status: HeroActive},
	}
	if !reflect.DeepEqual(s.Heroes, wantHeroes) {
		t.Fatalf("heroes:\n got  %+v\n want %+v", s.Heroes, wantHeroes)
	}

	if len(s.Monsters) != 2 || s.Monsters[0].Body != 1 || s.Monsters[0].MaxBody != 1 || s.Monsters[1].Body != 5 || s.Monsters[1].MaxBody != 5 {
		t.Fatalf("monster stats (catalog + override): %+v", s.Monsters)
	}
	if s.Monsters[0].Visibility != MonsterHidden || !s.Monsters[0].Alive || s.Monsters[0].Name != "Orc" {
		t.Fatalf("monsters start hidden and alive: %+v", s.Monsters[0])
	}

	if len(s.Doors) != 2 || s.Doors[0].State != maps.DoorClosed || s.Doors[1].Found {
		t.Fatalf("doors: %+v", s.Doors)
	}
	if len(s.Traps) != 1 || s.Traps[0].State != maps.TrapHidden {
		t.Fatalf("traps: %+v", s.Traps)
	}

	// The starting room is discovered: tiles (0,0) (1,0) (0,1) (1,1) = indexes 0 1 6 7.
	if !reflect.DeepEqual(s.Discovered, []int{0, 1, 6, 7}) {
		t.Fatalf("discovered = %v", s.Discovered)
	}
}

func TestNewSessionWithMoreHeroesThanStartSquares(t *testing.T) {
	b, q, cat := fixture()
	q.StartTiles = q.StartTiles[:1]
	s, err := NewSession(b, q, "Q", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Heroes[0].Placed || s.Heroes[1].Placed {
		t.Fatalf("second hero should wait off-board for the GM to place: %+v", s.Heroes)
	}
}

func TestNewSessionRejectsUnknownHeroClass(t *testing.T) {
	b, q, cat := fixture()
	heroes := party()
	heroes[0].Class = "paladin"
	if _, err := NewSession(b, q, "Q", heroes, cat); err == nil {
		t.Fatal("expected an error for an unknown hero class")
	}
}

func TestStateJSONRoundTrip(t *testing.T) {
	b, q, cat := fixture()
	s, err := NewSession(b, q, "Q", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back State
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&back, s) {
		t.Fatalf("round trip mismatch:\n%s", data)
	}
}

func TestCarryOverCopiesProgressBackToTheCampaign(t *testing.T) {
	b, q, cat := fixture()
	start := party()
	s, err := NewSession(b, q, "Q", start, cat)
	if err != nil {
		t.Fatal(err)
	}
	s.Heroes[0].Gold = 150
	s.Heroes[0].Equipment = "Broadsword, Helmet"
	s.Heroes[1].Notes = "Owes the elf a favour"
	s.Heroes[1].Status = HeroDead
	extra := CampaignHero{ID: "hero-9", Name: "Benched", Class: "elf", Gold: 5}

	out := s.CarryOver(append(start, extra))
	if out[0].Gold != 150 || out[0].Equipment != "Broadsword, Helmet" || out[0].Name != "Grom" {
		t.Fatalf("hero-1: %+v", out[0])
	}
	if out[1].Notes != "Owes the elf a favour" {
		t.Fatalf("hero-2: %+v", out[1])
	}
	if !reflect.DeepEqual(out[2], extra) {
		t.Fatalf("a hero who sat this quest out is unchanged: %+v", out[2])
	}
	if start[0].Gold != 30 {
		t.Fatal("CarryOver must not modify its input")
	}
}
