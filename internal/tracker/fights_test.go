package tracker

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

// fightState is a session with a Cleric who regenerates mana (16 max, 3 a fight
// round) and a Ranger with a long (5), a short (3) and a 1-round cooldown.
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
			{ID: "ability-3", Name: "Quick Shot", Kind: content.AbilityActive, Cooldown: 1},
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

func TestFightEndDropsCooldowns(t *testing.T) {
	s, cat := fightState(t)
	s, _ = applyWith(t, s, cmd(t, "fight.start", map[string]any{}), cat)
	for _, id := range []string{"ability-1", "ability-2", "ability-3"} {
		s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": id}), cat)
	}
	// The fight ends at once, with 5, 3 and 1 rounds left: long cooldowns drop to
	// 2, short ones to 1, and a 1-round cooldown is ready again.
	s, ev := applyWith(t, s, cmd(t, "fight.end", map[string]any{}), cat)
	vex := s.Heroes[1]
	got := [3]int{vex.CooldownLeft("ability-1", s.Round), vex.CooldownLeft("ability-2", s.Round), vex.CooldownLeft("ability-3", s.Round)}
	if got != [3]int{2, 1, 0} {
		t.Fatalf("after the fight: %v (%q)", got, ev.Summary)
	}
	if _, ok := vex.Cooldowns["ability-3"]; ok {
		t.Fatal("a cooldown that drops to 0 leaves the list")
	}
	want := "Fight over; cooldowns: Vex's Multi-Shot 2 rounds left, Vex's Fan of Cards 1 round left, Vex's Quick Shot ready"
	if ev.Summary != want {
		t.Fatalf("summary: %q, want %q", ev.Summary, want)
	}
	// They wait there until the next fight.
	s, _ = applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	s, _ = applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	vex = s.Heroes[1]
	if got := [2]int{vex.CooldownLeft("ability-1", s.Round), vex.CooldownLeft("ability-2", s.Round)}; got != [2]int{2, 1} {
		t.Fatalf("rounds out of a fight: %v", got)
	}
	// A cooldown already below its floor stays; nothing to drop means no note.
	s, _ = applyWith(t, s, cmd(t, "fight.start", map[string]any{}), cat)
	s, _ = applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	s, ev = applyWith(t, s, cmd(t, "fight.end", map[string]any{}), cat)
	if left := s.Heroes[1].CooldownLeft("ability-1", s.Round); left != 1 || ev.Summary != "Fight over" {
		t.Fatalf("1 round left stays at 1: %d %q", left, ev.Summary)
	}
}

func TestOneRoundCooldownFinishesOutOfAFight(t *testing.T) {
	s, cat := fightState(t)
	s, _ = applyWith(t, s, cmd(t, "ability.use", map[string]any{"heroId": "hero-2", "abilityId": "ability-3"}), cat)
	s, ev := applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	if left := s.Heroes[1].CooldownLeft("ability-3", s.Round); left != 0 || !strings.Contains(ev.Summary, "ready again: Vex's Quick Shot") {
		t.Fatalf("next round: %d %q", left, ev.Summary)
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

func TestDetermination(t *testing.T) {
	s, cat := fightState(t)
	s, _ = applyWith(t, s, cmd(t, "fight.start", map[string]any{}), cat)
	s, ev := applyWith(t, s, cmd(t, "hero.update", map[string]any{"id": "hero-2", "determination": 2}), cat)
	if s.Heroes[1].Determination != 2 || ev.Summary != "Vex: Determination +0 → +2" {
		t.Fatalf("a miss: %d %q", s.Heroes[1].Determination, ev.Summary)
	}
	// The TV shows it on the hero's card; the feed stays quiet about it.
	if ev.PlayerSummary != "" || PlayerView(s).Heroes[1].Determination != 2 {
		t.Fatalf("players: %q %+v", ev.PlayerSummary, PlayerView(s).Heroes[1])
	}
	if _, _, err := Apply(s, cmd(t, "hero.update", map[string]any{"id": "hero-2", "determination": -2}), cat); err == nil {
		t.Fatal("negative Determination should fail")
	}
	// A fight's end clears every streak.
	s, ev = applyWith(t, s, cmd(t, "fight.end", map[string]any{}), cat)
	if s.Heroes[1].Determination != 0 || ev.Summary != "Fight over; Determination reset: Vex" {
		t.Fatalf("fight over: %d %q", s.Heroes[1].Determination, ev.Summary)
	}
}
