package tracker

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

func TestTheQuestIsLostWhenNoHeroIsLeft(t *testing.T) {
	s := monstersPhase(t)
	s.Heroes[1].Status = HeroDead
	s.Heroes[0].Body = 2
	s, ev := hallApply(t, s, monsterAttack(t, "hero-1", 8, 8, 1, 2, 2))
	if s.Rules.Phase != PhaseOver || s.Rules.Outcome != OutcomeLost {
		t.Fatalf("rules %+v", s.Rules)
	}
	if !strings.HasSuffix(ev.Summary, "; the quest is lost") {
		t.Errorf("summary %q", ev.Summary)
	}
	if msg := hallErr(t, s, cmd(t, "phase.end", map[string]any{})); !strings.Contains(msg, "over") {
		t.Errorf("after the end: %q", msg)
	}
}

func TestAKillObjectiveWinsTheQuest(t *testing.T) {
	s := duelState(t)
	s.Quest.Objectives = []maps.Objective{{Kind: maps.ObjectiveKill}}
	s.Monsters[0].Body = 1
	s, ev := hallApply(t, s, attack(t, "hero-1", 15, 5))
	if s.Rules.Outcome != OutcomeWon || s.Rules.Phase != PhaseOver || s.Rules.Turn != nil {
		t.Fatalf("rules %+v", s.Rules)
	}
	if !strings.HasSuffix(ev.Summary, "; the quest is won") {
		t.Errorf("summary %q", ev.Summary)
	}
}

func TestAKillObjectiveCanNameItsMonsters(t *testing.T) {
	s := duelState(t)
	s.Quest.Objectives = []maps.Objective{{Kind: maps.ObjectiveKill, Monsters: []string{"boss"}}}
	s.Monsters = append(s.Monsters, Monster{ID: "boss", Type: "orc", Name: "Orc", X: 1, Y: 3, Body: 5, MaxBody: 5, Alive: true, Visibility: MonsterHidden, Width: 1, Height: 1})
	s.Monsters[0].Body = 1
	s, _ = hallApply(t, s, attack(t, "hero-1", 15, 5))
	if s.Rules.Outcome != "" {
		t.Fatalf("the boss still lives: %+v", s.Rules)
	}
}

func TestClearingEveryMonsterCompletesAQuestWithoutObjectives(t *testing.T) {
	s := duelState(t)
	s.Monsters[0].Body = 1
	s, ev := hallApply(t, s, attack(t, "hero-1", 15, 5))
	if s.Rules.Outcome != OutcomeWon || !strings.HasSuffix(ev.Summary, "; the quest is won") {
		t.Fatalf("rules %+v, %q", s.Rules, ev.Summary)
	}
}

func TestACollectObjective(t *testing.T) {
	s := duelState(t)
	s.Quest.Objectives = []maps.Objective{{Kind: maps.ObjectiveCollect, Item: "Soul Gem"}}
	s, _ = hallApply(t, s, cmd(t, "item.add", map[string]any{"heroId": "hero-2", "name": "soul gem"}))
	if s.Rules.Outcome != OutcomeWon {
		t.Fatalf("Ilsa carries the gem: %+v", s.Rules)
	}
}

// exitState: searchState with an exit at (4,2) and an escape objective.
func exitState(t *testing.T) *State {
	t.Helper()
	return searchState(t, func(q *maps.Quest) {
		q.ExitTiles = []maps.Tile{{X: 4, Y: 2}}
		q.Objectives = []maps.Objective{{Kind: maps.ObjectiveEscape}}
	})
}

func TestHeroesLeaveByAnExit(t *testing.T) {
	s := exitState(t)
	exit := func(hero string) Command { return as(seat(hero), cmd(t, "turn.exit", map[string]any{})) }
	if msg := hallErr(t, s, exit("hero-1")); !strings.Contains(msg, "exit") {
		t.Errorf("away from the exit: %q", msg)
	}
	s.Heroes[0].Y = 2
	s, ev := hallApply(t, s, exit("hero-1"))
	if s.Heroes[0].Status != HeroEscaped || s.Rules.Turn != nil || s.Rules.Outcome != "" {
		t.Fatalf("Grom %+v, rules %+v", s.Heroes[0], s.Rules)
	}
	if ev.Summary != "Grom (Barbarian) leaves by the exit at (4,2)" {
		t.Errorf("summary %q", ev.Summary)
	}
	s.Heroes[1].X, s.Heroes[1].Y = 4, 2
	s, _ = hallApply(t, s, as(seat("hero-2"), cmd(t, "turn.start", map[string]any{"hero": "hero-2"})))
	s, ev = hallApply(t, s, exit("hero-2"))
	if s.Rules.Outcome != OutcomeWon || !strings.HasSuffix(ev.Summary, "; the quest is won") {
		t.Fatalf("everyone is out: %+v, %q", s.Rules, ev.Summary)
	}
}

func TestLeavingBeforeTheQuestIsDoneLosesIt(t *testing.T) {
	s := duelState(t)
	s.Quest.ExitTiles = []maps.Tile{{X: 4, Y: 2}, {X: 4, Y: 1}}
	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.exit", map[string]any{})))
	s, _ = hallApply(t, s, as(seat("hero-2"), cmd(t, "turn.start", map[string]any{"hero": "hero-2"})))
	s, ev := hallApply(t, s, as(seat("hero-2"), cmd(t, "turn.exit", map[string]any{})))
	if s.Rules.Outcome != OutcomeLost || !strings.HasSuffix(ev.Summary, "; the quest is lost") {
		t.Fatalf("the orc lives on: %+v, %q", s.Rules, ev.Summary)
	}
}

func TestThePlayersSeeTheGoal(t *testing.T) {
	s := duelState(t)
	s.Quest.Goal = "Slay the orc and escape"
	if got := PlayerView(s).Goal; got != "Slay the orc and escape" {
		t.Errorf("goal %q", got)
	}
}

func TestTheGMEndsTheQuest(t *testing.T) {
	s := rulesState(t)
	if msg := applyErr(t, s, cmd(t, "quest.end", map[string]any{"outcome": "draw"})); !strings.Contains(msg, "won or lost") {
		t.Errorf("a bad outcome: %q", msg)
	}
	s, ev := apply(t, s, cmd(t, "quest.end", map[string]any{"outcome": "won"}))
	if s.Rules.Outcome != OutcomeWon || s.Rules.Phase != PhaseOver || ev.Summary != "The GM ends the quest: won" {
		t.Fatalf("rules %+v, %q", s.Rules, ev.Summary)
	}
	if msg := applyErr(t, s, as(seat("hero-1"), cmd(t, "turn.start", map[string]any{"hero": "hero-1"}))); !strings.Contains(msg, "over") {
		t.Errorf("a turn after the end: %q", msg)
	}
	if msg := applyErr(t, s, as(seat("hero-1"), cmd(t, "quest.end", map[string]any{"outcome": "won"}))); !strings.Contains(msg, "GM") {
		t.Errorf("from a seat: %q", msg)
	}
}
