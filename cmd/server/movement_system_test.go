package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

func createTestMovementSystem() *HeroActionSystem {
	// Create a simple test board
	segment := geometry.Segment{
		Width:           10,
		Height:          10,
		WallsVertical:   []geometry.EdgeAddress{}, // No walls for simple testing
		WallsHorizontal: []geometry.EdgeAddress{}, // No walls for simple testing
		DoorSockets:     []geometry.EdgeAddress{},
	}

	gameState := &GameState{
		Segment: segment,
		Entities: map[string]protocol.TileAddress{
			"hero-1": {X: 5, Y: 5},
		},
		Doors:              make(map[string]*DoorInfo),
		DoorByEdge:         make(map[geometry.EdgeAddress]string),
		RevealedRegions:    make(map[int]bool),
		KnownRegions:       make(map[int]bool),
		KnownDoors:         make(map[string]bool),
		KnownBlockingWalls: make(map[string]bool),
		BlockedWalls:       make(map[geometry.EdgeAddress]bool),
		BlockedTiles:       make(map[protocol.TileAddress]bool),
	}

	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}

	debugConfig := DebugConfig{
		Enabled:            true,
		AllowStateChanges:  true,
		AllowTeleportation: true,
		AllowMapReveal:     true,
		AllowDiceOverride:  true,
		LogDebugActions:    true,
	}
	debugSystem := NewDebugSystem(debugConfig, gameState, broadcaster, logger)
	diceSystem := NewDiceSystem(debugSystem)
	turnManager := NewTurnManager(broadcaster, logger, diceSystem)

	// Add test player
	player := NewPlayer("player-1", "Test Hero", "hero-1", Barbarian)
	turnManager.AddPlayer(player)

	return NewHeroActionSystem(gameState, turnManager, broadcaster, logger, debugSystem)
}

func TestMovement_OncePerTurn(t *testing.T) {
	has := createTestMovementSystem()

	// Roll movement dice to allow movement
	_, err := has.turnManager.RollMovementDice()
	if err != nil {
		t.Fatalf("Failed to roll movement dice: %v", err)
	}

	// First movement should succeed
	request1 := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 1.0,
			"dy": 0.0,
		},
	}

	result1, err := has.ProcessMovement(request1)
	_ = result1 // Mark result1 as used to avoid unused variable warning

	if err != nil {
		t.Fatalf("Expected first movement to succeed, got: %v", err)
	}

	if !result1.Success {
		t.Fatalf("Expected first movement success, got: %s", result1.Message)
	}

	// Check that hero moved
	newPos := has.gameState.Entities["hero-1"]
	if newPos.X != 6 || newPos.Y != 5 {
		t.Errorf("Expected hero at (6,5), got (%d,%d)", newPos.X, newPos.Y)
	}

	// Check movement state
	turnState := has.turnManager.GetTurnState()
	if !turnState.HasMoved {
		t.Error("Expected HasMoved to be true after movement")
	}

	// Second movement should fail
	request2 := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveAfterAction,
		Parameters: map[string]any{
			"dx": 0.0,
			"dy": 1.0,
		},
	}

	_, err = has.ProcessMovement(request2)

	if err == nil {
		t.Fatal("Expected second movement to fail")
	}

	if err.Error() != "player cannot move right now" {
		t.Errorf("Expected 'cannot move' error, got: %v", err)
	}
}

func TestMovement_ConsumesmovementDice(t *testing.T) {
	has := createTestMovementSystem()

	// Roll movement dice to set up movement points
	_, err := has.turnManager.RollMovementDice()
	if err != nil {
		t.Fatalf("Failed to roll movement dice: %v", err)
	}

	// Check initial movement points
	initialTurnState := has.turnManager.GetTurnState()
	initialMovement := initialTurnState.MovementLeft

	request := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 1.0,
			"dy": 1.0, // 2 squares diagonal
		},
	}

	_, err = has.ProcessMovement(request)

	if err != nil {
		t.Fatalf("Expected movement to succeed, got: %v", err)
	}

	// Check movement points were consumed
	newTurnState := has.turnManager.GetTurnState()
	expectedMovement := initialMovement - 2 // dx + dy = 1 + 1 = 2

	if newTurnState.MovementLeft != expectedMovement {
		t.Errorf("Expected movement left %d, got %d", expectedMovement, newTurnState.MovementLeft)
	}
}

func TestMovement_BlockedByBounds(t *testing.T) {
	has := createTestMovementSystem()

	// Try to move outside board bounds
	request := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 10.0, // Would move to x=15, outside 10x10 board
			"dy": 0.0,
		},
	}

	_, err := has.ProcessMovement(request)

	if err == nil {
		t.Fatal("Expected movement to fail due to bounds")
	}

	// Hero should still be at original position
	pos := has.gameState.Entities["hero-1"]
	if pos.X != 5 || pos.Y != 5 {
		t.Errorf("Expected hero to remain at (5,5), got (%d,%d)", pos.X, pos.Y)
	}
}

