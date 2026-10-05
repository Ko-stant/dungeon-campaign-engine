package maps

import (
	"reflect"
	"testing"
)

func TestFootprintTilesRowByRowUpward(t *testing.T) {
	got, err := FootprintTiles(Tile{3, 4}, 2, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []Tile{{3, 4}, {4, 4}, {3, 5}, {4, 5}, {3, 6}, {4, 6}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FootprintTiles = %v, want %v", got, want)
	}
}

func TestFootprintTilesSwapAtQuarterTurns(t *testing.T) {
	for _, rot := range []int{90, 270} {
		got, err := FootprintTiles(Tile{1, 1}, 2, 3, rot)
		if err != nil {
			t.Fatal(err)
		}
		want := []Tile{{1, 1}, {2, 1}, {3, 1}, {1, 2}, {2, 2}, {3, 2}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("rotation %d: %v, want %v", rot, got, want)
		}
	}
}

func TestFootprintTilesRejectsBadRotation(t *testing.T) {
	if _, err := FootprintTiles(Tile{1, 1}, 1, 1, 45); err == nil {
		t.Error("expected an error for rotation 45")
	}
}

func TestDoorCoversBothEdgesOfAWideDoor(t *testing.T) {
	d := Door{Edge: Edge{X: 4, Y: 2, Orientation: Vertical}, Span: 2}
	for _, e := range []Edge{{X: 4, Y: 2, Orientation: Vertical}, {X: 4, Y: 3, Orientation: Vertical}} {
		if !d.Covers(e) {
			t.Errorf("door should cover %v", e)
		}
	}
	for _, e := range []Edge{{X: 4, Y: 4, Orientation: Vertical}, {X: 4, Y: 2, Orientation: Horizontal}, {X: 5, Y: 2, Orientation: Vertical}} {
		if d.Covers(e) {
			t.Errorf("door should not cover %v", e)
		}
	}
	single := Door{Edge: Edge{X: 2, Y: 2, Orientation: Horizontal}}
	if !single.Covers(single.Edge) || single.Covers(Edge{X: 3, Y: 2, Orientation: Horizontal}) {
		t.Error("a one-edge door covers only its own edge")
	}
}
