package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// Searching and disarming in rules mode (ONLINE_RULES.md, D9): searching for
// treasure, for traps and for secret doors are three separate actions, none
// of them while a monster is revealed (combat is on). Treasure is searched
// beside a piece of furniture, once per piece for the whole party; a trap
// set on the piece goes off. The GM tells what is found until rewards are
// data.

// Search kinds.
const (
	SearchTreasure = "treasure"
	SearchTraps    = "traps"
	SearchDoors    = "doors"
)

// disarmDie is the Rogue's disarm roll (Nimble Fingers): it fails only on a 1.
const disarmDie = 8

// takeAction marks the turn's action taken; acting after moving ends the
// movement step (D4).
func (turn *Turn) takeAction() {
	turn.Acted = true
	if turn.MoveRoll > 0 {
		turn.MoveDone = true
	}
}

// besideTiles reports whether a hero stands orthogonally beside one of the
// squares, with no wall between.
func (a *applier) besideTiles(h *Hero, tiles []maps.Tile) bool {
	at := maps.Tile{X: h.X, Y: h.Y}
	t := a.terrain()
	for _, tile := range tiles {
		if e, ok := maps.EdgeBetween(at, tile); ok && t.Passable(e) {
			return true
		}
	}
	return false
}

func (a *applier) turnSearch(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Kind      string `json:"kind"`
		Furniture string `json:"furniture"`
	}](payload)
	if err != nil {
		return "", err
	}
	turn, h, err := a.activeTurn()
	if err != nil {
		return "", err
	}
	switch {
	case p.Kind != SearchTreasure && p.Kind != SearchTraps && p.Kind != SearchDoors:
		return "", errors.New("kind must be treasure, traps or doors")
	case turn.Acted:
		return "", fmt.Errorf("%s has already acted this turn", h.Name)
	case a.s.revealedMonster():
		return "", errors.New("no searching while a monster is revealed: combat is on")
	}
	var summary string
	switch p.Kind {
	case SearchTreasure:
		summary, err = a.searchTreasure(h, p.Furniture)
	case SearchTraps:
		summary = a.searchTraps(h)
	default:
		summary = a.searchDoors(h)
	}
	if err != nil {
		return "", err
	}
	turn.takeAction()
	return summary, nil
}

// checkTreasure checks a hero can search a piece of furniture in view for
// treasure: nobody has yet, and the hero stands beside it.
func (a *applier) checkTreasure(h *Hero, id string) (maps.Furniture, []maps.Tile, error) {
	if id == "" {
		return maps.Furniture{}, nil, errors.New("which piece of furniture? Treasure is searched beside one")
	}
	i := slices.IndexFunc(a.s.Quest.Furniture, func(f maps.Furniture) bool { return f.ID == id })
	if i < 0 || !slices.Contains(a.s.SeenFurniture, id) {
		return maps.Furniture{}, nil, fmt.Errorf("no furniture in view called %q", id)
	}
	f := a.s.Quest.Furniture[i]
	tiles := a.furnitureTiles(f)
	switch {
	case slices.Contains(a.s.Rules.Searched, id):
		return f, nil, fmt.Errorf("the %s has already been searched for treasure", a.furnitureLabel(f))
	case !a.besideTiles(h, tiles):
		return f, nil, fmt.Errorf("%s is not beside the %s", h.Name, a.furnitureLabel(f))
	}
	return f, tiles, nil
}

func (a *applier) searchTreasure(h *Hero, id string) (string, error) {
	f, tiles, err := a.checkTreasure(h, id)
	if err != nil {
		return "", err
	}
	a.s.Rules.Searched = append(a.s.Rules.Searched, id)
	summary := fmt.Sprintf("%s searches the %s for treasure", a.heroLabel(h), a.furnitureLabel(f))
	for j := range a.s.Traps {
		ts := &a.s.Traps[j]
		_, qt, err := a.trap(ts.ID)
		if err == nil && qt.FurnitureID == id && (ts.State == maps.TrapHidden || ts.State == maps.TrapRevealed) {
			summary += "; " + a.triggerTrap(ts, qt)
		}
	}
	for _, n := range a.s.Quest.Notes {
		if !slices.Contains(a.s.ConsumedNotes, n.ID) && slices.Contains(tiles, maps.Tile{X: n.X, Y: n.Y}) {
			summary += fmt.Sprintf("; note %s is here", n.Label)
		}
	}
	return summary, nil
}

// inView returns the squares a hero can see, as a set.
func (a *applier) inView(h *Hero) map[maps.Tile]bool {
	out := map[maps.Tile]bool{}
	for _, t := range maps.VisibleTiles(a.terrain(), maps.Tile{X: h.X, Y: h.Y}) {
		out[t] = true
	}
	return out
}

func (a *applier) searchTraps(h *Hero) string {
	view := a.inView(h)
	var found []string
	for i := range a.s.Traps {
		ts := &a.s.Traps[i]
		_, qt, err := a.trap(ts.ID)
		if err != nil || ts.State != maps.TrapHidden || qt.Kind == maps.TrapTrigger {
			continue
		}
		if slices.ContainsFunc(a.trapTiles(ts, qt), func(t maps.Tile) bool { return view[t] }) {
			ts.State = maps.TrapRevealed
			found = append(found, a.trapKindLabel(qt)+" "+ts.ID)
		}
	}
	if len(found) == 0 {
		return a.heroLabel(h) + " searches for traps: found none"
	}
	return a.heroLabel(h) + " searches for traps: found " + strings.Join(found, ", ")
}

