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
	if resp, _ := postForm(t, noRedirects(), srv.URL+"/monsters", ogreForm()); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("custom monster: %d", resp.StatusCode)
	}
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
	// The screen's own catalog: names, sizes and artwork, never a custom monster's notes.
	if !strings.Contains(string(data), `"Cave Ogre"`) || view.Catalog.Monsters == nil || view.Catalog.Furniture == nil {
		t.Fatalf("player catalog: %+v", view.Catalog)
	}
	for _, secret := range []string{"carries the key", "stranger lies", "monster-1", "Round 2 begins", "Smashes doors"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("the player view leaks %q", secret)
		}
	}
	if code, _ := c(http.MethodGet, "/api/sessions/01900000-0000-7000-8000-000000000000/player", nil); code != http.StatusNotFound {
		t.Fatalf("unknown session: %d", code)
	}
	status, page := get(t, noRedirects(), srv.URL+"/play/"+sess.ID+"/players")
	if status != http.StatusOK || !strings.Contains(page, `id="players"`) || !strings.Contains(page, "players.js") || !strings.Contains(page, sess.ID) {
		t.Fatalf("player screen page: %d", status)
	}
	if status, _ := get(t, noRedirects(), srv.URL+"/play/01900000-0000-7000-8000-000000000000/players"); status != http.StatusNotFound {
		t.Fatalf("unknown session page: %d", status)
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

	// Removing a living monster (added by mistake) takes back its sighting: the
	// screen gets the whole feed again, without it.
	command("monster.remove", map[string]any{"id": "monster-1"})
	u, raw = read()
	if u.Event != nil || u.Feed == nil || strings.Contains(raw, "Spotted") || strings.Contains(raw, "monster-1") {
		t.Fatalf("removal pushes the feed without the sighting: %s", raw)
	}
	if len(u.Feed) != 2 || u.Feed[0].Summary != "The quest begins" || u.Feed[1].Summary != "Round 2" {
		t.Fatalf("feed after removal: %+v", u.Feed)
	}
	if _, data := c(http.MethodGet, "/api/sessions/"+sess.ID+"/player", nil); strings.Contains(string(data), "Spotted") {
		t.Fatalf("the sighting stays gone on reload: %s", data)
	}
}
