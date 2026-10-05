package app

import (
	"net/http"
	"strings"
	"testing"
)

func TestTheGMEndpointIgnoresAClaimedActor(t *testing.T) {
	srv := testServer(t)
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "C"})
	camp := decodeAny[CampaignResponse](t, data)
	_, data = c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID, "name": "Night"})
	sess := decodeAny[SessionResponse](t, data)

	// A seat could not send "move"; the GM's endpoint treats every command
	// as the GM's, whatever the client says.
	code, data := c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{
		"type": "door.set", "payload": map[string]any{"id": "door-1", "state": "open"},
		"actor": map[string]any{"kind": "seat", "heroId": "someone"},
	})
	if code != http.StatusOK {
		t.Fatalf("command: %d %s", code, data)
	}
	if strings.Contains(string(data), `"actor"`) {
		t.Errorf("the event should not carry the claimed actor: %s", data)
	}
}

func TestTheGMsActionsAtTheTable(t *testing.T) {
	srv := testServer(t)
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "C"})
	camp := decodeAny[CampaignResponse](t, data)
	_, data = c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID, "name": "Night"})
	sess := decodeAny[SessionResponse](t, data)
	// With the rules off there is nothing to list (the GM does it all by hand).
	code, data := c(http.MethodGet, "/api/sessions/"+sess.ID+"/actions", nil)
	if got := decodeAny[ActionsResponse](t, data); code != http.StatusOK || got.Actions == nil || len(got.Actions) != 0 {
		t.Errorf("table mode: %d %s", code, data)
	}
	if code, _ := c(http.MethodGet, "/api/sessions/0190c6a0-0000-7000-8000-000000000000/actions", nil); code != http.StatusNotFound {
		t.Errorf("a missing session: %d", code)
	}
}
