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
