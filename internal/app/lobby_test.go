package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// lobbyGame: the GM (dev sign-in) makes a campaign with Grom and a session
// on the setupQuest map (one start square, at (1,2), where Grom stands).
func lobbyGame(t *testing.T) (srvURL string, gm *browser, campaignID, sessionID string) {
	t.Helper()
	srv, _ := devServer(t)
	gm = newBrowser(t, srv.URL)
	gm.signInAs("GM")
	c := func(method, path string, v any) (int, []byte) {
		resp, data := gm.sendJSON(method, path, v)
		return resp.StatusCode, []byte(data)
	}
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "Three Plagues"})
	camp := decodeAny[CampaignResponse](t, data)
	if code, data := c(http.MethodPut, "/api/campaigns/"+camp.ID, map[string]any{"name": "Three Plagues", "heroes": []map[string]any{{"name": "Grom", "class": "elf"}}}); code != http.StatusOK {
		t.Fatalf("heroes: %d %s", code, data)
	}
	code, data := c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID, "name": "Game night"})
	if code != http.StatusCreated {
		t.Fatalf("session: %d %s", code, data)
	}
	return srv.URL, gm, camp.ID, decodeAny[SessionResponse](t, data).ID
}

func seatView(t *testing.T, b *browser, sessionID string) (int, SeatResponse) {
	t.Helper()
	resp, body := b.get("/api/sessions/" + sessionID + "/seat")
	var seat SeatResponse
	if resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal([]byte(body), &seat); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode, seat
}

func campaignHeroes(t *testing.T, gm *browser, campaignID string) []tracker.CampaignHero {
	t.Helper()
	_, body := gm.get("/api/campaigns/" + campaignID)
	return decodeAny[CampaignResponse](t, []byte(body)).Heroes
}

