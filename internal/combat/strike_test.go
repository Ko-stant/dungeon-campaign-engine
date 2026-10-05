package combat

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/dice"
)

// These cases mirror internal/web/src/combat/simulate.test.ts.

var barbarian = HeroAttacker{HitDice: dice.MustParse("1d20"), Accuracy: 3, CritFrom: 17, CritMultiplier: 2, NearMiss: 2, Damage: 10}

func scripted(t *testing.T, values ...int) *Script {
	t.Helper()
	s := &Script{Dice: values}
	t.Cleanup(func() {
		if err := s.Err(); err != nil {
			t.Errorf("scripted dice: %v", err)
		}
	})
	return s
}

func TestHeroStrikeMeetsAvoidanceAndCritsDouble(t *testing.T) {
	if got, want := HeroStrike(barbarian, 12, scripted(t, 9, 5), StrikeOptions{}), (StrikeResult{Damage: 10, Hit: true}); got != want {
		t.Errorf("plain hit = %+v, want %+v", got, want)
	}
	if got, want := HeroStrike(barbarian, 12, scripted(t, 15, 18), StrikeOptions{}), (StrikeResult{Damage: 20, Hit: true, Crit: true}); got != want {
		t.Errorf("crit = %+v, want %+v", got, want)
	}
}

func TestHeroStrikeNearMiss(t *testing.T) {
	if got, want := HeroStrike(barbarian, 12, scripted(t, 7, 17), StrikeOptions{}), (StrikeResult{Damage: 10, Hit: true}); got != want {
		t.Errorf("near miss = %+v, want %+v", got, want)
	}
	if HeroStrike(barbarian, 12, scripted(t, 5, 19), StrikeOptions{}).Hit {
		t.Error("a near miss that still falls short should miss")
	}
}

func TestHeroStrikeExtremes(t *testing.T) {
	if got, want := HeroStrike(barbarian, 2, scripted(t, 1, 1), StrikeOptions{}), (StrikeResult{CriticalMiss: true}); got != want {
		t.Errorf("all ones = %+v, want %+v", got, want)
	}
	if got, want := HeroStrike(barbarian, 30, scripted(t, 20, 20), StrikeOptions{}), (StrikeResult{Damage: 20, Hit: true, Crit: true}); got != want {
		t.Errorf("all maximums = %+v, want %+v", got, want)
	}
}

func TestHeroStrikeOptions(t *testing.T) {
	if !HeroStrike(barbarian, 12, scripted(t, 7, 3), StrikeOptions{Bonus: 2}).Hit {
		t.Error("a +2 bonus should turn 7+3 into a hit against 12")
	}
	adv := scripted(t, 4, 15, 3)
	if !HeroStrike(barbarian, 18, adv, StrikeOptions{Advantage: true}).Hit {
		t.Error("advantage keeps the better hit roll")
	}
	if adv.Left() != 0 {
		t.Errorf("advantage should roll the hit dice twice, %d dice left", adv.Left())
	}
	if got, want := HeroStrike(barbarian, 99, scripted(t, 3), StrikeOptions{Sure: true}), (StrikeResult{Damage: 10, Hit: true}); got != want {
		t.Errorf("sure hit = %+v, want %+v", got, want)
	}
	if got := HeroStrike(barbarian, 99, scripted(t, 18), StrikeOptions{Sure: true}).Damage; got != 20 {
		t.Errorf("sure crit damage = %d, want 20", got)
	}
	if got := HeroStrike(barbarian, 12, scripted(t, 15, 3), StrikeOptions{ExtraDamage: 4}).Damage; got != 14 {
		t.Errorf("extra damage = %d, want 14", got)
	}
	if got := HeroStrike(barbarian, 12, scripted(t, 15, 3), StrikeOptions{DamageFactor: 3}).Damage; got != 30 {
		t.Errorf("damage factor = %d, want 30", got)
	}
}

var (
	orc  = MonsterAttacker{HitDice: dice.MustParse("2d8"), Damage: 5}
	hero = HeroDefender{DefenseDice: dice.MustParse("1d6"), BaseAvoidance: 4, Mitigation: 1}
)

func TestMonsterStrikeBeatsTheHeroTotal(t *testing.T) {
	if got, want := MonsterStrike(orc, hero, scripted(t, 6, 5, 3, 1, 1), true), (MonsterStrikeResult{Damage: 4, Hit: true}); got != want {
		t.Errorf("hit = %+v, want %+v", got, want)
	}
	tie := scripted(t, 4, 4, 4)
	if got := MonsterStrike(orc, hero, tie, true); got != (MonsterStrikeResult{}) {
		t.Errorf("a tie goes to the hero, got %+v", got)
	}
	if tie.Left() != 0 {
		t.Error("a miss rolls no crit dice")
	}
}

func TestMonsterStrikeDoubleTwenties(t *testing.T) {
	if got, want := MonsterStrike(orc, hero, scripted(t, 8, 8, 2, 20, 20), true), (MonsterStrikeResult{Damage: 9, Hit: true, Crit: true}); got != want {
		t.Errorf("crit = %+v, want %+v", got, want)
	}
}

func TestMonsterStrikeWhenTheHeroCannotDefend(t *testing.T) {
	if got, want := MonsterStrike(orc, hero, scripted(t, 3, 2, 1, 1), false), (MonsterStrikeResult{Damage: 4, Hit: true}); got != want {
		t.Errorf("cannot defend = %+v, want %+v", got, want)
	}
	tough := hero
	tough.Mitigation = 5
	if got := MonsterStrike(orc, tough, scripted(t, 8, 8, 1, 1, 2), true).Damage; got != 0 {
		t.Errorf("mitigation should bring the hit to 0, got %d", got)
	}
}
