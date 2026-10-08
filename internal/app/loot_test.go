package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

func TestCampaignLootForms(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "Three Plagues"})
	camp := decodeAny[CampaignResponse](t, data)
	page := srv.URL + "/campaigns/" + camp.ID
	loot := func() []tracker.Item {
		t.Helper()
		code, data := c(http.MethodGet, "/api/campaigns/"+camp.ID+"/loot", nil)
		if code != http.StatusOK {
			t.Fatalf("loot API: %d %s", code, data)
		}
		return decodeAny[[]tracker.Item](t, data)
	}
	if got := loot(); len(got) != 0 {
		t.Fatalf("a new campaign has no loot: %+v", got)
	}

	post := func(u string, form url.Values, want int) string {
		t.Helper()
		resp, body := postForm(t, client, u, form)
		if resp.StatusCode != want {
			t.Fatalf("POST %s %v: %d, want %d", u, form, resp.StatusCode, want)
		}
		if want == http.StatusSeeOther && resp.Header.Get("Location") != "/campaigns/"+camp.ID+"#loot" {
			t.Fatalf("redirect to %s", resp.Header.Get("Location"))
		}
		return body
	}
	post(page+"/loot", url.Values{"name": {"Healing Potion"}, "kind": {"potion"}, "heal_body": {"6"}, "notes": {"Drinking is free."}}, http.StatusSeeOther)
	post(page+"/loot", url.Values{"name": {"Wardens' Longbow"}, "kind": {"bow"}, "damage": {"6"}}, http.StatusSeeOther)
	// The same name again replaces the entry.
	post(page+"/loot", url.Values{"name": {"healing potion"}, "kind": {"potion"}, "heal_body": {"8"}}, http.StatusSeeOther)
	got := loot()
	if len(got) != 2 || got[0].ID != "loot-1" || got[0].Name != "healing potion" || got[0].HealBody != 8 || got[1].Damage != 6 {
		t.Fatalf("loot after the forms: %+v", got)
	}
	if _, body := get(t, client, page); !strings.Contains(body, `id="loot"`) || !strings.Contains(body, "heals 8 Body") || !strings.Contains(body, "Wardens&#39; Longbow") {
		t.Fatal("the campaign page lists the loot")
	}
	if body := post(page+"/loot", url.Values{"name": {"Elixir"}, "heal_body": {"100"}}, http.StatusBadRequest); !strings.Contains(body, `role="alert"`) {
		t.Fatal("too much healing shows an error")
	}
	post(page+"/loot/loot-2/delete", url.Values{}, http.StatusSeeOther)
	if got := loot(); len(got) != 1 || got[0].Name != "healing potion" {
		t.Fatalf("after removing: %+v", got)
	}
	if code, _ := c(http.MethodGet, "/api/campaigns/01900000-0000-7000-8000-000000000000/loot", nil); code != http.StatusNotFound {
		t.Fatalf("unknown campaign: %d", code)
	}
}

func TestCampaignHeroItemFromLoot(t *testing.T) {
	srv := testServer(t)
	client := noRedirects()
	c := func(method, path string, body any) (int, []byte) { return call(t, srv, method, path, body) }
	_, data := c(http.MethodPost, "/api/campaigns", map[string]any{"name": "Three Plagues"})
	camp := decodeAny[CampaignResponse](t, data)
	_, data = c(http.MethodPut, "/api/campaigns/"+camp.ID, map[string]any{"name": "Three Plagues", "heroes": []tracker.CampaignHero{{Name: "Derrick", Class: "elf"}}})
	camp = decodeAny[CampaignResponse](t, data)
	page := srv.URL + "/campaigns/" + camp.ID
	heroURL := page + "/heroes/" + camp.Heroes[0].ID + "/items"
	for _, form := range []url.Values{
		{"name": {"Healing Potion"}, "kind": {"potion"}, "heal_body": {"8"}, "notes": {"Drinking is free."}},
		{"name": {"Wardens' Longbow"}, "kind": {"bow"}, "damage": {"6"}},
	} {
		if resp, _ := postForm(t, client, page+"/loot", form); resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("add loot: %d", resp.StatusCode)
		}
	}
	if _, body := get(t, client, page); !strings.Contains(body, `name="loot_id"`) || !strings.Contains(body, "Add from the loot list") {
		t.Fatal("each hero's inventory offers the loot list")
	}

	// Two from the loot list: its kind, stats, use and notes come along, unequipped.
	resp, _ := postForm(t, client, heroURL, url.Values{"loot_id": {"loot-1"}, "quantity": {"2"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/campaigns/"+camp.ID+"#inventory" {
		t.Fatalf("add from loot: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	// A custom item still works beside it.
	if resp, _ := postForm(t, client, heroURL, url.Values{"name": {"Rope"}, "quantity": {"1"}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("custom item: %d", resp.StatusCode)
	}
	_, data = c(http.MethodGet, "/api/campaigns/"+camp.ID, nil)
	items := decodeAny[CampaignResponse](t, data).Heroes[0].Items
	if len(items) != 2 {
		t.Fatalf("items: %+v", items)
	}
	potion := items[0]
	if potion.Name != "Healing Potion" || potion.Quantity != 2 || potion.Kind != "potion" || potion.HealBody != 8 || potion.Notes != "Drinking is free." || potion.Equipped {
		t.Fatalf("the loot item: %+v", potion)
	}
	if items[1].Name != "Rope" {
		t.Fatalf("the custom item: %+v", items[1])
	}

	resp, body := postForm(t, client, heroURL, url.Values{"loot_id": {"loot-9"}, "quantity": {"1"}})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, `role="alert"`) {
		t.Fatalf("an unknown loot item: %d", resp.StatusCode)
	}
}
