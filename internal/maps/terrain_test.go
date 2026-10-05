package maps

import "testing"

// twoRooms is room 1 (columns 1-2), a corridor (column 3, rows 1-2) and room
// 2 (columns 4-5), with an open door at the bottom left of the corridor and a
// closed one at the top right.
func twoRooms(t *testing.T) (*Board, Door, Door) {
	b := asciiBoard(t,
		"11#22",
		"11.22",
		"11.22",
	)
	open := Door{ID: "d1", Edge: Edge{X: 3, Y: 1, Orientation: Vertical}, Kind: DoorNormal, State: DoorOpen}
	closed := Door{ID: "d2", Edge: Edge{X: 4, Y: 2, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed}
	return b, open, closed
}

func TestTerrainPassableEdges(t *testing.T) {
	b, open, closed := twoRooms(t)
	b.DrawnWalls = []Edge{{X: 2, Y: 2, Orientation: Horizontal}}
	tr := NewTerrain(b, TerrainSpec{OpenDoors: []Door{open}})
	cases := []struct {
		e    Edge
		want bool
		why  string
	}{
		{open.Edge, true, "an open door"},
		{closed.Edge, false, "a closed door is a wall"},
		{Edge{X: 3, Y: 2, Orientation: Vertical}, false, "a derived wall"},
		{Edge{X: 2, Y: 1, Orientation: Vertical}, true, "inside a room"},
		{Edge{X: 3, Y: 2, Orientation: Horizontal}, true, "along the corridor"},
		{Edge{X: 2, Y: 2, Orientation: Horizontal}, false, "a drawn wall"},
		{Edge{X: 1, Y: 1, Orientation: Vertical}, false, "the board's edge"},
	}
	for _, c := range cases {
		if got := tr.Passable(c.e); got != c.want {
			t.Errorf("Passable(%v) = %v, want %v (%s)", c.e, got, c.want, c.why)
		}
	}
}

func TestTerrainOpenSquares(t *testing.T) {
	b, _, _ := twoRooms(t)
	tr := NewTerrain(b, TerrainSpec{Blocked: []Tile{{1, 3}}})
	cases := []struct {
		t    Tile
		want bool
	}{
		{Tile{3, 1}, true},
		{Tile{3, 3}, false}, // solid rock
		{Tile{1, 3}, false}, // blocked square
		{Tile{0, 1}, false}, // off the board
		{Tile{6, 1}, false},
		{Tile{5, 3}, true},
	}
	for _, c := range cases {
		if got := tr.Open(c.t); got != c.want {
			t.Errorf("Open(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}

func TestTerrainStep(t *testing.T) {
	b, open, _ := twoRooms(t)
	tr := NewTerrain(b, TerrainSpec{OpenDoors: []Door{open}, Blocked: []Tile{{1, 2}}})
	if !tr.Step(Tile{2, 1}, Tile{3, 1}) {
		t.Error("stepping through the open door should be allowed")
	}
	if tr.Step(Tile{3, 2}, Tile{4, 2}) {
		t.Error("stepping through the closed door should not be allowed")
	}
	if tr.Step(Tile{1, 1}, Tile{1, 2}) {
		t.Error("stepping onto a blocked square should not be allowed")
	}
	if tr.Step(Tile{1, 1}, Tile{2, 2}) {
		t.Error("diagonal steps are not steps")
	}
}

func TestTerrainSightBlockers(t *testing.T) {
	b, _, _ := twoRooms(t)
	tr := NewTerrain(b, TerrainSpec{SightBlocked: []Tile{{4, 3}}})
	if !tr.BlocksSight(Tile{4, 3}) || tr.BlocksSight(Tile{4, 2}) {
		t.Error("only listed squares block sight")
	}
}

func TestTerrainWideOpenDoor(t *testing.T) {
	b := asciiBoard(t,
		"11",
		"11",
		"..",
	)
	wide := Door{ID: "d", Edge: Edge{X: 1, Y: 2, Orientation: Horizontal}, Kind: DoorNormal, State: DoorOpen, Span: 2}
	tr := NewTerrain(b, TerrainSpec{OpenDoors: []Door{wide}})
	if !tr.Step(Tile{2, 1}, Tile{2, 2}) {
		t.Error("the second edge of a wide open door should be passable")
	}
}
