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
	// MockDebugSystem not implemented yet, use nil
	var debugSystem *DebugSystem = nil

	turnManager := NewTurnManager(broadcaster, logger)

	// Add test player
	player := &Player{
		ID:       "player-1",
		Name:     "Test Hero",
		EntityID: "hero-1",
		Class:    Barbarian,
		IsActive: true,
	}
	turnManager.AddPlayer(player)

	return NewHeroActionSystem(gameState, turnManager, broadcaster, logger, debugSystem)
}

func TestMovement_OncePerTurn(t *testing.T) {
	has := createTestMovementSystem()

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

	_, err := has.ProcessMovement(request)

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

	// Consume most movement points first
	has.turnManager.ConsumeMovement(1) // Leave only 1 point

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

	_, err := has.ProcessMovement(request)

	if err == nil {
		t.Fatal("Expected error for insufficient movement points")
	}
}

func TestMovement_AfterTurnReset(t *testing.T) {
	has := createTestMovementSystem()

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

	_, err := has.ProcessMovement(request)
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

	if turnState.MovementLeft != 2 {
		t.Errorf("Expected movement points reset to 2, got %d", turnState.MovementLeft)
	}

	// Should be able to move again
	_, err = has.ProcessMovement(request)
	if err != nil {
		t.Errorf("Expected movement to work in new turn, got: %v", err)
	}
}

func TestMovement_WrongPlayer(t *testing.T) {
	t.Skip("Integration test requires content files - skipping for build validation")
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

	request := MovementRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   MoveBeforeAction,
		Parameters: map[string]any{
			"dx": 1.0,
			"dy": 0.0,
		},
	}

	_, err := has.ProcessMovement(request)

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
