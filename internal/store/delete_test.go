package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// deleteFixture is a board with a quest that is a chapter of a campaign with
// one session.
type deleteFixture struct {
	board, quest, campaign, session string
}

func newDeleteFixture(t *testing.T, s *store.Store) deleteFixture {
	t.Helper()
	ctx := context.Background()
	b, err := s.CreateBoard(ctx, "Board", 2, 2, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	q, err := s.CreateQuest(ctx, b.ID, "Quest", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CreateCampaign(ctx, "Campaign", json.RawMessage(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetChapters(ctx, c.ID, []string{q.ID}); err != nil {
		t.Fatal(err)
	}
	ss, err := s.CreateSession(ctx, c.ID, q.ID, "Night one", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	return deleteFixture{board: b.ID, quest: q.ID, campaign: c.ID, session: ss.ID}
}

func TestDeleteCampaignTakesItsChaptersAndSessions(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	f := newDeleteFixture(t, s)

	if err := s.DeleteCampaign(ctx, f.campaign); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetCampaign(ctx, f.campaign); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("campaign after delete: err = %v, want ErrNotFound", err)
	}
	if _, err := s.GetSession(ctx, f.session); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("session after delete: err = %v, want ErrNotFound", err)
	}
	if chapters, err := s.ListAllChapters(ctx); err != nil || len(chapters) != 0 {
		t.Errorf("chapters after delete: %v, %v", chapters, err)
	}
	// The maps stay.
	if _, err := s.GetQuest(ctx, f.quest); err != nil {
		t.Errorf("quest should stay: %v", err)
	}
	if err := s.DeleteCampaign(ctx, f.campaign); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("second delete: err = %v, want ErrNotFound", err)
	}
	if err := s.DeleteCampaign(ctx, "not-an-id"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("bad id: err = %v, want ErrNotFound", err)
	}
}

func TestDeleteSessionOnlyWithinItsCampaign(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	f := newDeleteFixture(t, s)
	other, err := s.CreateCampaign(ctx, "Other", json.RawMessage(`[]`))
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteSession(ctx, other.ID, f.session); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("delete through another campaign: err = %v, want ErrNotFound", err)
	}
	if _, err := s.GetSession(ctx, f.session); err != nil {
		t.Fatalf("session should survive a delete through another campaign: %v", err)
	}
	if err := s.DeleteSession(ctx, f.campaign, f.session); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSession(ctx, f.session); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("session after delete: err = %v, want ErrNotFound", err)
	}
	if _, err := s.GetCampaign(ctx, f.campaign); err != nil {
		t.Errorf("campaign should stay: %v", err)
	}
}

func TestDeleteBoardWithQuests(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	f := newDeleteFixture(t, s)

	if err := s.DeleteBoardWithQuests(ctx, f.board); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetBoard(ctx, f.board); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("board after delete: err = %v, want ErrNotFound", err)
	}
	if _, err := s.GetQuest(ctx, f.quest); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("quest after delete: err = %v, want ErrNotFound", err)
	}
	if chapters, err := s.ListChapters(ctx, f.campaign); err != nil || len(chapters) != 0 {
		t.Errorf("chapters after delete: %v, %v", chapters, err)
	}
	// Sessions keep their frozen copy of the map.
	ss, err := s.GetSession(ctx, f.session)
	if err != nil {
		t.Fatalf("session should stay: %v", err)
	}
	if ss.QuestID != nil {
		t.Errorf("session quest id = %v, want nil", *ss.QuestID)
	}
	if err := s.DeleteBoardWithQuests(ctx, f.board); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("second delete: err = %v, want ErrNotFound", err)
	}
}
