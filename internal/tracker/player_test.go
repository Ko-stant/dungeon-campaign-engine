package tracker

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

func TestPlayerViewShowsOnlyWhatThePlayersKnow(t *testing.T) {
	s := newState(t)
	s.Monsters[0].Notes = "secret: flees at half Body"
	s.Quest.Traps[0].Label = "T1"
	pv := PlayerView(s)
	if len(pv.Monsters) != 0 || len(pv.Doors) != 0 || len(pv.Furniture) != 0 || len(pv.Blocks) != 0 || len(pv.Traps) != 0 {
		t.Fatalf("nothing is shown at the start: %+v", pv)
	}
	if pv.Width != 6 || pv.Height != 4 || len(pv.Regions) != 24 || len(pv.Heroes) != 2 || pv.Round != 1 || pv.QuestName != "The Test" {
		t.Fatalf("layout and party: %+v", pv)
	}
	if !slices.Equal(pv.Discovered, s.Discovered) {
		t.Fatalf("discovered: %v", pv.Discovered)
	}

	for _, c := range []Command{
		cmd(t, "seen.set", map[string]any{"id": "monster-1", "seen": true}),
		cmd(t, "seen.set", map[string]any{"id": "door-1", "seen": true}),
		cmd(t, "seen.set", map[string]any{"id": "door-2", "seen": true}),
		cmd(t, "seen.set", map[string]any{"id": "furniture-1", "seen": true}),
		cmd(t, "seen.set", map[string]any{"id": "blocked-1", "seen": true}),
		cmd(t, "trap.set", map[string]any{"id": "trap-1", "state": "revealed"}),
		cmd(t, "block.add", map[string]any{"x": 1, "y": 1}),
		cmd(t, "effect.add", map[string]any{"target": "monster-1", "name": "Poisoned", "rounds": 3}),
	} {
		s, _ = apply(t, s, c)
	}
	pv = PlayerView(s)
	if len(pv.Monsters) != 1 || pv.Monsters[0].ID != "monster-1" || pv.Monsters[0].Name != "Orc" || pv.Monsters[0].Body == nil || *pv.Monsters[0].Body != 1 ||
		len(pv.Monsters[0].Effects) != 1 {
		t.Fatalf("seen monster: %+v", pv.Monsters)
	}
	if len(pv.Doors) != 2 || pv.Doors[1].Kind != maps.DoorNormal {
		t.Fatalf("doors (a found secret door looks like any door): %+v", pv.Doors)
	}
	if len(pv.Furniture) != 1 || pv.Furniture[0].ID != "furniture-1" {
		t.Fatalf("furniture: %+v", pv.Furniture)
	}
	if len(pv.Blocks) != 2 || pv.Blocks[0].ID != "blocked-1" || pv.Blocks[1].ID != "added-1" {
		t.Fatalf("blocks (seen ones and added ones): %+v", pv.Blocks)
	}
	if len(pv.Traps) != 1 || pv.Traps[0].Kind != "pit" || pv.Traps[0].State != maps.TrapRevealed {
		t.Fatalf("traps: %+v", pv.Traps)
	}

	// Nothing secret is in what goes over the wire.
	data, _ := json.Marshal(pv)
	for _, secret := range []string{"flees at half Body", "84 gold", "T1", "Lair", "monster-2", "hiddenDoor", "notes", "label"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("the player view leaks %q:\n%s", secret, data)
		}
	}

	// Body can be hidden; a monster at a quarter of its Body or less is Wounded.
	s.Monsters[0].MaxBody, s.Monsters[0].Body = 8, 2
	s.Players.HideMonsterBody = true
	pv = PlayerView(s)
	if m := pv.Monsters[0]; m.Body != nil || m.MaxBody != nil || !m.Wounded {
		t.Fatalf("hidden Body, wounded: %+v", m)
	}

	// Dead monsters, removed blocks and GM trigger markers never show.
	s.Monsters[0].Alive = false
	s.RemovedBlocks = []string{"blocked-1"}
	s.Quest.Traps = append(s.Quest.Traps, maps.Trap{ID: "trap-9", Kind: maps.TrapTrigger, X: 1, Y: 1, State: maps.TrapRevealed})
	s.Traps = append(s.Traps, TrapState{ID: "trap-9", State: maps.TrapRevealed})
	pv = PlayerView(s)
	if len(pv.Monsters) != 0 || len(pv.Blocks) != 1 || len(pv.Traps) != 1 {
		t.Fatalf("dead, removed, markers: %+v %+v %+v", pv.Monsters, pv.Blocks, pv.Traps)
	}
}

