package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

func TestNewMonsterTurnState(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	if mts.MonsterID != "monster-1" {
		t.Errorf("Expected MonsterID 'monster-1', got '%s'", mts.MonsterID)
	}
	if mts.EntityID != "entity-orc-1" {
		t.Errorf("Expected EntityID 'entity-orc-1', got '%s'", mts.EntityID)
	}
	if mts.TurnNumber != 1 {
		t.Errorf("Expected TurnNumber 1, got %d", mts.TurnNumber)
	}
	if mts.FixedMovement != 8 {
		t.Errorf("Expected FixedMovement 8, got %d", mts.FixedMovement)
	}
	if mts.MovementRemaining != 8 {
		t.Errorf("Expected MovementRemaining 8, got %d", mts.MovementRemaining)
	}
	if mts.AttackDice != 3 {
		t.Errorf("Expected AttackDice 3, got %d", mts.AttackDice)
	}
	if mts.DefenseDice != 2 {
		t.Errorf("Expected DefenseDice 2, got %d", mts.DefenseDice)
	}
	if mts.BodyPoints != 5 {
		t.Errorf("Expected BodyPoints 5, got %d", mts.BodyPoints)
	}
	if mts.CurrentBody != 5 {
		t.Errorf("Expected CurrentBody 5, got %d", mts.CurrentBody)
	}
	if mts.HasMoved {
		t.Error("Expected HasMoved to be false")
	}
	if mts.ActionTaken {
		t.Error("Expected ActionTaken to be false")
	}
}

func TestMonsterCanMove(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	// Should be able to move initially
	canMove, reason := mts.CanMove()
	if !canMove {
		t.Errorf("Expected to be able to move, got reason: %s", reason)
	}

	// Consume all movement
	for i := 0; i < 8; i++ {
		newPos := protocol.TileAddress{X: 5 + i + 1, Y: 10}
		if err := mts.RecordMovement(newPos); err != nil {
			t.Fatalf("Failed to record movement: %v", err)
		}
	}

	// Should not be able to move after consuming all movement
	canMove, reason = mts.CanMove()
	if canMove {
		t.Error("Expected to not be able to move after consuming all movement")
	}
	if reason != "no movement remaining" {
		t.Errorf("Expected reason 'no movement remaining', got '%s'", reason)
	}
}

func TestMonsterRecordMovement(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	newPos := protocol.TileAddress{X: 6, Y: 10}
	if err := mts.RecordMovement(newPos); err != nil {
		t.Fatalf("Failed to record movement: %v", err)
	}

	if !mts.HasMoved {
		t.Error("Expected HasMoved to be true after movement")
	}
	if mts.MovementUsed != 1 {
		t.Errorf("Expected MovementUsed 1, got %d", mts.MovementUsed)
	}
	if mts.MovementRemaining != 7 {
		t.Errorf("Expected MovementRemaining 7, got %d", mts.MovementRemaining)
	}
	if len(mts.MovementPath) != 1 {
		t.Errorf("Expected MovementPath length 1, got %d", len(mts.MovementPath))
	}
	if mts.CurrentPosition.X != 6 || mts.CurrentPosition.Y != 10 {
		t.Errorf("Expected CurrentPosition (6,10), got (%d,%d)", mts.CurrentPosition.X, mts.CurrentPosition.Y)
	}
	if len(mts.TurnEvents) != 1 {
		t.Errorf("Expected 1 turn event, got %d", len(mts.TurnEvents))
	}
	if mts.TurnEvents[0].EventType != "moved" {
		t.Errorf("Expected event type 'moved', got '%s'", mts.TurnEvents[0].EventType)
	}
}

func TestMonsterCanTakeAction(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	// Should be able to take action initially
	canAct, reason := mts.CanTakeAction()
	if !canAct {
		t.Errorf("Expected to be able to take action, got reason: %s", reason)
	}

	// Record an action
	action := MonsterActionRecord{
		ActionType: "attack",
		TargetID:   "hero-1",
		Success:    true,
		Details:    map[string]interface{}{"damage": 2},
	}
	if err := mts.RecordAction(action); err != nil {
		t.Fatalf("Failed to record action: %v", err)
	}

	// Should not be able to take action after taking one
	canAct, reason = mts.CanTakeAction()
	if canAct {
		t.Error("Expected to not be able to take action after taking one")
	}
	if reason != "action already taken this turn" {
		t.Errorf("Expected reason 'action already taken this turn', got '%s'", reason)
	}
}

