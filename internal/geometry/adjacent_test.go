package geometry

import "testing"

func TestRegionsAcrossDoor(t *testing.T) {
	// 6x3 board: columns 0-2 are region 1, columns 3-5 are region 2.
	seg := Segment{ID: "test", Width: 6, Height: 3}
	regions := make([]int, 6*3)
	for y := range 3 {
		for x := range 6 {
			if x < 3 {
				regions[y*6+x] = 1
			} else {
				regions[y*6+x] = 2
			}
		}
	}
	rm := RegionMap{TileRegionIDs: regions, RegionsCount: 3}

	// A vertical edge (3, y) is the left side of tile (3, y): it separates the two regions.
	a, b := RegionsAcrossDoor(rm, seg, EdgeAddress{X: 3, Y: 1, Orientation: Vertical})
	if a != 1 || b != 2 {
		t.Fatalf("vertical door at x=3: got %d|%d, want 1|2", a, b)
	}

	// A horizontal edge (1, 1) is the top side of tile (1, 1): both sides are region 1.
	a, b = RegionsAcrossDoor(rm, seg, EdgeAddress{X: 1, Y: 1, Orientation: Horizontal})
	if a != 1 || b != 1 {
		t.Fatalf("horizontal edge inside region 1: got %d|%d", a, b)
	}

	// Edges on the board boundary report -1 for the off-board side.
	a, b = RegionsAcrossDoor(rm, seg, EdgeAddress{X: 0, Y: 0, Orientation: Vertical})
	if a != -1 || b != 1 {
		t.Fatalf("boundary edge: got %d|%d, want -1|1", a, b)
	}
}
