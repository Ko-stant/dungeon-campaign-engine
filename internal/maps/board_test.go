package maps

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestNewBoardIsAllVoid(t *testing.T) {
	b, err := NewBoard(3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if b.Width != 3 || b.Height != 2 || len(b.Regions) != 6 || len(b.Rooms) != 0 {
		t.Fatalf("unexpected board: %+v", b)
	}
	for i, r := range b.Regions {
		if r != Void {
			t.Fatalf("region %d = %d, want Void", i, r)
		}
	}
	if err := b.Validate(); err != nil {
		t.Fatalf("new board should validate: %v", err)
	}
}

func TestNewBoardRejectsBadSizes(t *testing.T) {
	for _, size := range [][2]int{{0, 1}, {1, 0}, {MaxBoardSize + 1, 1}, {-3, 4}} {
		if _, err := NewBoard(size[0], size[1]); err == nil {
			t.Errorf("NewBoard(%d, %d): expected an error", size[0], size[1])
		}
	}
}

func TestBoardValidate(t *testing.T) {
	valid := func() *Board {
		return &Board{Version: CurrentVersion, Width: 2, Height: 2, Regions: []int{Void, Corridor, 1, 1}, Rooms: []Room{{ID: 1, Name: "Hall"}}}
	}
	if err := valid().Validate(); err != nil {
		t.Fatalf("valid board: %v", err)
	}

	cases := map[string]func(b *Board){
		"too wide":           func(b *Board) { b.Width = MaxBoardSize + 1 },
		"regions length":     func(b *Board) { b.Regions = b.Regions[:3] },
		"region below void":  func(b *Board) { b.Regions[0] = -2 },
		"unknown room":       func(b *Board) { b.Regions[2] = 7 },
		"duplicate room id":  func(b *Board) { b.Rooms = append(b.Rooms, Room{ID: 1, Name: "Again"}) },
		"non-positive room":  func(b *Board) { b.Rooms = append(b.Rooms, Room{ID: 0, Name: "Zero"}) },
		"future doc version": func(b *Board) { b.Version = CurrentVersion + 1 },
		"top-left version 1": func(b *Board) { b.Version = 1 },
		"missing version":    func(b *Board) { b.Version = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			b := valid()
			mutate(b)
			if err := b.Validate(); err == nil {
				t.Fatal("expected a validation error")
			}
		})
	}
}

func TestRegionAtTreatsOffBoardAsVoid(t *testing.T) {
	b := &Board{Width: 2, Height: 1, Regions: []int{Corridor, 3}, Rooms: []Room{{ID: 3}}}
	if got := b.RegionAt(2, 1); got != 3 {
		t.Fatalf("RegionAt(2,1) = %d", got)
	}
	for _, p := range [][2]int{{0, 1}, {3, 1}, {1, 0}, {1, 2}} {
		if got := b.RegionAt(p[0], p[1]); got != Void {
			t.Errorf("RegionAt(%d,%d) = %d, want Void", p[0], p[1], got)
		}
	}
}

func TestWallsMatchTheClientRule(t *testing.T) {
	// The same cases as internal/web/src/board/model.test.ts deriveWalls.
	single := &Board{Width: 1, Height: 1, Regions: []int{Corridor}}
	want := []Edge{
		{X: 1, Y: 1, Orientation: Vertical},
		{X: 2, Y: 1, Orientation: Vertical},
		{X: 1, Y: 1, Orientation: Horizontal},
		{X: 1, Y: 2, Orientation: Horizontal},
	}
	if got := single.Walls(); !reflect.DeepEqual(got, want) {
		t.Fatalf("single tile walls = %+v", got)
	}

	voidEdge := &Board{Width: 3, Height: 1, Regions: []int{Void, Void, Corridor}}
	want = []Edge{
		{X: 3, Y: 1, Orientation: Vertical},
		{X: 4, Y: 1, Orientation: Vertical},
		{X: 3, Y: 1, Orientation: Horizontal},
		{X: 3, Y: 2, Orientation: Horizontal},
	}
	if got := voidEdge.Walls(); !reflect.DeepEqual(got, want) {
		t.Fatalf("void edge walls = %+v", got)
	}

	big, _ := NewBoard(30, 24)
	for i := range big.Regions {
		big.Regions[i] = Corridor
	}
	if got := len(big.Walls()); got != 2*30+2*24 {
		t.Fatalf("30x24 open board has %d walls, want only the perimeter", got)
	}
}

