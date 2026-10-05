package tracker

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

func joinCmd(t *testing.T, h CampaignHero) Command {
	return cmd(t, "hero.join", map[string]any{"hero": h})
}

func TestAHeroJoinsOnAFreeStartSquare(t *testing.T) {
	s := newState(t) // Grom and Ilsa on the first two of three start squares
	s, ev := apply(t, s, joinCmd(t, CampaignHero{ID: "hero-3", Name: "Vex", Player: "Sam", Class: "wizard", UserID: "user-sam"}))
	h := s.Heroes[2]
	if h.ID != "hero-3" || !h.Placed || h.X != 1 || h.Y != 3 || h.Body != 4 || h.Status != HeroActive {
		t.Fatalf("hero %+v", h)
	}
	if ev.Summary != "Vex (Wizard) joins the game at (1,3)" {
		t.Errorf("summary %q", ev.Summary)
	}
	if msg := applyErr(t, s, joinCmd(t, CampaignHero{ID: "hero-3", Name: "Vex", Class: "wizard"})); !strings.Contains(msg, "already") {
		t.Errorf("joining twice: %q", msg)
	}
}

func TestAHeroWithoutAFreeStartSquareWaitsOffTheBoard(t *testing.T) {
	s := newState(t)
	s.Quest.StartTiles = []maps.Tile{{X: 1, Y: 4}}
	s, ev := apply(t, s, joinCmd(t, CampaignHero{ID: "hero-3", Name: "Vex", Class: "wizard"}))
	if s.Heroes[2].Placed || ev.Summary != "Vex (Wizard) joins the game (the GM places them)" {
		t.Fatalf("hero %+v, %q", s.Heroes[2], ev.Summary)
	}
}

func TestAHeroJoiningUnderTheRulesGetsTheirMovement(t *testing.T) {
	s := rulesState(t)
	s, _ = apply(t, s, joinCmd(t, CampaignHero{ID: "hero-3", Name: "Vex", Class: "wizard"}))
	if s.Heroes[2].Movement != "1d6+2" {
		t.Errorf("movement %q", s.Heroes[2].Movement)
	}
	if msg := applyErr(t, s, joinCmd(t, CampaignHero{ID: "hero-4", Name: "Nobody", Class: "bard"})); !strings.Contains(msg, "bard") {
		t.Errorf("an unknown class: %q", msg)
	}
	if msg := applyErr(t, s, as(seat("hero-1"), joinCmd(t, CampaignHero{ID: "hero-5", Name: "X", Class: "wizard"}))); !strings.Contains(msg, "GM") {
		t.Errorf("from a seat: %q", msg)
	}
}
