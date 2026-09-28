package content

import (
	"testing"
	"testing/fstest"
)

func fixture() fstest.MapFS {
	return fstest.MapFS{
		"furniture/table.json": {Data: []byte(`{
			"id": "table", "name": "Table", "blocksLineOfSight": false, "blocksMovement": true,
			"gridSize": {"width": 3, "height": 2},
			"rendering": {"tileImage": "assets/tiles/table.jpg", "tileImageCleaned": "assets/tiles_cleaned/table.png"}
		}`)},
		"furniture/chest.json": {Data: []byte(`{
			"id": "chest", "name": "Chest", "gridSize": {"width": 1, "height": 1},
			"rendering": {"tileImage": "assets/tiles/chest.jpg"}
		}`)},
		"monsters/orc.json": {Data: []byte(`{
			"id": "orc", "name": "Orc",
			"stats": {"movementSquares": 8, "attackDice": 3, "defendDice": 2, "bodyPoints": 1, "mindPoints": 2},
			"rendering": {"tileImageCleaned": "assets/tiles_cleaned/monsters/monster_orc.png"}
		}`)},
		"heroes/barbarian.json": {Data: []byte(`{
			"id": "barbarian", "name": "Barbarian", "description": "A warrior.",
			"stats": {"bodyPoints": 8, "mindPoints": 2, "attackDice": 3, "defenseDice": 2, "movementDice": 2}
		}`)},
		"heroes/README.md": {Data: []byte("not a card")},
		"traps/pit.json": {Data: []byte(`{
			"id": "pit", "name": "Pit Trap", "gridSize": {"width": 1, "height": 1},
			"rendering": {"tileImage": "assets/tiles/traps/trap_pit.jpg", "tileImageCleaned": "assets/tiles_cleaned/traps/trap_pit.png"}
		}`)},
		"traps/boulder.json": {Data: []byte(`{
			"id": "boulder", "name": "Boulder", "gridSize": {"width": 1, "height": 1}, "movable": true
		}`)},
		"traps/long_pit.json": {Data: []byte(`{
			"id": "long_pit", "name": "Long Pit Trap", "gridSize": {"width": 1, "height": 2},
			"rendering": {"tileImageCleaned": "assets/tiles_cleaned/traps/trap_long_pit.png"}
		}`)},
	}
}

func TestLoadReadsAllCatalogs(t *testing.T) {
	c, err := Load(fixture())
	if err != nil {
		t.Fatal(err)
	}

	if len(c.Furniture) != 2 || c.Furniture[0].ID != "chest" || c.Furniture[1].ID != "table" {
		t.Fatalf("furniture should be sorted by id: %+v", c.Furniture)
	}
	table := c.Furniture[1]
	if table.Width != 3 || table.Height != 2 || !table.BlocksMovement || table.Image != "assets/tiles_cleaned/table.png" {
		t.Fatalf("table: %+v", table)
	}
	if c.Furniture[0].Image != "assets/tiles/chest.jpg" {
		t.Fatalf("chest should fall back to tileImage: %+v", c.Furniture[0])
	}

	orc, ok := c.Monster("orc")
	if !ok || orc.Body != 1 || orc.Mind != 2 || orc.Attack != 3 || orc.Defense != 2 || orc.Movement != 8 {
		t.Fatalf("orc: %+v ok=%v", orc, ok)
	}

	hero, ok := c.Hero("barbarian")
	if !ok || hero.Body != 8 || hero.Mind != 2 || hero.MovementDice != 2 || hero.Description != "A warrior." {
		t.Fatalf("barbarian: %+v ok=%v", hero, ok)
	}
	if len(c.Heroes) != 1 {
		t.Fatalf("non-JSON files must be ignored: %+v", c.Heroes)
	}
}

func TestLoadReadsMonsterGridSize(t *testing.T) {
	fsys := fixture()
	fsys["monsters/giant_wolf.json"] = &fstest.MapFile{Data: []byte(`{
		"id": "giant_wolf", "name": "Giant Wolf", "gridSize": {"width": 2, "height": 1},
		"rendering": {"tileImageCleaned": "assets/tiles_cleaned/monsters/monster_giant_wolf.png"}
	}`)}
	c, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}

	wolf, _ := c.Monster("giant_wolf")
	if w, h := wolf.Size(); w != 2 || h != 1 {
		t.Fatalf("giant_wolf size = %dx%d, want 2x1", w, h)
	}
	orc, _ := c.Monster("orc")
	if w, h := orc.Size(); w != 1 || h != 1 {
		t.Fatalf("a monster without gridSize should be 1x1, got %dx%d", w, h)
	}
}

func TestLoadReadsTraps(t *testing.T) {
	c, err := Load(fixture())
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Traps) != 3 || c.Traps[0].ID != "boulder" || c.Traps[1].ID != "long_pit" || c.Traps[2].ID != "pit" {
		t.Fatalf("traps should be sorted by id: %+v", c.Traps)
	}
	if !c.Traps[0].Movable || c.Traps[1].Movable {
		t.Fatalf("only the boulder is movable: %+v", c.Traps)
	}
	long := c.Traps[1]
	if long.Name != "Long Pit Trap" || long.Width != 1 || long.Height != 2 || long.Image != "assets/tiles_cleaned/traps/trap_long_pit.png" {
		t.Fatalf("long_pit: %+v", long)
	}
	if w, h, ok := c.TrapSize("long_pit"); !ok || w != 1 || h != 2 {
		t.Fatalf("TrapSize(long_pit) = %d, %d, %v", w, h, ok)
	}
	if _, _, ok := c.TrapSize("chest"); ok {
		t.Fatal("a kind without a catalog entry should not be found")
	}
	if def, ok := c.TrapByID("pit"); !ok || def.Name != "Pit Trap" {
		t.Fatalf("TrapByID(pit) = %+v, %v", def, ok)
	}
}

func TestFurnitureSizeMatchesMapsLookup(t *testing.T) {
	c, _ := Load(fixture())
	if w, h, ok := c.FurnitureSize("table"); !ok || w != 3 || h != 2 {
		t.Fatalf("FurnitureSize(table) = %d, %d, %v", w, h, ok)
	}
	if _, _, ok := c.FurnitureSize("harpsichord"); ok {
		t.Fatal("unknown furniture should not be found")
	}
}

func TestLoadToleratesMissingDirectories(t *testing.T) {
	c, err := Load(fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Furniture)+len(c.Monsters)+len(c.Heroes)+len(c.Traps) != 0 || c.Traps == nil {
		t.Fatalf("expected an empty catalog: %+v", c)
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	cases := map[string]fstest.MapFS{
		"malformed json": {"monsters/bad.json": {Data: []byte(`{"id":`)}},
		"missing id":     {"furniture/x.json": {Data: []byte(`{"name": "Nameless", "gridSize": {"width": 1, "height": 1}}`)}},
		"duplicate id": {
			"heroes/a.json": {Data: []byte(`{"id": "elf", "name": "Elf"}`)},
			"heroes/b.json": {Data: []byte(`{"id": "elf", "name": "Elf again"}`)},
		},
		"zero furniture size": {"furniture/x.json": {Data: []byte(`{"id": "x", "gridSize": {"width": 0, "height": 1}}`)}},
		"zero trap size":      {"traps/x.json": {Data: []byte(`{"id": "x", "gridSize": {"width": 1, "height": 0}}`)}},
	}
	for name, fsys := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(fsys); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
