package maps

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"
)

// The fixture is computed by the TypeScript client code (bun run parity:gen,
// scripts/parity/fixtures.ts); Go must agree with every case.

type geometryFixture struct {
	Boards []struct {
		Board     Board    `json:"board"`
		Quest     Quest    `json:"quest"`
		Walls     []Edge   `json:"walls"`
		Distances [][3]int `json:"distances"`
	} `json:"boards"`
	Edges []struct {
		A    Tile  `json:"a"`
		B    Tile  `json:"b"`
		Edge *Edge `json:"edge"`
	} `json:"edges"`
	Footprints []struct {
		Origin   Tile   `json:"origin"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Rotation int    `json:"rotation"`
		Tiles    []Tile `json:"tiles"`
	} `json:"footprints"`
	Covers []struct {
		Door   Door `json:"door"`
		Edge   Edge `json:"edge"`
		Covers bool `json:"covers"`
	} `json:"covers"`
}

// sameList treats nil and empty as the same list.
func sameList[T any](a, b []T) bool {
	return (len(a) == 0 && len(b) == 0) || reflect.DeepEqual(a, b)
}

func readGeometry(t *testing.T) geometryFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/parity/geometry.json")
	if err != nil {
		t.Fatal(err)
	}
	var f geometryFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParityWallsAndWalkDistances(t *testing.T) {
	f := readGeometry(t)
	if len(f.Boards) == 0 {
		t.Fatal("no boards")
	}
	for i, c := range f.Boards {
		if err := c.Board.Validate(); err != nil {
			t.Fatalf("board %d: %v", i, err)
		}
		if got := c.Board.Walls(); !sameList(got, c.Walls) {
			t.Errorf("board %d walls:\n%v\nwant\n%v", i, got, c.Walls)
		}
		var got [][3]int
		for tile, d := range WalkDistances(&c.Board, &c.Quest) {
			got = append(got, [3]int{tile.X, tile.Y, d})
		}
		slices.SortFunc(got, func(a, b [3]int) int {
			if a[1] != b[1] {
				return a[1] - b[1]
			}
			return a[0] - b[0]
		})
		if !sameList(got, c.Distances) {
			t.Errorf("board %d distances:\n%v\nwant\n%v", i, got, c.Distances)
		}
	}
}

func TestParityEdgeBetween(t *testing.T) {
	for _, c := range readGeometry(t).Edges {
		got, ok := EdgeBetween(c.A, c.B)
		switch {
		case c.Edge == nil && ok:
			t.Errorf("EdgeBetween(%v, %v) = %v, want none", c.A, c.B, got)
		case c.Edge != nil && (!ok || got != *c.Edge):
			t.Errorf("EdgeBetween(%v, %v) = %v, %v; want %v", c.A, c.B, got, ok, *c.Edge)
		}
	}
}

func TestParityFootprints(t *testing.T) {
	for _, c := range readGeometry(t).Footprints {
		got, err := FootprintTiles(c.Origin, c.Width, c.Height, c.Rotation)
		if err != nil || !reflect.DeepEqual(got, c.Tiles) {
			t.Errorf("FootprintTiles(%v, %d, %d, %d) = %v, %v; want %v", c.Origin, c.Width, c.Height, c.Rotation, got, err, c.Tiles)
		}
	}
}

func TestParityDoorCovers(t *testing.T) {
	for _, c := range readGeometry(t).Covers {
		if got := c.Door.Covers(c.Edge); got != c.Covers {
			t.Errorf("door %v span %d covers %v = %v, want %v", c.Door.Edge, c.Door.Span, c.Edge, got, c.Covers)
		}
	}
}
