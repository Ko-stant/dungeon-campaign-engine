package maps

import (
	"errors"
	"fmt"
)

// Door kinds and states.
const (
	DoorNormal = "normal"
	DoorSecret = "secret"
	DoorOpen   = "open"
	DoorClosed = "closed"
)

// Trap states. Any transition between them is allowed during play.
const (
	TrapHidden    = "hidden"
	TrapRevealed  = "revealed"
	TrapTriggered = "triggered"
	TrapDisarmed  = "disarmed"
)

// Door sits on a tile edge.
type Door struct {
	ID    string `json:"id"`
	Edge  Edge   `json:"edge"`
	Kind  string `json:"kind"`
	State string `json:"state"`
}

// Rect is a block of impassable squares (rubble / blocked-square tiles),
// anchored at its bottom-left square.
type Rect struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
	W  int    `json:"w"`
	H  int    `json:"h"`
}

// Furniture is placed by the bottom-left square of its rotated footprint; its
// size comes from the furniture catalog and swaps at 90 and 270 degrees.
type Furniture struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Rotation int    `json:"rotation"`
}

// Monster placement. Body and Mind override the catalog stats when set.
type Monster struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Body  *int   `json:"body,omitempty"`
	Mind  *int   `json:"mind,omitempty"`
	Notes string `json:"notes,omitempty"`
}

// Trap on a tile, optionally attached to a piece of furniture (a chest trap).
type Trap struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	FurnitureID string `json:"furnitureId,omitempty"`
	State       string `json:"state"`
}

// Note is a lettered quest note marker ("A", "B", ...) with its text.
type Note struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Text  string `json:"text"`
}

// Quest is the set of layers placed on a board.
type Quest struct {
	Version          int         `json:"version"`
	BoardChecksum    string      `json:"boardChecksum"`
	Description      string      `json:"description,omitempty"`
	WanderingMonster string      `json:"wanderingMonster,omitempty"`
	Doors            []Door      `json:"doors"`
	BlockedSquares   []Rect      `json:"blockedSquares"`
	Furniture        []Furniture `json:"furniture"`
	Monsters         []Monster   `json:"monsters"`
	Traps            []Trap      `json:"traps"`
	Notes            []Note      `json:"notes"`
	StartTiles       []Tile      `json:"startTiles"`
	// ExitTiles mark where the heroes leave the dungeon.
	ExitTiles []Tile `json:"exitTiles,omitempty"`
}

// NewQuest returns an empty quest bound to the board's current layout.
func NewQuest(b *Board) *Quest {
	return &Quest{
		Version:        CurrentVersion,
		BoardChecksum:  b.Checksum(),
		Doors:          []Door{},
		BlockedSquares: []Rect{},
		Furniture:      []Furniture{},
		Monsters:       []Monster{},
		Traps:          []Trap{},
		Notes:          []Note{},
		StartTiles:     []Tile{},
		ExitTiles:      []Tile{},
	}
}

// RotatedSize returns a piece's footprint after rotation.
func RotatedSize(width, height, rotation int) (int, int, error) {
	switch rotation {
	case 0, 180:
		return width, height, nil
	case 90, 270:
		return height, width, nil
	}
	return 0, 0, fmt.Errorf("rotation must be 0, 90, 180 or 270, got %d", rotation)
}

// Validate reports malformed data. Placement problems are advisory; see Check.
func (q *Quest) Validate() error {
	if err := checkVersion("quest", q.Version); err != nil {
		return err
	}
	var errs []error
	seen := map[string]bool{}
	id := func(kind, v string) {
		if v == "" {
			errs = append(errs, fmt.Errorf("%s has an empty id", kind))
			return
		}
		if seen[v] {
			errs = append(errs, fmt.Errorf("duplicate id %q", v))
		}
		seen[v] = true
	}

	for _, d := range q.Doors {
		id("door", d.ID)
		if d.Edge.Orientation != Vertical && d.Edge.Orientation != Horizontal {
			errs = append(errs, fmt.Errorf("door %q: invalid orientation %q", d.ID, d.Edge.Orientation))
		}
		if d.Kind != DoorNormal && d.Kind != DoorSecret {
			errs = append(errs, fmt.Errorf("door %q: invalid kind %q", d.ID, d.Kind))
		}
		if d.State != DoorOpen && d.State != DoorClosed {
			errs = append(errs, fmt.Errorf("door %q: invalid state %q", d.ID, d.State))
		}
	}
	for _, r := range q.BlockedSquares {
		id("blocked square", r.ID)
		if r.W < 1 || r.H < 1 {
			errs = append(errs, fmt.Errorf("blocked square %q: size must be at least 1x1, got %dx%d", r.ID, r.W, r.H))
		}
	}
	for _, f := range q.Furniture {
		id("furniture", f.ID)
		if f.Type == "" {
			errs = append(errs, fmt.Errorf("furniture %q: empty type", f.ID))
		}
		if _, _, err := RotatedSize(1, 1, f.Rotation); err != nil {
			errs = append(errs, fmt.Errorf("furniture %q: %w", f.ID, err))
		}
	}
	for _, m := range q.Monsters {
		id("monster", m.ID)
		if m.Type == "" {
			errs = append(errs, fmt.Errorf("monster %q: empty type", m.ID))
		}
		if (m.Body != nil && *m.Body < 0) || (m.Mind != nil && *m.Mind < 0) {
			errs = append(errs, fmt.Errorf("monster %q: stat overrides must not be negative", m.ID))
		}
	}
	for _, tr := range q.Traps {
		id("trap", tr.ID)
		switch tr.State {
		case TrapHidden, TrapRevealed, TrapTriggered, TrapDisarmed:
		default:
			errs = append(errs, fmt.Errorf("trap %q: invalid state %q", tr.ID, tr.State))
		}
	}
	for _, n := range q.Notes {
		id("note", n.ID)
	}
	return errors.Join(errs...)
}

