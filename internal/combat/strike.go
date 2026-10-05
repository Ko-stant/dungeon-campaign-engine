package combat

import "github.com/Ko-stant/dungeon-campaign-engine/internal/dice"

// HeroAttacker is what a hero's attack needs: class combat stats plus
// equipment. Mirrors HeroAttacker in internal/web/src/combat/odds.ts.
type HeroAttacker struct {
	// HitDice are rolled with the d20 crit die.
	HitDice  dice.Expr
	Accuracy int
	// CritFrom is the lowest natural crit-die roll that crits.
	CritFrom       int
	CritMultiplier int
	// NearMiss is added to a missed hit total when the crit die crits.
	NearMiss int
	Damage   int
}

// StrikeOptions change one hero attack. The zero value is a plain attack.
type StrikeOptions struct {
	// Bonus is added to the hit total (Determination, a mark, or a penalty
	// when negative).
	Bonus int
	// Advantage rolls the hit dice twice and keeps the better total.
	Advantage bool
	// Sure skips the hit roll: the target cannot defend or the shot cannot
	// miss. Only the crit die is rolled.
	Sure bool
	// ExtraDamage is added to damage before any factor.
	ExtraDamage int
	// DamageFactor multiplies damage; 0 means 1.
	DamageFactor int
}

// StrikeResult is the outcome of one hero attack.
type StrikeResult struct {
	Damage int  `json:"damage"`
	Hit    bool `json:"hit"`
	Crit   bool `json:"crit"`
	// CriticalMiss: every die showed 1, and the hero loses their next turn.
	CriticalMiss bool `json:"criticalMiss"`
}

// HeroStrike resolves one hero attack against a monster's Avoidance. Dice are
// thrown in this order: the hit dice (twice with advantage), then the crit
// die. Mirrors heroStrike in simulate.ts.
func HeroStrike(a HeroAttacker, avoidance int, r Roller, o StrikeOptions) StrikeResult {
	factor := o.DamageFactor
	if factor == 0 {
		factor = 1
	}
	damage := (a.Damage + o.ExtraDamage) * factor
	if o.Sure {
		crit := r.Die(20) >= a.CritFrom
		if crit {
			damage *= a.CritMultiplier
		}
		return StrikeResult{Damage: damage, Hit: true, Crit: crit}
	}
	total, rolls := a.HitDice.Roll(r.Die)
	if o.Advantage {
		if second, secondRolls := a.HitDice.Roll(r.Die); second > total {
			total, rolls = second, secondRolls
		}
	}
	critDie := r.Die(20)
	allOnes, allMax := true, true
	for _, d := range rolls {
		allOnes = allOnes && d.Value == 1
		allMax = allMax && d.Value == d.Sides
	}
	if critDie == 1 && allOnes {
		return StrikeResult{CriticalMiss: true}
	}
	crit := critDie >= a.CritFrom
	special := critDie == 20 && allMax
	total += a.Accuracy + o.Bonus
	if special || (crit && total >= avoidance) {
		return StrikeResult{Damage: damage * a.CritMultiplier, Hit: true, Crit: true}
	}
	if total >= avoidance || (crit && total+a.NearMiss >= avoidance) {
		return StrikeResult{Damage: damage, Hit: true}
	}
	return StrikeResult{}
}

// MonsterAttacker is what a monster's attack needs.
type MonsterAttacker struct {
	HitDice dice.Expr
	Damage  int
}

// HeroDefender is what a hero defends with.
type HeroDefender struct {
	DefenseDice   dice.Expr
	BaseAvoidance int
	Mitigation    int
}

// MonsterStrikeResult is the outcome of one monster attack.
type MonsterStrikeResult struct {
	Damage int  `json:"damage"`
	Hit    bool `json:"hit"`
	Crit   bool `json:"crit"`
}

// MonsterStrike resolves one monster attack against a hero's opposed roll;
// ties go to the hero. Dice are thrown in this order: the monster's hit dice,
// the hero's defense dice (unless the hero cannot defend), then two d20 for
// the crit check, on a hit only. Mirrors monsterStrike in simulate.ts.
func MonsterStrike(m MonsterAttacker, d HeroDefender, r Roller, canDefend bool) MonsterStrikeResult {
	attack, _ := m.HitDice.Roll(r.Die)
	defense := d.BaseAvoidance
	if canDefend {
		roll, _ := d.DefenseDice.Roll(r.Die)
		defense += roll
	}
	if attack <= defense {
		return MonsterStrikeResult{}
	}
	first, second := r.Die(20), r.Die(20)
	crit := first == 20 && second == 20
	damage := m.Damage
	if crit {
		damage *= 2
	}
	return MonsterStrikeResult{Damage: max(0, damage-d.Mitigation), Hit: true, Crit: crit}
}
