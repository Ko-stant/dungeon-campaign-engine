package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// MaxLogNoteLength bounds free-text log notes.
const MaxLogNoteLength = 2000

// Command is one GM change, as sent by the tracker page.
type Command struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Event describes an applied command for the session log.
type Event struct {
	Round   int             `json:"round"`
	Kind    string          `json:"kind"`
	Summary string          `json:"summary"`
	Payload json.RawMessage `json:"payload"`
	// PlayerSummary is the line the player screen shows (see player.go);
	// empty when the players hear nothing about this change.
	PlayerSummary string `json:"playerSummary,omitempty"`
}

// Apply returns the state after the command and the event describing it. It
// never modifies s. Errors mean the command itself is malformed (unknown ids,
// squares off the board, invalid values); no game rule is ever enforced.
func Apply(s *State, c Command, catalog *content.Catalog) (*State, Event, error) {
	if s.Version != StateVersion {
		return nil, Event{}, fmt.Errorf("session state version %d is not the current version %d (squares now count from the bottom-left); start a new session", s.Version, StateVersion)
	}
	next, err := clone(s)
	if err != nil {
		return nil, Event{}, err
	}
	a := applier{s: next, catalog: catalog}

	var summary string
	switch c.Type {
	case "move":
		summary, err = a.move(c.Payload)
	case "hero.update":
		summary, err = a.heroUpdate(c.Payload)
	case "monster.add":
		summary, err = a.monsterAdd(c.Payload)
	case "monster.update":
		summary, err = a.monsterUpdate(c.Payload)
	case "monster.remove":
		summary, err = a.monsterRemove(c.Payload)
	case "door.set":
		summary, err = a.doorSet(c.Payload)
	case "trap.set":
		summary, err = a.trapSet(c.Payload)
	case "blocked.set":
		summary, err = a.blockedSet(c.Payload)
	case "area.reveal":
		summary, err = a.areaReveal(c.Payload)
	case "tiles.reveal":
		summary, err = a.tilesSet(c.Payload, true)
	case "tiles.hide":
		summary, err = a.tilesSet(c.Payload, false)
	case "note.consume":
		summary, err = a.noteConsume(c.Payload)
	case "round.advance":
		summary = a.roundAdvance()
	case "fight.start":
		summary, err = a.fightStart(c.Payload)
	case "fight.end":
		summary, err = a.fightEnd(c.Payload)
	case "effect.add":
		summary, err = a.effectAdd(c.Payload)
	case "effect.remove":
		summary, err = a.effectRemove(c.Payload)
	case "round.set":
		summary, err = a.roundSet(c.Payload)
	case "ability.use":
		summary, err = a.abilityUse(c.Payload)
	case "ability.reset":
		summary, err = a.abilityReset(c.Payload)
	case "item.add":
		summary, err = a.itemAdd(c.Payload)
	case "item.update":
		summary, err = a.itemUpdate(c.Payload)
	case "item.remove":
		summary, err = a.itemRemove(c.Payload)
	case "item.give":
		summary, err = a.itemGive(c.Payload)
	case "trap.trigger":
		summary, err = a.trapTrigger(c.Payload)
	case "trap.block":
		summary, err = a.trapBlock(c.Payload)
	case "block.add":
		summary, err = a.blockAdd(c.Payload)
	case "block.remove":
		summary, err = a.blockRemove(c.Payload)
	case "seen.set":
		summary, err = a.seenSet(c.Payload)
	case "players.set":
		summary, err = a.playersSet(c.Payload)
	case "item.equip":
		summary, err = a.itemEquip(c.Payload)
	case "gold.set":
		summary, err = a.goldSet(c.Payload)
	case "passage.read":
		summary, err = a.passageRead(c.Payload)
	case "log.note":
		summary, err = a.logNote(c.Payload)
	default:
		err = fmt.Errorf("unknown command %q", c.Type)
	}
	if err != nil {
		return nil, Event{}, fmt.Errorf("%s: %w", c.Type, err)
	}

	cmdPayload := c.Payload
	if len(cmdPayload) == 0 {
		cmdPayload = json.RawMessage(`{}`)
	}
	payload, err := json.Marshal(map[string]json.RawMessage{"command": cmdPayload})
	if err != nil {
		return nil, Event{}, err
	}
	return next, Event{Round: next.Round, Kind: c.Type, Summary: summary, Payload: payload, PlayerSummary: PlayerSummary(s, next, c, summary)}, nil
}

