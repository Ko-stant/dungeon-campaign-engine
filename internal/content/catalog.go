// Package content loads the reference catalogs (heroes, monsters, furniture, traps)
// from the content directory. The files are gitignored, so everything here
// takes an fs.FS: os.DirFS("content") in the server, fstest.MapFS in tests.
package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// FurnitureDef is a furniture catalog entry.
type FurnitureDef struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Width             int    `json:"width"`
	Height            int    `json:"height"`
	BlocksMovement    bool   `json:"blocksMovement"`
	BlocksLineOfSight bool   `json:"blocksLineOfSight"`
	Image             string `json:"image,omitempty"`
}

// TrapDef is a trap catalog entry: a trap kind with its artwork and footprint.
// Quest traps whose kind has no entry are drawn as single-square markers.
// Movable traps (a rolling boulder) can be moved on the board during play.
type TrapDef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Image   string `json:"image,omitempty"`
	Movable bool   `json:"movable,omitempty"`
}

// MonsterDef is a monster catalog entry with its base stats.
type MonsterDef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Body     int    `json:"body"`
	Mind     int    `json:"mind"`
	Attack   int    `json:"attack"`
	Defense  int    `json:"defense"`
	Movement int    `json:"movement"`
	Image    string `json:"image,omitempty"`
	// Monsters may cover several squares (gridSize in the catalog file, or the
	// size chosen for a custom monster). Width and Height of 0 mean 1. Custom
	// monsters (made by the GM) have a color instead of artwork.
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Color  string `json:"color,omitempty"`
	Notes  string `json:"notes,omitempty"`
	Custom bool   `json:"custom,omitempty"`
	// Combat holds a campaign's combat stats for this monster (see
	// Catalog.WithMonsterStats); nil when the campaign has none for it.
	Combat *MonsterCombat `json:"combat,omitempty"`
}

// MonsterCombat is a monster's combat stats under The Three Plagues rules (see
// docs/campaigns/three-plagues/RULES_AND_CLASSES.md): heroes hit when they meet
// or beat Avoidance; the monster rolls HitDice against a hero's avoidance and
// deals Damage. The traits change who it can reach.
type MonsterCombat struct {
	Avoidance int    `json:"avoidance"`
	HitDice   string `json:"hitDice"`
	Damage    int    `json:"damage"`
	// Ranged monsters attack from range; reaching ones strike past heroes holding a doorway.
	Ranged bool `json:"ranged,omitempty"`
	Reach  bool `json:"reach,omitempty"`
	// Line: each attack also strikes this many more heroes in a straight line.
	Line int `json:"line,omitempty"`
	// Splash: each attack also blasts SplashTargets heroes beside the target for SplashDamage.
	SplashDamage  int  `json:"splashDamage,omitempty"`
	SplashTargets int  `json:"splashTargets,omitempty"`
	Undead        bool `json:"undead,omitempty"`
	// Abilities is what the monster can do, in the GM's words; the player
	// screen shows it on the monster's card.
	Abilities string `json:"abilities,omitempty"`
}

// MonsterStats is a campaign's stat line for one monster type: its Body,
// movement and combat stats. Movement 0 keeps the catalog's movement.
type MonsterStats struct {
	Body     int `json:"body"`
	Movement int `json:"movement,omitempty"`
	MonsterCombat
}

// BodyOnly reports a stat line with no hit dice: the monster gets its Body
// and no combat stats (a non-fighting character such as a prisoner).
func (st MonsterStats) BodyOnly() bool {
	return st.HitDice == ""
}

// CombatEmpty reports whether nothing but Body is set (what a body-only line
// must be).
func (st MonsterStats) CombatEmpty() bool {
	return st.MonsterCombat == MonsterCombat{}
}