func (a *applier) searchDoors(h *Hero) string {
	view := a.inView(h)
	on := map[int]bool{}
	for t := range view {
		on[a.s.Board.Index(t.X, t.Y)] = true
	}
	found := 0
	for _, qd := range a.s.Quest.Doors {
		for i := range a.s.Doors {
			if d := &a.s.Doors[i]; d.ID == qd.ID && !d.Found && a.doorTouches(on, qd) {
				d.Found, d.Seen = true, true
				found++
			}
		}
	}
	// Blocked squares that hide a secret door give it up too.
	for _, r := range a.s.Quest.BlockedSquares {
		if r.HiddenDoor && !slices.Contains(a.s.RemovedBlocks, r.ID) && a.covers(on, r.X, r.Y, r.W, r.H) {
			a.s.RemovedBlocks = append(a.s.RemovedBlocks, r.ID)
			found++
		}
	}
	if found == 0 {
		return a.heroLabel(h) + " searches for secret doors: found none"
	}
	return a.heroLabel(h) + " searches for secret doors: found " + strconv.Itoa(found)
}

// canDisarm reports whether a hero's class may disarm traps.
func (a *applier) canDisarm(h *Hero) bool {
	if a.catalog == nil {
		return false
	}
	def, ok := a.catalog.Hero(h.Class)
	return ok && slices.Contains(def.Exclusives, content.ExclusiveDisarm)
}

// checkDisarm checks a hero can disarm a known trap: their class can, and
// they stand beside the piece it is set on, or beside a free square of a
// floor trap (returned: the square they step onto).
func (a *applier) checkDisarm(h *Hero, id string) (*TrapState, maps.Trap, *maps.Tile, error) {
	if !a.canDisarm(h) {
		return nil, maps.Trap{}, nil, fmt.Errorf("only a class that can disarm traps may disarm one, and %s's cannot", h.Name)
	}
	ts, qt, err := a.trap(id)
	if err != nil || ts.State != maps.TrapRevealed {
		return nil, maps.Trap{}, nil, fmt.Errorf("no known trap called %q", id)
	}
	tiles := a.trapTiles(ts, qt)
	onFurniture := slices.ContainsFunc(a.s.Quest.Furniture, func(f maps.Furniture) bool { return f.ID == qt.FurnitureID })
	if !onFurniture {
		// Only the squares the hero could step onto count.
		open := a.terrain()
		tiles = slices.DeleteFunc(slices.Clone(tiles), func(tile maps.Tile) bool { return !open.Open(tile) })
	}
	if !a.besideTiles(h, tiles) {
		return nil, maps.Trap{}, nil, fmt.Errorf("%s is not beside %s", h.Name, a.trapKindLabel(qt))
	}
	if onFurniture {
		return ts, qt, nil, nil
	}
	from, t := maps.Tile{X: h.X, Y: h.Y}, a.terrain()
	at := tiles[slices.IndexFunc(tiles, func(tile maps.Tile) bool {
		e, ok := maps.EdgeBetween(from, tile)
		return ok && t.Passable(e)
	})]
	if other := a.heroAt(at, h.ID); other != nil {
		return nil, maps.Trap{}, nil, fmt.Errorf("%s stands on %s", other.Name, a.trapKindLabel(qt))
	}
	if m := a.monsterAt(at, ""); m != nil {
		return nil, maps.Trap{}, nil, fmt.Errorf("%s stands on %s", monsterLabel(m), a.trapKindLabel(qt))
	}
	return ts, qt, &at, nil
}

// turnDisarm disarms a known trap (ONLINE_RULES.md): a trap on the floor by
// stepping onto its square from beside it, a trap on furniture from beside
// the piece. The roll is a d8, failing only on a 1, which sets the trap off
// (under the hero, for a floor trap). The step is part of the action and
// costs no movement.
func (a *applier) turnDisarm(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Trap string `json:"trap"`
	}](payload)
	if err != nil {
		return "", err
	}
	turn, h, err := a.activeTurn()
	if err != nil {
		return "", err
	}
	if turn.Acted {
		return "", fmt.Errorf("%s has already acted this turn", h.Name)
	}
	ts, qt, at, err := a.checkDisarm(h, p.Trap)
	if err != nil {
		return "", err
	}
	step := ""
	if at != nil {
		h.X, h.Y = at.X, at.Y
		step = fmt.Sprintf(" steps onto (%d,%d) and", at.X, at.Y)
	}
	r, err := a.roller()
	if err != nil {
		return "", err
	}
	roll := r.Die(disarmDie)
	turn.takeAction()
	label := a.trapKindLabel(qt) + " " + ts.ID
	seen := ""
	if step != "" {
		seen = a.revealFrom(maps.Tile{X: h.X, Y: h.Y}).String()
	}
	if roll == 1 {
		return fmt.Sprintf("%s%s fails to disarm %s (1d%d: 1)%s; %s", a.heroLabel(h), step, label, disarmDie, seen, a.triggerTrap(ts, qt)), nil
	}
	ts.State = maps.TrapDisarmed
	return fmt.Sprintf("%s%s disarms %s (1d%d: %d)%s", a.heroLabel(h), step, label, disarmDie, roll, seen), nil
}
