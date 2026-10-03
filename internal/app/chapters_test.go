package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// before reports whether a appears before b in s (both must appear).
func before(t *testing.T, s, a, b string) bool {
	t.Helper()
	i, j := strings.Index(s, a), strings.Index(s, b)
	if i < 0 || j < 0 {
		t.Fatalf("page is missing %q or %q", a, b)
	}
	return i < j
}

func TestCampaignChaptersAndMapsGrouping(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	trialID := setupQuest(t, urlServer{srv.URL}, c)

	resp, _ := postForm(t, client, srv.URL+"/campaigns", url.Values{"name": {"Beginners Herald"}})
	campaignURL := resp.Header.Get("Location")
	postForm(t, client, srv.URL+campaignURL+"/heroes", url.Values{"name": {"Faelyn"}, "class": {"elf"}})

	code, body := get(t, client, srv.URL+campaignURL)
	if code != http.StatusOK || !strings.Contains(body, "No chapters yet") {
		t.Fatalf("empty chapters: %d", code)
	}

	// Add an existing quest as chapter 1.
	resp, _ = postForm(t, client, srv.URL+campaignURL+"/chapters", url.Values{"questId": {trialID}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("add chapter: %d", resp.StatusCode)
	}

	// Create a new map for the campaign: a board plus a quest, added as chapter 2,
	// opened in the editor with that quest selected.
	resp, _ = postForm(t, client, srv.URL+campaignURL+"/maps", url.Values{"name": {"Lower Vaults"}, "width": {"30"}, "height": {"24"}})
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/maps/") || !strings.Contains(loc, "/edit?quest=") {
		t.Fatalf("new map: %d %q", resp.StatusCode, loc)
	}
	resp, body = postForm(t, client, srv.URL+campaignURL+"/maps", url.Values{"name": {"Too Big"}, "width": {"300"}, "height": {"24"}})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `role="alert"`) {
		t.Fatalf("bad new map: %d", resp.StatusCode)
	}

	_, body = get(t, client, srv.URL+campaignURL)
	if !before(t, body, "The Trial", "Lower Vaults") {
		t.Fatal("chapters should be in the order they were added")
	}
	if strings.Contains(body, "No chapters yet") {
		t.Fatal("chapters should be listed")
	}

	// Move chapter 2 up.
	vaultsID := strings.TrimPrefix(loc[strings.Index(loc, "?quest="):], "?quest=")
	resp, _ = postForm(t, client, srv.URL+campaignURL+"/chapters/"+vaultsID+"/up", url.Values{})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("move up: %d", resp.StatusCode)
	}
	_, body = get(t, client, srv.URL+campaignURL)
	if !before(t, body, "Lower Vaults", "The Trial") {
		t.Fatal("Lower Vaults should now be chapter 1")
	}

	// The start form suggests the next chapter not yet played.
	if !strings.Contains(body, `value="`+vaultsID+`" selected`) {
		t.Fatal("the next unplayed chapter should be preselected")
	}

	// Playing it marks it in progress.
	resp, _ = postForm(t, client, srv.URL+campaignURL+"/sessions", url.Values{"questId": {vaultsID}, "name": {"Night one"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("start: %d", resp.StatusCode)
	}
	_, body = get(t, client, srv.URL+campaignURL)
	if !strings.Contains(body, "in progress") || !strings.Contains(body, `value="`+trialID+`" selected`) {
		t.Fatal("chapter 1 should be in progress and chapter 2 suggested next")
	}

	// The maps page groups the campaign's maps.
	_, body = get(t, client, srv.URL+"/maps")
	if !before(t, body, "Beginners Herald", "Lower Vaults") || !strings.Contains(body, "/edit?quest="+vaultsID) {
		t.Fatal("maps page should list the campaign's chapters with editor links")
	}

	// Remove a chapter (the quest itself stays).
	resp, _ = postForm(t, client, srv.URL+campaignURL+"/chapters/"+trialID+"/remove", url.Values{})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("remove: %d", resp.StatusCode)
	}
	_, body = get(t, client, srv.URL+campaignURL)
	if strings.Contains(body, "chapter-"+trialID) {
		t.Fatal("removed chapter still listed")
	}
	if code, _ := c(http.MethodGet, "/api/quests/"+trialID, nil); code != http.StatusOK {
		t.Fatalf("removing a chapter must not delete the quest: %d", code)
	}

	resp, body = postForm(t, client, srv.URL+campaignURL+"/chapters", url.Values{"questId": {"01900000-0000-7000-8000-000000000000"}})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `role="alert"`) {
		t.Fatalf("unknown quest: %d", resp.StatusCode)
	}
}

