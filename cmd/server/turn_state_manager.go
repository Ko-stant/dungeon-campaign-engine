package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// TurnStateManager manages turn state for all heroes and monsters in a quest
type TurnStateManager struct {
	currentTurn     int
	heroStates      map[string]*HeroTurnState    // Hero ID -> state
	monsterStates   map[string]*MonsterTurnState // Monster ID -> state
	selectedMonster string                       // Currently selected monster ID (for GM control)
	reactionStack   []ReactionContext            // For handling interrupts/reactions
	turnHistory     []TurnHistoryEntry           // For replay/undo (future)
	logger          Logger
	mutex           sync.RWMutex
}

// ReactionContext represents an interrupt/reaction scenario
type ReactionContext struct {
	TriggerEvent       string // "monster_attack", "hero_damaged", "trap_triggered"
	ActiveHeroID       string // Whose turn is being interrupted
	TargetHeroID       string // Who is being attacked/affected
	AvailableReactions []AvailableReaction
	Timestamp          int64
}

// AvailableReaction represents a reaction that a hero can use
type AvailableReaction struct {
	HeroID      string
	AbilityID   string
	AbilityName string
	CanUse      bool
	Reason      string // Why can/can't use
}

// TurnHistoryEntry represents a historical turn record (future use)
type TurnHistoryEntry struct {
	TurnNumber int
	HeroID     string
	Events     []string // Summary of events
	Timestamp  int64
}

// NewTurnStateManager creates a new turn state manager
func NewTurnStateManager(logger Logger) *TurnStateManager {
	return &TurnStateManager{
		currentTurn:     1,
		heroStates:      make(map[string]*HeroTurnState),
		monsterStates:   make(map[string]*MonsterTurnState),
		selectedMonster: "",
		reactionStack:   make([]ReactionContext, 0),
		turnHistory:     make([]TurnHistoryEntry, 0),
		logger:          logger,
	}
}

// StartHeroTurn initializes a new turn for a hero
func (tsm *TurnStateManager) StartHeroTurn(heroID, playerID string, startPosition protocol.TileAddress) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	// Check if hero already has an active turn
	if state, exists := tsm.heroStates[heroID]; exists {
		if !state.ActionTaken || state.MovementDice.MovementRemaining > 0 {
			return &GameError{Code: "turn_active", Message: "hero already has an active turn"}
		}
		// Turn complete, reset for new turn
		state.ResetForNewTurn(tsm.currentTurn)
	} else {
		// Create new turn state
		tsm.heroStates[heroID] = NewHeroTurnState(heroID, playerID, tsm.currentTurn, startPosition)
	}

	tsm.logger.Printf("Turn started for hero %s (player %s), turn %d", heroID, playerID, tsm.currentTurn)
	return nil
}

// RollMovementDice records a movement dice roll for a hero
func (tsm *TurnStateManager) RollMovementDice(heroID string, diceResults []int) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "hero has no active turn"}
	}

	err := state.RollMovementDice(diceResults)
	if err != nil {
		return err
	}

	tsm.logger.Printf("Hero %s rolled movement dice: %v (total: %d)", heroID, diceResults, state.MovementDice.TotalMovement)
	return nil
}

// RecordMovement records a movement step for a hero
func (tsm *TurnStateManager) RecordMovement(heroID string, to protocol.TileAddress) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "hero has no active turn"}
	}

	err := state.RecordMovement(to)
	if err != nil {
		return err
	}

	tsm.logger.Printf("Hero %s moved to (%d,%d), movement remaining: %d",
		heroID, to.X, to.Y, state.MovementDice.MovementRemaining)
	return nil
}

// RecordAction records an action for a hero
func (tsm *TurnStateManager) RecordAction(heroID string, action ActionRecord) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "hero has no active turn"}
	}

	err := state.RecordAction(action)
	if err != nil {
		return err
	}

	tsm.logger.Printf("Hero %s took action: %s (success: %t)", heroID, action.ActionType, action.Success)
	return nil
}

// RecordActivity records a non-action activity for a hero
func (tsm *TurnStateManager) RecordActivity(heroID string, activity Activity) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "hero has no active turn"}
	}

	state.RecordActivity(activity)
	tsm.logger.Printf("Hero %s performed activity: %s", heroID, activity.Type)
	return nil
}

