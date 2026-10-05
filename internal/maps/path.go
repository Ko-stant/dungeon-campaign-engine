package maps

// Move describes a piece about to move: orthogonal steps only, one square of
// movement per step (online rules D2-D4).
type Move struct {
	// From is the piece's anchor (bottom-left) square.
	From Tile
	// Width and Height are the piece's footprint; 0 means 1.
	Width, Height int
	// Budget is the number of steps the piece may take.
	Budget int
	// Occupied reports squares holding another piece; nil means none are.
	// The caller leaves the moving piece itself out.
	Occupied func(Tile) bool
	// Through reports occupied squares the piece may pass (allies) but not
	// stop on; nil means it may pass none.
	Through func(Tile) bool
}

func (m Move) size() (int, int) {
	return max(m.Width, 1), max(m.Height, 1)
}

func (m Move) occupied(s Tile) bool {
	return m.Occupied != nil && m.Occupied(s)
}

func (m Move) through(s Tile) bool {
	return m.Through != nil && m.Through(s)
}

// step reports whether the piece anchored at a may move one square to the
// adjacent anchor b, and whether it may stop there. Every square of the
// footprint moves across its own edge.
func (m Move) step(t *Terrain, a, b Tile) (enter, stop bool) {
	w, h := m.size()
	dx, dy := b.X-a.X, b.Y-a.Y
	stop = true
	for y := range h {
		for x := range w {
			from := Tile{X: a.X + x, Y: a.Y + y}
			to := Tile{X: from.X + dx, Y: from.Y + dy}
			if !t.Step(from, to) {
				return false, false
			}
			if m.occupied(to) {
				if !m.through(to) {
					return false, false
				}
				stop = false
			}
		}
	}
	return true, stop
}

// search runs a breadth-first search from m.From up to m.Budget steps. It
// returns the cost and the previous anchor of every anchor reached, stoppable
// or not. Neighbors are tried east, north, west, south and the first
// discovery wins, so paths are the same every time.
func search(t *Terrain, m Move) (cost map[Tile]int, prev map[Tile]Tile, stop map[Tile]bool) {
	cost = map[Tile]int{m.From: 0}
	prev = map[Tile]Tile{}
	stop = map[Tile]bool{m.From: true}
	frontier := []Tile{m.From}
	for step := 1; step <= m.Budget && len(frontier) > 0; step++ {
		var next []Tile
		for _, a := range frontier {
			for _, b := range Neighbors4(a) {
				if _, seen := cost[b]; seen {
					continue
				}
				enter, canStop := m.step(t, a, b)
				if !enter {
					continue
				}
				cost[b] = step
				prev[b] = a
				stop[b] = canStop
				next = append(next, b)
			}
		}
		frontier = next
	}
	return cost, prev, stop
}

// Reachable returns every anchor square the piece can end its move on within
// its budget, with the steps it takes; the start square costs 0. Squares the
// piece may only pass through are left out.
func Reachable(t *Terrain, m Move) map[Tile]int {
	cost, _, stop := search(t, m)
	for s := range cost {
		if !stop[s] {
			delete(cost, s)
		}
	}
	return cost
}

// Path returns the steps (excluding the start, ending at to) of a shortest
// move to a square the piece may end on, within its budget. Ties are broken
// the same way every time (see search).
func Path(t *Terrain, m Move, to Tile) ([]Tile, bool) {
	cost, prev, stop := search(t, m)
	n, ok := cost[to]
	if !ok || !stop[to] {
		return nil, false
	}
	path := make([]Tile, n)
	for s, i := to, n-1; i >= 0; s, i = prev[s], i-1 {
		path[i] = s
	}
	return path, true
}

// WalkDistances returns the number of steps from the nearest start square to
// every square the party could walk to, as the encounter planner sees the
// quest: every door is passable and blocks that hide a secret door are open.
// Mirrors walkDistances in internal/web/src/combat/encounters.ts.
func WalkDistances(b *Board, q *Quest) map[Tile]int {
	spec := TerrainSpec{OpenDoors: q.Doors}
	for _, r := range q.BlockedSquares {
		if r.HiddenDoor {
			continue
		}
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				spec.Blocked = append(spec.Blocked, Tile{X: x, Y: y})
			}
		}
	}
	t := NewTerrain(b, spec)
	dist := map[Tile]int{}
	var frontier []Tile
	for _, s := range q.StartTiles {
		if _, seen := dist[s]; !seen && t.Open(s) {
			dist[s] = 0
			frontier = append(frontier, s)
		}
	}
	for step := 1; len(frontier) > 0; step++ {
		var next []Tile
		for _, a := range frontier {
			for _, n := range Neighbors4(a) {
				if _, seen := dist[n]; seen || !t.Step(a, n) {
					continue
				}
				dist[n] = step
				next = append(next, n)
			}
		}
		frontier = next
	}
	return dist
}
