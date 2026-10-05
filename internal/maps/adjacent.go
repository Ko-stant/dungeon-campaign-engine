package maps

// EdgeBetween returns the edge shared by two orthogonally adjacent tiles.
// Mirrors edgeBetween in internal/web/src/board/geometry.ts.
func EdgeBetween(a, b Tile) (Edge, bool) {
	dx, dy := b.X-a.X, b.Y-a.Y
	if abs(dx)+abs(dy) != 1 {
		return Edge{}, false
	}
	if dx != 0 {
		return Edge{X: max(a.X, b.X), Y: a.Y, Orientation: Vertical}, true
	}
	return Edge{X: a.X, Y: max(a.Y, b.Y), Orientation: Horizontal}, true
}

// EdgeTiles returns the two tiles an edge separates, left or below first.
// Either may be off the board. Mirrors edgeTiles in geometry.ts.
func EdgeTiles(e Edge) (Tile, Tile) {
	if e.Orientation == Vertical {
		return Tile{X: e.X - 1, Y: e.Y}, Tile{X: e.X, Y: e.Y}
	}
	return Tile{X: e.X, Y: e.Y - 1}, Tile{X: e.X, Y: e.Y}
}

// Neighbors4 returns the orthogonal neighbors in a fixed order (east, north,
// west, south), so searches that use it break ties the same way every time.
func Neighbors4(t Tile) [4]Tile {
	return [4]Tile{{t.X + 1, t.Y}, {t.X, t.Y + 1}, {t.X - 1, t.Y}, {t.X, t.Y - 1}}
}

// Neighbors8 returns all eight neighbors, counterclockwise from the east.
func Neighbors8(t Tile) [8]Tile {
	return [8]Tile{
		{t.X + 1, t.Y}, {t.X + 1, t.Y + 1}, {t.X, t.Y + 1}, {t.X - 1, t.Y + 1},
		{t.X - 1, t.Y}, {t.X - 1, t.Y - 1}, {t.X, t.Y - 1}, {t.X + 1, t.Y - 1},
	}
}

// Adjacent reports whether two tiles touch: orthogonally, or also diagonally
// when diagonal is true. A tile is not adjacent to itself.
func Adjacent(a, b Tile, diagonal bool) bool {
	dx, dy := abs(b.X-a.X), abs(b.Y-a.Y)
	if diagonal {
		return max(dx, dy) == 1
	}
	return dx+dy == 1
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
