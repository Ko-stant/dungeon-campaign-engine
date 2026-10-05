// Package campaignfill loads a campaign's agreed combat numbers (classes and
// their abilities, the starting kit and monster stat lines, e.g.
// docs/campaigns/three-plagues/combat.json, which the simulator reads too)
// into the app: it updates the custom hero classes, the campaign's monster
// stats and its heroes' starting gear. cmd/fill-campaign runs it.
package campaignfill

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/dice"
	mapdocs "github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// Data is a campaign's combat numbers.
type Data struct {
	About       string                          `json:"about,omitempty"`
	Classes     []Class                         `json:"classes"`
	StartingKit []KitItem                       `json:"startingKit"`
	Monsters    map[string]content.MonsterStats `json:"monsters"`
	// Loot is the campaign's loot list: finds and potions the GM hands out.
	Loot []LootItem `json:"loot,omitempty"`
}

// LootItem is an item on the loot list, with what the simulator needs to
// know about it: the class it is for, the starting item it replaces, and its
// board note and the encounter it comes after. Those become the item's notes
// in the app.
type LootItem struct {
	tracker.Item
	Hero     string `json:"hero,omitempty"`
	Replaces string `json:"replaces,omitempty"`
	Label    string `json:"label,omitempty"`
	After    string `json:"after,omitempty"`
	Note     string `json:"note,omitempty"`
}

// Class is a hero class's combat stats and abilities. Movement and the
// description are left to the GM; Mind (the heroes' Will) is set when given.
type Class struct {
	Name        string   `json:"name"`
	Color       string   `json:"color"`
	Body        int      `json:"body"`
	Mind        int      `json:"mind,omitempty"`
	HitDice     string   `json:"hitDice"`
	Accuracy    int      `json:"accuracy"`
	CritFrom    int      `json:"critFrom"`
	Damage      int      `json:"damage"`
	Avoidance   int      `json:"avoidance"`
	DefenseDice string   `json:"defenseDice"`
	Mitigation  int      `json:"mitigation"`
	Mana        int      `json:"mana,omitempty"`
	ManaRegen   int      `json:"manaRegen,omitempty"`
	Exclusives  []string `json:"exclusives,omitempty"`
	// Reach is what the basic attack reaches (content.Reach*); empty leaves
	// the class's own.
	Reach     string    `json:"reach,omitempty"`
	Abilities []Ability `json:"abilities"`
}

// Ability is one class ability. Formerly lists names it had before, so a
// renamed ability keeps its id (sessions refer to abilities by id).
type Ability struct {
	Name     string   `json:"name"`
	Formerly []string `json:"formerly,omitempty"`
	Kind     string   `json:"kind"`
	Cooldown int      `json:"cooldown,omitempty"`
	ManaCost int      `json:"manaCost,omitempty"`
	Text     string   `json:"text,omitempty"`
}

// KitItem is a starting item for every hero of the class named Hero.
type KitItem struct {
	Hero string `json:"hero"`
	tracker.Item
}

// Load reads and checks combat numbers; dice expressions come back canonical.
func Load(data []byte) (Data, error) {
	var d Data
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return d, err
	}
	classes := map[string]bool{}
	for i := range d.Classes {
		c := &d.Classes[i]
		if err := checkClass(c); err != nil {
			return d, fmt.Errorf("class %q: %w", c.Name, err)
		}
		key := strings.ToLower(c.Name)
		if classes[key] {
			return d, fmt.Errorf("class %q is listed twice", c.Name)
		}
		classes[key] = true
	}
	perHero := map[string][]tracker.Item{}
	for _, k := range d.StartingKit {
		if !classes[strings.ToLower(k.Hero)] {
			return d, fmt.Errorf("starting kit %q: no class %q", k.Name, k.Hero)
		}
		items, _, err := tracker.AddItem(perHero[k.Hero], k.Item)
		if err != nil {
			return d, fmt.Errorf("starting kit %q: %w", k.Name, err)
		}
		if len(items) == len(perHero[k.Hero]) {
			return d, fmt.Errorf("starting kit: %s has %q twice", k.Hero, k.Name)
		}
		perHero[k.Hero] = items
	}
	names := map[string]bool{}
	for _, l := range d.Loot {
		key := strings.ToLower(strings.TrimSpace(l.Name))
		if names[key] {
			return d, fmt.Errorf("loot: %q is listed twice", l.Name)
		}
		names[key] = true
		if _, err := tracker.NormalizeItems([]tracker.Item{l.Item}); err != nil {
			return d, fmt.Errorf("loot %q: %w", l.Name, err)
		}
	}
	for typ, m := range d.Monsters {
		if m.Body < 1 || m.Body > 999 || m.Avoidance < 0 || m.Avoidance > 99 || m.Damage < 0 || m.Damage > 99 ||
			m.Line < 0 || m.Line > 3 || m.SplashDamage < 0 || m.SplashTargets < 0 || m.SplashTargets > 8 {
			return d, fmt.Errorf("monster %q: a number is out of range", typ)
		}
		if m.CombatEmpty() {
			continue // Body only: no combat stats
		}
		hit, err := canonicalDice(m.HitDice)
		if err != nil {
			return d, fmt.Errorf("monster %q: hit dice: %w", typ, err)
		}
		m.HitDice = hit
		d.Monsters[typ] = m
	}
	return d, nil
}

