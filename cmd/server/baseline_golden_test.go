package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
)

// updateGolden regenerates golden files: go test ./cmd/server -run TestBaselineGolden -update
var updateGolden = flag.Bool("update", false, "rewrite golden files")

// baselineCore captures the board, quest, wall, door, furniture and visibility
// state derived from content/board.json and quest-01. It guards against
// behaviour drift during the toolchain and module upgrades.
type baselineCore struct {
	Width               int                 `json:"width"`
	Height              int                 `json:"height"`
	RegionsCount        int                 `json:"regionsCount"`
	TileRegionIDs       []int               `json:"tileRegionIds"`
	WallsVertical       []string            `json:"wallsVertical"`
	WallsHorizontal     []string            `json:"wallsHorizontal"`
	BlockedTiles        []string            `json:"blockedTiles"`
	Doors               []baselineDoor      `json:"doors"`
	KnownDoors          []string            `json:"knownDoors"`
	Furniture           []string            `json:"furniture"`
	Hero                string              `json:"hero"`
	VisibleRooms        []int               `json:"visibleRoomsFromHero"`
	TileVisibility      []string            `json:"tileVisibilityFromHero"`
	DoorVisibility      map[string]bool     `json:"doorVisibilityFromHero"`
	OpenDoorsVisibility map[string][]string `json:"openDoorsVisibility"` // corridor vantage tile -> grid with every door open
}

type baselineDoor struct {
	ID      string `json:"id"`
	Edge    string `json:"edge"`
	RegionA int    `json:"regionA"`
	RegionB int    `json:"regionB"`
	State   string `json:"state"`
}

func edgeString(e geometry.EdgeAddress) string {
	return fmt.Sprintf("%d,%d,%v", e.X, e.Y, e.Orientation)
}

func TestBaselineGolden(t *testing.T) {
	if _, err := os.Stat("../../content/board.json"); err != nil {
		t.Skip("content/ not present (gitignored); skipping golden baseline")
	}

	board, quest, err := loadGameContent()
	if err != nil {
		t.Fatalf("load content: %v", err)
	}

	furnitureSystem := NewFurnitureSystem(log.New(io.Discard, "", 0))
	if err := furnitureSystem.LoadFurnitureDefinitions("../../content"); err != nil {
		t.Fatalf("load furniture: %v", err)
	}
	if err := furnitureSystem.CreateFurnitureInstancesFromQuest(quest); err != nil {
		t.Fatalf("create furniture: %v", err)
	}

	state, hero, err := initializeGameState(board, quest, furnitureSystem)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}

	core := baselineCore{
		Width:          state.Segment.Width,
		Height:         state.Segment.Height,
		RegionsCount:   state.RegionMap.RegionsCount,
		TileRegionIDs:  state.RegionMap.TileRegionIDs,
		Hero:           fmt.Sprintf("%d,%d", hero.X, hero.Y),
		DoorVisibility: map[string]bool{},
	}

	for _, w := range state.Segment.WallsVertical {
		core.WallsVertical = append(core.WallsVertical, edgeString(w))
	}
	for _, w := range state.Segment.WallsHorizontal {
		core.WallsHorizontal = append(core.WallsHorizontal, edgeString(w))
	}
	for tile := range state.BlockedTiles {
		core.BlockedTiles = append(core.BlockedTiles, fmt.Sprintf("%d,%d", tile.X, tile.Y))
	}
	for id, d := range state.Doors {
		core.Doors = append(core.Doors, baselineDoor{ID: id, Edge: edgeString(d.Edge), RegionA: d.RegionA, RegionB: d.RegionB, State: d.State})
		core.DoorVisibility[id] = isEdgeVisible(state, hero.X, hero.Y, d.Edge)
	}
	for id := range state.KnownDoors {
		core.KnownDoors = append(core.KnownDoors, id)
	}
	for id, f := range furnitureSystem.GetAllInstances() {
		core.Furniture = append(core.Furniture, fmt.Sprintf("%s %s %d,%d r%d", id, f.Type, f.Position.X, f.Position.Y, f.Rotation))
	}
	core.VisibleRooms = computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)

	for y := 0; y < state.Segment.Height; y++ {
		row := make([]byte, state.Segment.Width)
		for x := 0; x < state.Segment.Width; x++ {
			row[x] = '.'
			if isTileCenterVisible(state, hero.X, hero.Y, x, y) {
				row[x] = 'V'
			}
		}
		core.TileVisibility = append(core.TileVisibility, string(row))
	}

	// Second scenario: every door open, viewed from a spread of corridor tiles.
	for _, d := range state.Doors {
		d.State = "open"
	}
	core.OpenDoorsVisibility = map[string][]string{}
	var corridorTiles [][2]int
	for y := 0; y < state.Segment.Height; y++ {
		for x := 0; x < state.Segment.Width; x++ {
			if state.RegionMap.TileRegionIDs[y*state.Segment.Width+x] == state.CorridorRegion {
				corridorTiles = append(corridorTiles, [2]int{x, y})
			}
		}
	}
	for i := 0; i < len(corridorTiles); i += len(corridorTiles) / 5 {
		from := corridorTiles[i]
		var rows []string
		for y := 0; y < state.Segment.Height; y++ {
			row := make([]byte, state.Segment.Width)
			for x := 0; x < state.Segment.Width; x++ {
				row[x] = '.'
				if isTileCenterVisible(state, from[0], from[1], x, y) {
					row[x] = 'V'
				}
			}
			rows = append(rows, string(row))
		}
		core.OpenDoorsVisibility[fmt.Sprintf("%d,%d", from[0], from[1])] = rows
	}

	sort.Strings(core.WallsVertical)
	sort.Strings(core.WallsHorizontal)
	sort.Strings(core.BlockedTiles)
	sort.Strings(core.KnownDoors)
	sort.Strings(core.Furniture)
	sort.Ints(core.VisibleRooms)
	sort.Slice(core.Doors, func(i, j int) bool { return core.Doors[i].ID < core.Doors[j].ID })

	got, err := json.MarshalIndent(core, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')

	goldenPath := filepath.Join("testdata", "golden", "quest01_core.json")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("baseline drift: output differs from %s; diff it against a -update run to inspect", goldenPath)
	}
}
