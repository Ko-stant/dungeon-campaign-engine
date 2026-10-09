package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// rogueForm is a class with two abilities and a blank ability row, as the
// form posts it.
func rogueForm() url.Values {
	return url.Values{
		"name": {"Rogue"}, "color": {"#334455"}, "description": {"Quick hands."},
		"body": {"30"}, "mind": {"4"}, "attack": {"1d8 + 1"}, "defense": {"1d6"}, "movement": {"2d6"},
		"accuracy": {"2"}, "mana": {"0"}, "exclusives": {"disarm"},
		"crit_from": {"15"}, "damage": {"1"}, "avoidance": {"4"}, "mitigation": {"0"}, "mana_regen": {"0"},
		"ability_id":       {"", "", ""},
		"ability_name":     {"Nimble Fingers", "Fan of Blades", ""},
		"ability_kind":     {"passive", "active", "active"},
		"ability_mana":     {"", "0", ""},
		"ability_cooldown": {"", "3", ""},
		"ability_text":     {"Disarms on 1d8; fails only on a 1.", "Weapon damage to every enemy around.", ""},
	}
}

func customClasses(t *testing.T, c func(method, path string, body any) (int, []byte)) []content.HeroDef {
	t.Helper()
	_, data := c(http.MethodGet, "/api/catalog", nil)
	cat := decodeAny[content.Catalog](t, data)
	var out []content.HeroDef
	for _, h := range cat.Heroes {
		if h.Custom {
			out = append(out, h)
		}
	}
	return out
}