func TestMovement_InvalidParameters(t *testing.T) {
	has := createTestMovementSystem()

	// Test missing parameters
	request := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 1.0,
			// Missing dy
		},
	}

	result, err := has.ProcessMovement(request)

	if err == nil {
		t.Fatal("Expected error for missing parameters")
	}

	if result != nil && result.Success {
		t.Error("Expected movement to fail with missing parameters")
	}
}

func TestMovement_ZeroMovement(t *testing.T) {
	has := createTestMovementSystem()

	request := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 0.0,
			"dy": 0.0,
		},
	}

	result, err := has.ProcessMovement(request)

	if err == nil {
		t.Fatal("Expected error for zero movement")
	}

	if result != nil && result.Success {
		t.Error("Expected movement to fail with zero distance")
	}
}

func TestMovement_InsufficientmovementDice(t *testing.T) {
	has := createTestMovementSystem()

	// Roll movement dice first
	_, err := has.turnManager.RollMovementDice()
	if err != nil {
		t.Fatalf("Failed to roll movement dice: %v", err)
	}

	// Check initial movement points
	initialState := has.turnManager.GetTurnState()
	t.Logf("Initial movement points: %d", initialState.MovementLeft)

	// Consume most movement points first (should leave some points remaining)
	// Need to ensure we don't consume all points, just most of them
	pointsToConsume := initialState.MovementLeft - 1 // Leave 1 point
	if pointsToConsume <= 0 {
		t.Skip("Test requires at least 2 movement points from dice roll")
	}
	has.turnManager.ConsumeMovement(pointsToConsume, "move_before")

	// Check remaining points
	midState := has.turnManager.GetTurnState()
	t.Logf("Movement points after consuming %d: %d", pointsToConsume, midState.MovementLeft)

	// Try to move 2 squares
	request := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 2.0,
			"dy": 0.0,
		},
	}

	_, err = has.ProcessMovement(request)

	if err == nil {
		t.Fatal("Expected error for insufficient movement points")
	}
}

func TestMovement_AfterTurnReset(t *testing.T) {
	has := createTestMovementSystem()

	// Roll movement dice for first turn
	_, err := has.turnManager.RollMovementDice()
	if err != nil {
		t.Fatalf("Failed to roll movement dice: %v", err)
	}

	// Use movement in first turn
	request := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 1.0,
			"dy": 0.0,
		},
	}

	_, err = has.ProcessMovement(request)
	if err != nil {
		t.Fatalf("Expected first movement to succeed, got: %v", err)
	}

	// End turn and start new turn (simulate turn advancement)
	has.turnManager.EndTurn()
	has.turnManager.EndTurn() // Complete GM turn

	// Check that movement is reset for new turn
	turnState := has.turnManager.GetTurnState()
	if turnState.HasMoved {
		t.Error("Expected HasMoved to be false after turn reset")
	}

	if turnState.MovementDiceRolled {
		t.Error("Expected MovementDiceRolled to be false after turn reset")
	}

	// Roll movement dice for new turn
	_, err = has.turnManager.RollMovementDice()
	if err != nil {
		t.Fatalf("Failed to roll movement dice in new turn: %v", err)
	}

	// Check movement points are available
	turnState = has.turnManager.GetTurnState()
	if turnState.MovementLeft <= 0 {
		t.Errorf("Expected positive movement points after rolling dice, got %d", turnState.MovementLeft)
	}

	// Should be able to move again
	_, err = has.ProcessMovement(request)
	if err != nil {
		t.Errorf("Expected movement to work in new turn, got: %v", err)
	}
}

func TestMovement_WrongPlayer(t *testing.T) {
	has := createTestMovementSystem()

	request := MovementRequest{
		PlayerID: "wrong-player",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 1.0,
			"dy": 0.0,
		},
	}

	_, err := has.ProcessMovement(request)

	if err == nil {
		t.Fatal("Expected error for wrong player")
	}
}

func TestMovement_WrongEntity(t *testing.T) {
	has := createTestMovementSystem()

	request := MovementRequest{
		PlayerID: "player-1",
		EntityID: "wrong-hero",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 1.0,
			"dy": 0.0,
		},
	}

	_, err := has.ProcessMovement(request)

	if err == nil {
		t.Fatal("Expected error for wrong entity")
	}
}

func TestMovement_BroadcastsUpdate(t *testing.T) {
	has := createTestMovementSystem()
	broadcaster := has.broadcaster.(*MockBroadcaster)

	// Roll movement dice to allow movement
	_, err := has.turnManager.RollMovementDice()
	if err != nil {
		t.Fatalf("Failed to roll movement dice: %v", err)
	}

	request := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 1.0,
			"dy": 0.0,
		},
	}

	_, err = has.ProcessMovement(request)

	if err != nil {
		t.Fatalf("Expected movement to succeed, got: %v", err)
	}

	if broadcaster.LastEvent != "EntityUpdated" {
		t.Errorf("Expected EntityUpdated event, got: %s", broadcaster.LastEvent)
	}

	if broadcaster.LastPayload == nil {
		t.Error("Expected payload to be broadcast")
	}
}

