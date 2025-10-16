package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// DebugConfig holds debug mode configuration
type DebugConfig struct {
	Enabled            bool
	AllowStateChanges  bool
	AllowTeleportation bool
	AllowMapReveal     bool
	AllowDiceOverride  bool
	LogDebugActions    bool
}

// DebugSystem provides development and testing utilities
type DebugSystem struct {
	config          DebugConfig
	gameState       *GameState
	broadcaster     Broadcaster
	logger          Logger
	diceOverride    map[string]int   // Override next dice rolls (single die)
	diceOverrideSeq map[string][]int // Override sequences for multiple dice
}

// NewDebugSystem creates a new debug system
func NewDebugSystem(config DebugConfig, gameState *GameState, broadcaster Broadcaster, logger Logger) *DebugSystem {
	return &DebugSystem{
		config:          config,
		gameState:       gameState,
		broadcaster:     broadcaster,
		logger:          logger,
		diceOverride:    make(map[string]int),
		diceOverrideSeq: make(map[string][]int),
	}
}

// SetDiceOverride sets an override for a specific dice roll type (for testing)
func (ds *DebugSystem) SetDiceOverride(rollType string, value int) {
	if ds.config.AllowDiceOverride {
		ds.diceOverride[rollType] = value
	}
}

// GetDiceOverride gets a dice override if one exists
func (ds *DebugSystem) GetDiceOverride(rollType string) (int, bool) {
	if !ds.config.AllowDiceOverride {
		return 0, false
	}
	value, exists := ds.diceOverride[rollType]
	return value, exists
}

// GetDiceOverrideSequence gets multiple dice results from a sequence
func (ds *DebugSystem) GetDiceOverrideSequence(rollType string, count int) ([]int, bool) {
	if !ds.config.AllowDiceOverride {
		return nil, false
	}

	sequence, exists := ds.diceOverrideSeq[rollType]
	if !exists || len(sequence) == 0 {
		return nil, false
	}

	// Return up to the requested count, or all available results
	resultCount := count
	if resultCount > len(sequence) {
		resultCount = len(sequence)
	}

	results := make([]int, resultCount)
	copy(results, sequence[:resultCount])

	// Remove used results from the sequence
	ds.diceOverrideSeq[rollType] = sequence[resultCount:]

	// If sequence is empty, remove it
	if len(ds.diceOverrideSeq[rollType]) == 0 {
		delete(ds.diceOverrideSeq, rollType)
	}

	return results, true
}

// ClearDiceOverride removes a dice override
func (ds *DebugSystem) ClearDiceOverride(rollType string) {
	if ds.config.AllowDiceOverride {
		delete(ds.diceOverride, rollType)
		delete(ds.diceOverrideSeq, rollType)
	}
}

// SetCombatDiceOverride sets multiple dice overrides for testing combat scenarios
func (ds *DebugSystem) SetCombatDiceOverride(attackResults []int, defenseResults []int) {
	if !ds.config.AllowDiceOverride {
		return
	}

	// Clear existing combat overrides
	delete(ds.diceOverride, "attack")
	delete(ds.diceOverride, "defense")

	// Store attack results as comma-separated values
	if len(attackResults) > 0 {
		ds.setCombatRollSequence("attack", attackResults)
	}

	// Store defense results as comma-separated values
	if len(defenseResults) > 0 {
		ds.setCombatRollSequence("defense", defenseResults)
	}
}

// setCombatRollSequence stores a sequence of dice results for combat testing
func (ds *DebugSystem) setCombatRollSequence(rollType string, results []int) {
	if len(results) > 0 {
		// Store the entire sequence for multi-die rolling
		ds.diceOverrideSeq[rollType] = make([]int, len(results))
		copy(ds.diceOverrideSeq[rollType], results)
	}
}

// DebugAction represents a debug action taken
type DebugAction struct {
	Type        string         `json:"type"`
	Parameters  map[string]any `json:"parameters"`
	Success     bool           `json:"success"`
	Message     string         `json:"message"`
	Timestamp   string         `json:"timestamp"`
	StateChange *StateChange   `json:"stateChange,omitempty"`
}

// StateChange represents changes made to game state
type StateChange struct {
	EntityPositions map[string]protocol.TileAddress `json:"entityPositions,omitempty"`
	DoorStates      map[string]string               `json:"doorStates,omitempty"`
	RevealedRegions []int                           `json:"revealedRegions,omitempty"`
	AddedMonsters   []string                        `json:"addedMonsters,omitempty"`
	RemovedMonsters []string                        `json:"removedMonsters,omitempty"`
}