func TestCustomHeroClassPages(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }

	code, body := get(t, client, srv.URL+"/classes")
	if code != http.StatusOK || !strings.Contains(body, `href="/classes/new"`) {
		t.Fatalf("classes page: %d", code)
	}
	if code, body = get(t, client, srv.URL+"/classes/new"); code != http.StatusOK || !strings.Contains(body, `action="/classes"`) {
		t.Fatalf("new class page: %d", code)
	}

	resp, _ := postForm(t, client, srv.URL+"/classes", rogueForm())
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/classes" {
		t.Fatalf("create: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	list := customClasses(t, c)
	if len(list) != 1 {
		t.Fatalf("catalog custom classes: %+v", list)
	}
	rogue := list[0]
	if !strings.HasPrefix(rogue.ID, "custom-") || rogue.Name != "Rogue" || rogue.Color != "#334455" ||
		rogue.Description != "Quick hands." || rogue.Body != 30 || rogue.Mind != 4 ||
		rogue.AttackDice != "1d8+1" || rogue.DefenseDice != "1d6" || rogue.Movement != "2d6" ||
		rogue.Accuracy != 2 || rogue.Mana != 0 || len(rogue.Exclusives) != 1 || rogue.Exclusives[0] != "disarm" ||
		rogue.CritFrom != 15 || rogue.Damage != 1 || rogue.Avoidance != 4 || rogue.Mitigation != 0 || rogue.ManaRegen != 0 {
		t.Fatalf("custom class in catalog: %+v", rogue)
	}
	want := []content.Ability{
		{ID: "ability-1", Name: "Nimble Fingers", Kind: "passive", Text: "Disarms on 1d8; fails only on a 1."},
		{ID: "ability-2", Name: "Fan of Blades", Kind: "active", Cooldown: 3, Text: "Weapon damage to every enemy around."},
	}
	if len(rogue.Abilities) != len(want) || rogue.Abilities[0] != want[0] || rogue.Abilities[1] != want[1] {
		t.Fatalf("abilities: %+v", rogue.Abilities)
	}

	dbID := strings.TrimPrefix(rogue.ID, "custom-")
	if code, body = get(t, client, srv.URL+"/classes/"+dbID); code != http.StatusOK ||
		!strings.Contains(body, "Fan of Blades") || !strings.Contains(body, `value="1d8+1"`) {
		t.Fatalf("edit page: %d", code)
	}
	if code, _ = get(t, client, srv.URL+"/classes/not-an-id"); code != http.StatusNotFound {
		t.Fatalf("unknown class page: %d", code)
	}
	if _, body = get(t, client, srv.URL+"/classes"); !strings.Contains(body, "Rogue") {
		t.Fatal("the list should show the new class")
	}

	for name, change := range map[string][2]string{
		"no name":          {"name", " "},
		"bad color":        {"color", "red"},
		"bad dice":         {"attack", "1d7"},
		"missing dice":     {"defense", ""},
		"negative body":    {"body", "-1"},
		"no body":          {"body", "0"},
		"too much mana":    {"mana", "1000"},
		"crit too low":     {"crit_from", "1"},
		"crit too high":    {"crit_from", "21"},
		"negative damage":  {"damage", "-1"},
		"too much regen":   {"mana_regen", "100"},
		"bad exclusive":    {"exclusives", "flying"},
		"description long": {"description", strings.Repeat("x", 2001)},
	} {
		form := rogueForm()
		form.Set(change[0], change[1])
		resp, body := postForm(t, client, srv.URL+"/classes", form)
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `role="alert"`) {
			t.Errorf("%s: status %d", name, resp.StatusCode)
		}
	}
	for name, change := range map[string][2]string{
		"text without name": {"ability_name", ""},
		"bad kind":          {"ability_kind", "sneaky"},
		"bad cooldown":      {"ability_cooldown", "-1"},
		"bad mana":          {"ability_mana", "lots"},
	} {
		form := rogueForm()
		form[change[0]][1] = change[1]
		resp, body := postForm(t, client, srv.URL+"/classes", form)
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `role="alert"`) {
			t.Errorf("ability %s: status %d", name, resp.StatusCode)
		}
		if !strings.Contains(body, "Fan of Blades") && name != "text without name" {
			t.Errorf("ability %s: the form should keep what was typed", name)
		}
	}
	if len(customClasses(t, c)) != 1 {
		t.Fatal("rejected forms must not create classes")
	}

	// Editing keeps ability ids; a new ability gets the next free id, and a
	// cleared row drops its ability.
	form := rogueForm()
	form.Set("name", "Shadow Rogue")
	form["ability_id"] = []string{"ability-1", "ability-2", ""}
	form["ability_name"] = []string{"", "Fan of Blades", "Vanish From Sight"}
	form["ability_text"] = []string{"", "Weapon damage to every enemy around.", "Cannot be targeted."}
	form["ability_kind"] = []string{"passive", "active", "reaction"}
	form["ability_cooldown"] = []string{"", "3", "5"}
	resp, _ = postForm(t, client, srv.URL+"/classes/"+dbID, form)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("update: %d", resp.StatusCode)
	}
	list = customClasses(t, c)
	if len(list) != 1 || list[0].Name != "Shadow Rogue" || list[0].ID != rogue.ID {
		t.Fatalf("after update: %+v", list)
	}
	if got := list[0].Abilities; len(got) != 2 || got[0].ID != "ability-2" || got[1].ID != "ability-3" ||
		got[1].Kind != "reaction" || got[1].Cooldown != 5 {
		t.Fatalf("abilities after update: %+v", got)
	}

	// A campaign hero can take the class, and a session starts with its Body and Mind.
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "Three Plagues"})
	camp := decodeAny[CampaignResponse](t, data)
	code, data = c(http.MethodPut, "/api/campaigns/"+camp.ID, map[string]any{
		"name": "Three Plagues", "heroes": []tracker.CampaignHero{{Name: "Vex", Class: rogue.ID}},
	})
	if code != http.StatusOK {
		t.Fatalf("hero with a custom class: %d %s", code, data)
	}
	if _, body = get(t, client, srv.URL+"/campaigns/"+camp.ID); !strings.Contains(body, "Shadow Rogue") {
		t.Fatal("the campaign page should name the custom class")
	}
	questID := setupQuest(t, urlServer{srv.URL}, c)
	code, data = c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("start session: %d %s", code, data)
	}
	sess := decodeAny[SessionResponse](t, data)
	if h := sess.State.Heroes[0]; h.Body != 30 || h.MaxBody != 30 || h.Mind != 4 {
		t.Fatalf("session hero: %+v", h)
	}

	// Classes are deactivated, never deleted.
	if resp, _ = postForm(t, client, srv.URL+"/classes/"+dbID+"/delete", url.Values{}); resp.StatusCode == http.StatusSeeOther {
		t.Fatal("classes can no longer be deleted")
	}
	picker := `<option value="` + rogue.ID + `">`
	if _, body = get(t, client, srv.URL+"/campaigns/"+camp.ID); !strings.Contains(body, picker) {
		t.Fatal("an active class should be offered for new heroes")
	}
	resp, _ = postForm(t, client, srv.URL+"/classes/"+dbID+"/deactivate", url.Values{})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/classes" {
		t.Fatalf("deactivate: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	_, body = get(t, client, srv.URL+"/campaigns/"+camp.ID)
	if strings.Contains(body, picker) {
		t.Error("a deactivated class should not be offered for new heroes")
	}
	if !strings.Contains(body, "Shadow Rogue") {
		t.Error("a hero who has the deactivated class keeps it")
	}
	if list = customClasses(t, c); len(list) != 1 || !list[0].Inactive {
		t.Fatalf("the catalog keeps a deactivated class, marked: %+v", list)
	}
	for _, page := range []string{"/classes", "/classes/" + dbID} {
		if _, body = get(t, client, srv.URL+page); !strings.Contains(body, `action="/classes/`+dbID+`/reactivate"`) {
			t.Errorf("%s should offer to reactivate the class", page)
		}
	}
	if resp, _ = postForm(t, client, srv.URL+"/classes/"+dbID+"/reactivate", url.Values{}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("reactivate: %d", resp.StatusCode)
	}
	if _, body = get(t, client, srv.URL+"/campaigns/"+camp.ID); !strings.Contains(body, picker) {
		t.Error("a reactivated class should be offered again")
	}
	if _, body = get(t, client, srv.URL+"/classes/"+dbID); !strings.Contains(body, `action="/classes/`+dbID+`/deactivate"`) {
		t.Error("the class page should offer to deactivate the class")
	}
	if resp, _ = postForm(t, client, srv.URL+"/classes/01a0ea8a-e8d5-7517-94a0-7176179e5bb3/deactivate", url.Values{}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("deactivate a missing class: %d", resp.StatusCode)
	}
}

func TestCustomHeroClassUpdateUnknown(t *testing.T) {
	srv := testServer(t)
	resp, _ := postForm(t, noRedirects(), srv.URL+"/classes/0190c6a0-0000-7000-8000-000000000000", rogueForm())
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("update unknown class: %d", resp.StatusCode)
	}
}

func TestCustomHeroClassCombatDefaults(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }

	// A form without the combat fields (an older page) crits on a 20 and adds nothing.
	form := rogueForm()
	for _, f := range []string{"crit_from", "damage", "avoidance", "mitigation", "mana_regen"} {
		form.Del(f)
	}
	if resp, _ := postForm(t, client, srv.URL+"/classes", form); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create: %d", resp.StatusCode)
	}
	list := customClasses(t, c)
	if len(list) != 1 || list[0].CritFrom != 20 || list[0].Damage != 0 || list[0].Avoidance != 0 {
		t.Fatalf("defaults: %+v", list)
	}
	if _, body := get(t, client, srv.URL+"/classes"); !strings.Contains(body, "Hit 1d8+1 +2") || !strings.Contains(body, "Crit 20") {
		t.Fatalf("class list summary: %s", body)
	}
}
