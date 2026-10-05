package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// Hero movement in rules mode (online rules D1-D8): orthogonal steps within
// the rolled movement, through allies but never monsters, ending on an empty
// square; doors are opened for free on the way; each step reveals what the
// hero can see, and a trap that goes off ends the move.

// liveTrapsAt returns the traps on a square that stepping onto sets off:
// hidden or revealed ones (not already triggered, disarmed or removed).
func (a *applier) liveTrapsAt(t maps.Tile) []int {
	var out []int
	for i := range a.s.Traps {
		ts := &a.s.Traps[i]
		if ts.State != maps.TrapHidden && ts.State != maps.TrapRevealed {
			continue
		}
		_, qt, err := a.trap(ts.ID)
		if err != nil {
			continue
		}
		if slices.Contains(a.trapTiles(ts, qt), t) {
			out = append(out, i)
		}
	}
	return out
}

// heroMove is the move a hero can make: through allies, not through monsters
// the heroes know about. Hidden monsters are left out so a path never gives
// them away; walking into one stops the hero instead.
func (a *applier) heroMove(h *Hero, budget int, avoid map[maps.Tile]bool) maps.Move {
	return maps.Move{
		From:   maps.Tile{X: h.X, Y: h.Y},
		Budget: budget,
		Occupied: func(t maps.Tile) bool {
			if avoid[t] || a.heroAt(t, h.ID) != nil {
				return true
			}
			m := a.monsterAt(t, "")
			return m != nil && m.Visibility == MonsterSeen
		},
		Through: func(t maps.Tile) bool { return !avoid[t] && a.heroAt(t, h.ID) != nil },
	}
}

// planPath finds the steps to a square: a shortest way around known traps if
// there is one, otherwise any shortest way. It reports a clear error when
// the square can't be reached or is too far.
func (a *applier) planPath(h *Hero, to maps.Tile, left int) ([]maps.Tile, error) {
	if err := a.onBoard(to.X, to.Y); err != nil {
		return nil, err
	}
	if other := a.heroAt(to, h.ID); other != nil {
		return nil, fmt.Errorf("%s is at (%d,%d)", other.Name, to.X, to.Y)
	}
	if m := a.monsterAt(to, ""); m != nil && m.Visibility == MonsterSeen {
		return nil, fmt.Errorf("%s is at (%d,%d)", m.Name, to.X, to.Y)
	}
	t := a.terrain()
	unlimited := a.s.Board.Width * a.s.Board.Height
	known := map[maps.Tile]bool{}
	for i := range a.s.Traps {
		ts := &a.s.Traps[i]
		if ts.State != maps.TrapRevealed && ts.State != maps.TrapTriggered {
			continue
		}
		if _, qt, err := a.trap(ts.ID); err == nil {
			for _, tile := range a.trapTiles(ts, qt) {
				known[tile] = true
			}
		}
	}
	path, ok := maps.Path(t, a.heroMove(h, unlimited, known), to)
	if !ok || len(path) > left {
		path, ok = maps.Path(t, a.heroMove(h, unlimited, nil), to)
	}
	switch {
	case ok && len(path) == 0:
		return nil, fmt.Errorf("%s is at (%d,%d) already", h.Name, to.X, to.Y)
	case !ok:
		return nil, fmt.Errorf("there is no way to (%d,%d) from (%d,%d)", to.X, to.Y, h.X, h.Y)
	case len(path) > left:
		return nil, fmt.Errorf("(%d,%d) is %s away; %s has only %s of movement left", to.X, to.Y, squares(len(path)), h.Name, squares(left))
	}
	return path, nil
}

// checkPath checks a path the player chose step by step.
func (a *applier) checkPath(h *Hero, path []maps.Tile, left int) error {
	if len(path) == 0 {
		return errors.New("the path is empty")
	}
	if len(path) > left {
		return fmt.Errorf("the path is %s long; %s has only %s of movement left", squares(len(path)), h.Name, squares(left))
	}
	t := a.terrain()
	from := maps.Tile{X: h.X, Y: h.Y}
	for _, step := range path {
		if !t.Step(from, step) {
			return fmt.Errorf("there is no step from (%d,%d) to (%d,%d)", from.X, from.Y, step.X, step.Y)
		}
		if m := a.monsterAt(step, ""); m != nil && m.Visibility == MonsterSeen {
			return fmt.Errorf("%s is at (%d,%d)", m.Name, step.X, step.Y)
		}
		from = step
	}
	if other := a.heroAt(from, h.ID); other != nil {
		return fmt.Errorf("%s is at (%d,%d)", other.Name, from.X, from.Y)
	}
	return nil
}

