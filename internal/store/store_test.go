package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	s, _ := storetest.New(t)
	return s
}

// jsonEqual compares JSON semantically; jsonb does not preserve formatting or key order.
func jsonEqual(t *testing.T, got, want json.RawMessage) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("unmarshal got: %v (%s)", err, got)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatalf("unmarshal want: %v (%s)", err, want)
	}
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("json mismatch:\n got  %s\n want %s", got, want)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	_, url := storetest.New(t)
	if err := store.Migrate(context.Background(), url); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestBoardLifecycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	doc := json.RawMessage(`{"regions":[0,0,1,1],"rooms":[{"id":1,"name":"Hall"}]}`)
	created, err := s.CreateBoard(ctx, "Tiny", 2, 2, doc)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.Name != "Tiny" || created.Width != 2 || created.Height != 2 {
		t.Fatalf("unexpected created board: %+v", created)
	}

	got, err := s.GetBoard(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	jsonEqual(t, got.Doc, doc)

	list, err := s.ListBoards(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID || list[0].Width != 2 {
		t.Fatalf("unexpected list: %+v", list)
	}

	newDoc := json.RawMessage(`{"regions":[-1,0,0,0,0,0],"rooms":[]}`)
	updated, err := s.UpdateBoard(ctx, created.ID, "Tiny (wide)", 3, 2, newDoc)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Tiny (wide)" || updated.Width != 3 || updated.UpdatedAt.Before(created.UpdatedAt) {
		t.Fatalf("unexpected updated board: %+v", updated)
	}
	jsonEqual(t, updated.Doc, newDoc)

	if err := s.DeleteBoard(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetBoard(ctx, created.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("get after delete: err = %v, want store.ErrNotFound", err)
	}
}

func TestMissingAndMalformedIDsAreNotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	for _, id := range []string{"01900000-0000-7000-8000-000000000000", "not-a-uuid", ""} {
		if _, err := s.GetBoard(ctx, id); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetBoard(%q): err = %v, want store.ErrNotFound", id, err)
		}
		if _, err := s.UpdateBoard(ctx, id, "x", 1, 1, json.RawMessage(`{}`)); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("UpdateBoard(%q): err = %v, want store.ErrNotFound", id, err)
		}
		if err := s.DeleteBoard(ctx, id); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("DeleteBoard(%q): err = %v, want store.ErrNotFound", id, err)
		}
	}
}

func TestBoardSizeIsValidated(t *testing.T) {
	s := testStore(t)
	for _, size := range [][2]int{{0, 5}, {5, 0}, {201, 5}} {
		if _, err := s.CreateBoard(context.Background(), "bad", size[0], size[1], json.RawMessage(`{}`)); err == nil {
			t.Errorf("CreateBoard %dx%d: expected an error", size[0], size[1])
		}
	}
}

