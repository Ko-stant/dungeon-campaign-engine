package app

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// userIDOf is the signed-in user's id, from /api/me.
func userIDOf(t *testing.T, b *browser) string {
	t.Helper()
	_, body := b.get("/api/me")
	return decodeAny[MeResponse](t, []byte(body)).ID
}

// A player who can't come lets a friend play their hero for the night: the
// GM hands the hero over, and back again afterward.
func TestTheGMChoosesWhoPlaysAHero(t *testing.T) {
	base, gm, campaignID, sessionID := lobbyGame(t)
	gm.postForm("/play/"+sessionID+"/start", nil)
	jo, sam := newBrowser(t, base), newBrowser(t, base)
	jo.signInAs("Jo")
	sam.signInAs("Sam")
	joID, samID := userIDOf(t, jo), userIDOf(t, sam)
	grom := campaignHeroes(t, gm, campaignID)[0]
	sam.postForm("/join/"+sessionID+"/claim/"+grom.ID, nil)

	// The campaign page offers every member, and who plays each hero.
	_, page := gm.get("/campaigns/" + campaignID)
	if !strings.Contains(page, `action="/campaigns/`+campaignID+`/heroes/`+grom.ID+`/player"`) || !strings.Contains(page, `value="`+joID+`"`) ||
		!strings.Contains(page, "Nobody (free)") {
		t.Fatalf("the Played by choice: %s", page)
	}

	// Jo plays their own Vex, with the seat open, when the GM also gives
	// them Grom: the seat updates at once, and Jo plays both.
	jo.postForm("/join/"+sessionID+"/hero", url.Values{"name": {"Vex"}, "class": {"elf"}})
	joConn := seatSocket(t, jo, sessionID)
	readSeatUpdate(t, joConn) // Jo's own arrival
	if resp, _ := gm.postForm("/campaigns/"+campaignID+"/heroes/"+grom.ID+"/player", url.Values{"user": {joID}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("hand Grom to Jo: %d", resp.StatusCode)
	}
	// Jo plays Grom now; Grom is still Sam's hero (who usually plays him).
	if h := campaignHeroes(t, gm, campaignID)[0]; h.UserID != joID || h.Player != "Sam" {
		t.Fatalf("Grom played by Jo, usually Sam: %+v", h)
	}
	got := map[string]bool{}
	for range 2 { // the seat and who is here, in either order
		u := readSeatUpdate(t, joConn)
		if u.Seat != nil && len(u.Seat.Heroes) == 2 && slices.ContainsFunc(u.Seat.Heroes, func(h tracker.SeatHero) bool { return h.Name == "Grom" }) {
			got["seat"] = true
		}
		if len(u.Presence) == 1 && slices.Contains(u.Presence[0].Heroes, "Grom") {
			got["presence"] = true
		}
	}
	if !got["seat"] || !got["presence"] {
		t.Errorf("Jo's seat and presence after the handover: %v", got)
	}
	if code, seat := seatView(t, sam, sessionID); code != http.StatusForbidden {
		t.Errorf("Sam no longer plays here: %d %+v", code, seat)
	}

	// Back to Sam, then nobody.
	gm.postForm("/campaigns/"+campaignID+"/heroes/"+grom.ID+"/player", url.Values{"user": {samID}})
	if code, seat := seatView(t, sam, sessionID); code != http.StatusOK || len(seat.Seat.Heroes) != 1 {
		t.Errorf("Grom back with Sam: %d %+v", code, seat)
	}
	gm.postForm("/campaigns/"+campaignID+"/heroes/"+grom.ID+"/player", url.Values{"user": {""}})
	if h := campaignHeroes(t, gm, campaignID)[0]; h.UserID != "" {
		t.Errorf("Grom is free: %+v", h)
	}
	if _, join := jo.get("/join/" + sessionID); !strings.Contains(join, "/claim/"+grom.ID) || !strings.Contains(join, "usually Sam") {
		t.Errorf("a freed hero can be picked on the join page, named with who usually plays it: %s", join)
	}
	if _, page := gm.get("/campaigns/" + campaignID); !strings.Contains(page, "usually Sam") {
		t.Errorf("a free hero keeps its table player's name on the campaign page: %s", page)
	}

	// Only the GM hands heroes over, and only to someone who signed in.
	if resp, _ := jo.postForm("/campaigns/"+campaignID+"/heroes/"+grom.ID+"/player", url.Values{"user": {joID}}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("a player taking a hero: %d", resp.StatusCode)
	}
	if resp, _ := gm.postForm("/campaigns/"+campaignID+"/heroes/"+grom.ID+"/player", url.Values{"user": {"0190c6a0-0000-7000-8000-000000000000"}}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("an unknown player: %d", resp.StatusCode)
	}
	if resp, _ := gm.postForm("/campaigns/"+campaignID+"/heroes/nope/player", url.Values{"user": {joID}}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("an unknown hero: %d", resp.StatusCode)
	}
}

// Starting online play also opens the session to players, and the campaign
// page says plainly whether players can join.
func TestStartingOnlinePlayOpensTheSession(t *testing.T) {
	base, gm, campaignID, sessionID := lobbyGame(t)
	_, page := gm.get("/campaigns/" + campaignID)
	if !strings.Contains(page, "Closed to players") {
		t.Errorf("a new session is shown closed: %s", page)
	}
	gm.postForm("/play/"+sessionID+"/start", nil)
	sam := newBrowser(t, base)
	sam.signInAs("Sam")
	if _, lobby := sam.get("/lobby"); !strings.Contains(lobby, "/join/"+sessionID) {
		t.Errorf("started online, the session is in the lobby: %s", lobby)
	}
	if _, page = gm.get("/campaigns/" + campaignID); !strings.Contains(page, "Open to players") || !strings.Contains(page, ">Close<") {
		t.Errorf("an open session says so, with a Close button: %s", page)
	}
}

// A player lands on Campaigns after signing in; with none of their own, it
// points them to the games they can join.
func TestPlayersAreShownWhereToJoin(t *testing.T) {
	base, gm, _, _ := lobbyGame(t)
	sam := newBrowser(t, base)
	sam.signInAs("Sam")
	if _, page := sam.get("/campaigns"); !strings.Contains(page, "Here to play?") || !strings.Contains(page, `href="/lobby" class="font-semibold`) {
		t.Errorf("a player's Campaigns page: %s", page)
	}
	if _, page := gm.get("/campaigns"); strings.Contains(page, "Here to play?") {
		t.Error("the GM, with campaigns of their own, isn't told")
	}
}
