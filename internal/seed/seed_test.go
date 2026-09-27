package seed

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/legacy"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
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
