package geometry

import "testing"

func TestRegionsAcrossDoor(t *testing.T) {
	seg := Segment{
		ID:     "dev",
		Width:  8,
		Height: 6,
		WallsVertical: []EdgeAddress{
			{X: 3, Y: 0, Orientation: Vertical},
			{X: 3, Y: 1, Orientation: Vertical},
			{X: 3, Y: 2, Orientation: Vertical},
			{X: 3, Y: 3, Orientation: Vertical},
			{X: 3, Y: 4, Orientation: Vertical},
			{X: 3, Y: 5, Orientation: Vertical},
		},
		DoorSockets: []EdgeAddress{{X: 3, Y: 2, Orientation: Vertical}},
	}
	rm := BuildRegionMap(seg)
	a, b := RegionsAcrossDoor(rm, seg, seg.DoorSockets[0])
	if a == b {
		t.Fatalf("expected different regions across door, got %d and %d", a, b)
	}
}