func (a *applier) turnMove(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		To   *maps.Tile  `json:"to"`
		Path []maps.Tile `json:"path"`
	}](payload)
	if err != nil {
		return "", err
	}
	turn, h, err := a.activeTurn()
	if err != nil {
		return "", err
	}
	switch {
	case turn.MoveDone:
		return "", fmt.Errorf("%s's movement is over this turn", h.Name)
	case turn.MoveRoll == 0:
		return "", fmt.Errorf("%s must roll for movement first", h.Name)
	case turn.MoveLeft == 0:
		return "", fmt.Errorf("%s has no movement left", h.Name)
	case (p.To == nil) == (len(p.Path) == 0):
		return "", errors.New("give either a square to move to or a path")
	}
	path := p.Path
	if p.To != nil {
		if path, err = a.planPath(h, *p.To, turn.MoveLeft); err != nil {
			return "", err
		}
	} else if err := a.checkPath(h, path, turn.MoveLeft); err != nil {
		return "", err
	}
	return a.walk(turn, h, path), nil
}

// walk moves a hero along a checked path one step at a time, revealing what
// they see after each step. A monster in the way (one that was hidden) stops
// them on the last square they may stand on; a trap sets off and ends the
// move.
func (a *applier) walk(turn *Turn, h *Hero, path []maps.Tile) string {
	start := maps.Tile{X: h.X, Y: h.Y}
	stand, steps := start, 0
	var seen seenCount
	trapped, blocked := "", false
	for i, step := range path {
		if a.monsterAt(step, "") != nil {
			blocked = true
			break
		}
		h.X, h.Y = step.X, step.Y
		seen.add(a.revealFrom(step))
		if a.heroAt(step, h.ID) == nil {
			stand, steps = step, i+1
		}
		if traps := a.liveTrapsAt(step); len(traps) > 0 {
			for _, ti := range traps {
				ts := &a.s.Traps[ti]
				_, qt, _ := a.trap(ts.ID)
				trapped += "; " + a.triggerTrap(ts, qt)
			}
			break
		}
	}
	h.X, h.Y = stand.X, stand.Y
	turn.MoveLeft -= steps
	summary := fmt.Sprintf("%s moves %s to (%d,%d)", a.heroLabel(h), squares(steps), stand.X, stand.Y)
	summary += seen.String()
	switch {
	case trapped != "":
		turn.MoveLeft, turn.MoveDone = 0, true
		summary += trapped + "; the move ends"
	case blocked:
		summary += "; something blocks the way"
	}
	return summary
}

// carried returns the hero's item with the given name (any case), or nil.
func carried(h *Hero, name string) *Item {
	if name == "" {
		return nil
	}
	for i := range h.Items {
		if strings.EqualFold(strings.TrimSpace(h.Items[i].Name), strings.TrimSpace(name)) && h.Items[i].Quantity > 0 {
			return &h.Items[i]
		}
	}
	return nil
}

// beside reports whether a hero stands on either side of one of a door's edges.
func beside(h *Hero, d maps.Door) bool {
	at := maps.Tile{X: h.X, Y: h.Y}
	for _, e := range d.Edges() {
		if t1, t2 := maps.EdgeTiles(e); t1 == at || t2 == at {
			return true
		}
	}
	return false
}

// checkDoor checks a hero can open a found, closed door beside them: an
// unlocked one, or a locked one whose key they carry (returned).
func (a *applier) checkDoor(h *Hero, id string) (*DoorState, *Item, error) {
	i := slices.IndexFunc(a.s.Doors, func(d DoorState) bool { return d.ID == id })
	qi := slices.IndexFunc(a.s.Quest.Doors, func(d maps.Door) bool { return d.ID == id })
	if i < 0 || qi < 0 || !a.s.Doors[i].Found {
		return nil, nil, fmt.Errorf("no door %q", id)
	}
	d, qd := &a.s.Doors[i], a.s.Quest.Doors[qi]
	switch {
	case d.State == maps.DoorOpen:
		return nil, nil, fmt.Errorf("%s is open already", a.doorLabel(d.ID))
	case !beside(h, qd):
		return nil, nil, fmt.Errorf("%s is not beside %s", h.Name, a.doorLabel(d.ID))
	case !d.Locked:
		return d, nil, nil
	}
	if key := carried(h, qd.Key); key != nil {
		return d, key, nil
	}
	if qd.Key != "" {
		return nil, nil, fmt.Errorf("%s is locked; it opens with the %s", a.doorLabel(d.ID), qd.Key)
	}
	return nil, nil, fmt.Errorf("%s is locked", a.doorLabel(d.ID))
}

// turnDoor opens a door beside the hero, for free, during their movement
// (online rules D5), and reveals what they now see.
func (a *applier) turnDoor(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Door string `json:"door"`
	}](payload)
	if err != nil {
		return "", err
	}
	turn, h, err := a.activeTurn()
	if err != nil {
		return "", err
	}
	if turn.MoveDone {
		return "", fmt.Errorf("%s's movement is over this turn", h.Name)
	}
	d, key, err := a.checkDoor(h, p.Door)
	if err != nil {
		return "", err
	}
	verb := "opens " + a.doorLabel(d.ID)
	if key != nil {
		d.Locked = false
		verb = fmt.Sprintf("unlocks %s with the %s and opens it", a.doorLabel(d.ID), key.Name)
	}
	d.State, d.Seen = maps.DoorOpen, true
	return fmt.Sprintf("%s %s%s", a.heroLabel(h), verb, a.revealFrom(maps.Tile{X: h.X, Y: h.Y}).String()), nil
}
