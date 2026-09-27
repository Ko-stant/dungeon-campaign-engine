// Package maps defines the board and quest documents built by the map creator
// and stored as jsonb. Conventions match internal/web/src/board:
//
//   - Tiles are addressed by column x (0..Width-1) and row y (0..Height-1).
//   - Regions are row-major: Void (-1) is solid rock, Corridor (0) is open
//     corridor, and positive ids are rooms.
//   - A vertical edge (x, y) is the left side of tile (x, y); a horizontal edge
//     is its top side.
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
const CurrentVersion = 1

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
	if b.Version > CurrentVersion {
		return fmt.Errorf("board document version %d is newer than supported version %d", b.Version, CurrentVersion)
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
			errs = append(errs, fmt.Errorf("tile (%d,%d) has invalid region %d", i%b.Width, i/b.Width, id))
		case id > Corridor && !rooms[id]:
			errs = append(errs, fmt.Errorf("tile (%d,%d) uses room %d, which is not in rooms", i%b.Width, i/b.Width, id))
		}
		if len(errs) >= 5 {
			break
		}
	}
	return errors.Join(errs...)
}

// RegionAt returns the region of a tile; anything off the board is Void.
func (b *Board) RegionAt(x, y int) int {
	if x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return Void
	}
	i := y*b.Width + x
	if i >= len(b.Regions) {
		return Void
	}
	return b.Regions[i]
}

// Walls returns every wall edge: vertical edges row by row, then horizontal
// edges row by row.
func (b *Board) Walls() []Edge {
	var walls []Edge
	for y := 0; y < b.Height; y++ {
		for x := 0; x <= b.Width; x++ {
			if b.RegionAt(x-1, y) != b.RegionAt(x, y) {
				walls = append(walls, Edge{X: x, Y: y, Orientation: Vertical})
			}
		}
	}
	for y := 0; y <= b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.RegionAt(x, y-1) != b.RegionAt(x, y) {
				walls = append(walls, Edge{X: x, Y: y, Orientation: Horizontal})
			}
		}
	}
	return walls
}

// EdgeRegions returns the regions on either side of an edge (left/up first).
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
