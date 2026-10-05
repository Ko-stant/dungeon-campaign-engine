package tracker

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/combat"
)

func seat(hero string) Actor { return Actor{Kind: ActorSeat, HeroID: hero} }

// as returns the command sent by an actor.
func as(actor Actor, c Command) Command {
	c.Actor = actor
	return c
}

// applyErr applies a command that should fail and returns the error text.
func applyErr(t *testing.T, s *State, c Command) string {
	t.Helper()
	_, _, cat := fixture()
	_, _, err := Apply(s, c, cat)
	if err == nil {
		t.Fatalf("%s by %+v: expected an error", c.Type, c.Actor)
	}
	return err.Error()
}

func rulesState(t *testing.T) *State {
	t.Helper()
	s, _ := apply(t, newState(t), cmd(t, "rules.enable", map[string]any{"seed": 42}))
	return s
}

func TestRulesEnableStartsTheHeroesPhase(t *testing.T) {
	s, ev := apply(t, newState(t), cmd(t, "rules.enable", map[string]any{"seed": 42}))
	r := s.Rules
	if r == nil || r.Ruleset != RulesetThreePlagues || r.RNG != 42 || r.Phase != PhaseHeroes {
		t.Fatalf("rules = %+v", r)
	}
	if s.Heroes[0].Movement != "2d6" || s.Heroes[1].Movement != "1d6+2" {
		t.Errorf("hero movement frozen from the class: %q, %q", s.Heroes[0].Movement, s.Heroes[1].Movement)
	}
	if s.Monsters[0].Movement != 8 {
		t.Errorf("monster movement frozen from the catalog: %d", s.Monsters[0].Movement)
	}
	if ev.Summary != "Rules on (three-plagues/1): the heroes' turns" {
		t.Errorf("summary %q", ev.Summary)
	}
	if !strings.Contains(applyErr(t, s, cmd(t, "rules.enable", map[string]any{"seed": 1})), "already on") {
		t.Error("enabling twice should say the rules are already on")
	}
}

func TestRulesDisableKeepsTheGame(t *testing.T) {
	s := rulesState(t)
	s, ev := apply(t, s, cmd(t, "rules.disable", map[string]any{}))
	if s.Rules != nil || ev.Summary != "Rules off: the GM records the game" {
		t.Fatalf("rules %+v, summary %q", s.Rules, ev.Summary)
	}
	applyErr(t, s, cmd(t, "rules.disable", map[string]any{}))
}

func TestPlayersOnlySendTurnCommandsInRulesMode(t *testing.T) {
	table := newState(t)
	if msg := applyErr(t, table, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"}))); !strings.Contains(msg, "rules are off") {
		t.Errorf("table mode: %q", msg)
	}
	s := rulesState(t)
	if msg := applyErr(t, s, as(seat("hero-1"), cmd(t, "move", map[string]any{"id": "hero-1", "x": 3, "y": 3}))); !strings.Contains(msg, "GM") {
		t.Errorf("a GM command from a seat: %q", msg)
	}
	if msg := applyErr(t, s, as(seat("hero-1"), cmd(t, "rules.disable", map[string]any{}))); !strings.Contains(msg, "GM") {
		t.Errorf("rules.disable from a seat: %q", msg)
	}
	bad := cmd(t, "turn.start", map[string]any{"hero": "hero-1"})
	bad.Actor = Actor{Kind: "wizard"}
	applyErr(t, s, bad)
}