func clone(s *State) (*State, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var out State
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type applier struct {
	s       *State
	catalog *content.Catalog
}

func decode[T any](payload json.RawMessage) (T, error) {
	var v T
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("invalid payload: %w", err)
	}
	return v, nil
}

func (a *applier) onBoard(x, y int) error {
	if !a.s.Board.OnBoard(x, y) {
		return fmt.Errorf("square (%d,%d) is off the %dx%d board", x, y, a.s.Board.Width, a.s.Board.Height)
	}
	return nil
}

func (a *applier) hero(id string) (*Hero, error) {
	for i := range a.s.Heroes {
		if a.s.Heroes[i].ID == id {
			return &a.s.Heroes[i], nil
		}
	}
	return nil, fmt.Errorf("no hero %q", id)
}

func (a *applier) monster(id string) (*Monster, int, error) {
	for i := range a.s.Monsters {
		if a.s.Monsters[i].ID == id {
			return &a.s.Monsters[i], i, nil
		}
	}
	return nil, -1, fmt.Errorf("no monster %q", id)
}

func (a *applier) heroLabel(h *Hero) string {
	class := h.Class
	if def, ok := a.catalog.Hero(h.Class); ok {
		class = def.Name
	}
	return fmt.Sprintf("%s (%s)", h.Name, class)
}

func monsterLabel(m *Monster) string {
	return fmt.Sprintf("%s (%s)", m.Name, m.ID)
}

// trap finds a trap's live state and its quest entry.
func (a *applier) trap(id string) (*TrapState, maps.Trap, error) {
	for i := range a.s.Traps {
		if a.s.Traps[i].ID != id {
			continue
		}
		for _, qt := range a.s.Quest.Traps {
			if qt.ID == id {
				return &a.s.Traps[i], qt, nil
			}
		}
		return &a.s.Traps[i], maps.Trap{ID: id}, nil
	}
	return nil, maps.Trap{}, fmt.Errorf("no trap %q", id)
}

// trapKindLabel names a trap for the log: its catalog name (or kind), plus its label.
func (a *applier) trapKindLabel(qt maps.Trap) string {
	name := strings.ReplaceAll(qt.Kind, "_", " ")
	if qt.Kind == maps.TrapTrigger {
		name = "Trigger"
	}
	if a.catalog != nil {
		if def, ok := a.catalog.TrapByID(qt.Kind); ok {
			name = def.Name
		}
	}
	if name == "" {
		name = "trap"
	}
	if qt.Label != "" {
		name += " " + qt.Label
	}
	return name
}

// trapAt is where a trap is now: moved during play, or where the quest put it.
func trapAt(t *TrapState, qt maps.Trap) maps.Tile {
	if t.At != nil {
		return *t.At
	}
	return maps.Tile{X: qt.X, Y: qt.Y}
}

func (a *applier) move(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID string `json:"id"`
		X  int    `json:"x"`
		Y  int    `json:"y"`
	}](payload)
	if err != nil {
		return "", err
	}
	if err := a.onBoard(p.X, p.Y); err != nil {
		return "", err
	}
	if h, err := a.hero(p.ID); err == nil {
		wasPlaced, fromX, fromY := h.Placed, h.X, h.Y
		h.X, h.Y, h.Placed = p.X, p.Y, true
		if !wasPlaced {
			return fmt.Sprintf("Placed %s at (%d,%d)", a.heroLabel(h), p.X, p.Y), nil
		}
		return fmt.Sprintf("Moved %s from (%d,%d) to (%d,%d)", a.heroLabel(h), fromX, fromY, p.X, p.Y), nil
	}
	if m, _, err := a.monster(p.ID); err == nil {
		fromX, fromY := m.X, m.Y
		m.X, m.Y = p.X, p.Y
		return fmt.Sprintf("Moved %s from (%d,%d) to (%d,%d)", monsterLabel(m), fromX, fromY, p.X, p.Y), nil
	}
	t, qt, err := a.trap(p.ID)
	if err != nil {
		return "", fmt.Errorf("no hero, monster or trap %q", p.ID)
	}
	from := trapAt(t, qt)
	t.At = &maps.Tile{X: p.X, Y: p.Y}
	return fmt.Sprintf("Moved trap %s (%s) from (%d,%d) to (%d,%d)", t.ID, a.trapKindLabel(qt), from.X, from.Y, p.X, p.Y), nil
}

func nonNegative(name string, v *int) error {
	if v != nil && *v < 0 {
		return fmt.Errorf("%s must not be negative", name)
	}
	return nil
}

