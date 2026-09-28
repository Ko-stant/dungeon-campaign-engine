package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

// CooldownLeft is how many rounds remain before the ability is ready in
// round; 0 means it is ready. Used in round R with cooldown C, an ability is
// ready again in round R+C.
func (h *Hero) CooldownLeft(abilityID string, round int) int {
	ready, ok := h.Cooldowns[abilityID]
	if !ok || ready <= round {
		return 0
	}
	return ready - round
}

func (h *Hero) ability(id string) (*content.Ability, error) {
	for i := range h.Abilities {
		if h.Abilities[i].ID == id {
			return &h.Abilities[i], nil
		}
	}
	return nil, fmt.Errorf("%s has no ability %q", h.Name, id)
}

func (a *applier) abilityUse(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID    string `json:"heroId"`
		AbilityID string `json:"abilityId"`
	}](payload)
	if err != nil {
		return "", err
	}
	h, err := a.hero(p.HeroID)
	if err != nil {
		return "", err
	}
	ab, err := h.ability(p.AbilityID)
	if err != nil {
		return "", err
	}

	verb := "used"
	if ab.Kind == content.AbilitySpell {
		verb = "cast"
	}
	summary := fmt.Sprintf("%s %s %s", h.Name, verb, ab.Name)
	// Nothing is refused: using an ability early or without the mana is
	// recorded with a note, and the GM decides.
	if left := h.CooldownLeft(ab.ID, a.s.Round); left > 0 {
		summary += fmt.Sprintf(" (still cooling down, %s left)", plural(left, "round"))
	}
	if ab.ManaCost > h.Mana {
		summary += fmt.Sprintf(" (needs %d mana, had %d)", ab.ManaCost, h.Mana)
	}

	var effects []string
	if ab.ManaCost > 0 {
		before := h.Mana
		h.Mana = max(h.Mana-ab.ManaCost, 0)
		effects = append(effects, fmt.Sprintf("mana %d → %d", before, h.Mana))
	}
	if ab.Cooldown > 0 {
		if h.Cooldowns == nil {
			h.Cooldowns = map[string]int{}
		}
		h.Cooldowns[ab.ID] = a.s.Round + ab.Cooldown
		effects = append(effects, fmt.Sprintf("ready again in round %d", a.s.Round+ab.Cooldown))
	}
	if len(effects) > 0 {
		summary += ": " + strings.Join(effects, ", ")
	}
	return summary, nil
}

// abilityReset ends one ability's cooldown, or every cooldown of the hero
// when no ability is given (Divine Blessing, or a GM correction).
func (a *applier) abilityReset(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID    string `json:"heroId"`
		AbilityID string `json:"abilityId"`
	}](payload)
	if err != nil {
		return "", err
	}
	h, err := a.hero(p.HeroID)
	if err != nil {
		return "", err
	}
	if p.AbilityID != "" {
		ab, err := h.ability(p.AbilityID)
		if err != nil {
			return "", err
		}
		if h.CooldownLeft(ab.ID, a.s.Round) == 0 {
			return "", fmt.Errorf("%s is already ready", ab.Name)
		}
		delete(h.Cooldowns, ab.ID)
		return fmt.Sprintf("%s: %s is ready again", h.Name, ab.Name), nil
	}
	var names []string
	for _, ab := range h.Abilities {
		if h.CooldownLeft(ab.ID, a.s.Round) > 0 {
			names = append(names, ab.Name)
		}
	}
	if len(names) == 0 {
		return "", errors.New("no ability is cooling down")
	}
	h.Cooldowns = nil
	return fmt.Sprintf("%s: all abilities are ready again (%s)", h.Name, strings.Join(names, ", ")), nil
}

// readyAgain drops cooldowns that have ended by the current round and names
// those abilities ("Vex's Multi-Shot"), in hero and ability order.
func (a *applier) readyAgain() []string {
	var out []string
	for i := range a.s.Heroes {
		h := &a.s.Heroes[i]
		if len(h.Cooldowns) == 0 {
			continue
		}
		for _, ab := range h.Abilities {
			if ready, ok := h.Cooldowns[ab.ID]; ok && ready <= a.s.Round {
				out = append(out, fmt.Sprintf("%s's %s", h.Name, ab.Name))
			}
		}
		for id, ready := range h.Cooldowns {
			if ready <= a.s.Round {
				delete(h.Cooldowns, id)
			}
		}
		if len(h.Cooldowns) == 0 {
			h.Cooldowns = nil
		}
	}
	return out
}

func (a *applier) roundAdvance() string {
	a.s.Round++
	summary := fmt.Sprintf("Round %d begins", a.s.Round)
	if ready := a.readyAgain(); len(ready) > 0 {
		summary += "; ready again: " + strings.Join(ready, ", ")
	}
	return summary
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
