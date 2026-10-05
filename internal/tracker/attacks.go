package tracker

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/combat"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/dice"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

// Attacks in rules mode (RULES_AND_CLASSES.md, "Combat"; ONLINE_RULES.md):
// a hero's basic attack is their turn's action; a monster attacks once per
// round in the monsters' phase. The engine rolls the dice (or uses the GM's)
// and applies the result.

const (
	// determinationStep is the Accuracy each miss in a row adds, up to determinationCap.
	determinationStep = 2
	determinationCap  = 4
	// falterPenalty is the Avoidance a badly hurt monster loses (see wounded).
	falterPenalty = 4
	// critMultiplier and nearMiss are the same for every class.
	critMultiplier = 2
	nearMissBonus  = 2
	// reachLine is a monster's reach: adjacent, or two squares away in a
	// straight line past whatever stands between (never diagonal).
	reachLine = "line"
)

// footprint lists the squares a monster covers.
func (m *Monster) footprint() []maps.Tile {
	return rectTiles(m.X, m.Y, m.Width, m.Height)
}

// piecesBut reports squares holding a hero or monster other than those named:
// what blocks attack sight between an attacker and a target.
func (a *applier) piecesBut(ids ...string) func(maps.Tile) bool {
	return func(t maps.Tile) bool {
		if h := a.heroAt(t, ""); h != nil && !slices.Contains(ids, h.ID) {
			return true
		}
		m := a.monsterAt(t, "")
		return m != nil && !slices.Contains(ids, m.ID)
	}
}

// reaches reports whether an attack from any of the squares from reaches any
// of the squares to. Adjacent means orthogonally beside with no wall or
// closed door between; diagonal adds the diagonal squares (around a corner
// only when the corner leaves a gap); line adds the square two away in a
// straight line, past a piece but not a wall, closed door or tall
// furniture; sight is a clear line of attack sight at any range, with pieces
// in between blocking it.
func (a *applier) reaches(from, to []maps.Tile, reach string, pieces func(maps.Tile) bool) bool {
	t := a.terrain()
	for _, f := range from {
		for _, g := range to {
			switch {
			case maps.Adjacent(f, g, false):
				if e, _ := maps.EdgeBetween(f, g); t.Passable(e) {
					return true
				}
			case reach == content.ReachDiagonal && maps.Adjacent(f, g, true):
				if maps.LineOfSight(t, f, g, nil) {
					return true
				}
			case reach == reachLine && straightTwo(f, g):
				mid := maps.Tile{X: (f.X + g.X) / 2, Y: (f.Y + g.Y) / 2}
				e1, _ := maps.EdgeBetween(f, mid)
				e2, _ := maps.EdgeBetween(mid, g)
				if t.Passable(e1) && t.Passable(e2) && !t.BlocksSight(mid) {
					return true
				}
			case reach == content.ReachSight:
				if maps.LineOfSight(t, f, g, pieces) {
					return true
				}
			}
		}
	}
	return false
}

// straightTwo reports whether two squares are two apart in a row or column.
func straightTwo(a, b maps.Tile) bool {
	dx, dy := b.X-a.X, b.Y-a.Y
	return (dx == 0 && (dy == 2 || dy == -2)) || (dy == 0 && (dx == 2 || dx == -2))
}

// diceDetail is "1d20: 15" (or "2d8: 6, 5") for the first dice of an
// expression in rolls, and how many rolls that used.
func diceDetail(e dice.Expr, rolls []combat.Die) (string, int, int) {
	n := 0
	for _, t := range e.Terms {
		n += t.Count
	}
	n = min(n, len(rolls))
	total, parts := e.Modifier, make([]string, n)
	for i, d := range rolls[:n] {
		parts[i] = strconv.Itoa(d.Value)
		total += d.Value
	}
	return e.String() + ": " + strings.Join(parts, ", "), n, total
}

func signed(label string, v int) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf(", %s %+d", label, v)
}

// seenMonster finds a living monster the heroes can see.
func (a *applier) seenMonster(id string) (*Monster, error) {
	m, _, err := a.monster(id)
	if err != nil || !m.Alive || m.Visibility != MonsterSeen {
		return nil, fmt.Errorf("no monster in view called %q", id)
	}
	return m, nil
}

