package main

import (
	"testing"
)

// testLogger implements Logger for testing
type testLogger struct{}

func (tl *testLogger) Printf(format string, args ...interface{}) {}
func (tl *testLogger) Println(args ...interface{})              {}

func TestContentManager_LoadCampaign(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)

	err := cm.LoadCampaign("base")
	if err != nil {
		t.Fatalf("Failed to load base campaign: %v", err)
	}

	// Verify campaign metadata loaded
	campaign := cm.GetCampaign()
	if campaign == nil {
		t.Fatal("Campaign metadata is nil")
	}
	if campaign.ID != "base" {
		t.Errorf("Expected campaign ID 'base', got '%s'", campaign.ID)
	}
	if campaign.Name != "HeroQuest Base Game" {
		t.Errorf("Expected campaign name 'HeroQuest Base Game', got '%s'", campaign.Name)
	}

	// Verify equipment loaded
	equipment := cm.GetAllEquipment()
	if len(equipment) == 0 {
		t.Error("No equipment cards loaded")
	}
	t.Logf("Loaded %d equipment cards", len(equipment))

	// Verify specific equipment cards
	crossbow, ok := cm.GetEquipmentCard("crossbow")
	if !ok {
		t.Error("Crossbow card not found")
	} else {
		if crossbow.Name != "Crossbow" {
			t.Errorf("Expected crossbow name 'Crossbow', got '%s'", crossbow.Name)
		}
		if crossbow.Type != "weapon" {
			t.Errorf("Expected crossbow type 'weapon', got '%s'", crossbow.Type)
		}
		if crossbow.AttackDice != 3 {
			t.Errorf("Expected crossbow attack dice 3, got %d", crossbow.AttackDice)
		}
	}

	battleAxe, ok := cm.GetEquipmentCard("battle_axe")
	if !ok {
		t.Error("Battle Axe card not found")
	} else {
		if battleAxe.AttackDice != 4 {
			t.Errorf("Expected battle axe attack dice 4, got %d", battleAxe.AttackDice)
		}
	}

	// Verify artifacts loaded
	artifacts := cm.GetAllArtifacts()
	t.Logf("Loaded %d artifact cards", len(artifacts))

	// Verify treasures loaded
	treasures := cm.GetAllTreasures()
	if len(treasures) == 0 {
		t.Error("No treasure cards loaded")
	}
	t.Logf("Loaded %d treasure cards", len(treasures))

	// Verify specific treasure cards
	gold15, ok := cm.GetTreasureCard("gold_15")
	if !ok {
		t.Error("Gold 15 card not found")
	} else {
		if gold15.Type != "gold" {
			t.Errorf("Expected gold_15 type 'gold', got '%s'", gold15.Type)
		}
		if gold15.Value != 15 {
			t.Errorf("Expected gold_15 value 15, got %d", gold15.Value)
		}
	}

	wanderingMonster, ok := cm.GetTreasureCard("wandering_monster")
	if !ok {
		t.Error("Wandering Monster card not found")
	} else {
		if wanderingMonster.Type != "monster" {
			t.Errorf("Expected wandering_monster type 'monster', got '%s'", wanderingMonster.Type)
		}
		if !wanderingMonster.ReturnToDeck {
			t.Error("Expected wandering_monster to return to deck")
		}
	}

	hazardArrow, ok := cm.GetTreasureCard("hazard_arrow")
	if !ok {
		t.Error("Hazard Arrow card not found")
	} else {
		if hazardArrow.Type != "hazard" {
			t.Errorf("Expected hazard_arrow type 'hazard', got '%s'", hazardArrow.Type)
		}
		if hazardArrow.Damage != 1 {
			t.Errorf("Expected hazard_arrow damage 1, got %d", hazardArrow.Damage)
		}
	}
}

func TestContentManager_GetEquipmentCard(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)

	err := cm.LoadCampaign("base")
	if err != nil {
		t.Fatalf("Failed to load base campaign: %v", err)
	}

	// Test existing card
	card, ok := cm.GetEquipmentCard("longsword")
	if !ok {
		t.Error("Longsword card not found")
	}
	if card != nil && card.ID != "longsword" {
		t.Errorf("Expected card ID 'longsword', got '%s'", card.ID)
	}

	// Test non-existing card
	_, ok = cm.GetEquipmentCard("nonexistent")
	if ok {
		t.Error("Expected false for non-existent card")
	}
}

