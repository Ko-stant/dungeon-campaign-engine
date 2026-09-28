package tracker

import (
	"strings"
	"testing"
)

func TestAddItemMergesByName(t *testing.T) {
	items, it, err := AddItem(nil, "  Healing Potion ", 2, "Heals 1d6")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || it.ID != "item-1" || it.Name != "Healing Potion" || it.Quantity != 2 || it.Notes != "Heals 1d6" {
		t.Fatalf("first add: %+v %+v", items, it)
	}
	items, it, err = AddItem(items, "healing potion", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || it.ID != "item-1" || it.Quantity != 3 || it.Notes != "Heals 1d6" {
		t.Fatalf("merged add: %+v %+v", items, it)
	}
	items, it, err = AddItem(items, "Rope", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || it.ID != "item-2" || it.Quantity != 1 {
		t.Fatalf("quantity 0 means 1: %+v", it)
	}
	for name, args := range map[string]struct {
		name  string
		qty   int
		notes string
	}{
		"no name":       {" ", 1, ""},
		"negative":      {"Rope", -1, ""},
		"too many":      {"Rope", MaxItemQuantity + 1, ""},
		"long name":     {strings.Repeat("x", MaxItemName+1), 1, ""},
		"long notes":    {"Rope", 1, strings.Repeat("x", MaxItemNotes+1)},
		"merge too big": {"Healing Potion", MaxItemQuantity, ""},
	} {
		if _, _, err := AddItem(items, args.name, args.qty, args.notes); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestRemoveAndUpdateItem(t *testing.T) {
	items := []Item{{ID: "item-1", Name: "Healing Potion", Quantity: 3}, {ID: "item-2", Name: "Rope", Quantity: 1}}

	out, it, removed, err := RemoveItem(items, "item-1", 1)
	if err != nil || removed != 1 || it.Quantity != 2 || len(out) != 2 || out[0].Quantity != 2 {
		t.Fatalf("remove one: %+v %+v %d %v", out, it, removed, err)
	}
	if items[0].Quantity != 3 {
		t.Fatal("RemoveItem modified its input")
	}
	out, it, removed, err = RemoveItem(out, "item-1", 0)
	if err != nil || removed != 2 || it.Quantity != 0 || len(out) != 1 || out[0].ID != "item-2" {
		t.Fatalf("remove all: %+v %+v %d %v", out, it, removed, err)
	}
	if out, _, removed, _ = RemoveItem(items, "item-2", 5); removed != 1 || len(out) != 1 {
		t.Fatalf("removing more than there is removes the item: %+v %d", out, removed)
	}
	if _, _, _, err = RemoveItem(items, "item-9", 1); err == nil {
		t.Fatal("unknown item")
	}

	name, qty, notes := "Greater Healing Potion", 5, "Heals 2d6"
	out, it, err = UpdateItem(items, "item-1", &name, &qty, &notes)
	if err != nil || it.Name != name || it.Quantity != 5 || it.Notes != notes || out[0] != it {
		t.Fatalf("update: %+v %v", it, err)
	}
	if items[0].Name != "Healing Potion" {
		t.Fatal("UpdateItem modified its input")
	}
	zero := 0
	if _, _, err = UpdateItem(items, "item-1", nil, &zero, nil); err == nil {
		t.Fatal("quantity 0 is a removal, not an update")
	}
	blank := " "
	if _, _, err = UpdateItem(items, "item-1", &blank, nil, nil); err == nil {
		t.Fatal("blank name")
	}
}

func TestNormalizeItems(t *testing.T) {
	out, err := NormalizeItems([]Item{{Name: " Rope "}, {ID: "item-4", Name: "Torch", Quantity: 2}, {ID: "item-4", Name: "Map", Quantity: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if out[0].ID != "item-5" || out[0].Name != "Rope" || out[0].Quantity != 1 || out[1].ID != "item-4" || out[2].ID != "item-6" {
		t.Fatalf("normalized: %+v", out)
	}
	if out, err = NormalizeItems(nil); err != nil || out == nil || len(out) != 0 {
		t.Fatalf("nil items: %+v %v", out, err)
	}
	if _, err := NormalizeItems([]Item{{Name: ""}}); err == nil {
		t.Fatal("an item needs a name")
	}
}

func TestItemCommands(t *testing.T) {
	s := newState(t)

	s, ev := apply(t, s, cmd(t, "item.add", map[string]any{"heroId": "hero-1", "name": "Healing Potion", "quantity": 2}))
	if it := s.Heroes[0].Items; len(it) != 1 || it[0].Quantity != 2 || ev.Summary != "Grom gained 2 Healing Potion (now 2)" {
		t.Fatalf("add: %+v %q", it, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "item.add", map[string]any{"heroId": "hero-1", "name": "Rope"}))
	if ev.Summary != "Grom gained Rope" {
		t.Fatalf("add one: %q", ev.Summary)
	}

	s, ev = apply(t, s, cmd(t, "item.remove", map[string]any{"heroId": "hero-1", "itemId": "item-1", "quantity": 1}))
	if s.Heroes[0].Items[0].Quantity != 1 || ev.Summary != "Grom lost 1 Healing Potion (1 left)" {
		t.Fatalf("remove one: %+v %q", s.Heroes[0].Items, ev.Summary)
	}

	s, ev = apply(t, s, cmd(t, "item.give", map[string]any{"heroId": "hero-1", "itemId": "item-1", "toHeroId": "hero-2"}))
	if len(s.Heroes[0].Items) != 1 || len(s.Heroes[1].Items) != 1 || s.Heroes[1].Items[0].Name != "Healing Potion" ||
		ev.Summary != "Grom gave 1 Healing Potion to Ilsa" {
		t.Fatalf("give: %+v / %+v %q", s.Heroes[0].Items, s.Heroes[1].Items, ev.Summary)
	}

	s, ev = apply(t, s, cmd(t, "item.update", map[string]any{"heroId": "hero-1", "itemId": "item-2", "name": "Elven Rope", "notes": "50 ft"}))
	if it := s.Heroes[0].Items[0]; it.Name != "Elven Rope" || it.Notes != "50 ft" || ev.Summary != "Grom: Rope renamed to Elven Rope, notes updated" {
		t.Fatalf("update: %+v %q", it, ev.Summary)
	}

	s, ev = apply(t, s, cmd(t, "item.remove", map[string]any{"heroId": "hero-1", "itemId": "item-2"}))
	if len(s.Heroes[0].Items) != 0 || ev.Summary != "Grom lost Elven Rope" {
		t.Fatalf("remove all: %+v %q", s.Heroes[0].Items, ev.Summary)
	}

	_, _, cat := fixture()
	for name, c := range map[string]Command{
		"unknown hero":      cmd(t, "item.add", map[string]any{"heroId": "hero-9", "name": "Rope"}),
		"no name":           cmd(t, "item.add", map[string]any{"heroId": "hero-1", "name": ""}),
		"unknown item":      cmd(t, "item.remove", map[string]any{"heroId": "hero-1", "itemId": "item-9"}),
		"give to self":      cmd(t, "item.give", map[string]any{"heroId": "hero-2", "itemId": "item-1", "toHeroId": "hero-2"}),
		"give to no one":    cmd(t, "item.give", map[string]any{"heroId": "hero-2", "itemId": "item-1", "toHeroId": "hero-9"}),
		"update to nothing": cmd(t, "item.update", map[string]any{"heroId": "hero-2", "itemId": "item-1"}),
	} {
		if _, _, err := Apply(s, c, cat); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestCarryOverKeepsItems(t *testing.T) {
	s := newState(t)
	s, _ = apply(t, s, cmd(t, "item.add", map[string]any{"heroId": "hero-1", "name": "Soul Gem"}))
	out := s.CarryOver(party())
	if len(out[0].Items) != 1 || out[0].Items[0].Name != "Soul Gem" {
		t.Fatalf("carried items: %+v", out[0].Items)
	}

	// Items on the campaign hero start the next session.
	b, q, cat := fixture()
	next, err := NewSession(b, q, "Next", out, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Heroes[0].Items) != 1 || next.Heroes[0].Items[0].ID != "item-1" {
		t.Fatalf("next session items: %+v", next.Heroes[0].Items)
	}
	if next.Heroes[1].Items == nil {
		t.Fatal("heroes without items get an empty list")
	}
}

func TestGoldChange(t *testing.T) {
	for in, want := range map[string]int{"+25": 55, " - 10 ": 20, "40": 40, "0": 0, "-30": 0} {
		got, err := GoldChange(30, in)
		if err != nil || got != want {
			t.Errorf("GoldChange(30, %q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "+", "abc", "-31", "+1000000", "1.5"} {
		if _, err := GoldChange(30, in); err == nil {
			t.Errorf("GoldChange(30, %q) should fail", in)
		}
	}
}