func (a *applier) turnAttack(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Target string `json:"target"`
	}](payload)
	if err != nil {
		return "", err
	}
	turn, h, err := a.activeTurn()
	if err != nil {
		return "", err
	}
	if turn.Acted {
		return "", fmt.Errorf("%s has already acted this turn", h.Name)
	}
	m, err := a.seenMonster(p.Target)
	if err != nil {
		return "", err
	}
	totals := h.CombatTotals()
	switch {
	case totals == nil:
		return "", fmt.Errorf("%s's class has no combat stats", h.Name)
	case m.Combat == nil:
		return "", fmt.Errorf("%s has no combat stats", monsterLabel(m))
	}
	from := []maps.Tile{{X: h.X, Y: h.Y}}
	if !a.reaches(from, m.footprint(), totals.Reach, a.piecesBut(h.ID, m.ID)) {
		if totals.Reach == content.ReachSight {
			return "", fmt.Errorf("%s has no line of sight to %s", h.Name, monsterLabel(m))
		}
		return "", fmt.Errorf("%s is out of reach of %s", monsterLabel(m), h.Name)
	}
	hitDice, err := dice.Parse(totals.HitDice)
	if err != nil {
		return "", fmt.Errorf("%s's hit dice: %w", h.Name, err)
	}
	r, err := a.roller()
	if err != nil {
		return "", err
	}

	avoidance, faltering := m.Combat.Avoidance, ""
	if wounded(m) {
		avoidance, faltering = avoidance-falterPenalty, ", faltering"
	}
	start := len(a.dice.log.Rolls)
	res := combat.HeroStrike(combat.HeroAttacker{
		HitDice: hitDice, Accuracy: totals.Accuracy, CritFrom: totals.CritFrom,
		CritMultiplier: critMultiplier, NearMiss: nearMissBonus, Damage: totals.Damage,
	}, avoidance, r, combat.StrikeOptions{Bonus: h.Determination})
	rolls := a.dice.log.Rolls[start:]
	detail, n, rolled := diceDetail(hitDice, rolls)
	total := rolled + totals.Accuracy + h.Determination
	critDie := 0
	if n < len(rolls) {
		critDie = rolls[n].Value
	}
	summary := fmt.Sprintf("%s attacks %s: %d vs %d%s (%s%s%s; crit die %d)", a.heroLabel(h), monsterLabel(m), total, avoidance, faltering,
		detail, signed("Accuracy", totals.Accuracy), signed("Determination", h.Determination), critDie)

	turn.Acted = true
	if turn.MoveRoll > 0 {
		turn.MoveDone = true
	}
	if !res.Hit {
		h.Determination = min(h.Determination+determinationStep, determinationCap)
		if res.CriticalMiss {
			a.s.Rules.SkipNext = append(a.s.Rules.SkipNext, h.ID)
			return fmt.Sprintf("%s: critical miss, %s loses their next turn (Determination +%d)", summary, h.Name, h.Determination), nil
		}
		return fmt.Sprintf("%s: miss (Determination +%d)", summary, h.Determination), nil
	}
	h.Determination = 0
	switch {
	case res.Crit:
		summary += fmt.Sprintf(": crit for %d", res.Damage)
	case total < avoidance:
		summary += fmt.Sprintf(": near miss, a hit for %d", res.Damage)
	default:
		summary += fmt.Sprintf(": hit for %d", res.Damage)
	}
	m.Body = max(0, m.Body-res.Damage)
	if m.Body == 0 {
		m.Alive = false
		return summary + "; " + monsterLabel(m) + " dies", nil
	}
	return summary + fmt.Sprintf(" (%s %d/%d)", m.Name, m.Body, m.MaxBody), nil
}

// monsterTurn finds a living monster for a monsters' phase command.
func (a *applier) monsterTurn(id string) (*RulesState, *Monster, error) {
	r, err := a.rules()
	if err != nil {
		return nil, nil, err
	}
	if r.Phase != PhaseMonsters {
		return nil, nil, fmt.Errorf("it is the %s' phase; monsters act in the monsters' phase", r.Phase)
	}
	m, _, err := a.monster(id)
	if err != nil {
		return nil, nil, err
	}
	if !m.Alive {
		return nil, nil, fmt.Errorf("%s is dead", monsterLabel(m))
	}
	return r, m, nil
}