func TestContentManager_ThreadSafety(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)

	err := cm.LoadCampaign("base")
	if err != nil {
		t.Fatalf("Failed to load base campaign: %v", err)
	}

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			card, _ := cm.GetEquipmentCard("crossbow")
			if card != nil && card.Name != "Crossbow" {
				t.Errorf("Concurrent read failed")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestContentManager_LoadAllHeroes(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)

	err := cm.LoadCampaign("base")
	if err != nil {
		t.Fatalf("Failed to load base campaign: %v", err)
	}

	// Verify heroes loaded
	heroes := cm.GetAllHeroes()
	if len(heroes) == 0 {
		t.Fatal("No hero cards loaded")
	}
	t.Logf("Loaded %d hero cards", len(heroes))

	// Expected heroes based on the files in content/heroes/
	expectedHeroes := []string{
		"barbarian",
		"dwarf",
		"elf",
		"wizard",
	}

	// Verify each expected hero loaded correctly
	for _, heroID := range expectedHeroes {
		t.Run(heroID, func(t *testing.T) {
			hero, ok := cm.GetHeroCard(heroID)
			if !ok {
				t.Fatalf("Hero %s not found", heroID)
			}

			// Validate hero has required fields
			if hero.ID == "" {
				t.Error("Hero ID is empty")
			}
			if hero.Name == "" {
				t.Error("Hero Name is empty")
			}
			if hero.Class == "" {
				t.Error("Hero Class is empty")
			}

			// Validate stats
			if hero.Stats.BodyPoints <= 0 {
				t.Errorf("Invalid BodyPoints: %d", hero.Stats.BodyPoints)
			}
			if hero.Stats.MindPoints <= 0 {
				t.Errorf("Invalid MindPoints: %d", hero.Stats.MindPoints)
			}
			if hero.Stats.AttackDice <= 0 {
				t.Errorf("Invalid AttackDice: %d", hero.Stats.AttackDice)
			}
			if hero.Stats.DefenseDice <= 0 {
				t.Errorf("Invalid DefenseDice: %d", hero.Stats.DefenseDice)
			}
			if hero.Stats.MovementDice <= 0 {
				t.Errorf("Invalid MovementDice: %d", hero.Stats.MovementDice)
			}

			t.Logf("✓ %s (%s): Body=%d Mind=%d Attack=%dd Defense=%dd Movement=%dd",
				hero.Name, hero.Class,
				hero.Stats.BodyPoints, hero.Stats.MindPoints,
				hero.Stats.AttackDice, hero.Stats.DefenseDice, hero.Stats.MovementDice)

			// Validate starting equipment references exist
			for _, weaponID := range hero.StartingEquipment.Weapons {
				if _, ok := cm.GetEquipmentCard(weaponID); !ok {
					t.Errorf("Starting weapon '%s' not found in equipment deck", weaponID)
				} else {
					t.Logf("  Starting weapon: %s", weaponID)
				}
			}
			for _, armorID := range hero.StartingEquipment.Armor {
				if _, ok := cm.GetEquipmentCard(armorID); !ok {
					t.Errorf("Starting armor '%s' not found in equipment deck", armorID)
				} else {
					t.Logf("  Starting armor: %s", armorID)
				}
			}
			for _, itemID := range hero.StartingEquipment.Items {
				if _, ok := cm.GetEquipmentCard(itemID); !ok {
					t.Errorf("Starting item '%s' not found in equipment deck", itemID)
				} else {
					t.Logf("  Starting item: %s", itemID)
				}
			}
		})
	}
}