func TestMonsterRecordAction(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	action := MonsterActionRecord{
		ActionType: "attack",
		TargetID:   "hero-1",
		Success:    true,
		Details:    map[string]interface{}{"damage": 2},
	}
	if err := mts.RecordAction(action); err != nil {
		t.Fatalf("Failed to record action: %v", err)
	}

	if !mts.ActionTaken {
		t.Error("Expected ActionTaken to be true after recording action")
	}
	if mts.Action == nil {
		t.Fatal("Expected Action to be set")
	}
	if mts.Action.ActionType != "attack" {
		t.Errorf("Expected ActionType 'attack', got '%s'", mts.Action.ActionType)
	}
	if mts.Action.TargetID != "hero-1" {
		t.Errorf("Expected TargetID 'hero-1', got '%s'", mts.Action.TargetID)
	}
	if len(mts.TurnEvents) != 1 {
		t.Errorf("Expected 1 turn event, got %d", len(mts.TurnEvents))
	}
}

func TestMonsterSpecialAbilities(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	// Add a special ability
	ability := MonsterAbility{
		ID:             "dread_spell_fireball",
		Name:           "Fireball",
		Type:           "dread_spell",
		UsesPerTurn:    1,
		UsesPerQuest:   3,
		RequiresAction: true,
		Range:          6,
		Description:    "Cast a fireball at a hero",
		EffectDetails:  map[string]interface{}{"damage": "2d6"},
	}
	mts.SpecialAbilities = append(mts.SpecialAbilities, ability)
	mts.QuestAbilityUsageLeft["dread_spell_fireball"] = 3

	// Should be able to use ability
	canUse, reason := mts.CanUseAbility("dread_spell_fireball")
	if !canUse {
		t.Errorf("Expected to be able to use ability, got reason: %s", reason)
	}

	// Use the ability
	targetPos := protocol.TileAddress{X: 10, Y: 10}
	if err := mts.UseAbility("dread_spell_fireball", "hero-1", &targetPos, true, nil); err != nil {
		t.Fatalf("Failed to use ability: %v", err)
	}

	// Check usage tracking
	if mts.SpecialAbilitiesUsed["dread_spell_fireball"] != 1 {
		t.Errorf("Expected ability used 1 time, got %d", mts.SpecialAbilitiesUsed["dread_spell_fireball"])
	}
	if mts.QuestAbilityUsageLeft["dread_spell_fireball"] != 2 {
		t.Errorf("Expected 2 uses left, got %d", mts.QuestAbilityUsageLeft["dread_spell_fireball"])
	}

	// Should not be able to use again this turn (UsesPerTurn = 1)
	canUse, reason = mts.CanUseAbility("dread_spell_fireball")
	if canUse {
		t.Error("Expected to not be able to use ability again this turn")
	}
	if reason != "ability already used maximum times this turn" {
		t.Errorf("Expected reason about per-turn limit, got '%s'", reason)
	}

	// Action should be taken since ability requires action
	if !mts.ActionTaken {
		t.Error("Expected ActionTaken to be true after using ability that requires action")
	}
}

func TestMonsterRecordDamage(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	// Record damage
	mts.RecordDamage(2)

	if mts.CurrentBody != 3 {
		t.Errorf("Expected CurrentBody 3, got %d", mts.CurrentBody)
	}
	if !mts.IsAlive() {
		t.Error("Expected monster to be alive")
	}

	// Record lethal damage
	mts.RecordDamage(10)

	if mts.CurrentBody != 0 {
		t.Errorf("Expected CurrentBody 0, got %d", mts.CurrentBody)
	}
	if mts.IsAlive() {
		t.Error("Expected monster to be dead")
	}

	// Check events
	if len(mts.TurnEvents) != 2 {
		t.Errorf("Expected 2 damage events, got %d", len(mts.TurnEvents))
	}
}

