package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// setupQuest creates a 4x2 corridor board with a quest that has one door,
// one start square and one orc, and returns the quest id.
func setupQuest(t *testing.T, srv interface{ URL() string }, callFn func(method, path string, body any) (int, []byte)) string {
	t.Helper()
	_, data := callFn(http.MethodPost, "/api/boards", map[string]any{"name": "Board", "width": 4, "height": 2})
	b := decodeAny[BoardResponse](t, data)
	board := b.Board
	for i := range board.Regions {
		board.Regions[i] = maps.Corridor
	}
	callFn(http.MethodPut, "/api/boards/"+b.ID, map[string]any{"name": "Board", "board": board})
	_, data = callFn(http.MethodPost, "/api/boards/"+b.ID+"/quests", map[string]any{"name": "The Trial"})
	q := decodeAny[QuestResponse](t, data)
	quest := q.Quest
	quest.Doors = []maps.Door{{ID: "door-1", Edge: maps.Edge{X: 3, Y: 2, Orientation: maps.Vertical}, Kind: maps.DoorNormal, State: maps.DoorClosed}}
	quest.Monsters = []maps.Monster{{ID: "monster-1", Type: "orc", X: 4, Y: 1}}
	quest.StartTiles = []maps.Tile{{X: 1, Y: 2}}
	callFn(http.MethodPut, "/api/quests/"+q.ID, map[string]any{"name": "The Trial", "quest": quest})
	return q.ID
}

func decodeAny[T any](t *testing.T, data []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
	return v
}

type urlServer struct{ url string }

func (u urlServer) URL() string { return u.url }

func TestCampaignSessionAndCommandsFlow(t *testing.T) {
	srv := testServer(t)
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	questID := setupQuest(t, urlServer{srv.URL}, c)

	// Campaign with one hero.
	code, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "Winter Campaign"})
	if code != http.StatusCreated {
		t.Fatalf("create campaign: %d %s", code, data)
	}
	camp := decodeAny[CampaignResponse](t, data)
	code, data = c(http.MethodPut, "/api/campaigns/"+camp.ID, map[string]any{
		"name":   "Winter Campaign",
		"heroes": []map[string]any{{"name": "Faelyn", "player": "Jo", "class": "elf"}},
	})
	if code != http.StatusOK {
		t.Fatalf("update campaign: %d %s", code, data)
	}
	camp = decodeAny[CampaignResponse](t, data)
	if len(camp.Heroes) != 1 || camp.Heroes[0].ID == "" {
		t.Fatalf("heroes should get ids: %+v", camp.Heroes)
	}

	// Start a session.
	code, data = c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID, "name": "Night one"})
	if code != http.StatusCreated {
		t.Fatalf("start session: %d %s", code, data)
	}
	sess := decodeAny[SessionResponse](t, data)
	if sess.Status != "active" || sess.State.QuestName != "The Trial" || len(sess.State.Heroes) != 1 || !sess.State.Heroes[0].Placed || sess.EventSeq != 1 {
		t.Fatalf("session: %+v", sess)
	}

	// Apply commands.
	code, data = c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "door.set", "payload": map[string]any{"id": "door-1", "state": "open"}})
	if code != http.StatusOK {
		t.Fatalf("command: %d %s", code, data)
	}
	res := decodeAny[CommandResponse](t, data)
	if res.State.Doors[0].State != "open" || res.Event.Summary != "Opened door-1" || res.EventSeq != 2 {
		t.Fatalf("command result: %+v", res)
	}

	code, data = c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "door.set", "payload": map[string]any{"id": "door-9", "state": "open"}})
	if code != http.StatusBadRequest || !strings.Contains(string(data), "door-9") {
		t.Fatalf("bad command: %d %s", code, data)
	}

	// Resume: a fresh GET returns the saved state.
	_, data = c(http.MethodGet, "/api/sessions/"+sess.ID, nil)
	resumed := decodeAny[SessionResponse](t, data)
	if resumed.State.Doors[0].State != "open" || resumed.EventSeq != 2 {
		t.Fatalf("resumed: %+v", resumed)
	}

	// The event log reads in order.
	_, data = c(http.MethodGet, "/api/sessions/"+sess.ID+"/events", nil)
	events := decodeAny[[]EventResponse](t, data)
	if len(events) != 2 || events[0].Kind != "session.start" || events[1].Summary != "Opened door-1" {
		t.Fatalf("events: %+v", events)
	}
	_, data = c(http.MethodGet, "/api/sessions/"+sess.ID+"/events?after=1", nil)
	if later := decodeAny[[]EventResponse](t, data); len(later) != 1 || later[0].Seq != 2 {
		t.Fatalf("events after 1: %+v", later)
	}

	// Heroes gain gold, the quest is completed, and gold carries back to the campaign.
	heroID := sess.State.Heroes[0].ID
	c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "hero.update", "payload": map[string]any{"id": heroID, "gold": 84}})
	code, data = c(http.MethodPost, "/api/sessions/"+sess.ID+"/complete", nil)
	if code != http.StatusOK {
		t.Fatalf("complete: %d %s", code, data)
	}
	_, data = c(http.MethodGet, "/api/campaigns/"+camp.ID, nil)
	after := decodeAny[CampaignResponse](t, data)
	if after.Heroes[0].Gold != 84 {
		t.Fatalf("gold should carry over: %+v", after.Heroes)
	}

	// Completed sessions refuse commands until reopened.
	code, _ = c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "round.advance", "payload": map[string]any{}})
	if code != http.StatusConflict {
		t.Fatalf("command on completed session: %d", code)
	}
	if code, _ = c(http.MethodPost, "/api/sessions/"+sess.ID+"/reopen", nil); code != http.StatusOK {
		t.Fatalf("reopen: %d", code)
	}
	if code, _ = c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "round.advance", "payload": map[string]any{}}); code != http.StatusOK {
		t.Fatalf("command after reopen: %d", code)
	}
}

