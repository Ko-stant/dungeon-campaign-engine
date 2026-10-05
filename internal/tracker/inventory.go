package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Item limits.
const (
	MaxItemName     = 120
	MaxItemNotes    = 500
	MaxItemQuantity = 9999
	MaxItemKind     = 40
	// MaxItemStat bounds each item stat either way (a cursed item may take away).
	MaxItemStat = 99
)

const itemIDPrefix = "item-"

// Item is something a hero carries. There is no slot or weight limit; an
// equipped item adds its stats to the hero's combat totals (The Three
// Plagues rules), and otherwise the GM decides what an item does.
type Item struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Notes    string `json:"notes,omitempty"`
	// Kind says what the item is (weapon, bow, chest, feet, trinket...); free text, no slots.
	Kind     string `json:"kind,omitempty"`
	Equipped bool   `json:"equipped,omitempty"`
	ItemStats
	// HealBody and RestoreMana make the item usable (a potion): using one
	// spends it and gives its hero that much Body or mana, up to the maximum.
	HealBody    int `json:"healBody,omitempty"`
	RestoreMana int `json:"restoreMana,omitempty"`
}

// Usable reports an item that does something when used (item.use).
func (it Item) Usable() bool {
	return it.HealBody > 0 || it.RestoreMana > 0
}

// Summary is the item's stats and what using it does, e.g. "damage +7" or
// "heals 8 Body" ("" for neither).
func (it Item) Summary() string {
	parts := []string{}
	if st := it.ItemStats.Summary(); st != "" {
		parts = append(parts, st)
	}
	if it.HealBody > 0 {
		parts = append(parts, fmt.Sprintf("heals %d Body", it.HealBody))
	}
	if it.RestoreMana > 0 {
		parts = append(parts, fmt.Sprintf("restores %d mana", it.RestoreMana))
	}
	return strings.Join(parts, ", ")
}

func (it Item) check() error {
	if err := it.ItemStats.check(); err != nil {
		return err
	}
	if it.HealBody < 0 || it.HealBody > MaxItemStat || it.RestoreMana < 0 || it.RestoreMana > MaxItemStat {
		return fmt.Errorf("an item heals or restores from 0 to %d", MaxItemStat)
	}
	return nil
}

// ItemStats are an item's bonuses to its hero's combat stats, counted once
// (whatever the quantity) while the item is equipped.
type ItemStats struct {
	Damage     int `json:"damage,omitempty"`
	Accuracy   int `json:"accuracy,omitempty"`
	Avoidance  int `json:"avoidance,omitempty"`
	Mitigation int `json:"mitigation,omitempty"`
	Mana       int `json:"mana,omitempty"`
	ManaRegen  int `json:"manaRegen,omitempty"`
}

func (st ItemStats) fields() []struct {
	label string
	value int
} {
	return []struct {
		label string
		value int
	}{
		{"damage", st.Damage}, {"Accuracy", st.Accuracy}, {"avoidance", st.Avoidance},
		{"mitigation", st.Mitigation}, {"mana", st.Mana}, {"mana regen", st.ManaRegen},
	}
}

// Summary is e.g. "damage +7" or "mana +2, mana regen +1" ("" for no stats).
func (st ItemStats) Summary() string {
	var parts []string
	for _, f := range st.fields() {
		if f.value != 0 {
			parts = append(parts, fmt.Sprintf("%s %+d", f.label, f.value))
		}
	}
	return strings.Join(parts, ", ")
}

func (st ItemStats) check() error {
	for _, f := range st.fields() {
		if f.value < -MaxItemStat || f.value > MaxItemStat {
			return fmt.Errorf("item %s must be from %d to %d", f.label, -MaxItemStat, MaxItemStat)
		}
	}
	return nil
}

func (st ItemStats) add(o ItemStats) ItemStats {
	return ItemStats{
		Damage: st.Damage + o.Damage, Accuracy: st.Accuracy + o.Accuracy, Avoidance: st.Avoidance + o.Avoidance,
		Mitigation: st.Mitigation + o.Mitigation, Mana: st.Mana + o.Mana, ManaRegen: st.ManaRegen + o.ManaRegen,
	}
}

