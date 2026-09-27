// Package maps defines the board and quest documents built by the map creator
// and stored as jsonb. Conventions match internal/web/src/board:
//
//   - Squares count from (1, 1) at the bottom-left corner: column x runs
//     1..Width left to right and row y runs 1..Height bottom to top.
//   - Regions are row-major starting from the bottom row (index
//     (y-1)*Width + (x-1)): Void (-1) is solid rock, Corridor (0) is open
//     corridor, and positive ids are rooms.
//   - A vertical edge (x, y) is the left side of tile (x, y); a horizontal edge
//     is its bottom side. So vertical x runs 1..Width+1 and horizontal y runs
//     1..Height+1.
//   - Rectangles and furniture are anchored at their bottom-left square and
//     extend right and up.
//   - Walls are derived, never stored: one exists wherever neighbouring tiles
//     belong to different regions, with anything off the board treated as Void.
package maps

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

// CurrentVersion is the document schema version written by this code.
// Version 1 counted from (0,0) at the top-left; version 2 counts from (1,1) at
// the bottom-left. Older documents are rejected rather than misread.
const CurrentVersion = 2

// MaxBoardSize bounds both board dimensions (mirrors the database CHECK).
const MaxBoardSize = 200

// Region ids with special meaning; rooms use positive ids.
const (
	Void     = -1
	Corridor = 0
)

// Orientation of a tile edge.
type Orientation string

const (
	Vertical   Orientation = "vertical"
	Horizontal Orientation = "horizontal"
)

// Edge is a tile edge; see the package comment for the convention.
type Edge struct {
	X           int         `json:"x"`
	Y           int         `json:"y"`
	Orientation Orientation `json:"orientation"`
}

// Tile addresses one square.
type Tile struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Room names a region id.
type Room struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Board is the dungeon layout document.
type Board struct {
	Version int    `json:"version"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Regions []int  `json:"regions"`
	Rooms   []Room `json:"rooms"`
}

func checkSize(width, height int) error {
	if width < 1 || height < 1 || width > MaxBoardSize || height > MaxBoardSize {
		return fmt.Errorf("board size must be 1..%d in each dimension, got %dx%d", MaxBoardSize, width, height)
	}
	return nil
}

// NewBoard returns a blank (all Void) board ready for painting.
func NewBoard(width, height int) (*Board, error) {
	if err := checkSize(width, height); err != nil {
		return nil, err
	}
	regions := make([]int, width*height)
	for i := range regions {
		regions[i] = Void
	}
	return &Board{Version: CurrentVersion, Width: width, Height: height, Regions: regions, Rooms: []Room{}}, nil
}

// Validate reports structural problems that make the document unusable.
func (b *Board) Validate() error {
	if err := checkVersion("board", b.Version); err != nil {
		return err
	}
	if err := checkSize(b.Width, b.Height); err != nil {
		return err
	}
	if len(b.Regions) != b.Width*b.Height {
		return fmt.Errorf("board has %d regions, want %d for %dx%d", len(b.Regions), b.Width*b.Height, b.Width, b.Height)
	}
	rooms := make(map[int]bool, len(b.Rooms))
	for _, r := range b.Rooms {
		if r.ID <= 0 {
			return fmt.Errorf("room id must be positive, got %d", r.ID)
		}
		if rooms[r.ID] {
			return fmt.Errorf("duplicate room id %d", r.ID)
		}
		rooms[r.ID] = true
	}
	var errs []error
	for i, id := range b.Regions {
		switch {
		case id < Void:
			t := b.TileAt(i)
			errs = append(errs, fmt.Errorf("tile (%d,%d) has invalid region %d", t.X, t.Y, id))
		case id > Corridor && !rooms[id]:
			t := b.TileAt(i)
			errs = append(errs, fmt.Errorf("tile (%d,%d) uses room %d, which is not in rooms", t.X, t.Y, id))
		}
		if len(errs) >= 5 {
			break
		}
	}
	return errors.Join(errs...)
}

func checkVersion(kind string, version int) error {
	switch {
	case version > CurrentVersion:
		return fmt.Errorf("%s document version %d is newer than supported version %d", kind, version, CurrentVersion)
	case version < CurrentVersion:
		return fmt.Errorf("%s document version %d uses the old top-left coordinates; re-import or recreate it", kind, version)
	}
	return nil
}

// OnBoard reports whether (x, y) is a square of the board.
func (b *Board) OnBoard(x, y int) bool {
	return x >= 1 && y >= 1 && x <= b.Width && y <= b.Height
}

// Index returns the Regions index of an on-board square.
func (b *Board) Index(x, y int) int {
	return (y-1)*b.Width + (x - 1)
}

// TileAt returns the square at a Regions index.
func (b *Board) TileAt(i int) Tile {
	return Tile{X: i%b.Width + 1, Y: i/b.Width + 1}
}

// RegionAt returns the region of a tile; anything off the board is Void.
func (b *Board) RegionAt(x, y int) int {
	if !b.OnBoard(x, y) {
		return Void
	}
	i := b.Index(x, y)
	if i >= len(b.Regions) {
		return Void
	}
	return b.Regions[i]
}

// Walls returns every wall edge: vertical edges row by row, then horizontal
// edges row by row, from the bottom row up.
func (b *Board) Walls() []Edge {
	var walls []Edge
	for y := 1; y <= b.Height; y++ {
		for x := 1; x <= b.Width+1; x++ {
			if b.RegionAt(x-1, y) != b.RegionAt(x, y) {
				walls = append(walls, Edge{X: x, Y: y, Orientation: Vertical})
			}
		}
	}
	for y := 1; y <= b.Height+1; y++ {
		for x := 1; x <= b.Width; x++ {
			if b.RegionAt(x, y-1) != b.RegionAt(x, y) {
				walls = append(walls, Edge{X: x, Y: y, Orientation: Horizontal})
			}
		}
	}
	return walls
}

// EdgeRegions returns the regions on either side of an edge (left or below first).
func (b *Board) EdgeRegions(e Edge) (int, int) {
	if e.Orientation == Vertical {
		return b.RegionAt(e.X-1, e.Y), b.RegionAt(e.X, e.Y)
	}
	return b.RegionAt(e.X, e.Y-1), b.RegionAt(e.X, e.Y)
}

// Checksum identifies the layout (size and regions, not room names). Quests
// record it so an edit to their board after the fact can be flagged.
func (b *Board) Checksum() string {
	h := sha256.New()
	var buf [8]byte
	write := func(v int) {
		binary.BigEndian.PutUint64(buf[:], uint64(int64(v)))
		h.Write(buf[:])
	}
	write(b.Width)
	write(b.Height)
	for _, r := range b.Regions {
		write(r)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
