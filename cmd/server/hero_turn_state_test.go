package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

func TestHeroTurnState_InitialState(t *testing.T) {
	startPos := protocol.TileAddress{X: 5, Y: 10}
	state := NewHeroTurnState("hero-1", "player-1", 1, startPos)

	if state.HeroID != "hero-1" {
		t.Errorf("Expected HeroID 'hero-1', got '%s'", state.HeroID)
	}

	if state.PlayerID != "player-1" {
		t.Errorf("Expected PlayerID 'player-1', got '%s'", state.PlayerID)
	}

	if state.TurnNumber != 1 {
		t.Errorf("Expected TurnNumber 1, got %d", state.TurnNumber)
	}

	if state.HasMoved {
		t.Error("HasMoved should be false initially")
	}

	if state.ActionTaken {
		t.Error("ActionTaken should be false initially")
	}

	if state.MovementDice.Rolled {
		t.Error("MovementDice.Rolled should be false initially")
	}

	if state.TurnStartPosition != startPos {
		t.Errorf("Expected TurnStartPosition %v, got %v", startPos, state.TurnStartPosition)
	}
}

func TestHeroTurnState_MovementDiceRoll(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})

	// Roll dice
	diceResults := []int{2, 3, 4}
	err := state.RollMovementDice(diceResults)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !state.MovementDice.Rolled {
		t.Error("MovementDice.Rolled should be true after rolling")
	}

	expectedTotal := 9
	if state.MovementDice.TotalMovement != expectedTotal {
		t.Errorf("Expected TotalMovement %d, got %d", expectedTotal, state.MovementDice.TotalMovement)
	}

	if state.MovementDice.MovementRemaining != expectedTotal {
		t.Errorf("Expected MovementRemaining %d, got %d", expectedTotal, state.MovementDice.MovementRemaining)
	}

	// Try rolling again (should fail)
	err = state.RollMovementDice([]int{1, 1})
	if err == nil {
		t.Error("Expected error when rolling dice twice, got nil")
	}
}

func TestHeroTurnState_CanMove_BeforeRoll(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})

	canMove, reason := state.CanMove()
	if canMove {
		t.Error("Should not be able to move before rolling dice")
	}
	if reason != "must roll movement dice first" {
		t.Errorf("Expected 'must roll movement dice first', got '%s'", reason)
	}
}

func TestHeroTurnState_CanTakeAction_BeforeRoll(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})

	canAct, reason := state.CanTakeAction()
	if canAct {
		t.Error("Should not be able to take action before rolling dice")
	}
	if reason != "must roll movement dice first" {
		t.Errorf("Expected 'must roll movement dice first', got '%s'", reason)
	}
}

func TestHeroTurnState_MovementFirst_Strategy(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})
	state.RollMovementDice([]int{3, 3, 3})

	// Initially can choose
	if strategy := state.GetTurnStrategy(); strategy != "choose" {
		t.Errorf("Expected strategy 'choose', got '%s'", strategy)
	}

	// Move once
	err := state.RecordMovement(protocol.TileAddress{X: 1, Y: 0})
	if err != nil {
		t.Fatalf("Expected no error moving, got: %v", err)
	}

	// Should be locked into move-first strategy
	if strategy := state.GetTurnStrategy(); strategy != "move_first" {
		t.Errorf("Expected strategy 'move_first', got '%s'", strategy)
	}

	if !state.HasMoved {
		t.Error("HasMoved should be true after moving")
	}

	// Should still be able to move
	canMove, _ := state.CanMove()
	if !canMove {
		t.Error("Should be able to continue moving before taking action")
	}

	// Should be able to take action
	canAct, _ := state.CanTakeAction()
	if !canAct {
		t.Error("Should be able to take action after moving")
	}

	// Take action
	action := ActionRecord{
		ActionType: "attack",
		TargetID:   "monster-1",
		Success:    true,
		Details:    make(map[string]interface{}),
	}
	err = state.RecordAction(action)
	if err != nil {
		t.Fatalf("Expected no error taking action, got: %v", err)
	}

	if !state.ActionTaken {
		t.Error("ActionTaken should be true after taking action")
	}

	// Strategy should now be complete (or trying to move more would fail)
	if strategy := state.GetTurnStrategy(); strategy != "complete" {
		t.Errorf("Expected strategy 'complete', got '%s'", strategy)
	}

	// Should NOT be able to move again (move-first strategy, action taken)
	canMove, reason := state.CanMove()
	if canMove {
		t.Error("Should not be able to move after action in move-first strategy")
	}
	if reason == "" {
		t.Error("Expected reason why can't move")
	}
}