// GearBonus is the sum of the equipped items' stats.
func GearBonus(items []Item) ItemStats {
	var out ItemStats
	for _, it := range items {
		if it.Equipped {
			out = out.add(it.ItemStats)
		}
	}
	return out
}

// CombatTotals is the hero's class combat stats plus their equipped items
// (nil for heroes without combat stats).
func (h *Hero) CombatTotals() *Combat {
	if h.Combat == nil {
		return nil
	}
	g := GearBonus(h.Items)
	c := *h.Combat
	c.Accuracy += g.Accuracy
	c.Damage += g.Damage
	c.Avoidance += g.Avoidance
	c.Mitigation += g.Mitigation
	c.ManaRegen += g.ManaRegen
	return &c
}

// ManaCap is the hero's maximum mana with their equipped items.
func (h *Hero) ManaCap() int {
	return max(0, h.MaxMana+GearBonus(h.Items).Mana)
}

// ManaRegen is the mana the hero regains each fight round, with their equipped items.
func (h *Hero) ManaRegen() int {
	regen := GearBonus(h.Items).ManaRegen
	if h.Combat != nil {
		regen += h.Combat.ManaRegen
	}
	return regen
}

// withBonus is a flat bonus on a dice expression: "1d20+3", or "2d8+1 +4"
// when the dice have their own modifier.
func withBonus(dice string, bonus int) string {
	switch {
	case bonus == 0:
		return dice
	case strings.ContainsAny(dice, "+-"):
		return fmt.Sprintf("%s %+d", dice, bonus)
	default:
		return fmt.Sprintf("%s%+d", dice, bonus)
	}
}

// CombatLine sums up combat stats on one line, e.g. "Hit 1d20+3 · Crit 17-20 ·
// Damage 10 · Avoid 3+1d6 · Mitigation 2" (as the tracker's combatLine does).
func CombatLine(c Combat) string {
	crit := "Crit 20"
	if c.CritFrom < 20 {
		crit = fmt.Sprintf("Crit %d-20", c.CritFrom)
	}
	avoid := c.DefenseDice
	if c.Avoidance > 0 {
		avoid = fmt.Sprintf("%d+%s", c.Avoidance, c.DefenseDice)
	}
	parts := []string{"Hit " + withBonus(c.HitDice, c.Accuracy), crit, fmt.Sprintf("Damage %d", c.Damage), "Avoid " + avoid}
	if c.Mitigation > 0 {
		parts = append(parts, fmt.Sprintf("Mitigation %d", c.Mitigation))
	}
	if c.ManaRegen > 0 {
		parts = append(parts, fmt.Sprintf("Mana +%d a fight round", c.ManaRegen))
	}
	return strings.Join(parts, " · ")
}

func nextItemID(items []Item) string {
	highest := 0
	for _, it := range items {
		if n, err := strconv.Atoi(strings.TrimPrefix(it.ID, itemIDPrefix)); err == nil && strings.HasPrefix(it.ID, itemIDPrefix) && n > highest {
			highest = n
		}
	}
	return itemIDPrefix + strconv.Itoa(highest+1)
}

func cleanItemName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("an item needs a name")
	}
	if len(name) > MaxItemName {
		return "", fmt.Errorf("item names must be at most %d characters", MaxItemName)
	}
	return name, nil
}

func checkItemNotes(notes string) error {
	if len(notes) > MaxItemNotes {
		return fmt.Errorf("item notes must be at most %d characters", MaxItemNotes)
	}
	return nil
}

func cleanItemKind(kind string) (string, error) {
	kind = strings.TrimSpace(kind)
	if len(kind) > MaxItemKind {
		return "", fmt.Errorf("item kinds must be at most %d characters", MaxItemKind)
	}
	return kind, nil
}

func checkQuantity(q int) error {
	if q < 1 || q > MaxItemQuantity {
		return fmt.Errorf("quantity must be from 1 to %d", MaxItemQuantity)
	}
	return nil
}

func itemIndex(items []Item, id string) int {
	return slices.IndexFunc(items, func(it Item) bool { return it.ID == id })
}

