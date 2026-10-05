package tracker

import (
	"encoding/json"
	"errors"
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// Quest outcomes in rules mode: the quest is lost when no hero is left
// standing, and won when every one of its objectives is met (a quest without
// objectives is won when the GM says so). The GM can end it either way.

// Outcomes.
const (
	OutcomeWon  = "won"
	OutcomeLost = "lost"
)

// questOutcome is the outcome the quest has reached, or "".
func (s *State) questOutcome() string {
	standing := slices.ContainsFunc(s.Heroes, func(h Hero) bool { return h.Status == HeroActive })
	escaped := slices.ContainsFunc(s.Heroes, func(h Hero) bool { return h.Status == HeroEscaped })
	if !standing && !escaped {
		return OutcomeLost
	}
	if len(s.Quest.Objectives) == 0 {
		return ""
	}
	for _, o := range s.Quest.Objectives {
		if !s.objectiveMet(o) {
			return ""
		}
	}
	return OutcomeWon
}

func (s *State) objectiveMet(o maps.Objective) bool {
	switch o.Kind {
	case maps.ObjectiveKill:
		return !slices.ContainsFunc(s.Monsters, func(m Monster) bool {
			return m.Alive && (len(o.Monsters) == 0 || slices.Contains(o.Monsters, m.ID))
		})
	case maps.ObjectiveEscape:
		exits := s.Quest.ExitTiles
		return !slices.ContainsFunc(s.Heroes, func(h Hero) bool {
			return h.Status == HeroActive && !slices.Contains(exits, maps.Tile{X: h.X, Y: h.Y})
		})
	}
	return false
}

// checkOutcome ends the quest once it is won or lost and describes it.
func (a *applier) checkOutcome() string {
	r := a.s.Rules
	if r == nil || r.Phase == PhaseOver {
		return ""
	}
	outcome := a.s.questOutcome()
	if outcome == "" {
		return ""
	}
	r.Phase, r.Outcome, r.Turn = PhaseOver, outcome, nil
	return "; the quest is " + outcome
}

func (a *applier) questEnd(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Outcome string `json:"outcome"`
	}](payload)
	if err != nil {
		return "", err
	}
	r, err := a.rules()
	if err != nil {
		return "", err
	}
	if p.Outcome != OutcomeWon && p.Outcome != OutcomeLost {
		return "", errors.New("the outcome must be won or lost")
	}
	r.Phase, r.Outcome, r.Turn = PhaseOver, p.Outcome, nil
	return "The GM ends the quest: " + p.Outcome, nil
}
