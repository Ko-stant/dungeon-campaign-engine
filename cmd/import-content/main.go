// Command import-content loads the legacy content/board.json and a quest file
// into the database as map-creator documents, and the base game's hero
// classes (content/heroes) into custom_hero_class. Safe to run repeatedly.
//
//	go run ./cmd/import-content                     # base board + quest-01 + classes
//	go run ./cmd/import-content -quest base/quests/quest-02.json
//	go run ./cmd/import-content -classes-only -db "$HOSTED_DATABASE_URL"
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
	classesOnly := flag.Bool("classes-only", false, "import only the base game's hero classes")
	flag.Parse()

	if *dbURL == "" {
		log.Fatal("no database URL: set DATABASE_URL (see .env) or pass -db")
	}
	ctx := context.Background()

	// The catalog: hero classes to import, and furniture sizes, which place
	// each piece by its bottom-left square.
	catalog, err := content.Load(os.DirFS(*contentDir))
	if err != nil {
		log.Fatal(err)
	}
	if len(catalog.Heroes) == 0 {
		log.Fatalf("no hero classes in %s/heroes", *contentDir)
	}

	if err := store.Migrate(ctx, *dbURL); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(ctx, *dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	classes, err := seed.ImportClasses(ctx, st, catalog.Heroes)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("hero classes: %d created, %d updated, %d unchanged\n", classes.Created, classes.Updated, classes.Unchanged)
	if *classesOnly {
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
	res, err := seed.ImportLegacy(ctx, st, boardDef, questDef, catalog.FurnitureSize)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("board %q: %s (%s)\n", boardDef.Name, res.BoardID, createdOrExisting(res.BoardCreated))
	fmt.Printf("quest %q: %s (%s)\n", questDef.Name, res.QuestID, createdOrExisting(res.QuestCreated))
}

func createdOrExisting(created bool) string {
	if created {
		return "created"
	}
	return "already imported"
}
