package tracker

import (
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

// A seat is one player's place at an online game: their heroes' own sheets
// and what each may do now. What everyone sees (the board, the party, the
// feed) is the PlayerView; a seat adds only the player's own heroes, and the
// legal actions never reveal anything hidden (see LegalActions).

// SeatState is a player's seat.
type SeatState struct {
	Round int    `json:"round"`
	Phase string `json:"phase,omitempty"`
	// TurnHero is the hero whose turn is under way, if any (any player's).
	TurnHero string     `json:"turnHero,omitempty"`
	Outcome  string     `json:"outcome,omitempty"`
	Heroes   []SeatHero `json:"heroes"`
}

// SeatHero is one of the seat's heroes, with their whole sheet.
type SeatHero struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ClassName     string  `json:"className"`
	Placed        bool    `json:"placed"`
	X             int     `json:"x"`
	Y             int     `json:"y"`
	Status        string  `json:"status"`
	Body          int     `json:"body"`
	MaxBody       int     `json:"maxBody"`
	Mind          int     `json:"mind"`
	MaxMind       int     `json:"maxMind"`
	Mana          int     `json:"mana,omitempty"`
	MaxMana       int     `json:"maxMana,omitempty"`
	Movement      string  `json:"movement,omitempty"`
	Determination int     `json:"determination,omitempty"`
	Combat        *Combat `json:"combat,omitempty"`
	// Abilities with the rounds left before each is ready again.
	Abilities []SeatAbility `json:"abilities,omitempty"`
	Items     []Item        `json:"items"`
	Effects   []Effect      `json:"effects,omitempty"`
	// Turn is this hero's turn, while it is under way.
	Turn *Turn `json:"turn,omitempty"`
	// Acted: this hero's turn this round is over. SkipNext: they lose their
	// next turn (a critical miss).
	Acted    bool     `json:"acted,omitempty"`
	SkipNext bool     `json:"skipNext,omitempty"`
	Actions  []Action `json:"actions"`
}

// SeatAbility is an ability on a hero's sheet.
type SeatAbility struct {
	content.Ability
	ReadyIn int `json:"readyIn,omitempty"`
}

// SeatView builds the seat of the player who plays the given heroes.
func SeatView(s *State, heroIDs []string, catalog *content.Catalog) SeatState {
	out := SeatState{Round: s.Round, Heroes: []SeatHero{}}
	r := s.Rules
	if r != nil {
		out.Phase, out.Outcome = r.Phase, r.Outcome
		if r.Turn != nil {
			out.TurnHero = r.Turn.HeroID
		}
	}
	for i := range s.Heroes {
		h := &s.Heroes[i]
		if !slices.Contains(heroIDs, h.ID) {
			continue
		}
		sh := SeatHero{
			ID: h.ID, Name: h.Name, ClassName: h.Class, Placed: h.Placed, X: h.X, Y: h.Y, Status: h.Status,
			Body: h.Body, MaxBody: h.MaxBody, Mind: h.Mind, MaxMind: h.MaxMind, Mana: h.Mana, MaxMana: h.ManaCap(),
			Movement: h.Movement, Determination: h.Determination, Combat: h.CombatTotals(),
			Items: slices.Clone(h.Items), Effects: slices.Clone(h.Effects),
		}
		if catalog != nil {
			if def, ok := catalog.Hero(h.Class); ok {
				sh.ClassName = def.Name
			}
		}
		for _, ab := range h.Abilities {
			sh.Abilities = append(sh.Abilities, SeatAbility{Ability: ab, ReadyIn: h.CooldownLeft(ab.ID, s.Round)})
		}
		if r != nil {
			if r.Turn != nil && r.Turn.HeroID == h.ID {
				turn := *r.Turn
				sh.Turn = &turn
			}
			sh.Acted = slices.Contains(r.Acted, h.ID)
			sh.SkipNext = slices.Contains(r.SkipNext, h.ID)
		}
		sh.Actions = LegalActions(s, Actor{Kind: ActorSeat, HeroID: h.ID}, catalog)
		if sh.Actions == nil {
			sh.Actions = []Action{}
		}
		if sh.Items == nil {
			sh.Items = []Item{}
		}
		out.Heroes = append(out.Heroes, sh)
	}
	return out
}
