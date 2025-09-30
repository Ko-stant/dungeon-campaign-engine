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
		if battleAxe.AttackDice != 3 {
			t.Errorf("Expected battle axe attack dice 3, got %d", battleAxe.AttackDice)
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
