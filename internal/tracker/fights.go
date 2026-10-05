package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Fights (The Three Plagues rules, see docs/campaigns/three-plagues/RULES_AND_CLASSES.md):
// in a fight, each new round finishes cooldowns, regenerates mana (a class's
// mana per fight round) and counts effects down. Out of a fight, cooldowns keep
// counting down but stop at 1 round left (cooldowns of 1-3) or 2 (4 or more),
// so stalling between fights never fully resets an ability; no mana comes back
// and effects wait. Nothing here enforces a rule: the GM can start or end a
// fight, reset cooldowns or change mana at any time.

// Effect is a named condition on a hero or monster ("Poisoned", "Raging",
// "Marked"), with an optional countdown in fight rounds and a note. The
// tracker only counts it down and reminds; the GM applies what it does.
type Effect struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Rounds left, counted down at each new fight round; 0 lasts until removed.
	Rounds int    `json:"rounds,omitempty"`
	Note   string `json:"note,omitempty"`
}

const (
	maxEffectName   = 40
	maxEffectNote   = 200
	maxEffectRounds = 99
)

func (a *applier) fightStart(payload json.RawMessage) (string, error) {
	if _, err := decode[struct{}](payload); err != nil {
		return "", err
	}
	if a.s.Fight {
		return "", errors.New("a fight is already on")
	}
	a.s.Fight = true
	return fmt.Sprintf("Fight started in round %d", a.s.Round), nil
}

// fightEnd ends the fight and the effects that had a countdown (they belong
// to the fight); effects without one stay until removed.
func (a *applier) fightEnd(payload json.RawMessage) (string, error) {
	if _, err := decode[struct{}](payload); err != nil {
		return "", err
	}
	if !a.s.Fight {
		return "", errors.New("there is no fight to end")
	}
	a.s.Fight = false
	summary := "Fight over"
	if ended := a.removeEffects(func(e *Effect) bool { return e.Rounds > 0 }); len(ended) > 0 {
		summary += "; ended: " + strings.Join(ended, ", ")
	}
	return summary, nil
}

// cooldownFloor is how many rounds an ability's cooldown keeps out of a fight.
func cooldownFloor(cooldown int) int {
	if cooldown <= 3 {
		return 1
	}
	return 2
}

// holdCooldowns runs after a round passes out of a fight: each cooldown counts
// down as usual but stops at its floor, and one already below the floor stays.
func (a *applier) holdCooldowns() {
	for i := range a.s.Heroes {
		h := &a.s.Heroes[i]
		for id, ready := range h.Cooldowns {
			before := ready - (a.s.Round - 1)
			if before <= 0 {
				continue
			}
			floor := cooldownFloor(4)
			if ab, err := h.ability(id); err == nil {
				floor = cooldownFloor(ab.Cooldown)
			}
			h.Cooldowns[id] = a.s.Round + max(before-1, min(before, floor))
		}
	}
}

// regenerateMana gives each living hero their mana per fight round (class and
// equipped items), up to their maximum, and names the changes ("Mira 10 → 13").
func (a *applier) regenerateMana() []string {
	var out []string
	for i := range a.s.Heroes {
		h := &a.s.Heroes[i]
		regen, limit := h.ManaRegen(), h.ManaCap()
		if h.Status == HeroDead || regen <= 0 || h.Mana >= limit {
			continue
		}
		before := h.Mana
		h.Mana = min(limit, h.Mana+regen)
		out = append(out, fmt.Sprintf("%s %d → %d", h.Name, before, h.Mana))
	}
	return out
}

// effectHolder is one hero's or monster's effects, with how to name it.
type effectHolder struct {
	label   string
	effects *[]Effect
}

func (a *applier) effectHolders() []effectHolder {
	var out []effectHolder
	for i := range a.s.Heroes {
		out = append(out, effectHolder{a.s.Heroes[i].Name, &a.s.Heroes[i].Effects})
	}
	for i := range a.s.Monsters {
		out = append(out, effectHolder{monsterLabel(&a.s.Monsters[i]), &a.s.Monsters[i].Effects})
	}
	return out
}

