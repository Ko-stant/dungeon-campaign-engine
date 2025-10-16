package main

import (
	"fmt"
	"sync"
	"time"
)

// TurnPhaseType defines the current phase of the game
type TurnPhaseType string

const (
	// QuestSetupPhase - players choosing starting positions
	QuestSetupPhase TurnPhaseType = "quest_setup"

	// HeroPhaseActive - a hero is currently taking their turn
	HeroPhaseActive TurnPhaseType = "hero_phase_active"

	// HeroPhaseElection - waiting for next hero to elect themselves
	HeroPhaseElection TurnPhaseType = "hero_phase_election"

	// GMPhase - GM controlling monsters and environment
	GMPhase TurnPhaseType = "gm_phase"
)

// SimpleMonsterTurnState tracks basic state of a monster during the GM phase
// This is a lightweight version used by DynamicTurnOrderManager for phase tracking
type SimpleMonsterTurnState struct {
	HasMoved    bool
	ActionTaken bool
}

// DynamicTurnOrderManager manages the dynamic turn order system
type DynamicTurnOrderManager struct {
	// Current phase state
	currentPhase TurnPhaseType
	cycleNumber  int // Which hero/GM cycle we're in

	// Hero phase tracking
	activeHeroPlayerID   string          // Currently acting hero (during HeroPhaseActive)
	heroesActedThisCycle map[string]bool // PlayerID -> has acted this cycle
	electedPlayerID      string          // Player who elected themselves as next (during election)
	electionStartTime    *time.Time      // When election started (for timeout handling)

	// Quest setup tracking
	playersReady         map[string]bool     // PlayerID -> ready state
	playerStartPositions map[string]Position // PlayerID -> chosen starting position

	// GM phase tracking
	monsterTurnStates map[string]*SimpleMonsterTurnState // MonsterID -> turn state

	// Configuration
	electionTimeoutSec int  // Seconds before auto-selecting a random player (0 = disabled)
	requireAllHeroes   bool // Whether all heroes must act before advancing to GM phase

	logger Logger
	mutex  sync.RWMutex
}

// Position represents a tile position
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// NewDynamicTurnOrderManager creates a new dynamic turn order manager
func NewDynamicTurnOrderManager(logger Logger) *DynamicTurnOrderManager {
	return &DynamicTurnOrderManager{
		currentPhase:         QuestSetupPhase,
		cycleNumber:          0,
		heroesActedThisCycle: make(map[string]bool),
		playersReady:         make(map[string]bool),
		playerStartPositions: make(map[string]Position),
		monsterTurnStates:    make(map[string]*SimpleMonsterTurnState),
		electionTimeoutSec:   0,    // Disabled by default
		requireAllHeroes:     true, // All heroes must act by default
		logger:               logger,
	}
}

// ==== Quest Setup Phase Methods ====

// SelectStartingPosition records a player's chosen starting position
func (dtom *DynamicTurnOrderManager) SelectStartingPosition(playerID string, pos Position) error {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != QuestSetupPhase {
		return &GameError{Code: "invalid_phase", Message: "can only select starting position during quest setup"}
	}

	dtom.playerStartPositions[playerID] = pos
	dtom.logger.Printf("Player %s selected starting position (%d, %d)", playerID, pos.X, pos.Y)

	return nil
}

// SetPlayerReady marks a player as ready to begin the quest
func (dtom *DynamicTurnOrderManager) SetPlayerReady(playerID string, ready bool) error {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != QuestSetupPhase {
		return &GameError{Code: "invalid_phase", Message: "can only set ready status during quest setup"}
	}

	dtom.playersReady[playerID] = ready
	dtom.logger.Printf("Player %s ready status: %t", playerID, ready)

	return nil
}

// AreAllPlayersReady checks if all registered players are ready
func (dtom *DynamicTurnOrderManager) AreAllPlayersReady() bool {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	for _, ready := range dtom.playersReady {
		if !ready {
			return false
		}
	}

	return len(dtom.playersReady) > 0
}

// StartQuestAfterSetup transitions from setup to first hero phase election
func (dtom *DynamicTurnOrderManager) StartQuestAfterSetup() error {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != QuestSetupPhase {
		return &GameError{Code: "invalid_phase", Message: "quest already started"}
	}

	if !dtom.areAllPlayersReadyLocked() {
		return &GameError{Code: "not_ready", Message: "not all players are ready"}
	}

	// Start first hero phase cycle with election
	dtom.currentPhase = HeroPhaseElection
	dtom.cycleNumber = 1
	now := time.Now()
	dtom.electionStartTime = &now

	dtom.logger.Printf("Quest started: Beginning turn cycle %d with hero election", dtom.cycleNumber)

	return nil
}

