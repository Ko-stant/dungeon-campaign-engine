package maps

import (
	"reflect"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
)

// legacyBoard is a synthetic 4x3 legacy board:
//
//	y0: 1 1 . .
//	y1: 1 1 . .
//	y2: . . . 2
func legacyBoard() *geometry.BoardDefinition {
	b := &geometry.BoardDefinition{ID: "tiny", Name: "Tiny Board"}
	b.Dimensions.Width, b.Dimensions.Height = 4, 3
	b.Rooms = []geometry.Room{
		{ID: 1, Name: "Entry", Tiles: []geometry.TileCoordinate{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 1}}},
		{ID: 2, Name: "Vault", Tiles: []geometry.TileCoordinate{{X: 3, Y: 2}}},
	}
	return b
}

func TestBoardFromLegacy(t *testing.T) {
	b, err := BoardFromLegacy(legacyBoard())
	if err != nil {
		t.Fatal(err)
	}
	want := []int{
		1, 1, 0, 0,
		1, 1, 0, 0,
		0, 0, 0, 2,
	}
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
	cases := map[string]func(b *geometry.BoardDefinition){
		"tile off board":    func(b *geometry.BoardDefinition) { b.Rooms[1].Tiles[0] = geometry.TileCoordinate{X: 4, Y: 0} },
		"tile in two rooms": func(b *geometry.BoardDefinition) { b.Rooms[1].Tiles[0] = geometry.TileCoordinate{X: 0, Y: 0} },
		"room id zero":      func(b *geometry.BoardDefinition) { b.Rooms[1].ID = 0 },
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
	def := &geometry.QuestDefinition{
		ID:               "q1",
		Name:             "Test Quest",
		Description:      "Find the vault.",
		StartingRoom:     1,
		WanderingMonster: "orc",
		Doors: []geometry.QuestDoor{
			{ID: "door-1", X: 2, Y: 1, Orientation: "vertical", State: "closed", Type: "normal"},
			{ID: "door-2", X: 3, Y: 2, Orientation: "horizontal", State: "", Type: "secret"},
		},
		BlockingWalls: []geometry.QuestBlockingWall{
			{ID: "wall-1", X: 2, Y: 0, Orientation: "vertical", Size: 2},
			{ID: "wall-2", X: 2, Y: 2, Orientation: "horizontal", Size: 0},
		},
		Furniture: []geometry.QuestFurniture{{ID: "furniture-1", Type: "table", X: 0, Y: 0, Rotation: 90, SwapAspectOnRotate: true}},
		Monsters:  []geometry.QuestMonster{{ID: "monster-1", Type: "orc", X: 3, Y: 2, Notes: "Guards the vault"}},
		QuestNotes: map[string]*geometry.QuestTreasureNote{
			"B": {NoteID: "B", Location: geometry.TreasureLocation{X: 3, Y: 2}, Description: "The vault is empty."},
			"A": {NoteID: "A", Location: geometry.TreasureLocation{X: 1, Y: 1}, Description: "A rusty key."},
		},
	}

	q, err := QuestFromLegacy(def, board)
	if err != nil {
		t.Fatal(err)
	}

	if q.BoardChecksum != board.Checksum() || q.Description != "Find the vault." || q.WanderingMonster != "orc" {
		t.Fatalf("quest header: %+v", q)
	}
	wantDoors := []Door{
		{ID: "door-1", Edge: Edge{X: 2, Y: 1, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},
		{ID: "door-2", Edge: Edge{X: 3, Y: 2, Orientation: Horizontal}, Kind: DoorSecret, State: DoorClosed},
	}
	if !reflect.DeepEqual(q.Doors, wantDoors) {
		t.Fatalf("doors: %+v", q.Doors)
	}
	wantBlocked := []Rect{{ID: "wall-1", X: 2, Y: 0, W: 1, H: 2}, {ID: "wall-2", X: 2, Y: 2, W: 1, H: 1}}
	if !reflect.DeepEqual(q.BlockedSquares, wantBlocked) {
		t.Fatalf("blocked squares: %+v", q.BlockedSquares)
	}
	if !reflect.DeepEqual(q.Furniture, []Furniture{{ID: "furniture-1", Type: "table", X: 0, Y: 0, Rotation: 90}}) {
		t.Fatalf("furniture: %+v", q.Furniture)
	}
	if !reflect.DeepEqual(q.Monsters, []Monster{{ID: "monster-1", Type: "orc", X: 3, Y: 2, Notes: "Guards the vault"}}) {
		t.Fatalf("monsters: %+v", q.Monsters)
	}
	wantNotes := []Note{
		{ID: "note-A", Label: "A", X: 1, Y: 1, Text: "A rusty key."},
		{ID: "note-B", Label: "B", X: 3, Y: 2, Text: "The vault is empty."},
	}
	if !reflect.DeepEqual(q.Notes, wantNotes) {
		t.Fatalf("notes: %+v", q.Notes)
	}
	wantStart := []Tile{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 1}}
	if !reflect.DeepEqual(q.StartTiles, wantStart) {
		t.Fatalf("start tiles: %+v", q.StartTiles)
	}
	if q.Traps == nil || len(q.Traps) != 0 {
		t.Fatalf("traps should be an empty list: %+v", q.Traps)
	}
	if err := q.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestQuestFromLegacyRejectsBadOrientation(t *testing.T) {
	board, _ := BoardFromLegacy(legacyBoard())
	def := &geometry.QuestDefinition{Doors: []geometry.QuestDoor{{ID: "d", X: 1, Y: 1, Orientation: "sideways"}}}
	if _, err := QuestFromLegacy(def, board); err == nil {
		t.Fatal("expected an error")
	}
}