func canonicalDice(s string) (string, error) {
	e, err := dice.Parse(s)
	if err != nil {
		return "", err
	}
	return e.String(), nil
}

func checkClass(c *Class) error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return errors.New("a class needs a name")
	}
	if !mapdocs.IsHexColor(c.Color) {
		return errors.New("color must be a #rrggbb color")
	}
	for _, f := range []struct {
		label  string
		v      int
		lo, hi int
	}{
		{"body", c.Body, 1, 999}, {"mind", c.Mind, 0, 999}, {"accuracy", c.Accuracy, 0, 99}, {"crit", c.CritFrom, 2, 20}, {"damage", c.Damage, 0, 99},
		{"avoidance", c.Avoidance, 0, 99}, {"mitigation", c.Mitigation, 0, 99}, {"mana", c.Mana, 0, 999}, {"mana regen", c.ManaRegen, 0, 99},
	} {
		if f.v < f.lo || f.v > f.hi {
			return fmt.Errorf("%s must be from %d to %d", f.label, f.lo, f.hi)
		}
	}
	if c.Reach != "" && !slices.Contains(content.Reaches, c.Reach) {
		return fmt.Errorf("reach must be one of %s", strings.Join(content.Reaches, ", "))
	}
	var err error
	if c.HitDice, err = canonicalDice(c.HitDice); err != nil {
		return fmt.Errorf("hit dice: %w", err)
	}
	if c.DefenseDice, err = canonicalDice(c.DefenseDice); err != nil {
		return fmt.Errorf("defense dice: %w", err)
	}
	names := map[string]bool{}
	for _, a := range c.Abilities {
		if strings.TrimSpace(a.Name) == "" {
			return errors.New("an ability needs a name")
		}
		if !slices.Contains(content.AbilityKinds, a.Kind) {
			return fmt.Errorf("ability %q: kind must be one of %s", a.Name, strings.Join(content.AbilityKinds, ", "))
		}
		if a.Cooldown < 0 || a.Cooldown > 99 || a.ManaCost < 0 || a.ManaCost > 999 {
			return fmt.Errorf("ability %q: cooldown or mana cost out of range", a.Name)
		}
		for _, n := range append([]string{a.Name}, a.Formerly...) {
			if names[strings.ToLower(n)] {
				return fmt.Errorf("ability %q is listed twice", n)
			}
			names[strings.ToLower(n)] = true
		}
	}
	return nil
}

// classDoc is a custom hero class's stored doc (see the app's customClassDoc),
// kept as raw fields so fields the fill doesn't know survive.
type classDoc map[string]json.RawMessage

func (d classDoc) int(key string, missing int) int {
	var n int
	if raw, ok := d[key]; ok && json.Unmarshal(raw, &n) == nil {
		return n
	}
	return missing
}

