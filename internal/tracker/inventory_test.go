package tracker

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

func TestAddItemMergesByName(t *testing.T) {
	items, it, err := AddItem(nil, Item{Name: "  Healing Potion ", Quantity: 2, Notes: "Heals 1d6"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || it.ID != "item-1" || it.Name != "Healing Potion" || it.Quantity != 2 || it.Notes != "Heals 1d6" {
		t.Fatalf("first add: %+v %+v", items, it)
	}
	items, it, err = AddItem(items, Item{Name: "healing potion", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || it.ID != "item-1" || it.Quantity != 3 || it.Notes != "Heals 1d6" {
		t.Fatalf("merged add: %+v %+v", items, it)
	}
	items, it, err = AddItem(items, Item{Name: "Rope"})
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
		if _, _, err := AddItem(items, Item{Name: args.name, Quantity: args.qty, Notes: args.notes}); err == nil {
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
	out, it, err = UpdateItem(items, "item-1", ItemPatch{Name: &name, Quantity: &qty, Notes: &notes})
	if err != nil || it.Name != name || it.Quantity != 5 || it.Notes != notes || out[0] != it {
		t.Fatalf("update: %+v %v", it, err)
	}
	if items[0].Name != "Healing Potion" {
		t.Fatal("UpdateItem modified its input")
	}
	zero := 0
	if _, _, err = UpdateItem(items, "item-1", ItemPatch{Quantity: &zero}); err == nil {
		t.Fatal("quantity 0 is a removal, not an update")
	}
	blank := " "
	if _, _, err = UpdateItem(items, "item-1", ItemPatch{Name: &blank}); err == nil {
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
	if it := s.Heroes[0].Items; len(it) != 1 || it[0].Quantity != 2 || ev.Summary != "Grom gained 2 Healing Potions" {
		t.Fatalf("add: %+v %q", it, ev.Summary)
	}
	// Adding to a stack says how many there are now.
	if _, ev := apply(t, s, cmd(t, "item.add", map[string]any{"heroId": "hero-1", "name": "Healing Potion", "quantity": 3})); ev.Summary != "Grom gained 3 Healing Potions (now 5)" {
		t.Fatalf("add to a stack: %q", ev.Summary)
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

func TestItemStats(t *testing.T) {
	bow := Item{Name: "Wardens' Longbow", Kind: "bow", ItemStats: ItemStats{Damage: 6}}
	items, it, err := AddItem(nil, bow)
	if err != nil || it.Kind != "bow" || it.Damage != 6 || it.Quantity != 1 || it.Equipped {
		t.Fatalf("add with stats: %+v %v", it, err)
	}
	// A plain add of the same name keeps the stats; an add with stats fills in an item without any.
	if _, it, _ = AddItem(items, Item{Name: "wardens' longbow", Kind: "trophy", ItemStats: ItemStats{Damage: 1}}); it.Kind != "bow" || it.Damage != 6 {
		t.Fatalf("merge keeps stats: %+v", it)
	}
	items, _, _ = AddItem(nil, Item{Name: "Rope"})
	if _, it, _ = AddItem(items, Item{Name: "Rope", Kind: "tool", ItemStats: ItemStats{Avoidance: 1}}); it.Kind != "tool" || it.Avoidance != 1 || it.Quantity != 2 {
		t.Fatalf("merge fills stats: %+v", it)
	}
	for name, add := range map[string]Item{
		"long kind":      {Name: "Rope", Kind: strings.Repeat("x", MaxItemKind+1)},
		"huge damage":    {Name: "Rope", ItemStats: ItemStats{Damage: MaxItemStat + 1}},
		"huge curse":     {Name: "Rope", ItemStats: ItemStats{Accuracy: -MaxItemStat - 1}},
		"huge mana":      {Name: "Rope", ItemStats: ItemStats{Mana: MaxItemStat + 1}},
		"huge regen":     {Name: "Rope", ItemStats: ItemStats{ManaRegen: MaxItemStat + 1}},
		"huge avoidance": {Name: "Rope", ItemStats: ItemStats{Avoidance: MaxItemStat + 1}},
		"huge mitigaton": {Name: "Rope", ItemStats: ItemStats{Mitigation: MaxItemStat + 1}},
	} {
		if _, _, err := AddItem(nil, add); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := NormalizeItems([]Item{{Name: "Rope", ItemStats: ItemStats{Damage: 100}}}); err == nil {
		t.Error("NormalizeItems checks stats")
	}

	kind, stats := "two-handed weapon", ItemStats{Damage: 9, Accuracy: -1}
	out, it, err := UpdateItem([]Item{{ID: "item-1", Name: "Greatsword", Quantity: 1, Kind: "weapon"}}, "item-1", ItemPatch{Kind: &kind, Stats: &stats})
	if err != nil || it.Kind != kind || it.ItemStats != stats || out[0] != it {
		t.Fatalf("update stats: %+v %v", it, err)
	}
}

func TestItemStatsSummary(t *testing.T) {
	for want, st := range map[string]ItemStats{
		"":                          {},
		"damage +7":                 {Damage: 7},
		"Accuracy +1, avoidance +2": {Accuracy: 1, Avoidance: 2},
		"mitigation +1":             {Mitigation: 1},
		"mana +2, mana regen +1":    {Mana: 2, ManaRegen: 1},
		"damage -1":                 {Damage: -1},
	} {
		if got := st.Summary(); got != want {
			t.Errorf("%+v: %q, want %q", st, got, want)
		}
	}
}

// gearHero is a hero with combat stats, an equipped greataxe and cuirass, and boots in the pack.
func gearHero() Hero {
	return Hero{
		Name: "Grom", Mana: 4, MaxMana: 4,
		Combat: &Combat{HitDice: "1d20", Accuracy: 3, CritFrom: 17, Damage: 3, DefenseDice: "1d6", Avoidance: 2, Mitigation: 1},
		Items: []Item{
			{ID: "item-1", Name: "Greataxe", Quantity: 1, Equipped: true, ItemStats: ItemStats{Damage: 7}},
			{ID: "item-2", Name: "Hide Cuirass", Quantity: 1, Equipped: true, ItemStats: ItemStats{Mitigation: 1, Mana: 2, ManaRegen: 1}},
			{ID: "item-3", Name: "Iron-shod Boots", Quantity: 1, ItemStats: ItemStats{Avoidance: 1}},
		},
	}
}

func TestHeroTotalsAddEquippedItems(t *testing.T) {
	h := gearHero()
	if g := GearBonus(h.Items); g != (ItemStats{Damage: 7, Mitigation: 1, Mana: 2, ManaRegen: 1}) {
		t.Fatalf("gear: %+v", g)
	}
	tot := h.CombatTotals()
	want := Combat{HitDice: "1d20", Accuracy: 3, CritFrom: 17, Damage: 10, DefenseDice: "1d6", Avoidance: 2, Mitigation: 2, ManaRegen: 1}
	if tot == nil || *tot != want {
		t.Fatalf("totals: %+v", tot)
	}
	if h.Combat.Damage != 3 {
		t.Fatal("CombatTotals modified the class stats")
	}
	if h.ManaCap() != 6 || h.ManaRegen() != 1 {
		t.Fatalf("mana cap %d, regen %d", h.ManaCap(), h.ManaRegen())
	}
	h.Combat = nil
	if h.CombatTotals() != nil {
		t.Fatal("a hero without combat stats has no totals")
	}
}

func TestCombatLine(t *testing.T) {
	c := Combat{HitDice: "1d20", Accuracy: 3, CritFrom: 17, Damage: 10, DefenseDice: "1d6", Avoidance: 3, Mitigation: 2, ManaRegen: 1}
	if got := CombatLine(c); got != "Hit 1d20+3 · Crit 17-20 · Damage 10 · Avoid 3+1d6 · Mitigation 2 · Mana +1 a fight round" {
		t.Fatalf("line: %q", got)
	}
	c = Combat{HitDice: "2d8+1", Accuracy: 4, CritFrom: 20, Damage: 5, DefenseDice: "1d6"}
	if got := CombatLine(c); got != "Hit 2d8+1 +4 · Crit 20 · Damage 5 · Avoid 1d6" {
		t.Fatalf("line: %q", got)
	}
}

func TestNewSessionStartsWithGearMana(t *testing.T) {
	ch := party()[0]
	ch.Class = "custom-cleric"
	ch.Items = []Item{{ID: "item-1", Name: "Prayer Beads", Quantity: 1, Equipped: true, ItemStats: ItemStats{Mana: 2}}}
	b, q, cat := fixture()
	cat.Heroes = append(cat.Heroes, content.HeroDef{ID: "custom-cleric", Name: "Cleric", Body: 28, Mana: 16, ManaRegen: 2, Custom: true, AttackDice: "2d8", DefenseDice: "1d6"})
	next, err := NewSession(b, q, "Q", []CampaignHero{ch}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if h := next.Heroes[0]; h.Mana != 18 || h.MaxMana != 16 || h.ManaCap() != 18 || h.Combat.ManaRegen != 2 {
		t.Fatalf("cleric with beads: mana %d / max %d / cap %d", h.Mana, h.MaxMana, h.ManaCap())
	}
}

func TestItemEquipCommand(t *testing.T) {
	s := newState(t)
	s.Heroes[0] = gearHero()
	s.Heroes[0].ID = "hero-1"

	s, ev := apply(t, s, cmd(t, "item.equip", map[string]any{"heroId": "hero-1", "itemId": "item-3", "equipped": true}))
	if !s.Heroes[0].Items[2].Equipped || ev.Summary != "Grom equipped Iron-shod Boots (avoidance +1)" {
		t.Fatalf("equip: %+v %q", s.Heroes[0].Items[2], ev.Summary)
	}
	// Taking off a mana item lowers mana to the new cap.
	s.Heroes[0].Mana = 6
	s, ev = apply(t, s, cmd(t, "item.equip", map[string]any{"heroId": "hero-1", "itemId": "item-2", "equipped": false}))
	if s.Heroes[0].Items[1].Equipped || s.Heroes[0].Mana != 4 || ev.Summary != "Grom unequipped Hide Cuirass (mana 6 → 4)" {
		t.Fatalf("unequip: %+v %d %q", s.Heroes[0].Items[1], s.Heroes[0].Mana, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "item.equip", map[string]any{"heroId": "hero-1", "itemId": "item-1", "equipped": false}))
	if ev.Summary != "Grom unequipped Greataxe" {
		t.Fatalf("unequip plain: %q", ev.Summary)
	}
	for name, payload := range map[string]map[string]any{
		"already equipped": {"heroId": "hero-1", "itemId": "item-3", "equipped": true},
		"unknown item":     {"heroId": "hero-1", "itemId": "item-9", "equipped": true},
		"unknown hero":     {"heroId": "hero-9", "itemId": "item-1", "equipped": true},
	} {
		if _, _, err := Apply(s, cmd(t, "item.equip", payload), nil); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestItemCommandsCarryStats(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "item.add", map[string]any{"heroId": "hero-1", "name": "Wardens' Greatsword", "kind": "two-handed weapon", "damage": 9}))
	if it := s.Heroes[0].Items[0]; it.Damage != 9 || it.Kind != "two-handed weapon" || ev.Summary != "Grom gained Wardens' Greatsword (damage +9)" {
		t.Fatalf("add with stats: %+v %q", it, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "item.update", map[string]any{"heroId": "hero-1", "itemId": "item-1", "kind": "weapon", "stats": map[string]any{"damage": 8, "accuracy": 1}}))
	if it := s.Heroes[0].Items[0]; it.Damage != 8 || it.Accuracy != 1 || it.Kind != "weapon" ||
		ev.Summary != "Grom: Wardens' Greatsword kind two-handed weapon → weapon, stats damage +9 → damage +8, Accuracy +1" {
		t.Fatalf("update stats: %+v %q", it, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "item.update", map[string]any{"heroId": "hero-1", "itemId": "item-1", "kind": "", "stats": map[string]any{}}))
	if ev.Summary != "Grom: Wardens' Greatsword kind cleared, stats cleared" {
		t.Fatalf("clear: %q", ev.Summary)
	}
	s, _ = apply(t, s, cmd(t, "item.update", map[string]any{"heroId": "hero-1", "itemId": "item-1", "kind": "weapon", "stats": map[string]any{"damage": 9}}))
	s, _ = apply(t, s, cmd(t, "item.equip", map[string]any{"heroId": "hero-1", "itemId": "item-1", "equipped": true}))

	// Given away, the item keeps its kind and stats but the new owner has not equipped it.
	s, _ = apply(t, s, cmd(t, "item.give", map[string]any{"heroId": "hero-1", "itemId": "item-1", "toHeroId": "hero-2"}))
	if len(s.Heroes[0].Items) != 0 {
		t.Fatalf("giver: %+v", s.Heroes[0].Items)
	}
	if it := s.Heroes[1].Items[0]; it.Damage != 9 || it.Kind != "weapon" || it.Equipped {
		t.Fatalf("receiver: %+v", it)
	}
}

func TestFightsRegenerateGearMana(t *testing.T) {
	s, cat := fightState(t)
	s.Heroes[0].Items = []Item{
		{ID: "item-1", Name: "Holy Tome", Quantity: 1, Equipped: true, ItemStats: ItemStats{ManaRegen: 1}},
		{ID: "item-2", Name: "Prayer Beads", Quantity: 1, Equipped: true, ItemStats: ItemStats{Mana: 2}},
	}
	s.Heroes[0].Mana = 10
	s, _ = applyWith(t, s, cmd(t, "fight.start", map[string]any{}), cat)
	s, ev := applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	if s.Heroes[0].Mana != 14 || !strings.Contains(ev.Summary, "Mira 10 → 14") {
		t.Fatalf("regen 3+1: %d %q", s.Heroes[0].Mana, ev.Summary)
	}
	s, _ = applyWith(t, s, cmd(t, "round.advance", map[string]any{}), cat)
	if s.Heroes[0].Mana != 18 {
		t.Fatalf("capped at 16+2: %d", s.Heroes[0].Mana)
	}
}

func TestPartyGold(t *testing.T) {
	s := newState(t)
	s, ev := apply(t, s, cmd(t, "gold.set", map[string]any{"gold": 84}))
	if s.Gold != 84 || ev.Summary != "Party gold 0 → 84" {
		t.Fatalf("set: %d %q", s.Gold, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "gold.set", map[string]any{"gold": 60}))
	if s.Gold != 60 || ev.Summary != "Party gold 84 → 60" {
		t.Fatalf("spend: %d %q", s.Gold, ev.Summary)
	}
	for name, payload := range map[string]map[string]any{
		"negative":       {"gold": -1},
		"too much":       {"gold": MaxGold + 1},
		"nothing change": {"gold": 60},
	} {
		if _, _, err := Apply(s, cmd(t, "gold.set", payload), nil); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, _, err := Apply(s, cmd(t, "hero.update", map[string]any{"id": "hero-1", "gold": 5}), nil); err == nil {
		t.Error("heroes no longer carry their own gold")
	}
}

func TestGoldChange(t *testing.T) {
	for in, want := range map[string]int{"+25": 55, " - 10 ": 20, "=40": 40, "= 0": 0, "-30": 0} {
		got, err := GoldChange(30, in)
		if err != nil || got != want {
			t.Errorf("GoldChange(30, %q) = %d, %v; want %d", in, got, err, want)
		}
	}
	// A bare number is refused: +, - or = says what it does.
	for _, in := range []string{"", "+", "=", "40", "abc", "-31", "+1000000", "1.5", "=-5"} {
		if _, err := GoldChange(30, in); err == nil {
			t.Errorf("GoldChange(30, %q) should fail", in)
		}
	}
}

func TestCountedName(t *testing.T) {
	for _, c := range []struct {
		n          int
		name, want string
	}{
		{1, "Healing Potion", "Healing Potion"},
		{3, "Healing Potion", "3 Healing Potions"},
		{2, "Soft Boots", "2 Soft Boots"},
		{2, "Potion of Healing", "2 Potions of Healing"},
		{2, "Torch", "2 Torches"},
		{2, "Ruby", "2 Rubies"},
		{2, "Key", "2 Keys"},
	} {
		if got := countedName(c.n, c.name); got != c.want {
			t.Errorf("countedName(%d, %q) = %q, want %q", c.n, c.name, got, c.want)
		}
	}
}

func TestItemUse(t *testing.T) {
	s, cat := fightState(t) // Mira: Cleric, 28 Body, 16 mana; Vex: Ranger, 30 Body
	s.Heroes[1].Body = 20
	s.Heroes[0].Mana = 14
	s, ev := applyWith(t, s, cmd(t, "item.add", map[string]any{"heroId": "hero-2", "name": "Healing Potion", "quantity": 2, "healBody": 8}), cat)
	if it := s.Heroes[1].Items[0]; it.HealBody != 8 || ev.Summary != "Vex gained 2 Healing Potions (heals 8 Body)" {
		t.Fatalf("add a potion: %+v %q", it, ev.Summary)
	}
	// Using one spends it and heals, as one change.
	s, ev = applyWith(t, s, cmd(t, "item.use", map[string]any{"heroId": "hero-2", "itemId": "item-1"}), cat)
	if s.Heroes[1].Body != 28 || s.Heroes[1].Items[0].Quantity != 1 || ev.Summary != "Vex used Healing Potion: Body 20 → 28 (1 left)" {
		t.Fatalf("use: %d %+v %q", s.Heroes[1].Body, s.Heroes[1].Items, ev.Summary)
	}
	if ev.PlayerSummary != ev.Summary {
		t.Fatalf("the players hear about it: %q", ev.PlayerSummary)
	}
	// Healing stops at the maximum; the last one is gone once used.
	s, ev = applyWith(t, s, cmd(t, "item.use", map[string]any{"heroId": "hero-2", "itemId": "item-1"}), cat)
	if s.Heroes[1].Body != 30 || len(s.Heroes[1].Items) != 0 || ev.Summary != "Vex used Healing Potion: Body 28 → 30 (none left)" {
		t.Fatalf("use the last: %d %+v %q", s.Heroes[1].Body, s.Heroes[1].Items, ev.Summary)
	}
	// A mana potion restores mana, up to the hero's maximum.
	s, _ = applyWith(t, s, cmd(t, "item.add", map[string]any{"heroId": "hero-1", "name": "Mana Potion", "restoreMana": 6}), cat)
	s, ev = applyWith(t, s, cmd(t, "item.use", map[string]any{"heroId": "hero-1", "itemId": "item-1"}), cat)
	if s.Heroes[0].Mana != 16 || ev.Summary != "Mira used Mana Potion: Mana 14 → 16 (none left)" {
		t.Fatalf("mana potion: %d %q", s.Heroes[0].Mana, ev.Summary)
	}
	s, _ = applyWith(t, s, cmd(t, "item.add", map[string]any{"heroId": "hero-1", "name": "Rope"}), cat)
	if _, _, err := Apply(s, cmd(t, "item.use", map[string]any{"heroId": "hero-1", "itemId": "item-1"}), cat); err == nil {
		t.Fatal("an item with nothing to use should fail")
	}
	if _, err := NormalizeItems([]Item{{Name: "Elixir", HealBody: 100}}); err == nil {
		t.Fatal("healing over 99 should fail")
	}
}
