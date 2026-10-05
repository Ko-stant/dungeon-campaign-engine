package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
)

// heroJoin brings a campaign hero into a running session (a player joining
// online mid-game): on the first free start square, or off the board for
// the GM to place. Under the rules they get their movement too. The server
// sends it as the GM when a player joins with a new hero.
func (a *applier) heroJoin(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Hero CampaignHero `json:"hero"`
	}](payload)
	if err != nil {
		return "", err
	}
	if p.Hero.ID == "" {
		return "", errors.New("the hero needs an id")
	}
	if _, err := a.hero(p.Hero.ID); err == nil {
		return "", fmt.Errorf("%s is already in the game", p.Hero.Name)
	}
	if a.catalog == nil {
		return "", errors.New("no catalog")
	}
	h, err := sessionHero(p.Hero, a.catalog)
	if err != nil {
		return "", err
	}
	if a.s.Rules != nil {
		h.Movement = a.heroMovement(h.Class)
	}
	for _, t := range a.s.Quest.StartTiles {
		if a.heroAt(t, "") == nil && a.monsterAt(t, "") == nil {
			h.X, h.Y, h.Placed = t.X, t.Y, true
			break
		}
	}
	a.s.Heroes = append(a.s.Heroes, h)
	added := &a.s.Heroes[len(a.s.Heroes)-1]
	if !added.Placed {
		return a.heroLabel(added) + " joins the game (the GM places them)", nil
	}
	return fmt.Sprintf("%s joins the game at (%d,%d)", a.heroLabel(added), added.X, added.Y), nil
}