// AddActiveEffect adds a pending effect to a hero
func (tsm *TurnStateManager) AddActiveEffect(heroID string, effect ActiveEffect) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "hero has no active turn"}
	}

	state.AddActiveEffect(effect)
	tsm.logger.Printf("Hero %s gained effect: %s (trigger: %s)", heroID, effect.EffectType, effect.Trigger)
	return nil
}

// TriggerEffects triggers effects for a hero with matching trigger
func (tsm *TurnStateManager) TriggerEffects(heroID string, trigger string) []ActiveEffect {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return nil
	}

	triggered := state.TriggerEffects(trigger)
	if len(triggered) > 0 {
		tsm.logger.Printf("Hero %s triggered %d effects for: %s", heroID, len(triggered), trigger)
	}
	return triggered
}

// CanMove validates whether a hero can move
func (tsm *TurnStateManager) CanMove(heroID string) (bool, string) {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return false, "hero has no active turn"
	}

	return state.CanMove()
}

// CanTakeAction validates whether a hero can take an action
func (tsm *TurnStateManager) CanTakeAction(heroID string, actionType string) (bool, string) {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return false, "hero has no active turn"
	}

	canAct, reason := state.CanTakeAction()
	if !canAct {
		return false, reason
	}

	// Additional validation based on action type can go here
	// For example, checking if hero has the spell they're trying to cast

	return true, ""
}

// CanUseItem validates whether a hero can use an item
func (tsm *TurnStateManager) CanUseItem(heroID string, itemID string, itemDef *ItemDefinition) (bool, string) {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return false, "hero has no active turn"
	}

	// Check if item has per-turn usage limit
	if itemDef != nil && itemDef.MaxUsesPerTurn > 0 {
		usageCount := state.ItemUsageThisTurn[itemID]
		if usageCount >= itemDef.MaxUsesPerTurn {
			return false, fmt.Sprintf("item can only be used %d time(s) per turn", itemDef.MaxUsesPerTurn)
		}
	}

	return true, ""
}

// ItemDefinition represents item metadata (placeholder - will be in ItemManager)
type ItemDefinition struct {
	ID              string
	Name            string
	MaxUsesPerTurn  int
	MaxUsesPerQuest int
}

// CanSearchTreasure validates whether a hero can search for treasure at a location
func (tsm *TurnStateManager) CanSearchTreasure(heroID string, locationKey string) (bool, string) {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return false, "hero has no active turn"
	}

	return state.CanSearchTreasure(locationKey)
}

// RecordSearch records a search action for a hero
func (tsm *TurnStateManager) RecordSearch(heroID string, searchType string, locationKey string, locationType string, position protocol.TileAddress, success bool, foundItems []string) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "hero has no active turn"}
	}

	state.RecordSearch(searchType, locationKey, locationType, position, success, foundItems)
	tsm.logger.Printf("Hero %s searched for %s at %s (success: %t)", heroID, searchType, locationKey, success)
	return nil
}

// RecordTurnEvent logs a turn event for a hero
func (tsm *TurnStateManager) RecordTurnEvent(heroID string, eventType string, entityID string, details map[string]interface{}) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "hero has no active turn"}
	}

	state.RecordTurnEvent(eventType, entityID, details)
	return nil
}

// CompleteHeroTurn marks a hero's turn as complete
func (tsm *TurnStateManager) CompleteHeroTurn(heroID string) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.heroStates[heroID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "hero has no active turn"}
	}

	tsm.logger.Printf("Hero %s completed turn %d", heroID, state.TurnNumber)

	// Note: We keep the state in memory for now
	// In the future, we might archive it to turnHistory

	return nil
}

// GetHeroTurnState retrieves a hero's turn state (read-only)
func (tsm *TurnStateManager) GetHeroTurnState(heroID string) *HeroTurnState {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	return tsm.heroStates[heroID]
}

// GetAllHeroStates returns all hero states (for snapshot generation)
func (tsm *TurnStateManager) GetAllHeroStates() map[string]*HeroTurnState {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	// Return a copy of the map to prevent external modification
	copy := make(map[string]*HeroTurnState)
	for k, v := range tsm.heroStates {
		copy[k] = v
	}
	return copy
}