// removeEffects drops every effect matching drop and names them ("Grom's
// Raging"), heroes first, then monsters.
func (a *applier) removeEffects(drop func(e *Effect) bool) []string {
	var out []string
	for _, holder := range a.effectHolders() {
		kept := (*holder.effects)[:0]
		for _, e := range *holder.effects {
			if drop(&e) {
				out = append(out, holder.label+"'s "+e.Name)
				continue
			}
			kept = append(kept, e)
		}
		*holder.effects = kept
		if len(kept) == 0 {
			*holder.effects = nil
		}
	}
	return out
}

// tickEffects counts every countdown down by one round and names the effects
// that ran out.
func (a *applier) tickEffects() []string {
	return a.removeEffects(func(e *Effect) bool {
		if e.Rounds <= 0 {
			return false
		}
		e.Rounds--
		return e.Rounds == 0
	})
}

// effectTarget finds the hero or monster an effect command names.
func (a *applier) effectTarget(id string) (effectHolder, error) {
	for i := range a.s.Heroes {
		if a.s.Heroes[i].ID == id {
			return effectHolder{a.s.Heroes[i].Name, &a.s.Heroes[i].Effects}, nil
		}
	}
	for i := range a.s.Monsters {
		if a.s.Monsters[i].ID == id {
			return effectHolder{monsterLabel(&a.s.Monsters[i]), &a.s.Monsters[i].Effects}, nil
		}
	}
	return effectHolder{}, fmt.Errorf("no hero or monster %q", id)
}

func (a *applier) effectAdd(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Target string `json:"target"`
		Name   string `json:"name"`
		Rounds int    `json:"rounds"`
		Note   string `json:"note"`
	}](payload)
	if err != nil {
		return "", err
	}
	holder, err := a.effectTarget(p.Target)
	if err != nil {
		return "", err
	}
	name, note := strings.TrimSpace(p.Name), strings.TrimSpace(p.Note)
	switch {
	case name == "":
		return "", errors.New("an effect needs a name")
	case len(name) > maxEffectName:
		return "", fmt.Errorf("an effect name must be at most %d characters", maxEffectName)
	case len(note) > maxEffectNote:
		return "", fmt.Errorf("an effect note must be at most %d characters", maxEffectNote)
	case p.Rounds < 0 || p.Rounds > maxEffectRounds:
		return "", fmt.Errorf("rounds must be from 0 to %d", maxEffectRounds)
	}
	highest := 0
	for _, e := range *holder.effects {
		var n int
		if _, err := fmt.Sscanf(e.ID, "effect-%d", &n); err == nil && n > highest {
			highest = n
		}
	}
	e := Effect{ID: fmt.Sprintf("effect-%d", highest+1), Name: name, Rounds: p.Rounds, Note: note}
	*holder.effects = append(*holder.effects, e)
	summary := holder.label + ": " + name
	if e.Rounds > 0 {
		summary += " (" + plural(e.Rounds, "round") + ")"
	}
	if note != "" {
		summary += ": " + note
	}
	return summary, nil
}

func (a *applier) effectRemove(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Target string `json:"target"`
		ID     string `json:"id"`
	}](payload)
	if err != nil {
		return "", err
	}
	holder, err := a.effectTarget(p.Target)
	if err != nil {
		return "", err
	}
	for i, e := range *holder.effects {
		if e.ID == p.ID {
			*holder.effects = append((*holder.effects)[:i], (*holder.effects)[i+1:]...)
			if len(*holder.effects) == 0 {
				*holder.effects = nil
			}
			return holder.label + ": " + e.Name + " removed", nil
		}
	}
	return "", fmt.Errorf("%s has no effect %q", holder.label, p.ID)
}
