package dice

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	for in, want := range map[string]string{
		"1d8":           "1d8",
		"2d6+1":         "2d6+1",
		" 2 D6 + 1 ":    "2d6+1",
		"d20":           "1d20",
		"1d4+1":         "1d4+1",
		"1d8+1d6":       "1d8+1d6",
		"1d12-2":        "1d12-2",
		"3":             "3",
		"0":             "0",
		"2+1d10":        "1d10+2",
		"1d6+2+1":       "1d6+3",
		"1d8-1d4":       "1d8-1d4",
		"2d6+1d6":       "2d6+1d6",
		"1d6-1":         "1d6-1",
		"-1+1d6":        "1d6-1",
		"20d20+99":      "20d20+99",
		"1d4+1d6+1d8+2": "1d4+1d6+1d8+2",
	} {
		e, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q): %v", in, err)
			continue
		}
		if got := e.String(); got != want {
			t.Errorf("Parse(%q).String() = %q, want %q", in, got, want)
		}
	}
}

func TestParseRejects(t *testing.T) {
	for in, wantMsg := range map[string]string{
		"":        "empty",
		"   ":     "empty",
		"1d7":     "d4, d6, d8, d10, d12 or d20",
		"1d100":   "d4, d6, d8, d10, d12 or d20",
		"0d6":     "at least 1",
		"21d6":    "at most 20",
		"1d6+":    "missing",
		"+":       "missing",
		"1d6++1":  "missing",
		"abc":     "not a dice expression",
		"1d6*2":   "not a dice expression",
		"1dd6":    "not a dice expression",
		"1d6+100": "between -99 and 99",
		"-100":    "between -99 and 99",
		"d":       "not a dice expression",
		"1d6 1":   "not a dice expression",
	} {
		_, err := Parse(in)
		if err == nil {
			t.Errorf("Parse(%q) should fail", in)
			continue
		}
		if !strings.Contains(err.Error(), wantMsg) {
			t.Errorf("Parse(%q) error %q should mention %q", in, err, wantMsg)
		}
	}
}

func TestTooManyDice(t *testing.T) {
	in := strings.Repeat("20d6+", 3) + "1"
	if _, err := Parse(in); err == nil || !strings.Contains(err.Error(), "at most 50 dice") {
		t.Fatalf("Parse(%q) = %v, want a dice limit error", in, err)
	}
}

func TestRange(t *testing.T) {
	for in, want := range map[string][2]int{
		"1d8":     {1, 8},
		"2d6+1":   {3, 13},
		"1d8-1d4": {-3, 7},
		"1d4-2":   {-1, 2},
		"5":       {5, 5},
	} {
		e := MustParse(in)
		if lo, hi := e.Min(), e.Max(); lo != want[0] || hi != want[1] {
			t.Errorf("%s range = %d..%d, want %d..%d", in, lo, hi, want[0], want[1])
		}
	}
}

func TestRoll(t *testing.T) {
	e := MustParse("2d6+1d4-1")
	var asked []int
	// A fake die that always rolls its highest face records which dice were thrown.
	total, rolls := e.Roll(func(sides int) int { asked = append(asked, sides); return sides })
	if total != 15 {
		t.Fatalf("2d6+1d4-1 with max faces = %d, want 15", total)
	}
	if len(asked) != 3 || asked[0] != 6 || asked[1] != 6 || asked[2] != 4 {
		t.Fatalf("dice thrown: %v", asked)
	}
	if len(rolls) != 3 || rolls[2].Sides != 4 || rolls[2].Value != 4 || rolls[2].Negative {
		t.Fatalf("rolls: %+v", rolls)
	}

	neg := MustParse("1d8-1d4")
	total, rolls = neg.Roll(func(sides int) int { return 2 })
	if total != 0 || !rolls[1].Negative {
		t.Fatalf("1d8-1d4 with twos = %d, rolls %+v", total, rolls)
	}
}

func TestZeroValueIsZero(t *testing.T) {
	var e Expr
	if e.String() != "0" || e.Min() != 0 || e.Max() != 0 || !e.IsZero() {
		t.Fatalf("zero expr: %q %d %d", e.String(), e.Min(), e.Max())
	}
	if MustParse("1d6").IsZero() {
		t.Fatal("1d6 is not zero")
	}
}

func TestJSON(t *testing.T) {
	e := MustParse("2d6+1")
	data, err := e.MarshalText()
	if err != nil || string(data) != "2d6+1" {
		t.Fatalf("marshal: %s %v", data, err)
	}
	var back Expr
	if err := back.UnmarshalText([]byte("1d8 + 1d4")); err != nil || back.String() != "1d8+1d4" {
		t.Fatalf("unmarshal: %v %v", back, err)
	}
	if err := back.UnmarshalText([]byte("1d7")); err == nil {
		t.Fatal("unmarshal should validate")
	}
}
