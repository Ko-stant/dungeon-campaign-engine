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
)

const itemIDPrefix = "item-"

// Item is something a hero carries. There is no slot or weight limit and
// nothing is equipped; the GM decides what an item does.
type Item struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Notes    string `json:"notes,omitempty"`
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

func checkQuantity(q int) error {
	if q < 1 || q > MaxItemQuantity {
		return fmt.Errorf("quantity must be from 1 to %d", MaxItemQuantity)
	}
	return nil
}

func itemIndex(items []Item, id string) int {
	return slices.IndexFunc(items, func(it Item) bool { return it.ID == id })
}

// AddItem returns items with quantity more of the named item (0 means 1).
// An item with the same name (ignoring case) gains the quantity and keeps
// its notes; otherwise a new item is added. It never modifies items.
func AddItem(items []Item, name string, quantity int, notes string) ([]Item, Item, error) {
	name, err := cleanItemName(name)
	if err != nil {
		return nil, Item{}, err
	}
	if quantity == 0 {
		quantity = 1
	}
	if err := checkQuantity(quantity); err != nil {
		return nil, Item{}, err
	}
	notes = strings.TrimSpace(notes)
	if err := checkItemNotes(notes); err != nil {
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
		return out, out[i], nil
	}
	it := Item{ID: nextItemID(out), Name: name, Quantity: quantity, Notes: notes}
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

// UpdateItem changes an item's name, quantity or notes (nil leaves a field
// alone). A quantity of 0 is a removal, so it is refused here.
func UpdateItem(items []Item, id string, name *string, quantity *int, notes *string) ([]Item, Item, error) {
	i := itemIndex(items, id)
	if i < 0 {
		return nil, Item{}, fmt.Errorf("no item %q", id)
	}
	out := slices.Clone(items)
	it := &out[i]
	if name != nil {
		n, err := cleanItemName(*name)
		if err != nil {
			return nil, Item{}, err
		}
		it.Name = n
	}
	if quantity != nil {
		if err := checkQuantity(*quantity); err != nil {
			return nil, Item{}, err
		}
		it.Quantity = *quantity
	}
	if notes != nil {
		n := strings.TrimSpace(*notes)
		if err := checkItemNotes(n); err != nil {
			return nil, Item{}, err
		}
		it.Notes = n
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
func countedName(n int, name string) string {
	if n == 1 {
		return name
	}
	return fmt.Sprintf("%d %s", n, name)
}

func (a *applier) itemAdd(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID   string `json:"heroId"`
		Name     string `json:"name"`
		Quantity int    `json:"quantity"`
		Notes    string `json:"notes"`
	}](payload)
	if err != nil {
		return "", err
	}
	h, err := a.hero(p.HeroID)
	if err != nil {
		return "", err
	}
	items, it, err := AddItem(h.Items, p.Name, p.Quantity, p.Notes)
	if err != nil {
		return "", err
	}
	h.Items = items
	added := max(p.Quantity, 1)
	summary := fmt.Sprintf("%s gained %s", h.Name, countedName(added, it.Name))
	if it.Quantity > 1 {
		summary += fmt.Sprintf(" (now %d)", it.Quantity)
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
	notes := from.Items[i].Notes
	fromItems, it, moved, err := RemoveItem(from.Items, p.ItemID, p.Quantity)
	if err != nil {
		return "", err
	}
	toItems, _, err := AddItem(to.Items, it.Name, moved, notes)
	if err != nil {
		return "", err
	}
	from.Items, to.Items = fromItems, toItems
	return fmt.Sprintf("%s gave %d %s to %s", from.Name, moved, it.Name, to.Name), nil
}

func (a *applier) itemUpdate(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		HeroID   string  `json:"heroId"`
		ItemID   string  `json:"itemId"`
		Name     *string `json:"name"`
		Quantity *int    `json:"quantity"`
		Notes    *string `json:"notes"`
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
	items, it, err := UpdateItem(h.Items, p.ItemID, p.Name, p.Quantity, p.Notes)
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
	if it.Notes != before.Notes {
		changes = append(changes, "notes updated")
	}
	if len(changes) == 0 {
		return "", errors.New("nothing changed")
	}
	h.Items = items
	return fmt.Sprintf("%s: %s", h.Name, strings.Join(changes, ", ")), nil
}

// MaxGold bounds a hero's gold.
const MaxGold = 999999

// GoldChange applies what the GM typed to a hero's gold: "+25" adds, "-10"
// takes away and a plain number sets it. Gold never goes below 0.
func GoldChange(current int, input string) (int, error) {
	s := strings.ReplaceAll(strings.TrimSpace(input), " ", "")
	sign := 0
	if strings.HasPrefix(s, "+") {
		sign, s = 1, s[1:]
	} else if strings.HasPrefix(s, "-") {
		sign, s = -1, s[1:]
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("gold: %q is not an amount (e.g. 40, +25 or -10)", strings.TrimSpace(input))
	}
	out := n
	if sign != 0 {
		out = current + sign*n
	}
	if out < 0 {
		return 0, fmt.Errorf("gold: taking %d leaves less than 0 (the hero has %d)", n, current)
	}
	if out > MaxGold {
		return 0, fmt.Errorf("gold must be at most %d", MaxGold)
	}
	return out, nil
}