// Debug API endpoints
func (ds *DebugSystem) RegisterDebugRoutes(mux *http.ServeMux) {
	if !ds.config.Enabled {
		return
	}

	// Hero manipulation
	mux.HandleFunc("/debug/hero/teleport", ds.handleHeroTeleport)
	mux.HandleFunc("/debug/hero/god-mode", ds.handleGodMode)

	// Map manipulation
	mux.HandleFunc("/debug/map/reveal", ds.handleRevealMap)
	mux.HandleFunc("/debug/doors/open-all", ds.handleOpenAllDoors)
	mux.HandleFunc("/debug/doors/close-all", ds.handleCloseAllDoors)

	// Quest manipulation
	mux.HandleFunc("/debug/quest/complete", ds.handleCompleteQuest)
	mux.HandleFunc("/debug/quest/reset", ds.handleResetQuest)

	// Monster manipulation
	mux.HandleFunc("/debug/monsters/spawn", ds.handleSpawnMonster)
	mux.HandleFunc("/debug/monsters/kill-all", ds.handleKillAllMonsters)
	mux.HandleFunc("/debug/monsters/reveal-all", ds.handleRevealAllMonsters)

	// Dice manipulation
	mux.HandleFunc("/debug/dice/override", ds.handleDiceOverride)
	mux.HandleFunc("/debug/dice/clear-override", ds.handleClearDiceOverride)
	mux.HandleFunc("/debug/dice/combat-test", ds.handleCombatDiceTest)

	// State manipulation
	mux.HandleFunc("/debug/state/export", ds.handleExportState)
	mux.HandleFunc("/debug/state/import", ds.handleImportState)
	mux.HandleFunc("/debug/state/reset", ds.handleResetState)

	// Turn manipulation
	mux.HandleFunc("/debug/turn/advance", ds.handleAdvanceTurn)
	mux.HandleFunc("/debug/turn/set-gamemaster", ds.handleSetGameMasterTurn)

	// Information endpoints
	mux.HandleFunc("/debug/info/state", ds.handleGetDebugInfo)
	mux.HandleFunc("/debug/info/actions", ds.handleGetActionHistory)
}

