// Command narration-text writes a campaign's read-aloud script as paste-ready
// text for a text-to-speech tool (see internal/narration): one <id>.txt per
// passage with only the spoken words, and a README.md with each passage's
// notes, speaker turns and the pronunciation of spoken names.
//
//	go run ./cmd/narration-text                  # every passage, into ./narration
//	go run ./cmd/narration-text -only P0,Q1      # the prologue and Quest 1
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/narration"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/script"
)

func main() {
	dir := flag.String("dir", "docs/campaigns/three-plagues/script", "script folder")
	castFile := flag.String("cast", "docs/campaigns/three-plagues/NARRATIVE.md", "narrative with the ## Cast table (voices and pronunciation)")
	out := flag.String("out", "narration", "folder to write (its old .txt files and README.md are replaced)")
	only := flag.String("only", "", "comma-separated passage id prefixes, e.g. P0,Q1 (default: all)")
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
	var cast []narration.Cast
	if *castFile != "" {
		md, err := os.ReadFile(*castFile)
		if err != nil {
			log.Fatal(err)
		}
		cast = narration.ParseCast(string(md))
	}
	var prefixes []string
	if *only != "" {
		prefixes = strings.Split(*only, ",")
	}
	files := narration.Export(sc, cast, prefixes)
	if len(files) == 1 {
		log.Fatalf("no passage id starts with %s", *only)
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	old, err := filepath.Glob(filepath.Join(*out, "*.txt"))
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			log.Fatal(err)
		}
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(*out, f.Name), []byte(f.Text), 0o644); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("Wrote %d passages and README.md to %s/.\n", len(files)-1, *out)
}
