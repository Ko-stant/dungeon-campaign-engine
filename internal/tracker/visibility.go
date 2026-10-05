package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// What the players see on the player screen: the GM shows each piece (a
// monster, a piece of furniture, blocked squares, a door) once the heroes
// can see it. Rooms and corridors are seen when discovered; traps show
// themselves once revealed, triggered or disarmed.

// PlayerSettings are the player screen's settings for a session.
type PlayerSettings struct {
	// HideMonsterBody keeps monsters' Body off the player screen.
	HideMonsterBody bool `json:"hideMonsterBody,omitempty"`
}

// setSeen adds or removes id in a list of seen ids and reports whether it changed.
func setSeen(list *[]string, id string, seen bool) bool {
	has := slices.Contains(*list, id)
	switch {
	case seen && !has:
		*list = append(*list, id)
	case !seen && has:
		*list = slices.DeleteFunc(*list, func(s string) bool { return s == id })
	default:
		return false
	}
	return true
}

func (a *applier) furnitureLabel(f maps.Furniture) string {
	name := f.Type
	if a.catalog != nil {
		if def, ok := a.catalog.FurnitureByID(f.Type); ok {
			name = def.Name
		}
	}
	return fmt.Sprintf("%s (%s)", name, f.ID)
}

func (a *applier) doorLabel(id string) string {
	for _, qd := range a.s.Quest.Doors {
		if qd.ID == id && qd.Kind == maps.DoorGate {
			return "gate " + id
		}
		if qd.ID == id && qd.Kind == maps.DoorExit {
			return "exit door " + id
		}
	}
	return id
}

func (a *applier) seenSet(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID   string `json:"id"`
		Seen bool   `json:"seen"`
	}](payload)
	if err != nil {
		return "", err
	}
	label, extra, changed, err := a.applySeen(p.ID, p.Seen)
	if err != nil {
		return "", err
	}
	if !changed {
		return "", errors.New("nothing changed")
	}
	verb := "Shown to the players: "
	if !p.Seen {
		verb = "Hidden from the players: "
	}
	return verb + label + extra, nil
}

// applySeen shows or hides one piece and names it.
func (a *applier) applySeen(id string, seen bool) (label, extra string, changed bool, err error) {
	if i := slices.IndexFunc(a.s.Quest.Furniture, func(f maps.Furniture) bool { return f.ID == id }); i >= 0 {
		return a.furnitureLabel(a.s.Quest.Furniture[i]), "", setSeen(&a.s.SeenFurniture, id, seen), nil
	}
	if i := slices.IndexFunc(a.s.Quest.BlockedSquares, func(r maps.Rect) bool { return r.ID == id }); i >= 0 {
		r := a.s.Quest.BlockedSquares[i]
		return fmt.Sprintf("blocked squares at (%d,%d)", r.X, r.Y), "", setSeen(&a.s.SeenBlocks, id, seen), nil
	}
	for i := range a.s.Doors {
		d := &a.s.Doors[i]
		if d.ID != id {
			continue
		}
		if d.Seen == seen {
			return a.doorLabel(id), "", false, nil
		}
		d.Seen = seen
		if seen && !d.Found {
			d.Found = true
			extra = " (secret door found)"
		}
		return a.doorLabel(id), extra, true, nil
	}
	if m, _, err := a.monster(id); err == nil {
		want := MonsterHidden
		if seen {
			want = MonsterSeen
		}
		changed := m.Visibility != want
		m.Visibility = want
		return monsterLabel(m), "", changed, nil
	}
	if slices.ContainsFunc(a.s.Traps, func(t TrapState) bool { return t.ID == id }) {
		return "", "", false, errors.New("traps show on the player screen once revealed, triggered or disarmed")
	}
	return "", "", false, fmt.Errorf("nothing to show called %q (monsters, furniture, blocked squares and doors can be shown)", id)
}

func (a *applier) playersSet(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HideMonsterBody *bool `json:"hideMonsterBody"`
	}](payload)
	if err != nil {
		return "", err
	}
	if p.HideMonsterBody == nil || *p.HideMonsterBody == a.s.Players.HideMonsterBody {
		return "", errors.New("nothing changed")
	}
	a.s.Players.HideMonsterBody = *p.HideMonsterBody
	if *p.HideMonsterBody {
		return "Player screen: monster Body hidden", nil
	}
	return "Player screen: monster Body shown", nil
}

