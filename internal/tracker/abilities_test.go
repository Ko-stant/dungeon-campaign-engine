package tracker

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

// abilityState is a session with a Cleric (mana, spells) and a Ranger
// (cooldowns) from custom classes.
func abilityState(t *testing.T) (*State, *content.Catalog) {
	t.Helper()
	b, q, cat := fixture()
	cat.Heroes = append(cat.Heroes,
		content.HeroDef{ID: "custom-cleric", Name: "Cleric", Body: 20, Mind: 4, Mana: 10, Custom: true, Abilities: []content.Ability{
			{ID: "ability-1", Name: "Heal Minor Wounds", Kind: content.AbilitySpell, ManaCost: 3},
			{ID: "ability-2", Name: "Divine Blessing", Kind: content.AbilitySpell, ManaCost: 2, Cooldown: 10},
		}},
		content.HeroDef{ID: "custom-ranger", Name: "Ranger", Body: 18, Mind: 3, Custom: true, Abilities: []content.Ability{
			{ID: "ability-1", Name: "Rain of Arrows", Kind: content.AbilityActive, Cooldown: 6},
			{ID: "ability-2", Name: "Multi-Shot", Kind: content.AbilityActive, Cooldown: 2},
			{ID: "ability-3", Name: "Keen Eye", Kind: content.AbilityPassive},
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

func applyWith(t *testing.T, s *State, c Command, cat *content.Catalog) (*State, Event) {
	t.Helper()
	next, ev, err := Apply(s, c, cat)
	if err != nil {
		t.Fatalf("%s: %v", c.Type, err)
	}
	return next, ev
}

func TestNewSessionCopiesClassAbilitiesAndMana(t *testing.T) {
	s, _ := abilityState(t)
	mira, vex := s.Heroes[0], s.Heroes[1]
	if mira.Mana != 10 || mira.MaxMana != 10 || len(mira.Abilities) != 2 || mira.Abilities[1].Cooldown != 10 {
		t.Fatalf("cleric: %+v", mira)
	}
	if vex.Mana != 0 || len(vex.Abilities) != 3 || len(vex.Cooldowns) != 0 {
		t.Fatalf("ranger: %+v", vex)
	}
	// The session keeps its own copy: later class edits never change a game.
	_, _, cat := fixture()
	if len(cat.Heroes[0].Abilities) != 0 {
		t.Fatal("fixture classes changed")
	}
}

func TestNewSessionCopiesClassCombatStats(t *testing.T) {
	b, q, cat := fixture()
	cat.Heroes = append(cat.Heroes, content.HeroDef{
		ID: "custom-barbarian", Name: "Barbarian", Body: 40, Mind: 2, Custom: true,
		AttackDice: "1d20", DefenseDice: "1d6", Accuracy: 3, CritFrom: 17, Damage: 3, Avoidance: 2, Mitigation: 1,
	}, content.HeroDef{
		ID: "custom-cleric", Name: "Cleric", Body: 28, Mind: 4, Custom: true, Mana: 16, ManaRegen: 2,
		AttackDice: "2d8", DefenseDice: "1d6", Accuracy: 4, CritFrom: 20, Damage: 2, Avoidance: 3,
	})
	s, err := NewSession(b, q, "The Test", []CampaignHero{
		{ID: "hero-1", Name: "Brak", Class: "custom-barbarian"},
		{ID: "hero-2", Name: "Mira", Class: "custom-cleric"},
		{ID: "hero-3", Name: "Old", Class: cat.Heroes[0].ID},
	}, cat)
	if err != nil {
		t.Fatal(err)
	}
	want := Combat{HitDice: "1d20", Accuracy: 3, CritFrom: 17, Damage: 3, DefenseDice: "1d6", Avoidance: 2, Mitigation: 1}
	if got := s.Heroes[0].Combat; got == nil || *got != want {
		t.Fatalf("barbarian combat: %+v", got)
	}
	if got := s.Heroes[1].Combat; got == nil || got.ManaRegen != 2 || got.CritFrom != 20 {
		t.Fatalf("cleric combat: %+v", got)
	}
	// Built-in classes roll combat dice: no combat stats to copy.
	if s.Heroes[2].Combat != nil {
		t.Fatalf("built-in class: %+v", s.Heroes[2].Combat)
	}
}

func TestAbilityCooldowns(t *testing.T) {
	s, cat := abilityState(t)
	s.Round = 4

	s, ev := applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-1"}), cat)
	if s.Heroes[1].Cooldowns["ability-1"] != 10 || ev.Summary != "Vex used Rain of Arrows: ready again in round 10" {
		t.Fatalf("use: %+v %q", s.Heroes[1].Cooldowns, ev.Summary)
	}
	if left := s.Heroes[1].CooldownLeft("ability-1", s.Round); left != 6 {
		t.Fatalf("cooldown left in round 4: %d", left)
	}
	s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-2"}), cat)

	// Using it again while cooling down is recorded, never refused.
	s.Round = 5
	s, ev = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-1"}), cat)
	if s.Heroes[1].Cooldowns["ability-1"] != 11 || ev.Summary != "Vex used Rain of Arrows (still cooling down, 5 rounds left): ready again in round 11" {
		t.Fatalf("reuse: %+v %q", s.Heroes[1].Cooldowns, ev.Summary)
	}

	// Multi-Shot (used in round 4, cooldown 2) is ready in round 6; advancing announces it.
	s, ev = applyWith(t, s, cmd(t, "round.advance", nil), cat)
	if ev.Summary != "Round 6 begins; ready again: Vex's Multi-Shot" {
		t.Fatalf("advance: %q", ev.Summary)
	}
	if _, ok := s.Heroes[1].Cooldowns["ability-2"]; ok {
		t.Fatal("ready abilities leave the cooldown list")
	}
	s, ev = applyWith(t, s, cmd(t, "round.advance", nil), cat)
	if ev.Summary != "Round 7 begins" {
		t.Fatalf("quiet advance: %q", ev.Summary)
	}

	// A reset (Divine Blessing, or a GM fix) ends cooldowns.
	s, ev = applyWith(t, s, cmd(t, "ability.reset", map[string]any{"heroId": "hero-2", "abilityId": "ability-1"}), cat)
	if len(s.Heroes[1].Cooldowns) != 0 || ev.Summary != "Vex: Rain of Arrows is ready again" {
		t.Fatalf("reset one: %+v %q", s.Heroes[1].Cooldowns, ev.Summary)
	}
	s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-1"}), cat)
	s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-2"}), cat)
	s, ev = applyWith(t, s, cmd(t, "ability.reset", map[string]any{"heroId": "hero-2"}), cat)
	if len(s.Heroes[1].Cooldowns) != 0 || ev.Summary != "Vex: all abilities are ready again (Rain of Arrows, Multi-Shot)" {
		t.Fatalf("reset all: %+v %q", s.Heroes[1].Cooldowns, ev.Summary)
	}

	for name, c := range map[string]Command{
		"unknown ability":  cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-9"}),
		"unknown hero":     cmd(t, "ability.use", map[string]any{"heroId": "hero-9", "abilityId": "ability-1"}),
		"nothing to reset": cmd(t, "ability.reset", map[string]any{"heroId": "hero-2"}),
		"ready already":    cmd(t, "ability.reset", map[string]any{"heroId": "hero-2", "abilityId": "ability-1"}),
	} {
		if _, _, err := Apply(s, c, cat); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestSpellsSpendMana(t *testing.T) {
	s, cat := abilityState(t)

	s, ev := applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-1", "abilityId": "ability-1"}), cat)
	if s.Heroes[0].Mana != 7 || ev.Summary != "Mira cast Heal Minor Wounds: mana 10 → 7" {
		t.Fatalf("cast: %d %q", s.Heroes[0].Mana, ev.Summary)
	}
	s, ev = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-1", "abilityId": "ability-2"}), cat)
	if s.Heroes[0].Mana != 5 || ev.Summary != "Mira cast Divine Blessing: mana 7 → 5, ready again in round 11" {
		t.Fatalf("cast with cooldown: %d %q", s.Heroes[0].Mana, ev.Summary)
	}

	// Not enough mana is noted, not refused; mana stops at 0.
	s.Heroes[0].Mana = 1
	s, ev = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-1", "abilityId": "ability-1"}), cat)
	if s.Heroes[0].Mana != 0 || ev.Summary != "Mira cast Heal Minor Wounds (needs 3 mana, had 1): mana 1 → 0" {
		t.Fatalf("short of mana: %d %q", s.Heroes[0].Mana, ev.Summary)
	}

	// Mana is set like Body and Mind.
	s, ev = applyWith(t, s, cmd(t, "hero.update", map[string]any{"id": "hero-1", "mana": 4, "maxMana": 12}), cat)
	if s.Heroes[0].Mana != 4 || s.Heroes[0].MaxMana != 12 || ev.Summary != "Mira: mana 0 → 4, max mana 10 → 12" {
		t.Fatalf("set mana: %+v %q", s.Heroes[0], ev.Summary)
	}
	if _, _, err := Apply(s, cmd(t, "hero.update", map[string]any{"id": "hero-1", "mana": -1}), cat); err == nil {
		t.Fatal("negative mana")
	}
}