// PushReactionContext adds a reaction context to the stack
func (tsm *TurnStateManager) PushReactionContext(ctx ReactionContext) {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	tsm.reactionStack = append(tsm.reactionStack, ctx)
	tsm.logger.Printf("Reaction context pushed: %s (target: %s)", ctx.TriggerEvent, ctx.TargetHeroID)
}

// PopReactionContext removes and returns the top reaction context
func (tsm *TurnStateManager) PopReactionContext() *ReactionContext {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	if len(tsm.reactionStack) == 0 {
		return nil
	}

	ctx := tsm.reactionStack[len(tsm.reactionStack)-1]
	tsm.reactionStack = tsm.reactionStack[:len(tsm.reactionStack)-1]

	tsm.logger.Printf("Reaction context popped: %s", ctx.TriggerEvent)
	return &ctx
}

// GetCurrentReactionContext returns the current reaction context without removing it
func (tsm *TurnStateManager) GetCurrentReactionContext() *ReactionContext {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	if len(tsm.reactionStack) == 0 {
		return nil
	}

	ctx := tsm.reactionStack[len(tsm.reactionStack)-1]
	return &ctx
}

// ====== Monster Turn State Management ======

// StartMonsterTurn initializes a new turn for a monster
func (tsm *TurnStateManager) StartMonsterTurn(monsterID, entityID string, startPosition protocol.TileAddress, fixedMovement, attackDice, defenseDice, bodyPoints, currentBody int) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	// Check if monster already has an active turn
	if state, exists := tsm.monsterStates[monsterID]; exists {
		// Turn complete, reset for new turn
		state.ResetForNewTurn(tsm.currentTurn)
	} else {
		// Create new turn state
		tsm.monsterStates[monsterID] = NewMonsterTurnState(monsterID, entityID, tsm.currentTurn, startPosition, fixedMovement, attackDice, defenseDice, bodyPoints, currentBody)
	}

	tsm.logger.Printf("Turn started for monster %s (entity %s), turn %d", monsterID, entityID, tsm.currentTurn)
	return nil
}

// SelectMonster sets the currently selected monster for GM control
func (tsm *TurnStateManager) SelectMonster(monsterID string) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	// Validate monster exists
	if monsterID != "" {
		if _, exists := tsm.monsterStates[monsterID]; !exists {
			return &GameError{Code: "monster_not_found", Message: "monster does not have an active turn state"}
		}
	}

	tsm.selectedMonster = monsterID
	tsm.logger.Printf("GM selected monster: %s", monsterID)
	return nil
}

// GetSelectedMonster returns the currently selected monster ID
func (tsm *TurnStateManager) GetSelectedMonster() string {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	return tsm.selectedMonster
}

// RecordMonsterMovement records a movement step for a monster
func (tsm *TurnStateManager) RecordMonsterMovement(monsterID string, to protocol.TileAddress) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.monsterStates[monsterID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "monster has no active turn"}
	}

	err := state.RecordMovement(to)
	if err != nil {
		return err
	}

	tsm.logger.Printf("Monster %s moved to (%d,%d), movement remaining: %d",
		monsterID, to.X, to.Y, state.MovementRemaining)
	return nil
}

// RecordMonsterAction records an action for a monster
func (tsm *TurnStateManager) RecordMonsterAction(monsterID string, action MonsterActionRecord) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.monsterStates[monsterID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "monster has no active turn"}
	}

	err := state.RecordAction(action)
	if err != nil {
		return err
	}

	tsm.logger.Printf("Monster %s took action: %s (target: %s, success: %t)",
		monsterID, action.ActionType, action.TargetID, action.Success)
	return nil
}

// UseMonsterAbility records usage of a monster's special ability
func (tsm *TurnStateManager) UseMonsterAbility(monsterID string, abilityID string, targetID string, targetPosition *protocol.TileAddress, success bool, details map[string]interface{}) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.monsterStates[monsterID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "monster has no active turn"}
	}

	err := state.UseAbility(abilityID, targetID, targetPosition, success, details)
	if err != nil {
		return err
	}

	tsm.logger.Printf("Monster %s used ability: %s (target: %s, success: %t)",
		monsterID, abilityID, targetID, success)
	return nil
}

