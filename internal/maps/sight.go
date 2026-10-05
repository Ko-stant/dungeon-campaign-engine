package maps

import "cmp"

// LineOfSight reports whether a line from the center of square a to the
// center of square b is clear (online rules D6 and D7):
//
//   - it may not cross a wall or a closed door (an edge that is not Passable);
//   - squares strictly between the two ends block it when the terrain says so
//     (tall furniture) or when pieces reports them (heroes and monsters);
//   - a line through a grid corner is clear if either way around the corner
//     is clear, so a line that only touches a wall corner is not blocked.
//
// Pass the occupied squares as pieces for attack sight (pieces block) and nil
// for the view (what the heroes can see; pieces don't block). The result is
// the same in both directions.
func LineOfSight(t *Terrain, a, b Tile, pieces func(Tile) bool) bool {
	blocks := func(s Tile) bool {
		return s != b && (t.BlocksSight(s) || (pieces != nil && pieces(s)))
	}
	across := func(from, to Tile) bool {
		e, ok := EdgeBetween(from, to)
		return ok && t.Passable(e)
	}
	// Work in doubled coordinates so square centers and grid lines are whole
	// numbers: centers are odd, lines even. The line starts 1 unit from the
	// next grid line on each axis and the lines are 2 apart.
	dx, dy := 2*(b.X-a.X), 2*(b.Y-a.Y)
	sx, sy := cmp.Compare(dx, 0), cmp.Compare(dy, 0)
	adx, ady := abs(dx), abs(dy)
	nextX, nextY := 1, 1
	cur := a
	for cur != b {
		// Which grid line comes first: compare nextX/adx with nextY/ady.
		order := 0
		switch {
		case adx == 0:
			order = 1
		case ady == 0:
			order = -1
		default:
			order = cmp.Compare(nextX*ady, nextY*adx)
		}
		switch {
		case order < 0:
			next := Tile{X: cur.X + sx, Y: cur.Y}
			if !across(cur, next) {
				return false
			}
			cur = next
			nextX += 2
		case order > 0:
			next := Tile{X: cur.X, Y: cur.Y + sy}
			if !across(cur, next) {
				return false
			}
			cur = next
			nextY += 2
		default:
			next := Tile{X: cur.X + sx, Y: cur.Y + sy}
			side1 := Tile{X: cur.X + sx, Y: cur.Y}
			side2 := Tile{X: cur.X, Y: cur.Y + sy}
			around := func(side Tile) bool {
				return across(cur, side) && !blocks(side) && across(side, next)
			}
			if !around(side1) && !around(side2) {
				return false
			}
			cur = next
			nextX += 2
			nextY += 2
		}
		if blocks(cur) {
			return false
		}
	}
	return true
}

// VisibleTiles returns the squares in view from a square (pieces don't block
// the view), row by row from the bottom; solid rock is never returned.
func VisibleTiles(t *Terrain, from Tile) []Tile {
	b := t.Board()
	var out []Tile
	for y := 1; y <= b.Height; y++ {
		for x := 1; x <= b.Width; x++ {
			s := Tile{X: x, Y: y}
			if b.RegionAt(x, y) != Void && LineOfSight(t, from, s, nil) {
				out = append(out, s)
			}
		}
	}
	return out
}