func TestATurnBelongsToItsHero(t *testing.T) {
	s := rulesState(t)
	if msg := applyErr(t, s, as(seat("hero-2"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"}))); !strings.Contains(msg, "your own hero") {
		t.Errorf("starting another hero's turn: %q", msg)
	}
	s, ev := apply(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"})))
	if s.Rules.Turn == nil || s.Rules.Turn.HeroID != "hero-1" || ev.Summary != "Grom (Barbarian) starts their turn" {
		t.Fatalf("turn %+v, summary %q", s.Rules.Turn, ev.Summary)
	}
	if msg := applyErr(t, s, as(seat("hero-2"), cmd(t, "turn.start", map[string]any{"hero": "hero-2"}))); !strings.Contains(msg, "Grom") {
		t.Errorf("one turn at a time: %q", msg)
	}
	if msg := applyErr(t, s, as(seat("hero-2"), cmd(t, "turn.end", map[string]any{}))); !strings.Contains(msg, "your own hero") {
		t.Errorf("ending another hero's turn: %q", msg)
	}
	s, ev = apply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	if s.Rules.Turn != nil || len(s.Rules.Acted) != 1 || ev.Summary != "Grom (Barbarian) ends their turn" {
		t.Fatalf("after the turn: %+v, summary %q", s.Rules, ev.Summary)
	}
	if msg := applyErr(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"}))); !strings.Contains(msg, "already") {
		t.Errorf("a second turn in a round: %q", msg)
	}
}

func TestTheGMCanRunAndEndAnyTurn(t *testing.T) {
	s := rulesState(t)
	s, _ = apply(t, s, cmd(t, "turn.start", map[string]any{"hero": "hero-2"}))
	s, _ = apply(t, s, cmd(t, "turn.end", map[string]any{}))
	if len(s.Rules.Acted) != 1 || s.Rules.Acted[0] != "hero-2" {
		t.Fatalf("acted = %v", s.Rules.Acted)
	}
}

func TestDeadHeroesTakeNoTurn(t *testing.T) {
	s := rulesState(t)
	s.Heroes[0].Status = HeroDead
	if msg := applyErr(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"}))); !strings.Contains(msg, "dead") {
		t.Errorf("dead hero: %q", msg)
	}
}

func TestACriticalMissCostsTheNextTurn(t *testing.T) {
	s := rulesState(t)
	s.Rules.SkipNext = []string{"hero-1"}
	s, ev := apply(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"})))
	if s.Rules.Turn != nil || len(s.Rules.SkipNext) != 0 || len(s.Rules.Acted) != 1 {
		t.Fatalf("skipped turn: %+v", s.Rules)
	}
	if ev.Summary != "Grom (Barbarian) loses this turn (critical miss)" {
		t.Errorf("summary %q", ev.Summary)
	}
}

func TestPhasesRunHeroesMonstersThenANewRound(t *testing.T) {
	s := rulesState(t)
	s, _ = apply(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"})))
	if msg := applyErr(t, s, cmd(t, "phase.monsters", map[string]any{})); !strings.Contains(msg, "Grom") {
		t.Errorf("the monsters wait for the active turn to end: %q", msg)
	}
	s, _ = apply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	s, ev := apply(t, s, cmd(t, "phase.monsters", map[string]any{}))
	if s.Rules.Phase != PhaseMonsters || ev.Summary != "The monsters' turn (1 hero did not act: Ilsa)" {
		t.Fatalf("phase %q, summary %q", s.Rules.Phase, ev.Summary)
	}
	if msg := applyErr(t, s, as(seat("hero-2"), cmd(t, "turn.start", map[string]any{"hero": "hero-2"}))); !strings.Contains(msg, "monsters") {
		t.Errorf("no hero turns in the monsters' phase: %q", msg)
	}
	if msg := applyErr(t, s, as(seat("hero-2"), cmd(t, "phase.end", map[string]any{}))); !strings.Contains(msg, "GM") {
		t.Errorf("phase.end from a seat: %q", msg)
	}
	s, ev = apply(t, s, cmd(t, "phase.end", map[string]any{}))
	if s.Round != 2 || s.Rules.Phase != PhaseHeroes || len(s.Rules.Acted) != 0 {
		t.Fatalf("new round: round %d, %+v", s.Round, s.Rules)
	}
	if ev.Summary != "Round 2 begins: the heroes' turns" {
		t.Errorf("summary %q", ev.Summary)
	}
}

func TestTheGMMaySkipTheMonstersPhase(t *testing.T) {
	s := rulesState(t)
	s, _ = apply(t, s, cmd(t, "phase.end", map[string]any{}))
	if s.Round != 2 || s.Rules.Phase != PhaseHeroes {
		t.Fatalf("round %d phase %q", s.Round, s.Rules.Phase)
	}
}

func TestRulesModeRunsFightsFromRevealedMonsters(t *testing.T) {
	s := newState(t)
	s, _ = apply(t, s, cmd(t, "seen.set", map[string]any{"id": "monster-1", "seen": true}))
	s, ev := apply(t, s, cmd(t, "rules.enable", map[string]any{"seed": 1}))
	if !s.Fight || !strings.HasSuffix(ev.Summary, "; fight started") {
		t.Fatalf("a revealed monster starts a fight: fight %v, %q", s.Fight, ev.Summary)
	}
	s.Monsters[0].Alive = false
	s, ev = apply(t, s, cmd(t, "phase.end", map[string]any{}))
	if s.Fight || !strings.Contains(ev.Summary, "; fight over") {
		t.Fatalf("no revealed monster left ends it: fight %v, %q", s.Fight, ev.Summary)
	}
}

func TestEventsRecordTheActor(t *testing.T) {
	s := rulesState(t)
	_, ev := apply(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"})))
	var p struct {
		Actor Actor `json:"actor"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.Actor != seat("hero-1") {
		t.Errorf("payload actor = %+v (%s)", p.Actor, ev.Payload)
	}
	_, ev = apply(t, s, cmd(t, "turn.start", map[string]any{"hero": "hero-1"}))
	if strings.Contains(string(ev.Payload), "actor") {
		t.Errorf("GM commands keep the old payload shape: %s", ev.Payload)
	}
}

func TestOnlyTheGMGivesDice(t *testing.T) {
	s := rulesState(t)
	c := as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"}))
	c.Dice = []int{6, 6}
	if msg := applyErr(t, s, c); !strings.Contains(msg, "dice") {
		t.Errorf("dice from a seat: %q", msg)
	}
	c.Actor = Actor{}
	if msg := applyErr(t, s, c); !strings.Contains(msg, "not used") {
		t.Errorf("dice that nothing rolls: %q", msg)
	}
}

func startTurn(t *testing.T, s *State, hero string) *State {
	t.Helper()
	s, _ = apply(t, s, as(seat(hero), cmd(t, "turn.start", map[string]any{"hero": hero})))
	return s
}

func TestMovementIsRolledFromTheSeededDice(t *testing.T) {
	s := startTurn(t, rulesState(t), "hero-1")
	s, ev := apply(t, s, as(seat("hero-1"), cmd(t, "turn.roll-move", map[string]any{})))

	want := &combat.Mulberry32{State: 42}
	a, b := want.Die(6), want.Die(6)
	if s.Rules.Turn.MoveRoll != a+b || s.Rules.Turn.MoveLeft != a+b {
		t.Fatalf("turn = %+v, want a roll of %d", s.Rules.Turn, a+b)
	}
	if s.Rules.RNG != want.State {
		t.Errorf("the roller state should move on: %d, want %d", s.Rules.RNG, want.State)
	}
	if wantSummary := fmt.Sprintf("Grom (Barbarian) rolls %d for movement (2d6: %d, %d)", a+b, a, b); ev.Summary != wantSummary {
		t.Errorf("summary %q, want %q", ev.Summary, wantSummary)
	}
	var p struct {
		Rolls []combat.Die `json:"rolls"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Rolls) != 2 || p.Rolls[0] != (combat.Die{Sides: 6, Value: a}) {
		t.Errorf("rolls in the event: %+v", p.Rolls)
	}
	if msg := applyErr(t, s, as(seat("hero-1"), cmd(t, "turn.roll-move", map[string]any{}))); !strings.Contains(msg, "already") {
		t.Errorf("a second roll: %q", msg)
	}
}

func TestTheGMMayRollMovementWithRealDice(t *testing.T) {
	s := startTurn(t, rulesState(t), "hero-2")
	c := cmd(t, "turn.roll-move", map[string]any{})
	c.Dice = []int{5}
	s, ev := apply(t, s, c)
	if s.Rules.Turn.MoveLeft != 7 || s.Rules.RNG != 42 {
		t.Fatalf("1d6+2 with a 5: %+v, rng %d", s.Rules.Turn, s.Rules.RNG)
	}
	if ev.Summary != "Ilsa (Wizard) rolls 7 for movement (1d6+2: 5)" {
		t.Errorf("summary %q", ev.Summary)
	}
	c.Dice = []int{9}
	if msg := applyErr(t, startTurn(t, rulesState(t), "hero-2"), c); !strings.Contains(msg, "d6") {
		t.Errorf("a 9 on a d6: %q", msg)
	}
}
