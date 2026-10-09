package app

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

var (
	deleteForm   = regexp.MustCompile(`(?s)<form[^>]*\saction="([^"]*/(?:delete|remove))"[^>]*>(.*?)</form>`)
	deleteButton = regexp.MustCompile(`<button[^>]*\sformaction="[^"]*/delete"[^>]*>`)
)

// assertDeletesAsk checks that every delete or remove button on a page asks
// first (data-confirm opens the Yes/No dialog), and that none uses the
// browser's own confirm box.
func assertDeletesAsk(t *testing.T, page, body string) {
	t.Helper()
	forms := deleteForm.FindAllStringSubmatch(body, -1)
	buttons := deleteButton.FindAllString(body, -1)
	if len(forms)+len(buttons) == 0 {
		t.Fatalf("%s: no delete or remove buttons found", page)
	}
	for _, f := range forms {
		if !strings.Contains(f[2], `data-confirm="`) {
			t.Errorf("%s: the form posting to %s deletes without asking", page, f[1])
		}
	}
	for _, b := range buttons {
		if !strings.Contains(b, `data-confirm="`) {
			t.Errorf("%s: %s deletes without asking", page, b)
		}
	}
	if strings.Contains(body, "return confirm(") {
		t.Errorf("%s: still uses the browser's confirm box", page)
	}
}

// confirmText returns the data-confirm text of the form posting to action.
func confirmText(t *testing.T, body, action string) string {
	t.Helper()
	for _, f := range deleteForm.FindAllStringSubmatch(body, -1) {
		if f[1] == action {
			m := regexp.MustCompile(`data-confirm="([^"]*)"`).FindStringSubmatch(f[2])
			if m == nil {
				t.Fatalf("the form posting to %s has no data-confirm", action)
			}
			return m[1]
		}
	}
	t.Fatalf("no form posts to %s", action)
	return ""
}