func TestFriendsJoinFromTheLobbyAndPlayATurn(t *testing.T) {
	base, gm, campaignID, sessionID := lobbyGame(t)
	sam, jo, pat := newBrowser(t, base), newBrowser(t, base), newBrowser(t, base)
	sam.signInAs("Sam")
	jo.signInAs("Jo")
	pat.signInAs("Pat")

	// Closed: not listed, can't be joined.
	if _, body := sam.get("/lobby"); strings.Contains(body, "Game night") {
		t.Error("a closed session is not in the lobby")
	}
	if resp, _ := sam.get("/join/" + sessionID); resp.StatusCode != http.StatusNotFound {
		t.Errorf("joining a closed session: %d", resp.StatusCode)
	}
	if resp, _ := sam.postForm("/play/"+sessionID+"/open", url.Values{"open": {"1"}}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("a player opening the GM's session: %d", resp.StatusCode)
	}
	if resp, _ := gm.postForm("/play/"+sessionID+"/open", url.Values{"open": {"1"}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("opening: %d", resp.StatusCode)
	}

	// The lobby lists it with its campaign and GM; the join page offers Grom and a new hero.
	_, lobby := sam.get("/lobby")
	if !strings.Contains(lobby, "Game night") || !strings.Contains(lobby, "Three Plagues") || !strings.Contains(lobby, "GM") || !strings.Contains(lobby, "/join/"+sessionID) {
		t.Fatalf("lobby: %s", lobby)
	}
	resp, join := sam.get("/join/" + sessionID)
	if resp.StatusCode != http.StatusOK || !strings.Contains(join, "Grom") || !strings.Contains(join, `value="elf"`) {
		t.Fatalf("join page: %d %s", resp.StatusCode, join)
	}

	// Sam makes a new hero, who joins the running session (off the board: the one start square is taken).
	if resp, body := sam.postForm("/join/"+sessionID+"/hero", url.Values{"name": {"Vex"}, "class": {"elf"}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("new hero: %d %s", resp.StatusCode, body)
	}
	var vex tracker.CampaignHero
	for _, h := range campaignHeroes(t, gm, campaignID) {
		if h.Name == "Vex" {
			vex = h
		}
	}
	if vex.ID == "" || vex.UserID == "" || vex.Player != "Sam" {
		t.Fatalf("Vex %+v", vex)
	}
	// Jo takes Grom.
	grom := campaignHeroes(t, gm, campaignID)[0]
	if resp, _ := jo.postForm("/join/"+sessionID+"/claim/"+grom.ID, nil); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("claim: %d", resp.StatusCode)
	}
	if resp, _ := sam.postForm("/join/"+sessionID+"/claim/"+grom.ID, nil); resp.StatusCode != http.StatusConflict {
		t.Errorf("claiming a taken hero: %d", resp.StatusCode)
	}

	// Saving the heroes from the GM's side keeps who plays them.
	heroes := campaignHeroes(t, gm, campaignID)
	for i := range heroes {
		heroes[i].UserID = ""
	}
	if resp, _ := gm.sendJSON(http.MethodPut, "/api/campaigns/"+campaignID, map[string]any{"name": "Three Plagues", "heroes": heroes}); resp.StatusCode != http.StatusOK {
		t.Fatal("GM save")
	}
	if got := campaignHeroes(t, gm, campaignID); got[0].UserID == "" || got[1].UserID == "" {
		t.Fatalf("player links lost on a GM save: %+v", got)
	}

	// Each player sees only their own heroes; outsiders see nothing.
	if code, seat := seatView(t, sam, sessionID); code != http.StatusOK || len(seat.Seat.Heroes) != 1 || seat.Seat.Heroes[0].ID != vex.ID {
		t.Fatalf("Sam's seat: %d %+v", code, seat)
	}
	if code, _ := seatView(t, pat, sessionID); code != http.StatusForbidden {
		t.Errorf("Pat has no hero here: %d", code)
	}
	if resp, _ := jo.get("/api/sessions/" + sessionID); resp.StatusCode != http.StatusForbidden {
		t.Errorf("a player reading the GM's full state: %d", resp.StatusCode)
	}
	if resp, _ := jo.get("/api/sessions/" + sessionID + "/player"); resp.StatusCode != http.StatusOK {
		t.Errorf("a seated player reading the player view: %d", resp.StatusCode)
	}

	// The GM starts online play: the rules come on.
	if resp, _ := gm.postForm("/play/"+sessionID+"/start", nil); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("start: %d", resp.StatusCode)
	}
	code, seat := seatView(t, jo, sessionID)
	if code != http.StatusOK || seat.Seat.Phase != tracker.PhaseHeroes || len(seat.Seat.Heroes) != 1 || len(seat.Seat.Heroes[0].Actions) != 1 || seat.Seat.Heroes[0].Actions[0].Label != "Start Grom's turn" {
		t.Fatalf("Jo's seat after the start: %d %+v", code, seat)
	}

	// Jo plays Grom's turn from the seat; Sam can't act for Grom.
	send := func(b *browser, hero, kind string, payload map[string]any) (*http.Response, string) {
		return b.sendJSON(http.MethodPost, "/api/sessions/"+sessionID+"/seat-commands", map[string]any{"hero": hero, "type": kind, "payload": payload})
	}
	if resp, body := send(sam, grom.ID, "turn.start", map[string]any{"hero": grom.ID}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("Sam acting for Grom: %d %s", resp.StatusCode, body)
	}
	for _, step := range []struct {
		kind    string
		payload map[string]any
	}{
		{"turn.start", map[string]any{"hero": grom.ID}},
		{"turn.roll-move", map[string]any{}},
		{"turn.end", map[string]any{}},
	} {
		if resp, body := send(jo, grom.ID, step.kind, step.payload); resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %s", step.kind, resp.StatusCode, body)
		}
	}
	if resp, body := send(jo, grom.ID, "move", map[string]any{"id": grom.ID, "x": 2, "y": 2}); resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "GM") {
		t.Errorf("a GM command from a seat: %d %s", resp.StatusCode, body)
	}
	_, events := gm.get("/api/sessions/" + sessionID + "/events")
	if !strings.Contains(events, "Grom (Elf) starts their turn") || !strings.Contains(events, "Grom (Elf) ends their turn") {
		t.Errorf("the turn is in the log: %s", events)
	}
}

func TestStartingTwiceAndClosing(t *testing.T) {
	base, gm, _, sessionID := lobbyGame(t)
	gm.postForm("/play/"+sessionID+"/open", url.Values{"open": {"1"}})
	gm.postForm("/play/"+sessionID+"/start", nil)
	if resp, _ := gm.postForm("/play/"+sessionID+"/start", nil); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("starting again just goes back: %d", resp.StatusCode)
	}
	gm.postForm("/play/"+sessionID+"/open", url.Values{"open": {"0"}})
	sam := newBrowser(t, base)
	sam.signInAs("Sam")
	if _, body := sam.get("/lobby"); strings.Contains(body, "Game night") {
		t.Error("a closed session leaves the lobby")
	}
}

