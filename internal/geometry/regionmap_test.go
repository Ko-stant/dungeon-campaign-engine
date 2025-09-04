package geometry

import "testing"

func TestBuildRegionMap_SplitsByWallsAndDoorSockets(t *testing.T) {
	seg := Segment{
		ID:     "t",
		Width:  6,
		Height: 4,
		WallsVertical: []EdgeAddress{
			{X: 2, Y: 0, Orientation: Vertical},
			{X: 2, Y: 1, Orientation: Vertical},
			{X: 2, Y: 2, Orientation: Vertical},
			{X: 2, Y: 3, Orientation: Vertical},
		},
		WallsHorizontal: []EdgeAddress{
			{X: 0, Y: 2, Orientation: Horizontal},
			{X: 1, Y: 2, Orientation: Horizontal},
			{X: 2, Y: 2, Orientation: Horizontal},
			{X: 3, Y: 2, Orientation: Horizontal},
			{X: 4, Y: 2, Orientation: Horizontal},
			{X: 5, Y: 2, Orientation: Horizontal},
		},
		DoorSockets: []EdgeAddress{
			{X: 2, Y: 1, Orientation: Vertical},
		},
	}
	rm := BuildRegionMap(seg)
	if rm.RegionsCount <= 2 {
		t.Fatalf("expected more than 2 regions, got %d", rm.RegionsCount)
	}
}
