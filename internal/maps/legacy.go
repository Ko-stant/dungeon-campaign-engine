package maps

import (
	"fmt"
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/legacy"
)

// Legacy files count squares from (0,0) at the top-left, with a horizontal
// edge on the top side of its tile and rectangles anchored at their top-left
// square. These helpers convert to this package's bottom-left, 1-based
// convention on a board of the given height.

func legacyTile(height, x, y int) (int, int) { return x + 1, height - y }

func legacyEdge(height, x, y int, o Orientation) Edge {
	if o == Horizontal {
		// The top side of legacy row y is the bottom side of the row above it.
		return Edge{X: x + 1, Y: height - y + 1, Orientation: o}
	}
	return Edge{X: x + 1, Y: height - y, Orientation: o}
}

// legacyAnchor converts the top-left square of a block h squares tall to its
// bottom-left square.
func legacyAnchor(height, x, y, h int) (int, int) { return x + 1, height - y - h + 1 }

// BoardFromLegacy converts a legacy board.json definition. Legacy boards have
// no solid rock: every tile outside a room is corridor.
func BoardFromLegacy(def *legacy.BoardDefinition) (*Board, error) {
	b, err := NewBoard(def.Dimensions.Width, def.Dimensions.Height)
	if err != nil {
		return nil, err
	}
	for i := range b.Regions {
		b.Regions[i] = Corridor
	}
	for _, room := range def.Rooms {
		if room.ID <= 0 {
			return nil, fmt.Errorf("legacy room %q has invalid id %d", room.Name, room.ID)
		}
		b.Rooms = append(b.Rooms, Room{ID: room.ID, Name: room.Name})
		for _, t := range room.Tiles {
			if t.X < 0 || t.Y < 0 || t.X >= b.Width || t.Y >= b.Height {
				return nil, fmt.Errorf("legacy room %d tile (%d,%d) is off the board", room.ID, t.X, t.Y)
			}
			i := b.Index(legacyTile(b.Height, t.X, t.Y))
			if b.Regions[i] != Corridor {
				return nil, fmt.Errorf("tile (%d,%d) is in rooms %d and %d", t.X, t.Y, b.Regions[i], room.ID)
			}
			b.Regions[i] = room.ID
		}
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b, nil
}

// QuestFromLegacy converts a legacy quest definition placed on board. Furniture
// sizes are needed to find each piece's bottom-left square; a type the lookup
// does not know (or a nil lookup) is treated as 1x1.
func QuestFromLegacy(def *legacy.QuestDefinition, board *Board, sizes SizeLookup) (*Quest, error) {
	h := board.Height
	q := NewQuest(board)
	q.Description = def.Description
	q.WanderingMonster = def.WanderingMonster

	for _, d := range def.Doors {
		orientation, err := legacyOrientation(d.Orientation)
		if err != nil {
			return nil, fmt.Errorf("door %q: %w", d.ID, err)
		}
		kind := DoorNormal
		if d.Type == "secret" {
			kind = DoorSecret
		}
		state := DoorClosed
		if d.State == DoorOpen {
			state = DoorOpen
		}
		q.Doors = append(q.Doors, Door{ID: d.ID, Edge: legacyEdge(h, d.X, d.Y, orientation), Kind: kind, State: state})
	}

	// Legacy "blocking walls" are runs of blocked squares along an orientation.
	for _, w := range def.BlockingWalls {
		orientation, err := legacyOrientation(w.Orientation)
		if err != nil {
			return nil, fmt.Errorf("blocking wall %q: %w", w.ID, err)
		}
		size := max(w.Size, 1)
		r := Rect{ID: w.ID, W: 1, H: size}
		if orientation == Horizontal {
			r.W, r.H = size, 1
		}
		r.X, r.Y = legacyAnchor(h, w.X, w.Y, r.H)
		q.BlockedSquares = append(q.BlockedSquares, r)
	}

	for _, f := range def.Furniture {
		fw, fh, ok := 1, 1, false
		if sizes != nil {
			fw, fh, ok = sizes(f.Type)
		}
		if !ok {
			fw, fh = 1, 1
		}
		_, rh, err := RotatedSize(fw, fh, f.Rotation)
		if err != nil {
			return nil, fmt.Errorf("furniture %q: %w", f.ID, err)
		}
		x, y := legacyAnchor(h, f.X, f.Y, rh)
		q.Furniture = append(q.Furniture, Furniture{ID: f.ID, Type: f.Type, X: x, Y: y, Rotation: f.Rotation})
	}
	for _, m := range def.Monsters {
		x, y := legacyTile(h, m.X, m.Y)
		q.Monsters = append(q.Monsters, Monster{ID: m.ID, Type: m.Type, X: x, Y: y, Notes: m.Notes})
	}

	labels := make([]string, 0, len(def.QuestNotes))
	for label := range def.QuestNotes {
		labels = append(labels, label)
	}
	slices.Sort(labels)
	for _, label := range labels {
		n := def.QuestNotes[label]
		if n == nil {
			continue
		}
		x, y := legacyTile(h, n.Location.X, n.Location.Y)
		q.Notes = append(q.Notes, Note{ID: "note-" + label, Label: label, X: x, Y: y, Text: n.Description})
	}

	if def.StartingRoom > 0 {
		for y := 1; y <= board.Height; y++ {
			for x := 1; x <= board.Width; x++ {
				if board.RegionAt(x, y) == def.StartingRoom {
					q.StartTiles = append(q.StartTiles, Tile{X: x, Y: y})
				}
			}
		}
	}

	if err := q.Validate(); err != nil {
		return nil, err
	}
	return q, nil
}

func legacyOrientation(s string) (Orientation, error) {
	switch s {
	case "vertical":
		return Vertical, nil
	case "horizontal":
		return Horizontal, nil
	}
	return "", fmt.Errorf("invalid orientation %q", s)
}