// AddMonsterActiveEffect adds a buff/debuff to a monster
func (tsm *TurnStateManager) AddMonsterActiveEffect(monsterID string, effect MonsterActiveEffect) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.monsterStates[monsterID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "monster has no active turn"}
	}

	state.AddActiveEffect(effect)
	tsm.logger.Printf("Monster %s gained effect: %s (trigger: %s)", monsterID, effect.EffectType, effect.Trigger)
	return nil
}

// RecordMonsterDamage updates a monster's current body points
func (tsm *TurnStateManager) RecordMonsterDamage(monsterID string, damage int) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.monsterStates[monsterID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "monster has no active turn"}
	}

	state.RecordDamage(damage)
	tsm.logger.Printf("Monster %s took %d damage, current body: %d/%d",
		monsterID, damage, state.CurrentBody, state.BodyPoints)

	if !state.IsAlive() {
		tsm.logger.Printf("Monster %s has been killed", monsterID)
	}

	return nil
}

// GetMonsterTurnState retrieves a monster's turn state (read-only)
func (tsm *TurnStateManager) GetMonsterTurnState(monsterID string) *MonsterTurnState {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	return tsm.monsterStates[monsterID]
}

// GetAllMonsterStates returns all monster states (for snapshot generation)
func (tsm *TurnStateManager) GetAllMonsterStates() map[string]*MonsterTurnState {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	// Return a copy of the map to prevent external modification
	copy := make(map[string]*MonsterTurnState)
	for k, v := range tsm.monsterStates {
		copy[k] = v
	}
	return copy
}

// AddMonsterAbility adds a special ability to a monster (from quest configuration)
func (tsm *TurnStateManager) AddMonsterAbility(monsterID string, ability MonsterAbility) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	state := tsm.monsterStates[monsterID]
	if state == nil {
		return &GameError{Code: "no_active_turn", Message: "monster has no active turn"}
	}

	state.SpecialAbilities = append(state.SpecialAbilities, ability)

	// Initialize per-quest usage tracking if applicable
	if ability.UsesPerQuest > 0 {
		state.QuestAbilityUsageLeft[ability.ID] = ability.UsesPerQuest
	}

	tsm.logger.Printf("Monster %s gained ability: %s (type: %s)", monsterID, ability.Name, ability.Type)
	return nil
}

// RemoveMonsterState removes a monster's turn state (when monster dies or is removed from board)
func (tsm *TurnStateManager) RemoveMonsterState(monsterID string) {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	delete(tsm.monsterStates, monsterID)

	// Clear selection if this was the selected monster
	if tsm.selectedMonster == monsterID {
		tsm.selectedMonster = ""
	}

	tsm.logger.Printf("Monster %s turn state removed", monsterID)
}

// RemoveHeroState removes a hero's turn state (when canceling election)
func (tsm *TurnStateManager) RemoveHeroState(heroID string) {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	delete(tsm.heroStates, heroID)
	tsm.logger.Printf("Hero %s turn state removed", heroID)
}

// AdvanceTurn advances to the next turn and clears all hero turn states
func (tsm *TurnStateManager) AdvanceTurn() {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	tsm.currentTurn++
	tsm.logger.Printf("TurnStateManager: Advanced to turn %d, clearing all hero states", tsm.currentTurn)

	// Clear all hero states for the new turn
	// They will be recreated when each hero rolls movement dice
	tsm.heroStates = make(map[string]*HeroTurnState)
}

// SerializeForPersistence serializes the turn state manager to JSON (for future database storage)
func (tsm *TurnStateManager) SerializeForPersistence() ([]byte, error) {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	data := map[string]interface{}{
		"currentTurn":     tsm.currentTurn,
		"heroStates":      tsm.heroStates,
		"monsterStates":   tsm.monsterStates,
		"selectedMonster": tsm.selectedMonster,
		"reactionStack":   tsm.reactionStack,
		"turnHistory":     tsm.turnHistory,
	}

	return json.Marshal(data)
}

// RestoreFromPersistence restores the turn state manager from JSON (for future database loading)
func (tsm *TurnStateManager) RestoreFromPersistence(data []byte) error {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	var restored map[string]interface{}
	if err := json.Unmarshal(data, &restored); err != nil {
		return err
	}

	// This is a placeholder - full implementation would properly deserialize all fields
	tsm.logger.Printf("Turn state manager restored from persistence (placeholder)")
	return nil
}
