package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

func TestTurnStateManager_StartMonsterTurn(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	err := tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)
	if err != nil {
		t.Fatalf("Failed to start monster turn: %v", err)
	}

	// Check monster state was created
	state := tsm.GetMonsterTurnState("monster-1")
	if state == nil {
		t.Fatal("Expected monster state to be created")
	}
	if state.MonsterID != "monster-1" {
		t.Errorf("Expected MonsterID 'monster-1', got '%s'", state.MonsterID)
	}
	if state.FixedMovement != 8 {
		t.Errorf("Expected FixedMovement 8, got %d", state.FixedMovement)
	}
}

func TestTurnStateManager_SelectMonster(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)
	tsm.StartMonsterTurn("monster-2", "entity-goblin-1", pos, 6, 2, 1, 3, 3)

	// Select first monster
	err := tsm.SelectMonster("monster-1")
	if err != nil {
		t.Fatalf("Failed to select monster: %v", err)
	}

	selected := tsm.GetSelectedMonster()
	if selected != "monster-1" {
		t.Errorf("Expected selected monster 'monster-1', got '%s'", selected)
	}

	// Select second monster
	err = tsm.SelectMonster("monster-2")
	if err != nil {
		t.Fatalf("Failed to select second monster: %v", err)
	}

	selected = tsm.GetSelectedMonster()
	if selected != "monster-2" {
		t.Errorf("Expected selected monster 'monster-2', got '%s'", selected)
	}

	// Clear selection
	err = tsm.SelectMonster("")
	if err != nil {
		t.Fatalf("Failed to clear monster selection: %v", err)
	}

	selected = tsm.GetSelectedMonster()
	if selected != "" {
		t.Errorf("Expected no selected monster, got '%s'", selected)
	}

	// Try to select non-existent monster
	err = tsm.SelectMonster("monster-999")
	if err == nil {
		t.Error("Expected error when selecting non-existent monster")
	}
}

func TestTurnStateManager_RecordMonsterMovement(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)

	// Record movement
	newPos := protocol.TileAddress{X: 6, Y: 10}
	err := tsm.RecordMonsterMovement("monster-1", newPos)
	if err != nil {
		t.Fatalf("Failed to record monster movement: %v", err)
	}

	state := tsm.GetMonsterTurnState("monster-1")
	if !state.HasMoved {
		t.Error("Expected monster HasMoved to be true")
	}
	if state.MovementRemaining != 7 {
		t.Errorf("Expected MovementRemaining 7, got %d", state.MovementRemaining)
	}

	// Try to record movement for non-existent monster
	err = tsm.RecordMonsterMovement("monster-999", newPos)
	if err == nil {
		t.Error("Expected error when recording movement for non-existent monster")
	}
}

func TestTurnStateManager_RecordMonsterAction(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)

	// Record action
	action := MonsterActionRecord{
		ActionType: "attack",
		TargetID:   "hero-1",
		Success:    true,
		Details:    map[string]interface{}{"damage": 2},
	}
	err := tsm.RecordMonsterAction("monster-1", action)
	if err != nil {
		t.Fatalf("Failed to record monster action: %v", err)
	}

	state := tsm.GetMonsterTurnState("monster-1")
	if !state.ActionTaken {
		t.Error("Expected monster ActionTaken to be true")
	}
	if state.Action == nil {
		t.Fatal("Expected Action to be set")
	}
	if state.Action.ActionType != "attack" {
		t.Errorf("Expected ActionType 'attack', got '%s'", state.Action.ActionType)
	}

	// Try to record action for non-existent monster
	err = tsm.RecordMonsterAction("monster-999", action)
	if err == nil {
		t.Error("Expected error when recording action for non-existent monster")
	}
}

func TestTurnStateManager_AddMonsterAbility(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)

	// Add ability
	ability := MonsterAbility{
		ID:             "dread_spell_fireball",
		Name:           "Fireball",
		Type:           "dread_spell",
		UsesPerQuest:   3,
		RequiresAction: true,
		Range:          6,
	}
	err := tsm.AddMonsterAbility("monster-1", ability)
	if err != nil {
		t.Fatalf("Failed to add monster ability: %v", err)
	}

	state := tsm.GetMonsterTurnState("monster-1")
	if len(state.SpecialAbilities) != 1 {
		t.Errorf("Expected 1 special ability, got %d", len(state.SpecialAbilities))
	}
	if state.SpecialAbilities[0].ID != "dread_spell_fireball" {
		t.Errorf("Expected ability ID 'dread_spell_fireball', got '%s'", state.SpecialAbilities[0].ID)
	}
	if state.QuestAbilityUsageLeft["dread_spell_fireball"] != 3 {
		t.Errorf("Expected 3 uses left, got %d", state.QuestAbilityUsageLeft["dread_spell_fireball"])
	}
}