func TestHeroTurnState_ActionFirst_Strategy(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})
	state.RollMovementDice([]int{3, 3, 3})

	// Take action first
	action := ActionRecord{
		ActionType:  "search_treasure",
		LocationKey: "room-17",
		Success:     true,
		Details:     make(map[string]interface{}),
	}
	err := state.RecordAction(action)
	if err != nil {
		t.Fatalf("Expected no error taking action, got: %v", err)
	}

	// Should be locked into act-first strategy
	if strategy := state.GetTurnStrategy(); strategy != "act_first" {
		t.Errorf("Expected strategy 'act_first', got '%s'", strategy)
	}

	if state.ActionTaken {
		// Action taken
	} else {
		t.Error("ActionTaken should be true")
	}

	if state.HasMoved {
		t.Error("HasMoved should still be false")
	}

	// Should still be able to move
	canMove, _ := state.CanMove()
	if !canMove {
		t.Error("Should be able to move after taking action (act-first strategy)")
	}

	// Should NOT be able to take another action
	canAct, _ := state.CanTakeAction()
	if canAct {
		t.Error("Should not be able to take another action")
	}

	// Move
	err = state.RecordMovement(protocol.TileAddress{X: 1, Y: 0})
	if err != nil {
		t.Fatalf("Expected no error moving, got: %v", err)
	}

	if !state.HasMoved {
		t.Error("HasMoved should be true after moving")
	}

	// Strategy should now be complete
	if strategy := state.GetTurnStrategy(); strategy != "complete" {
		t.Errorf("Expected strategy 'complete', got '%s'", strategy)
	}
}

func TestHeroTurnState_SplitMovement_WithFlag(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})
	state.RollMovementDice([]int{3, 3, 3})

	// Enable split movement
	state.TurnFlags["can_split_movement"] = true

	// Move
	state.RecordMovement(protocol.TileAddress{X: 1, Y: 0})

	// Take action
	action := ActionRecord{ActionType: "attack", TargetID: "monster-1", Success: true, Details: make(map[string]interface{})}
	state.RecordAction(action)

	// Should still be able to move (split movement allowed)
	canMove, reason := state.CanMove()
	if !canMove {
		t.Errorf("Should be able to move with split movement flag, reason: %s", reason)
	}

	// Move again
	err := state.RecordMovement(protocol.TileAddress{X: 2, Y: 0})
	if err != nil {
		t.Fatalf("Expected no error moving after action with split flag, got: %v", err)
	}
}

func TestHeroTurnState_MovementConsumption(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})
	state.RollMovementDice([]int{1, 1, 1}) // Total 3

	// Move 3 times
	for i := 1; i <= 3; i++ {
		err := state.RecordMovement(protocol.TileAddress{X: i, Y: 0})
		if err != nil {
			t.Fatalf("Expected no error on move %d, got: %v", i, err)
		}
	}

	if state.MovementDice.MovementUsed != 3 {
		t.Errorf("Expected MovementUsed 3, got %d", state.MovementDice.MovementUsed)
	}

	if state.MovementDice.MovementRemaining != 0 {
		t.Errorf("Expected MovementRemaining 0, got %d", state.MovementDice.MovementRemaining)
	}

	// Try to move again (should fail - no movement left)
	err := state.RecordMovement(protocol.TileAddress{X: 4, Y: 0})
	if err == nil {
		t.Error("Expected error when moving with no movement remaining")
	}
}