// Issue is an advisory placement problem. Issues never block saving: the GM
// may deliberately bend the board.
type Issue struct {
	Code    string `json:"code"`
	ItemID  string `json:"itemId,omitempty"`
	Message string `json:"message"`
}

// SizeLookup returns a furniture type's unrotated size from the catalog.
type SizeLookup func(furnitureType string) (width, height int, ok bool)

// Check reports placement issues against a board.
func (q *Quest) Check(b *Board, sizes SizeLookup) []Issue {
	var issues []Issue
	add := func(code, itemID, format string, args ...any) {
		issues = append(issues, Issue{Code: code, ItemID: itemID, Message: fmt.Sprintf(format, args...)})
	}

	if q.BoardChecksum != "" && q.BoardChecksum != b.Checksum() {
		add("board-changed", "", "the board layout changed after this quest was saved; check placements")
	}

	for _, d := range q.Doors {
		e := d.Edge
		var inRange, boundary bool
		if e.Orientation == Vertical {
			inRange = e.X >= 1 && e.X <= b.Width+1 && e.Y >= 1 && e.Y <= b.Height
			boundary = e.X == 1 || e.X == b.Width+1
		} else {
			inRange = e.X >= 1 && e.X <= b.Width && e.Y >= 1 && e.Y <= b.Height+1
			boundary = e.Y == 1 || e.Y == b.Height+1
		}
		ra, rb := b.EdgeRegions(e)
		switch {
		case !inRange:
			add("door-off-board", d.ID, "door %s is off the board", d.ID)
		case boundary:
			add("door-on-board-edge", d.ID, "door %s is on the outer edge of the board", d.ID)
		case ra == rb && !b.IsWall(e):
			add("door-same-region", d.ID, "door %s does not separate two different areas", d.ID)
		case ra == Void || rb == Void:
			add("door-into-void", d.ID, "door %s opens into solid rock", d.ID)
		}
	}

	for _, r := range q.BlockedSquares {
		if code := b.areaIssue(r.X, r.Y, r.W, r.H); code != "" {
			add("blocked-"+code, r.ID, "blocked square %s is %s", r.ID, issueText(code))
		}
	}

	for _, f := range q.Furniture {
		w, h, ok := 0, 0, false
		if sizes != nil {
			w, h, ok = sizes(f.Type)
		}
		if !ok {
			add("furniture-unknown-type", f.ID, "furniture %s has unknown type %q", f.ID, f.Type)
			continue
		}
		rw, rh, err := RotatedSize(w, h, f.Rotation)
		if err != nil {
			continue // reported by Validate
		}
		if code := b.areaIssue(f.X, f.Y, rw, rh); code != "" {
			add("furniture-"+code, f.ID, "furniture %s is %s", f.ID, issueText(code))
		}
	}

	piece := func(id string, x, y int) {
		switch {
		case !b.OnBoard(x, y):
			add("piece-off-board", id, "%s is off the board", id)
		case b.RegionAt(x, y) == Void:
			add("piece-on-void", id, "%s is on solid rock", id)
		}
	}
	for _, m := range q.Monsters {
		piece(m.ID, m.X, m.Y)
	}
	for _, tr := range q.Traps {
		piece(tr.ID, tr.X, tr.Y)
	}
	for _, n := range q.Notes {
		piece(n.ID, n.X, n.Y)
	}

	for _, s := range q.StartTiles {
		switch {
		case !b.OnBoard(s.X, s.Y):
			add("start-off-board", "", "start tile (%d,%d) is off the board", s.X, s.Y)
		case b.RegionAt(s.X, s.Y) == Void:
			add("start-on-void", "", "start tile (%d,%d) is on solid rock", s.X, s.Y)
		}
	}
	for _, s := range q.ExitTiles {
		switch {
		case !b.OnBoard(s.X, s.Y):
			add("exit-off-board", "", "exit tile (%d,%d) is off the board", s.X, s.Y)
		case b.RegionAt(s.X, s.Y) == Void:
			add("exit-on-void", "", "exit tile (%d,%d) is on solid rock", s.X, s.Y)
		}
	}
	return issues
}

// areaIssue returns "off-board", "on-void" or "" for a rectangle of tiles.
func (b *Board) areaIssue(x, y, w, h int) string {
	onVoid := false
	for ty := y; ty < y+h; ty++ {
		for tx := x; tx < x+w; tx++ {
			if !b.OnBoard(tx, ty) {
				return "off-board"
			}
			if b.RegionAt(tx, ty) == Void {
				onVoid = true
			}
		}
	}
	if onVoid {
		return "on-void"
	}
	return ""
}

func issueText(code string) string {
	if code == "off-board" {
		return "partly off the board"
	}
	return "partly on solid rock"
}