// ==== Hero Phase Election Methods ====

// ElectSelfAsNextPlayer allows a hero to volunteer to go next
func (dtom *DynamicTurnOrderManager) ElectSelfAsNextPlayer(playerID string) error {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != HeroPhaseElection {
		return &GameError{Code: "invalid_phase", Message: "not in election phase"}
	}

	// Check if player has already acted this cycle
	if dtom.heroesActedThisCycle[playerID] {
		return &GameError{Code: "already_acted", Message: "player has already acted this cycle"}
	}

	// Check if another player has already elected themselves
	if dtom.electedPlayerID != "" && dtom.electedPlayerID != playerID {
		return &GameError{Code: "already_elected", Message: "another player has already elected themselves"}
	}

	dtom.electedPlayerID = playerID
	dtom.logger.Printf("Player %s elected themselves to go next", playerID)

	return nil
}

// CancelPlayerElection allows a hero to cancel their self-election
func (dtom *DynamicTurnOrderManager) CancelPlayerElection(playerID string) error {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != HeroPhaseElection {
		return &GameError{Code: "invalid_phase", Message: "not in election phase"}
	}

	if dtom.electedPlayerID != playerID {
		return &GameError{Code: "not_elected", Message: "player has not elected themselves"}
	}

	dtom.electedPlayerID = ""
	dtom.logger.Printf("Player %s cancelled their election", playerID)

	return nil
}

// ConfirmElectionAndStartHeroTurn confirms the election and starts the elected hero's turn
func (dtom *DynamicTurnOrderManager) ConfirmElectionAndStartHeroTurn() (string, error) {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != HeroPhaseElection {
		return "", &GameError{Code: "invalid_phase", Message: "not in election phase"}
	}

	if dtom.electedPlayerID == "" {
		return "", &GameError{Code: "no_election", Message: "no player has elected themselves"}
	}

	// Transition to active hero phase
	playerID := dtom.electedPlayerID
	dtom.activeHeroPlayerID = playerID
	dtom.currentPhase = HeroPhaseActive
	dtom.electionStartTime = nil
	dtom.electedPlayerID = ""

	dtom.logger.Printf("Player %s confirmed as active hero for turn", playerID)

	return playerID, nil
}

// GetElectedPlayer returns the currently elected player (if any)
func (dtom *DynamicTurnOrderManager) GetElectedPlayer() string {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()
	return dtom.electedPlayerID
}

// GetEligibleHeroes returns list of heroes who can still elect themselves
func (dtom *DynamicTurnOrderManager) GetEligibleHeroes(allPlayerIDs []string) []string {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	eligible := make([]string, 0)
	for _, playerID := range allPlayerIDs {
		// Skip GM players (they don't participate in hero phase)
		if playerID == "gamemaster" {
			continue
		}
		// Include only heroes who haven't acted this cycle
		if !dtom.heroesActedThisCycle[playerID] {
			eligible = append(eligible, playerID)
		}
	}

	return eligible
}

// ==== Hero Turn Completion Methods ====

// CompleteHeroTurn marks the active hero's turn as complete and transitions to election
func (dtom *DynamicTurnOrderManager) CompleteHeroTurn() error {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != HeroPhaseActive {
		return &GameError{Code: "invalid_phase", Message: "no active hero turn"}
	}

	// Mark hero as acted this cycle
	dtom.heroesActedThisCycle[dtom.activeHeroPlayerID] = true

	dtom.logger.Printf("Player %s completed their turn", dtom.activeHeroPlayerID)

	// Clear active player
	dtom.activeHeroPlayerID = ""

	// Check if all heroes have acted this cycle
	if dtom.shouldAdvanceToGMPhase() {
		return dtom.advanceToGMPhaseLocked()
	}

	// Otherwise, transition to election phase
	dtom.currentPhase = HeroPhaseElection
	now := time.Now()
	dtom.electionStartTime = &now

	dtom.logger.Printf("Transitioning to hero election phase")

	return nil
}

// ==== GM Phase Methods ====

