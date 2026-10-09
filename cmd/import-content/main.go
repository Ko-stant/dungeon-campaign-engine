// Command import-content loads the base game's catalog (content/heroes,
// monsters, furniture and traps) into the database, where the server reads
// it, and the legacy content/board.json and a quest file as map-creator
// documents. Safe to run repeatedly.
//
//	go run ./cmd/import-content                     # catalog + base board + quest-01
//	go run ./cmd/import-content -quest base/quests/quest-02.json
//	go run ./cmd/import-content -catalog-only -db "$HOSTED_DATABASE_URL"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/legacy"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/seed"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func main() {
	contentDir := flag.String("content", "content", "content directory")
	boardFile := flag.String("board", "board.json", "board file, relative to -content")
	questFile := flag.String("quest", "base/quests/quest-01.json", "quest file, relative to -content")
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "Postgres URL (defaults to $DATABASE_URL)")
	catalogOnly := flag.Bool("catalog-only", false, "import only the base game's catalog (classes, monsters, furniture, traps)")
	flag.Parse()

	if *dbURL == "" {
		log.Fatal("no database URL: set DATABASE_URL (see .env) or pass -db")
	}
	ctx := context.Background()

	// The catalog to import; its furniture sizes also place each piece of
	// the legacy quest by its bottom-left square.
	catalog, err := content.Load(os.DirFS(*contentDir))
	if err != nil {
		log.Fatal(err)
	}
	if len(catalog.Heroes) == 0 || len(catalog.Monsters) == 0 || len(catalog.Furniture) == 0 || len(catalog.Traps) == 0 {
		log.Fatalf("%s should hold heroes/, monsters/, furniture/ and traps/ (found %d, %d, %d, %d)",
			*contentDir, len(catalog.Heroes), len(catalog.Monsters), len(catalog.Furniture), len(catalog.Traps))
	}

	if err := store.Migrate(ctx, *dbURL); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(ctx, *dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	res, err := seed.ImportCatalog(ctx, st, catalog)
	if err != nil {
		log.Fatal(err)
	}
	for _, kind := range []string{"classes", "monsters", "furniture", "traps"} {
		c := res.ByKind()[kind]
		fmt.Printf("%s: %d created, %d updated, %d unchanged\n", kind, c.Created, c.Updated, c.Unchanged)
	}
	if *catalogOnly {
		return
	}

	boardDef, err := legacy.LoadBoardFromFile(filepath.Join(*contentDir, *boardFile))
	if err != nil {
		log.Fatal(err)
	}
	questDef, err := legacy.LoadQuestFromFile(filepath.Join(*contentDir, *questFile))
	if err != nil {
		log.Fatal(err)
	}
	legacyRes, err := seed.ImportLegacy(ctx, st, boardDef, questDef, catalog.FurnitureSize)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("board %q: %s (%s)\n", boardDef.Name, legacyRes.BoardID, createdOrExisting(legacyRes.BoardCreated))
	fmt.Printf("quest %q: %s (%s)\n", questDef.Name, legacyRes.QuestID, createdOrExisting(legacyRes.QuestCreated))
}

func createdOrExisting(created bool) string {
	if created {
		return "created"
	}
	return "already imported"
}
