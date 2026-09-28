package maps

import (
	"os"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/legacy"
)

// TestLegacyQuestOneConvertsCleanly converts the real (gitignored) base board
// and quest-01 and expects no placement issues. Skipped without content/.
func TestLegacyQuestOneConvertsCleanly(t *testing.T) {
	const root = "../../content"
	if _, err := os.Stat(root + "/board.json"); err != nil {
		t.Skip("content/ not present (gitignored)")
	}
	boardDef, err := legacy.LoadBoardFromFile(root + "/board.json")
	if err != nil {
		t.Fatal(err)
	}
	questDef, err := legacy.LoadQuestFromFile(root + "/base/quests/quest-01.json")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}

	board, err := BoardFromLegacy(boardDef)
	if err != nil {
		t.Fatal(err)
	}
	quest, err := QuestFromLegacy(questDef, board, catalog.FurnitureSize)
	if err != nil {
		t.Fatal(err)
	}
	if board.Width != 26 || board.Height != 19 || len(board.Rooms) != 22 {
		t.Fatalf("board: %dx%d with %d rooms", board.Width, board.Height, len(board.Rooms))
	}
	if issues := quest.Check(board, catalog.FurnitureSize, catalog.TrapSize); len(issues) != 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	for _, m := range quest.Monsters {
		if _, ok := catalog.Monster(m.Type); !ok {
			t.Errorf("monster %s has type %q missing from the catalog", m.ID, m.Type)
		}
	}
}
