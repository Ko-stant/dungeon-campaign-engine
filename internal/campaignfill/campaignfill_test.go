package campaignfill

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

const fixture = `{
  "about": "test numbers",
  "classes": [
    {
      "name": "Rogue", "color": "#4b5563", "body": 28, "hitDice": "2d10", "accuracy": 2, "critFrom": 15,
      "damage": 1, "avoidance": 4, "defenseDice": "1d6", "mitigation": 0, "exclusives": ["disarm"],
      "abilities": [
        {"name": "Nimble Fingers", "kind": "passive", "text": "Disarms on 1d8."},
        {"name": "Fan of Cards", "formerly": ["Fan of Blades"], "kind": "active", "cooldown": 3, "text": "Cards at every enemy around."},
        {"name": "Venom Vial", "kind": "active", "cooldown": 4, "text": "Poison."}
      ]
    },
    {
      "name": "Cleric", "color": "#ca8a04", "body": 28, "hitDice": "2d8", "accuracy": 4, "critFrom": 20,
      "damage": 2, "avoidance": 3, "defenseDice": "1d6", "mitigation": 0, "mana": 16, "manaRegen": 2,
      "abilities": [{"name": "Smite", "kind": "spell", "manaCost": 2}]
    }
  ],
  "startingKit": [
    {"name": "Sword", "hero": "Rogue", "kind": "weapon", "damage": 3},
    {"name": "Soft Boots", "hero": "Rogue", "kind": "feet", "avoidance": 1},
    {"name": "Holy Tome", "hero": "Cleric", "kind": "off-hand", "manaRegen": 1}
  ],
  "monsters": {
    "goblin": {"body": 5, "avoidance": 6, "hitDice": "1d12", "damage": 4},
    "gargoyle": {"body": 109, "avoidance": 14, "hitDice": "2d10 + 3", "damage": 12, "line": 1},
    "stranger": {"body": 1}
  },
  "loot": [
    {"name": "Wardens' Dirk", "hero": "Rogue", "kind": "weapon", "damage": 3, "replaces": "Sword", "label": "S", "after": "Room 5"},
    {"name": "Healing Potion", "healBody": 8, "label": "V", "note": "Drinking is free."}
  ]
}`

