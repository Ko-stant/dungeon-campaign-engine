package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// What triggering a trap does depends on its kind (HeroQuest's traps): a
// pit stays on the board for the rest of the quest; a falling block turns its
// square into blocked squares; a spear trap is gone; a boulder rolls, and the
// GM blocks the square where it stops (trap.block once it is moved there, or
// block.add). Other kinds are marked triggered for the GM to keep or remove.
// The GM can still set any state by hand with trap.set.
const (
	onTriggerStay   = "stay"
	onTriggerBlock  = "block"
	onTriggerRemove = "remove"
	onTriggerRoll   = "roll"
)

var trapTriggers = map[string]string{
	"pit":           onTriggerStay,
	"long_pit":      onTriggerStay,
	"falling_block": onTriggerBlock,
	"spear":         onTriggerRemove,
	"boulder":       onTriggerRoll,
}

const addedBlockPrefix = "added-"

// liveTrap finds a trap that is still on the board.
func (a *applier) liveTrap(id string) (*TrapState, maps.Trap, error) {
	t, qt, err := a.trap(id)
	if err != nil {
		return nil, maps.Trap{}, err
	}
	if t.State == maps.TrapRemoved {
		return nil, maps.Trap{}, fmt.Errorf("trap %s has been removed", id)
	}
	return t, qt, nil
}

func (a *applier) trapTrigger(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID string `json:"id"`
	}](payload)
	if err != nil {
		return "", err
	}
	t, qt, err := a.liveTrap(p.ID)
	if err != nil {
		return "", err
	}
	label := a.trapKindLabel(qt) + " " + t.ID
	switch trapTriggers[qt.Kind] {
	case onTriggerStay:
		t.State = maps.TrapTriggered
		return label + " triggered: it stays on the board", nil
	case onTriggerBlock:
		r := a.blockTrap(t, qt)
		return fmt.Sprintf("%s triggered: %s blocked now", label, squaresAt(r)), nil
	case onTriggerRemove:
		t.State = maps.TrapRemoved
		return label + " triggered and is gone", nil
	case onTriggerRoll:
		t.State = maps.TrapTriggered
		return label + " triggered: block the square where it stops", nil
	}
	t.State = maps.TrapTriggered
	return label + " triggered", nil
}

func (a *applier) trapBlock(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID string `json:"id"`
	}](payload)
	if err != nil {
		return "", err
	}
	t, qt, err := a.liveTrap(p.ID)
	if err != nil {
		return "", err
	}
	r := a.blockTrap(t, qt)
	return fmt.Sprintf("%s %s turned into blocked squares at (%d,%d)", a.trapKindLabel(qt), t.ID, r.X, r.Y), nil
}

// blockTrap removes a trap and blocks the squares it covers where it is now.
func (a *applier) blockTrap(t *TrapState, qt maps.Trap) maps.Rect {
	at := trapAt(t, qt)
	w, h := 1, 1
	if a.catalog != nil {
		if tw, th, ok := a.catalog.TrapSize(qt.Kind); ok {
			if rw, rh, err := maps.RotatedSize(tw, th, qt.Rotation); err == nil {
				w, h = rw, rh
			}
		}
	}
	t.State = maps.TrapRemoved
	return a.addBlock(at.X, at.Y, w, h)
}

func (a *applier) addBlock(x, y, w, h int) maps.Rect {
	highest := 0
	for _, r := range a.s.AddedBlocks {
		if n, err := strconv.Atoi(strings.TrimPrefix(r.ID, addedBlockPrefix)); err == nil && n > highest {
			highest = n
		}
	}
	r := maps.Rect{ID: addedBlockPrefix + strconv.Itoa(highest+1), X: x, Y: y, W: w, H: h}
	a.s.AddedBlocks = append(a.s.AddedBlocks, r)
	return r
}

// squaresAt is "(3,3) is" for one square and "(3,3)-(4,3) are" for more.
func squaresAt(r maps.Rect) string {
	if r.W*r.H == 1 {
		return fmt.Sprintf("(%d,%d) is", r.X, r.Y)
	}
	return fmt.Sprintf("(%d,%d)-(%d,%d) are", r.X, r.Y, r.X+r.W-1, r.Y+r.H-1)
}

func (a *applier) blockAdd(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		X int `json:"x"`
		Y int `json:"y"`
	}](payload)
	if err != nil {
		return "", err
	}
	if err := a.onBoard(p.X, p.Y); err != nil {
		return "", err
	}
	if slices.ContainsFunc(a.s.AddedBlocks, func(r maps.Rect) bool {
		return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
	}) {
		return "", fmt.Errorf("(%d,%d) is blocked already", p.X, p.Y)
	}
	a.addBlock(p.X, p.Y, 1, 1)
	return fmt.Sprintf("Blocked (%d,%d)", p.X, p.Y), nil
}

func (a *applier) blockRemove(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID string `json:"id"`
	}](payload)
	if err != nil {
		return "", err
	}
	i := slices.IndexFunc(a.s.AddedBlocks, func(r maps.Rect) bool { return r.ID == p.ID })
	if i < 0 {
		if slices.ContainsFunc(a.s.Quest.BlockedSquares, func(r maps.Rect) bool { return r.ID == p.ID }) {
			return "", errors.New("that is one of the quest's blocked squares; remove it with blocked.set")
		}
		return "", fmt.Errorf("no added blocked square %q", p.ID)
	}
	r := a.s.AddedBlocks[i]
	a.s.AddedBlocks = slices.Delete(slices.Clone(a.s.AddedBlocks), i, i+1)
	if r.W*r.H == 1 {
		return fmt.Sprintf("Cleared the blocked square at (%d,%d)", r.X, r.Y), nil
	}
	return fmt.Sprintf("Cleared the blocked squares at (%d,%d)-(%d,%d)", r.X, r.Y, r.X+r.W-1, r.Y+r.H-1), nil
}
