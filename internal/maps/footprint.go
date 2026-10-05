package maps

// FootprintTiles lists every tile covered by a piece anchored at its
// bottom-left tile, row by row upward. Mirrors footprintTiles in
// internal/web/src/board/geometry.ts.
func FootprintTiles(origin Tile, width, height, rotation int) ([]Tile, error) {
	w, h, err := RotatedSize(width, height, rotation)
	if err != nil {
		return nil, err
	}
	tiles := make([]Tile, 0, w*h)
	for y := range h {
		for x := range w {
			tiles = append(tiles, Tile{X: origin.X + x, Y: origin.Y + y})
		}
	}
	return tiles, nil
}

// Covers reports whether the door (either edge of a two-wide one) is on edge
// e. Mirrors doorCovers in geometry.ts; look doors up this way, not by their
// stored edge.
func (d Door) Covers(e Edge) bool {
	for _, de := range d.Edges() {
		if de == e {
			return true
		}
	}
	return false
}
