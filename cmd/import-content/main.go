// Command import-content loads the legacy content/board.json and a quest file
// into the database as map-creator documents. Safe to run repeatedly.
//
//	go run ./cmd/import-content                     # base board + quest-01
//	go run ./cmd/import-content -quest base/quests/quest-02.json
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/legacy"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/seed"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func main() {
	contentDir := flag.String("content", "content", "content directory")
	boardFile := flag.String("board", "board.json", "board file, relative to -content")
	questFile := flag.String("quest", "base/quests/quest-01.json", "quest file, relative to -content")
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "Postgres URL (defaults to $DATABASE_URL)")
	flag.Parse()

	if *dbURL == "" {
		log.Fatal("no database URL: set DATABASE_URL (see .env) or pass -db")
	}
	ctx := context.Background()

	boardDef, err := legacy.LoadBoardFromFile(filepath.Join(*contentDir, *boardFile))
	if err != nil {
		log.Fatal(err)
	}
	questDef, err := legacy.LoadQuestFromFile(filepath.Join(*contentDir, *questFile))
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Migrate(ctx, *dbURL); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(ctx, *dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	res, err := seed.ImportLegacy(ctx, st, boardDef, questDef)
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
