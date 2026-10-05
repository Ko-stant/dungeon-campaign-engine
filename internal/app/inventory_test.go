package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

func TestCampaignInventoryForms(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }

	if resp, _ := postForm(t, client, srv.URL+"/classes", rogueForm()); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create class: %d", resp.StatusCode)
	}
	rogue := customClasses(t, c)[0].ID

	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "Three Plagues"})
	camp := decodeAny[CampaignResponse](t, data)
	code, data := c(http.MethodPut, "/api/campaigns/"+camp.ID, map[string]any{"name": "Three Plagues", "gold": 5, "heroes": []tracker.CampaignHero{
		{Name: "Vex", Class: "elf", Items: []tracker.Item{{Name: " Rope "}}},
		{Name: "Bram", Class: rogue, Items: []tracker.Item{{Name: "Dirk", Kind: "weapon", Equipped: true, ItemStats: tracker.ItemStats{Damage: 2}}}},
	}})
	if code != http.StatusOK {
		t.Fatalf("put heroes: %d %s", code, data)
	}
	camp = decodeAny[CampaignResponse](t, data)
	vex := camp.Heroes[0]
	if camp.Gold != 5 || !camp.Heroes[1].Items[0].Equipped || camp.Heroes[1].Items[0].Damage != 2 {
		t.Fatalf("purse and item stats through the API: %+v", camp)
	}
	if len(vex.Items) != 1 || vex.Items[0].ID != "item-1" || vex.Items[0].Name != "Rope" || vex.Items[0].Quantity != 1 {
		t.Fatalf("items through the API are normalized: %+v", vex.Items)
	}
	heroURL := srv.URL + "/campaigns/" + camp.ID + "/heroes/" + vex.ID
	reloadCampaign := func() CampaignResponse {
		t.Helper()
		_, data := c(http.MethodGet, "/api/campaigns/"+camp.ID, nil)
		return decodeAny[CampaignResponse](t, data)
	}
	reload := func() tracker.CampaignHero {
		t.Helper()
		return reloadCampaign().Heroes[0]
	}
	post := func(u string, form url.Values, want int) string {
		t.Helper()
		resp, body := postForm(t, client, u, form)
		if resp.StatusCode != want {
			t.Fatalf("POST %s %v: %d, want %d", u, form, resp.StatusCode, want)
		}
		if want == http.StatusSeeOther && resp.Header.Get("Location") != "/campaigns/"+camp.ID+"#inventory" {
			t.Fatalf("redirect to %s", resp.Header.Get("Location"))
		}
		return body
	}

	goldURL := srv.URL + "/campaigns/" + camp.ID + "/gold"
	post(goldURL, url.Values{"gold": {"+25"}}, http.StatusSeeOther)
	post(goldURL, url.Values{"gold": {"-10"}}, http.StatusSeeOther)
	if g := reloadCampaign().Gold; g != 20 {
		t.Fatalf("party gold: %d", g)
	}
	if body := post(goldURL, url.Values{"gold": {"-100"}}, http.StatusBadRequest); !strings.Contains(body, `role="alert"`) {
		t.Fatal("too little gold should show an error")
	}

	post(heroURL+"/items", url.Values{"name": {"Healing Potion"}, "quantity": {"2"}, "notes": {"Heals 1d6"}}, http.StatusSeeOther)
	post(heroURL+"/items", url.Values{"name": {"healing potion"}, "quantity": {""}}, http.StatusSeeOther)
	h := reload()
	if len(h.Items) != 2 || h.Items[1].ID != "item-2" || h.Items[1].Quantity != 3 || h.Items[1].Notes != "Heals 1d6" {
		t.Fatalf("items after adds: %+v", h.Items)
	}
	post(heroURL+"/items", url.Values{"name": {""}}, http.StatusBadRequest)
	post(heroURL+"/items", url.Values{"name": {"Wardens' Longbow"}, "kind": {"bow"}, "damage": {"6"}, "equipped": {"on"}}, http.StatusSeeOther)
	if bow := reload().Items[2]; bow.Kind != "bow" || bow.Damage != 6 || !bow.Equipped {
		t.Fatalf("item with stats: %+v", bow)
	}
	post(heroURL+"/items", url.Values{"name": {"Cursed Ring"}, "accuracy": {"-100"}}, http.StatusBadRequest)
	post(heroURL+"/items/item-3", url.Values{"name": {"Wardens' Longbow"}, "quantity": {"1"}, "kind": {"longbow"}, "damage": {"7"}, "accuracy": {"1"}}, http.StatusSeeOther)
	if bow := reload().Items[2]; bow.Kind != "longbow" || bow.Damage != 7 || bow.Accuracy != 1 || !bow.Equipped {
		t.Fatalf("item stats after update: %+v", bow)
	}
	post(heroURL+"/items/item-3/equip", url.Values{"equipped": {"false"}}, http.StatusSeeOther)
	if bow := reload().Items[2]; bow.Equipped {
		t.Fatalf("unequipped: %+v", bow)
	}
	post(heroURL+"/items/item-3/equip", url.Values{"equipped": {"true"}}, http.StatusSeeOther)
	post(heroURL+"/items/item-9/equip", url.Values{"equipped": {"true"}}, http.StatusNotFound)
	post(heroURL+"/items/item-3/delete", url.Values{}, http.StatusSeeOther)

	post(heroURL+"/items/item-2", url.Values{"name": {"Healing Potion"}, "quantity": {"1"}, "notes": {"Heals 2d6"}}, http.StatusSeeOther)
	if h = reload(); h.Items[1].Quantity != 1 || h.Items[1].Notes != "Heals 2d6" {
		t.Fatalf("item after update: %+v", h.Items[1])
	}
	post(heroURL+"/items/item-2", url.Values{"name": {"Healing Potion"}, "quantity": {"0"}}, http.StatusBadRequest)

	post(heroURL+"/items/item-1/delete", url.Values{}, http.StatusSeeOther)
	if h = reload(); len(h.Items) != 1 || h.Items[0].ID != "item-2" {
		t.Fatalf("items after delete: %+v", h.Items)
	}
	post(heroURL+"/items/item-9/delete", url.Values{}, http.StatusNotFound)
	post(srv.URL+"/campaigns/"+camp.ID+"/heroes/hero-9/items", url.Values{"name": {"Rope"}}, http.StatusNotFound)

	_, body := get(t, client, srv.URL+"/campaigns/"+camp.ID)
	if !strings.Contains(body, `id="inventory"`) || !strings.Contains(body, "Healing Potion") || !strings.Contains(body, "Heals 2d6") {
		t.Fatal("the campaign page should show the inventory")
	}
	// Bram's totals: the rogue class (damage 1) and an equipped dirk (+2).
	if !strings.Contains(body, "Crit 15-20 · Damage 3 · Avoid 4+1d6") || !strings.Contains(body, "damage +2") {
		t.Fatal("the campaign page should show hero totals and item stats")
	}
	if strings.Contains(body, "in a quest") {
		t.Fatal("no quest is running yet")
	}

	// A quest takes the inventory along and brings it back when completed.
	questID := setupQuest(t, urlServer{srv.URL}, c)
	_, data = c(http.MethodPost, "/api/campaigns/"+camp.ID+"/sessions", map[string]any{"questId": questID})
	sess := decodeAny[SessionResponse](t, data)
	if items := sess.State.Heroes[0].Items; len(items) != 1 || items[0].Name != "Healing Potion" {
		t.Fatalf("session items: %+v", items)
	}
	if sess.State.Gold != 20 {
		t.Fatalf("the session takes the purse: %d", sess.State.Gold)
	}
	if _, body = get(t, client, srv.URL+"/campaigns/"+camp.ID); !strings.Contains(body, "in a quest") {
		t.Fatal("the page should warn that a running quest will overwrite inventory changes")
	}
	code, data = c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "item.add", "payload": map[string]any{"heroId": vex.ID, "name": "Soul Gem"}})
	if code != http.StatusOK {
		t.Fatalf("item.add: %d %s", code, data)
	}
	c(http.MethodPost, "/api/sessions/"+sess.ID+"/commands", map[string]any{"type": "gold.set", "payload": map[string]any{"gold": 104}})
	if code, data = c(http.MethodPost, "/api/sessions/"+sess.ID+"/complete", nil); code != http.StatusOK {
		t.Fatalf("complete: %d %s", code, data)
	}
	after := reloadCampaign()
	if h = after.Heroes[0]; len(h.Items) != 2 || h.Items[1].Name != "Soul Gem" {
		t.Fatalf("carried back: %+v", h.Items)
	}
	if after.Gold != 104 {
		t.Fatalf("the purse is carried back: %d", after.Gold)
	}
}
