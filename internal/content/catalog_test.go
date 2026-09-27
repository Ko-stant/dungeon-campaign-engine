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
	if len(c.Furniture)+len(c.Monsters)+len(c.Heroes) != 0 {
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
	}
	for name, fsys := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(fsys); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
