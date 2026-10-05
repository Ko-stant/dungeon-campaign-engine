package tracker

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

// fightState is a session with a Cleric who regenerates mana (16 max, 3 a fight
// round) and a Ranger with a long (5) and a short (3) cooldown.
func fightState(t *testing.T) (*State, *content.Catalog) {
	t.Helper()
	b, q, cat := fixture()
	cat.Heroes = append(cat.Heroes,
		content.HeroDef{ID: "custom-cleric", Name: "Cleric", Body: 28, Mind: 4, Mana: 16, ManaRegen: 3, Custom: true, AttackDice: "2d8", DefenseDice: "1d6", Abilities: []content.Ability{
			{ID: "ability-1", Name: "Heal Minor Wounds", Kind: content.AbilitySpell, ManaCost: 6},
		}},
		content.HeroDef{ID: "custom-ranger", Name: "Ranger", Body: 30, Mind: 3, Custom: true, AttackDice: "2d10", DefenseDice: "1d6", Abilities: []content.Ability{
			{ID: "ability-1", Name: "Multi-Shot", Kind: content.AbilityActive, Cooldown: 5},
			{ID: "ability-2", Name: "Fan of Cards", Kind: content.AbilityActive, Cooldown: 3},
		}},
	)
	s, err := NewSession(b, q, "The Test", []CampaignHero{
		{ID: "hero-1", Name: "Mira", Class: "custom-cleric"},
		{ID: "hero-2", Name: "Vex", Class: "custom-ranger"},
	}, cat)
	if err != nil {
		t.Fatal(err)
	}
	return s, cat
}

func TestFightStartAndEnd(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "fight.start", map[string]any{}))
	if !s.Fight || ev.Summary != "Fight started in round 1" {
		t.Fatalf("start: %v %q", s.Fight, ev.Summary)
	}
	if _, _, err := Apply(s, cmd(t, "fight.start", map[string]any{}), nil); err == nil {
		t.Fatal("starting a fight during a fight should fail")
	}
	s, ev = apply(t, s, cmd(t, "fight.end", map[string]any{}))
	if s.Fight || ev.Summary != "Fight over" {
		t.Fatalf("end: %v %q", s.Fight, ev.Summary)
	}
	if _, _, err := Apply(s, cmd(t, "fight.end", map[string]any{}), nil); err == nil {
		t.Fatal("ending a fight outside a fight should fail")
	}
}

func TestFightRoundsRegenerateManaAndFinishCooldowns(t *testing.T) {
	s, cat := fightState(t)
	s, _ = applyWith(t, s, cmd(t, "fight.start", map[string]any{}), cat)
	s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-1", "abilityId": "ability-1"}), cat)
	s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-2"}), cat)
	if s.Heroes[0].Mana != 10 {
		t.Fatalf("mana after the heal: %d", s.Heroes[0].Mana)
	}
	s, ev := applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	if s.Heroes[0].Mana != 13 || !strings.Contains(ev.Summary, "mana: Mira 10 → 13") {
		t.Fatalf("round 2: %d %q", s.Heroes[0].Mana, ev.Summary)
	}
	s, _ = applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	s, ev = applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	// Capped at the maximum; a full hero isn't mentioned. Fan of Cards (used in round 1, cooldown 3) is ready in round 4.
	if s.Heroes[0].Mana != 16 || strings.Contains(ev.Summary, "Mira 16") || !strings.Contains(ev.Summary, "ready again: Vex's Fan of Cards") {
		t.Fatalf("round 4: %d %q", s.Heroes[0].Mana, ev.Summary)
	}
}

func TestCooldownsStopShortOutOfAFight(t *testing.T) {
	s, cat := fightState(t)
	s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-1"}), cat) // long: 5
	s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-2"}), cat) // short: 3
	s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-1", "abilityId": "ability-1"}), cat) // mana 10
	left := func(s *State) [2]int {
		vex := s.Heroes[1]
		return [2]int{vex.CooldownLeft("ability-1", s.Round), vex.CooldownLeft("ability-2", s.Round)}
	}
	// Out of a fight: long cooldowns stop at 2 rounds left, short ones at 1.
	want := [][2]int{{4, 2}, {3, 1}, {2, 1}, {2, 1}}
	for i, w := range want {
		var ev Event
		s, ev = applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
		if got := left(s); got != w {
			t.Fatalf("round %d out of a fight: %v, want %v (%q)", i+2, got, w, ev.Summary)
		}
	}
	if s.Heroes[0].Mana != 10 {
		t.Fatalf("no mana comes back out of a fight: %d", s.Heroes[0].Mana)
	}
	// In a fight they finish.
	s, _ = applyWith(t, s, cmd(t, "fight.start", map[string]any{}), cat)
	s, ev := applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	if got := left(s); got != [2]int{1, 0} || !strings.Contains(ev.Summary, "ready again: Vex's Fan of Cards") {
		t.Fatalf("first fight round: %v %q", got, ev.Summary)
	}
	// A long cooldown already down to 1 stays at 1 out of a fight.
	s, _ = applyWith(t, s, cmd(t, "fight.end", map[string]any{}), cat)
	s, _ = applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	if got := left(s); got != [2]int{1, 0} {
		t.Fatalf("after the fight: %v", got)
	}
}

