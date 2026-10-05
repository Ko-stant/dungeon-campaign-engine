package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestPlayerScreenAPI(t *testing.T) {
	srv := testServer(t)
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "C"})
	camp := decodeAny[CampaignResponse](t, data)
	_, data = c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID, "name": "S"})
	sess := decodeAny[SessionResponse](t, data)
	command := func(kind string, payload map[string]any) {
		t.Helper()
		if code, data := c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": kind, "payload": payload}); code != http.StatusOK {
			t.Fatalf("%s: %d %s", kind, code, data)
		}
	}
	command("monster.update", map[string]any{"id": "monster-1", "notes": "the orc carries the key"})
	command("log.note", map[string]any{"text": "the stranger lies"})
	command("round.advance", map[string]any{})

	code, data := c(http.MethodGet, "/api/sessions/"+sess.ID+"/player", nil)
	if code != http.StatusOK {
		t.Fatalf("player view: %d %s", code, data)
	}
	view := decodeAny[PlayerResponse](t, data)
	if view.State.Round != 2 || len(view.State.Monsters) != 0 || view.EventSeq != 4 {
		t.Fatalf("player view: %+v", view)
	}
	if len(view.Events) != 2 || view.Events[0].Summary != "The quest begins" || view.Events[1].Summary != "Round 2" {
		t.Fatalf("player events: %+v", view.Events)
	}
	for _, secret := range []string{"carries the key", "stranger lies", "monster-1", "Round 2 begins"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("the player view leaks %q", secret)
		}
	}
	if code, _ := c(http.MethodGet, "/api/sessions/01900000-0000-7000-8000-000000000000/player", nil); code != http.StatusNotFound {
		t.Fatalf("unknown session: %d", code)
	}

	// The player stream pushes the filtered view, with the player line when there is one.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/sessions/"+sess.ID+"/player-stream", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
	read := func() (PlayerUpdate, string) {
		t.Helper()
		_, msg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var u PlayerUpdate
		if err := json.Unmarshal(msg, &u); err != nil {
			t.Fatal(err)
		}
		return u, string(msg)
	}

	command("seen.set", map[string]any{"id": "monster-1", "seen": true})
	u, raw := read()
	if u.Event == nil || u.Event.Summary != "Spotted: Orc" || len(u.State.Monsters) != 1 || u.EventSeq != 5 || strings.Contains(raw, "carries the key") {
		t.Fatalf("pushed: %s", raw)
	}
	command("log.note", map[string]any{"text": "the orc is a spy"})
	if u, raw = read(); u.Event != nil || u.EventSeq != 6 || strings.Contains(raw, "spy") {
		t.Fatalf("a GM note pushes the state but no line: %s", raw)
	}
}
