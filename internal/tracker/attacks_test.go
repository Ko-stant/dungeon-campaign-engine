package tracker

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// duelState: hallBoard without furniture, Grom (diagonal reach) at (4,2)
// and Ilsa (line of sight) at (4,1) in room 2, an orc (12 Body, Avoidance
// 10) seen at (5,3), the rules on and Grom's turn under way.
//
//	y3: 1 1 . 2 O
//	y2: 1 1 . G 2
//	y1: 1 1 . I 2
func duelState(t *testing.T) *State {
	t.Helper()
	b, q, cat := hallBoard()
	q.Furniture = nil
	q.StartTiles = []maps.Tile{{X: 4, Y: 2}, {X: 4, Y: 1}}
	s, err := NewSession(b, q, "Hall", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	s.Heroes[0].Combat = &Combat{HitDice: "1d20", Accuracy: 3, CritFrom: 17, Damage: 3, DefenseDice: "1d6", Avoidance: 2, Mitigation: 1, Reach: content.ReachDiagonal}
	s.Heroes[0].Body, s.Heroes[0].MaxBody = 40, 40
	s.Heroes[1].Combat = &Combat{HitDice: "2d8", Accuracy: 2, CritFrom: 18, Damage: 4, DefenseDice: "1d6", Avoidance: 3, Reach: content.ReachSight}
	o := &s.Monsters[0]
	o.Body, o.MaxBody = 12, 12
	o.Combat = &content.MonsterCombat{Avoidance: 10, HitDice: "2d8", Damage: 5}
	s, _ = hallApply(t, s, cmd(t, "rules.enable", map[string]any{"seed": 3}))
	if s.Monsters[0].Visibility != MonsterSeen || !s.Fight {
		t.Fatalf("the orc should be in view: %+v", s.Monsters[0])
	}
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"})))
	return s
}

// attack is Grom's (or the given seat's) attack on the orc with the GM's dice.
func attack(t *testing.T, hero string, dice ...int) Command {
	c := as(seat(hero), cmd(t, "turn.attack", map[string]any{"target": "orc"}))
	if len(dice) > 0 {
		c.Actor = Actor{}
		c.Dice = dice
	}
	return c
}

func TestAHitDealsDamage(t *testing.T) {
	s, ev := hallApply(t, duelState(t), attack(t, "hero-1", 15, 5))
	if s.Monsters[0].Body != 9 || !s.Rules.Turn.Acted {
		t.Fatalf("orc %d Body, turn %+v", s.Monsters[0].Body, s.Rules.Turn)
	}
	if want := "Grom (Barbarian) attacks Orc (orc): 18 vs 10 (1d20: 15, Accuracy +3; crit die 5): hit for 3 (Orc 9/12)"; ev.Summary != want {
		t.Errorf("summary\n %q\nwant\n %q", ev.Summary, want)
	}
	if msg := hallErr(t, s, attack(t, "hero-1", 15, 5)); !strings.Contains(msg, "already") {
		t.Errorf("a second attack: %q", msg)
	}
}

func TestACritDoublesAndANearMissStillHits(t *testing.T) {
	s, ev := hallApply(t, duelState(t), attack(t, "hero-1", 15, 18))
	if s.Monsters[0].Body != 6 || !strings.Contains(ev.Summary, ": crit for 6 (Orc 6/12)") {
		t.Errorf("crit: %d Body, %q", s.Monsters[0].Body, ev.Summary)
	}
	s, ev = hallApply(t, duelState(t), attack(t, "hero-1", 6, 17))
	if s.Monsters[0].Body != 9 || !strings.Contains(ev.Summary, ": near miss, a hit for 3") {
		t.Errorf("near miss: %d Body, %q", s.Monsters[0].Body, ev.Summary)
	}
}

func TestAMissBuildsDetermination(t *testing.T) {
	s, ev := hallApply(t, duelState(t), attack(t, "hero-1", 2, 3))
	if s.Heroes[0].Determination != 2 || !strings.HasSuffix(ev.Summary, ": miss (Determination +2)") {
		t.Fatalf("determination %d, %q", s.Heroes[0].Determination, ev.Summary)
	}
	// Determination counts on the next attack and a hit resets it.
	s = duelState(t)
	s.Heroes[0].Determination = 4
	s, ev = hallApply(t, s, attack(t, "hero-1", 3, 3))
	if s.Heroes[0].Determination != 0 || !strings.Contains(ev.Summary, "10 vs 10 (1d20: 3, Accuracy +3, Determination +4; crit die 3): hit") {
		t.Errorf("determination %d, %q", s.Heroes[0].Determination, ev.Summary)
	}
}

func TestACriticalMissCostsTheNextTurnAndCounts(t *testing.T) {
	s, ev := hallApply(t, duelState(t), attack(t, "hero-1", 1, 1))
	if len(s.Rules.SkipNext) != 1 || s.Heroes[0].Determination != 2 {
		t.Fatalf("skip %v, determination %d", s.Rules.SkipNext, s.Heroes[0].Determination)
	}
	if !strings.HasSuffix(ev.Summary, ": critical miss, Grom loses their next turn (Determination +2)") {
		t.Errorf("summary %q", ev.Summary)
	}
}

func TestAFalteringMonsterIsEasierToHit(t *testing.T) {
	s := duelState(t)
	s.Monsters[0].Body = 3
	_, ev := hallApply(t, s, attack(t, "hero-1", 4, 2))
	if !strings.Contains(ev.Summary, ": 7 vs 6, faltering (1d20: 4, Accuracy +3; crit die 2): hit for 3") {
		t.Errorf("summary %q", ev.Summary)
	}
}

func TestKillingTheLastMonsterEndsTheFight(t *testing.T) {
	s := duelState(t)
	s.Monsters[0].Body = 2
	s, ev := hallApply(t, s, attack(t, "hero-1", 15, 5))
	if s.Monsters[0].Alive || s.Monsters[0].Body != 0 || s.Fight {
		t.Fatalf("orc %+v, fight %v", s.Monsters[0], s.Fight)
	}
	if !strings.Contains(ev.Summary, "hit for 3; Orc (orc) dies; fight over") {
		t.Errorf("summary %q", ev.Summary)
	}
}

func TestReach(t *testing.T) {
	// Ilsa shoots along a line of sight; Grom stands in it.
	s := duelState(t)
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	s, _ = hallApply(t, s, as(seat("hero-2"), cmd(t, "turn.start", map[string]any{"hero": "hero-2"})))
	if msg := hallErr(t, s, attack(t, "hero-2")); !strings.Contains(msg, "line of sight") {
		t.Errorf("Grom blocks the shot: %q", msg)
	}
	s.Heroes[0].X, s.Heroes[0].Y = 3, 2
	if _, ev := hallApply(t, s, attack(t, "hero-2")); !strings.HasPrefix(ev.Summary, "Ilsa (Wizard) attacks Orc (orc)") {
		t.Errorf("a clear shot: %q", ev.Summary)
	}

	// An adjacent-only hero can't reach a diagonal.
	s = duelState(t)
	s.Heroes[0].Combat.Reach = content.ReachAdjacent
	if msg := hallErr(t, s, attack(t, "hero-1")); !strings.Contains(msg, "out of reach") {
		t.Errorf("adjacent only: %q", msg)
	}
	// Nor through a closed door.
	s = duelState(t)
	s.Heroes[0].X, s.Heroes[0].Y = 3, 2
	s.Monsters[0].X, s.Monsters[0].Y = 4, 2
	if msg := hallErr(t, s, attack(t, "hero-1")); !strings.Contains(msg, "out of reach") {
		t.Errorf("through a closed door: %q", msg)
	}
}

func TestOnlyMonstersInViewCanBeAttacked(t *testing.T) {
	s := duelState(t)
	s.Monsters[0].Visibility = MonsterHidden
	if msg := hallErr(t, s, attack(t, "hero-1")); !strings.Contains(msg, "no monster in view") {
		t.Errorf("a hidden monster: %q", msg)
	}
	s = duelState(t)
	s.Heroes[0].Combat = nil
	if msg := hallErr(t, s, attack(t, "hero-1")); !strings.Contains(msg, "combat stats") {
		t.Errorf("no combat stats: %q", msg)
	}
}

func TestActingAfterMovingEndsTheMovement(t *testing.T) {
	s := duelState(t)
	roll := cmd(t, "turn.roll-move", map[string]any{})
	roll.Dice = []int{2, 2}
	s, _ = hallApply(t, s, roll)
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.move", map[string]any{"to": map[string]int{"x": 4, "y": 3}})))
	s, _ = hallApply(t, s, attack(t, "hero-1", 15, 5))
	if !s.Rules.Turn.MoveDone {
		t.Fatal("moving, then acting, ends the movement (D4)")
	}
	// Acting first leaves the movement for afterward.
	s = duelState(t)
	s, _ = hallApply(t, s, attack(t, "hero-1", 15, 5))
	if s.Rules.Turn.MoveDone {
		t.Fatal("the hero may still move after acting")
	}
}

// monstersPhase is duelState in the monsters' phase, with the orc moved
// to (5,2), beside Grom.
func monstersPhase(t *testing.T) *State {
	t.Helper()
	s := duelState(t)
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	s, _ = hallApply(t, s, cmd(t, "phase.monsters", map[string]any{}))
	s.Monsters[0].Y = 2
	return s
}

func monsterAttack(t *testing.T, target string, dice ...int) Command {
	c := cmd(t, "monster.attack", map[string]any{"monster": "orc", "target": target})
	c.Dice = dice
	return c
}

func TestAMonsterAttacksAHeroBeside(t *testing.T) {
	s := monstersPhase(t)
	// 2d8: 6+5 = 11 against 1d6 (3) + 2 avoidance = 5: a hit; no crit (1, 1); 5 - 1 mitigation.
	s, ev := hallApply(t, s, monsterAttack(t, "hero-1", 6, 5, 3, 1, 1))
	if s.Heroes[0].Body != 36 {
		t.Fatalf("Grom has %d Body", s.Heroes[0].Body)
	}
	if want := "Orc (orc) attacks Grom (Barbarian): 11 vs 5 (2d8: 6, 5; 1d6: 3, Avoidance +2): hit for 4 (Grom 36/40)"; ev.Summary != want {
		t.Errorf("summary\n %q\nwant\n %q", ev.Summary, want)
	}
	if msg := hallErr(t, s, monsterAttack(t, "hero-1", 6, 5, 3, 1, 1)); !strings.Contains(msg, "already") {
		t.Errorf("a second attack: %q", msg)
	}
}

func TestAMonsterCanKillAHero(t *testing.T) {
	s := monstersPhase(t)
	s.Heroes[0].Body = 2
	s, ev := hallApply(t, s, monsterAttack(t, "hero-1", 8, 8, 1, 2, 2))
	if s.Heroes[0].Body != 0 || s.Heroes[0].Status != HeroDead || !strings.HasSuffix(ev.Summary, "; Grom (Barbarian) falls") {
		t.Fatalf("Grom %+v, %q", s.Heroes[0], ev.Summary)
	}
}

func TestMonsterReachAndPhase(t *testing.T) {
	s := monstersPhase(t)
	if msg := hallErr(t, s, monsterAttack(t, "hero-2")); !strings.Contains(msg, "out of reach") {
		t.Errorf("Ilsa is diagonal to a melee monster: %q", msg)
	}
	s.Monsters[0].Combat.Reach = true
	if msg := hallErr(t, s, monsterAttack(t, "hero-2")); !strings.Contains(msg, "out of reach") {
		t.Errorf("reach never strikes diagonally: %q", msg)
	}
	// Reach strikes two squares in a straight line, past whoever stands
	// between: the orc at (5,3) over Grom at (5,2) to Ilsa at (5,1).
	s.Monsters[0].Y = 3
	s.Heroes[0].X, s.Heroes[0].Y = 5, 2
	s.Heroes[1].X, s.Heroes[1].Y = 5, 1
	if _, ev := hallApply(t, s, monsterAttack(t, "hero-2", 1, 1, 6)); !strings.HasPrefix(ev.Summary, "Orc (orc) attacks Ilsa") {
		t.Errorf("a reaching monster strikes two squares away: %q", ev.Summary)
	}
	s.Monsters[0].Combat.Reach = false
	if msg := hallErr(t, s, monsterAttack(t, "hero-2")); !strings.Contains(msg, "out of reach") {
		t.Errorf("without reach, two squares is too far: %q", msg)
	}
	// Three squares is too far even with reach, and a wall stops it.
	s.Monsters[0].Combat.Reach = true
	s.Monsters[0].X, s.Monsters[0].Y = 3, 2
	s.Heroes[1].X, s.Heroes[1].Y = 1, 2
	if msg := hallErr(t, s, monsterAttack(t, "hero-2")); !strings.Contains(msg, "out of reach") {
		t.Errorf("through a wall: %q", msg)
	}
	if msg := hallErr(t, duelState(t), monsterAttack(t, "hero-1", 6, 5, 3, 1, 1)); !strings.Contains(msg, "monsters' phase") {
		t.Errorf("in the heroes' phase: %q", msg)
	}
	if msg := hallErr(t, s, as(seat("hero-1"), monsterAttack(t, "hero-1"))); !strings.Contains(msg, "GM") {
		t.Errorf("from a seat: %q", msg)
	}
}

func TestAMonsterMovesThenMayAttack(t *testing.T) {
	s := monstersPhase(t)
	s.Monsters[0].Movement = 3
	s.Monsters[0].Y = 3
	s.Heroes[0].X, s.Heroes[0].Y = 4, 3
	s, ev := hallApply(t, s, cmd(t, "monster.move", map[string]any{"monster": "orc", "to": map[string]int{"x": 5, "y": 1}}))
	if m := s.Monsters[0]; m.X != 5 || m.Y != 1 || ev.Summary != "Orc (orc) moves 2 squares to (5,1)" {
		t.Fatalf("orc at (%d,%d), %q", m.X, m.Y, ev.Summary)
	}
	if msg := hallErr(t, s, cmd(t, "monster.move", map[string]any{"monster": "orc", "to": map[string]int{"x": 5, "y": 2}})); !strings.Contains(msg, "already") {
		t.Errorf("a second move: %q", msg)
	}
	s, _ = hallApply(t, s, monsterAttack(t, "hero-2", 6, 5, 3, 1, 1))
	if s.Heroes[1].Body == s.Heroes[1].MaxBody {
		t.Error("the orc should reach Ilsa from (5,1)")
	}
	if msg := hallErr(t, s, cmd(t, "monster.move", map[string]any{"monster": "orc", "to": map[string]int{"x": 9, "y": 9}})); !strings.Contains(msg, "already") {
		t.Errorf("moving after moving then attacking: %q", msg)
	}
}

func TestAMonsterMayAttackThenStepAway(t *testing.T) {
	s := monstersPhase(t)
	s.Monsters[0].Movement = 2
	s, _ = hallApply(t, s, monsterAttack(t, "hero-1", 6, 5, 3, 1, 1))
	s, _ = hallApply(t, s, cmd(t, "monster.move", map[string]any{"monster": "orc", "to": map[string]int{"x": 5, "y": 1}}))
	if s.Monsters[0].Y != 1 {
		t.Fatal("attack, then step aside (D11)")
	}
}

func TestMonstersDoNotPassHeroes(t *testing.T) {
	s := monstersPhase(t)
	s.Monsters[0].Movement = 6
	s.Monsters[0].Y = 3
	if msg := hallErr(t, s, cmd(t, "monster.move", map[string]any{"monster": "orc", "to": map[string]int{"x": 4, "y": 1}})); !strings.Contains(msg, "Ilsa") {
		t.Errorf("onto a hero: %q", msg)
	}
	s.Heroes[1].X, s.Heroes[1].Y = 5, 2
	if msg := hallErr(t, s, cmd(t, "monster.move", map[string]any{"monster": "orc", "to": map[string]int{"x": 5, "y": 1}})); !strings.Contains(msg, "no way") {
		t.Errorf("past two heroes: %q", msg)
	}
}

func TestANewRoundLetsMonstersActAgain(t *testing.T) {
	s := monstersPhase(t)
	s, _ = hallApply(t, s, monsterAttack(t, "hero-1", 1, 1, 6))
	s, _ = hallApply(t, s, cmd(t, "phase.end", map[string]any{}))
	if len(s.Rules.MonstersActed) != 0 || len(s.Rules.MonstersMoved) != 0 {
		t.Fatalf("rules after the round: %+v", s.Rules)
	}
}