func (a *applier) heroUpdate(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID        string  `json:"id"`
		Body      *int    `json:"body"`
		MaxBody   *int    `json:"maxBody"`
		Mind      *int    `json:"mind"`
		MaxMind   *int    `json:"maxMind"`
		Mana      *int    `json:"mana"`
		MaxMana   *int    `json:"maxMana"`
		Equipment *string `json:"equipment"`
		Notes     *string `json:"notes"`
		Status    *string `json:"status"`
	}](payload)
	if err != nil {
		return "", err
	}
	h, err := a.hero(p.ID)
	if err != nil {
		return "", err
	}
	for name, v := range map[string]*int{"body": p.Body, "maxBody": p.MaxBody, "mind": p.Mind, "maxMind": p.MaxMind, "mana": p.Mana, "maxMana": p.MaxMana} {
		if err := nonNegative(name, v); err != nil {
			return "", err
		}
	}
	if p.Status != nil && *p.Status != HeroActive && *p.Status != HeroDead && *p.Status != HeroEscaped {
		return "", fmt.Errorf("invalid status %q", *p.Status)
	}

	var changes []string
	setInt := func(label string, field *int, v *int) {
		if v != nil && *v != *field {
			changes = append(changes, fmt.Sprintf("%s %d → %d", label, *field, *v))
			*field = *v
		}
	}
	setInt("body", &h.Body, p.Body)
	setInt("max body", &h.MaxBody, p.MaxBody)
	setInt("mind", &h.Mind, p.Mind)
	setInt("max mind", &h.MaxMind, p.MaxMind)
	setInt("mana", &h.Mana, p.Mana)
	setInt("max mana", &h.MaxMana, p.MaxMana)
	if p.Equipment != nil && *p.Equipment != h.Equipment {
		h.Equipment = *p.Equipment
		changes = append(changes, "equipment updated")
	}
	if p.Notes != nil && *p.Notes != h.Notes {
		h.Notes = *p.Notes
		changes = append(changes, "notes updated")
	}
	if p.Status != nil && *p.Status != h.Status {
		changes = append(changes, fmt.Sprintf("status %s → %s", h.Status, *p.Status))
		h.Status = *p.Status
	}
	if len(changes) == 0 {
		return "", errors.New("nothing changed")
	}
	return fmt.Sprintf("%s: %s", h.Name, strings.Join(changes, ", ")), nil
}

func (a *applier) nextMonsterID() string {
	highest := 0
	for _, m := range a.s.Monsters {
		if n, err := strconv.Atoi(strings.TrimPrefix(m.ID, "monster-")); err == nil && strings.HasPrefix(m.ID, "monster-") && n > highest {
			highest = n
		}
	}
	return fmt.Sprintf("monster-%d", highest+1)
}

func validVisibility(v string) bool {
	return v == MonsterHidden || v == MonsterSeen
}

func (a *applier) monsterAdd(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Type       string `json:"type"`
		X          int    `json:"x"`
		Y          int    `json:"y"`
		Visibility string `json:"visibility"`
	}](payload)
	if err != nil {
		return "", err
	}
	def, ok := a.catalog.Monster(p.Type)
	if !ok {
		return "", fmt.Errorf("unknown monster type %q", p.Type)
	}
	if err := a.onBoard(p.X, p.Y); err != nil {
		return "", err
	}
	if p.Visibility == "" {
		p.Visibility = MonsterSeen
	}
	if !validVisibility(p.Visibility) {
		return "", fmt.Errorf("invalid visibility %q", p.Visibility)
	}
	m := Monster{
		ID: a.nextMonsterID(), Type: def.ID, Name: def.Name, X: p.X, Y: p.Y,
		Body: def.Body, MaxBody: def.Body, Mind: def.Mind, Visibility: p.Visibility, Alive: true,
		Color: def.Color, Combat: combatCopy(def.Combat),
	}
	m.Width, m.Height = def.Size()
	a.s.Monsters = append(a.s.Monsters, m)
	return fmt.Sprintf("Added %s at (%d,%d)", monsterLabel(&m), p.X, p.Y), nil
}

