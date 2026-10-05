package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/combat"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

// Rules mode (docs/ONLINE_AND_RULES_PLAN.md, Phase 2): a session can switch
// on the online rules (docs/campaigns/three-plagues/ONLINE_RULES.md). Heroes
// then play through turn commands, which follow the rules and roll the dice
// from the session's seeded roller. Players (seats and bots) may send only
// those; the GM keeps every command, so anything can still be put right by
// hand. With the rules off (State.Rules nil) the tracker is the table
// companion it always was.

// RulesetThreePlagues is the only ruleset so far.
const RulesetThreePlagues = "three-plagues/1"

// Phases of a round in rules mode.
const (
	PhaseHeroes   = "heroes"
	PhaseMonsters = "monsters"
	PhaseOver     = "over"
)

// RulesState is the rules engine's part of a session.
type RulesState struct {
	Ruleset string `json:"ruleset"`
	// RNG is the seeded roller's state (combat.Mulberry32): every die the
	// engine throws comes from it, so a game replays exactly from its seed.
	RNG   uint32 `json:"rng"`
	Phase string `json:"phase"`
	// Acted lists the heroes whose turn this round is over.
	Acted []string `json:"acted,omitempty"`
	// Turn is the hero turn under way, if any; one at a time.
	Turn *Turn `json:"turn,omitempty"`
	// SkipNext lists heroes who lose their next turn (a critical miss).
	SkipNext []string `json:"skipNext,omitempty"`
}

// Turn is one hero's turn: Move then Action, or Action then Move (online
// rules D4).
type Turn struct {
	HeroID string `json:"heroId"`
	// MoveRoll is the movement rolled this turn; 0 until rolled.
	MoveRoll int `json:"moveRoll,omitempty"`
	// MoveLeft is how many squares the hero may still move.
	MoveLeft int `json:"moveLeft,omitempty"`
	// Acted: the hero has taken their action.
	Acted bool `json:"acted,omitempty"`
	// MoveDone: the movement step is over (the hero acted after moving, or a
	// trap stopped them).
	MoveDone bool `json:"moveDone,omitempty"`
}

// Actor kinds. The zero Actor is the GM.
const (
	ActorGM     = "gm"
	ActorSeat   = "seat"
	ActorBot    = "bot"
	ActorAutoGM = "autogm"
)

// Actor is who sent a command. The server sets it from the endpoint used,
// never from the client.
type Actor struct {
	Kind string `json:"kind,omitempty"`
	// HeroID is the hero a seat or bot plays.
	HeroID string `json:"heroId,omitempty"`
}

// Player reports whether the actor plays a hero (a seat or a bot) rather
// than running the game.
func (a Actor) Player() bool {
	return a.Kind == ActorSeat || a.Kind == ActorBot
}

func (a Actor) validate() error {
	switch a.Kind {
	case "", ActorGM, ActorAutoGM:
		return nil
	case ActorSeat, ActorBot:
		if a.HeroID == "" {
			return fmt.Errorf("a %s must play a hero", a.Kind)
		}
		return nil
	}
	return fmt.Errorf("unknown actor %q", a.Kind)
}

// playerCommands are the commands a seat or bot may send.
var playerCommands = map[string]bool{
	"turn.start":     true,
	"turn.roll-move": true,
	"turn.move":      true,
	"turn.door":      true,
	"turn.end":       true,
}

// rulesCommands change the game under the rules; a fight starts or ends after
// them as monsters are revealed or killed.
func rulesCommand(kind string) bool {
	return strings.HasPrefix(kind, "turn.") || strings.HasPrefix(kind, "phase.") || kind == "rules.enable"
}

// checkActor refuses what the actor may not send at all.
func checkActor(s *State, c Command) error {
	if err := c.Actor.validate(); err != nil {
		return err
	}
	if !c.Actor.Player() {
		return nil
	}
	if s.Rules == nil {
		return errors.New("the rules are off: players act through the GM")
	}
	if !playerCommands[c.Type] {
		return fmt.Errorf("%s is the GM's command", c.Type)
	}
	if len(c.Dice) > 0 {
		return errors.New("only the GM gives dice")
	}
	return nil
}

