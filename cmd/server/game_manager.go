package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// GameManager coordinates all game systems
type GameManager struct {
	gameState       *GameState
	turnManager     *TurnManager
	heroActions     *HeroActionSystem
	monsterSystem   *MonsterSystem
	furnitureSystem *FurnitureSystem
	debugSystem     *DebugSystem
	broadcaster     Broadcaster
	logger          Logger
	sequenceGen     SequenceGenerator
	mutex           sync.RWMutex
}

// NewGameManager creates a new game manager with all systems
func NewGameManager(broadcaster Broadcaster, logger Logger, sequenceGen SequenceGenerator, debugConfig DebugConfig) (*GameManager, error) {
	// Initialize game state
	gameState, furnitureSystem, err := initializeGameStateForManager(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize game state: %w", err)
	}

	// Create turn manager
	turnManager := NewTurnManager(broadcaster, logger)

	// Create debug system
	debugSystem := NewDebugSystem(debugConfig, gameState, broadcaster, logger)

	// Create hero action system
	heroActions := NewHeroActionSystem(gameState, turnManager, broadcaster, logger, debugSystem)

	// Create dice system for monsters
	diceSystem := NewDiceSystem(debugSystem)

	// Create monster system
	monsterSystem := NewMonsterSystem(gameState, turnManager, diceSystem, broadcaster, logger)

	// Add default player (will be replaced with dynamic player loading later)
	defaultPlayer := &Player{
		ID:       "player-1",
		Name:     "Hero",
		EntityID: "hero-1",
		Class:    Barbarian,
		IsActive: true,
	}

	if err := turnManager.AddPlayer(defaultPlayer); err != nil {
		return nil, fmt.Errorf("failed to add default player: %w", err)
	}

	return &GameManager{
		gameState:       gameState,
		turnManager:     turnManager,
		heroActions:     heroActions,
		monsterSystem:   monsterSystem,
		furnitureSystem: furnitureSystem,
		debugSystem:     debugSystem,
		broadcaster:     broadcaster,
		logger:          logger,
		sequenceGen:     sequenceGen,
	}, nil
}

// ProcessHeroAction processes a hero action request
func (gm *GameManager) ProcessHeroAction(request ActionRequest) (*ActionResult, error) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return gm.heroActions.ProcessAction(request)
}

// ProcessMonsterAction processes a monster action during GameMaster turn
func (gm *GameManager) ProcessMonsterAction(request MonsterActionRequest) (*MonsterActionResult, error) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	if !gm.turnManager.IsGameMasterTurn() {
		return nil, fmt.Errorf("monster actions can only be performed during GameMaster turns")
	}

	return gm.monsterSystem.ProcessAction(request)
}

// ProcessMovement handles legacy movement requests
func (gm *GameManager) ProcessMovement(req protocol.RequestMove) error {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	// Convert legacy movement to movement action (once per turn, before or after main action)
	movementRequest := MovementRequest{
		PlayerID: "player-1", // TODO: Get from request context
		EntityID: req.EntityID,
		Action:   MoveBeforeAction, // Default to move before action
		Parameters: map[string]any{
			"dx": float64(req.DX),
			"dy": float64(req.DY),
		},
	}

	result, err := gm.heroActions.ProcessMovement(movementRequest)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("movement failed: %s", result.Message)
	}

	return nil
}

// ProcessDoorToggle handles legacy door toggle requests
func (gm *GameManager) ProcessDoorToggle(req protocol.RequestToggleDoor) error {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	// Use existing door toggle logic directly
	// TODO: Implement proper quest loading for this to work
	seqPtr := &gm.sequenceGen.(*SequenceGeneratorImpl).counter
	handleRequestToggleDoor(req, gm.gameState, gm.broadcaster.(*BroadcasterImpl).hub, seqPtr, nil, nil)
	return nil
}

// GetTurnState returns the current turn state
func (gm *GameManager) GetTurnState() TurnState {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return gm.turnManager.GetTurnState()
}

// EndTurn advances to the next turn
func (gm *GameManager) EndTurn() error {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	return gm.turnManager.EndTurn()
}

// GetGameState returns the current game state (read-only)
func (gm *GameManager) GetGameState() *GameState {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return gm.gameState
}

// GetMonsters returns all active monsters
func (gm *GameManager) GetMonsters() map[string]*Monster {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return gm.monsterSystem.GetMonsters()
}

// SpawnMonster spawns a new monster (debug/GM function)
func (gm *GameManager) SpawnMonster(monsterType MonsterType, position protocol.TileAddress) (*Monster, error) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	return gm.monsterSystem.SpawnMonster(monsterType, position)
}

// GetDebugSystem returns the debug system for HTTP handler registration
func (gm *GameManager) GetDebugSystem() *DebugSystem {
	return gm.debugSystem
}

// Shutdown gracefully shuts down the game manager
func (gm *GameManager) Shutdown() {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	gm.logger.Printf("Game manager shutting down")
	// TODO: Implement cleanup logic
}