// WithMonsterStats returns a copy of the catalog whose monsters use a
// campaign's stat lines (Body, movement and combat stats), keyed by monster type. Types
// the catalog lacks are ignored; with no stats the catalog itself is returned.
func (c *Catalog) WithMonsterStats(stats map[string]MonsterStats) *Catalog {
	if len(stats) == 0 {
		return c
	}
	out := *c
	out.Monsters = make([]MonsterDef, len(c.Monsters))
	for i, m := range c.Monsters {
		if st, ok := stats[m.ID]; ok {
			m.Body, m.Combat = st.Body, nil
			if st.Movement > 0 {
				m.Movement = st.Movement
			}
			if !st.BodyOnly() {
				combat := st.MonsterCombat
				m.Combat = &combat
			}
		}
		out.Monsters[i] = m
	}
	return &out
}

// Size returns the monster's footprint in squares (at least 1x1).
func (m MonsterDef) Size() (int, int) {
	return max(m.Width, 1), max(m.Height, 1)
}

// HeroDef is a hero class with its starting stats.
type HeroDef struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Body         int    `json:"body"`
	Mind         int    `json:"mind"`
	Attack       int    `json:"attack"`
	Defense      int    `json:"defense"`
	MovementDice int    `json:"movementDice"`

	// Custom classes (made on the Classes page) roll dice expressions
	// ("2d6+1") instead of combat dice, and may have accuracy, mana,
	// class-only equipment and abilities.
	Custom      bool      `json:"custom,omitempty"`
	Color       string    `json:"color,omitempty"`
	AttackDice  string    `json:"attackDice,omitempty"`
	DefenseDice string    `json:"defenseDice,omitempty"`
	Movement    string    `json:"movement,omitempty"`
	Accuracy    int       `json:"accuracy,omitempty"`
	Mana        int       `json:"mana,omitempty"`
	Exclusives  []string  `json:"exclusives,omitempty"`
	Abilities   []Ability `json:"abilities,omitempty"`

	// The Three Plagues combat (see docs/campaigns/three-plagues/RULES_AND_CLASSES.md):
	// AttackDice is the hit dice and DefenseDice the defense dice; a crit on the d20
	// crit die from CritFrom up; the class's own damage, avoidance and mitigation
	// (gear adds to them); mana regenerated each fight round. Zero means none (CritFrom
	// zero: not set, on classes saved before these existed).
	CritFrom   int `json:"critFrom,omitempty"`
	Damage     int `json:"damage,omitempty"`
	Avoidance  int `json:"avoidance,omitempty"`
	Mitigation int `json:"mitigation,omitempty"`
	ManaRegen  int `json:"manaRegen,omitempty"`
	// Reach is what the class's basic attack reaches (Reach*); empty means
	// adjacent.
	Reach string `json:"reach,omitempty"`

	// Inactive marks a deactivated custom class: left out of the new-hero
	// picker, still found for heroes who already have it.
	Inactive bool `json:"inactive,omitempty"`
}

// What a basic attack reaches (RULES_AND_CLASSES.md, "Who a basic attack
// reaches"): orthogonally adjacent squares, those plus the diagonals, or
// anything in line of sight.
const (
	ReachAdjacent = "adjacent"
	ReachDiagonal = "diagonal"
	ReachSight    = "sight"
)

// Reaches lists the reaches in display order.
var Reaches = []string{ReachAdjacent, ReachDiagonal, ReachSight}

// Ability kinds.
const (
	AbilityActive   = "active"
	AbilityPassive  = "passive"
	AbilityReaction = "reaction"
	AbilitySpell    = "spell"
)

// AbilityKinds lists the ability kinds in display order.
var AbilityKinds = []string{AbilityActive, AbilityPassive, AbilityReaction, AbilitySpell}

// Class exclusives: things only a class with the tag may do. Advice only.
const (
	ExclusiveTwoHanded = "two-handed"
	ExclusiveRanged    = "ranged"
	ExclusiveSpells    = "spells"
	ExclusiveDisarm    = "disarm"
)

// Exclusives lists the class exclusives in display order.
var Exclusives = []string{ExclusiveTwoHanded, ExclusiveRanged, ExclusiveSpells, ExclusiveDisarm}

