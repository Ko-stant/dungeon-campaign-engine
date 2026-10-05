package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func orcStatsForm() url.Values {
	return url.Values{
		"type": {"orc"}, "body": {"22"}, "avoidance": {"8"}, "hit_dice": {"2d8"}, "damage": {"9"},
		"line": {"0"}, "splash_damage": {"0"}, "splash_targets": {"0"}, "ranged": {"on"},
	}
}

func TestCampaignMonsterStatsForms(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "Three Plagues"})
	camp := decodeAny[CampaignResponse](t, data)
	c(http.MethodPut, "/api/campaigns/"+camp.ID, map[string]any{"name": "Three Plagues", "heroes": []map[string]any{{"name": "Faelyn", "class": "elf"}}})
	page := srv.URL + "/campaigns/" + camp.ID

	withAbilities := orcStatsForm()
	withAbilities.Set("abilities", "  Shoots from the shadows.  ")
	resp, _ := postForm(t, client, page+"/monsters", withAbilities)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/campaigns/"+camp.ID+"#monster-stats" {
		t.Fatalf("save stats: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if code, body := get(t, client, page); code != http.StatusOK || !strings.Contains(body, "Avoid 8 · Hit 2d8 · Damage 9 · ranged") || !strings.Contains(body, "Shoots from the shadows.") {
		t.Fatalf("campaign page: %d", code)
	}

	for name, change := range map[string][2]string{
		"unknown type":   {"type", "dragon"},
		"no body":        {"body", "0"},
		"bad dice":       {"hit_dice", "1d7"},
		"too much harm":  {"damage", "100"},
		"long line":      {"line", "4"},
		"many targets":   {"splash_targets", "9"},
		"bad avoidance":  {"avoidance", "-2"},
		"long abilities": {"abilities", strings.Repeat("x", 501)},
	} {
		form := orcStatsForm()
		form.Set(change[0], change[1])
		resp, body := postForm(t, client, page+"/monsters", form)
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `role="alert"`) {
			t.Errorf("%s: status %d", name, resp.StatusCode)
		}
	}

	// Saving the same type again replaces its stats.
	form := orcStatsForm()
	form.Set("body", "25")
	form.Del("ranged")
	form.Set("undead", "on")
	if resp, _ := postForm(t, client, page+"/monsters", form); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("update stats: %d", resp.StatusCode)
	}
	if _, body := get(t, client, page); !strings.Contains(body, "Avoid 8 · Hit 2d8 · Damage 9 · undead") || strings.Contains(body, "· ranged") {
		t.Fatal("the page should show the replaced stats")
	}

	// A new session in this campaign uses them.
	code, data := c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID})
	if code != http.StatusCreated {
		t.Fatalf("start session: %d %s", code, data)
	}
	sess := decodeAny[SessionResponse](t, data)
	orc := sess.State.Monsters[0]
	if orc.Body != 25 || orc.Combat == nil || orc.Combat.Avoidance != 8 || !orc.Combat.Undead {
		t.Fatalf("session orc: %+v %+v", orc, orc.Combat)
	}
	// Monsters added mid-game do too.
	code, data = c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "monster.add", "payload": map[string]any{"type": "orc", "x": 2, "y": 2}})
	if code != http.StatusOK {
		t.Fatalf("add monster: %d %s", code, data)
	}
	res := decodeAny[CommandResponse](t, data)
	if added := res.State.Monsters[len(res.State.Monsters)-1]; added.Combat == nil || added.Body != 25 {
		t.Fatalf("added orc: %+v", added)
	}

	if resp, _ := postForm(t, client, page+"/monsters/orc/delete", url.Values{}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete stats: %d", resp.StatusCode)
	}
	if _, body := get(t, client, page); strings.Contains(body, "Hit 2d8") {
		t.Fatal("deleted stats should be gone")
	}
	if code, _ := get(t, client, srv.URL+"/campaigns/0190c6a0-0000-7000-8000-000000000000/monsters"); code != http.StatusMethodNotAllowed && code != http.StatusNotFound {
		t.Fatalf("GET on the form route: %d", code)
	}
}