func load(t *testing.T) Data {
	t.Helper()
	d, err := Load([]byte(fixture))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestLoad(t *testing.T) {
	d := load(t)
	if len(d.Classes) != 2 || len(d.StartingKit) != 3 || len(d.Monsters) != 3 {
		t.Fatalf("loaded: %+v", d)
	}
	if st := d.Monsters["stranger"]; st.Body != 1 || st.HitDice != "" {
		t.Fatalf("a body-only line (no combat stats): %+v", st)
	}
	if l := d.Loot; len(l) != 2 || l[0].Name != "Wardens' Dirk" || l[0].Damage != 3 || l[0].Replaces != "Sword" || l[1].HealBody != 8 {
		t.Fatalf("loot: %+v", d.Loot)
	}
	if d.Monsters["gargoyle"].HitDice != "2d10+3" {
		t.Fatalf("dice are made canonical: %q", d.Monsters["gargoyle"].HitDice)
	}
	if k := d.StartingKit[0]; k.Hero != "Rogue" || k.Name != "Sword" || k.Damage != 3 || k.Kind != "weapon" {
		t.Fatalf("kit item: %+v", k)
	}

	bad := map[string]func(s string) string{
		"unknown field":       func(s string) string { return strings.Replace(s, `"about"`, `"aboot"`, 1) },
		"bad hit dice":        func(s string) string { return strings.Replace(s, `"hitDice": "2d10",`, `"hitDice": "2d7",`, 1) },
		"bad monster dice":    func(s string) string { return strings.Replace(s, `"1d12"`, `"lots"`, 1) },
		"crit out of range":   func(s string) string { return strings.Replace(s, `"critFrom": 15`, `"critFrom": 21`, 1) },
		"bad ability kind":    func(s string) string { return strings.Replace(s, `"kind": "passive"`, `"kind": "lazy"`, 1) },
		"repeated ability":    func(s string) string { return strings.Replace(s, `"Venom Vial"`, `"Fan of Cards"`, 1) },
		"repeated class":      func(s string) string { return strings.Replace(s, `"name": "Cleric"`, `"name": "rogue"`, 1) },
		"kit for no class":    func(s string) string { return strings.Replace(s, `"hero": "Cleric"`, `"hero": "Wizard"`, 1) },
		"repeated kit item":   func(s string) string { return strings.Replace(s, `"Soft Boots"`, `"Sword"`, 1) },
		"bad color":           func(s string) string { return strings.Replace(s, `"#4b5563"`, `"gray"`, 1) },
		"no body":             func(s string) string { return strings.Replace(s, `"body": 5,`, `"body": 0,`, 1) },
		"huge item stat":      func(s string) string { return strings.Replace(s, `"damage": 3}`, `"damage": 300}`, 1) },
		"repeated loot":       func(s string) string { return strings.Replace(s, `"Healing Potion"`, `"Wardens' Dirk"`, 1) },
		"loot heals too much": func(s string) string { return strings.Replace(s, `"healBody": 8`, `"healBody": 100`, 1) },
		"stats without dice": func(s string) string {
			return strings.Replace(s, `"stranger": {"body": 1}`, `"stranger": {"body": 1, "damage": 2}`, 1)
		},
	}
	for name, change := range bad {
		changed := change(fixture)
		if changed == fixture {
			t.Fatalf("%s: the fixture did not change", name)
		}
		if _, err := Load([]byte(changed)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// The real file: the simulator's numbers must also load into the app.
func TestCampaignFileLoads(t *testing.T) {
	data, err := os.ReadFile("../../docs/campaigns/three-plagues/combat.json")
	if err != nil {
		t.Fatal(err)
	}
	d, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Classes) != 4 || len(d.StartingKit) != 13 || len(d.Monsters) != 13 || len(d.Loot) != 9 {
		t.Fatalf("combat.json: %d classes, %d kit items, %d monsters, %d loot", len(d.Classes), len(d.StartingKit), len(d.Monsters), len(d.Loot))
	}
	minds := map[string]int{}
	for _, c := range d.Classes {
		minds[c.Name] = c.Mind
	}
	reach := map[string]string{}
	for _, c := range d.Classes {
		reach[c.Name] = c.Reach
	}
	if reach["Barbarian"] != content.ReachDiagonal || reach["Ranger"] != content.ReachSight || reach["Rogue"] != content.ReachAdjacent || reach["Cleric"] != content.ReachAdjacent {
		t.Errorf("basic attack reach (RULES_AND_CLASSES.md): %v", reach)
	}
	if minds["Barbarian"] != 3 || minds["Ranger"] != 4 || minds["Rogue"] != 5 || minds["Cleric"] != 6 {
		t.Fatalf("Mind (Will): %v", minds)
	}
}

func decodeDoc(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMergeClass(t *testing.T) {
	rogue := load(t).Classes[0]
	old := json.RawMessage(`{"color": "#111111", "description": "Quick hands.", "body": 20, "mind": 3, "attack": "1d6", "defense": "1d6",
		"movement": "2d6", "accuracy": 0, "mana": 0, "exclusives": ["disarm"], "nextAbility": 4, "abilities": [
		{"id": "ability-1", "name": "Nimble Fingers", "kind": "passive", "text": "Disarms a trap on 1d8."},
		{"id": "ability-2", "name": "Vanish From Sight", "kind": "reaction", "cooldown": 5},
		{"id": "ability-3", "name": "Fan of Blades", "kind": "active", "cooldown": 3}]}`)
	doc, changes, err := MergeClass(old, rogue)
	if err != nil {
		t.Fatal(err)
	}
	m := decodeDoc(t, doc)
	// The GM's own fields stay; the combat numbers are set.
	for k, want := range map[string]any{
		"color": "#111111", "description": "Quick hands.", "mind": 3.0, "movement": "2d6",
		"body": 28.0, "attack": "2d10", "defense": "1d6", "accuracy": 2.0, "critFrom": 15.0, "damage": 1.0, "avoidance": 4.0,
		"nextAbility": 5.0,
	} {
		if !reflect.DeepEqual(m[k], want) {
			t.Errorf("%s: %v, want %v", k, m[k], want)
		}
	}
	for _, k := range []string{"mitigation", "manaRegen"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s is 0 and left out, as the app writes it", k)
		}
	}
	var abilities []content.Ability
	_ = json.Unmarshal(doc, &struct {
		Abilities *[]content.Ability `json:"abilities"`
	}{&abilities})
	want := []content.Ability{
		{ID: "ability-1", Name: "Nimble Fingers", Kind: "passive", Text: "Disarms on 1d8."},
		{ID: "ability-3", Name: "Fan of Cards", Kind: "active", Cooldown: 3, Text: "Cards at every enemy around."},
		{ID: "ability-4", Name: "Venom Vial", Kind: "active", Cooldown: 4, Text: "Poison."},
	}
	if !reflect.DeepEqual(abilities, want) {
		t.Fatalf("abilities keep their ids by name (or former name), new ones get the next id:\n%+v", abilities)
	}
	got := strings.Join(changes, "; ")
	for _, s := range []string{"body 20 → 28", "hit dice 1d6 → 2d10", "crit 20 → 15", "Fan of Blades renamed Fan of Cards", "added Venom Vial", "removed Vanish From Sight", "Nimble Fingers updated"} {
		if !strings.Contains(got, s) {
			t.Errorf("changes %q should mention %q", got, s)
		}
	}

	// Merging again changes nothing.
	again, changes, err := MergeClass(doc, rogue)
	if err != nil || len(changes) != 0 || !reflect.DeepEqual(decodeDoc(t, again), m) {
		t.Fatalf("second merge: %v %v", changes, err)
	}

	// Mind, when given, is set too (it is the heroes' Will).
	withMind := rogue
	withMind.Mind = 5
	minded, changes, err := MergeClass(doc, withMind)
	if err != nil || decodeDoc(t, minded)["mind"] != 5.0 || !strings.Contains(strings.Join(changes, "; "), "mind 3 → 5") {
		t.Fatalf("mind: %v %v", changes, err)
	}

	// A new class gets the defaults the class form uses.
	fresh, changes, err := MergeClass(nil, rogue)
	if err != nil || len(changes) == 0 {
		t.Fatalf("new class: %v %v", changes, err)
	}
	if f := decodeDoc(t, fresh); f["color"] != "#4b5563" || f["mind"] != 3.0 || f["movement"] != "2d6" || f["nextAbility"] != 4.0 {
		t.Fatalf("new class doc: %v", f)
	}
}

func TestMergeMonsterStats(t *testing.T) {
	want := load(t).Monsters
	cur := map[string]content.MonsterStats{
		"goblin": {Body: 3, MonsterCombat: content.MonsterCombat{Avoidance: 6, HitDice: "1d12", Damage: 4}},
		"ogre":   {Body: 80, MonsterCombat: content.MonsterCombat{Avoidance: 12, HitDice: "2d10", Damage: 14}},
	}
	out, changes := MergeMonsterStats(cur, want)
	if out["goblin"].Body != 5 || out["gargoyle"].Line != 1 || out["ogre"].Body != 80 || cur["goblin"].Body != 3 {
		t.Fatalf("merged: %+v", out)
	}
	if got := strings.Join(changes, "; "); got != "gargoyle added; goblin updated; stranger added" {
		t.Fatalf("changes: %q", got)
	}
	if _, changes = MergeMonsterStats(out, want); len(changes) != 0 {
		t.Fatalf("again: %v", changes)
	}
	// The GM's own abilities text survives a fill that has none.
	gm := out["goblin"]
	gm.Abilities = "Steals a potion on a hit."
	out["goblin"] = gm
	kept, changes := MergeMonsterStats(out, want)
	if kept["goblin"].Abilities != "Steals a potion on a hit." || len(changes) != 0 {
		t.Fatalf("abilities kept: %+v %v", kept["goblin"], changes)
	}
}

func TestGiveKits(t *testing.T) {
	kit := load(t).StartingKit
	classIDs := map[string]string{"Rogue": "custom-r", "Cleric": "custom-c"}
	heroes := []tracker.CampaignHero{
		{ID: "hero-1", Name: "Vex", Class: "custom-r", Items: []tracker.Item{{ID: "item-1", Name: "Rope", Quantity: 1}}},
		{ID: "hero-2", Name: "Bram", Class: "custom-r", Items: []tracker.Item{{ID: "item-1", Name: "sword", Quantity: 1}}},
		{ID: "hero-3", Name: "Ilsa", Class: "wizard"},
	}
	out, changes, err := GiveKits(heroes, classIDs, kit)
	if err != nil {
		t.Fatal(err)
	}
	vex := out[0].Items
	if len(vex) != 3 || vex[1].Name != "Sword" || !vex[1].Equipped || vex[1].Damage != 3 || vex[1].Kind != "weapon" || vex[2].ID != "item-3" || !vex[2].Equipped {
		t.Fatalf("Vex's kit: %+v", vex)
	}
	if len(heroes[0].Items) != 1 {
		t.Fatal("GiveKits modified its input")
	}
	if len(out[1].Items) != 1 {
		t.Fatalf("a hero who already has kit gear is left alone: %+v", out[1].Items)
	}
	got := strings.Join(changes, "; ")
	for _, s := range []string{"Vex: Sword, Soft Boots (equipped)", "Bram already carries starting gear", "no Cleric hero"} {
		if !strings.Contains(got, s) {
			t.Errorf("changes %q should mention %q", got, s)
		}
	}
	if strings.Contains(got, "Ilsa") {
		t.Errorf("heroes of other classes are not mentioned: %q", got)
	}

	// No heroes at all: one line says to add the party first.
	_, changes, _ = GiveKits(nil, classIDs, kit)
	if len(changes) != 1 || !strings.Contains(changes[0], "no heroes yet") {
		t.Fatalf("no heroes: %q", changes)
	}
}

func TestMergeLoot(t *testing.T) {
	want := load(t).Loot
	cur := []tracker.Item{
		{ID: "loot-1", Name: "healing potion", Quantity: 1, HealBody: 6},
		{ID: "loot-2", Name: "Rope", Quantity: 1, Notes: "the GM's own"},
	}
	out, changes := MergeLoot(cur, want)
	if got := strings.Join(changes, "; "); got != "Healing Potion updated; Wardens' Dirk added" {
		t.Fatalf("changes: %q", got)
	}
	byName := map[string]tracker.Item{}
	for _, it := range out {
		byName[it.Name] = it
	}
	dirk, potion := byName["Wardens' Dirk"], byName["Healing Potion"]
	if len(out) != 3 || byName["Rope"].Notes != "the GM's own" || potion.ID != "loot-1" || potion.HealBody != 8 || dirk.ID != "loot-3" {
		t.Fatalf("merged: %+v", out)
	}
	// Who it's for, what it replaces and the board note become the item's notes.
	if dirk.Notes != "For the Rogue; replaces Sword; note S (after Room 5)" || potion.Notes != "Note V. Drinking is free." {
		t.Fatalf("notes: %q / %q", dirk.Notes, potion.Notes)
	}
	if cur[0].HealBody != 6 {
		t.Fatal("cur must not change")
	}
	if _, changes = MergeLoot(out, want); len(changes) != 0 {
		t.Fatalf("again: %v", changes)
	}
}

func TestMergeClassReach(t *testing.T) {
	rogue := load(t).Classes[0]
	old := json.RawMessage(`{"color": "#111111", "body": 20, "attack": "1d6", "defense": "1d6", "movement": "2d6", "reach": "adjacent"}`)
	// No reach in the file leaves the class's own.
	doc, _, err := MergeClass(old, rogue)
	if err != nil || decodeDoc(t, doc)["reach"] != "adjacent" {
		t.Fatalf("reach kept: %v", err)
	}
	rogue.Reach = content.ReachDiagonal
	doc, changes, err := MergeClass(old, rogue)
	if err != nil || decodeDoc(t, doc)["reach"] != "diagonal" || !strings.Contains(strings.Join(changes, "; "), "reach adjacent → diagonal") {
		t.Fatalf("reach set: %v %v", changes, err)
	}
	if _, err := Load([]byte(strings.Replace(fixture, `"critFrom": 15,`, `"critFrom": 15, "reach": "far",`, 1))); err == nil || !strings.Contains(err.Error(), "reach") {
		t.Errorf("an unknown reach: %v", err)
	}
}