func TestTheLobbyAndSeatsNeedSignInOn(t *testing.T) {
	srv := testServer(t) // AUTH_MODE none: the table companion
	b := newBrowser(t, srv.URL)
	for _, path := range []string{"/lobby", "/join/0190c6a0-0000-7000-8000-000000000000", "/api/sessions/0190c6a0-0000-7000-8000-000000000000/seat"} {
		if resp, _ := b.get(path); resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s with sign-in off: %d", path, resp.StatusCode)
		}
	}
	if resp, _ := b.sendJSON(http.MethodPost, "/api/sessions/0190c6a0-0000-7000-8000-000000000000/seat-commands", map[string]any{"hero": "hero-1", "type": "turn.end"}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("a seat command with sign-in off: %d", resp.StatusCode)
	}
}

// seatSocket opens a player's seat stream with their browser's cookies.
func seatSocket(t *testing.T, b *browser, sessionID string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(b.base, "http")+"/api/sessions/"+sessionID+"/seat-stream",
		&websocket.DialOptions{HTTPClient: b.client})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })
	return conn
}

func readSeatUpdate(t *testing.T, conn *websocket.Conn) SeatUpdate {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var u SeatUpdate
	if err := json.Unmarshal(data, &u); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestSeatsGetTheirOwnUpdatesAndSeeWhoIsHere(t *testing.T) {
	base, gm, campaignID, sessionID := lobbyGame(t)
	gm.postForm("/play/"+sessionID+"/open", url.Values{"open": {"1"}})
	jo, sam := newBrowser(t, base), newBrowser(t, base)
	jo.signInAs("Jo")
	sam.signInAs("Sam")
	grom := campaignHeroes(t, gm, campaignID)[0]
	jo.postForm("/join/"+sessionID+"/claim/"+grom.ID, nil)
	sam.postForm("/join/"+sessionID+"/hero", url.Values{"name": {"Vex"}, "class": {"elf"}})

	joConn := seatSocket(t, jo, sessionID)
	if u := readSeatUpdate(t, joConn); len(u.Presence) != 1 || u.Presence[0].Name != "Jo" || u.Presence[0].Heroes[0] != "Grom" {
		t.Fatalf("Jo arrives: %+v", u)
	}
	samConn := seatSocket(t, sam, sessionID)
	if u := readSeatUpdate(t, joConn); len(u.Presence) != 2 || u.Presence[1].Name != "Sam" {
		t.Fatalf("Sam arrives: %+v", u)
	}
	readSeatUpdate(t, samConn) // Sam's own arrival

	// A change reaches each seat with that player's own heroes.
	gm.postForm("/play/"+sessionID+"/start", nil)
	joUpdate, samUpdate := readSeatUpdate(t, joConn), readSeatUpdate(t, samConn)
	if joUpdate.Seat == nil || joUpdate.Player == nil || len(joUpdate.Seat.Heroes) != 1 || joUpdate.Seat.Heroes[0].Name != "Grom" || joUpdate.Seat.Phase != tracker.PhaseHeroes {
		t.Fatalf("Jo's update: %+v", joUpdate)
	}
	if samUpdate.Seat == nil || len(samUpdate.Seat.Heroes) != 1 || samUpdate.Seat.Heroes[0].Name != "Vex" {
		t.Fatalf("Sam's update: %+v", samUpdate)
	}

	// The GM sees who is here; an outsider can't open a seat stream.
	_, body := gm.get("/api/sessions/" + sessionID + "/presence")
	if !strings.Contains(body, `"name":"Jo"`) || !strings.Contains(body, `"name":"Sam"`) {
		t.Errorf("presence: %s", body)
	}
	pat := newBrowser(t, base)
	pat.signInAs("Pat")
	if resp, _ := pat.get("/api/sessions/" + sessionID + "/seat-stream"); resp.StatusCode != http.StatusForbidden {
		t.Errorf("an outsider's seat stream: %d", resp.StatusCode)
	}

	// Leaving is announced.
	_ = samConn.Close(websocket.StatusNormalClosure, "")
	if u := readSeatUpdate(t, joConn); len(u.Presence) != 1 || u.Presence[0].Name != "Jo" {
		t.Errorf("Sam leaves: %+v", u)
	}
}
