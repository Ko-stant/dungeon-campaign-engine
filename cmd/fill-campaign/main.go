// Command fill-campaign loads a campaign's agreed combat numbers into the app:
// the custom hero classes (stats and abilities), the campaign's monster stat
// lines and each hero's starting kit (see internal/campaignfill). Without
// -apply it only reports what it would change.
//
//	go run ./cmd/fill-campaign -campaign "Three Plagues"          # dry run
//	go run ./cmd/fill-campaign -campaign "Three Plagues" -apply
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/campaignfill"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func main() {
	file := flag.String("file", "docs/campaigns/three-plagues/combat.json", "combat numbers (classes, starting kit, monsters)")
	campaign := flag.String("campaign", "", "campaign name or id")
	apply := flag.Bool("apply", false, "save the changes (without it, only report them)")
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "Postgres URL (defaults to $DATABASE_URL)")
	flag.Parse()
	log.SetFlags(0)

	raw, err := os.ReadFile(*file)
	if err != nil {
		log.Fatal(err)
	}
	data, err := campaignfill.Load(raw)
	if err != nil {
		log.Fatalf("%s: %v", *file, err)
	}
	if *dbURL == "" {
		log.Fatal("no database URL: set DATABASE_URL (see .env) or pass -db")
	}
	ctx := context.Background()
	if err := store.Migrate(ctx, *dbURL); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(ctx, *dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	report, err := campaignfill.Run(ctx, st, data, *campaign, *apply)
	if err != nil {
		log.Fatal(err)
	}
	for _, line := range report {
		fmt.Println(line)
	}
}
