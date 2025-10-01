package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

func TestTreasureSystem_QuestNoteResolution(t *testing.T) {
	logger := &testLogger{}

	// Load content
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	// Load quest
	quest, err := geometry.LoadQuestFromFile("../../content/base/quests/quest-01.json")
	if err != nil {
		t.Fatalf("Failed to load quest: %v", err)
	}

	// Create treasure deck
	treasureDeck := NewTreasureDeckManager(cm, logger)
	if err := treasureDeck.InitializeDeck(); err != nil {
		t.Fatalf("Failed to initialize treasure deck: %v", err)
	}

	// Create treasure resolver
	resolver := NewTreasureResolver(cm, treasureDeck, quest, logger)

	// Test Note A - Weapons rack (crossbow, battle_axe)
	result, err := resolver.ResolveTreasureSearch("hero-1", 18, protocol.TileAddress{X: 11, Y: 15}, "furniture-4")
	if err != nil {
		t.Fatalf("Failed to resolve treasure: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful treasure search")
	}

	if result.NoteID != "A" {
		t.Errorf("Expected note ID 'A', got '%s'", result.NoteID)
	}

	if len(result.FoundItems) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(result.FoundItems))
	}

	// Check items
	foundCrossbow := false
	foundBattleAxe := false
	for _, item := range result.FoundItems {
		if item.ID == "crossbow" {
			foundCrossbow = true
		}
		if item.ID == "battle_axe" {
			foundBattleAxe = true
		}
	}

	if !foundCrossbow {
		t.Error("Expected to find crossbow")
	}
	if !foundBattleAxe {
		t.Error("Expected to find battle_axe")
	}

	// Test Note B - Chest with 84 gold
	result, err = resolver.ResolveTreasureSearch("hero-1", 19, protocol.TileAddress{X: 17, Y: 16}, "furniture-6")
	if err != nil {
		t.Fatalf("Failed to resolve treasure: %v", err)
	}

	if result.NoteID != "B" {
		t.Errorf("Expected note ID 'B', got '%s'", result.NoteID)
	}

	if result.FoundGold != 84 {
		t.Errorf("Expected 84 gold, got %d", result.FoundGold)
	}

	// Test Note C - Empty chest
	result, err = resolver.ResolveTreasureSearch("hero-1", 3, protocol.TileAddress{X: 10, Y: 5}, "furniture-12")
	if err != nil {
		t.Fatalf("Failed to resolve treasure: %v", err)
	}

	if result.NoteID != "C" {
		t.Errorf("Expected note ID 'C', got '%s'", result.NoteID)
	}

	if !result.IsEmpty {
		t.Error("Expected empty chest")
	}

	// Test consumed note - searching again should return empty message
	result, err = resolver.ResolveTreasureSearch("hero-2", 18, protocol.TileAddress{X: 11, Y: 15}, "furniture-4")
	if err != nil {
		t.Fatalf("Failed to resolve treasure: %v", err)
	}

	if !result.IsEmpty {
		t.Error("Expected empty result for consumed quest note")
	}
}

func TestTreasureSystem_DeckDrawing(t *testing.T) {
	logger := &testLogger{}

	// Load content
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	// Create treasure deck
	treasureDeck := NewTreasureDeckManager(cm, logger)
	if err := treasureDeck.InitializeDeck(); err != nil {
		t.Fatalf("Failed to initialize treasure deck: %v", err)
	}

	initialSize := treasureDeck.GetDeckSize()
	if initialSize == 0 {
		t.Fatal("Expected non-empty treasure deck")
	}

	t.Logf("Initial deck size: %d", initialSize)

	// Draw a card
	card, err := treasureDeck.DrawCard()
	if err != nil {
		t.Fatalf("Failed to draw card: %v", err)
	}

	if card == nil {
		t.Fatal("Expected card to be non-nil")
	}

	t.Logf("Drew card: %s (type: %s)", card.Name, card.Type)

	// Check deck size changed (or stayed same if card returns to deck)
	newSize := treasureDeck.GetDeckSize()
	if card.ReturnToDeck {
		if newSize != initialSize {
			t.Errorf("Expected deck size to remain %d for return-to-deck card, got %d", initialSize, newSize)
		}
	} else {
		if newSize != initialSize-1 {
			t.Errorf("Expected deck size to be %d, got %d", initialSize-1, newSize)
		}
		if treasureDeck.GetDiscardSize() != 1 {
			t.Errorf("Expected discard pile size 1, got %d", treasureDeck.GetDiscardSize())
		}
	}
}

func TestInventorySystem_TreasureIntegration(t *testing.T) {
	logger := &testLogger{}

	// Load content
	cm := NewContentManager(logger)
	if err := cm.LoadCampaign("base"); err != nil {
		t.Fatalf("Failed to load campaign: %v", err)
	}

	// Create inventory manager
	im := NewInventoryManager(cm, logger)
	if err := im.InitializeHeroInventory("hero-1"); err != nil {
		t.Fatalf("Failed to initialize inventory: %v", err)
	}

	// Simulate finding treasure
	if err := im.AddGold("hero-1", 84); err != nil {
		t.Fatalf("Failed to add gold: %v", err)
	}

	if err := im.AddItem("hero-1", "crossbow"); err != nil {
		t.Fatalf("Failed to add crossbow: %v", err)
	}

	if err := im.AddItem("hero-1", "battle_axe"); err != nil {
		t.Fatalf("Failed to add battle_axe: %v", err)
	}

	// Verify inventory
	inventory, err := im.GetInventory("hero-1")
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}

	if inventory.Gold != 84 {
		t.Errorf("Expected 84 gold, got %d", inventory.Gold)
	}

	if len(inventory.Carried) != 2 {
		t.Errorf("Expected 2 carried items, got %d", len(inventory.Carried))
	}

	// Equip crossbow
	if err := im.EquipItem("hero-1", "crossbow"); err != nil {
		t.Fatalf("Failed to equip crossbow: %v", err)
	}

	inventory, _ = im.GetInventory("hero-1")
	if _, ok := inventory.Equipment["weapon"]; !ok {
		t.Error("Expected crossbow to be equipped in weapon slot")
	}

	if len(inventory.Carried) != 1 {
		t.Errorf("Expected 1 carried item after equipping, got %d", len(inventory.Carried))
	}
}