// AddItem returns items with add.Quantity more of the named item (0 means
// 1). An item with the same name (ignoring case) gains the quantity and keeps
// its notes, kind and stats (taking add's where it has none); otherwise add
// is added with a new id. It never modifies items.
func AddItem(items []Item, add Item) ([]Item, Item, error) {
	name, err := cleanItemName(add.Name)
	if err != nil {
		return nil, Item{}, err
	}
	quantity := add.Quantity
	if quantity == 0 {
		quantity = 1
	}
	if err := checkQuantity(quantity); err != nil {
		return nil, Item{}, err
	}
	notes := strings.TrimSpace(add.Notes)
	if err := checkItemNotes(notes); err != nil {
		return nil, Item{}, err
	}
	kind, err := cleanItemKind(add.Kind)
	if err != nil {
		return nil, Item{}, err
	}
	if err := add.check(); err != nil {
		return nil, Item{}, err
	}
	out := slices.Clone(items)
	if i := slices.IndexFunc(out, func(it Item) bool { return strings.EqualFold(it.Name, name) }); i >= 0 {
		if err := checkQuantity(out[i].Quantity + quantity); err != nil {
			return nil, Item{}, err
		}
		out[i].Quantity += quantity
		if out[i].Notes == "" {
			out[i].Notes = notes
		}
		if out[i].Kind == "" {
			out[i].Kind = kind
		}
		if out[i].ItemStats == (ItemStats{}) {
			out[i].ItemStats = add.ItemStats
		}
		if !out[i].Usable() {
			out[i].HealBody, out[i].RestoreMana = add.HealBody, add.RestoreMana
		}
		return out, out[i], nil
	}
	it := Item{ID: nextItemID(out), Name: name, Quantity: quantity, Notes: notes, Kind: kind, Equipped: add.Equipped, ItemStats: add.ItemStats, HealBody: add.HealBody, RestoreMana: add.RestoreMana}
	return append(out, it), it, nil
}

// RemoveItem takes quantity of an item away (0, or more than there is,
// takes all of it). It returns the item as left (Quantity 0 when gone) and
// how many were removed. It never modifies items.
func RemoveItem(items []Item, id string, quantity int) ([]Item, Item, int, error) {
	i := itemIndex(items, id)
	if i < 0 {
		return nil, Item{}, 0, fmt.Errorf("no item %q", id)
	}
	if quantity < 0 {
		return nil, Item{}, 0, errors.New("quantity must not be negative")
	}
	out := slices.Clone(items)
	it := out[i]
	if quantity == 0 || quantity >= it.Quantity {
		removed := it.Quantity
		it.Quantity = 0
		return slices.Delete(out, i, i+1), it, removed, nil
	}
	out[i].Quantity -= quantity
	return out, out[i], quantity, nil
}

// ItemPatch is the fields of an item to change; nil leaves a field alone.
type ItemPatch struct {
	Name     *string
	Quantity *int
	Notes    *string
	Kind     *string
	Stats    *ItemStats
	// HealBody and RestoreMana change what using the item does.
	HealBody    *int
	RestoreMana *int
}

// UpdateItem changes an item's fields. A quantity of 0 is a removal, so it
// is refused here. It never modifies items.
func UpdateItem(items []Item, id string, p ItemPatch) ([]Item, Item, error) {
	i := itemIndex(items, id)
	if i < 0 {
		return nil, Item{}, fmt.Errorf("no item %q", id)
	}
	out := slices.Clone(items)
	it := &out[i]
	if p.Name != nil {
		n, err := cleanItemName(*p.Name)
		if err != nil {
			return nil, Item{}, err
		}
		it.Name = n
	}
	if p.Quantity != nil {
		if err := checkQuantity(*p.Quantity); err != nil {
			return nil, Item{}, err
		}
		it.Quantity = *p.Quantity
	}
	if p.Notes != nil {
		n := strings.TrimSpace(*p.Notes)
		if err := checkItemNotes(n); err != nil {
			return nil, Item{}, err
		}
		it.Notes = n
	}
	if p.Kind != nil {
		k, err := cleanItemKind(*p.Kind)
		if err != nil {
			return nil, Item{}, err
		}
		it.Kind = k
	}
	if p.HealBody != nil {
		it.HealBody = *p.HealBody
	}
	if p.RestoreMana != nil {
		it.RestoreMana = *p.RestoreMana
	}
	if err := it.check(); err != nil {
		return nil, Item{}, err
	}
	if p.Stats != nil {
		if err := p.Stats.check(); err != nil {
			return nil, Item{}, err
		}
		it.ItemStats = *p.Stats
	}
	return out, *it, nil
}

