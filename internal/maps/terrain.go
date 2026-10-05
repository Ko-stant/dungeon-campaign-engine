package maps

// TerrainSpec lists what a board alone doesn't know: which doors are open and
// which squares block movement or sight right now. The caller decides, for
// example from a session's door states, blocked squares and furniture.
type TerrainSpec struct {
	// OpenDoors are passable and see-through on every edge they cover. Any
	// other door edge on a wall stays a wall.
	OpenDoors []Door
	// Blocked squares can't be entered (rubble, furniture that blocks
	// movement).
	Blocked []Tile
	// SightBlocked squares block lines of sight passing through them (tall
	// furniture). The squares at either end of a line never block it.
	SightBlocked []Tile
}

// Terrain answers movement and sight questions for one moment of play.
// Pieces are not part of it; searches take them separately.
type Terrain struct {
	board        *Board
	drawn        map[Edge]bool
	openDoor     map[Edge]bool
	blocked      map[Tile]bool
	sightBlocked map[Tile]bool
}

// NewTerrain builds a Terrain for a board in the given state.
func NewTerrain(b *Board, spec TerrainSpec) *Terrain {
	t := &Terrain{
		board:        b,
		drawn:        make(map[Edge]bool, len(b.DrawnWalls)),
		openDoor:     make(map[Edge]bool, len(spec.OpenDoors)),
		blocked:      make(map[Tile]bool, len(spec.Blocked)),
		sightBlocked: make(map[Tile]bool, len(spec.SightBlocked)),
	}
	for _, e := range b.DrawnWalls {
		t.drawn[e] = true
	}
	for _, d := range spec.OpenDoors {
		for _, e := range d.Edges() {
			t.openDoor[e] = true
		}
	}
	for _, s := range spec.Blocked {
		t.blocked[s] = true
	}
	for _, s := range spec.SightBlocked {
		t.sightBlocked[s] = true
	}
	return t
}

// Board returns the board the terrain was built for.
func (t *Terrain) Board() *Board { return t.board }

// Passable reports whether an edge can be crossed (and seen through): it is
// not a wall, or an open door covers it.
func (t *Terrain) Passable(e Edge) bool {
	if t.openDoor[e] {
		return true
	}
	ra, rb := t.board.EdgeRegions(e)
	return ra == rb && ra != Void && !t.drawn[e]
}

// Open reports whether a square is on the board, not solid rock and not
// blocked.
func (t *Terrain) Open(s Tile) bool {
	return t.board.RegionAt(s.X, s.Y) != Void && !t.blocked[s]
}

// Step reports whether a piece may move from a to the orthogonally adjacent
// square b, pieces aside.
func (t *Terrain) Step(a, b Tile) bool {
	e, ok := EdgeBetween(a, b)
	return ok && t.Open(b) && t.Passable(e)
}

// BlocksSight reports whether a square blocks lines of sight through it.
func (t *Terrain) BlocksSight(s Tile) bool {
	return t.sightBlocked[s]
}