func TestHeroTurnState_SearchTreasure_OncePerRoom(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})

	locationKey := "room-17"

	// First search should be allowed
	canSearch, _ := state.CanSearchTreasure(locationKey)
	if !canSearch {
		t.Error("First treasure search should be allowed")
	}

	// Record treasure search
	state.RecordSearch("treasure", locationKey, "room", protocol.TileAddress{X: 5, Y: 5}, true, []string{"gold"})

	// Second search should NOT be allowed
	canSearch, reason := state.CanSearchTreasure(locationKey)
	if canSearch {
		t.Error("Second treasure search should not be allowed")
	}
	if reason == "" {
		t.Error("Expected reason why treasure search not allowed")
	}
}

func TestHeroTurnState_MultipleHeroes_SeparateSearches(t *testing.T) {
	// This test simulates two heroes searching the same room
	hero1 := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})
	hero2 := NewHeroTurnState("hero-2", "player-2", 1, protocol.TileAddress{X: 1, Y: 0})

	locationKey := "room-17"

	// Hero 1 searches
	hero1.RecordSearch("treasure", locationKey, "room", protocol.TileAddress{X: 5, Y: 5}, true, []string{"gold"})

	// Hero 1 cannot search again
	canSearch, _ := hero1.CanSearchTreasure(locationKey)
	if canSearch {
		t.Error("Hero 1 should not be able to search again")
	}

	// Hero 2 should still be able to search (separate hero)
	canSearch, _ = hero2.CanSearchTreasure(locationKey)
	if !canSearch {
		t.Error("Hero 2 should be able to search (different hero)")
	}
}

func TestHeroTurnState_ActiveEffects(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})

	// Add effect
	effect := ActiveEffect{
		Source:     "potion_of_strength",
		EffectType: "bonus_attack_dice",
		Value:      2,
		Trigger:    "next_attack",
		ExpiresOn:  "after_trigger",
	}
	state.AddActiveEffect(effect)

	if len(state.ActiveEffects) != 1 {
		t.Errorf("Expected 1 active effect, got %d", len(state.ActiveEffects))
	}

	// Trigger effects
	triggered := state.TriggerEffects("next_attack")
	if len(triggered) != 1 {
		t.Errorf("Expected 1 triggered effect, got %d", len(triggered))
	}

	if !state.ActiveEffects[0].Applied {
		t.Error("Effect should be marked as applied")
	}

	// Triggering again should return no effects (already applied)
	triggered = state.TriggerEffects("next_attack")
	if len(triggered) != 0 {
		t.Errorf("Expected 0 triggered effects on second trigger, got %d", len(triggered))
	}
}

func TestHeroTurnState_ResetForNewTurn(t *testing.T) {
	state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})

	// Do some stuff
	state.RollMovementDice([]int{3, 3})
	state.RecordMovement(protocol.TileAddress{X: 1, Y: 0})
	state.RecordAction(ActionRecord{ActionType: "attack", Success: true, Details: make(map[string]interface{})})
	state.RecordActivity(Activity{Type: "use_item", ItemID: "potion-1", ItemName: "Healing Potion"})

	// Reset for new turn
	state.ResetForNewTurn(2)

	if state.TurnNumber != 2 {
		t.Errorf("Expected TurnNumber 2, got %d", state.TurnNumber)
	}

	if state.MovementDice.Rolled {
		t.Error("MovementDice.Rolled should be false after reset")
	}

	if state.HasMoved {
		t.Error("HasMoved should be false after reset")
	}

	if state.ActionTaken {
		t.Error("ActionTaken should be false after reset")
	}

	if len(state.Activities) != 0 {
		t.Errorf("Activities should be empty after reset, got %d", len(state.Activities))
	}

	if len(state.ItemUsageThisTurn) != 0 {
		t.Errorf("ItemUsageThisTurn should be empty after reset, got %d", len(state.ItemUsageThisTurn))
	}
}