func TestCampaignValidation(t *testing.T) {
	srv := testServer(t)
	if code, _ := call(t, srv, http.MethodPost, "/api/campaigns", map[string]any{"name": " "}); code != http.StatusBadRequest {
		t.Fatalf("empty name: %d", code)
	}
	_, data := call(t, srv, http.MethodPost, "/api/campaigns", map[string]any{"name": "C"})
	camp := decodeAny[CampaignResponse](t, data)
	code, data := call(t, srv, http.MethodPut, "/api/campaigns/"+camp.ID, map[string]any{
		"name": "C", "heroes": []map[string]any{{"name": "X", "class": "paladin"}},
	})
	if code != http.StatusBadRequest || !strings.Contains(string(data), "paladin") {
		t.Fatalf("unknown class: %d %s", code, data)
	}
	code, _ = call(t, srv, http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": "01900000-0000-7000-8000-000000000000", "name": "x"})
	if code != http.StatusNotFound {
		t.Fatalf("session on missing quest: %d", code)
	}
}

func TestSessionStreamPushesEvents(t *testing.T) {
	srv := testServer(t)
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "C"})
	camp := decodeAny[CampaignResponse](t, data)
	_, data = c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID, "name": "S"})
	sess := decodeAny[SessionResponse](t, data)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/sessions/" + sess.ID + "/stream"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "round.advance", "payload": map[string]any{}})

	_, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var pushed CommandResponse
	if err := json.Unmarshal(msg, &pushed); err != nil {
		t.Fatal(err)
	}
	if pushed.Event.Summary != "Round 2 begins" || pushed.State.Round != 2 || pushed.EventSeq != 2 {
		t.Fatalf("pushed: %+v", pushed)
	}
}

// Keep the tracker import used for response types in this file.
var _ = tracker.StateVersion
