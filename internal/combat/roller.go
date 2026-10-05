// Package combat resolves Three Plagues attacks with real dice (see
// docs/campaigns/three-plagues/RULES_AND_CLASSES.md, "Combat"). It is a port
// of the strike math in internal/web/src/combat/simulate.ts and must draw
// dice in the same order, so seeded games match the simulator; the parity
// fixtures in testdata/parity check that.
package combat

import "fmt"

// Roller throws one die.
type Roller interface {
	// Die returns a whole number from 1 to sides.
	Die(sides int) int
}

// Mulberry32 is a small seeded roller, an exact port of seededRoller in
// simulate.ts. Its State is all there is to it: save it and a new roller
// built from it carries on the same sequence.
type Mulberry32 struct {
	State uint32
}

// Die implements Roller.
func (m *Mulberry32) Die(sides int) int {
	m.State += 0x6d2b79f5
	t := m.State
	t = (t ^ t>>15) * (t | 1)
	t ^= t + (t^t>>7)*(t|61)
	t ^= t >> 14
	// floor(t / 2^32 * sides) without floating point.
	return 1 + int(uint64(t)*uint64(sides)>>32)
}

// Script hands out fixed dice in order: for tests, and for dice rolled at the
// table. A missing or impossible die is remembered as an error (see Err) and
// a 1 is returned in its place.
type Script struct {
	Dice []int
	next int
	err  error
}

// Die implements Roller.
func (s *Script) Die(sides int) int {
	if s.next >= len(s.Dice) {
		s.fail(fmt.Errorf("die %d: no value given for a d%d", s.next+1, sides))
		s.next++
		return 1
	}
	v := s.Dice[s.next]
	s.next++
	if v < 1 || v > sides {
		s.fail(fmt.Errorf("die %d: %d is not a d%d result", s.next, v, sides))
		return 1
	}
	return v
}

func (s *Script) fail(err error) {
	if s.err == nil {
		s.err = err
	}
}

// Err returns the first missing or impossible die, if any.
func (s *Script) Err() error { return s.err }

// Left returns how many given dice have not been used.
func (s *Script) Left() int { return max(len(s.Dice)-s.next, 0) }

// Die is one die thrown.
type Die struct {
	Sides int `json:"sides"`
	Value int `json:"value"`
}

// Log passes dice through from another roller and records each one, so an
// event can show every die behind a result.
type Log struct {
	Roller Roller
	Rolls  []Die
}

// Die implements Roller.
func (l *Log) Die(sides int) int {
	v := l.Roller.Die(sides)
	l.Rolls = append(l.Rolls, Die{Sides: sides, Value: v})
	return v
}
