package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// GameManager coordinates all game systems
type GameManager struct {
	gameState        *GameState
	turnManager      *TurnManager
	turnStateManager *TurnStateManager
	contentManager   *ContentManager
	inventoryManager *InventoryManager
	treasureDeck     *TreasureDeckManager
	treasureResolver *TreasureResolver
	heroActions      *HeroActionSystem
	monsterSystem    *MonsterSystem
	furnitureSystem  *FurnitureSystem
	debugSystem      *DebugSystem
	broadcaster      Broadcaster
	logger           Logger
	sequenceGen      SequenceGenerator
	mutex            sync.RWMutex
}

// NewGameManager creates a new game manager with all systems
func NewGameManager(broadcaster Broadcaster, logger Logger, sequenceGen SequenceGenerator, debugConfig DebugConfig) (*GameManager, error) {
	// Initialize game state
	gameState, furnitureSystem, quest, err := initializeGameStateForManager(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize game state: %w", err)
	}

	return createGameManager(gameState, furnitureSystem, quest, broadcaster, logger, sequenceGen, debugConfig)
}

// NewGameManagerWithFurniture creates a new game manager with pre-loaded furniture system
func NewGameManagerWithFurniture(broadcaster Broadcaster, logger Logger, sequenceGen SequenceGenerator, debugConfig DebugConfig, furnitureSystem *FurnitureSystem, quest *geometry.QuestDefinition) (*GameManager, error) {
	// Initialize game state using the provided furniture system
	board, err := geometry.LoadBoardFromFile("content/board.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load board: %w", err)
	}

	gameState, _, err := initializeGameState(board, quest, furnitureSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize game state: %w", err)
	}

	return createGameManager(gameState, furnitureSystem, quest, broadcaster, logger, sequenceGen, debugConfig)
}

// createGameManager is a helper function to create the GameManager with all systems
func createGameManager(gameState *GameState, furnitureSystem *FurnitureSystem, quest *geometry.QuestDefinition, broadcaster Broadcaster, logger Logger, sequenceGen SequenceGenerator, debugConfig DebugConfig) (*GameManager, error) {
	// Create debug system first
	debugSystem := NewDebugSystem(debugConfig, gameState, broadcaster, logger)

	// Create dice system
	diceSystem := NewDiceSystem(debugSystem)

	// Create turn manager with dice system
	turnManager := NewTurnManager(broadcaster, logger, diceSystem)

	// Create turn state manager for per-hero turn tracking
	turnStateManager := NewTurnStateManager(logger)

	// Wire up turn state manager to turn manager
	turnManager.SetTurnStateManager(turnStateManager)

	// Create content manager and load campaign content
	contentManager := NewContentManager(logger)
	if err := contentManager.LoadCampaign("base"); err != nil {
		return nil, fmt.Errorf("failed to load campaign content: %w", err)
	}

	// Create inventory manager
	inventoryManager := NewInventoryManager(contentManager, logger)

	// Create treasure deck and resolver
	treasureDeck := NewTreasureDeckManager(contentManager, logger)
	if err := treasureDeck.InitializeDeck(); err != nil {
		return nil, fmt.Errorf("failed to initialize treasure deck: %w", err)
	}
	treasureResolver := NewTreasureResolver(contentManager, treasureDeck, quest, logger)

	// Create hero action system
	heroActions := NewHeroActionSystem(gameState, turnManager, broadcaster, logger, debugSystem)

	// Create monster system
	monsterSystem := NewMonsterSystem(gameState, turnManager, diceSystem, broadcaster, logger)

	// Update hero action system with complete movement validator and monster system
	movementValidator := NewMovementValidatorWithSystems(logger, monsterSystem, furnitureSystem)
	heroActions.SetMovementValidator(movementValidator)
	heroActions.SetMonsterSystem(monsterSystem)
	heroActions.SetQuest(quest)
	heroActions.SetTurnStateManager(turnStateManager)

	// Add default player (will be replaced with dynamic player loading later)
	defaultPlayer := NewPlayer("player-1", "Hero", "hero-1", Barbarian)

	if err := turnManager.AddPlayer(defaultPlayer); err != nil {
		return nil, fmt.Errorf("failed to add default player: %w", err)
	}

	// Initialize inventory for hero
	if err := inventoryManager.InitializeHeroInventory("hero-1"); err != nil {
		return nil, fmt.Errorf("failed to initialize hero inventory: %w", err)
	}

	return &GameManager{
		gameState:        gameState,
		turnManager:      turnManager,
		turnStateManager: turnStateManager,
		contentManager:   contentManager,
		inventoryManager: inventoryManager,
		treasureDeck:     treasureDeck,
		treasureResolver: treasureResolver,
		heroActions:      heroActions,
		monsterSystem:    monsterSystem,
		furnitureSystem:  furnitureSystem,
		debugSystem:      debugSystem,
		broadcaster:      broadcaster,
		logger:           logger,
		sequenceGen:      sequenceGen,
	}, nil
}