func TestContentManager_SpecificHeroStats(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)

	err := cm.LoadCampaign("base")
	if err != nil {
		t.Fatalf("Failed to load base campaign: %v", err)
	}

	// Test Barbarian specific stats
	barbarian, ok := cm.GetHeroCard("barbarian")
	if !ok {
		t.Fatal("Barbarian not found")
	}
	if barbarian.Name != "Barbarian" {
		t.Errorf("Expected Barbarian name, got '%s'", barbarian.Name)
	}
	if barbarian.Stats.BodyPoints != 8 {
		t.Errorf("Expected Barbarian Body 8, got %d", barbarian.Stats.BodyPoints)
	}
	if barbarian.Stats.MindPoints != 2 {
		t.Errorf("Expected Barbarian Mind 2, got %d", barbarian.Stats.MindPoints)
	}
	if barbarian.Stats.AttackDice != 3 {
		t.Errorf("Expected Barbarian Attack 3, got %d", barbarian.Stats.AttackDice)
	}
	// Verify starting equipment
	if len(barbarian.StartingEquipment.Weapons) == 0 {
		t.Error("Barbarian should have starting weapons")
	}
	hasBroadsword := false
	for _, weapon := range barbarian.StartingEquipment.Weapons {
		if weapon == "broadsword" {
			hasBroadsword = true
			break
		}
	}
	if !hasBroadsword {
		t.Error("Barbarian should start with broadsword")
	}
}

func TestNewPlayerFromContent(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)

	err := cm.LoadCampaign("base")
	if err != nil {
		t.Fatalf("Failed to load base campaign: %v", err)
	}

	// Create inventory manager for testing
	inventoryMgr := NewInventoryManager(cm, logger)
	if err := inventoryMgr.InitializeHeroInventory("test-hero-1"); err != nil {
		t.Fatalf("Failed to initialize inventory: %v", err)
	}

	// Test creating each hero
	heroes := []string{"barbarian", "dwarf", "elf", "wizard"}
	for _, heroID := range heroes {
		t.Run(heroID, func(t *testing.T) {
			heroCard, ok := cm.GetHeroCard(heroID)
			if !ok {
				t.Fatalf("Hero %s not found", heroID)
			}

			entityID := "test-hero-" + heroID
			if err := inventoryMgr.InitializeHeroInventory(entityID); err != nil {
				t.Fatalf("Failed to initialize inventory for %s: %v", heroID, err)
			}

			player, err := NewPlayerFromContent("player-"+heroID, entityID, heroCard, cm, inventoryMgr)
			if err != nil {
				t.Fatalf("Failed to create player from %s: %v", heroID, err)
			}

			// Verify player created correctly
			if player == nil {
				t.Fatal("Player is nil")
			}
			if player.Name != heroCard.Name {
				t.Errorf("Expected player name '%s', got '%s'", heroCard.Name, player.Name)
			}
			if player.Character.BaseStats.BodyPoints != heroCard.Stats.BodyPoints {
				t.Errorf("Expected Body %d, got %d", heroCard.Stats.BodyPoints, player.Character.BaseStats.BodyPoints)
			}
			if player.Character.CurrentBody != heroCard.Stats.BodyPoints {
				t.Errorf("Expected CurrentBody %d, got %d", heroCard.Stats.BodyPoints, player.Character.CurrentBody)
			}

			// Verify starting equipment was added to inventory
			inventory, err := inventoryMgr.GetInventory(entityID)
			if err != nil {
				t.Errorf("Failed to get inventory: %v", err)
			} else if inventory == nil {
				t.Error("Inventory not found for hero")
			} else {
				totalItems := len(inventory.Equipment) + len(inventory.Carried)
				t.Logf("Hero %s inventory has %d items (%d equipped, %d carried)",
					heroID, totalItems, len(inventory.Equipment), len(inventory.Carried))

				// Check if starting weapons are in inventory (either equipped or carried)
				for _, weaponID := range heroCard.StartingEquipment.Weapons {
					found := false
					equipped := false

					// Check equipped items
					for _, item := range inventory.Equipment {
						if item != nil && item.ID == weaponID {
							found = true
							equipped = true
							break
						}
					}

					// Check carried items if not found in equipped
					if !found {
						for _, item := range inventory.Carried {
							if item != nil && item.ID == weaponID {
								found = true
								break
							}
						}
					}

					if !found {
						t.Errorf("Starting weapon '%s' not found in inventory", weaponID)
					} else if equipped {
						t.Logf("  ✓ Starting weapon '%s' is equipped", weaponID)
					} else {
						t.Logf("  ⚠ Starting weapon '%s' is in inventory but not equipped", weaponID)
					}
				}
			}

			t.Logf("✓ Successfully created player from %s hero card with %d starting items",
				heroID, len(heroCard.StartingEquipment.Weapons)+len(heroCard.StartingEquipment.Armor)+len(heroCard.StartingEquipment.Items))
		})
	}
}
