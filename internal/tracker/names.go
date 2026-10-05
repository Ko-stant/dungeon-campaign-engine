package tracker

import (
	"math"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// Pieces as the players know them: by what they are and where they lie
// from a hero, never by the GM's ids (those are for the GM's lines and the
// commands themselves).

// furnitureName is a piece of furniture's catalog name.
func (a *applier) furnitureName(f maps.Furniture) string {
	if a.catalog != nil {
		if def, ok := a.catalog.FurnitureByID(f.Type); ok {
			return def.Name
		}
	}
	return f.Type
}

// questDoor finds a door's quest entry.
func (a *applier) questDoor(id string) maps.Door {
	for _, qd := range a.s.Quest.Doors {
		if qd.ID == id {
			return qd
		}
	}
	return maps.Door{ID: id, Kind: maps.DoorNormal}
}

// doorNoun is what kind of door the players see.
func doorNoun(d maps.Door) string {
	switch d.Kind {
	case maps.DoorGate:
		return "gate"
	case maps.DoorExit:
		return "exit door"
	case maps.DoorSecret:
		return "secret door"
	}
	return "door"
}

// doorName is a door as the players hear of it in the feed: "a door", "a
// gate", "the exit door".
func doorName(d maps.Door) string {
	if d.Kind == maps.DoorExit {
		return "the exit door"
	}
	return "a " + doorNoun(d)
}

// towardTiles is the way from a hero to the middle of some squares, e.g.
// "to the northeast", or "here" when they stand on it.
func towardTiles(h *Hero, tiles []maps.Tile) string {
	if len(tiles) == 0 {
		return "here"
	}
	var x, y float64
	for _, t := range tiles {
		x += float64(t.X) + 0.5
		y += float64(t.Y) + 0.5
	}
	n := float64(len(tiles))
	return toward(h, x/n, y/n)
}

// towardDoor is the way from a hero to the middle of a door's edges.
func towardDoor(h *Hero, d maps.Door) string {
	edges := d.Edges()
	var x, y float64
	for _, e := range edges {
		if e.Orientation == maps.Vertical {
			x, y = x+float64(e.X), y+float64(e.Y)+0.5
		} else {
			x, y = x+float64(e.X)+0.5, y+float64(e.Y)
		}
	}
	n := float64(len(edges))
	return toward(h, x/n, y/n)
}

// toward names the compass direction from the center of a hero's square to
// a point in square units (squares count from 1, y up).
func toward(h *Hero, x, y float64) string {
	dx, dy := x-(float64(h.X)+0.5), y-(float64(h.Y)+0.5)
	if dx == 0 && dy == 0 {
		return "here"
	}
	ns, ew := "north", "east"
	if dy < 0 {
		ns = "south"
	}
	if dx < 0 {
		ew = "west"
	}
	// Within about 22.5 degrees of an axis the direction is that axis.
	const axis = 2.4
	switch ax, ay := math.Abs(dx), math.Abs(dy); {
	case ax > axis*ay:
		return "to the " + ew
	case ay > axis*ax:
		return "to the " + ns
	}
	return "to the " + ns + ew
}

// pieceIDs swaps the GM's labels of the board's pieces in a line for the
// names the players know them by.
func (a *applier) pieceIDs(line string) string {
	for _, f := range a.s.Quest.Furniture {
		line = strings.ReplaceAll(line, a.furnitureLabel(f), a.furnitureName(f))
	}
	for _, qt := range a.s.Quest.Traps {
		name := a.trapKindLabel(qt)
		line = strings.ReplaceAll(line, name+" "+qt.ID, name)
	}
	for _, qd := range a.s.Quest.Doors {
		line = replaceWord(line, a.doorLabel(qd.ID), doorName(qd))
	}
	return line
}

// replaceWord replaces old in s where it stands as a whole word (so door
// "d1" leaves "d10" alone).
func replaceWord(s, old, repl string) string {
	if old == "" {
		return s
	}
	var b strings.Builder
	for {
		i := strings.Index(s, old)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		end := i + len(old)
		if wordEdge(s, i-1) && wordEdge(s, end) {
			b.WriteString(s[:i])
			b.WriteString(repl)
		} else {
			b.WriteString(s[:end])
		}
		s = s[end:]
	}
}

// wordEdge reports whether the byte at i is outside s or not part of a word
// (letters, digits, '-' and '_', as in ids like "door-1").
func wordEdge(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	return !strings.ContainsRune(idChars, rune(s[i]))
}

const idChars = "-_0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
