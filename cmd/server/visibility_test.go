package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

func TestVisibilityCalculator_ComputeVisibleRegions(t *testing.T) {
	// Arrange
	state := createTestGameStateWithDoors()
	logger := &MockLogger{}
	calculator := NewVisibilityCalculator(logger)

	from := protocol.TileAddress{X: 5, Y: 5}

	// Act
	result := calculator.ComputeVisibleRegions(state, from, 0)

	// Assert
	if len(result) == 0 {
		t.Error("Expected some visible regions, but got none")
	}
}

func TestIsEdgeVisible_DirectPath(t *testing.T) {
	// Arrange
	state := createSimpleTestState()

	// Act - test direct horizontal line of sight
	result := isEdgeVisible(state, 1, 1, geometry.EdgeAddress{X: 3, Y: 1, Orientation: geometry.Vertical})

	// Assert
	if !result {
		t.Error("Expected direct horizontal path to be visible")
	}
}

func TestIsEdgeVisible_BlockedByWall(t *testing.T) {
	// Arrange
	state := createSimpleTestState()
	// Add a blocking wall
	state.BlockedWalls[geometry.EdgeAddress{X: 2, Y: 1, Orientation: geometry.Vertical}] = true

	// Act
	result := isEdgeVisible(state, 1, 1, geometry.EdgeAddress{X: 3, Y: 1, Orientation: geometry.Vertical})

	// Assert
	if result {
		t.Error("Expected path blocked by wall to not be visible")
	}
}

func TestIsEdgeVisible_BlockedByClosedDoor(t *testing.T) {
	// Arrange
	state := createSimpleTestState()
	// Add a closed door
	edge := geometry.EdgeAddress{X: 2, Y: 1, Orientation: geometry.Vertical}
	state.DoorByEdge[edge] = "door1"
	state.Doors["door1"] = &DoorInfo{
		Edge:  edge,
		State: "closed",
	}

	// Act
	result := isEdgeVisible(state, 1, 1, geometry.EdgeAddress{X: 3, Y: 1, Orientation: geometry.Vertical})

	// Assert
	if result {
		t.Error("Expected path blocked by closed door to not be visible")
	}
}

func TestIsEdgeVisible_ThroughOpenDoor(t *testing.T) {
	// Arrange
	state := createSimpleTestState()
	// Add an open door
	edge := geometry.EdgeAddress{X: 2, Y: 1, Orientation: geometry.Vertical}
	state.DoorByEdge[edge] = "door1"
	state.Doors["door1"] = &DoorInfo{
		Edge:  edge,
		State: "open",
	}

	// Act
	result := isEdgeVisible(state, 1, 1, geometry.EdgeAddress{X: 3, Y: 1, Orientation: geometry.Vertical})

	// Assert
	if !result {
		t.Error("Expected path through open door to be visible")
	}
}

func TestIsTileCenterVisible_SameTile(t *testing.T) {
	// Arrange
	state := createSimpleTestState()

	// Act
	result := isTileCenterVisible(state, 1, 1, 1, 1)

	// Assert
	if !result {
		t.Error("Expected same tile to be visible")
	}
}

func TestIsTileCenterVisible_DiagonalPath(t *testing.T) {
	// Arrange
	state := createSimpleTestState()

	// Act
	result := isTileCenterVisible(state, 1, 1, 3, 3)

	// Assert
	if !result {
		t.Error("Expected diagonal path to be visible in empty space")
	}
}

func createSimpleTestState() *GameState {
	segment := geometry.Segment{
		ID:     "test-segment",
		Width:  10,
		Height: 10,
	}

	state := &GameState{
		Segment:            segment,
		BlockedWalls:       make(map[geometry.EdgeAddress]bool),
		BlockedTiles:       make(map[protocol.TileAddress]bool),
		Doors:              make(map[string]*DoorInfo),
		DoorByEdge:         make(map[geometry.EdgeAddress]string),
		Entities:           make(map[string]protocol.TileAddress),
		RevealedRegions:    make(map[int]bool),
		KnownRegions:       make(map[int]bool),
		KnownDoors:         make(map[string]bool),
		KnownBlockingWalls: make(map[string]bool),
		CorridorRegion:     0,
	}

	return state
}

func createTestGameStateWithDoors() *GameState {
	state := createSimpleTestState()

	// Add some doors for visibility testing
	door1Edge := geometry.EdgeAddress{X: 3, Y: 5, Orientation: geometry.Vertical}
	state.Doors["door1"] = &DoorInfo{
		Edge:    door1Edge,
		RegionA: 1,
		RegionB: 2,
		State:   "closed",
	}
	state.DoorByEdge[door1Edge] = "door1"

	door2Edge := geometry.EdgeAddress{X: 7, Y: 5, Orientation: geometry.Vertical}
	state.Doors["door2"] = &DoorInfo{
		Edge:    door2Edge,
		RegionA: 2,
		RegionB: 3,
		State:   "open",
	}
	state.DoorByEdge[door2Edge] = "door2"

	return state
}

// Benchmark tests for visibility calculations
func BenchmarkIsEdgeVisible_DirectPath(b *testing.B) {
	state := createSimpleTestState()
	target := geometry.EdgeAddress{X: 9, Y: 9, Orientation: geometry.Vertical}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isEdgeVisible(state, 0, 0, target)
	}
}

func BenchmarkIsEdgeVisible_ComplexPath(b *testing.B) {
	state := createSimpleTestState()
	// Add some walls to make pathfinding more complex
	state.BlockedWalls[geometry.EdgeAddress{X: 2, Y: 2, Orientation: geometry.Vertical}] = true
	state.BlockedWalls[geometry.EdgeAddress{X: 4, Y: 4, Orientation: geometry.Horizontal}] = true
	state.BlockedWalls[geometry.EdgeAddress{X: 6, Y: 6, Orientation: geometry.Vertical}] = true

	target := geometry.EdgeAddress{X: 9, Y: 9, Orientation: geometry.Vertical}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isEdgeVisible(state, 0, 0, target)
	}
}

func BenchmarkIsTileCenterVisible_LongDistance(b *testing.B) {
	state := createSimpleTestState()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isTileCenterVisible(state, 0, 0, 9, 9)
	}
}

func BenchmarkComputeVisibleRoomRegionsNow(b *testing.B) {
	state := createTestGameStateWithDoors()
	from := protocol.TileAddress{X: 5, Y: 5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		computeVisibleRoomRegionsNow(state, from, 0)
	}
}