// NormalizeItems validates items (a quantity of 0 means 1) and gives items
// without an id, or with a repeated one, a new id. It never returns nil.
func NormalizeItems(items []Item) ([]Item, error) {
	out := make([]Item, 0, len(items))
	seen := map[string]bool{}
	next := nextItemID(items)
	for _, it := range items {
		name, err := cleanItemName(it.Name)
		if err != nil {
			return nil, err
		}
		it.Name = name
		if it.Quantity == 0 {
			it.Quantity = 1
		}
		if err := checkQuantity(it.Quantity); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		it.Notes = strings.TrimSpace(it.Notes)
		if err := checkItemNotes(it.Notes); err != nil {
			return nil, err
		}
		if it.Kind, err = cleanItemKind(it.Kind); err != nil {
			return nil, err
		}
		if err := it.check(); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if it.ID == "" || seen[it.ID] {
			it.ID = next
			n, _ := strconv.Atoi(strings.TrimPrefix(next, itemIDPrefix))
			next = itemIDPrefix + strconv.Itoa(n+1)
		}
		seen[it.ID] = true
		out = append(out, it)
	}
	return out, nil
}

// countedName is "Rope" for one and "3 Rope" for more.
// countedName is "Rope", or "3 Healing Potions" for more than one.
func countedName(n int, name string) string {
	if n == 1 {
		return name
	}
	return fmt.Sprintf("%d %s", n, pluralName(name))
}

// pluralName is an item name in the plural: "Potion of Healing" becomes
// "Potions of Healing", "Torch" "Torches", "Ruby" "Rubies". A name that
// already ends in s ("Soft Boots") is taken as plural already.
func pluralName(name string) string {
	head, tail, found := strings.Cut(name, " of ")
	word := head
	switch lower := strings.ToLower(word); {
	case strings.HasSuffix(lower, "s"):
	case strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "z"), strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "sh"):
		word += "es"
	case len(lower) > 1 && strings.HasSuffix(lower, "y") && !strings.ContainsRune("aeiou", rune(lower[len(lower)-2])):
		word = word[:len(word)-1] + "ies"
	default:
		word += "s"
	}
	if found {
		return word + " of " + tail
	}
	return word
}

func (a *applier) itemAdd(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID string `json:"heroId"`
		Item
	}](payload)
	if err != nil {
		return "", err
	}
	h, err := a.hero(p.HeroID)
	if err != nil {
		return "", err
	}
	items, it, err := AddItem(h.Items, p.Item)
	if err != nil {
		return "", err
	}
	h.Items = items
	added := max(p.Quantity, 1)
	summary := fmt.Sprintf("%s gained %s", h.Name, countedName(added, it.Name))
	var more []string
	if st := it.Summary(); st != "" {
		more = append(more, st)
	}
	if it.Quantity > added {
		// Added to a stack the hero already had.
		more = append(more, fmt.Sprintf("now %d", it.Quantity))
	}
	if len(more) > 0 {
		summary += " (" + strings.Join(more, "; ") + ")"
	}
	return summary, nil
}

func (a *applier) itemRemove(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID   string `json:"heroId"`
		ItemID   string `json:"itemId"`
		Quantity int    `json:"quantity"`
	}](payload)
	if err != nil {
		return "", err
	}
	h, err := a.hero(p.HeroID)
	if err != nil {
		return "", err
	}
	items, it, removed, err := RemoveItem(h.Items, p.ItemID, p.Quantity)
	if err != nil {
		return "", err
	}
	h.Items = items
	if it.Quantity > 0 {
		return fmt.Sprintf("%s lost %d %s (%d left)", h.Name, removed, it.Name, it.Quantity), nil
	}
	return fmt.Sprintf("%s lost %s", h.Name, countedName(removed, it.Name)), nil
}

