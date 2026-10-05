package tracker

import (
	"reflect"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

func TestSeatViewShowsOnlyTheSeatsHeroes(t *testing.T) {
	s := rulesState(t)
	_, _, cat := fixture()
	view := SeatView(s, []string{"hero-2"}, cat)
	if view.Round != 1 || view.Phase != PhaseHeroes || len(view.Heroes) != 1 {
		t.Fatalf("seat %+v", view)
	}
	h := view.Heroes[0]
	if h.ID != "hero-2" || h.Name != "Ilsa" || h.ClassName != "Wizard" || h.Body != 4 || h.MaxBody != 4 || h.Movement != "1d6+2" || !h.Placed {
		t.Errorf("hero %+v", h)
	}
	if got := labels(h.Actions); !reflect.DeepEqual(got, []string{"Start Ilsa's turn"}) {
		t.Errorf("actions %v", got)
	}
	if empty := SeatView(s, nil, cat); len(empty.Heroes) != 0 {
		t.Errorf("no heroes: %+v", empty.Heroes)
	}
}

func TestSeatViewFollowsTheTurn(t *testing.T) {
	s := rulesState(t)
	s.Heroes[0].Abilities = []content.Ability{{ID: "charge", Name: "Charge", Kind: content.AbilityActive, Cooldown: 3}}
	s.Heroes[0].Cooldowns = map[string]int{"charge": 3}
	s = startTurn(t, s, "hero-1")
	_, _, cat := fixture()
	view := SeatView(s, []string{"hero-1", "hero-2"}, cat)
	if view.TurnHero != "hero-1" {
		t.Errorf("turn hero %q", view.TurnHero)
	}
	grom, ilsa := view.Heroes[0], view.Heroes[1]
	if grom.Turn == nil || grom.Turn.HeroID != "hero-1" || ilsa.Turn != nil {
		t.Errorf("turns: %+v %+v", grom.Turn, ilsa.Turn)
	}
	if len(grom.Abilities) != 1 || grom.Abilities[0].ReadyIn != 2 {
		t.Errorf("abilities %+v", grom.Abilities)
	}
	if len(ilsa.Actions) != 0 {
		t.Errorf("Ilsa waits: %v", labels(ilsa.Actions))
	}
	s, _ = apply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	if view := SeatView(s, []string{"hero-1"}, cat); !view.Heroes[0].Acted || view.TurnHero != "" {
		t.Errorf("after the turn: %+v", view.Heroes[0])
	}
}

func TestSeatViewInTableMode(t *testing.T) {
	_, _, cat := fixture()
	view := SeatView(newState(t), []string{"hero-1"}, cat)
	if view.Phase != "" || len(view.Heroes) != 1 || len(view.Heroes[0].Actions) != 0 {
		t.Errorf("table mode: %+v", view)
	}
}
