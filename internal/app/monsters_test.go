package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

func ogreForm() url.Values {
	return url.Values{
		"name": {"Cave Ogre"}, "color": {"#aa3300"}, "width": {"2"}, "height": {"2"},
		"body": {"6"}, "mind": {"1"}, "attack": {"4"}, "defense": {"3"}, "movement": {"6"}, "notes": {"Smashes doors."},
	}
}

func customMonsters(t *testing.T, srv interface{ Close() }, callFn func(method, path string, body any) (int, []byte)) []content.MonsterDef {
	t.Helper()
	_, data := callFn(http.MethodGet, "/api/catalog", nil)
	cat := decodeAny[content.Catalog](t, data)
	var out []content.MonsterDef
	for _, m := range cat.Monsters {
		if m.Custom {
			out = append(out, m)
		}
	}
	return out
}

func TestCustomMonsterLibrary(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }

	code, body := get(t, client, srv.URL+"/monsters")
	if code != http.StatusOK || !strings.Contains(body, `action="/monsters"`) {
		t.Fatalf("monsters page: %d", code)
	}

	resp, _ := postForm(t, client, srv.URL+"/monsters", ogreForm())
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/monsters" {
		t.Fatalf("create: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}

	list := customMonsters(t, srv, c)
	if len(list) != 1 {
		t.Fatalf("catalog custom monsters: %+v", list)
	}
	ogre := list[0]
	if !strings.HasPrefix(ogre.ID, "custom-") || ogre.Name != "Cave Ogre" || ogre.Width != 2 || ogre.Height != 2 ||
		ogre.Color != "#aa3300" || ogre.Body != 6 || ogre.Attack != 4 || ogre.Notes != "Smashes doors." {
		t.Fatalf("custom monster in catalog: %+v", ogre)
	}
	if _, body = get(t, client, srv.URL+"/monsters"); !strings.Contains(body, "Cave Ogre") {
		t.Fatal("the page should list the new monster")
	}

	for name, change := range map[string][2]string{
		"no name":      {"name", "  "},
		"bad color":    {"color", "red"},
		"too wide":     {"width", "9"},
		"negative":     {"body", "-1"},
		"not a number": {"mind", "lots"},
	} {
		form := ogreForm()
		form.Set(change[0], change[1])
		resp, body := postForm(t, client, srv.URL+"/monsters", form)
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `role="alert"`) {
			t.Errorf("%s: status %d", name, resp.StatusCode)
		}
	}

	dbID := strings.TrimPrefix(ogre.ID, "custom-")
	form := ogreForm()
	form.Set("name", "Cave Ogre Chief")
	form.Set("width", "3")
	resp, _ = postForm(t, client, srv.URL+"/monsters/"+dbID, form)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("update: %d", resp.StatusCode)
	}
	if list = customMonsters(t, srv, c); len(list) != 1 || list[0].Name != "Cave Ogre Chief" || list[0].Width != 3 || list[0].ID != ogre.ID {
		t.Fatalf("after update: %+v", list)
	}

	// A session can bring the custom monster onto the board with its size and color.
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "C"})
	camp := decodeAny[CampaignResponse](t, data)
	_, data = c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID, "name": "Night"})
	sess := decodeAny[SessionResponse](t, data)
	code, data = c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "monster.add", "payload": map[string]any{"type": ogre.ID, "x": 1, "y": 1}})
	if code != http.StatusOK {
		t.Fatalf("add custom monster: %d %s", code, data)
	}
	res := decodeAny[CommandResponse](t, data)
	added := res.State.Monsters[len(res.State.Monsters)-1]
	if added.Width != 3 || added.Height != 2 || added.Color != "#aa3300" || added.Name != "Cave Ogre Chief" {
		t.Fatalf("session monster: %+v", added)
	}

	resp, _ = postForm(t, client, srv.URL+"/monsters/"+dbID+"/delete", url.Values{})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	if list = customMonsters(t, srv, c); len(list) != 0 {
		t.Fatalf("after delete: %+v", list)
	}
	resp, _ = postForm(t, client, srv.URL+"/monsters/"+dbID+"/delete", url.Values{})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete: %d", resp.StatusCode)
	}
}
