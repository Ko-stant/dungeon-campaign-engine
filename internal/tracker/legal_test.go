package tracker

import (
	"encoding/json"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

func labels(actions []Action) []string {
	out := []string{}
	for _, a := range actions {
		out = append(out, a.Label)
	}
	return out
}

func hallLegal(s *State, actor Actor) []Action {
	_, _, cat := hallBoard()
	return LegalActions(s, actor, cat)
}

// A player's buttons name pieces as the players see them, with the way to
// them from the hero, never by the GM's ids.
func TestHeroActionsNamePiecesAsThePlayersSeeThem(t *testing.T) {
	has := func(actions []Action, label string) bool {
		return slices.Contains(labels(actions), label)
	}
	s := searchState(t, nil)
	if got := hallLegal(s, seat("hero-1")); !has(got, "Search the Chest to the east for treasure") {
		t.Errorf("Grom beside the chest: %v", labels(got))
	}
	s = rogueState(t)
	if got := hallLegal(s, seat("hero-2")); !has(got, "Disarm the Pit Trap to the north") || !has(got, "Open the door to the east") {
		t.Errorf("Ilsa beside the pit and door d2: %v", labels(got))
	}
	s = searchState(t, func(q *maps.Quest) {
		q.Doors[1].Kind, q.Doors[1].Locked, q.Doors[1].Key = maps.DoorGate, true, "Iron Key"
		q.StartTiles[0] = maps.Tile{X: 4, Y: 2}
	})
	s.Heroes[0].Items = []Item{{Name: "Iron Key", Quantity: 1}}
	if got := hallLegal(s, seat("hero-1")); !has(got, "Unlock the gate to the west with the Iron Key") {
		t.Errorf("a locked gate: %v", labels(got))
	}
}

func TestLegalActionsOnAHeroTurn(t *testing.T) {
	s := duelState(t)
	want := []string{"Roll for movement", "Open the door to the west", "Attack the Orc to the northeast", "End the turn"}
	if got := labels(hallLegal(s, seat("hero-1"))); !reflect.DeepEqual(got, want) {
		t.Errorf("Grom's turn: %v, want %v", got, want)
	}
	if got := hallLegal(s, seat("hero-2")); len(got) != 0 {
		t.Errorf("not Ilsa's turn: %v", labels(got))
	}
	if got := hallLegal(s, Actor{}); len(got) != 0 {
		t.Errorf("the GM waits for the turn to end: %v", labels(got))
	}

	s, _ = hallApply(t, s, as(seat("hero-1"), cmd(t, "turn.end", map[string]any{})))
	if got := labels(hallLegal(s, seat("hero-2"))); !reflect.DeepEqual(got, []string{"Start Ilsa's turn"}) {
		t.Errorf("between turns: %v", got)
	}
	if got := labels(hallLegal(s, Actor{})); !reflect.DeepEqual(got, []string{"The monsters' turn", "End the round"}) {
		t.Errorf("the GM between turns: %v", got)
	}
}

func TestLegalActionsListEveryReachableSquare(t *testing.T) {
	s := hallTurn(t, 1, 1)
	var moves []string
	for _, a := range hallLegal(s, seat("hero-1")) {
		if a.Command.Type == "turn.move" {
			moves = append(moves, a.Label)
		}
	}
	// From (1,2) with 2 squares: room 1 only (doors closed), not onto Ilsa at (1,1).
	want := []string{"Move to (2,1)", "Move to (2,2)", "Move to (1,3)", "Move to (2,3)"}
	if !reflect.DeepEqual(moves, want) {
		t.Errorf("moves %v, want %v", moves, want)
	}
}

func TestLegalActionsInTheMonstersPhase(t *testing.T) {
	s := monstersPhase(t)
	s.Monsters[0].Movement = 1
	want := []string{"Orc (orc): move to (5,1)", "Orc (orc): move to (5,3)", "Orc (orc): attack Grom", "End the round"}
	if got := labels(hallLegal(s, Actor{})); !reflect.DeepEqual(got, want) {
		t.Errorf("monsters' phase: %v, want %v", got, want)
	}
}

func TestLegalActionsAfterTheEnd(t *testing.T) {
	s := rulesState(t)
	s.Rules.Phase = PhaseOver
	if got := LegalActions(s, seat("hero-1"), nil); got != nil {
		t.Errorf("over: %v", labels(got))
	}
	if got := LegalActions(newState(t), Actor{}, nil); got != nil {
		t.Errorf("table mode: %v", labels(got))
	}
}

// key identifies a command for comparing candidates with listed actions.
func key(c Command) string { return c.Type + " " + string(c.Payload) }

// candidates lists every command of the kinds LegalActions covers that the
// actor could try: each square, door, monster, piece of furniture, trap and
// hero. Payloads are built as LegalActions builds them.
func candidates(s *State, actor Actor) []Command {
	var out []Command
	add := func(kind string, payload map[string]any) {
		data, _ := json.Marshal(payload)
		out = append(out, Command{Type: kind, Payload: data, Actor: actor})
	}
	var squares []maps.Tile
	for y := 1; y <= s.Board.Height; y++ {
		for x := 1; x <= s.Board.Width; x++ {
			squares = append(squares, maps.Tile{X: x, Y: y})
		}
	}
	if actor.Player() {
		for _, h := range s.Heroes {
			add("turn.start", map[string]any{"hero": h.ID})
		}
		add("turn.roll-move", map[string]any{})
		for _, sq := range squares {
			add("turn.move", map[string]any{"to": tileJSON(sq)})
		}
		for _, d := range s.Doors {
			add("turn.door", map[string]any{"door": d.ID})
		}
		for _, m := range s.Monsters {
			add("turn.attack", map[string]any{"target": m.ID})
		}
		add("turn.search", map[string]any{"kind": SearchTraps})
		add("turn.search", map[string]any{"kind": SearchDoors})
		for _, f := range s.Quest.Furniture {
			add("turn.search", map[string]any{"kind": SearchTreasure, "furniture": f.ID})
		}
		for _, tr := range s.Traps {
			add("turn.disarm", map[string]any{"trap": tr.ID})
		}
		add("turn.exit", map[string]any{})
		add("turn.end", map[string]any{})
		return out
	}
	add("phase.monsters", map[string]any{})
	add("phase.end", map[string]any{})
	for _, m := range s.Monsters {
		for _, sq := range squares {
			add("monster.move", map[string]any{"monster": m.ID, "to": tileJSON(sq)})
		}
		for _, h := range s.Heroes {
			add("monster.attack", map[string]any{"monster": m.ID, "target": h.ID})
		}
	}
	return out
}

// checkLegal holds LegalActions to Apply for every actor: each listed action
// applies, each candidate not listed is refused, and listing changes nothing.
func checkLegal(t *testing.T, s *State, actors []Actor) []Action {
	t.Helper()
	_, _, cat := hallBoard()
	before, _ := json.Marshal(s)
	var all []Action
	for _, actor := range actors {
		listed := map[string]bool{}
		for _, a := range LegalActions(s, actor, cat) {
			listed[key(a.Command)] = true
			if _, _, err := Apply(s, a.Command, cat); err != nil {
				t.Fatalf("listed %q for %+v but: %v", a.Label, actor, err)
			}
			all = append(all, a)
		}
		for _, c := range candidates(s, actor) {
			if listed[key(c)] {
				continue
			}
			if _, ev, err := Apply(s, c, cat); err == nil {
				t.Fatalf("%s for %+v applies (%q) but is not listed", key(c), actor, ev.Summary)
			}
		}
	}
	if after, _ := json.Marshal(s); string(after) != string(before) {
		t.Fatal("LegalActions changed the state")
	}
	return all
}

// TestLegalActionsMatchApply plays random games from several positions,
// checking LegalActions against Apply at every step.
func TestLegalActionsMatchApply(t *testing.T) {
	_, _, cat := hallBoard()
	starts := map[string]func(*testing.T) *State{
		"duel":    duelState,
		"monster": monstersPhase,
		"search":  func(t *testing.T) *State { return searchState(t, nil) },
		"rogue":   rogueState,
		"exit":    exitState,
		"hall":    func(t *testing.T) *State { return hallTurn(t, 3, 4) },
	}
	for name, start := range starts {
		t.Run(name, func(t *testing.T) {
			for game := range 4 {
				r := rand.New(rand.NewPCG(uint64(game), 99))
				s := start(t)
				for range 40 {
					actors := []Actor{{}}
					for _, h := range s.Heroes {
						actors = append(actors, seat(h.ID))
					}
					actions := checkLegal(t, s, actors)
					if len(actions) == 0 {
						break
					}
					next, _, err := Apply(s, actions[r.IntN(len(actions))].Command, cat)
					if err != nil {
						t.Fatal(err)
					}
					s = next
				}
			}
		})
	}
}

// TestAGameReplaysFromItsSeed: the same commands from the same start give
// the same game, dice included (the seeded roller lives in the state).
func TestAGameReplaysFromItsSeed(t *testing.T) {
	_, _, cat := hallBoard()
	play := func(s *State, commands []Command) (*State, []string) {
		var summaries []string
		for _, c := range commands {
			next, ev, err := Apply(s, c, cat)
			if err != nil {
				t.Fatal(err)
			}
			s, summaries = next, append(summaries, ev.Summary)
		}
		return s, summaries
	}
	r := rand.New(rand.NewPCG(7, 7))
	s := duelState(t)
	var commands []Command
	for range 60 {
		var actions []Action
		for _, actor := range []Actor{{}, seat("hero-1"), seat("hero-2")} {
			actions = append(actions, LegalActions(s, actor, cat)...)
		}
		if len(actions) == 0 {
			break
		}
		c := actions[r.IntN(len(actions))].Command
		commands = append(commands, c)
		s, _ = play(s, []Command{c})
	}
	first, firstLog := play(duelState(t), commands)
	second, secondLog := play(duelState(t), commands)
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) || !reflect.DeepEqual(firstLog, secondLog) {
		t.Fatal("replaying the same commands gave a different game")
	}
}
