package maps

import (
	"fmt"
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
)

// BoardFromLegacy converts a legacy board.json definition. Legacy boards have
// no solid rock: every tile outside a room is corridor.
func BoardFromLegacy(def *geometry.BoardDefinition) (*Board, error) {
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
			i := t.Y*b.Width + t.X
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

// QuestFromLegacy converts a legacy quest definition placed on board.
func QuestFromLegacy(def *geometry.QuestDefinition, board *Board) (*Quest, error) {
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
		q.Doors = append(q.Doors, Door{ID: d.ID, Edge: Edge{X: d.X, Y: d.Y, Orientation: orientation}, Kind: kind, State: state})
	}

	// Legacy "blocking walls" are runs of blocked squares along an orientation.
	for _, w := range def.BlockingWalls {
		orientation, err := legacyOrientation(w.Orientation)
		if err != nil {
			return nil, fmt.Errorf("blocking wall %q: %w", w.ID, err)
		}
		size := max(w.Size, 1)
		r := Rect{ID: w.ID, X: w.X, Y: w.Y, W: 1, H: size}
		if orientation == Horizontal {
			r.W, r.H = size, 1
		}
		q.BlockedSquares = append(q.BlockedSquares, r)
	}

	for _, f := range def.Furniture {
		q.Furniture = append(q.Furniture, Furniture{ID: f.ID, Type: f.Type, X: f.X, Y: f.Y, Rotation: f.Rotation})
	}
	for _, m := range def.Monsters {
		q.Monsters = append(q.Monsters, Monster{ID: m.ID, Type: m.Type, X: m.X, Y: m.Y, Notes: m.Notes})
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
		q.Notes = append(q.Notes, Note{ID: "note-" + label, Label: label, X: n.Location.X, Y: n.Location.Y, Text: n.Description})
	}

	if def.StartingRoom > 0 {
		for y := 0; y < board.Height; y++ {
			for x := 0; x < board.Width; x++ {
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