func (a *applier) itemGive(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID   string `json:"heroId"`
		ItemID   string `json:"itemId"`
		ToHeroID string `json:"toHeroId"`
		Quantity int    `json:"quantity"`
	}](payload)
	if err != nil {
		return "", err
	}
	from, err := a.hero(p.HeroID)
	if err != nil {
		return "", err
	}
	to, err := a.hero(p.ToHeroID)
	if err != nil {
		return "", err
	}
	if from == to {
		return "", errors.New("a hero cannot give an item to themselves")
	}
	i := itemIndex(from.Items, p.ItemID)
	if i < 0 {
		return "", fmt.Errorf("no item %q", p.ItemID)
	}
	given := from.Items[i]
	fromItems, it, moved, err := RemoveItem(from.Items, p.ItemID, p.Quantity)
	if err != nil {
		return "", err
	}
	// The new owner has not equipped it.
	toItems, _, err := AddItem(to.Items, Item{Name: given.Name, Quantity: moved, Notes: given.Notes, Kind: given.Kind, ItemStats: given.ItemStats})
	if err != nil {
		return "", err
	}
	from.Items, to.Items = fromItems, toItems
	return fmt.Sprintf("%s gave %d %s to %s", from.Name, moved, it.Name, to.Name), nil
}

func (a *applier) itemEquip(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID   string `json:"heroId"`
		ItemID   string `json:"itemId"`
		Equipped bool   `json:"equipped"`
	}](payload)
	if err != nil {
		return "", err
	}
	h, err := a.hero(p.HeroID)
	if err != nil {
		return "", err
	}
	i := itemIndex(h.Items, p.ItemID)
	if i < 0 {
		return "", fmt.Errorf("no item %q", p.ItemID)
	}
	it := h.Items[i]
	if it.Equipped == p.Equipped {
		if p.Equipped {
			return "", fmt.Errorf("%s is already equipped", it.Name)
		}
		return "", fmt.Errorf("%s is not equipped", it.Name)
	}
	h.Items = slices.Clone(h.Items)
	h.Items[i].Equipped = p.Equipped
	var notes []string
	verb := "unequipped"
	if p.Equipped {
		verb = "equipped"
		if st := it.Summary(); st != "" {
			notes = append(notes, st)
		}
	}
	if limit := h.ManaCap(); h.Mana > limit {
		notes = append(notes, fmt.Sprintf("mana %d → %d", h.Mana, limit))
		h.Mana = limit
	}
	summary := fmt.Sprintf("%s %s %s", h.Name, verb, it.Name)
	if len(notes) > 0 {
		summary += " (" + strings.Join(notes, "; ") + ")"
	}
	return summary, nil
}

func (a *applier) itemUpdate(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID   string     `json:"heroId"`
		ItemID   string     `json:"itemId"`
		Name     *string    `json:"name"`
		Quantity *int       `json:"quantity"`
		Notes    *string    `json:"notes"`
		Kind     *string    `json:"kind"`
		Stats    *ItemStats `json:"stats"`
		// HealBody and RestoreMana change what using the item does.
		HealBody    *int `json:"healBody"`
		RestoreMana *int `json:"restoreMana"`
	}](payload)
	if err != nil {
		return "", err
	}
	h, err := a.hero(p.HeroID)
	if err != nil {
		return "", err
	}
	i := itemIndex(h.Items, p.ItemID)
	if i < 0 {
		return "", fmt.Errorf("no item %q", p.ItemID)
	}
	before := h.Items[i]
	items, it, err := UpdateItem(h.Items, p.ItemID, ItemPatch{Name: p.Name, Quantity: p.Quantity, Notes: p.Notes, Kind: p.Kind, Stats: p.Stats, HealBody: p.HealBody, RestoreMana: p.RestoreMana})
	if err != nil {
		return "", err
	}
	var changes []string
	if it.Name != before.Name {
		changes = append(changes, fmt.Sprintf("%s renamed to %s", before.Name, it.Name))
	}
	if it.Quantity != before.Quantity {
		changes = append(changes, fmt.Sprintf("%s %d → %d", it.Name, before.Quantity, it.Quantity))
	}
	named := len(changes) > 0
	if it.Kind != before.Kind {
		changes = append(changes, "kind "+changeText(before.Kind, it.Kind))
	}
	if it.ItemStats != before.ItemStats || it.HealBody != before.HealBody || it.RestoreMana != before.RestoreMana {
		changes = append(changes, "stats "+changeText(before.Summary(), it.Summary()))
	}
	if it.Notes != before.Notes {
		changes = append(changes, "notes updated")
	}
	if len(changes) == 0 {
		return "", errors.New("nothing changed")
	}
	if !named {
		changes[0] = it.Name + " " + changes[0]
	}
	h.Items = items
	return fmt.Sprintf("%s: %s", h.Name, strings.Join(changes, ", ")), nil
}