// ProcessHeroAction processes a hero action request
func (gm *GameManager) ProcessHeroAction(request ActionRequest) (*ActionResult, error) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return gm.heroActions.ProcessAction(request)
}

// ProcessInstantAction processes an instant action (doesn't consume main action)
func (gm *GameManager) ProcessInstantAction(request InstantActionRequest) (*ActionResult, error) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return gm.heroActions.ProcessInstantAction(request)
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

// ProcessMovementRequest handles new turn-based movement requests
func (gm *GameManager) ProcessMovementRequest(req MovementRequest) (*ActionResult, error) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return gm.heroActions.ProcessMovement(req)
}

// ProcessDoorToggle handles legacy door toggle requests
func (gm *GameManager) ProcessDoorToggle(req protocol.RequestToggleDoor) error {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	// Load quest data for door toggle processing
	_, quest, err := loadGameContent()
	if err != nil {
		gm.logger.Printf("Failed to load quest data for door toggle: %v", err)
		// Continue without quest data - better than crashing
		quest = nil
	}

	// Ensure we have valid parameters before calling legacy handler
	if gm.gameState == nil || gm.broadcaster == nil || gm.furnitureSystem == nil || gm.monsterSystem == nil {
		return fmt.Errorf("game manager not properly initialized")
	}

	// Use existing door toggle logic directly
	seqPtr := &gm.sequenceGen.(*SequenceGeneratorImpl).counter
	handleRequestToggleDoor(req, gm.gameState, gm.broadcaster.(*BroadcasterImpl).hub, seqPtr, quest, gm.furnitureSystem, gm.monsterSystem)
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

// PassGMTurn skips the current GM turn and advances to the next hero turn (debug function)
func (gm *GameManager) PassGMTurn() error {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	return gm.turnManager.PassGMTurn()
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

// GetTurnStateManager returns the turn state manager
func (gm *GameManager) GetTurnStateManager() *TurnStateManager {
	return gm.turnStateManager
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
func initializeGameStateForManager(logger Logger) (*GameState, *FurnitureSystem, *geometry.QuestDefinition, error) {
	// Use existing initialization logic
	board, quest, err := loadGameContent()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load game content: %w", err)
	}
	furnitureSystem := NewFurnitureSystem(log.New(os.Stdout, "", log.LstdFlags))
	state, _, err := initializeGameState(board, quest, furnitureSystem)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize game state: %w", err)
	}

	logger.Printf("Game state initialized with board, quest, and furniture data")
	return state, furnitureSystem, quest, nil
}

// GetVisibleMonsters returns all visible monsters
func (gm *GameManager) GetVisibleMonsters() []*Monster {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return gm.monsterSystem.GetVisibleMonsters()
}

// GetMonsterSystem returns the monster system (for legacy handler compatibility)
func (gm *GameManager) GetMonsterSystem() *MonsterSystem {
	return gm.monsterSystem
}

// GetFurnitureForSnapshot returns furniture in revealed regions for client snapshot
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

		// Check if furniture is in a revealed region (only include known furniture)
		if !gm.gameState.KnownFurniture[instance.ID] {
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
			Rotation:           instance.Rotation,
			SwapAspectOnRotate: instance.SwapAspectOnRotate,
			TileImage:          instance.Definition.Rendering.TileImage,
			TileImageCleaned:   instance.Definition.Rendering.TileImageCleaned,
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
		gm.logger.Printf("DEBUG: Added known furniture item to snapshot: %s (%s) at (%d,%d) rotation=%d",
			instance.ID, instance.Type, instance.Position.X, instance.Position.Y, instance.Rotation)
	}

	gm.logger.Printf("DEBUG: Returning %d known furniture items for snapshot", len(furniture))
	return furniture
}