// DebugTeleportHero teleports a hero to a specific position (debug only)
func (gm *GameManager) DebugTeleportHero(entityID string, x, y int) error {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	if entityID == "" {
		entityID = "hero-1"
	}

	// Validate coordinates
	if x < 0 || y < 0 || x >= gm.gameState.Segment.Width || y >= gm.gameState.Segment.Height {
		return fmt.Errorf("coordinates (%d,%d) out of bounds", x, y)
	}

	gm.gameState.Lock.Lock()
	oldPos := gm.gameState.Entities[entityID]
	newPos := protocol.TileAddress{
		SegmentID: oldPos.SegmentID,
		X:         x,
		Y:         y,
	}
	gm.gameState.Entities[entityID] = newPos
	gm.gameState.Lock.Unlock()

	// Broadcast entity update
	gm.broadcaster.BroadcastEvent("EntityUpdated", protocol.EntityUpdated{
		ID:   entityID,
		Tile: newPos,
	})

	gm.logger.Printf("DEBUG: Teleported %s from (%d,%d) to (%d,%d)", entityID, oldPos.X, oldPos.Y, x, y)
	return nil
}

// DebugRevealMap reveals the entire map (debug only)
func (gm *GameManager) DebugRevealMap() error {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	gm.gameState.Lock.Lock()
	// Reveal all regions
	for i := 0; i < gm.gameState.RegionMap.RegionsCount; i++ {
		gm.gameState.RevealedRegions[i] = true
		gm.gameState.KnownRegions[i] = true
	}

	// Reveal all doors
	for doorID := range gm.gameState.Doors {
		gm.gameState.KnownDoors[doorID] = true
	}
	gm.gameState.Lock.Unlock()

	// Get all region IDs
	allRegions := make([]int, gm.gameState.RegionMap.RegionsCount)
	for i := 0; i < gm.gameState.RegionMap.RegionsCount; i++ {
		allRegions[i] = i
	}

	// Broadcast updates
	gm.broadcaster.BroadcastEvent("RegionsRevealed", protocol.RegionsRevealed{IDs: allRegions})
	gm.broadcaster.BroadcastEvent("RegionsKnown", protocol.RegionsKnown{IDs: allRegions})

	// Create door list
	var doors []protocol.ThresholdLite
	for id, info := range gm.gameState.Doors {
		doors = append(doors, protocol.ThresholdLite{
			ID:          id,
			X:           info.Edge.X,
			Y:           info.Edge.Y,
			Orientation: string(info.Edge.Orientation),
			Kind:        "DoorSocket",
			State:       info.State,
		})
	}
	gm.broadcaster.BroadcastEvent("DoorsVisible", protocol.DoorsVisible{Doors: doors})

	gm.logger.Printf("DEBUG: Revealed entire map (%d regions, %d doors)", len(allRegions), len(doors))
	return nil
}

// Helper function to initialize game state using existing initialization logic
func initializeGameStateForManager(logger Logger) (*GameState, *FurnitureSystem, error) {
	// Use existing initialization logic
	board, quest, err := loadGameContent()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load game content: %w", err)
	}
	furnitureSystem := NewFurnitureSystem(log.New(os.Stdout, "", log.LstdFlags))
	state, _, err := initializeGameState(board, quest, furnitureSystem)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize game state: %w", err)
	}

	logger.Printf("Game state initialized with board, quest, and furniture data")
	return state, furnitureSystem, nil
}

// GetVisibleMonsters returns all visible monsters
func (gm *GameManager) GetVisibleMonsters() []*Monster {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return gm.monsterSystem.GetVisibleMonsters()
}

// GetFurnitureForSnapshot returns all furniture in the format needed for client snapshot
func (gm *GameManager) GetFurnitureForSnapshot() []protocol.FurnitureLite {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	instances := gm.furnitureSystem.GetAllInstances()
	gm.logger.Printf("DEBUG: GetFurnitureForSnapshot called - found %d furniture instances", len(instances))
	furniture := make([]protocol.FurnitureLite, 0, len(instances))

	for _, instance := range instances {
		if instance.Definition == nil {
			gm.logger.Printf("Warning: Furniture instance %s has no definition", instance.ID)
			continue
		}

		furnitureItem := protocol.FurnitureLite{
			ID:   instance.ID,
			Type: instance.Type,
			Tile: instance.Position,
			GridSize: struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			}{
				Width:  instance.Definition.GridSize.Width,
				Height: instance.Definition.GridSize.Height,
			},
			TileImage:        instance.Definition.Rendering.TileImage,
			TileImageCleaned: instance.Definition.Rendering.TileImageCleaned,
			PixelDimensions: struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			}{
				Width:  instance.Definition.Rendering.PixelDimensions.Width,
				Height: instance.Definition.Rendering.PixelDimensions.Height,
			},
			BlocksLineOfSight: instance.Definition.BlocksLineOfSight,
			BlocksMovement:    instance.Definition.BlocksMovement,
			Contains:          instance.Contains,
		}

		furniture = append(furniture, furnitureItem)
		gm.logger.Printf("DEBUG: Added furniture item to snapshot: %s (%s) at (%d,%d)",
			instance.ID, instance.Type, instance.Position.X, instance.Position.Y)
	}

	gm.logger.Printf("DEBUG: Returning %d furniture items for snapshot", len(furniture))
	return furniture
}
