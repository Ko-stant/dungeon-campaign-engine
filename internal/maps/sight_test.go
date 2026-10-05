package maps

import (
	"math/rand/v2"
	"testing"
)

func TestSightAlongAnOpenRow(t *testing.T) {
	b := asciiBoard(t, ".....")
	if !LineOfSight(openTerrain(t, b), Tile{1, 1}, Tile{5, 1}, nil) {
		t.Error("an empty corridor should be clear")
	}
}

func TestSightIsBlockedByWallsAndClosedDoors(t *testing.T) {
	b, open, closed := twoRooms(t)
	tr := NewTerrain(b, TerrainSpec{OpenDoors: []Door{open}})
	if LineOfSight(tr, Tile{2, 2}, Tile{3, 2}, nil) {
		t.Error("a wall should block sight")
	}
	if !LineOfSight(tr, Tile{1, 1}, Tile{3, 1}, nil) {
		t.Error("an open door should not block sight")
	}
	if LineOfSight(tr, Tile{3, 2}, Tile{5, 2}, nil) {
		t.Error("a closed door should block sight")
	}
	tr = NewTerrain(b, TerrainSpec{OpenDoors: []Door{open, closed}})
	if !LineOfSight(tr, Tile{3, 2}, Tile{5, 2}, nil) {
		t.Error("once opened the door should not block sight")
	}
}

func TestPiecesBlockAttackSightButNotTheView(t *testing.T) {
	b := asciiBoard(t, ".....")
	tr := openTerrain(t, b)
	orc := func(t Tile) bool { return t == Tile{3, 1} }
	if LineOfSight(tr, Tile{1, 1}, Tile{5, 1}, orc) {
		t.Error("a piece in between should block attack sight")
	}
	if !LineOfSight(tr, Tile{1, 1}, Tile{5, 1}, nil) {
		t.Error("pieces should not block the view")
	}
	if !LineOfSight(tr, Tile{1, 1}, Tile{3, 1}, orc) {
		t.Error("the target's own square never blocks")
	}
}

func TestTallFurnitureBlocksSight(t *testing.T) {
	b := asciiBoard(t,
		".....",
		".....",
	)
	tr := NewTerrain(b, TerrainSpec{SightBlocked: []Tile{{3, 1}}})
	// The line from (1,1) to (5,2) runs through (3,1) and (3,2) on its way up.
	if LineOfSight(tr, Tile{1, 1}, Tile{5, 2}, nil) {
		t.Error("the bookcase at (3,1) should block the line")
	}
	tr = NewTerrain(b, TerrainSpec{SightBlocked: []Tile{{4, 1}}})
	if !LineOfSight(tr, Tile{1, 1}, Tile{5, 2}, nil) {
		t.Error("(4,1) is below the line and should not block it")
	}
	tr = NewTerrain(b, TerrainSpec{SightBlocked: []Tile{{5, 2}}})
	if !LineOfSight(tr, Tile{1, 1}, Tile{5, 2}, nil) {
		t.Error("a tall piece itself can be seen")
	}
}

func TestALineThroughACornerIsClearIfEitherWayAroundIs(t *testing.T) {
	oneSide := asciiBoard(t,
		"..",
		".#",
	)
	if !LineOfSight(openTerrain(t, oneSide), Tile{1, 1}, Tile{2, 2}, nil) {
		t.Error("the line only touches the rock's corner; the other side is open")
	}
	bothSides := asciiBoard(t,
		"#.",
		".#",
	)
	if LineOfSight(openTerrain(t, bothSides), Tile{1, 1}, Tile{2, 2}, nil) {
		t.Error("rock on both sides of the corner closes the gap")
	}
}

func TestSightIntoARoomThroughADoorway(t *testing.T) {
	// A corridor along the bottom and a room above it with an open door on
	// the bottom edge of (4,2).
	b := asciiBoard(t,
		"#11111#",
		"#11111#",
		"#11111#",
		".......",
	)
	door := Door{ID: "d", Edge: Edge{X: 4, Y: 2, Orientation: Horizontal}, Kind: DoorNormal, State: DoorOpen}
	tr := NewTerrain(b, TerrainSpec{OpenDoors: []Door{door}})
	// From the corridor's end the line to (4,2) meets the room's wall left of
	// the door; the line to (6,2) passes the door post exactly and is clear.
	if LineOfSight(tr, Tile{1, 1}, Tile{4, 2}, nil) {
		t.Error("(4,2) should be hidden from (1,1)")
	}
	if !LineOfSight(tr, Tile{1, 1}, Tile{6, 2}, nil) {
		t.Error("(6,2) should be visible from (1,1) past the door post")
	}
	got := VisibleTiles(tr, Tile{4, 1})
	in := map[Tile]bool{}
	for _, v := range got {
		in[v] = true
	}
	// A cone through the doorway; lines that graze a door post count as clear.
	want := rows(
		".xxxxx.",
		".xxxxx.",
		"..xxx..",
		"xxxxxxx",
	)
	if s := tileSet(b, func(t Tile) bool { return in[t] }); s != want {
		t.Errorf("visible from below the door:\n%s\nwant:\n%s", s, want)
	}
}

func TestVisibleTilesSeesAWholeOpenRoomAndNothingBehindWalls(t *testing.T) {
	b := asciiBoard(t,
		"111#22",
		"111#22",
	)
	got := VisibleTiles(openTerrain(t, b), Tile{1, 1})
	if len(got) != 6 {
		t.Errorf("visible = %v, want the 6 squares of room 1", got)
	}
	for _, v := range got {
		if b.RegionAt(v.X, v.Y) != 1 {
			t.Errorf("%v is outside room 1", v)
		}
	}
}

func TestSightIsSymmetric(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 30 {
		b, err := NewBoard(9, 7)
		if err != nil {
			t.Fatal(err)
		}
		for i := range b.Regions {
			if r.IntN(5) == 0 {
				b.Regions[i] = Void
			} else {
				b.Regions[i] = Corridor
			}
		}
		var blocked []Tile
		for range 4 {
			blocked = append(blocked, Tile{r.IntN(9) + 1, r.IntN(7) + 1})
		}
		tr := NewTerrain(b, TerrainSpec{SightBlocked: blocked})
		piece := Tile{r.IntN(9) + 1, r.IntN(7) + 1}
		pieces := func(t Tile) bool { return t == piece }
		for range 200 {
			a := Tile{r.IntN(9) + 1, r.IntN(7) + 1}
			c := Tile{r.IntN(9) + 1, r.IntN(7) + 1}
			if LineOfSight(tr, a, c, pieces) != LineOfSight(tr, c, a, pieces) {
				t.Fatalf("sight %v -> %v differs from %v -> %v on\n%s", a, c, c, a, tileSet(b, func(t Tile) bool { return b.RegionAt(t.X, t.Y) == Void }))
			}
		}
	}
}
