// Command load-script joins a campaign's read-aloud script folder (numbered
// Markdown files, see internal/script.Assemble), checks it and saves it as a
// campaign's script, the same as pasting it into the campaign page.
//
//	go run ./cmd/load-script -campaign "Three Plagues"
//	go run ./cmd/load-script -print            # write the joined script to stdout
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/script"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func main() {
	dir := flag.String("dir", "docs/campaigns/three-plagues/script", "script folder")
	campaign := flag.String("campaign", "", "campaign name or id")
	printOnly := flag.Bool("print", false, "write the joined script to stdout instead of saving it")
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "Postgres URL (defaults to $DATABASE_URL)")
	flag.Parse()
	log.SetFlags(0)

	text, sources, err := script.Assemble(os.DirFS(*dir))
	if err != nil {
		log.Fatalf("%s: %v", *dir, err)
	}
	sc, err := script.Parse(text)
	if err != nil {
		log.Fatalf("%s: %v", *dir, script.Locate(err, sources))
	}
	if *printOnly {
		fmt.Print(text)
		return
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

	list, err := st.ListCampaigns(ctx)
	if err != nil {
		log.Fatal(err)
	}
	c, err := store.FindCampaign(list, *campaign)
	if err != nil {
		log.Fatal(err)
	}
	if err := st.SetCampaignScript(ctx, c.ID, text); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loaded %d passages in %d sections from %d files into %q.\n", sc.Count(), len(sc.Sections), len(sources), c.Name)
	fmt.Println("A running quest picks it up with Reload in the tracker's Read aloud panel.")
}