// TestHeroTurnState_NoMovementAfterBothMoveAndAction verifies that heroes cannot continue moving
// after they have both moved and taken an action (fixes high priority bug #1)
func TestHeroTurnState_NoMovementAfterBothMoveAndAction(t *testing.T) {
	// Test Scenario 1: Move first, then action, then try to move again
	t.Run("MoveFirst_ThenAction_NoMoreMovement", func(t *testing.T) {
		state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})
		state.RollMovementDice([]int{3, 3, 3})

		// Move once
		err := state.RecordMovement(protocol.TileAddress{X: 1, Y: 0})
		if err != nil {
			t.Fatalf("Expected no error on first move, got: %v", err)
		}

		// Take action
		action := ActionRecord{
			ActionType: "attack",
			TargetID:   "monster-1",
			Success:    true,
			Details:    make(map[string]interface{}),
		}
		err = state.RecordAction(action)
		if err != nil {
			t.Fatalf("Expected no error taking action, got: %v", err)
		}

		// Try to move again - should FAIL
		canMove, reason := state.CanMove()
		if canMove {
			t.Error("Should NOT be able to move after both moving and taking an action")
		}
		if reason == "" {
			t.Error("Expected reason why movement is blocked")
		}

		// Verify the error
		err = state.RecordMovement(protocol.TileAddress{X: 2, Y: 0})
		if err == nil {
			t.Error("Expected error when trying to move after action in move-first strategy")
		}
	})

	// Test Scenario 2: Action first, then move, then try to move again
	t.Run("ActionFirst_ThenMove_NoMoreMovement", func(t *testing.T) {
		state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})
		state.RollMovementDice([]int{3, 3, 3})

		// Take action first
		action := ActionRecord{
			ActionType: "attack",
			TargetID:   "monster-1",
			Success:    true,
			Details:    make(map[string]interface{}),
		}
		err := state.RecordAction(action)
		if err != nil {
			t.Fatalf("Expected no error taking action, got: %v", err)
		}

		// Move once
		err = state.RecordMovement(protocol.TileAddress{X: 1, Y: 0})
		if err != nil {
			t.Fatalf("Expected no error on first move after action, got: %v", err)
		}

		// Try to move again - should FAIL
		canMove, reason := state.CanMove()
		if canMove {
			t.Error("Should NOT be able to move again after both acting and moving")
		}
		if reason == "" {
			t.Error("Expected reason why movement is blocked")
		}

		// Verify the error
		err = state.RecordMovement(protocol.TileAddress{X: 2, Y: 0})
		if err == nil {
			t.Error("Expected error when trying to move again after action in act-first strategy")
		}
	})

	// Test Scenario 3: With split movement flag, should still be able to move
	t.Run("WithSplitMovementFlag_CanMoveAfterBoth", func(t *testing.T) {
		state := NewHeroTurnState("hero-1", "player-1", 1, protocol.TileAddress{X: 0, Y: 0})
		state.RollMovementDice([]int{3, 3, 3})
		state.TurnFlags["can_split_movement"] = true

		// Move
		err := state.RecordMovement(protocol.TileAddress{X: 1, Y: 0})
		if err != nil {
			t.Fatalf("Expected no error on first move, got: %v", err)
		}

		// Take action
		action := ActionRecord{
			ActionType: "attack",
			TargetID:   "monster-1",
			Success:    true,
			Details:    make(map[string]interface{}),
		}
		err = state.RecordAction(action)
		if err != nil {
			t.Fatalf("Expected no error taking action, got: %v", err)
		}

		// Should be able to move again with split movement flag
		canMove, reason := state.CanMove()
		if !canMove {
			t.Errorf("Should be able to move with split movement flag, reason: %s", reason)
		}

		// Move again should succeed
		err = state.RecordMovement(protocol.TileAddress{X: 2, Y: 0})
		if err != nil {
			t.Fatalf("Expected no error on second move with split flag, got: %v", err)
		}
	})
}