func TestPlayerViewHeroesCarryTheirTotals(t *testing.T) {
	s, _ := fightState(t)
	s.Heroes[0].Items = []Item{{ID: "item-1", Name: "Mace", Quantity: 1, Equipped: true, ItemStats: ItemStats{Damage: 3, Mana: 2}}}
	s.Heroes[0].Notes = "owes the innkeeper"
	pv := PlayerView(s)
	h := pv.Heroes[0]
	if h.Name != "Mira" || h.Class != "custom-cleric" || h.ManaCap != 18 || h.Combat == nil || h.Combat.Damage != 3 {
		t.Fatalf("hero: %+v", h)
	}
	if data, _ := json.Marshal(pv); strings.Contains(string(data), "innkeeper") {
		t.Fatal("hero notes stay with the GM")
	}
}

func TestPlayerSummaries(t *testing.T) {
	_, _, cat := fixture()
	cat.Monsters = append(cat.Monsters, content.MonsterDef{ID: "goblin", Name: "Goblin", Body: 1})
	type step struct {
		c    Command
		want string
	}
	s := newState(t)
	run := func(steps []step) {
		t.Helper()
		for _, st := range steps {
			next, ev, err := Apply(s, st.c, cat)
			if err != nil {
				t.Fatalf("%s: %v", st.c.Type, err)
			}
			if ev.PlayerSummary != st.want {
				t.Errorf("%s %s: player summary %q, want %q (GM: %q)", st.c.Type, st.c.Payload, ev.PlayerSummary, st.want, ev.Summary)
			}
			s = next
		}
	}
	run([]step{
		// Hidden things say nothing.
		{cmd(t, "monster.update", map[string]any{"id": "monster-1", "body": 0}), ""},
		{cmd(t, "door.set", map[string]any{"id": "door-1", "state": "open"}), ""},
		{cmd(t, "effect.add", map[string]any{"target": "monster-2", "name": "Marked"}), ""},
		{cmd(t, "trap.trigger", map[string]any{"id": "trap-1"}), ""},
		{cmd(t, "block.add", map[string]any{"x": 1, "y": 1}), ""},
		{cmd(t, "log.note", map[string]any{"text": "the stranger lies"}), ""},
		{cmd(t, "move", map[string]any{"id": "monster-2", "x": 5, "y": 1}), ""},
		{cmd(t, "seen.set", map[string]any{"id": "furniture-1", "seen": true}), ""},
		{cmd(t, "monster.add", map[string]any{"type": "goblin", "x": 4, "y": 1, "visibility": "hidden"}), ""},
		// Seen things and the heroes are described.
		{cmd(t, "seen.set", map[string]any{"id": "monster-2", "seen": true}), "Spotted: Orc"},
		{cmd(t, "area.reveal", map[string]any{"x": 4, "y": 1, "seen": true}), "Spotted: Goblin"},
		{cmd(t, "monster.add", map[string]any{"type": "orc", "x": 3, "y": 2}), "Spotted: Orc"},
		{cmd(t, "monster.update", map[string]any{"id": "monster-2", "body": 3}), "Orc: Body 5 → 3"},
		{cmd(t, "monster.update", map[string]any{"id": "monster-2", "body": 1}), "Orc: Body 3 → 1 (Wounded)"},
		{cmd(t, "effect.add", map[string]any{"target": "monster-2", "name": "Poisoned", "rounds": 3}), "Orc: Poisoned (3 rounds)"},
		{cmd(t, "monster.update", map[string]any{"id": "monster-2", "alive": false}), "Orc slain"},
		{cmd(t, "seen.set", map[string]any{"id": "door-1", "seen": true}), ""},
		{cmd(t, "door.set", map[string]any{"id": "door-1", "state": "closed"}), "A door closed"},
		{cmd(t, "hero.update", map[string]any{"id": "hero-1", "body": 5, "equipment": "Axe"}), "Grom: Body 8 → 5"},
		{cmd(t, "hero.update", map[string]any{"id": "hero-1", "notes": "brave"}), ""},
		{cmd(t, "hero.update", map[string]any{"id": "hero-2", "status": "dead"}), "Ilsa has fallen"},
		{cmd(t, "effect.add", map[string]any{"target": "hero-1", "name": "Raging", "rounds": 3, "note": "+3 damage"}), "Grom: Raging (3 rounds)"},
		{cmd(t, "item.add", map[string]any{"heroId": "hero-1", "name": "Rope"}), "Grom gained Rope"},
		{cmd(t, "gold.set", map[string]any{"gold": 40}), "Party gold 0 → 40"},
		{cmd(t, "fight.start", map[string]any{}), "Fight!"},
		{cmd(t, "round.advance", map[string]any{}), "Round 2"},
		{cmd(t, "fight.end", map[string]any{}), "Fight over"},
	})

	// With Body hidden, a seen monster's hits are told without numbers.
	s.Players.HideMonsterBody = true
	run([]step{
		{cmd(t, "monster.update", map[string]any{"id": "monster-4", "body": 0}), "Orc took damage"},
	})
}