func (a *applier) monsterUpdate(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID         string  `json:"id"`
		Body       *int    `json:"body"`
		MaxBody    *int    `json:"maxBody"`
		Mind       *int    `json:"mind"`
		Visibility *string `json:"visibility"`
		Alive      *bool   `json:"alive"`
		Notes      *string `json:"notes"`
	}](payload)
	if err != nil {
		return "", err
	}
	m, _, err := a.monster(p.ID)
	if err != nil {
		return "", err
	}
	for name, v := range map[string]*int{"body": p.Body, "maxBody": p.MaxBody, "mind": p.Mind} {
		if err := nonNegative(name, v); err != nil {
			return "", err
		}
	}
	if p.Visibility != nil && !validVisibility(*p.Visibility) {
		return "", fmt.Errorf("invalid visibility %q", *p.Visibility)
	}

	var changes []string
	setInt := func(label string, field *int, v *int) {
		if v != nil && *v != *field {
			changes = append(changes, fmt.Sprintf("%s %d → %d", label, *field, *v))
			*field = *v
		}
	}
	setInt("body", &m.Body, p.Body)
	setInt("max body", &m.MaxBody, p.MaxBody)
	setInt("mind", &m.Mind, p.Mind)
	if p.Visibility != nil && *p.Visibility != m.Visibility {
		m.Visibility = *p.Visibility
		changes = append(changes, "now "+m.Visibility)
	}
	if p.Alive != nil && *p.Alive != m.Alive {
		m.Alive = *p.Alive
		if m.Alive {
			changes = append(changes, "revived")
		} else {
			changes = append(changes, "killed")
		}
	}
	if p.Notes != nil && *p.Notes != m.Notes {
		m.Notes = *p.Notes
		changes = append(changes, "notes updated")
	}
	if len(changes) == 0 {
		return "", errors.New("nothing changed")
	}
	return fmt.Sprintf("%s: %s", monsterLabel(m), strings.Join(changes, ", ")), nil
}

func (a *applier) monsterRemove(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID string `json:"id"`
	}](payload)
	if err != nil {
		return "", err
	}
	m, i, err := a.monster(p.ID)
	if err != nil {
		return "", err
	}
	label := monsterLabel(m)
	a.s.Monsters = slices.Delete(a.s.Monsters, i, i+1)
	return "Removed " + label, nil
}

func (a *applier) doorSet(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID     string  `json:"id"`
		State  *string `json:"state"`
		Found  *bool   `json:"found"`
		Locked *bool   `json:"locked"`
	}](payload)
	if err != nil {
		return "", err
	}
	var door *DoorState
	for i := range a.s.Doors {
		if a.s.Doors[i].ID == p.ID {
			door = &a.s.Doors[i]
		}
	}
	if door == nil {
		return "", fmt.Errorf("no door %q", p.ID)
	}
	if p.State == nil && p.Found == nil && p.Locked == nil {
		return "", errors.New("nothing to change")
	}
	label := a.doorLabel(door.ID)
	var parts []string
	if p.Found != nil {
		door.Found = *p.Found
		if *p.Found {
			parts = append(parts, "Found secret door "+door.ID)
		} else {
			parts = append(parts, "Hid secret door "+door.ID)
		}
	}
	if p.State != nil {
		switch *p.State {
		case maps.DoorOpen:
			parts = append(parts, "Opened "+label)
		case maps.DoorClosed:
			parts = append(parts, "Closed "+label)
		default:
			return "", fmt.Errorf("invalid door state %q", *p.State)
		}
		door.State = *p.State
	}
	if p.Locked != nil {
		door.Locked = *p.Locked
		if *p.Locked {
			parts = append(parts, "Locked "+label)
		} else {
			parts = append(parts, "Unlocked "+label)
		}
	}
	return strings.Join(parts, "; "), nil
}

func (a *applier) blockedSet(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID      string `json:"id"`
		Removed bool   `json:"removed"`
	}](payload)
	if err != nil {
		return "", err
	}
	i := slices.IndexFunc(a.s.Quest.BlockedSquares, func(r maps.Rect) bool { return r.ID == p.ID })
	if i < 0 {
		return "", fmt.Errorf("no blocked square %q", p.ID)
	}
	r := a.s.Quest.BlockedSquares[i]
	a.s.RemovedBlocks = slices.DeleteFunc(a.s.RemovedBlocks, func(id string) bool { return id == r.ID })
	if !p.Removed {
		return fmt.Sprintf("Put back %s at (%d,%d)", r.ID, r.X, r.Y), nil
	}
	a.s.RemovedBlocks = append(a.s.RemovedBlocks, r.ID)
	if r.HiddenDoor {
		return fmt.Sprintf("Found the secret door at (%d,%d) (%s removed)", r.X, r.Y, r.ID), nil
	}
	return fmt.Sprintf("Removed %s at (%d,%d)", r.ID, r.X, r.Y), nil
}

