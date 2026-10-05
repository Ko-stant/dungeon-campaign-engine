package maps

import (
	"strings"
	"testing"
)

// asciiBoard builds a board from rows written top row first, as they read on
// screen: '#' is solid rock, '.' is corridor and '1'..'9' are rooms. The
// bottom line is row 1.
func asciiBoard(t *testing.T, rows ...string) *Board {
	t.Helper()
	b, err := NewBoard(len(rows[0]), len(rows))
	if err != nil {
		t.Fatal(err)
	}
	rooms := map[int]bool{}
	for i, row := range rows {
		if len(row) != b.Width {
			t.Fatalf("row %d is %d wide, want %d", i, len(row), b.Width)
		}
		y := b.Height - i
		for x, c := range row {
			region := Void
			switch {
			case c == '.':
				region = Corridor
			case c >= '1' && c <= '9':
				region = int(c - '0')
				rooms[region] = true
			case c != '#':
				t.Fatalf("unknown board character %q", c)
			}
			b.Regions[b.Index(x+1, y)] = region
		}
	}
	for id := 1; id <= 9; id++ {
		if rooms[id] {
			b.Rooms = append(b.Rooms, Room{ID: id, Name: "Room " + string(rune('0'+id))})
		}
	}
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	return b
}

// tileSet renders a set of tiles in the same top-first layout: 'x' for a tile
// in the set, '.' otherwise. It makes failures readable.
func tileSet(b *Board, in func(Tile) bool) string {
	var sb strings.Builder
	for y := b.Height; y >= 1; y-- {
		for x := 1; x <= b.Width; x++ {
			if in(Tile{X: x, Y: y}) {
				sb.WriteByte('x')
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func rows(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func TestAsciiBoardReadsTopRowFirst(t *testing.T) {
	b := asciiBoard(t,
		"1#",
		"..",
	)
	if got := b.RegionAt(1, 2); got != 1 {
		t.Errorf("top-left region = %d, want room 1", got)
	}
	if got := b.RegionAt(2, 2); got != Void {
		t.Errorf("top-right region = %d, want Void", got)
	}
	if got := b.RegionAt(1, 1); got != Corridor {
		t.Errorf("bottom-left region = %d, want Corridor", got)
	}
}
