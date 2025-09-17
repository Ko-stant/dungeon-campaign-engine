package main

import (
	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// Broadcaster interface for WebSocket communication
type Broadcaster interface {
	BroadcastEvent(eventType string, payload interface{})
}

// Logger interface for logging abstraction
type Logger interface {
	Printf(format string, v ...interface{})
}

// SequenceGenerator interface for sequence number generation
type SequenceGenerator interface {
	Next() uint64
}

// GameEngine interface for core game logic
type GameEngine interface {
	ProcessMove(req protocol.RequestMove) (*MoveResult, error)
	ProcessDoorToggle(req protocol.RequestToggleDoor) (*DoorToggleResult, error)
	GetState() *GameState
}

// MoveResult contains the results of a move operation
type MoveResult struct {
	EntityUpdated         *protocol.EntityUpdated
	VisibleRegions        []int
	NewlyKnownRegions     []int
	NewlyVisibleDoors     []protocol.ThresholdLite
	NewlyVisibleWalls     []protocol.BlockingWallLite
	RegionsToReveal       []int
}

// DoorToggleResult contains the results of a door toggle operation
type DoorToggleResult struct {
	StateChange           *protocol.DoorStateChanged
	RegionsToReveal       []int
	VisibleRegions        []int
	NewlyKnownRegions     []int
	NewlyVisibleDoors     []protocol.ThresholdLite
	NewlyVisibleWalls     []protocol.BlockingWallLite
}

// VisibilityCalculator interface for visibility calculations
type VisibilityCalculator interface {
	ComputeVisibleRegions(state *GameState, from protocol.TileAddress, corridorRegion int) []int
	CheckNewlyVisibleDoors(state *GameState, hero protocol.TileAddress) []protocol.ThresholdLite
	GetVisibleBlockingWalls(state *GameState, hero protocol.TileAddress, quest *geometry.QuestDefinition) ([]protocol.BlockingWallLite, []protocol.BlockingWallLite)
	IsEdgeVisible(state *GameState, fromX, fromY int, target geometry.EdgeAddress) bool
	IsTileCenterVisible(state *GameState, fromX, fromY, toX, toY int) bool
}

// MovementValidator interface for movement validation
type MovementValidator interface {
	ValidateMove(state *GameState, entityID string, dx, dy int) (*protocol.TileAddress, error)
}

// StateManager interface for game state operations
type StateManager interface {
	GetEntity(entityID string) (protocol.TileAddress, bool)
	UpdateEntity(entityID string, tile protocol.TileAddress)
	AddKnownRegions(regions []int) []int
	ToggleDoor(doorID string) (*DoorInfo, []int, error)
	Lock()
	Unlock()
}