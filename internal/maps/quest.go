package maps

import (
	"errors"
	"fmt"
)

// Door kinds and states. A gate (portcullis) opens and closes like a door.
// An exit door leads off the map, so it may sit on the board's edge or
// against solid rock without a warning.
const (
	DoorNormal = "normal"
	DoorSecret = "secret"
	DoorGate   = "gate"
	DoorExit   = "exit"
	DoorOpen   = "open"
	DoorClosed = "closed"
)

// Trap states. Any transition between them is allowed during play. A quest
// trap starts in one of the first four; TrapRemoved exists only during play
// (the trap is gone from the board, e.g. after being disarmed).
const (
	TrapHidden    = "hidden"
	TrapRevealed  = "revealed"
	TrapTriggered = "triggered"
	TrapDisarmed  = "disarmed"
	TrapRemoved   = "removed"
)

// TrapTrigger is a built-in marker kind: a square the GM uses to set off their
// own effects (open a secret door, release a boulder). It has no catalog entry.
const TrapTrigger = "trigger"

// MaxDoorSpan is the widest door, in wall edges.
const MaxDoorSpan = 2

// Door sits on a tile edge. Locked is its starting lock; nothing enforces it.
// Span 2 makes it two edges wide (for a double door or gate model): Edge is
// the first, and the second continues along the wall, right for a horizontal
// edge and up for a vertical one. Span 0 means 1.
type Door struct {
	ID     string `json:"id"`
	Edge   Edge   `json:"edge"`
	Kind   string `json:"kind"`
	State  string `json:"state"`
	Locked bool   `json:"locked,omitempty"`
	Span   int    `json:"span,omitempty"`
}

// Edges lists the wall edges the door covers, first to last.
func (d Door) Edges() []Edge {
	out := []Edge{d.Edge}
	for i := 1; i < d.Span; i++ {
		e := d.Edge
		if e.Orientation == Vertical {
			e.Y += i
		} else {
			e.X += i
		}
		out = append(out, e)
	}
	return out
}

// Rect is a block of impassable squares (rubble / blocked-square tiles),
// anchored at its bottom-left square. HiddenDoor marks a block that hides a
// secret door: finding it during play removes the block.
type Rect struct {
	ID         string `json:"id"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	W          int    `json:"w"`
	H          int    `json:"h"`
	HiddenDoor bool   `json:"hiddenDoor,omitempty"`
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
	// Rotation turns a multi-square trap's catalog footprint (0, 90, 180, 270);
	// the anchor is the bottom-left square of the rotated footprint.
	Rotation int `json:"rotation,omitempty"`
	// Label (optional, short) tells traps apart on the board, e.g. trigger "1".
	Label string `json:"label,omitempty"`
}

// Note is a lettered quest note marker ("A", "B", ...) with its text.
type Note struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Text  string `json:"text"`
}

// Teleport is a teleport square. Label (optional, short) lets the GM pair
// squares up, e.g. two squares labeled "1". What it does is up to the GM.
type Teleport struct {
	ID    string `json:"id"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Label string `json:"label,omitempty"`
}

// MaxTeleportLabel bounds a teleport label's length in characters.
const MaxTeleportLabel = 8

// MaxTrapLabel bounds a trap label's length in characters.
const MaxTrapLabel = 8

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
	ExitTiles []Tile `json:"exitTiles"`
	// Teleports are teleport squares.
	Teleports []Teleport `json:"teleports"`
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
		Teleports:      []Teleport{},
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
		if d.Kind != DoorNormal && d.Kind != DoorSecret && d.Kind != DoorGate && d.Kind != DoorExit {
			errs = append(errs, fmt.Errorf("door %q: invalid kind %q", d.ID, d.Kind))
		}
		if d.Span < 0 || d.Span > MaxDoorSpan {
			errs = append(errs, fmt.Errorf("door %q: span must be 1 or 2, got %d", d.ID, d.Span))
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
		if _, _, err := RotatedSize(1, 1, tr.Rotation); err != nil {
			errs = append(errs, fmt.Errorf("trap %q: %w", tr.ID, err))
		}
		if len([]rune(tr.Label)) > MaxTrapLabel {
			errs = append(errs, fmt.Errorf("trap %q: label must be at most %d characters", tr.ID, MaxTrapLabel))
		}
	}
	for _, n := range q.Notes {
		id("note", n.ID)
	}
	for _, tp := range q.Teleports {
		id("teleport", tp.ID)
		if len([]rune(tp.Label)) > MaxTeleportLabel {
			errs = append(errs, fmt.Errorf("teleport %q: label must be at most %d characters", tp.ID, MaxTeleportLabel))
		}
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

// SizeLookup returns a furniture type's or trap kind's unrotated size from the catalog.
type SizeLookup func(kind string) (width, height int, ok bool)

// Check reports placement issues against a board. Furniture types must be in
// the catalog; trap kinds may be free text, and those without an entry are
// single-square markers.
func (q *Quest) Check(b *Board, sizes, trapSizes SizeLookup) []Issue {
	var issues []Issue
	add := func(code, itemID, format string, args ...any) {
		issues = append(issues, Issue{Code: code, ItemID: itemID, Message: fmt.Sprintf(format, args...)})
	}

	if q.BoardChecksum != "" && q.BoardChecksum != b.Checksum() {
		add("board-changed", "", "the board layout changed after this quest was saved; check placements")
	}

	for _, d := range q.Doors {
		// One issue per door: the first edge with a problem.
		for _, e := range d.Edges() {
			if code, msg := b.doorEdgeIssue(e, d.Kind == DoorExit); code != "" {
				add(code, d.ID, "door %s %s", d.ID, msg)
				break
			}
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
		w, h, ok := 0, 0, false
		if trapSizes != nil {
			w, h, ok = trapSizes(tr.Kind)
		}
		if !ok {
			piece(tr.ID, tr.X, tr.Y)
			continue
		}
		rw, rh, err := RotatedSize(w, h, tr.Rotation)
		if err != nil {
			continue // reported by Validate
		}
		if code := b.areaIssue(tr.X, tr.Y, rw, rh); code != "" {
			add("piece-"+code, tr.ID, "%s is %s", tr.ID, issueText(code))
		}
	}
	for _, n := range q.Notes {
		piece(n.ID, n.X, n.Y)
	}
	for _, tp := range q.Teleports {
		piece(tp.ID, tp.X, tp.Y)
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

// roomName returns a room's name, or "room N" when it has none.
func (b *Board) roomName(id int) string {
	for _, r := range b.Rooms {
		if r.ID == id && r.Name != "" {
			return r.Name
		}
	}
	return fmt.Sprintf("room %d", id)
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

// doorEdgeIssue checks one door edge. Exit doors may sit on the board's edge
// or against solid rock, since they lead off the map.
func (b *Board) doorEdgeIssue(e Edge, exit bool) (string, string) {
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
		return "door-off-board", "is off the board"
	case boundary && !exit:
		return "door-on-board-edge", "is on the outer edge of the board"
	case ra == rb && ra > Corridor && !b.IsWall(e):
		// A door in the open middle of a room is almost always a misclick.
		// Doors and gates across a corridor are normal, so they are not flagged.
		return "door-inside-room", fmt.Sprintf("is inside %s at (%d,%d), not on a wall; move it onto the room's wall or draw a wall there", b.roomName(ra), e.X, e.Y)
	case (ra == Void || rb == Void) && !exit:
		return "door-into-void", "opens into solid rock"
	}
	return "", ""
}