func TestDeleteSessionCampaignAndBoardFromThePages(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodGet, "/api/quests/"+questID, nil)
	boardID := decodeAny[QuestResponse](t, data).BoardID

	post := func(u string, form url.Values, want int) *http.Response {
		t.Helper()
		resp, body := postForm(t, client, srv.URL+u, form)
		if resp.StatusCode != want {
			t.Fatalf("POST %s: %d, want %d: %s", u, resp.StatusCode, want, body)
		}
		return resp
	}
	campaignURL := post("/campaigns", url.Values{"name": {"Winter Campaign"}}, http.StatusSeeOther).Header.Get("Location")
	campaignID := strings.TrimPrefix(campaignURL, "/campaigns/")
	post(campaignURL+"/heroes", url.Values{"name": {"Faelyn"}, "class": {"elf"}}, http.StatusSeeOther)
	post(campaignURL+"/chapters", url.Values{"questId": {questID}}, http.StatusSeeOther)
	post(campaignURL+"/loot", url.Values{"name": {"Healing Potion"}, "kind": {"potion"}, "heal_body": {"6"}}, http.StatusSeeOther)
	post(campaignURL+"/monsters", url.Values{"type": {"orc"}, "body": {"1"}, "hit_dice": {""}}, http.StatusSeeOther)
	playURL := post(campaignURL+"/sessions", url.Values{"questId": {questID}, "name": {"Night one"}}, http.StatusSeeOther).Header.Get("Location")
	sessionID := strings.TrimPrefix(playURL, "/play/")
	if resp, body := uploadClips(t, client, srv.URL+campaignURL+"/audio", map[string]string{"P0-01.mp3": "x"}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("upload: %d %s", resp.StatusCode, body)
	}
	_, page := get(t, client, srv.URL+campaignURL)
	heroURL := regexp.MustCompile(regexp.QuoteMeta(campaignURL) + `/heroes/[^/"]+`).FindString(page)
	post(strings.TrimPrefix(heroURL, srv.URL)+"/items", url.Values{"name": {"Rope"}}, http.StatusSeeOther)

	// Every delete or remove button on the campaign page asks first.
	_, page = get(t, client, srv.URL+campaignURL)
	assertDeletesAsk(t, "campaign page", page)
	warning := confirmText(t, page, campaignURL+"/delete")
	for _, want := range []string{"Winter Campaign", "1 session", "audio"} {
		if !strings.Contains(warning, want) {
			t.Errorf("campaign delete warning %q is missing %q", warning, want)
		}
	}
	if warning := confirmText(t, page, campaignURL+"/sessions/"+sessionID+"/delete"); !strings.Contains(warning, "Night one") {
		t.Errorf("session delete warning %q should name the session", warning)
	}

	// The maps page: deleting a board names its quests and the campaigns using them.
	_, page = get(t, client, srv.URL+"/maps")
	assertDeletesAsk(t, "maps page", page)
	warning = confirmText(t, page, "/maps/"+boardID+"/delete")
	for _, want := range []string{"Board", "The Trial", "Winter Campaign"} {
		if !strings.Contains(warning, want) {
			t.Errorf("board delete warning %q is missing %q", warning, want)
		}
	}

	// Deleting a session.
	post(campaignURL+"/sessions/"+sessionID+"/delete", nil, http.StatusSeeOther)
	if code, _ := get(t, client, srv.URL+playURL); code != http.StatusNotFound {
		t.Errorf("deleted session's page: %d, want 404", code)
	}
	post(campaignURL+"/sessions/"+sessionID+"/delete", nil, http.StatusNotFound)

	// Deleting the campaign takes its audio clips too (rows of audio_clip).
	if code, _ := get(t, client, srv.URL+"/audio/"+campaignID+"/P0-01.mp3"); code != http.StatusOK {
		t.Fatalf("the clip before the delete: %d", code)
	}
	if resp := post(campaignURL+"/delete", nil, http.StatusSeeOther); resp.Header.Get("Location") != "/campaigns" {
		t.Errorf("campaign delete redirects to %q", resp.Header.Get("Location"))
	}
	if code, _ := get(t, client, srv.URL+campaignURL); code != http.StatusNotFound {
		t.Errorf("deleted campaign's page: %d, want 404", code)
	}
	if code, _ := get(t, client, srv.URL+"/audio/"+campaignID+"/P0-01.mp3"); code != http.StatusNotFound {
		t.Errorf("the deleted campaign's clip: %d, want 404", code)
	}
	post(campaignURL+"/delete", nil, http.StatusNotFound)

	// Deleting the board takes its quests.
	if resp := post("/maps/"+boardID+"/delete", nil, http.StatusSeeOther); resp.Header.Get("Location") != "/maps" {
		t.Errorf("board delete redirects to %q", resp.Header.Get("Location"))
	}
	if code, _ := c(http.MethodGet, "/api/boards/"+boardID, nil); code != http.StatusNotFound {
		t.Errorf("deleted board: %d, want 404", code)
	}
	if code, _ := c(http.MethodGet, "/api/quests/"+questID, nil); code != http.StatusNotFound {
		t.Errorf("deleted board's quest: %d, want 404", code)
	}
	post("/maps/"+boardID+"/delete", nil, http.StatusNotFound)
}

func TestMonsterDeleteAsks(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	if resp, body := postForm(t, client, srv.URL+"/monsters", ogreForm()); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create monster: %d %s", resp.StatusCode, body)
	}
	_, page := get(t, client, srv.URL+"/monsters")
	assertDeletesAsk(t, "monsters page", page)
}

func TestDeleteBoardAPIWithQuests(t *testing.T) {
	srv := testServer(t)
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodGet, "/api/quests/"+questID, nil)
	boardID := decodeAny[QuestResponse](t, data).BoardID

	// Without asking for it, a board with quests is refused.
	if code, _ := c(http.MethodDelete, "/api/boards/"+boardID, nil); code != http.StatusConflict {
		t.Fatalf("delete a board with quests: %d, want 409", code)
	}
	if code, _ := c(http.MethodDelete, "/api/boards/"+boardID+"?withQuests=true", nil); code != http.StatusNoContent {
		t.Fatalf("delete with its quests: %d, want 204", code)
	}
	if code, _ := c(http.MethodGet, "/api/quests/"+questID, nil); code != http.StatusNotFound {
		t.Errorf("the board's quest: %d, want 404", code)
	}
}