func TestBoardInUseCannotBeDeleted(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	b, err := s.CreateBoard(ctx, "Board", 2, 2, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateQuest(ctx, b.ID, "Quest", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteBoard(ctx, b.ID); !errors.Is(err, store.ErrInUse) {
		t.Fatalf("delete board with quest: err = %v, want store.ErrInUse", err)
	}
}

func TestQuestLifecycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	b1, _ := s.CreateBoard(ctx, "One", 2, 2, json.RawMessage(`{}`))
	b2, _ := s.CreateBoard(ctx, "Two", 2, 2, json.RawMessage(`{}`))

	doc := json.RawMessage(`{"doors":[{"x":1,"y":0,"orientation":"vertical"}]}`)
	q1, err := s.CreateQuest(ctx, b1.ID, "The Trial", doc)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.CreateQuest(ctx, b2.ID, "Other", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateQuest(ctx, "01900000-0000-7000-8000-000000000000", "Orphan", json.RawMessage(`{}`)); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("create quest on missing board: err = %v, want store.ErrNotFound", err)
	}

	got, err := s.GetQuest(ctx, q1.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.BoardID != b1.ID || got.Name != "The Trial" {
		t.Fatalf("unexpected quest: %+v", got)
	}
	jsonEqual(t, got.Doc, doc)

	all, _ := s.ListQuests(ctx, "")
	onB1, _ := s.ListQuests(ctx, b1.ID)
	if len(all) != 2 || len(onB1) != 1 || onB1[0].ID != q1.ID {
		t.Fatalf("list: all=%+v onB1=%+v", all, onB1)
	}

	upd, err := s.UpdateQuest(ctx, q1.ID, "The Trial (v2)", json.RawMessage(`{"doors":[]}`))
	if err != nil || upd.Name != "The Trial (v2)" {
		t.Fatalf("update: %+v %v", upd, err)
	}

	if err := s.DeleteQuest(ctx, q1.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetQuest(ctx, q1.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("get after delete: %v", err)
	}
}

func TestCampaignLifecycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	heroes := json.RawMessage(`[{"name":"Grom","class":"barbarian","gold":0}]`)
	c, err := s.CreateCampaign(ctx, "Winter Campaign", heroes)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.GetCampaign(ctx, c.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	jsonEqual(t, got.Heroes, heroes)

	more := json.RawMessage(`[{"name":"Grom","class":"barbarian","gold":120}]`)
	upd, err := s.UpdateCampaign(ctx, c.ID, "Winter Campaign", more)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	jsonEqual(t, upd.Heroes, more)

	list, _ := s.ListCampaigns(ctx)
	if len(list) != 1 || list[0].Name != "Winter Campaign" {
		t.Fatalf("list: %+v", list)
	}
}

func TestSessionEventsUpdateStateInOrder(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, _ := s.CreateCampaign(ctx, "C", json.RawMessage(`[]`))
	b, _ := s.CreateBoard(ctx, "B", 2, 2, json.RawMessage(`{}`))
	q, _ := s.CreateQuest(ctx, b.ID, "Q", json.RawMessage(`{}`))

	sess, err := s.CreateSession(ctx, c.ID, q.ID, "Night one", json.RawMessage(`{"round":1}`))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if sess.Status != store.StatusActive || sess.EventSeq != 0 || sess.QuestID == nil || *sess.QuestID != q.ID {
		t.Fatalf("unexpected session: %+v", sess)
	}

	steps := []struct {
		state string
		ev    store.NewEvent
	}{
		{`{"round":1,"door":"open"}`, store.NewEvent{Round: 1, Kind: "door.state", Summary: "Opened door-1", Payload: json.RawMessage(`{"door":"door-1","to":"open"}`)}},
		{`{"round":1,"door":"closed"}`, store.NewEvent{Round: 1, Kind: "door.state", Summary: "Closed door-1", Payload: json.RawMessage(`{"door":"door-1","to":"closed"}`)}},
		{`{"round":2,"door":"closed"}`, store.NewEvent{Round: 2, Kind: "round.advance", Summary: "Round 2 begins"}},
	}
	for i, step := range steps {
		ev, err := s.RecordEvent(ctx, sess.ID, json.RawMessage(step.state), step.ev)
		if err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
		if ev.Seq != int64(i+1) || ev.Kind != step.ev.Kind || ev.SessionID != sess.ID {
			t.Fatalf("event %d: %+v", i, ev)
		}
	}

	got, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventSeq != 3 {
		t.Fatalf("EventSeq = %d, want 3", got.EventSeq)
	}
	jsonEqual(t, got.State, json.RawMessage(steps[2].state))

	events, err := s.ListEvents(ctx, sess.ID, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Seq != 2 || events[1].Seq != 3 || events[1].Summary != "Round 2 begins" {
		t.Fatalf("events after seq 1: %+v", events)
	}
	jsonEqual(t, events[1].Payload, json.RawMessage(`{}`))

	list, _ := s.ListSessions(ctx, c.ID)
	if len(list) != 1 || list[0].ID != sess.ID || list[0].EventSeq != 3 {
		t.Fatalf("list sessions: %+v", list)
	}
}

func TestRecordEventIsAtomic(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, _ := s.CreateCampaign(ctx, "C", json.RawMessage(`[]`))
	sess, _ := s.CreateSession(ctx, c.ID, "", "Night", json.RawMessage(`{"v":1}`))

	// A malformed payload fails the event insert, which must roll back the state update too.
	_, err := s.RecordEvent(ctx, sess.ID, json.RawMessage(`{"v":2}`), store.NewEvent{Round: 1, Kind: "x", Summary: "x", Payload: json.RawMessage(`{broken`)})
	if err == nil {
		t.Fatal("expected an error for a malformed payload")
	}

	got, _ := s.GetSession(ctx, sess.ID)
	if got.EventSeq != 0 {
		t.Fatalf("EventSeq = %d after failed record, want 0", got.EventSeq)
	}
	jsonEqual(t, got.State, json.RawMessage(`{"v":1}`))

	if _, err := s.RecordEvent(ctx, "01900000-0000-7000-8000-000000000000", json.RawMessage(`{}`), store.NewEvent{Kind: "x", Summary: "x"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("record on missing session: err = %v, want store.ErrNotFound", err)
	}
}

func TestSessionStatus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, _ := s.CreateCampaign(ctx, "C", json.RawMessage(`[]`))
	sess, _ := s.CreateSession(ctx, c.ID, "", "Night", json.RawMessage(`{}`))
	if sess.QuestID != nil {
		t.Fatalf("QuestID = %v, want nil when no quest given", *sess.QuestID)
	}

	if err := s.SetSessionStatus(ctx, sess.ID, store.StatusCompleted); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetSession(ctx, sess.ID)
	if got.Status != store.StatusCompleted {
		t.Fatalf("status = %q", got.Status)
	}
	if err := s.SetSessionStatus(ctx, sess.ID, "paused"); err == nil {
		t.Fatal("expected an error for an unknown status")
	}
}