// GetMonstersForSnapshot returns all monsters in the format needed for client snapshot
func (gm *GameManager) GetMonstersForSnapshot() []protocol.MonsterLite {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	allMonsters := gm.monsterSystem.GetMonsters()
	gm.logger.Printf("DEBUG: GetMonstersForSnapshot called - found %d monster instances", len(allMonsters))
	monsters := make([]protocol.MonsterLite, 0, len(allMonsters))

	for _, monster := range allMonsters {
		// Only include monsters that have been discovered
		if gm.gameState.KnownMonsters[monster.ID] {
			monsterItem := protocol.MonsterLite{
				ID:          monster.ID,
				Type:        string(monster.Type),
				Tile:        monster.Position,
				Body:        monster.Body,
				MaxBody:     monster.MaxBody,
				Mind:        monster.Mind,
				MaxMind:     monster.MaxMind,
				AttackDice:  monster.AttackDice,
				DefenseDice: monster.DefenseDice,
				IsVisible:   monster.IsVisible,
				IsAlive:     monster.IsAlive,
			}

			monsters = append(monsters, monsterItem)
			gm.logger.Printf("DEBUG: Added monster item to snapshot: %s (%s) at (%d,%d) - visible: %v, alive: %v",
				monster.ID, monster.Type, monster.Position.X, monster.Position.Y, monster.IsVisible, monster.IsAlive)
		}
	}

	gm.logger.Printf("DEBUG: Returning %d monster items for snapshot", len(monsters))
	return monsters
}

// GetHeroTurnStatesForSnapshot returns all hero turn states for client snapshot
func (gm *GameManager) GetHeroTurnStatesForSnapshot() map[string]protocol.HeroTurnStateLite {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	result := make(map[string]protocol.HeroTurnStateLite)

	if gm.turnStateManager == nil {
		return result
	}

	allStates := gm.turnStateManager.GetAllHeroStates()

	for heroID, state := range allStates {
		// Convert ActiveEffects
		effects := make([]protocol.ActiveEffectLite, 0, len(state.ActiveEffects))
		for _, effect := range state.ActiveEffects {
			effects = append(effects, protocol.ActiveEffectLite{
				Source:     effect.Source,
				EffectType: effect.EffectType,
				Value:      effect.Value,
				Trigger:    effect.Trigger,
				Applied:    effect.Applied,
			})
		}

		// Convert LocationSearches to summaries
		locationSearches := make(map[string]protocol.LocationSearchSummary)
		for locKey, locHistory := range state.LocationActions {
			if searchHistory, exists := locHistory.SearchesByHero[heroID]; exists {
				locationSearches[locKey] = protocol.LocationSearchSummary{
					LocationKey:        locKey,
					TreasureSearchDone: len(searchHistory.TreasureSearches) > 0,
				}
			}
		}

		// Get action type if action was taken
		actionType := ""
		if state.Action != nil {
			actionType = state.Action.ActionType
		}

		lite := protocol.HeroTurnStateLite{
			HeroID:              state.HeroID,
			PlayerID:            state.PlayerID,
			TurnNumber:          state.TurnNumber,
			MovementDiceRolled:  state.MovementDice.Rolled,
			MovementDiceResults: state.MovementDice.DiceResults,
			MovementTotal:       state.MovementDice.TotalMovement,
			MovementUsed:        state.MovementDice.MovementUsed,
			MovementRemaining:   state.MovementDice.MovementRemaining,
			HasMoved:            state.HasMoved,
			ActionTaken:         state.ActionTaken,
			ActionType:          actionType,
			TurnFlags:           state.TurnFlags,
			ActivitiesCount:     len(state.Activities),
			ActiveEffectsCount:  len(state.ActiveEffects),
			ActiveEffects:       effects,
			LocationSearches:    locationSearches,
			TurnStartPosition:   state.TurnStartPosition,
			CurrentPosition:     state.CurrentPosition,
		}

		result[heroID] = lite
		gm.logger.Printf("DEBUG: Added hero turn state to snapshot: %s (moved: %v, action: %v, movement: %d/%d)",
			heroID, state.HasMoved, state.ActionTaken, state.MovementDice.MovementUsed, state.MovementDice.TotalMovement)
	}

	gm.logger.Printf("DEBUG: Returning %d hero turn states for snapshot", len(result))
	return result
}
