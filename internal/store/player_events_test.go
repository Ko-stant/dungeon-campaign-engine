package store_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func TestPlayerEvents(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, _ := s.CreateCampaign(ctx, "C", json.RawMessage(`[]`))
	sess, err := s.CreateSession(ctx, c.ID, "", "Night one", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range []store.NewEvent{
		{Round: 1, Kind: "trap.trigger", Summary: "Pit Trap trap-1 triggered"},
		{Round: 1, Kind: "door.set", Summary: "Opened door-1", PlayerSummary: "A door opened"},
		{Round: 1, Kind: "log.note", Summary: "The stranger lies"},
		{Round: 2, Kind: "round.advance", Summary: "Round 2 begins; mana: Mira 10 → 12", PlayerSummary: "Round 2"},
		{Round: 2, Kind: "monster.update", Summary: "Orc (monster-1): body 9 → 0", PlayerSummary: "Orc slain"},
	} {
		saved, err := s.RecordEvent(ctx, sess.ID, json.RawMessage(`{}`), ev)
		if err != nil {
			t.Fatal(err)
		}
		if saved.PlayerSummary != ev.PlayerSummary {
			t.Fatalf("recorded player summary: %q", saved.PlayerSummary)
		}
	}
	all, _ := s.ListEvents(ctx, sess.ID, 0, 0)
	if len(all) != 5 || all[1].PlayerSummary != "A door opened" || all[0].PlayerSummary != "" {
		t.Fatalf("the GM's log keeps both summaries: %+v", all)
	}

	// The latest player lines, oldest first; events the players don't hear about are left out.
	got, err := s.ListPlayerEvents(ctx, sess.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].PlayerSummary != "Round 2" || got[1].PlayerSummary != "Orc slain" || got[1].Seq != 5 {
		t.Fatalf("player events: %+v", got)
	}
	if got, _ := s.ListPlayerEvents(ctx, "nope", 10); len(got) != 0 {
		t.Fatalf("bad id: %+v", got)
	}
}

func TestRetractSpotted(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, _ := s.CreateCampaign(ctx, "C", json.RawMessage(`[]`))
	sess, err := s.CreateSession(ctx, c.ID, "", "Night one", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	record := func(ev store.NewEvent) {
		t.Helper()
		if _, err := s.RecordEvent(ctx, sess.ID, json.RawMessage(`{}`), ev); err != nil {
			t.Fatal(err)
		}
	}
	record(store.NewEvent{Round: 1, Kind: "area.reveal", Summary: "Revealed", PlayerSpotted: json.RawMessage(`[{"id":"monster-3","name":"Goblin","map":"quest-a"},{"id":"monster-4","name":"Orc","map":"quest-a"}]`)})
	record(store.NewEvent{Round: 1, Kind: "seen.set", Summary: "Seen", PlayerSummary: "A door opened", PlayerSpotted: json.RawMessage(`[{"id":"monster-3","name":"Skeleton","map":"quest-b"}]`)})
	record(store.NewEvent{Round: 1, Kind: "log.note", Summary: "Note"})
	spotted := func() []string {
		t.Helper()
		got, err := s.ListPlayerEvents(ctx, sess.ID, 10)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, e := range got {
			out = append(out, string(e.PlayerSpotted)+"|"+e.PlayerSummary)
		}
		return out
	}
	if got := spotted(); len(got) != 2 {
		t.Fatalf("events with only sightings count as player lines: %q", got)
	}

	// Removing monster-3 on quest-a takes it out of that sighting only.
	record(store.NewEvent{Round: 1, Kind: "monster.remove", Summary: "Removed Goblin", RetractSpotted: &store.SpottedRef{ID: "monster-3", Map: "quest-a"}})
	got := spotted()
	if len(got) != 2 || got[0] != `[{"id": "monster-4", "map": "quest-a", "name": "Orc"}]|` || got[1] != `[{"id": "monster-3", "map": "quest-b", "name": "Skeleton"}]|A door opened` {
		t.Fatalf("after removing the goblin: %q", got)
	}
	// A line left with nothing to say drops out of the feed.
	record(store.NewEvent{Round: 1, Kind: "monster.remove", Summary: "Removed Orc", RetractSpotted: &store.SpottedRef{ID: "monster-4", Map: "quest-a"}})
	if got := spotted(); len(got) != 1 || got[0] != `[{"id": "monster-3", "map": "quest-b", "name": "Skeleton"}]|A door opened` {
		t.Fatalf("after removing the orc: %q", got)
	}
	// The GM's log is untouched.
	if all, _ := s.ListEvents(ctx, sess.ID, 0, 0); len(all) != 5 || all[0].Summary != "Revealed" {
		t.Fatalf("GM log: %+v", all)
	}
}