// SetMonsterMoved marks a monster as having moved during the GM phase
func (dtom *DynamicTurnOrderManager) SetMonsterMoved(monsterID string, hasMoved bool) error {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != GMPhase {
		return &GameError{Code: "invalid_phase", Message: "can only set monster state during GM phase"}
	}

	// Get or create monster state
	if dtom.monsterTurnStates[monsterID] == nil {
		dtom.monsterTurnStates[monsterID] = &SimpleMonsterTurnState{}
	}

	dtom.monsterTurnStates[monsterID].HasMoved = hasMoved
	dtom.logger.Printf("Monster %s moved state: %t", monsterID, hasMoved)

	return nil
}

// SetMonsterActionTaken marks a monster as having taken an action during the GM phase
func (dtom *DynamicTurnOrderManager) SetMonsterActionTaken(monsterID string, actionTaken bool) error {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != GMPhase {
		return &GameError{Code: "invalid_phase", Message: "can only set monster state during GM phase"}
	}

	// Get or create monster state
	if dtom.monsterTurnStates[monsterID] == nil {
		dtom.monsterTurnStates[monsterID] = &SimpleMonsterTurnState{}
	}

	dtom.monsterTurnStates[monsterID].ActionTaken = actionTaken
	dtom.logger.Printf("Monster %s action taken state: %t", monsterID, actionTaken)

	return nil
}

// GetMonsterTurnState returns the turn state for a monster
func (dtom *DynamicTurnOrderManager) GetMonsterTurnState(monsterID string) *SimpleMonsterTurnState {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	if state, exists := dtom.monsterTurnStates[monsterID]; exists {
		// Return a copy to avoid concurrent access issues
		return &SimpleMonsterTurnState{
			HasMoved:    state.HasMoved,
			ActionTaken: state.ActionTaken,
		}
	}

	// Return default state
	return &SimpleMonsterTurnState{
		HasMoved:    false,
		ActionTaken: false,
	}
}

// GetAllMonsterTurnStates returns a copy of all monster turn states
func (dtom *DynamicTurnOrderManager) GetAllMonsterTurnStates() map[string]*SimpleMonsterTurnState {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	copy := make(map[string]*SimpleMonsterTurnState)
	for monsterID, state := range dtom.monsterTurnStates {
		copy[monsterID] = &SimpleMonsterTurnState{
			HasMoved:    state.HasMoved,
			ActionTaken: state.ActionTaken,
		}
	}
	return copy
}

// CompleteGMTurn marks the GM turn as complete and advances to next hero cycle
func (dtom *DynamicTurnOrderManager) CompleteGMTurn() error {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	if dtom.currentPhase != GMPhase {
		return &GameError{Code: "invalid_phase", Message: "not in GM phase"}
	}

	// Reset monster turn states for next GM phase
	dtom.monsterTurnStates = make(map[string]*SimpleMonsterTurnState)
	dtom.logger.Printf("Reset monster turn states")

	// Start new hero cycle
	dtom.cycleNumber++
	dtom.heroesActedThisCycle = make(map[string]bool) // Reset acted tracking
	dtom.currentPhase = HeroPhaseElection
	now := time.Now()
	dtom.electionStartTime = &now

	dtom.logger.Printf("GM turn completed, starting hero cycle %d", dtom.cycleNumber)

	return nil
}

// ==== Query Methods ====

// GetCurrentPhase returns the current phase
func (dtom *DynamicTurnOrderManager) GetCurrentPhase() TurnPhaseType {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()
	return dtom.currentPhase
}

// GetActiveHeroPlayerID returns the currently active hero player ID (if any)
func (dtom *DynamicTurnOrderManager) GetActiveHeroPlayerID() string {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()
	return dtom.activeHeroPlayerID
}

// GetCycleNumber returns the current cycle number
func (dtom *DynamicTurnOrderManager) GetCycleNumber() int {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()
	return dtom.cycleNumber
}

// GetHeroesActedThisCycle returns a copy of the heroes acted map
func (dtom *DynamicTurnOrderManager) GetHeroesActedThisCycle() map[string]bool {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	copy := make(map[string]bool)
	for k, v := range dtom.heroesActedThisCycle {
		copy[k] = v
	}
	return copy
}

// GetPlayersReady returns a copy of the players ready map
func (dtom *DynamicTurnOrderManager) GetPlayersReady() map[string]bool {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	copy := make(map[string]bool)
	for k, v := range dtom.playersReady {
		copy[k] = v
	}
	return copy
}

