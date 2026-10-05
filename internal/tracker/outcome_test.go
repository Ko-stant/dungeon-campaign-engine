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

func TestAnEscapeObjective(t *testing.T) {
	s := searchState(t, func(q *maps.Quest) {
		q.ExitTiles = []maps.Tile{{X: 4, Y: 2}, {X: 1, Y: 2}}
		q.Objectives = []maps.Objective{{Kind: maps.ObjectiveEscape}}
	})
	roll := cmd(t, "turn.roll-move", map[string]any{})
	roll.Dice = []int{1, 1}
	s, _ = hallApply(t, s, roll)
	s, ev := hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.move", map[string]any{"to": map[string]int{"x": 4, "y": 2}})))
	if s.Rules.Outcome != "" {
		t.Fatalf("Ilsa is not out yet: %+v %q", s.Rules, ev.Summary)
	}
	s.Heroes[1].X, s.Heroes[1].Y = 1, 2
	s, ev = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	if s.Rules.Outcome != OutcomeWon {
		t.Fatalf("every living hero is on an exit: %+v %q", s.Rules, ev.Summary)
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