func TestEffectsCountDownInFights(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "effect.add", map[string]any{"target": "hero-1", "name": "Raging", "rounds": 3, "note": "+3 damage, +2 mitigation"}))
	if len(s.Heroes[0].Effects) != 1 || ev.Summary != "Grom: Raging (3 rounds): +3 damage, +2 mitigation" {
		t.Fatalf("hero effect: %+v %q", s.Heroes[0].Effects, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "effect.add", map[string]any{"target": "monster-1", "name": "Poisoned", "rounds": 3, "note": "3 a turn"}))
	if len(s.Monsters[0].Effects) != 1 || ev.Summary != "Orc (monster-1): Poisoned (3 rounds): 3 a turn" {
		t.Fatalf("monster effect: %+v %q", s.Monsters[0].Effects, ev.Summary)
	}
	s, _ = apply(t, s, cmd(t, "effect.add", map[string]any{"target": "hero-2", "name": "Cursed"}))

	// Countdowns only run in fight rounds.
	s, _ = apply(t, s, cmd(t, "round.advance", map[string]any{}))
	if s.Heroes[0].Effects[0].Rounds != 3 {
		t.Fatalf("out of a fight: %+v", s.Heroes[0].Effects)
	}
	s, _ = apply(t, s, cmd(t, "fight.start", map[string]any{}))
	s, _ = apply(t, s, cmd(t, "round.advance", map[string]any{}))
	s, _ = apply(t, s, cmd(t, "round.advance", map[string]any{}))
	if s.Heroes[0].Effects[0].Rounds != 1 || s.Monsters[0].Effects[0].Rounds != 1 {
		t.Fatalf("after two fight rounds: %+v %+v", s.Heroes[0].Effects, s.Monsters[0].Effects)
	}
	s, ev = apply(t, s, cmd(t, "round.advance", map[string]any{}))
	if len(s.Heroes[0].Effects) != 0 || len(s.Monsters[0].Effects) != 0 ||
		!strings.Contains(ev.Summary, "ended: Grom's Raging, Orc (monster-1)'s Poisoned") {
		t.Fatalf("countdowns run out: %+v %q", s.Heroes[0].Effects, ev.Summary)
	}

	// Ending a fight ends effects with a countdown; others stay until removed.
	s, _ = apply(t, s, cmd(t, "effect.add", map[string]any{"target": "monster-2", "name": "Marked", "rounds": 9}))
	s, ev = apply(t, s, cmd(t, "fight.end", map[string]any{}))
	if len(s.Monsters[1].Effects) != 0 || len(s.Heroes[1].Effects) != 1 || ev.Summary != "Fight over; ended: Orc (monster-2)'s Marked" {
		t.Fatalf("fight end: %+v %+v %q", s.Monsters[1].Effects, s.Heroes[1].Effects, ev.Summary)
	}
	id := s.Heroes[1].Effects[0].ID
	s, ev = apply(t, s, cmd(t, "effect.remove", map[string]any{"target": "hero-2", "id": id}))
	if len(s.Heroes[1].Effects) != 0 || ev.Summary != "Ilsa: Cursed removed" {
		t.Fatalf("remove: %q", ev.Summary)
	}

	for name, payload := range map[string]map[string]any{
		"unknown target":  {"target": "hero-9", "name": "Raging"},
		"no name":         {"target": "hero-1", "name": " "},
		"long name":       {"target": "hero-1", "name": strings.Repeat("x", 41)},
		"too many rounds": {"target": "hero-1", "name": "Raging", "rounds": 100},
		"long note":       {"target": "hero-1", "name": "Raging", "note": strings.Repeat("x", 201)},
	} {
		if _, _, err := Apply(s, cmd(t, "effect.add", payload), nil); err == nil {
			t.Errorf("effect.add %s should fail", name)
		}
	}
	if _, _, err := Apply(s, cmd(t, "effect.remove", map[string]any{"target": "hero-1", "id": "effect-99"}), nil); err == nil {
		t.Error("removing an unknown effect should fail")
	}
}