// Ability is a hero class ability. Cooldown counts rounds: used in round R,
// it is ready again in round R+Cooldown. Mana and cooldown may both apply.
type Ability struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	ManaCost int    `json:"manaCost,omitempty"`
	Cooldown int    `json:"cooldown,omitempty"`
	Text     string `json:"text,omitempty"`
}

// Catalog holds every entry, each list sorted by id.
type Catalog struct {
	Furniture []FurnitureDef `json:"furniture"`
	Monsters  []MonsterDef   `json:"monsters"`
	Heroes    []HeroDef      `json:"heroes"`
	Traps     []TrapDef      `json:"traps"`
}

type rendering struct {
	TileImage        string `json:"tileImage"`
	TileImageCleaned string `json:"tileImageCleaned"`
}

func (r rendering) image() string {
	if r.TileImageCleaned != "" {
		return r.TileImageCleaned
	}
	return r.TileImage
}

type furnitureFile struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	BlocksLineOfSight bool   `json:"blocksLineOfSight"`
	BlocksMovement    bool   `json:"blocksMovement"`
	GridSize          struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"gridSize"`
	Rendering rendering `json:"rendering"`
}

type trapFile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Movable  bool   `json:"movable"`
	GridSize struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"gridSize"`
	Rendering rendering `json:"rendering"`
}

