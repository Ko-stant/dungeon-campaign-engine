package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// Quest outcomes in rules mode: the quest is won when every one of its
// objectives is met, or, without objectives, once every monster is dead. It
// is lost when no hero is left standing, or when the heroes have all left
// before it was won. The GM can end it either way.

// Outcomes.
const (
	OutcomeWon  = "won"
	OutcomeLost = "lost"
)

// questOutcome is the outcome the quest has reached, or "".
func (s *State) questOutcome() string {
	standing := slices.ContainsFunc(s.Heroes, func(h Hero) bool { return h.Status == HeroActive })
	won := true
	if len(s.Quest.Objectives) == 0 {
		// Clearing every monster completes a quest without objectives (one
		// with no monsters at all ends only by the GM).
		won = len(s.Monsters) > 0 && !slices.ContainsFunc(s.Monsters, func(m Monster) bool { return m.Alive })
	}
	for _, o := range s.Quest.Objectives {
		won = won && s.objectiveMet(o)
	}
	switch {
	case won && (standing || slices.ContainsFunc(s.Heroes, func(h Hero) bool { return h.Status == HeroEscaped })):
		return OutcomeWon
	case !standing:
		return OutcomeLost
	}
	return ""
}

func (s *State) objectiveMet(o maps.Objective) bool {
	switch o.Kind {
	case maps.ObjectiveKill:
		return !slices.ContainsFunc(s.Monsters, func(m Monster) bool {
			return m.Alive && (len(o.Monsters) == 0 || slices.Contains(o.Monsters, m.ID))
		})
	case maps.ObjectiveCollect:
		return slices.ContainsFunc(s.Heroes, func(h Hero) bool { return h.Status != HeroDead && carried(&h, o.Item) != nil })
	case maps.ObjectiveEscape:
		return !slices.ContainsFunc(s.Heroes, func(h Hero) bool { return h.Status == HeroActive }) &&
			slices.ContainsFunc(s.Heroes, func(h Hero) bool { return h.Status == HeroEscaped })
	}
	return false
}

// turnExit takes a hero standing on an exit square off the board: they have
// escaped, and their turn is over.
func (a *applier) turnExit(payload json.RawMessage) (string, error) {
	if _, err := decode[struct{}](payload); err != nil {
		return "", err
	}
	_, h, err := a.activeTurn()
	if err != nil {
		return "", err
	}
	if !slices.Contains(a.s.Quest.ExitTiles, maps.Tile{X: h.X, Y: h.Y}) {
		return "", fmt.Errorf("%s is not on an exit square", h.Name)
	}
	h.Status = HeroEscaped
	a.s.Rules.Acted = append(a.s.Rules.Acted, h.ID)
	a.s.Rules.Turn = nil
	return fmt.Sprintf("%s leaves by the exit at (%d,%d)", a.heroLabel(h), h.X, h.Y), nil
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