func TestMovement_MultipleStepsWithinSameAction(t *testing.T) {
	has := createTestMovementSystem()

	// Roll movement dice to allow movement
	_, err := has.turnManager.RollMovementDice()
	if err != nil {
		t.Fatalf("Failed to roll movement dice: %v", err)
	}

	// First step using move_before
	request1 := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 1.0,
			"dy": 0.0,
		},
	}

	result1, err := has.ProcessMovement(request1)
	if err != nil {
		t.Fatalf("First movement step should succeed, got: %v", err)
	}

	if !result1.Success {
		t.Fatalf("First movement step should be successful, got: %s", result1.Message)
	}

	// Check that hero moved to (6,5)
	pos1 := has.gameState.Entities["hero-1"]
	if pos1.X != 6 || pos1.Y != 5 {
		t.Errorf("Expected hero at (6,5) after first step, got (%d,%d)", pos1.X, pos1.Y)
	}

	// Second step using the same move_before action should succeed
	request2 := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction, // Same action type
		Parameters: map[string]any{
			"dx": 0.0,
			"dy": 1.0,
		},
	}

	result2, err := has.ProcessMovement(request2)
	if err != nil {
		t.Fatalf("Second movement step within same action should succeed, got: %v", err)
	}

	if !result2.Success {
		t.Fatalf("Second movement step should be successful, got: %s", result2.Message)
	}

	// Check that hero moved to (6,6)
	pos2 := has.gameState.Entities["hero-1"]
	if pos2.X != 6 || pos2.Y != 6 {
		t.Errorf("Expected hero at (6,6) after second step, got (%d,%d)", pos2.X, pos2.Y)
	}

	// But switching to move_after should fail
	request3 := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveAfterAction, // Different action type
		Parameters: map[string]any{
			"dx": 1.0,
			"dy": 0.0,
		},
	}

	_, err = has.ProcessMovement(request3)
	if err == nil {
		t.Fatal("Switching to move_after should fail")
	}

	if err.Error() != "player cannot move right now" {
		t.Errorf("Expected 'cannot move' error, got: %v", err)
	}
}

func TestMovement_FivePointConsumption(t *testing.T) {
	has := createTestMovementSystem()

	// Set up debug system to override dice roll to 3 (each die will be 3)
	debugSystem := has.debugSystem
	debugSystem.SetDiceOverride("movement", 3) // 2 dice × 3 = 6 points total

	// Roll movement dice to get 6 points  
	rolls, err := has.turnManager.RollMovementDice()
	if err != nil {
		t.Fatalf("Failed to roll movement dice: %v", err)
	}

	// Verify we got expected points
	initialState := has.turnManager.GetTurnState()
	t.Logf("Initial movement points after rolling: %d", initialState.MovementLeft)
	t.Logf("Dice rolls: %v", rolls)

	// Track movement consumption step by step (try all 6 points)
	for step := 1; step <= 6; step++ {
		beforeState := has.turnManager.GetTurnState()
		t.Logf("Step %d - Before move: %d points remaining", step, beforeState.MovementLeft)

		if beforeState.MovementLeft <= 0 {
			t.Logf("Step %d: No movement points left", step)
			break
		}

		// Alternate movement directions to avoid hitting boundaries
		// Start at (5,5), move in different directions
		var dx, dy float64
		switch step % 4 {
		case 1: dx, dy = 1.0, 0.0 // East
		case 2: dx, dy = 0.0, 1.0 // South
		case 3: dx, dy = -1.0, 0.0 // West
		case 0: dx, dy = 0.0, -1.0 // North
		}

		request := MovementRequest{
			PlayerID: "player-1",
			EntityID: "hero-1",
			Action:   MoveBeforeAction,
			Parameters: map[string]any{
				"dx": dx,
				"dy": dy,
			},
		}

		result, err := has.ProcessMovement(request)
		if err != nil {
			t.Logf("Step %d movement failed: %v", step, err)
			break
		}

		if !result.Success {
			t.Logf("Step %d movement unsuccessful: %s", step, result.Message)
			break
		}

		afterState := has.turnManager.GetTurnState()
		t.Logf("Step %d - After move: %d points remaining", step, afterState.MovementLeft)

		// Verify we consumed exactly 1 point
		consumed := beforeState.MovementLeft - afterState.MovementLeft
		if consumed != 1 {
			t.Errorf("Step %d: Expected to consume 1 point, actually consumed %d", step, consumed)
		}
	}

	// Check final state
	finalState := has.turnManager.GetTurnState()
	t.Logf("Final movement points: %d", finalState.MovementLeft)
}