// GetPlayerStartPositions returns a copy of the player start positions map
func (dtom *DynamicTurnOrderManager) GetPlayerStartPositions() map[string]Position {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	copy := make(map[string]Position)
	for k, v := range dtom.playerStartPositions {
		copy[k] = v
	}
	return copy
}

// IsHeroTurn checks if it's a hero's turn (active or election)
func (dtom *DynamicTurnOrderManager) IsHeroTurn() bool {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()
	return dtom.currentPhase == HeroPhaseActive || dtom.currentPhase == HeroPhaseElection
}

// IsGMTurn checks if it's the GM's turn
func (dtom *DynamicTurnOrderManager) IsGMTurn() bool {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()
	return dtom.currentPhase == GMPhase
}

// IsQuestSetup checks if the quest is in setup phase
func (dtom *DynamicTurnOrderManager) IsQuestSetup() bool {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()
	return dtom.currentPhase == QuestSetupPhase
}

// CanPlayerAct checks if a specific player can take actions right now
func (dtom *DynamicTurnOrderManager) CanPlayerAct(playerID string) bool {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	return dtom.currentPhase == HeroPhaseActive && dtom.activeHeroPlayerID == playerID
}

// RegisterPlayer registers a new player for turn order tracking
func (dtom *DynamicTurnOrderManager) RegisterPlayer(playerID string) {
	dtom.mutex.Lock()
	defer dtom.mutex.Unlock()

	// Initialize tracking for this player
	if dtom.currentPhase == QuestSetupPhase {
		dtom.playersReady[playerID] = false
	}

	dtom.logger.Printf("Registered player %s for turn order", playerID)
}

// ==== Private Helper Methods ====

func (dtom *DynamicTurnOrderManager) shouldAdvanceToGMPhase() bool {
	if !dtom.requireAllHeroes {
		// If we don't require all heroes, check if at least one has acted
		return len(dtom.heroesActedThisCycle) > 0
	}

	// Check if all registered heroes have acted
	// Note: This assumes all non-GM players are registered
	// In practice, we'd compare against total hero count from TurnManager
	return len(dtom.heroesActedThisCycle) >= len(dtom.playersReady)
}

func (dtom *DynamicTurnOrderManager) advanceToGMPhaseLocked() error {
	dtom.currentPhase = GMPhase
	dtom.electionStartTime = nil

	dtom.logger.Printf("All heroes acted, advancing to GM phase for cycle %d", dtom.cycleNumber)

	return nil
}

func (dtom *DynamicTurnOrderManager) areAllPlayersReadyLocked() bool {
	for _, ready := range dtom.playersReady {
		if !ready {
			return false
		}
	}
	return len(dtom.playersReady) > 0
}

// ==== State Summary Methods (for debugging/UI) ====

// GetStateString returns a human-readable state summary
func (dtom *DynamicTurnOrderManager) GetStateString() string {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	switch dtom.currentPhase {
	case QuestSetupPhase:
		readyCount := 0
		for _, ready := range dtom.playersReady {
			if ready {
				readyCount++
			}
		}
		return fmt.Sprintf("Quest Setup: %d/%d players ready", readyCount, len(dtom.playersReady))

	case HeroPhaseElection:
		if dtom.electedPlayerID != "" {
			return fmt.Sprintf("Hero Election: Player %s elected (awaiting confirmation)", dtom.electedPlayerID)
		}
		return fmt.Sprintf("Hero Election: Cycle %d - waiting for hero to elect", dtom.cycleNumber)

	case HeroPhaseActive:
		return fmt.Sprintf("Hero Turn: Cycle %d - Player %s acting", dtom.cycleNumber, dtom.activeHeroPlayerID)

	case GMPhase:
		return fmt.Sprintf("GM Turn: Cycle %d", dtom.cycleNumber)

	default:
		return "Unknown Phase"
	}
}

// GetStateSummary returns a structured state summary (for snapshots)
func (dtom *DynamicTurnOrderManager) GetStateSummary() map[string]interface{} {
	dtom.mutex.RLock()
	defer dtom.mutex.RUnlock()

	return map[string]interface{}{
		"current_phase":         string(dtom.currentPhase),
		"cycle_number":          dtom.cycleNumber,
		"active_hero_player_id": dtom.activeHeroPlayerID,
		"elected_player_id":     dtom.electedPlayerID,
		"heroes_acted":          dtom.heroesActedThisCycle,
		"players_ready":         dtom.playersReady,
	}
}
