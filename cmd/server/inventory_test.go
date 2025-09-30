package main

import (
	"testing"
)

func TestInventoryManager_InitializeHeroInventory(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	im := NewInventoryManager(cm, logger)

	err := im.InitializeHeroInventory("hero-1")
	if err != nil {
		t.Fatalf("Failed to initialize inventory: %v", err)
	}

	inventory, err := im.GetInventory("hero-1")
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}

	if inventory.HeroID != "hero-1" {
		t.Errorf("Expected hero ID 'hero-1', got '%s'", inventory.HeroID)
	}
	if inventory.Gold != 0 {
		t.Errorf("Expected starting gold 0, got %d", inventory.Gold)
	}
	if len(inventory.Equipment) != 0 {
		t.Errorf("Expected empty equipment, got %d items", len(inventory.Equipment))
	}
	if len(inventory.Carried) != 0 {
		t.Errorf("Expected empty carried items, got %d items", len(inventory.Carried))
	}
}

func TestInventoryManager_AddGold(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	im := NewInventoryManager(cm, logger)
	if err := im.InitializeHeroInventory("hero-1"); err != nil {
		t.Fatalf("Failed to initialize inventory: %v", err)
	}

	// Add gold
	if err := im.AddGold("hero-1", 100); err != nil {
		t.Fatalf("Failed to add gold: %v", err)
	}

	inventory, _ := im.GetInventory("hero-1")
	if inventory.Gold != 100 {
		t.Errorf("Expected gold 100, got %d", inventory.Gold)
	}

	// Add more gold
	if err := im.AddGold("hero-1", 50); err != nil {
		t.Fatalf("Failed to add more gold: %v", err)
	}

	inventory, _ = im.GetInventory("hero-1")
	if inventory.Gold != 150 {
		t.Errorf("Expected gold 150, got %d", inventory.Gold)
	}
}

func TestInventoryManager_AddItem(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	im := NewInventoryManager(cm, logger)
	if err := im.InitializeHeroInventory("hero-1"); err != nil {
		t.Fatalf("Failed to initialize inventory: %v", err)
	}

	// Add crossbow
	if err := im.AddItem("hero-1", "crossbow"); err != nil {
		t.Fatalf("Failed to add crossbow: %v", err)
	}

	inventory, _ := im.GetInventory("hero-1")
	if len(inventory.Carried) != 1 {
		t.Fatalf("Expected 1 carried item, got %d", len(inventory.Carried))
	}

	if inventory.Carried[0].ID != "crossbow" {
		t.Errorf("Expected crossbow, got %s", inventory.Carried[0].ID)
	}
}

func TestInventoryManager_EquipItem(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	im := NewInventoryManager(cm, logger)
	if err := im.InitializeHeroInventory("hero-1"); err != nil {
		t.Fatalf("Failed to initialize inventory: %v", err)
	}

	// Add and equip longsword
	if err := im.AddItem("hero-1", "longsword"); err != nil {
		t.Fatalf("Failed to add longsword: %v", err)
	}

	if err := im.EquipItem("hero-1", "longsword"); err != nil {
		t.Fatalf("Failed to equip longsword: %v", err)
	}

	inventory, _ := im.GetInventory("hero-1")

	// Should be in equipment
	weapon, ok := inventory.Equipment["weapon"]
	if !ok {
		t.Fatal("Weapon slot not occupied")
	}
	if weapon.ID != "longsword" {
		t.Errorf("Expected longsword in weapon slot, got %s", weapon.ID)
	}

	// Should not be in carried
	if len(inventory.Carried) != 0 {
		t.Errorf("Expected no carried items, got %d", len(inventory.Carried))
	}
}

func TestInventoryManager_EquipItem_ReplaceExisting(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	im := NewInventoryManager(cm, logger)
	if err := im.InitializeHeroInventory("hero-1"); err != nil {
		t.Fatalf("Failed to initialize inventory: %v", err)
	}

	// Add and equip longsword
	if err := im.AddItem("hero-1", "longsword"); err != nil {
		t.Fatalf("Failed to add longsword: %v", err)
	}
	if err := im.EquipItem("hero-1", "longsword"); err != nil {
		t.Fatalf("Failed to equip longsword: %v", err)
	}

	// Add and equip battle axe (should replace longsword)
	if err := im.AddItem("hero-1", "battle_axe"); err != nil {
		t.Fatalf("Failed to add battle axe: %v", err)
	}
	if err := im.EquipItem("hero-1", "battle_axe"); err != nil {
		t.Fatalf("Failed to equip battle axe: %v", err)
	}

	inventory, _ := im.GetInventory("hero-1")

	// Battle axe should be equipped
	weapon, ok := inventory.Equipment["weapon"]
	if !ok {
		t.Fatal("Weapon slot not occupied")
	}
	if weapon.ID != "battle_axe" {
		t.Errorf("Expected battle_axe in weapon slot, got %s", weapon.ID)
	}

	// Longsword should be in carried
	if len(inventory.Carried) != 1 {
		t.Fatalf("Expected 1 carried item, got %d", len(inventory.Carried))
	}
	if inventory.Carried[0].ID != "longsword" {
		t.Errorf("Expected longsword in carried, got %s", inventory.Carried[0].ID)
	}
}

func TestInventoryManager_UnequipItem(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	im := NewInventoryManager(cm, logger)
	if err := im.InitializeHeroInventory("hero-1"); err != nil {
		t.Fatalf("Failed to initialize inventory: %v", err)
	}

	// Add and equip longsword
	if err := im.AddItem("hero-1", "longsword"); err != nil {
		t.Fatalf("Failed to add longsword: %v", err)
	}
	if err := im.EquipItem("hero-1", "longsword"); err != nil {
		t.Fatalf("Failed to equip longsword: %v", err)
	}

	// Unequip longsword
	if err := im.UnequipItem("hero-1", "weapon"); err != nil {
		t.Fatalf("Failed to unequip weapon: %v", err)
	}

	inventory, _ := im.GetInventory("hero-1")

	// Weapon slot should be empty
	if _, ok := inventory.Equipment["weapon"]; ok {
		t.Error("Expected weapon slot to be empty")
	}

	// Longsword should be in carried
	if len(inventory.Carried) != 1 {
		t.Fatalf("Expected 1 carried item, got %d", len(inventory.Carried))
	}
	if inventory.Carried[0].ID != "longsword" {
		t.Errorf("Expected longsword in carried, got %s", inventory.Carried[0].ID)
	}
}

func TestInventoryManager_AddSpell(t *testing.T) {
	logger := &testLogger{}
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	im := NewInventoryManager(cm, logger)
	if err := im.InitializeHeroInventory("hero-1"); err != nil {
		t.Fatalf("Failed to initialize inventory: %v", err)
	}

	// Add heal_body spell
	if err := im.AddSpell("hero-1", "heal_body"); err != nil {
		t.Fatalf("Failed to add spell: %v", err)
	}

	inventory, _ := im.GetInventory("hero-1")
	if len(inventory.Spells) != 1 {
		t.Fatalf("Expected 1 spell, got %d", len(inventory.Spells))
	}
	if inventory.Spells[0].ID != "heal_body" {
		t.Errorf("Expected heal_body spell, got %s", inventory.Spells[0].ID)
	}
}
