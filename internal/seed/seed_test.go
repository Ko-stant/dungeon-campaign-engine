package seed

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/legacy"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func legacyDefs() (*legacy.BoardDefinition, *legacy.QuestDefinition) {
	b := &legacy.BoardDefinition{ID: "tiny", Name: "Tiny Board"}
	b.Dimensions.Width, b.Dimensions.Height = 3, 2
	b.Rooms = []legacy.Room{{ID: 1, Name: "Entry", Tiles: []legacy.TileCoordinate{{X: 0, Y: 0}, {X: 0, Y: 1}}}}
	q := &legacy.QuestDefinition{
		ID:           "q1",
		Name:         "First Steps",
		StartingRoom: 1,
		Doors:        []legacy.QuestDoor{{ID: "door-1", X: 1, Y: 0, Orientation: "vertical", State: "closed", Type: "normal"}},
		Monsters:     []legacy.QuestMonster{{ID: "monster-1", Type: "orc", X: 2, Y: 1}},
	}
	return b, q
}

func TestImportLegacyCreatesBoardAndQuest(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	boardDef, questDef := legacyDefs()

	res, err := ImportLegacy(ctx, st, boardDef, questDef, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.BoardCreated || !res.QuestCreated || res.BoardID == "" || res.QuestID == "" {
		t.Fatalf("unexpected result: %+v", res)
	}

	storedBoard, err := st.GetBoard(ctx, res.BoardID)
	if err != nil {
		t.Fatal(err)
	}
	if storedBoard.Name != "Tiny Board" || storedBoard.Width != 3 || storedBoard.Height != 2 {
		t.Fatalf("stored board: %+v", storedBoard)
	}
	var board maps.Board
	if err := json.Unmarshal(storedBoard.Doc, &board); err != nil {
		t.Fatal(err)
	}
	if err := board.Validate(); err != nil {
		t.Fatalf("stored board document is invalid: %v", err)
	}

	storedQuest, err := st.GetQuest(ctx, res.QuestID)
	if err != nil {
		t.Fatal(err)
	}
	if storedQuest.BoardID != res.BoardID || storedQuest.Name != "First Steps" {
		t.Fatalf("stored quest: %+v", storedQuest)
	}
	var quest maps.Quest
	if err := json.Unmarshal(storedQuest.Doc, &quest); err != nil {
		t.Fatal(err)
	}
	if quest.BoardChecksum != board.Checksum() || len(quest.Doors) != 1 || len(quest.Monsters) != 1 || len(quest.StartTiles) != 2 {
		t.Fatalf("stored quest document: %+v", quest)
	}
}

func TestImportLegacyIsIdempotent(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	boardDef, questDef := legacyDefs()

	first, err := ImportLegacy(ctx, st, boardDef, questDef, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ImportLegacy(ctx, st, boardDef, questDef, nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.BoardCreated || second.QuestCreated || second.BoardID != first.BoardID || second.QuestID != first.QuestID {
		t.Fatalf("second import should reuse existing rows: first %+v second %+v", first, second)
	}

	boards, _ := st.ListBoards(ctx)
	quests, _ := st.ListQuests(ctx, "")
	if len(boards) != 1 || len(quests) != 1 {
		t.Fatalf("duplicates created: %d boards, %d quests", len(boards), len(quests))
	}
}

func TestImportCatalogIsIdempotent(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	cat := &content.Catalog{
		Heroes: []content.HeroDef{
			{ID: "barbarian", Name: "Barbarian", Body: 8, Mind: 2, Attack: 3, Defense: 2, MovementDice: 2},
			{ID: "elf", Name: "Elf", Body: 6, Mind: 4, Attack: 2, Defense: 2, MovementDice: 2},
		},
		Monsters:  []content.MonsterDef{{ID: "orc", Name: "Orc", Body: 1, Attack: 3, Defense: 2, Movement: 10, Image: "assets/tiles_cleaned/monsters/orc.png"}},
		Furniture: []content.FurnitureDef{{ID: "table", Name: "Table", Width: 3, Height: 2, BlocksMovement: true}},
		Traps:     []content.TrapDef{{ID: "boulder", Name: "Boulder", Width: 1, Height: 1, Movable: true}},
	}
	res, err := ImportCatalog(ctx, st, cat)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Counts{"classes": {Created: 2}, "monsters": {Created: 1}, "furniture": {Created: 1}, "traps": {Created: 1}}
	if got := res.ByKind(); !reflect.DeepEqual(got, want) {
		t.Fatalf("first import: %+v", got)
	}

	cat.Heroes[1].Mind = 5
	cat.Traps[0].Movable = false
	res, err = ImportCatalog(ctx, st, cat)
	if err != nil {
		t.Fatal(err)
	}
	want = map[string]Counts{"classes": {Updated: 1, Unchanged: 1}, "monsters": {Unchanged: 1}, "furniture": {Unchanged: 1}, "traps": {Updated: 1}}
	if got := res.ByKind(); !reflect.DeepEqual(got, want) {
		t.Fatalf("second import: %+v", got)
	}

	// What is stored reads back as the catalog entries.
	classes, _ := st.ListCustomHeroClasses(ctx)
	var elf content.HeroDef
	if err := json.Unmarshal(classes[1].Doc, &elf); err != nil || classes[1].CatalogID != "elf" || elf.Mind != 5 {
		t.Fatalf("the elf: %+v %v", elf, err)
	}
	monsters, _ := st.ListCustomMonsters(ctx)
	var orc content.MonsterDef
	if err := json.Unmarshal(monsters[0].Doc, &orc); err != nil || monsters[0].CatalogID != "orc" || orc.Movement != 10 || orc.Image == "" {
		t.Fatalf("the orc: %+v %v", orc, err)
	}
	furniture, _ := st.ListCatalogPieces(ctx, store.PieceFurniture)
	var table content.FurnitureDef
	if err := json.Unmarshal(furniture[0].Doc, &table); err != nil || table.Width != 3 || !table.BlocksMovement {
		t.Fatalf("the table: %+v %v", table, err)
	}
}