func (d classDoc) string(key string) string {
	var s string
	if raw, ok := d[key]; ok {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

func (d classDoc) set(key string, v any) {
	raw, _ := json.Marshal(v)
	d[key] = raw
}

// setInt stores n, leaving the key out when it is 0 and the app writes it omitempty.
func (d classDoc) setInt(key string, n int, omitZero bool) {
	if n == 0 && omitZero {
		delete(d, key)
		return
	}
	d.set(key, n)
}

// MergeClass sets a class's combat stats and abilities on its stored doc
// (nil for a new class) and describes what changed. The doc's other fields
// (description, movement, color once set, and mind when c has none) stay as
// the GM left them.
// Abilities keep their ids when their name, or a former name, matches.
func MergeClass(old json.RawMessage, c Class) (json.RawMessage, []string, error) {
	doc := classDoc{}
	if len(old) > 0 {
		if err := json.Unmarshal(old, &doc); err != nil {
			return nil, nil, err
		}
	}
	if doc.string("color") == "" {
		doc.set("color", c.Color)
	}
	if _, ok := doc["mind"]; !ok {
		doc.set("mind", 3)
	}
	if doc.string("movement") == "" {
		doc.set("movement", "2d6")
	}

	var changes []string
	setInt := func(label, key string, v, missing int, omitZero bool) {
		if before := doc.int(key, missing); before != v {
			changes = append(changes, fmt.Sprintf("%s %d → %d", label, before, v))
		}
		doc.setInt(key, v, omitZero)
	}
	setDice := func(label, key, v string) {
		if before := doc.string(key); before != v {
			changes = append(changes, fmt.Sprintf("%s %s → %s", label, orNone(before), v))
		}
		doc.set(key, v)
	}
	setInt("body", "body", c.Body, 0, false)
	if c.Mind > 0 {
		setInt("mind", "mind", c.Mind, 0, false)
	}
	setDice("hit dice", "attack", c.HitDice)
	setInt("accuracy", "accuracy", c.Accuracy, 0, false)
	setInt("crit", "critFrom", c.CritFrom, 20, true)
	setInt("damage", "damage", c.Damage, 0, true)
	setDice("defense dice", "defense", c.DefenseDice)
	setInt("avoidance", "avoidance", c.Avoidance, 0, true)
	setInt("mitigation", "mitigation", c.Mitigation, 0, true)
	setInt("mana", "mana", c.Mana, 0, false)
	setInt("mana per fight round", "manaRegen", c.ManaRegen, 0, true)
	if c.Reach != "" {
		if before := doc.string("reach"); before != c.Reach {
			changes = append(changes, fmt.Sprintf("reach %s → %s", orNone(before), c.Reach))
		}
		doc.set("reach", c.Reach)
	}
	if len(c.Exclusives) > 0 {
		var before []string
		_ = json.Unmarshal(doc["exclusives"], &before)
		if !slices.Equal(before, c.Exclusives) {
			changes = append(changes, fmt.Sprintf("exclusives %s → %s", orNone(strings.Join(before, ", ")), strings.Join(c.Exclusives, ", ")))
		}
		doc.set("exclusives", c.Exclusives)
	}

	var existing []content.Ability
	if raw, ok := doc["abilities"]; ok {
		if err := json.Unmarshal(raw, &existing); err != nil {
			return nil, nil, err
		}
	}
	next := doc.int("nextAbility", 0)
	for _, a := range existing {
		if n, err := strconv.Atoi(strings.TrimPrefix(a.ID, "ability-")); err == nil && n >= next {
			next = n + 1
		}
	}
	next = max(next, 1)
	used := map[string]bool{}
	var abilities []content.Ability
	var abilityChanges []string
	for _, spec := range c.Abilities {
		i := slices.IndexFunc(existing, func(a content.Ability) bool {
			if used[a.ID] {
				return false
			}
			for _, n := range append([]string{spec.Name}, spec.Formerly...) {
				if strings.EqualFold(a.Name, n) {
					return true
				}
			}
			return false
		})
		a := content.Ability{Name: spec.Name, Kind: spec.Kind, ManaCost: spec.ManaCost, Cooldown: spec.Cooldown, Text: spec.Text}
		switch {
		case i < 0:
			a.ID = "ability-" + strconv.Itoa(next)
			next++
			abilityChanges = append(abilityChanges, "added "+a.Name)
		default:
			was := existing[i]
			a.ID = was.ID
			used[a.ID] = true
			if was.Name != a.Name {
				abilityChanges = append(abilityChanges, fmt.Sprintf("%s renamed %s", was.Name, a.Name))
			} else if was != a {
				abilityChanges = append(abilityChanges, a.Name+" updated")
			}
		}
		abilities = append(abilities, a)
	}
	for _, a := range existing {
		if !used[a.ID] {
			abilityChanges = append(abilityChanges, "removed "+a.Name)
		}
	}
	changes = append(changes, abilityChanges...)
	doc.set("abilities", abilities)
	doc.set("nextAbility", next)

	out, err := json.Marshal(doc)
	return out, changes, err
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// MergeMonsterStats returns the campaign's monster stat lines with want's
// lines set (others kept; a line's abilities text stays when want has none)
// and names the lines added or updated. It never modifies cur.
func MergeMonsterStats(cur, want map[string]content.MonsterStats) (map[string]content.MonsterStats, []string) {
	out := maps.Clone(cur)
	if out == nil {
		out = map[string]content.MonsterStats{}
	}
	var changes []string
	for _, typ := range slices.Sorted(maps.Keys(want)) {
		before, ok := out[typ]
		next := want[typ]
		if next.Abilities == "" {
			// The GM's own abilities text stays when the numbers have none.
			next.Abilities = before.Abilities
		}
		switch {
		case !ok:
			changes = append(changes, typ+" added")
		case before != next:
			changes = append(changes, typ+" updated")
		}
		out[typ] = next
	}
	return out, changes
}

// GiveKits gives each hero of a filled class (classIDs: class name -> catalog
// id) their class's starting kit, equipped. A hero who already carries any
// starting-kit item is left alone, so a rerun never brings back gear the hero
// has replaced. It names what it did, and the classes with no hero.
func GiveKits(heroes []tracker.CampaignHero, classIDs map[string]string, kit []KitItem) ([]tracker.CampaignHero, []string, error) {
	if len(heroes) == 0 {
		return heroes, []string{"the campaign has no heroes yet: add the party on the campaign page (Heroes), then run again with -apply (APPLY=1) to give them their starting kits"}, nil
	}
	out := slices.Clone(heroes)
	var classes []string
	byClass := map[string][]KitItem{}
	for _, k := range kit {
		if _, ok := byClass[k.Hero]; !ok {
			classes = append(classes, k.Hero)
		}
		byClass[k.Hero] = append(byClass[k.Hero], k)
	}
	var changes []string
	served := map[string]bool{}
	for i := range out {
		h := &out[i]
		class := ""
		for name, id := range classIDs {
			if id == h.Class {
				class = name
			}
		}
		items, ok := byClass[class]
		if !ok {
			continue
		}
		served[class] = true
		if j := slices.IndexFunc(h.Items, func(it tracker.Item) bool {
			return slices.ContainsFunc(items, func(k KitItem) bool { return strings.EqualFold(k.Name, it.Name) })
		}); j >= 0 {
			changes = append(changes, fmt.Sprintf("%s already carries starting gear (%s); left alone", h.Name, h.Items[j].Name))
			continue
		}
		var names []string
		for _, k := range items {
			add := k.Item
			add.Equipped = true
			next, it, err := tracker.AddItem(h.Items, add)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", h.Name, err)
			}
			h.Items = next
			names = append(names, it.Name)
		}
		changes = append(changes, fmt.Sprintf("%s: %s (equipped)", h.Name, strings.Join(names, ", ")))
	}
	for _, class := range classes {
		if !served[class] {
			changes = append(changes, fmt.Sprintf("no %s hero in the campaign: add one on the campaign page and run again for the starting kit", class))
		}
	}
	return out, changes, nil
}

// lootNotes is a loot item's notes in the app: who it is for, what it
// replaces, its board note, then its own note.
func lootNotes(l LootItem) string {
	var parts []string
	if l.Hero != "" {
		parts = append(parts, "for the "+l.Hero)
	}
	if l.Replaces != "" {
		parts = append(parts, "replaces "+l.Replaces)
	}
	if l.Label != "" {
		label := "note " + l.Label
		if l.After != "" {
			label += " (after " + l.After + ")"
		}
		parts = append(parts, label)
	}
	out := strings.Join(parts, "; ")
	if out != "" {
		out = strings.ToUpper(out[:1]) + out[1:]
	}
	if note := strings.TrimSpace(l.Note); note != "" {
		if out != "" {
			out += ". "
		}
		out += note
	}
	return out
}

// MergeLoot returns the campaign's loot list with want's items set (matched
// by name; the GM's own items kept) and names the items added or updated.
// It never modifies cur.
func MergeLoot(cur []tracker.Item, want []LootItem) ([]tracker.Item, []string) {
	out := slices.Clone(cur)
	next := 1
	for _, it := range out {
		if n, err := strconv.Atoi(strings.TrimPrefix(it.ID, "loot-")); err == nil && n >= next {
			next = n + 1
		}
	}
	var changes []string
	for _, l := range want {
		it := l.Item
		it.Name, it.Notes, it.Quantity, it.Equipped = strings.TrimSpace(it.Name), lootNotes(l), 1, false
		i := slices.IndexFunc(out, func(c tracker.Item) bool { return strings.EqualFold(c.Name, it.Name) })
		if i < 0 {
			it.ID = fmt.Sprintf("loot-%d", next)
			next++
			out = append(out, it)
			changes = append(changes, it.Name+" added")
			continue
		}
		it.ID = out[i].ID
		if !reflect.DeepEqual(out[i], it) {
			out[i] = it
			changes = append(changes, it.Name+" updated")
		}
	}
	slices.Sort(changes)
	return out, changes
}