type monsterFile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stats struct {
		MovementSquares int `json:"movementSquares"`
		AttackDice      int `json:"attackDice"`
		DefendDice      int `json:"defendDice"`
		BodyPoints      int `json:"bodyPoints"`
		MindPoints      int `json:"mindPoints"`
	} `json:"stats"`
	GridSize struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"gridSize"`
	Rendering rendering `json:"rendering"`
}

type heroFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Stats       struct {
		BodyPoints   int `json:"bodyPoints"`
		MindPoints   int `json:"mindPoints"`
		AttackDice   int `json:"attackDice"`
		DefenseDice  int `json:"defenseDice"`
		MovementDice int `json:"movementDice"`
	} `json:"stats"`
}

// Load reads furniture/, monsters/, heroes/ and traps/ from fsys. A missing directory
// yields an empty list. make import-content uses it to import the hero classes.
func Load(fsys fs.FS) (*Catalog, error) {
	return load(fsys, true)
}

// LoadPieces reads furniture/, monsters/ and traps/ but not heroes/: the
// server's hero classes come from the database (custom_hero_class, the base
// game's imported from content/heroes by make import-content).
func LoadPieces(fsys fs.FS) (*Catalog, error) {
	return load(fsys, false)
}

func load(fsys fs.FS, heroes bool) (*Catalog, error) {
	c := &Catalog{Heroes: []HeroDef{}}
	var err error

	c.Furniture, err = loadDir(fsys, "furniture", func(f furnitureFile) (FurnitureDef, error) {
		if f.GridSize.Width < 1 || f.GridSize.Height < 1 {
			return FurnitureDef{}, fmt.Errorf("gridSize must be at least 1x1, got %dx%d", f.GridSize.Width, f.GridSize.Height)
		}
		return FurnitureDef{
			ID: f.ID, Name: f.Name, Width: f.GridSize.Width, Height: f.GridSize.Height,
			BlocksMovement: f.BlocksMovement, BlocksLineOfSight: f.BlocksLineOfSight, Image: f.Rendering.image(),
		}, nil
	})
	if err != nil {
		return nil, err
	}

	c.Monsters, err = loadDir(fsys, "monsters", func(f monsterFile) (MonsterDef, error) {
		return MonsterDef{
			ID: f.ID, Name: f.Name, Body: f.Stats.BodyPoints, Mind: f.Stats.MindPoints,
			Attack: f.Stats.AttackDice, Defense: f.Stats.DefendDice, Movement: f.Stats.MovementSquares,
			Image: f.Rendering.image(), Width: f.GridSize.Width, Height: f.GridSize.Height,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	if heroes {
		c.Heroes, err = loadDir(fsys, "heroes", func(f heroFile) (HeroDef, error) {
			return HeroDef{
				ID: f.ID, Name: f.Name, Description: f.Description, Body: f.Stats.BodyPoints, Mind: f.Stats.MindPoints,
				Attack: f.Stats.AttackDice, Defense: f.Stats.DefenseDice, MovementDice: f.Stats.MovementDice,
			}, nil
		})
		if err != nil {
			return nil, err
		}
	}

	c.Traps, err = loadDir(fsys, "traps", func(f trapFile) (TrapDef, error) {
		if f.GridSize.Width < 1 || f.GridSize.Height < 1 {
			return TrapDef{}, fmt.Errorf("gridSize must be at least 1x1, got %dx%d", f.GridSize.Width, f.GridSize.Height)
		}
		return TrapDef{
			ID: f.ID, Name: f.Name, Width: f.GridSize.Width, Height: f.GridSize.Height,
			Image: f.Rendering.image(), Movable: f.Movable,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

type identified interface {
	FurnitureDef | MonsterDef | HeroDef | TrapDef
}

func idOf[T identified](v T) string {
	switch d := any(v).(type) {
	case FurnitureDef:
		return d.ID
	case MonsterDef:
		return d.ID
	case HeroDef:
		return d.ID
	case TrapDef:
		return d.ID
	}
	return ""
}

func loadDir[F any, T identified](fsys fs.FS, dir string, convert func(F) (T, error)) ([]T, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []T{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("content: read %s: %w", dir, err)
	}

	out := []T{}
	seen := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		file := path.Join(dir, e.Name())
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, fmt.Errorf("content: %s: %w", file, err)
		}
		var raw F
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("content: %s: %w", file, err)
		}
		def, err := convert(raw)
		if err != nil {
			return nil, fmt.Errorf("content: %s: %w", file, err)
		}
		id := idOf(def)
		if id == "" {
			return nil, fmt.Errorf("content: %s: missing id", file)
		}
		if other, dup := seen[id]; dup {
			return nil, fmt.Errorf("content: %s: id %q already defined in %s", file, id, other)
		}
		seen[id] = file
		out = append(out, def)
	}
	slices.SortFunc(out, func(a, b T) int { return strings.Compare(idOf(a), idOf(b)) })
	return out, nil
}

// FurnitureSize returns a furniture type's unrotated size (a maps.SizeLookup).
func (c *Catalog) FurnitureSize(furnitureType string) (int, int, bool) {
	for _, f := range c.Furniture {
		if f.ID == furnitureType {
			return f.Width, f.Height, true
		}
	}
	return 0, 0, false
}

// TrapSize returns a trap kind's unrotated size (a maps.SizeLookup). Kinds
// without a catalog entry are not found.
func (c *Catalog) TrapSize(kind string) (int, int, bool) {
	if t, ok := c.TrapByID(kind); ok {
		return t.Width, t.Height, true
	}
	return 0, 0, false
}

// TrapByID looks up a trap entry.
func (c *Catalog) TrapByID(id string) (TrapDef, bool) {
	for _, t := range c.Traps {
		if t.ID == id {
			return t, true
		}
	}
	return TrapDef{}, false
}

// FurnitureByID looks up a furniture entry.
func (c *Catalog) FurnitureByID(id string) (FurnitureDef, bool) {
	for _, f := range c.Furniture {
		if f.ID == id {
			return f, true
		}
	}
	return FurnitureDef{}, false
}

// Monster looks up a monster entry.
func (c *Catalog) Monster(id string) (MonsterDef, bool) {
	for _, m := range c.Monsters {
		if m.ID == id {
			return m, true
		}
	}
	return MonsterDef{}, false
}

// Hero looks up a hero class.
func (c *Catalog) Hero(id string) (HeroDef, bool) {
	for _, h := range c.Heroes {
		if h.ID == id {
			return h, true
		}
	}
	return HeroDef{}, false
}