func TestTilesAreOneBasedFromTheBottomLeft(t *testing.T) {
	// 3x2, drawn top row first: row y=2 is "C 1 1", row y=1 is "V C C".
	b := &Board{Version: CurrentVersion, Width: 3, Height: 2, Regions: []int{Void, Corridor, Corridor, Corridor, 1, 1}, Rooms: []Room{{ID: 1}}}
	cases := []struct {
		x, y, index, region int
	}{
		{1, 1, 0, Void},     // bottom-left corner
		{3, 1, 2, Corridor}, // bottom-right
		{1, 2, 3, Corridor}, // top-left
		{3, 2, 5, 1},        // top-right
	}
	for _, c := range cases {
		if got := b.Index(c.x, c.y); got != c.index {
			t.Errorf("Index(%d,%d) = %d, want %d", c.x, c.y, got, c.index)
		}
		if got := b.TileAt(c.index); got != (Tile{X: c.x, Y: c.y}) {
			t.Errorf("TileAt(%d) = %+v, want (%d,%d)", c.index, got, c.x, c.y)
		}
		if got := b.RegionAt(c.x, c.y); got != c.region {
			t.Errorf("RegionAt(%d,%d) = %d, want %d", c.x, c.y, got, c.region)
		}
		if !b.OnBoard(c.x, c.y) {
			t.Errorf("OnBoard(%d,%d) = false", c.x, c.y)
		}
	}
	for _, p := range [][2]int{{0, 1}, {1, 0}, {4, 1}, {1, 3}} {
		if b.OnBoard(p[0], p[1]) {
			t.Errorf("OnBoard(%d,%d) = true, want false", p[0], p[1])
		}
	}
}

func TestChecksumTracksLayoutOnly(t *testing.T) {
	a := &Board{Width: 2, Height: 1, Regions: []int{Corridor, 1}, Rooms: []Room{{ID: 1, Name: "Hall"}}}
	renamed := &Board{Width: 2, Height: 1, Regions: []int{Corridor, 1}, Rooms: []Room{{ID: 1, Name: "Great Hall"}}}
	repainted := &Board{Width: 2, Height: 1, Regions: []int{1, 1}, Rooms: []Room{{ID: 1, Name: "Hall"}}}
	reshaped := &Board{Width: 1, Height: 2, Regions: []int{Corridor, 1}, Rooms: []Room{{ID: 1, Name: "Hall"}}}

	if a.Checksum() != renamed.Checksum() {
		t.Error("renaming a room must not change the checksum")
	}
	if a.Checksum() == repainted.Checksum() {
		t.Error("repainting a tile must change the checksum")
	}
	if a.Checksum() == reshaped.Checksum() {
		t.Error("changing dimensions must change the checksum")
	}
	if !strings.HasPrefix(a.Checksum(), "sha256:") {
		t.Errorf("checksum %q should be prefixed with its algorithm", a.Checksum())
	}
}

func TestBoardJSONRoundTrip(t *testing.T) {
	b := &Board{Version: CurrentVersion, Width: 2, Height: 1, Regions: []int{Void, 1}, Rooms: []Room{{ID: 1, Name: "Crypt"}}}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"regions":[-1,1]`) {
		t.Fatalf("unexpected JSON: %s", data)
	}
	var back Board
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&back, b) {
		t.Fatalf("round trip mismatch: %+v", back)
	}
}
