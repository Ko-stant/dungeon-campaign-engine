package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// Mock implementations for testing
type MockLogger struct {
	messages []string
}

func (m *MockLogger) Printf(format string, v ...interface{}) {
	// Store messages for verification in tests
	m.messages = append(m.messages, format)
}

type MockVisibilityCalculator struct {
	visibleRegions    []int
	newlyVisibleDoors []protocol.ThresholdLite
	newlyVisibleWalls []protocol.BlockingWallLite
}

func (m *MockVisibilityCalculator) ComputeVisibleRegions(state *GameState, from protocol.TileAddress, corridorRegion int) []int {
	return m.visibleRegions
}

func (m *MockVisibilityCalculator) CheckNewlyVisibleDoors(state *GameState, hero protocol.TileAddress) []protocol.ThresholdLite {
	return m.newlyVisibleDoors
}

func (m *MockVisibilityCalculator) GetVisibleBlockingWalls(state *GameState, hero protocol.TileAddress, quest *geometry.QuestDefinition) ([]protocol.BlockingWallLite, []protocol.BlockingWallLite) {
	return m.newlyVisibleWalls, m.newlyVisibleWalls
}

func (m *MockVisibilityCalculator) IsEdgeVisible(state *GameState, fromX, fromY int, target geometry.EdgeAddress) bool {
	return true
}

func (m *MockVisibilityCalculator) IsTileCenterVisible(state *GameState, fromX, fromY, toX, toY int) bool {
	return true
}

type MockMovementValidator struct {
	shouldFail bool
	newTile    *protocol.TileAddress
}

func (m *MockMovementValidator) ValidateMove(state *GameState, entityID string, dx, dy int) (*protocol.TileAddress, error) {
	if m.shouldFail {
		return nil, &MovementError{Reason: "test error"}
	}
	if m.newTile != nil {
		return m.newTile, nil
	}
	// Default behavior - just move the entity
	tile := state.Entities[entityID]
	newTile := protocol.TileAddress{
		SegmentID: tile.SegmentID,
		X:         tile.X + dx,
		Y:         tile.Y + dy,
	}
	return &newTile, nil
}

type MovementError struct {
	Reason string
}

func (e *MovementError) Error() string {
	return e.Reason
}

// Helper function to create test game state
func createTestGameState() *GameState {
	segment := geometry.Segment{
		ID:     "test-segment",
		Width:  10,
		Height: 10,
	}
	regionMap := geometry.RegionMap{
		RegionsCount:    5,
		TileRegionIDs:   make([]int, 100), // 10x10 grid
	}

	state := &GameState{
		Segment:         segment,
		RegionMap:       regionMap,
		BlockedWalls:    make(map[geometry.EdgeAddress]bool),
		BlockedTiles:    make(map[protocol.TileAddress]bool),
		Doors:           make(map[string]*DoorInfo),
		DoorByEdge:      make(map[geometry.EdgeAddress]string),
		Entities:        make(map[string]protocol.TileAddress),
		RevealedRegions: make(map[int]bool),
		KnownRegions:    make(map[int]bool),
		KnownDoors:      make(map[string]bool),
		KnownBlockingWalls: make(map[string]bool),
		CorridorRegion:  0,
	}

	// Add test entity
	state.Entities["test-hero"] = protocol.TileAddress{
		SegmentID: "test-segment",
		X:         5,
		Y:         5,
	}

	return state
}

