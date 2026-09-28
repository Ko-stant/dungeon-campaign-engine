// Package dice parses and describes dice expressions such as "2d6+1" or
// "1d8+1d4", using the campaign's dice: d4, d6, d8, d10, d12 and d20. The app
// records what the GM rolls; Roll exists for an optional on-screen roller.
package dice

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Sides lists the allowed die sizes.
var Sides = []int{4, 6, 8, 10, 12, 20}

const (
	// MaxCount bounds the dice in one term ("20d6").
	MaxCount = 20
	// MaxDice bounds the dice in a whole expression.
	MaxDice = 50
	// MaxModifier bounds the flat modifier either way.
	MaxModifier = 99
)

// Term is a group of identical dice, subtracted when Negative.
type Term struct {
	Count    int
	Sides    int
	Negative bool
}

// Expr is a sum of dice terms plus a flat modifier. The zero value is "0".
type Expr struct {
	Terms    []Term
	Modifier int
}

// Roll is one die thrown by Expr.Roll.
type Roll struct {
	Sides    int
	Value    int
	Negative bool
}

var (
	diceTerm     = regexp.MustCompile(`^(\d*)\s*[dD]\s*(\d+)$`)
	constantTerm = regexp.MustCompile(`^\d{1,4}$`)
)

// Parse reads an expression like "2d6+1", "d20", "1d8-1d4" or "3". Spaces
// are ignored around the operators and the "d"; constants are summed into
// the modifier, which is always written last.
func Parse(s string) (Expr, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Expr{}, errors.New("dice expression is empty")
	}
	var e Expr
	dice := 0
	negative := false
	start := 0
	for i := 0; i <= len(s); i++ {
		if i < len(s) && s[i] != '+' && s[i] != '-' {
			continue
		}
		raw := strings.TrimSpace(s[start:i])
		// A leading sign ("-1+1d6") has nothing before it.
		leading := start == 0 && raw == "" && i < len(s)
		if !leading {
			if raw == "" {
				return Expr{}, fmt.Errorf("%q is missing a term", s)
			}
			n, err := parseTerm(s, raw, negative, &e)
			if err != nil {
				return Expr{}, err
			}
			dice += n
		}
		if i < len(s) {
			negative = s[i] == '-'
		}
		start = i + 1
	}
	if dice > MaxDice {
		return Expr{}, fmt.Errorf("%q has %d dice; use at most %d dice", s, dice, MaxDice)
	}
	if e.Modifier < -MaxModifier || e.Modifier > MaxModifier {
		return Expr{}, fmt.Errorf("%q: the modifier must be between %d and %d", s, -MaxModifier, MaxModifier)
	}
	return e, nil
}

// parseTerm adds one term to e and returns how many dice it holds.
func parseTerm(s, raw string, negative bool, e *Expr) (int, error) {
	if constantTerm.MatchString(raw) {
		n, _ := strconv.Atoi(raw)
		if negative {
			n = -n
		}
		e.Modifier += n
		return 0, nil
	}
	m := diceTerm.FindStringSubmatch(raw)
	if m == nil {
		return 0, fmt.Errorf("%q is not a dice expression (e.g. 2d6+1)", s)
	}
	count := 1
	if m[1] != "" {
		var err error
		if count, err = strconv.Atoi(m[1]); err != nil {
			return 0, fmt.Errorf("%q is not a dice expression (e.g. 2d6+1)", s)
		}
	}
	sides, err := strconv.Atoi(m[2])
	if err != nil || !slices.Contains(Sides, sides) {
		return 0, fmt.Errorf("%q: dice must be d4, d6, d8, d10, d12 or d20", s)
	}
	if count < 1 {
		return 0, fmt.Errorf("%q: roll at least 1 die per term", s)
	}
	if count > MaxCount {
		return 0, fmt.Errorf("%q: roll at most %d dice per term", s, MaxCount)
	}
	e.Terms = append(e.Terms, Term{Count: count, Sides: sides, Negative: negative})
	return count, nil
}

// MustParse is Parse for expressions known to be valid; it panics otherwise.
func MustParse(s string) Expr {
	e, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return e
}

// IsZero reports whether the expression is "0".
func (e Expr) IsZero() bool {
	return len(e.Terms) == 0 && e.Modifier == 0
}

// String writes the canonical form, e.g. "2d6+1".
func (e Expr) String() string {
	var b strings.Builder
	for i, t := range e.Terms {
		if t.Negative {
			b.WriteByte('-')
		} else if i > 0 {
			b.WriteByte('+')
		}
		fmt.Fprintf(&b, "%dd%d", t.Count, t.Sides)
	}
	switch {
	case len(e.Terms) == 0:
		b.WriteString(strconv.Itoa(e.Modifier))
	case e.Modifier > 0:
		fmt.Fprintf(&b, "+%d", e.Modifier)
	case e.Modifier < 0:
		fmt.Fprintf(&b, "%d", e.Modifier)
	}
	return b.String()
}

// Min is the lowest possible total.
func (e Expr) Min() int {
	total := e.Modifier
	for _, t := range e.Terms {
		if t.Negative {
			total -= t.Count * t.Sides
		} else {
			total += t.Count
		}
	}
	return total
}

// Max is the highest possible total.
func (e Expr) Max() int {
	total := e.Modifier
	for _, t := range e.Terms {
		if t.Negative {
			total -= t.Count
		} else {
			total += t.Count * t.Sides
		}
	}
	return total
}

// Roll throws every die with die (which returns 1..sides) and returns the
// total and each die thrown, in order.
func (e Expr) Roll(die func(sides int) int) (int, []Roll) {
	total := e.Modifier
	var rolls []Roll
	for _, t := range e.Terms {
		for range t.Count {
			v := die(t.Sides)
			if t.Negative {
				total -= v
			} else {
				total += v
			}
			rolls = append(rolls, Roll{Sides: t.Sides, Value: v, Negative: t.Negative})
		}
	}
	return total, rolls
}

// MarshalText writes the canonical form, so JSON stores "2d6+1".
func (e Expr) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

// UnmarshalText parses and validates an expression.
func (e *Expr) UnmarshalText(data []byte) error {
	parsed, err := Parse(string(data))
	if err != nil {
		return err
	}
	*e = parsed
	return nil
}