// covers reports whether a w x h footprint anchored at (x, y) has a square among on.
func (a *applier) covers(on map[int]bool, x, y, w, h int) bool {
	for yy := y; yy < y+max(h, 1); yy++ {
		for xx := x; xx < x+max(w, 1); xx++ {
			if a.s.Board.OnBoard(xx, yy) && on[a.s.Board.Index(xx, yy)] {
				return true
			}
		}
	}
	return false
}

// doorTouches reports whether any edge of a door borders a square among on
// (the squares on both sides of each edge).
func (a *applier) doorTouches(on map[int]bool, d maps.Door) bool {
	for _, e := range d.Edges() {
		x2, y2 := e.X, e.Y-1
		if e.Orientation == maps.Vertical {
			x2, y2 = e.X-1, e.Y
		}
		if a.covers(on, e.X, e.Y, 1, 1) || a.covers(on, x2, y2, 1, 1) {
			return true
		}
	}
	return false
}

func counted(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// showContentsOn shows the players everything with a square among the board
// indexes: living monsters, furniture, blocked squares, and doors on their
// edges (secret doors only once found; traps never). It describes what
// became seen, e.g. " (seen: 2 monsters, 1 door)", or "".
func (a *applier) showContentsOn(indexes []int) string {
	return a.showContents(indexes).String()
}

// seenCount counts the pieces a reveal showed.
type seenCount struct {
	monsters, furniture, doors, blocks int
}

func (c *seenCount) add(o seenCount) {
	c.monsters += o.monsters
	c.furniture += o.furniture
	c.doors += o.doors
	c.blocks += o.blocks
}

// showContents shows the pieces on the given squares (see showContentsOn)
// and counts what became seen.
func (a *applier) showContents(indexes []int) seenCount {
	on := make(map[int]bool, len(indexes))
	for _, i := range indexes {
		on[i] = true
	}
	var monsters, furniture, doors, blocks int
	for i := range a.s.Monsters {
		m := &a.s.Monsters[i]
		if m.Alive && m.Visibility != MonsterSeen && a.covers(on, m.X, m.Y, m.Width, m.Height) {
			m.Visibility = MonsterSeen
			monsters++
		}
	}
	for _, f := range a.s.Quest.Furniture {
		w, h := 1, 1
		if a.catalog != nil {
			if fw, fh, ok := a.catalog.FurnitureSize(f.Type); ok {
				if rw, rh, err := maps.RotatedSize(fw, fh, f.Rotation); err == nil {
					w, h = rw, rh
				}
			}
		}
		if a.covers(on, f.X, f.Y, w, h) && setSeen(&a.s.SeenFurniture, f.ID, true) {
			furniture++
		}
	}
	for _, r := range a.s.Quest.BlockedSquares {
		if !slices.Contains(a.s.RemovedBlocks, r.ID) && a.covers(on, r.X, r.Y, r.W, r.H) && setSeen(&a.s.SeenBlocks, r.ID, true) {
			blocks++
		}
	}
	for _, qd := range a.s.Quest.Doors {
		for i := range a.s.Doors {
			d := &a.s.Doors[i]
			if d.ID == qd.ID && d.Found && !d.Seen && a.doorTouches(on, qd) {
				d.Seen = true
				doors++
			}
		}
	}
	return seenCount{monsters: monsters, furniture: furniture, doors: doors, blocks: blocks}
}

// String is " (seen: 2 monsters, 1 door)", or "" when nothing was seen.
func (c seenCount) String() string {
	var parts []string
	if c.monsters > 0 {
		parts = append(parts, counted(c.monsters, "monster", "monsters"))
	}
	if c.furniture > 0 {
		parts = append(parts, counted(c.furniture, "piece of furniture", "pieces of furniture"))
	}
	if c.doors > 0 {
		parts = append(parts, counted(c.doors, "door", "doors"))
	}
	if c.blocks > 0 {
		parts = append(parts, counted(c.blocks, "blocked square", "blocked squares"))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (seen: " + strings.Join(parts, ", ") + ")"
}