func TestGameEngine_ProcessMove_Success(t *testing.T) {
	// Arrange
	state := createTestGameState()
	logger := &MockLogger{}
	visibility := &MockVisibilityCalculator{
		visibleRegions: []int{1, 2, 3},
		newlyVisibleDoors: []protocol.ThresholdLite{
			{ID: "door1", X: 6, Y: 5, Orientation: "vertical"},
		},
	}
	validator := &MockMovementValidator{}
	quest := &geometry.QuestDefinition{}

	engine := NewGameEngine(state, visibility, validator, logger, quest)

	req := protocol.RequestMove{
		EntityID: "test-hero",
		DX:       1,
		DY:       0,
	}

	// Act
	result, err := engine.ProcessMove(req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.EntityUpdated == nil {
		t.Fatal("Expected EntityUpdated to be set")
	}

	if result.EntityUpdated.ID != "test-hero" {
		t.Errorf("Expected entity ID to be 'test-hero', got: %s", result.EntityUpdated.ID)
	}

	if result.EntityUpdated.Tile.X != 6 {
		t.Errorf("Expected X to be 6, got: %d", result.EntityUpdated.Tile.X)
	}

	if len(result.VisibleRegions) != 3 {
		t.Errorf("Expected 3 visible regions, got: %d", len(result.VisibleRegions))
	}

	if len(result.NewlyVisibleDoors) != 1 {
		t.Errorf("Expected 1 newly visible door, got: %d", len(result.NewlyVisibleDoors))
	}
}

func TestGameEngine_ProcessMove_InvalidDirection(t *testing.T) {
	// Arrange
	state := createTestGameState()
	logger := &MockLogger{}
	visibility := &MockVisibilityCalculator{}
	validator := &MockMovementValidator{}
	quest := &geometry.QuestDefinition{}

	engine := NewGameEngine(state, visibility, validator, logger, quest)

	req := protocol.RequestMove{
		EntityID: "test-hero",
		DX:       2, // Invalid - too large
		DY:       0,
	}

	// Act
	result, err := engine.ProcessMove(req)

	// Assert
	if err == nil {
		t.Fatal("Expected error for invalid move direction")
	}

	if result != nil {
		t.Error("Expected result to be nil on error")
	}
}

func TestGameEngine_ProcessMove_ValidationFailure(t *testing.T) {
	// Arrange
	state := createTestGameState()
	logger := &MockLogger{}
	visibility := &MockVisibilityCalculator{}
	validator := &MockMovementValidator{shouldFail: true}
	quest := &geometry.QuestDefinition{}

	engine := NewGameEngine(state, visibility, validator, logger, quest)

	req := protocol.RequestMove{
		EntityID: "test-hero",
		DX:       1,
		DY:       0,
	}

	// Act
	result, err := engine.ProcessMove(req)

	// Assert
	if err == nil {
		t.Fatal("Expected error from movement validator")
	}

	if result != nil {
		t.Error("Expected result to be nil on validation error")
	}
}

func TestMovementValidator_ValidateMove_Success(t *testing.T) {
	// Arrange
	state := createTestGameState()
	logger := &MockLogger{}
	validator := NewMovementValidator(logger)

	// Act
	result, err := validator.ValidateMove(state, "test-hero", 1, 0)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.X != 6 {
		t.Errorf("Expected X to be 6, got: %d", result.X)
	}

	if result.Y != 5 {
		t.Errorf("Expected Y to be 5, got: %d", result.Y)
	}
}

func TestMovementValidator_ValidateMove_OutOfBounds(t *testing.T) {
	// Arrange
	state := createTestGameState()
	logger := &MockLogger{}
	validator := NewMovementValidator(logger)

	// Act - try to move outside bounds
	result, err := validator.ValidateMove(state, "test-hero", 10, 0)

	// Assert
	if err == nil {
		t.Fatal("Expected error for out of bounds movement")
	}

	if result != nil {
		t.Error("Expected result to be nil on bounds error")
	}

	// Check that logger was called
	if len(logger.messages) == 0 {
		t.Error("Expected logger to be called for bounds check failure")
	}
}

func TestMovementValidator_ValidateMove_EntityNotFound(t *testing.T) {
	// Arrange
	state := createTestGameState()
	logger := &MockLogger{}
	validator := NewMovementValidator(logger)

	// Act
	result, err := validator.ValidateMove(state, "nonexistent", 1, 0)

	// Assert
	if err == nil {
		t.Fatal("Expected error for nonexistent entity")
	}

	if result != nil {
		t.Error("Expected result to be nil for nonexistent entity")
	}
}

// Benchmark tests for performance profiling
func BenchmarkGameEngine_ProcessMove(b *testing.B) {
	state := createTestGameState()
	logger := &MockLogger{}
	visibility := &MockVisibilityCalculator{
		visibleRegions: []int{1, 2, 3},
	}
	validator := &MockMovementValidator{}
	quest := &geometry.QuestDefinition{}

	engine := NewGameEngine(state, visibility, validator, logger, quest)

	req := protocol.RequestMove{
		EntityID: "test-hero",
		DX:       1,
		DY:       0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset position for each iteration
		state.Entities["test-hero"] = protocol.TileAddress{
			SegmentID: "test-segment",
			X:         5,
			Y:         5,
		}

		_, err := engine.ProcessMove(req)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkMovementValidator_ValidateMove(b *testing.B) {
	state := createTestGameState()
	logger := &MockLogger{}
	validator := NewMovementValidator(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateMove(state, "test-hero", 1, 0)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}