// Hero Teleportation
func (ds *DebugSystem) handleHeroTeleport(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	var req struct {
		EntityID string `json:"entityId"`
		X        int    `json:"x"`
		Y        int    `json:"y"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate coordinates
	if req.X < 0 || req.Y < 0 || req.X >= ds.gameState.Segment.Width || req.Y >= ds.gameState.Segment.Height {
		http.Error(w, "Coordinates out of bounds", http.StatusBadRequest)
		return
	}

	// Default to hero-1 if no entity specified
	if req.EntityID == "" {
		req.EntityID = "hero-1"
	}

	ds.gameState.Lock.Lock()
	oldPos := ds.gameState.Entities[req.EntityID]
	newPos := protocol.TileAddress{
		SegmentID: oldPos.SegmentID,
		X:         req.X,
		Y:         req.Y,
	}
	ds.gameState.Entities[req.EntityID] = newPos
	ds.gameState.Lock.Unlock()

	// Broadcast entity update
	ds.broadcaster.BroadcastEvent("EntityUpdated", protocol.EntityUpdated{
		ID:   req.EntityID,
		Tile: newPos,
	})

	// Log debug action
	ds.logDebugAction("hero_teleport", map[string]any{
		"entityId": req.EntityID,
		"from":     fmt.Sprintf("(%d,%d)", oldPos.X, oldPos.Y),
		"to":       fmt.Sprintf("(%d,%d)", req.X, req.Y),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Teleported %s to (%d,%d)", req.EntityID, req.X, req.Y),
	})
}

// Reveal entire map
func (ds *DebugSystem) handleRevealMap(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	ds.gameState.Lock.Lock()
	// Reveal all regions
	for i := 0; i < ds.gameState.RegionMap.RegionsCount; i++ {
		ds.gameState.RevealedRegions[i] = true
		ds.gameState.KnownRegions[i] = true
	}

	// Reveal all doors
	for doorID := range ds.gameState.Doors {
		ds.gameState.KnownDoors[doorID] = true
	}
	ds.gameState.Lock.Unlock()

	// Get all region IDs
	allRegions := make([]int, ds.gameState.RegionMap.RegionsCount)
	for i := 0; i < ds.gameState.RegionMap.RegionsCount; i++ {
		allRegions[i] = i
	}

	// Broadcast updates
	ds.broadcaster.BroadcastEvent("RegionsRevealed", protocol.RegionsRevealed{IDs: allRegions})
	ds.broadcaster.BroadcastEvent("RegionsKnown", protocol.RegionsKnown{IDs: allRegions})

	// Create door list
	var doors []protocol.ThresholdLite
	for id, info := range ds.gameState.Doors {
		doors = append(doors, protocol.ThresholdLite{
			ID:          id,
			X:           info.Edge.X,
			Y:           info.Edge.Y,
			Orientation: string(info.Edge.Orientation),
			Kind:        "DoorSocket",
			State:       info.State,
		})
	}
	ds.broadcaster.BroadcastEvent("DoorsVisible", protocol.DoorsVisible{Doors: doors})

	ds.logDebugAction("reveal_map", map[string]any{
		"regionsRevealed": len(allRegions),
		"doorsRevealed":   len(doors),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Revealed entire map (%d regions, %d doors)", len(allRegions), len(doors)),
	})
}

// Open all doors
func (ds *DebugSystem) handleOpenAllDoors(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	ds.gameState.Lock.Lock()
	doorCount := 0
	for doorID, door := range ds.gameState.Doors {
		if door.State != "open" {
			door.State = "open"
			doorCount++

			// Broadcast door state change
			ds.broadcaster.BroadcastEvent("DoorStateChanged", protocol.DoorStateChanged{
				ThresholdID: doorID,
				State:       "open",
			})
		}
	}
	ds.gameState.Lock.Unlock()

	ds.logDebugAction("open_all_doors", map[string]any{
		"doorsOpened": doorCount,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Opened %d doors", doorCount),
	})
}

// Get debug information
func (ds *DebugSystem) handleGetDebugInfo(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	ds.gameState.Lock.Lock()
	debugInfo := map[string]any{
		"gameState": map[string]any{
			"entities":        ds.gameState.Entities,
			"revealedRegions": len(ds.gameState.RevealedRegions),
			"knownRegions":    len(ds.gameState.KnownRegions),
			"knownDoors":      len(ds.gameState.KnownDoors),
			"totalDoors":      len(ds.gameState.Doors),
			"totalRegions":    ds.gameState.RegionMap.RegionsCount,
		},
		"mapInfo": map[string]any{
			"width":  ds.gameState.Segment.Width,
			"height": ds.gameState.Segment.Height,
		},
		"debugConfig":   ds.config,
		"diceOverrides": ds.diceOverride,
	}
	ds.gameState.Lock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(debugInfo)
}

// Dice override for testing
func (ds *DebugSystem) handleDiceOverride(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	var req struct {
		DiceType string `json:"diceType"` // "attack", "defense", "movement"
		Value    int    `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Value < 1 || req.Value > 6 {
		http.Error(w, "Dice value must be between 1 and 6", http.StatusBadRequest)
		return
	}

	ds.diceOverride[req.DiceType] = req.Value

	ds.logDebugAction("dice_override", map[string]any{
		"diceType": req.DiceType,
		"value":    req.Value,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Next %s dice will roll %d", req.DiceType, req.Value),
	})
}

// Helper methods

func (ds *DebugSystem) checkDebugEnabled(w http.ResponseWriter) bool {
	if !ds.config.Enabled {
		http.Error(w, "Debug mode not enabled", http.StatusForbidden)
		return false
	}
	return true
}

func (ds *DebugSystem) logDebugAction(actionType string, params map[string]any) {
	if ds.config.LogDebugActions {
		ds.logger.Printf("DEBUG ACTION: %s - %+v", actionType, params)
	}
}

// GetDebugConfigFromEnv creates debug config from environment variables
func GetDebugConfigFromEnv() DebugConfig {
	return DebugConfig{
		Enabled:            getEnvBool("DEBUG_MODE", false),
		AllowStateChanges:  getEnvBool("DEBUG_ALLOW_STATE_CHANGES", true),
		AllowTeleportation: getEnvBool("DEBUG_ALLOW_TELEPORT", true),
		AllowMapReveal:     getEnvBool("DEBUG_ALLOW_MAP_REVEAL", true),
		AllowDiceOverride:  getEnvBool("DEBUG_ALLOW_DICE_OVERRIDE", true),
		LogDebugActions:    getEnvBool("DEBUG_LOG_ACTIONS", true),
	}
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	result, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return result
}

// God mode toggle
func (ds *DebugSystem) handleGodMode(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	var req struct {
		EntityID string `json:"entityId"`
		Enabled  bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.EntityID == "" {
		req.EntityID = "hero-1"
	}

	// TODO: Implement god mode state tracking
	ds.logDebugAction("god_mode", map[string]any{
		"entityId": req.EntityID,
		"enabled":  req.Enabled,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("God mode %s for %s", map[bool]string{true: "enabled", false: "disabled"}[req.Enabled], req.EntityID),
	})
}

// Complete quest
func (ds *DebugSystem) handleCompleteQuest(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	var req struct {
		QuestID string `json:"questId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.QuestID == "" {
		req.QuestID = "quest-01"
	}

	// TODO: Implement quest completion logic
	ds.logDebugAction("complete_quest", map[string]any{
		"questId": req.QuestID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Quest %s completed", req.QuestID),
	})
}

// Reset quest
func (ds *DebugSystem) handleResetQuest(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	var req struct {
		QuestID string `json:"questId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.QuestID == "" {
		req.QuestID = "quest-01"
	}

	// TODO: Implement quest reset logic
	ds.logDebugAction("reset_quest", map[string]any{
		"questId": req.QuestID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Quest %s reset", req.QuestID),
	})
}

// Spawn monster
func (ds *DebugSystem) handleSpawnMonster(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	var req struct {
		MonsterType string `json:"monsterType"`
		X           int    `json:"x"`
		Y           int    `json:"y"`
		MonsterID   string `json:"monsterId,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.MonsterType == "" {
		req.MonsterType = "goblin"
	}

	if req.MonsterID == "" {
		req.MonsterID = fmt.Sprintf("%s_%d", req.MonsterType, time.Now().Unix())
	}

	// Validate coordinates
	if req.X < 0 || req.Y < 0 || req.X >= ds.gameState.Segment.Width || req.Y >= ds.gameState.Segment.Height {
		http.Error(w, "Coordinates out of bounds", http.StatusBadRequest)
		return
	}

	// TODO: Implement monster spawning logic
	ds.logDebugAction("spawn_monster", map[string]any{
		"monsterType": req.MonsterType,
		"monsterId":   req.MonsterID,
		"position":    fmt.Sprintf("(%d,%d)", req.X, req.Y),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Spawned %s at (%d,%d)", req.MonsterType, req.X, req.Y),
		"data": map[string]any{
			"monsterId": req.MonsterID,
		},
	})
}

// Kill all monsters
func (ds *DebugSystem) handleKillAllMonsters(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	// TODO: Implement monster killing logic
	killedCount := 0 // Placeholder

	ds.logDebugAction("kill_all_monsters", map[string]any{
		"killedCount": killedCount,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Killed %d monsters", killedCount),
	})
}

// Reveal all monsters
func (ds *DebugSystem) handleRevealAllMonsters(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	// TODO: Implement monster revelation logic
	revealedCount := 0 // Placeholder

	ds.logDebugAction("reveal_all_monsters", map[string]any{
		"revealedCount": revealedCount,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Revealed %d monsters", revealedCount),
	})
}

// Clear dice override
func (ds *DebugSystem) handleClearDiceOverride(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	overrideCount := len(ds.diceOverride)
	ds.diceOverride = make(map[string]int)

	ds.logDebugAction("clear_dice_override", map[string]any{
		"clearedCount": overrideCount,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Cleared %d dice overrides", overrideCount),
	})
}

// Combat dice test - allows testing dice values from 2-12 for comprehensive combat testing
func (ds *DebugSystem) handleCombatDiceTest(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	var req struct {
		AttackDice   []int  `json:"attackDice"`   // Array of dice results for attack (1-6 each)
		DefenseDice  []int  `json:"defenseDice"`  // Array of dice results for defense (1-6 each)
		TestScenario string `json:"testScenario"` // Optional description of test scenario
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate dice values (each die must be 1-6)
	for _, die := range req.AttackDice {
		if die < 1 || die > 6 {
			http.Error(w, "Each attack die must be between 1 and 6", http.StatusBadRequest)
			return
		}
	}

	for _, die := range req.DefenseDice {
		if die < 1 || die > 6 {
			http.Error(w, "Each defense die must be between 1 and 6", http.StatusBadRequest)
			return
		}
	}

	// Clear existing overrides and set new ones
	ds.SetCombatDiceOverride(req.AttackDice, req.DefenseDice)

	// Calculate what the combat results would be for logging
	attackSkulls := 0
	for _, die := range req.AttackDice {
		if die >= 4 { // Skulls on 4, 5, 6
			attackSkulls++
		}
	}

	defenseShields := 0
	for _, die := range req.DefenseDice {
		if die == 6 { // Black shield
			defenseShields++
		} else if die == 4 || die == 5 { // White shields
			defenseShields++
		}
	}

	netDamage := attackSkulls - defenseShields
	if netDamage < 0 {
		netDamage = 0
	}

	ds.logDebugAction("combat_dice_test", map[string]any{
		"attackDice":     req.AttackDice,
		"defenseDice":    req.DefenseDice,
		"attackSkulls":   attackSkulls,
		"defenseShields": defenseShields,
		"netDamage":      netDamage,
		"testScenario":   req.TestScenario,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success":        true,
		"message":        "Combat dice override set",
		"attackDice":     req.AttackDice,
		"defenseDice":    req.DefenseDice,
		"attackSkulls":   attackSkulls,
		"defenseShields": defenseShields,
		"expectedDamage": netDamage,
		"testScenario":   req.TestScenario,
	})
}

// Export state
func (ds *DebugSystem) handleExportState(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	ds.gameState.Lock.Lock()
	exportData := map[string]any{
		"entities":        ds.gameState.Entities,
		"doors":           ds.gameState.Doors,
		"revealedRegions": ds.gameState.RevealedRegions,
		"knownRegions":    ds.gameState.KnownRegions,
		"knownDoors":      ds.gameState.KnownDoors,
		"timestamp":       time.Now(),
		"version":         "1.0",
	}
	ds.gameState.Lock.Unlock()

	ds.logDebugAction("export_state", map[string]any{
		"exportSize": fmt.Sprintf("%d entities, %d doors", len(ds.gameState.Entities), len(ds.gameState.Doors)),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    exportData,
	})
}

// Import state
func (ds *DebugSystem) handleImportState(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	var req struct {
		Data map[string]any `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Implement state import logic with validation
	ds.logDebugAction("import_state", map[string]any{
		"hasData": req.Data != nil,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "State imported successfully",
	})
}

// Reset state
func (ds *DebugSystem) handleResetState(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	ds.gameState.Lock.Lock()
	// Reset to initial state
	for entityID := range ds.gameState.Entities {
		if entityID == "hero-1" {
			// Reset hero to starting position
			ds.gameState.Entities[entityID] = protocol.TileAddress{X: 0, Y: 0} // TODO: Get from quest definition
		} else {
			// Remove other entities
			delete(ds.gameState.Entities, entityID)
		}
	}

	// Close all doors
	for _, door := range ds.gameState.Doors {
		door.State = "closed"
	}

	// Reset visibility
	ds.gameState.RevealedRegions = make(map[int]bool)
	ds.gameState.KnownRegions = make(map[int]bool)
	ds.gameState.KnownDoors = make(map[string]bool)
	ds.gameState.Lock.Unlock()

	ds.logDebugAction("reset_state", map[string]any{
		"message": "Game state reset to initial conditions",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Game state reset",
	})
}

// Advance turn
func (ds *DebugSystem) handleAdvanceTurn(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	// TODO: Integrate with TurnManager
	ds.logDebugAction("advance_turn", map[string]any{
		"message": "Turn advanced",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Turn advanced",
	})
}

// Set GameMaster turn
func (ds *DebugSystem) handleSetGameMasterTurn(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	// TODO: Integrate with TurnManager
	ds.logDebugAction("set_gamemaster_turn", map[string]any{
		"message": "Switched to GameMaster turn",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Switched to GameMaster turn",
	})
}

// Close all doors
func (ds *DebugSystem) handleCloseAllDoors(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	ds.gameState.Lock.Lock()
	doorCount := 0
	for doorID, door := range ds.gameState.Doors {
		if door.State != "closed" {
			door.State = "closed"
			doorCount++

			// Broadcast door state change
			ds.broadcaster.BroadcastEvent("DoorStateChanged", protocol.DoorStateChanged{
				ThresholdID: doorID,
				State:       "closed",
			})
		}
	}
	ds.gameState.Lock.Unlock()

	ds.logDebugAction("close_all_doors", map[string]any{
		"doorsClosed": doorCount,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Closed %d doors", doorCount),
	})
}

// Get action history
func (ds *DebugSystem) handleGetActionHistory(w http.ResponseWriter, r *http.Request) {
	if !ds.checkDebugEnabled(w) {
		return
	}

	// TODO: Implement action history tracking
	history := []map[string]any{
		{
			"type":      "placeholder",
			"message":   "Action history not yet implemented",
			"timestamp": time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"history": history,
	})
}