func (a *applier) trapSet(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}](payload)
	if err != nil {
		return "", err
	}
	switch p.State {
	case maps.TrapHidden, maps.TrapRevealed, maps.TrapTriggered, maps.TrapDisarmed, maps.TrapRemoved:
	default:
		return "", fmt.Errorf("invalid trap state %q", p.State)
	}
	t, qt, err := a.trap(p.ID)
	if err != nil {
		return "", err
	}
	from := t.State
	t.State = p.State
	return fmt.Sprintf("Trap %s (%s): %s → %s", t.ID, a.trapKindLabel(qt), from, p.State), nil
}

func (a *applier) setDiscovered(indexes []int, discovered bool) {
	set := make(map[int]bool, len(a.s.Discovered))
	for _, i := range a.s.Discovered {
		set[i] = true
	}
	for _, i := range indexes {
		if discovered {
			set[i] = true
		} else {
			delete(set, i)
		}
	}
	a.s.Discovered = sortedKeys(set)
}

func squares(n int) string {
	if n == 1 {
		return "1 square"
	}
	return fmt.Sprintf("%d squares", n)
}

func (a *applier) areaReveal(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		X    int  `json:"x"`
		Y    int  `json:"y"`
		Seen bool `json:"seen"`
	}](payload)
	if err != nil {
		return "", err
	}
	if err := a.onBoard(p.X, p.Y); err != nil {
		return "", err
	}
	tiles := a.s.areaTiles(p.X, p.Y)
	a.setDiscovered(tiles, true)
	shown := ""
	if p.Seen {
		shown = a.showContentsOn(tiles)
	}
	region := a.s.Board.RegionAt(p.X, p.Y)
	for _, r := range a.s.Board.Rooms {
		if r.ID == region {
			return "Revealed " + r.Name + shown, nil
		}
	}
	return "Revealed " + squares(len(tiles)) + shown, nil
}

func (a *applier) tilesSet(payload json.RawMessage, discovered bool) (string, error) {
	p, err := decode[struct {
		Tiles []maps.Tile `json:"tiles"`
		Seen  bool        `json:"seen"`
	}](payload)
	if err != nil {
		return "", err
	}
	var indexes []int
	for _, t := range p.Tiles {
		if a.onBoard(t.X, t.Y) == nil {
			indexes = append(indexes, a.s.Board.Index(t.X, t.Y))
		}
	}
	if len(indexes) == 0 {
		return "", errors.New("no squares on the board")
	}
	a.setDiscovered(indexes, discovered)
	if discovered {
		shown := ""
		if p.Seen {
			shown = a.showContentsOn(indexes)
		}
		return "Revealed " + squares(len(indexes)) + shown, nil
	}
	return "Hid " + squares(len(indexes)), nil
}

func (a *applier) noteConsume(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID       string `json:"id"`
		Consumed bool   `json:"consumed"`
	}](payload)
	if err != nil {
		return "", err
	}
	label := ""
	for _, n := range a.s.Quest.Notes {
		if n.ID == p.ID {
			label = n.Label
		}
	}
	if label == "" {
		return "", fmt.Errorf("no note %q", p.ID)
	}
	a.s.ConsumedNotes = slices.DeleteFunc(a.s.ConsumedNotes, func(id string) bool { return id == p.ID })
	if p.Consumed {
		a.s.ConsumedNotes = append(a.s.ConsumedNotes, p.ID)
		slices.Sort(a.s.ConsumedNotes)
		return "Used note " + label, nil
	}
	return "Restored note " + label, nil
}

func (a *applier) roundSet(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Round int `json:"round"`
	}](payload)
	if err != nil {
		return "", err
	}
	if p.Round < 1 {
		return "", errors.New("round must be at least 1")
	}
	a.s.Round = p.Round
	summary := fmt.Sprintf("Round set to %d", p.Round)
	if ready := a.readyAgain(); len(ready) > 0 {
		summary += "; ready again: " + strings.Join(ready, ", ")
	}
	return summary, nil
}

func (a *applier) logNote(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Text string `json:"text"`
	}](payload)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(p.Text)
	if text == "" {
		return "", errors.New("note text is empty")
	}
	if len(text) > MaxLogNoteLength {
		return "", fmt.Errorf("note text must be at most %d characters", MaxLogNoteLength)
	}
	return text, nil
}