func TestTravelBetweenChapterMapsMidGame(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	upperID := setupQuest(t, urlServer{srv.URL}, c)

	// A second map with its own quest and start square.
	_, data := c(http.MethodPost, "/api/boards", map[string]any{"name": "Vaults Map", "width": 3, "height": 2})
	b := decodeAny[BoardResponse](t, data)
	board := b.Board
	for i := range board.Regions {
		board.Regions[i] = 0
	}
	c(http.MethodPut, "/api/boards/"+b.ID, map[string]any{"name": "Vaults Map", "board": board})
	_, data = c(http.MethodPost, "/api/boards/"+b.ID+"/quests", map[string]any{"name": "Lower Vaults"})
	lower := decodeAny[QuestResponse](t, data)
	lq := lower.Quest
	lq.StartTiles = []maps.Tile{{X: 3, Y: 2}}
	c(http.MethodPut, "/api/quests/"+lower.ID, map[string]any{"name": "Lower Vaults", "quest": lq})

	resp, _ := postForm(t, client, srv.URL+"/campaigns", url.Values{"name": {"Herald"}})
	campaignURL := resp.Header.Get("Location")
	campaignID := strings.TrimPrefix(campaignURL, "/campaigns/")
	postForm(t, client, srv.URL+campaignURL+"/heroes", url.Values{"name": {"Faelyn"}, "class": {"elf"}})
	postForm(t, client, srv.URL+campaignURL+"/chapters", url.Values{"questId": {upperID}})
	postForm(t, client, srv.URL+campaignURL+"/chapters", url.Values{"questId": {lower.ID}})

	code, data := c(http.MethodGet, "/api/campaigns/"+campaignID+"/chapters", nil)
	chapters := decodeAny[[]ChapterResponse](t, data)
	if code != http.StatusOK || len(chapters) != 2 || chapters[1].QuestID != lower.ID || chapters[1].BoardName != "Vaults Map" || chapters[1].Number != 2 {
		t.Fatalf("chapters API: %d %s", code, data)
	}

	_, data = c(http.MethodPost, "/api/campaigns/"+campaignID+"/sessions", map[string]any{"questId": upperID, "name": "Night"})
	sess := decodeAny[SessionResponse](t, data)
	if sess.State.QuestID != upperID {
		t.Fatalf("a new session records its map's quest: %q", sess.State.QuestID)
	}
	c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "hero.update", "payload": map[string]any{"id": sess.State.Heroes[0].ID, "body": 2}})

	code, data = c(http.MethodPost, "/api/sessions/"+sess.ID+"/travel", map[string]any{"questId": lower.ID})
	if code != http.StatusOK {
		t.Fatalf("travel: %d %s", code, data)
	}
	res := decodeAny[CommandResponse](t, data)
	h := res.State.Heroes[0]
	if res.State.QuestID != lower.ID || res.Event.Summary != "Traveled to Lower Vaults" || h.Body != 2 || h.X != 3 || h.Y != 2 || len(res.State.OtherMaps) != 1 {
		t.Fatalf("after travel: %+v / %q / hero %+v", res.State.QuestID, res.Event.Summary, h)
	}

	// Both chapters count as in progress for this session.
	_, body := get(t, client, srv.URL+campaignURL)
	if strings.Count(body, ">in progress<") != 2 {
		t.Fatal("both maps of a traveling session should show as in progress")
	}

	code, data = c(http.MethodPost, "/api/sessions/"+sess.ID+"/travel", map[string]any{"questId": upperID})
	if code != http.StatusOK || !strings.Contains(string(data), "Returned to The Trial") {
		t.Fatalf("travel back: %d %s", code, data)
	}
	if code, _ := c(http.MethodPost, "/api/sessions/"+sess.ID+"/travel", map[string]any{"questId": upperID}); code != http.StatusBadRequest {
		t.Fatalf("travel to the current map: %d", code)
	}
	if code, _ := c(http.MethodPost, "/api/sessions/"+sess.ID+"/travel", map[string]any{"questId": "01900000-0000-7000-8000-000000000000"}); code != http.StatusNotFound {
		t.Fatalf("travel to a missing quest: %d", code)
	}
}
