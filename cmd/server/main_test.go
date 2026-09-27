package main

import (
	"fmt"
	"os"
	"testing"
)

// TestMain skips this package when the gitignored content/ directory is
// absent (fresh clone, CI). Nearly every test here builds a game from
// content/board.json and quest-01, and would otherwise panic.
func TestMain(m *testing.M) {
	if _, err := os.Stat("../../content/board.json"); err != nil {
		fmt.Println("skipping cmd/server tests: content/ not present (it is gitignored)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
