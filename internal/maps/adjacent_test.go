package maps

import (
	"reflect"
	"testing"
)

func TestEdgeBetweenOrthogonalNeighbors(t *testing.T) {
	cases := []struct {
		a, b Tile
		want Edge
	}{
		{Tile{2, 3}, Tile{3, 3}, Edge{X: 3, Y: 3, Orientation: Vertical}},
		{Tile{3, 3}, Tile{2, 3}, Edge{X: 3, Y: 3, Orientation: Vertical}},
		{Tile{2, 3}, Tile{2, 4}, Edge{X: 2, Y: 4, Orientation: Horizontal}},
		{Tile{2, 4}, Tile{2, 3}, Edge{X: 2, Y: 4, Orientation: Horizontal}},
	}
	for _, c := range cases {
		got, ok := EdgeBetween(c.a, c.b)
		if !ok || got != c.want {
			t.Errorf("EdgeBetween(%v, %v) = %v, %v; want %v", c.a, c.b, got, ok, c.want)
		}
	}
}

func TestEdgeBetweenRejectsNonNeighbors(t *testing.T) {
	for _, b := range []Tile{{2, 2}, {3, 3}, {4, 2}, {2, 4}, {1, 1}} {
		if e, ok := EdgeBetween(Tile{2, 2}, b); ok {
			t.Errorf("EdgeBetween((2,2), %v) = %v, want none", b, e)
		}
	}
}

func TestEdgeTilesAreLeftOrBelowFirst(t *testing.T) {
	a, b := EdgeTiles(Edge{X: 3, Y: 2, Orientation: Vertical})
	if a != (Tile{2, 2}) || b != (Tile{3, 2}) {
		t.Errorf("vertical edge tiles = %v, %v", a, b)
	}
	a, b = EdgeTiles(Edge{X: 3, Y: 2, Orientation: Horizontal})
	if a != (Tile{3, 1}) || b != (Tile{3, 2}) {
		t.Errorf("horizontal edge tiles = %v, %v", a, b)
	}
}

func TestEdgeTilesAndEdgeBetweenAgree(t *testing.T) {
	for _, e := range []Edge{{X: 4, Y: 7, Orientation: Vertical}, {X: 4, Y: 7, Orientation: Horizontal}} {
		a, b := EdgeTiles(e)
		if got, ok := EdgeBetween(a, b); !ok || got != e {
			t.Errorf("EdgeBetween(EdgeTiles(%v)) = %v, %v", e, got, ok)
		}
	}
}

func TestNeighbors4OrderIsEastNorthWestSouth(t *testing.T) {
	got := Neighbors4(Tile{5, 5})
	want := [4]Tile{{6, 5}, {5, 6}, {4, 5}, {5, 4}}
	if got != want {
		t.Errorf("Neighbors4 = %v, want %v", got, want)
	}
}

func TestNeighbors8AddsTheDiagonals(t *testing.T) {
	got := Neighbors8(Tile{5, 5})
	want := [8]Tile{{6, 5}, {6, 6}, {5, 6}, {4, 6}, {4, 5}, {4, 4}, {5, 4}, {6, 4}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Neighbors8 = %v, want %v", got, want)
	}
}

func TestAdjacent(t *testing.T) {
	c := Tile{5, 5}
	if !Adjacent(c, Tile{5, 6}, false) || !Adjacent(c, Tile{4, 5}, false) {
		t.Error("orthogonal neighbors should be adjacent")
	}
	if Adjacent(c, Tile{6, 6}, false) {
		t.Error("a diagonal neighbor is not adjacent without diagonals")
	}
	if !Adjacent(c, Tile{6, 6}, true) {
		t.Error("a diagonal neighbor is adjacent with diagonals")
	}
	if Adjacent(c, c, true) || Adjacent(c, Tile{7, 5}, true) {
		t.Error("the same square and squares two away are not adjacent")
	}
}
