package tracker

import (
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// The board as the rules see it right now: which edges are open, which
// squares block movement or sight, who stands where, and what the heroes can
// see (online rules D3, D5-D7).

// rectTiles lists the squares of a w x h block anchored at (x, y).
func rectTiles(x, y, w, h int) []maps.Tile {
	out := make([]maps.Tile, 0, max(w, 1)*max(h, 1))
	for yy := y; yy < y+max(h, 1); yy++ {
		for xx := x; xx < x+max(w, 1); xx++ {
			out = append(out, maps.Tile{X: xx, Y: yy})
		}
	}
	return out
}

// furnitureTiles lists the squares a piece of furniture covers.
func (a *applier) furnitureTiles(f maps.Furniture) []maps.Tile {
	w, h := 1, 1
	if a.catalog != nil {
		if fw, fh, ok := a.catalog.FurnitureSize(f.Type); ok {
			if rw, rh, err := maps.RotatedSize(fw, fh, f.Rotation); err == nil {
				w, h = rw, rh
			}
		}
	}
	return rectTiles(f.X, f.Y, w, h)
}

// trapTiles lists the squares a trap covers where it is now.
func (a *applier) trapTiles(t *TrapState, qt maps.Trap) []maps.Tile {
	at := trapAt(t, qt)
	w, h := 1, 1
	if a.catalog != nil {
		if tw, th, ok := a.catalog.TrapSize(qt.Kind); ok {
			if rw, rh, err := maps.RotatedSize(tw, th, qt.Rotation); err == nil {
				w, h = rw, rh
			}
		}
	}
	return rectTiles(at.X, at.Y, w, h)
}

// terrain is the board now: open doors (found ones only), blocked squares
// (the quest's not yet removed, including those hiding a secret door, and
// those added in play), and furniture by its catalog flags. Blocked squares
// block sight as well as movement.
func (a *applier) terrain() *maps.Terrain {
	var spec maps.TerrainSpec
	for _, qd := range a.s.Quest.Doors {
		for _, d := range a.s.Doors {
			if d.ID == qd.ID && d.Found && d.State == maps.DoorOpen {
				spec.OpenDoors = append(spec.OpenDoors, qd)
			}
		}
	}
	block := func(tiles []maps.Tile) {
		spec.Blocked = append(spec.Blocked, tiles...)
		spec.SightBlocked = append(spec.SightBlocked, tiles...)
	}
	for _, r := range a.s.Quest.BlockedSquares {
		if !slices.Contains(a.s.RemovedBlocks, r.ID) {
			block(rectTiles(r.X, r.Y, r.W, r.H))
		}
	}
	for _, r := range a.s.AddedBlocks {
		block(rectTiles(r.X, r.Y, r.W, r.H))
	}
	if a.catalog != nil {
		for _, f := range a.s.Quest.Furniture {
			def, ok := a.catalog.FurnitureByID(f.Type)
			if !ok {
				continue
			}
			if def.BlocksMovement {
				spec.Blocked = append(spec.Blocked, a.furnitureTiles(f)...)
			}
			if def.BlocksLineOfSight {
				spec.SightBlocked = append(spec.SightBlocked, a.furnitureTiles(f)...)
			}
		}
	}
	return maps.NewTerrain(&a.s.Board, spec)
}

// heroAt returns the living hero on a square, other than skip.
func (a *applier) heroAt(t maps.Tile, skip string) *Hero {
	for i := range a.s.Heroes {
		h := &a.s.Heroes[i]
		if h.ID != skip && h.Placed && h.Status == HeroActive && h.X == t.X && h.Y == t.Y {
			return h
		}
	}
	return nil
}

// monsterAt returns the living monster covering a square, other than skip.
func (a *applier) monsterAt(t maps.Tile, skip string) *Monster {
	for i := range a.s.Monsters {
		m := &a.s.Monsters[i]
		if m.ID != skip && m.Alive && t.X >= m.X && t.X < m.X+max(m.Width, 1) && t.Y >= m.Y && t.Y < m.Y+max(m.Height, 1) {
			return m
		}
	}
	return nil
}

// revealFrom reveals what a hero standing on a square can see: the squares
// in view become discovered and the pieces on them seen, and are counted.
func (a *applier) revealFrom(t maps.Tile) seenCount {
	visible := maps.VisibleTiles(a.terrain(), t)
	indexes := make([]int, len(visible))
	for i, v := range visible {
		indexes[i] = a.s.Board.Index(v.X, v.Y)
	}
	a.setDiscovered(indexes, true)
	return a.showContents(indexes)
}

// revealFromHeroes reveals what every hero on the board can see.
func (a *applier) revealFromHeroes() string {
	var tiles []maps.Tile
	for _, h := range a.s.Heroes {
		if h.Placed && h.Status == HeroActive {
			tiles = append(tiles, maps.Tile{X: h.X, Y: h.Y})
		}
	}
	t := a.terrain()
	seen := map[int]bool{}
	for _, from := range tiles {
		for _, v := range maps.VisibleTiles(t, from) {
			seen[a.s.Board.Index(v.X, v.Y)] = true
		}
	}
	indexes := sortedKeys(seen)
	a.setDiscovered(indexes, true)
	return a.showContentsOn(indexes)
}