// diceSource is where a command's dice come from: the GM's own dice when given,
// otherwise the session's seeded roller. Every die is logged for the event.
type diceSource struct {
	log    combat.Log
	seeded *combat.Mulberry32
	given  *combat.Script
}

func newDice(s *State, given []int) *diceSource {
	d := &diceSource{}
	switch {
	case len(given) > 0:
		d.given = &combat.Script{Dice: given}
		d.log.Roller = d.given
	case s.Rules != nil:
		d.seeded = &combat.Mulberry32{State: s.Rules.RNG}
		d.log.Roller = d.seeded
	}
	return d
}

// roller returns the roller for this command, or an error when nothing can
// roll (the rules are off and the GM gave no dice).
func (a *applier) roller() (combat.Roller, error) {
	if a.dice.log.Roller == nil {
		return nil, errors.New("the rules are off and no dice were given")
	}
	return &a.dice.log, nil
}

// finish checks given dice were all used and saves the roller's state.
func (d *diceSource) finish(s *State) error {
	if d.given != nil {
		if err := d.given.Err(); err != nil {
			return err
		}
		if n := d.given.Left(); n > 0 {
			return fmt.Errorf("%d of the given dice were not used", n)
		}
	}
	if d.seeded != nil && s.Rules != nil {
		s.Rules.RNG = d.seeded.State
	}
	return nil
}

func (a *applier) rulesEnable(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Seed uint32 `json:"seed"`
	}](payload)
	if err != nil {
		return "", err
	}
	if a.s.Rules != nil {
		return "", errors.New("the rules are already on")
	}
	a.s.Rules = &RulesState{Ruleset: RulesetThreePlagues, RNG: p.Seed, Phase: PhaseHeroes}
	for i := range a.s.Heroes {
		h := &a.s.Heroes[i]
		h.Movement = a.heroMovement(h.Class)
	}
	for i := range a.s.Monsters {
		m := &a.s.Monsters[i]
		if a.catalog == nil {
			break
		}
		if def, ok := a.catalog.Monster(m.Type); ok {
			m.Movement = def.Movement
		}
	}
	return fmt.Sprintf("Rules on (%s): the heroes' turns%s", RulesetThreePlagues, a.revealFromHeroes()), nil
}

// heroMovement is a class's movement dice: its own expression, the original
// game's number of d6, or 2d6 (online rules D1).
func (a *applier) heroMovement(class string) string {
	var def content.HeroDef
	ok := false
	if a.catalog != nil {
		def, ok = a.catalog.Hero(class)
	}
	switch {
	case ok && def.Movement != "":
		return def.Movement
	case ok && def.MovementDice > 0:
		return fmt.Sprintf("%dd6", def.MovementDice)
	}
	return "2d6"
}

func (a *applier) rulesDisable(payload json.RawMessage) (string, error) {
	if _, err := decode[struct{}](payload); err != nil {
		return "", err
	}
	if a.s.Rules == nil {
		return "", errors.New("the rules are already off")
	}
	a.s.Rules = nil
	return "Rules off: the GM records the game", nil
}

// revealedMonster reports whether a living monster is revealed: combat is on.
func (s *State) revealedMonster() bool {
	return slices.ContainsFunc(s.Monsters, func(m Monster) bool { return m.Alive && m.Visibility == MonsterSeen })
}

// syncFight starts a fight when a living monster is revealed and ends it when
// none is left; it describes the change, or returns "".
func (a *applier) syncFight() string {
	switch on := a.s.revealedMonster(); {
	case on && !a.s.Fight:
		a.s.Fight = true
		return "; fight started"
	case !on && a.s.Fight:
		end := a.endFight()
		return "; " + strings.ToLower(end[:1]) + end[1:]
	}
	return ""
}
