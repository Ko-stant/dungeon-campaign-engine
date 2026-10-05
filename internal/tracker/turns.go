package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/dice"
)

// Turns in rules mode: in the heroes' phase each living hero takes one turn,
// in any order, one at a time; then the monsters' phase (the GM's, or the AI
// GM's); then a new round.

// rules returns the session's rules state, or an error when they are off.
func (a *applier) rules() (*RulesState, error) {
	if a.s.Rules == nil {
		return nil, errors.New("the rules are off")
	}
	return a.s.Rules, nil
}

// actsFor checks that a player acts for their own hero; the GM acts for any.
func (a *applier) actsFor(heroID string) error {
	if a.actor.Player() && a.actor.HeroID != heroID {
		return errors.New("you may act only for your own hero")
	}
	return nil
}

// activeTurn returns the turn under way and its hero, checking the actor
// plays that hero.
func (a *applier) activeTurn() (*Turn, *Hero, error) {
	r, err := a.rules()
	if err != nil {
		return nil, nil, err
	}
	if r.Turn == nil {
		return nil, nil, errors.New("no hero's turn is under way")
	}
	if err := a.actsFor(r.Turn.HeroID); err != nil {
		return nil, nil, err
	}
	h, err := a.hero(r.Turn.HeroID)
	if err != nil {
		return nil, nil, err
	}
	return r.Turn, h, nil
}

func (a *applier) turnStart(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Hero string `json:"hero"`
	}](payload)
	if err != nil {
		return "", err
	}
	r, err := a.rules()
	if err != nil {
		return "", err
	}
	if err := a.actsFor(p.Hero); err != nil {
		return "", err
	}
	h, err := a.hero(p.Hero)
	if err != nil {
		return "", err
	}
	switch {
	case r.Phase == PhaseOver:
		return "", errors.New("the quest is over")
	case r.Phase != PhaseHeroes:
		return "", fmt.Errorf("it is the %s' phase, not the heroes'", r.Phase)
	case r.Turn != nil:
		other, _ := a.hero(r.Turn.HeroID)
		return "", fmt.Errorf("%s's turn is under way", other.Name)
	case slices.Contains(r.Acted, h.ID):
		return "", fmt.Errorf("%s has already had a turn this round", h.Name)
	case h.Status != HeroActive:
		return "", fmt.Errorf("%s is %s", h.Name, h.Status)
	case !h.Placed:
		return "", fmt.Errorf("%s is not on the board", h.Name)
	}
	if i := slices.Index(r.SkipNext, h.ID); i >= 0 {
		r.SkipNext = slices.Delete(r.SkipNext, i, i+1)
		r.Acted = append(r.Acted, h.ID)
		return a.heroLabel(h) + " loses this turn (critical miss)", nil
	}
	r.Turn = &Turn{HeroID: h.ID}
	return a.heroLabel(h) + " starts their turn", nil
}

func (a *applier) turnEnd(payload json.RawMessage) (string, error) {
	if _, err := decode[struct{}](payload); err != nil {
		return "", err
	}
	_, h, err := a.activeTurn()
	if err != nil {
		return "", err
	}
	r := a.s.Rules
	r.Acted = append(r.Acted, h.ID)
	r.Turn = nil
	return a.heroLabel(h) + " ends their turn", nil
}

// waiting names the living heroes who have not acted this round.
func (a *applier) waiting() []string {
	var out []string
	for _, h := range a.s.Heroes {
		if h.Status == HeroActive && h.Placed && !slices.Contains(a.s.Rules.Acted, h.ID) {
			out = append(out, h.Name)
		}
	}
	return out
}

func (a *applier) phaseMonsters(payload json.RawMessage) (string, error) {
	if _, err := decode[struct{}](payload); err != nil {
		return "", err
	}
	r, err := a.rules()
	if err != nil {
		return "", err
	}
	if r.Phase != PhaseHeroes {
		return "", fmt.Errorf("it is the %s' phase, not the heroes'", r.Phase)
	}
	if r.Turn != nil {
		h, _ := a.hero(r.Turn.HeroID)
		return "", fmt.Errorf("%s's turn is still under way", h.Name)
	}
	r.Phase = PhaseMonsters
	summary := "The monsters' turn"
	if w := a.waiting(); len(w) > 0 {
		heroes := "1 hero"
		if len(w) > 1 {
			heroes = fmt.Sprintf("%d heroes", len(w))
		}
		summary += fmt.Sprintf(" (%s did not act: %s)", heroes, strings.Join(w, ", "))
	}
	return summary, nil
}

func (a *applier) phaseEnd(payload json.RawMessage) (string, error) {
	if _, err := decode[struct{}](payload); err != nil {
		return "", err
	}
	r, err := a.rules()
	if err != nil {
		return "", err
	}
	if r.Phase == PhaseOver {
		return "", errors.New("the quest is over")
	}
	if r.Turn != nil {
		h, _ := a.hero(r.Turn.HeroID)
		return "", fmt.Errorf("%s's turn is still under way", h.Name)
	}
	summary := a.roundAdvance()
	r.Phase, r.Acted, r.MonstersMoved, r.MonstersActed = PhaseHeroes, nil, nil, nil
	// "Round 2 begins; ready again: ..." becomes "Round 2 begins: the heroes' turns; ready again: ...".
	head, rest, _ := strings.Cut(summary, ";")
	summary = head + ": the heroes' turns"
	if rest != "" {
		summary += ";" + rest
	}
	return summary, nil
}

// turnRollMove rolls the hero's movement dice for this turn (online rules D1).
func (a *applier) turnRollMove(payload json.RawMessage) (string, error) {
	if _, err := decode[struct{}](payload); err != nil {
		return "", err
	}
	turn, h, err := a.activeTurn()
	if err != nil {
		return "", err
	}
	switch {
	case turn.MoveRoll > 0:
		return "", fmt.Errorf("%s has already rolled %d for movement", h.Name, turn.MoveRoll)
	case turn.MoveDone:
		return "", fmt.Errorf("%s's movement is over this turn", h.Name)
	}
	movement := h.Movement
	if movement == "" {
		movement = "2d6"
	}
	expr, err := dice.Parse(movement)
	if err != nil {
		return "", fmt.Errorf("%s's movement %q: %w", h.Name, movement, err)
	}
	r, err := a.roller()
	if err != nil {
		return "", err
	}
	total, rolls := expr.Roll(r.Die)
	total = max(total, 0)
	turn.MoveRoll, turn.MoveLeft = max(total, 1), total
	values := make([]string, len(rolls))
	for i, d := range rolls {
		values[i] = strconv.Itoa(d.Value)
	}
	return fmt.Sprintf("%s rolls %d for movement (%s: %s)", a.heroLabel(h), total, expr, strings.Join(values, ", ")), nil
}
