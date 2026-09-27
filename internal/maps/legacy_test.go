package maps

import (
	"reflect"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/legacy"
)

// legacyBoard is a synthetic 4x3 legacy board. Legacy files count from 0 at
// the top-left:
//
//	y0: 1 1 . .
//	y1: 1 1 . .
//	y2: . . . 2
//
// Converted, the same picture is rows y3 (top) to y1 (bottom), x1 to x4.
func legacyBoard() *legacy.BoardDefinition {
	b := &legacy.BoardDefinition{ID: "tiny", Name: "Tiny Board"}
	b.Dimensions.Width, b.Dimensions.Height = 4, 3
	b.Rooms = []legacy.Room{
		{ID: 1, Name: "Entry", Tiles: []legacy.TileCoordinate{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 1}}},
		{ID: 2, Name: "Vault", Tiles: []legacy.TileCoordinate{{X: 3, Y: 2}}},
	}
	return b
}

func TestBoardFromLegacy(t *testing.T) {
	b, err := BoardFromLegacy(legacyBoard())
	if err != nil {
		t.Fatal(err)
	}
	want := rowsTopFirst(
		[]int{1, 1, 0, 0},
		[]int{1, 1, 0, 0},
		[]int{0, 0, 0, 2},
	)
	if b.Width != 4 || b.Height != 3 || !reflect.DeepEqual(b.Regions, want) {
		t.Fatalf("board: %+v", b)
	}
	if !reflect.DeepEqual(b.Rooms, []Room{{ID: 1, Name: "Entry"}, {ID: 2, Name: "Vault"}}) {
		t.Fatalf("rooms: %+v", b.Rooms)
	}
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestBoardFromLegacyRejectsBadRooms(t *testing.T) {
	cases := map[string]func(b *legacy.BoardDefinition){
		"tile off board":    func(b *legacy.BoardDefinition) { b.Rooms[1].Tiles[0] = legacy.TileCoordinate{X: 4, Y: 0} },
		"tile in two rooms": func(b *legacy.BoardDefinition) { b.Rooms[1].Tiles[0] = legacy.TileCoordinate{X: 0, Y: 0} },
		"room id zero":      func(b *legacy.BoardDefinition) { b.Rooms[1].ID = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			def := legacyBoard()
			mutate(def)
			if _, err := BoardFromLegacy(def); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestQuestFromLegacy(t *testing.T) {
	board, err := BoardFromLegacy(legacyBoard())
	if err != nil {
		t.Fatal(err)
	}
	def := &legacy.QuestDefinition{
		ID:               "q1",
		Name:             "Test Quest",
		Description:      "Find the vault.",
		StartingRoom:     1,
		WanderingMonster: "orc",
		Doors: []legacy.QuestDoor{
			{ID: "door-1", X: 2, Y: 1, Orientation: "vertical", State: "closed", Type: "normal"},
			{ID: "door-2", X: 3, Y: 2, Orientation: "horizontal", State: "", Type: "secret"},
		},
		BlockingWalls: []legacy.QuestBlockingWall{
			{ID: "wall-1", X: 2, Y: 0, Orientation: "vertical", Size: 2},
			{ID: "wall-2", X: 2, Y: 2, Orientation: "horizontal", Size: 0},
		},
		Furniture: []legacy.QuestFurniture{{ID: "furniture-1", Type: "table", X: 0, Y: 0, Rotation: 90, SwapAspectOnRotate: true}},
		Monsters:  []legacy.QuestMonster{{ID: "monster-1", Type: "orc", X: 3, Y: 2, Notes: "Guards the vault"}},
		QuestNotes: map[string]*legacy.QuestTreasureNote{
			"B": {NoteID: "B", Location: legacy.TreasureLocation{X: 3, Y: 2}, Description: "The vault is empty."},
			"A": {NoteID: "A", Location: legacy.TreasureLocation{X: 1, Y: 1}, Description: "A rusty key."},
		},
	}

	q, err := QuestFromLegacy(def, board, sizes)
	if err != nil {
		t.Fatal(err)
	}

	if q.BoardChecksum != board.Checksum() || q.Description != "Find the vault." || q.WanderingMonster != "orc" {
		t.Fatalf("quest header: %+v", q)
	}
	wantDoors := []Door{
		// Legacy left side of (2,1) is the left side of (3,2).
		{ID: "door-1", Edge: Edge{X: 3, Y: 2, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},
		// Legacy top side of (3,2) (the vault) is the bottom side of (4,2).
		{ID: "door-2", Edge: Edge{X: 4, Y: 2, Orientation: Horizontal}, Kind: DoorSecret, State: DoorClosed},
	}
	if !reflect.DeepEqual(q.Doors, wantDoors) {
		t.Fatalf("doors: %+v", q.Doors)
	}
	// Rectangles are anchored at their bottom-left square.
	wantBlocked := []Rect{{ID: "wall-1", X: 3, Y: 2, W: 1, H: 2}, {ID: "wall-2", X: 3, Y: 1, W: 1, H: 1}}
	if !reflect.DeepEqual(q.BlockedSquares, wantBlocked) {
		t.Fatalf("blocked squares: %+v", q.BlockedSquares)
	}
	// The 2x1 table turned 90 degrees is 1x2, covering legacy (0,0) and (0,1).
	if !reflect.DeepEqual(q.Furniture, []Furniture{{ID: "furniture-1", Type: "table", X: 1, Y: 2, Rotation: 90}}) {
		t.Fatalf("furniture: %+v", q.Furniture)
	}
	if !reflect.DeepEqual(q.Monsters, []Monster{{ID: "monster-1", Type: "orc", X: 4, Y: 1, Notes: "Guards the vault"}}) {
		t.Fatalf("monsters: %+v", q.Monsters)
	}
	wantNotes := []Note{
		{ID: "note-A", Label: "A", X: 2, Y: 2, Text: "A rusty key."},
		{ID: "note-B", Label: "B", X: 4, Y: 1, Text: "The vault is empty."},
	}
	if !reflect.DeepEqual(q.Notes, wantNotes) {
		t.Fatalf("notes: %+v", q.Notes)
	}
	wantStart := []Tile{{X: 1, Y: 2}, {X: 2, Y: 2}, {X: 1, Y: 3}, {X: 2, Y: 3}}
	if !reflect.DeepEqual(q.StartTiles, wantStart) {
		t.Fatalf("start tiles: %+v", q.StartTiles)
	}
	if q.Traps == nil || len(q.Traps) != 0 {
		t.Fatalf("traps should be an empty list: %+v", q.Traps)
	}
	if err := q.Validate(); err != nil {
		t.Fatal(err)
	}
	if issues := q.Check(board, sizes); len(issues) != 0 {
		t.Fatalf("converted quest has placement issues: %+v", issues)
	}
}

func TestQuestFromLegacyRejectsBadOrientation(t *testing.T) {
	board, _ := BoardFromLegacy(legacyBoard())
	def := &legacy.QuestDefinition{Doors: []legacy.QuestDoor{{ID: "d", X: 1, Y: 1, Orientation: "sideways"}}}
	if _, err := QuestFromLegacy(def, board, sizes); err == nil {
		t.Fatal("expected an error")
	}
}
