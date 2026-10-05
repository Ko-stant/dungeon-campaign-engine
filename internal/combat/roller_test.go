package combat

import (
	"reflect"
	"testing"
)

func TestMulberry32MatchesTheSimulator(t *testing.T) {
	// Draws from seededRoller in internal/web/src/combat/simulate.ts, for the
	// sides 20, 20, 20, 6, 6, 6, 4, 8, 10, 12.
	sides := []int{20, 20, 20, 6, 6, 6, 4, 8, 10, 12}
	want := map[uint32][]int{
		0:          {6, 1, 5, 1, 3, 4, 3, 6, 5, 7},
		1:          {13, 1, 11, 6, 6, 2, 3, 6, 5, 12},
		42:         {13, 9, 18, 5, 2, 4, 2, 5, 9, 6},
		4294967295: {18, 4, 15, 6, 6, 4, 3, 4, 2, 12},
	}
	for seed, w := range want {
		r := &Mulberry32{State: seed}
		got := make([]int, len(sides))
		for i, s := range sides {
			got[i] = r.Die(s)
		}
		if !reflect.DeepEqual(got, w) {
			t.Errorf("seed %d: %v, want %v", seed, got, w)
		}
	}
}

func TestMulberry32StateCarriesOn(t *testing.T) {
	a := &Mulberry32{State: 7}
	a.Die(6)
	b := &Mulberry32{State: a.State}
	for range 100 {
		if a.Die(20) != b.Die(20) {
			t.Fatal("a roller rebuilt from the saved state should continue the same sequence")
		}
	}
}

func TestMulberry32StaysInRange(t *testing.T) {
	r := &Mulberry32{State: 3}
	sum := 0
	for range 10000 {
		v := r.Die(6)
		if v < 1 || v > 6 {
			t.Fatalf("d6 rolled %d", v)
		}
		sum += v
	}
	if mean := float64(sum) / 10000; mean < 3.4 || mean > 3.6 {
		t.Errorf("d6 mean %.3f, want about 3.5", mean)
	}
}

func TestScriptHandsOutDiceInOrder(t *testing.T) {
	s := &Script{Dice: []int{3, 20}}
	if s.Die(6) != 3 || s.Die(20) != 20 {
		t.Fatal("dice should come out in order")
	}
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Left() != 0 {
		t.Errorf("Left = %d, want 0", s.Left())
	}
}

func TestScriptReportsMissingAndBadDice(t *testing.T) {
	s := &Script{Dice: []int{7}}
	s.Die(6)
	if s.Err() == nil {
		t.Error("a 7 on a d6 should be an error")
	}
	s = &Script{}
	if v := s.Die(20); v < 1 || v > 20 {
		t.Errorf("a missing die still returns a value in range, got %d", v)
	}
	if s.Err() == nil {
		t.Error("running out of dice should be an error")
	}
}

func TestLogRecordsEveryDie(t *testing.T) {
	l := &Log{Roller: &Script{Dice: []int{4, 17}}}
	l.Die(6)
	l.Die(20)
	want := []Die{{Sides: 6, Value: 4}, {Sides: 20, Value: 17}}
	if !reflect.DeepEqual(l.Rolls, want) {
		t.Errorf("Rolls = %v, want %v", l.Rolls, want)
	}
}