// changeText is "x", "x → y" or "cleared" for a text field set, changed or emptied.
func changeText(before, after string) string {
	switch {
	case after == "":
		return "cleared"
	case before == "":
		return after
	default:
		return before + " → " + after
	}
}

// MaxGold bounds the party's gold.
const MaxGold = 999999

func (a *applier) goldSet(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Gold int `json:"gold"`
	}](payload)
	if err != nil {
		return "", err
	}
	if p.Gold < 0 || p.Gold > MaxGold {
		return "", fmt.Errorf("gold must be from 0 to %d", MaxGold)
	}
	if p.Gold == a.s.Gold {
		return "", errors.New("nothing changed")
	}
	before := a.s.Gold
	a.s.Gold = p.Gold
	return fmt.Sprintf("Party gold %d → %d", before, p.Gold), nil
}

// GoldChange applies what the GM typed to the party's gold: "+25" adds, "-10"
// takes away and "=40" sets it; a bare number is refused, so a slip can't
// replace the purse. Gold never goes below 0.
func GoldChange(current int, input string) (int, error) {
	s := strings.ReplaceAll(strings.TrimSpace(input), " ", "")
	var op byte
	if s != "" {
		op, s = s[0], s[1:]
	}
	n, err := strconv.Atoi(s)
	if (op != '+' && op != '-' && op != '=') || err != nil || n < 0 || strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("gold: %q is not an amount (e.g. +25, -10 or =40)", strings.TrimSpace(input))
	}
	out := n
	switch op {
	case '+':
		out = current + n
	case '-':
		out = current - n
	}
	if out < 0 {
		return 0, fmt.Errorf("gold: taking %d leaves less than 0 (the party has %d)", n, current)
	}
	if out > MaxGold {
		return 0, fmt.Errorf("gold must be at most %d", MaxGold)
	}
	return out, nil
}

// itemUse spends one of a usable item (a potion) and gives its hero the Body
// or mana it restores, up to their maximum, as one change.
func (a *applier) itemUse(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID string `json:"heroId"`
		ItemID string `json:"itemId"`
	}](payload)
	if err != nil {
		return "", err
	}
	h, err := a.hero(p.HeroID)
	if err != nil {
		return "", err
	}
	i := itemIndex(h.Items, p.ItemID)
	if i < 0 {
		return "", fmt.Errorf("no item %q", p.ItemID)
	}
	it := h.Items[i]
	if !it.Usable() {
		return "", fmt.Errorf("%s does nothing when used (give it healing or mana first)", it.Name)
	}
	var effects []string
	if it.HealBody > 0 {
		before := h.Body
		h.Body = max(h.Body, min(h.MaxBody, h.Body+it.HealBody))
		effects = append(effects, fmt.Sprintf("Body %d → %d", before, h.Body))
	}
	if it.RestoreMana > 0 {
		before := h.Mana
		h.Mana = max(h.Mana, min(h.ManaCap(), h.Mana+it.RestoreMana))
		effects = append(effects, fmt.Sprintf("Mana %d → %d", before, h.Mana))
	}
	items, left, _, err := RemoveItem(h.Items, it.ID, 1)
	if err != nil {
		return "", err
	}
	h.Items = items
	rest := "none left"
	if left.Quantity > 0 {
		rest = fmt.Sprintf("%d left", left.Quantity)
	}
	return fmt.Sprintf("%s used %s: %s (%s)", h.Name, it.Name, strings.Join(effects, ", "), rest), nil
}
