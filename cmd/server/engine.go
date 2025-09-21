package main

import (
	"errors"
	"fmt"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// GameEngineImpl implements the GameEngine interface
type GameEngineImpl struct {
	state      *GameState
	visibility VisibilityCalculator
	validator  MovementValidator
	logger     Logger
	quest      *geometry.QuestDefinition
}

// NewGameEngine creates a new game engine with dependencies
func NewGameEngine(state *GameState, visibility VisibilityCalculator, validator MovementValidator, logger Logger, quest *geometry.QuestDefinition) *GameEngineImpl {
	return &GameEngineImpl{
		state:      state,
		visibility: visibility,
		validator:  validator,
		logger:     logger,
		quest:      quest,
	}
}

func (e *GameEngineImpl) GetState() *GameState {
	return e.state
}

func (e *GameEngineImpl) ProcessMove(req protocol.RequestMove) (*MoveResult, error) {
	// Validate move request
	if (req.DX != 0 && req.DY != 0) || req.DX < -1 || req.DX > 1 || req.DY < -1 || req.DY > 1 {
		return nil, errors.New("invalid move direction")
	}
	if req.DX == 0 && req.DY == 0 {
		return nil, errors.New("no movement specified")
	}

	// Validate and execute movement
	newTile, err := e.validator.ValidateMove(e.state, req.EntityID, req.DX, req.DY)
	if err != nil {
		return nil, fmt.Errorf("movement validation failed: %w", err)
	}

	// Update entity position
	e.state.Lock.Lock()
	e.state.Entities[req.EntityID] = *newTile
	e.state.Lock.Unlock()

	// Calculate visibility updates
	hero := e.state.Entities[req.EntityID]
	visibleRegions := e.visibility.ComputeVisibleRegions(e.state, hero, e.state.CorridorRegion)

	e.state.Lock.Lock()
	newlyKnownRegions := addKnownRegions(e.state, visibleRegions)
	e.state.Lock.Unlock()

	// Check for newly visible elements
	newlyVisibleDoors := e.visibility.CheckNewlyVisibleDoors(e.state, hero)
	_, newlyVisibleWalls := e.visibility.GetVisibleBlockingWalls(e.state, hero, e.quest)

	e.logger.Printf("visibleNow (hero @ %d,%d): %v", hero.X, hero.Y, visibleRegions)
	if len(newlyVisibleDoors) > 0 {
		e.logger.Printf("sending %d newly visible doors to client", len(newlyVisibleDoors))
	}
	if len(newlyVisibleWalls) > 0 {
		e.logger.Printf("sending %d newly visible blocking walls to client", len(newlyVisibleWalls))
	}

	return &MoveResult{
		EntityUpdated:     &protocol.EntityUpdated{ID: req.EntityID, Tile: *newTile},
		VisibleRegions:    visibleRegions,
		NewlyKnownRegions: newlyKnownRegions,
		NewlyVisibleDoors: newlyVisibleDoors,
		NewlyVisibleWalls: newlyVisibleWalls,
	}, nil
}

func (e *GameEngineImpl) ProcessDoorToggle(req protocol.RequestToggleDoor) (*DoorToggleResult, error) {
	e.state.Lock.Lock()
	info, ok := e.state.Doors[req.ThresholdID]
	if !ok || info == nil || info.State == "open" {
		e.state.Lock.Unlock()
		return nil, errors.New("door cannot be opened")
	}

	info.State = "open"

	// Calculate regions to reveal
	var toReveal []int
	a, b := info.RegionA, info.RegionB
	if e.state.RevealedRegions[a] && !e.state.RevealedRegions[b] {
		e.state.RevealedRegions[b] = true
		toReveal = append(toReveal, b)
	} else if e.state.RevealedRegions[b] && !e.state.RevealedRegions[a] {
		e.state.RevealedRegions[a] = true
		toReveal = append(toReveal, a)
	}
	e.state.Lock.Unlock()

	// Calculate visibility updates
	hero := e.state.Entities["hero-1"] // TODO: Make this configurable
	visibleRegions := e.visibility.ComputeVisibleRegions(e.state, hero, e.state.CorridorRegion)

	e.state.Lock.Lock()
	newlyKnownRegions := addKnownRegions(e.state, visibleRegions)
	e.state.Lock.Unlock()

	// Check for newly visible elements
	newlyVisibleDoors := e.visibility.CheckNewlyVisibleDoors(e.state, hero)
	_, newlyVisibleWalls := e.visibility.GetVisibleBlockingWalls(e.state, hero, e.quest)

	return &DoorToggleResult{
		StateChange:       &protocol.DoorStateChanged{ThresholdID: req.ThresholdID, State: "open"},
		RegionsToReveal:   toReveal,
		VisibleRegions:    visibleRegions,
		NewlyKnownRegions: newlyKnownRegions,
		NewlyVisibleDoors: newlyVisibleDoors,
		NewlyVisibleWalls: newlyVisibleWalls,
	}, nil
}

// MovementValidatorImpl implements MovementValidator
type MovementValidatorImpl struct {
	logger          Logger
	monsterSystem   *MonsterSystem
	furnitureSystem *FurnitureSystem
}

func NewMovementValidator(logger Logger) *MovementValidatorImpl {
	return &MovementValidatorImpl{logger: logger}
}

func NewMovementValidatorWithSystems(logger Logger, monsterSystem *MonsterSystem, furnitureSystem *FurnitureSystem) *MovementValidatorImpl {
	return &MovementValidatorImpl{
		logger:          logger,
		monsterSystem:   monsterSystem,
		furnitureSystem: furnitureSystem,
	}
}

func (mv *MovementValidatorImpl) ValidateMove(state *GameState, entityID string, dx, dy int) (*protocol.TileAddress, error) {
	tile, ok := state.Entities[entityID]
	if !ok {
		return nil, errors.New("entity not found")
	}

	nx := tile.X + dx
	ny := tile.Y + dy

	// Bounds check
	if nx < 0 || ny < 0 || nx >= state.Segment.Width || ny >= state.Segment.Height {
		mv.logger.Printf("DEBUG: Movement blocked by bounds check: from (%d,%d) to (%d,%d), bounds: %dx%d",
			tile.X, tile.Y, nx, ny, state.Segment.Width, state.Segment.Height)
		return nil, errors.New("movement out of bounds")
	}

	// Check blocked tiles
	destTile := protocol.TileAddress{X: nx, Y: ny}
	if state.BlockedTiles[destTile] {
		mv.logger.Printf("DEBUG: Movement blocked by blocking wall tile: from (%d,%d) to (%d,%d)",
			tile.X, tile.Y, nx, ny)
		return nil, errors.New("destination tile blocked")
	}

	// Check if destination tile is blocked by furniture
	if mv.furnitureSystem != nil && mv.furnitureSystem.BlocksMovement(nx, ny) {
		mv.logger.Printf("DEBUG: Movement blocked by furniture: from (%d,%d) to (%d,%d)",
			tile.X, tile.Y, nx, ny)
		return nil, errors.New("furniture blocks movement")
	}

	// Check if destination tile is blocked by a monster (heroes cannot move onto monster tiles)
	if mv.monsterSystem != nil && mv.monsterSystem.IsMonsterAt(nx, ny) {
		mv.logger.Printf("DEBUG: Movement blocked by monster: from (%d,%d) to (%d,%d)",
			tile.X, tile.Y, nx, ny)
		return nil, errors.New("monster blocks movement")
	}

	// Check walls and doors
	edge := edgeForStep(tile.X, tile.Y, dx, dy)
	if state.BlockedWalls[edge] {
		mv.logger.Printf("DEBUG: Movement blocked by wall: from (%d,%d) to (%d,%d), blocked edge: %+v",
			tile.X, tile.Y, nx, ny, edge)
		return nil, errors.New("wall blocks movement")
	}

	if id, ok := state.DoorByEdge[edge]; ok {
		if d := state.Doors[id]; d != nil && d.State != "open" {
			return nil, errors.New("closed door blocks movement")
		}
	}

	newTile := protocol.TileAddress{
		SegmentID: tile.SegmentID,
		X:         nx,
		Y:         ny,
	}

	return &newTile, nil
}