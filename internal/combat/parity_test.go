package combat

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/dice"
)

// The fixtures are computed by the TypeScript simulator (bun run parity:gen,
// scripts/parity/fixtures.ts); Go must agree with every case.

func readFixture(t *testing.T, name string, v any) {
	t.Helper()
	data, err := os.ReadFile("testdata/parity/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

func TestParityMulberry32(t *testing.T) {
	var f struct {
		Sides []int `json:"sides"`
		Cases []struct {
			Seed  uint32 `json:"seed"`
			Draws []int  `json:"draws"`
		} `json:"cases"`
	}
	readFixture(t, "mulberry.json", &f)
	for _, c := range f.Cases {
		r := &Mulberry32{State: c.Seed}
		for i, want := range c.Draws {
			if got := r.Die(f.Sides[i%len(f.Sides)]); got != want {
				t.Fatalf("seed %d draw %d: %d, want %d", c.Seed, i, got, want)
			}
		}
	}
}

// replay checks a case twice: with its recorded dice (the rules must use all
// of them, in order) and, when it has one, from its seed.
func replay[R comparable](t *testing.T, i int, seed *uint32, rolled []int, want R, strike func(Roller) R) {
	t.Helper()
	s := &Script{Dice: rolled}
	if got := strike(s); got != want {
		t.Errorf("case %d with dice %v: %+v, want %+v", i, rolled, got, want)
	}
	if err := s.Err(); err != nil {
		t.Errorf("case %d: %v", i, err)
	}
	if s.Left() != 0 {
		t.Errorf("case %d: %d dice unused of %v", i, s.Left(), rolled)
	}
	if seed != nil {
		if got := strike(&Mulberry32{State: *seed}); got != want {
			t.Errorf("case %d from seed %d: %+v, want %+v", i, *seed, got, want)
		}
	}
}

func TestParityHeroStrikes(t *testing.T) {
	var f struct {
		Cases []struct {
			Attacker struct {
				HitDice        dice.Expr `json:"hitDice"`
				Accuracy       int       `json:"accuracy"`
				CritFrom       int       `json:"critFrom"`
				CritMultiplier int       `json:"critMultiplier"`
				NearMiss       int       `json:"nearMiss"`
				Damage         int       `json:"damage"`
			} `json:"attacker"`
			Avoidance int `json:"avoidance"`
			Options   struct {
				Bonus        int  `json:"bonus"`
				Advantage    bool `json:"advantage"`
				Sure         bool `json:"sure"`
				ExtraDamage  int  `json:"extraDamage"`
				DamageFactor int  `json:"damageFactor"`
			} `json:"options"`
			Seed   *uint32      `json:"seed"`
			Dice   []int        `json:"dice"`
			Result StrikeResult `json:"result"`
		} `json:"cases"`
	}
	readFixture(t, "hero_strikes.json", &f)
	if len(f.Cases) < 1000 {
		t.Fatalf("only %d cases", len(f.Cases))
	}
	for i, c := range f.Cases {
		a := HeroAttacker(c.Attacker)
		o := StrikeOptions(c.Options)
		replay(t, i, c.Seed, c.Dice, c.Result, func(r Roller) StrikeResult {
			return HeroStrike(a, c.Avoidance, r, o)
		})
	}
}

func TestParityMonsterStrikes(t *testing.T) {
	var f struct {
		Cases []struct {
			Monster struct {
				HitDice dice.Expr `json:"hitDice"`
				Damage  int       `json:"damage"`
			} `json:"monster"`
			Defender struct {
				DefenseDice   dice.Expr `json:"defenseDice"`
				BaseAvoidance int       `json:"baseAvoidance"`
				Mitigation    int       `json:"mitigation"`
			} `json:"defender"`
			CanDefend bool                `json:"canDefend"`
			Seed      *uint32             `json:"seed"`
			Dice      []int               `json:"dice"`
			Result    MonsterStrikeResult `json:"result"`
		} `json:"cases"`
	}
	readFixture(t, "monster_strikes.json", &f)
	if len(f.Cases) < 400 {
		t.Fatalf("only %d cases", len(f.Cases))
	}
	for i, c := range f.Cases {
		m := MonsterAttacker(c.Monster)
		d := HeroDefender(c.Defender)
		replay(t, i, c.Seed, c.Dice, c.Result, func(r Roller) MonsterStrikeResult {
			return MonsterStrike(m, d, r, c.CanDefend)
		})
	}
}