func (a *applier) monsterAttack(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Monster string `json:"monster"`
		Target  string `json:"target"`
	}](payload)
	if err != nil {
		return "", err
	}
	r, m, err := a.monsterTurn(p.Monster)
	if err != nil {
		return "", err
	}
	if slices.Contains(r.MonstersActed, m.ID) {
		return "", fmt.Errorf("%s has already attacked this round", monsterLabel(m))
	}
	h, err := a.hero(p.Target)
	if err != nil {
		return "", err
	}
	if h.Status != HeroActive || !h.Placed {
		return "", fmt.Errorf("%s is not on the board", h.Name)
	}
	totals := h.CombatTotals()
	switch {
	case m.Combat == nil:
		return "", fmt.Errorf("%s has no combat stats", monsterLabel(m))
	case totals == nil:
		return "", fmt.Errorf("%s's class has no combat stats", h.Name)
	}
	reach := content.ReachAdjacent
	switch {
	case m.Combat.Ranged:
		reach = content.ReachSight
	case m.Combat.Reach:
		reach = reachLine
	}
	if !a.reaches(m.footprint(), []maps.Tile{{X: h.X, Y: h.Y}}, reach, a.piecesBut(h.ID, m.ID)) {
		return "", fmt.Errorf("%s is out of reach of %s", h.Name, monsterLabel(m))
	}
	hitDice, err := dice.Parse(m.Combat.HitDice)
	if err != nil {
		return "", fmt.Errorf("%s's hit dice: %w", monsterLabel(m), err)
	}
	defenseDice, err := dice.Parse(totals.DefenseDice)
	if err != nil {
		return "", fmt.Errorf("%s's defense dice: %w", h.Name, err)
	}
	roller, err := a.roller()
	if err != nil {
		return "", err
	}

	start := len(a.dice.log.Rolls)
	res := combat.MonsterStrike(combat.MonsterAttacker{HitDice: hitDice, Damage: m.Combat.Damage},
		combat.HeroDefender{DefenseDice: defenseDice, BaseAvoidance: totals.Avoidance, Mitigation: totals.Mitigation}, roller, true)
	rolls := a.dice.log.Rolls[start:]
	attackDetail, n, attack := diceDetail(hitDice, rolls)
	defenseDetail, _, defense := diceDetail(defenseDice, rolls[n:])
	defense += totals.Avoidance
	r.MonstersActed = append(r.MonstersActed, m.ID)

	summary := fmt.Sprintf("%s attacks %s: %d vs %d (%s; %s%s)", monsterLabel(m), a.heroLabel(h), attack, defense,
		attackDetail, defenseDetail, signed("Avoidance", totals.Avoidance))
	if !res.Hit {
		return summary + ": miss", nil
	}
	if res.Crit {
		summary += fmt.Sprintf(": crit for %d", res.Damage)
	} else {
		summary += fmt.Sprintf(": hit for %d", res.Damage)
	}
	h.Body = max(0, h.Body-res.Damage)
	if h.Body == 0 {
		h.Status = HeroDead
		return summary + "; " + a.heroLabel(h) + " falls", nil
	}
	return summary + fmt.Sprintf(" (%s %d/%d)", h.Name, h.Body, h.MaxBody), nil
}

// monsterMove moves a monster up to its movement, through other monsters
// but not heroes, to a square where it fits. Monsters don't set off traps.
// A monster coming into the heroes' view is revealed.
func (a *applier) monsterMove(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		Monster string    `json:"monster"`
		To      maps.Tile `json:"to"`
	}](payload)
	if err != nil {
		return "", err
	}
	r, m, err := a.monsterTurn(p.Monster)
	if err != nil {
		return "", err
	}
	if slices.Contains(r.MonstersMoved, m.ID) {
		return "", fmt.Errorf("%s has already moved this round", monsterLabel(m))
	}
	if m.Movement <= 0 {
		return "", fmt.Errorf("%s does not move", monsterLabel(m))
	}
	if err := a.onBoard(p.To.X, p.To.Y); err != nil {
		return "", err
	}
	for _, t := range rectTiles(p.To.X, p.To.Y, m.Width, m.Height) {
		if h := a.heroAt(t, ""); h != nil {
			return "", fmt.Errorf("%s is at (%d,%d)", h.Name, t.X, t.Y)
		}
	}
	other := func(t maps.Tile) bool { return a.monsterAt(t, m.ID) != nil }
	move := maps.Move{
		From: maps.Tile{X: m.X, Y: m.Y}, Width: m.Width, Height: m.Height, Budget: a.s.Board.Width * a.s.Board.Height,
		Occupied: func(t maps.Tile) bool { return a.heroAt(t, "") != nil || other(t) },
		Through:  other,
	}
	path, ok := maps.Path(a.terrain(), move, p.To)
	switch {
	case !ok:
		return "", fmt.Errorf("there is no way to (%d,%d) for %s", p.To.X, p.To.Y, monsterLabel(m))
	case len(path) > m.Movement:
		return "", fmt.Errorf("(%d,%d) is %s away; %s moves %s", p.To.X, p.To.Y, squares(len(path)), monsterLabel(m), squares(m.Movement))
	}
	m.X, m.Y = p.To.X, p.To.Y
	r.MonstersMoved = append(r.MonstersMoved, m.ID)
	return fmt.Sprintf("%s moves %s to (%d,%d)%s", monsterLabel(m), squares(len(path)), m.X, m.Y, a.revealFromHeroes()), nil
}