func TestMonsterActiveEffects(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	// Add an active effect
	effect := MonsterActiveEffect{
		Source:     "hero_spell_courage",
		EffectType: "bonus_attack_dice",
		Value:      2,
		Trigger:    "on_attack",
		ExpiresOn:  "after_trigger",
	}
	mts.AddActiveEffect(effect)

	if len(mts.ActiveEffects) != 1 {
		t.Errorf("Expected 1 active effect, got %d", len(mts.ActiveEffects))
	}

	// Trigger the effect
	triggered := mts.TriggerEffects("on_attack")
	if len(triggered) != 1 {
		t.Errorf("Expected 1 triggered effect, got %d", len(triggered))
	}
	if triggered[0].Value != 2 {
		t.Errorf("Expected triggered effect value 2, got %d", triggered[0].Value)
	}

	// Effect should be marked as applied
	if !mts.ActiveEffects[0].Applied {
		t.Error("Expected effect to be marked as applied")
	}

	// Triggering again should not return the effect
	triggered = mts.TriggerEffects("on_attack")
	if len(triggered) != 0 {
		t.Errorf("Expected 0 triggered effects on second trigger, got %d", len(triggered))
	}
}

func TestMonsterResetForNewTurn(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	// Perform actions
	newPos := protocol.TileAddress{X: 6, Y: 10}
	mts.RecordMovement(newPos)
	action := MonsterActionRecord{ActionType: "attack", TargetID: "hero-1"}
	mts.RecordAction(action)
	mts.SpecialAbilitiesUsed["test_ability"] = 1

	// Add effect that should expire
	effect := MonsterActiveEffect{
		Source:     "test",
		EffectType: "test",
		ExpiresOn:  "end_of_gm_turn",
	}
	mts.AddActiveEffect(effect)

	// Reset for new turn
	mts.ResetForNewTurn(2)

	// Check reset state
	if mts.TurnNumber != 2 {
		t.Errorf("Expected TurnNumber 2, got %d", mts.TurnNumber)
	}
	if mts.HasMoved {
		t.Error("Expected HasMoved to be false after reset")
	}
	if mts.ActionTaken {
		t.Error("Expected ActionTaken to be false after reset")
	}
	if mts.MovementUsed != 0 {
		t.Errorf("Expected MovementUsed 0, got %d", mts.MovementUsed)
	}
	if mts.MovementRemaining != 8 {
		t.Errorf("Expected MovementRemaining 8, got %d", mts.MovementRemaining)
	}
	if len(mts.MovementPath) != 0 {
		t.Errorf("Expected empty MovementPath, got %d items", len(mts.MovementPath))
	}
	if mts.Action != nil {
		t.Error("Expected Action to be nil after reset")
	}
	if len(mts.TurnEvents) != 0 {
		t.Errorf("Expected empty TurnEvents, got %d items", len(mts.TurnEvents))
	}
	if len(mts.SpecialAbilitiesUsed) != 0 {
		t.Errorf("Expected empty SpecialAbilitiesUsed, got %d items", len(mts.SpecialAbilitiesUsed))
	}
	if len(mts.ActiveEffects) != 0 {
		t.Errorf("Expected expired effects to be cleared, got %d effects", len(mts.ActiveEffects))
	}
}

func TestMonsterGetTurnSummary(t *testing.T) {
	pos := protocol.TileAddress{X: 5, Y: 10}
	mts := NewMonsterTurnState("monster-1", "entity-orc-1", 1, pos, 8, 3, 2, 5, 5)

	// Initially no actions
	summary := mts.GetTurnSummary()
	if summary != "no actions" {
		t.Errorf("Expected 'no actions', got '%s'", summary)
	}

	// After moving
	mts.RecordMovement(protocol.TileAddress{X: 6, Y: 10})
	summary = mts.GetTurnSummary()
	if summary != "moved only" {
		t.Errorf("Expected 'moved only', got '%s'", summary)
	}

	// After also taking action
	mts.RecordAction(MonsterActionRecord{ActionType: "attack"})
	summary = mts.GetTurnSummary()
	if summary != "moved and acted" {
		t.Errorf("Expected 'moved and acted', got '%s'", summary)
	}

	// Reset and only take action
	mts.ResetForNewTurn(2)
	mts.RecordAction(MonsterActionRecord{ActionType: "attack"})
	summary = mts.GetTurnSummary()
	if summary != "acted only" {
		t.Errorf("Expected 'acted only', got '%s'", summary)
	}
}