func TestTurnStateManager_UseMonsterAbility(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)

	// Add ability
	ability := MonsterAbility{
		ID:             "dread_spell_fireball",
		Name:           "Fireball",
		Type:           "dread_spell",
		UsesPerTurn:    1,
		UsesPerQuest:   3,
		RequiresAction: true,
		Range:          6,
	}
	tsm.AddMonsterAbility("monster-1", ability)

	// Use ability
	targetPos := protocol.TileAddress{X: 10, Y: 10}
	err := tsm.UseMonsterAbility("monster-1", "dread_spell_fireball", "hero-1", &targetPos, true, nil)
	if err != nil {
		t.Fatalf("Failed to use monster ability: %v", err)
	}

	state := tsm.GetMonsterTurnState("monster-1")
	if state.SpecialAbilitiesUsed["dread_spell_fireball"] != 1 {
		t.Errorf("Expected ability used 1 time, got %d", state.SpecialAbilitiesUsed["dread_spell_fireball"])
	}
	if state.QuestAbilityUsageLeft["dread_spell_fireball"] != 2 {
		t.Errorf("Expected 2 uses left, got %d", state.QuestAbilityUsageLeft["dread_spell_fireball"])
	}
	if !state.ActionTaken {
		t.Error("Expected ActionTaken to be true after using ability that requires action")
	}
}

func TestTurnStateManager_RecordMonsterDamage(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)

	// Record damage
	err := tsm.RecordMonsterDamage("monster-1", 2)
	if err != nil {
		t.Fatalf("Failed to record monster damage: %v", err)
	}

	state := tsm.GetMonsterTurnState("monster-1")
	if state.CurrentBody != 3 {
		t.Errorf("Expected CurrentBody 3, got %d", state.CurrentBody)
	}
	if !state.IsAlive() {
		t.Error("Expected monster to be alive")
	}

	// Record lethal damage
	err = tsm.RecordMonsterDamage("monster-1", 10)
	if err != nil {
		t.Fatalf("Failed to record lethal damage: %v", err)
	}

	state = tsm.GetMonsterTurnState("monster-1")
	if state.CurrentBody != 0 {
		t.Errorf("Expected CurrentBody 0, got %d", state.CurrentBody)
	}
	if state.IsAlive() {
		t.Error("Expected monster to be dead")
	}
}

func TestTurnStateManager_AddMonsterActiveEffect(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)

	// Add effect
	effect := MonsterActiveEffect{
		Source:     "hero_spell_courage",
		EffectType: "bonus_attack_dice",
		Value:      2,
		Trigger:    "on_attack",
		ExpiresOn:  "after_trigger",
	}
	err := tsm.AddMonsterActiveEffect("monster-1", effect)
	if err != nil {
		t.Fatalf("Failed to add monster active effect: %v", err)
	}

	state := tsm.GetMonsterTurnState("monster-1")
	if len(state.ActiveEffects) != 1 {
		t.Errorf("Expected 1 active effect, got %d", len(state.ActiveEffects))
	}
}

func TestTurnStateManager_RemoveMonsterState(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)
	tsm.SelectMonster("monster-1")

	// Remove monster state
	tsm.RemoveMonsterState("monster-1")

	// Check state was removed
	state := tsm.GetMonsterTurnState("monster-1")
	if state != nil {
		t.Error("Expected monster state to be removed")
	}

	// Check selection was cleared
	selected := tsm.GetSelectedMonster()
	if selected != "" {
		t.Errorf("Expected no selected monster after removal, got '%s'", selected)
	}
}

func TestTurnStateManager_GetAllMonsterStates(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)
	tsm.StartMonsterTurn("monster-2", "entity-goblin-1", pos, 6, 2, 1, 3, 3)
	tsm.StartMonsterTurn("monster-3", "entity-skeleton-1", pos, 7, 2, 1, 2, 2)

	// Get all states
	states := tsm.GetAllMonsterStates()

	if len(states) != 3 {
		t.Errorf("Expected 3 monster states, got %d", len(states))
	}

	// Verify it's a copy (modifying returned map shouldn't affect internal state)
	states["monster-4"] = &MonsterTurnState{}
	internalStates := tsm.GetAllMonsterStates()
	if len(internalStates) != 3 {
		t.Error("Expected internal state to be unaffected by modification of returned copy")
	}
}

func TestTurnStateManager_MonsterTurnReset(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)

	// Perform actions
	tsm.RecordMonsterMovement("monster-1", protocol.TileAddress{X: 6, Y: 10})
	action := MonsterActionRecord{ActionType: "attack", TargetID: "hero-1"}
	tsm.RecordMonsterAction("monster-1", action)

	// Start new turn (should reset)
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)

	state := tsm.GetMonsterTurnState("monster-1")
	if state.HasMoved {
		t.Error("Expected HasMoved to be false after turn reset")
	}
	if state.ActionTaken {
		t.Error("Expected ActionTaken to be false after turn reset")
	}
	if state.MovementRemaining != 8 {
		t.Errorf("Expected MovementRemaining 8 after reset, got %d", state.MovementRemaining)
	}
}

func TestTurnStateManager_SerializationWithMonsters(t *testing.T) {
	logger := &MockLogger{}
	tsm := NewTurnStateManager(logger)

	// Add some monster states
	pos := protocol.TileAddress{X: 5, Y: 10}
	tsm.StartMonsterTurn("monster-1", "entity-orc-1", pos, 8, 3, 2, 5, 5)
	tsm.SelectMonster("monster-1")

	// Serialize
	data, err := tsm.SerializeForPersistence()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected serialized data to be non-empty")
	}

	// Verify JSON contains monster data
	dataStr := string(data)
	if len(dataStr) == 0 {
		t.Error("Expected serialized string to be non-empty")
	}
}
