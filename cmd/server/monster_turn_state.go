package main

import (
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// MonsterTurnState tracks comprehensive turn state for a single monster during GM turn
// Similar to HeroTurnState but adapted for monster-specific mechanics:
// - No movement dice rolling (fixed movement stat)
// - Cannot open doors
// - GM-controlled selection and targeting
type MonsterTurnState struct {
	// Identity
	MonsterID  string
	EntityID   string // Reference to the entity in game state
	TurnNumber int    // Which GM turn in the quest

	// Monster Stats (cached from entity for quick access)
	FixedMovement int // Monsters have fixed movement, don't roll dice
	AttackDice    int
	DefenseDice   int
	BodyPoints    int
	CurrentBody   int

	// Movement Tracking
	MovementPath      []protocol.TileAddress // All tiles moved this GM turn
	MovementUsed      int                    // How much movement consumed so far
	MovementRemaining int                    // FixedMovement - MovementUsed

	// Movement/Action State (same flag-based model as heroes)
	HasMoved    bool            // True once any movement happens
	ActionTaken bool            // True once action happens (attack or special ability)
	TurnFlags   map[string]bool // Generic flags: "can_use_dread_spell", "extra_attack", etc.

	// Action Tracking
	Action *MonsterActionRecord // The single action taken this turn (nil if not taken)

	// Special Abilities (e.g., dread spells for quest-specific monsters)
	SpecialAbilities      []MonsterAbility      // Available special abilities
	SpecialAbilitiesUsed  map[string]int        // Ability ID -> usage count this turn
	QuestAbilityUsageLeft map[string]int        // Ability ID -> remaining uses for quest
	ActiveEffects         []MonsterActiveEffect // Buffs/debuffs on this monster

	// Turn Event Log
	TurnEvents []MonsterTurnEvent

	// Position Tracking
	TurnStartPosition protocol.TileAddress
	CurrentPosition   protocol.TileAddress

	// Timestamps
	TurnStartedAt  time.Time
	LastActivityAt time.Time
}

// MonsterActionRecord represents the main action taken by a monster this turn
type MonsterActionRecord struct {
	ActionType     string                 // "attack", "use_dread_spell", "special_ability"
	TargetID       string                 // Hero ID or entity ID if targeting something
	TargetPosition *protocol.TileAddress  // Position if targeting a location
	Success        bool                   // Whether action succeeded
	Details        map[string]interface{} // Action-specific data (damage dealt, spell cast, etc.)
	Timestamp      time.Time
}

// MonsterAbility represents a special ability a monster can use
type MonsterAbility struct {
	ID             string // "dread_spell_fireball", "fear_aura", "regeneration"
	Name           string
	Type           string // "dread_spell", "passive", "active"
	UsesPerTurn    int    // Max uses per turn (0 = unlimited)
	UsesPerQuest   int    // Max uses per quest (0 = unlimited)
	RequiresAction bool   // Does using this consume the monster's action?
	Range          int    // Range in tiles (0 = self, -1 = unlimited)
	Description    string
	EffectDetails  map[string]interface{} // Ability-specific parameters
}

// MonsterActiveEffect represents a buff/debuff on a monster
type MonsterActiveEffect struct {
	Source     string // "hero_spell_courage", "trap_snare", "item_smoke_bomb"
	EffectType string // "bonus_attack_dice", "movement_reduced", "passable"
	Value      int    // Numeric value if applicable
	Trigger    string // When effect applies: "on_attack", "on_defend", "permanent"
	ExpiresOn  string // "end_of_gm_turn", "end_of_hero_phase", "end_of_quest"
	Applied    bool   // Whether effect has been consumed (for one-time effects)
	CreatedAt  time.Time
}

// MonsterTurnEvent represents a logged event during monster's turn
type MonsterTurnEvent struct {
	EventType string // "moved", "attacked", "used_ability", "damaged", "killed"
	TargetID  string // Hero ID or entity ID affected by event
	Details   map[string]interface{}
	Timestamp time.Time
}

// NewMonsterTurnState creates a new monster turn state
func NewMonsterTurnState(monsterID, entityID string, turnNumber int, startPosition protocol.TileAddress, fixedMovement, attackDice, defenseDice, bodyPoints, currentBody int) *MonsterTurnState {
	return &MonsterTurnState{
		MonsterID:             monsterID,
		EntityID:              entityID,
		TurnNumber:            turnNumber,
		FixedMovement:         fixedMovement,
		AttackDice:            attackDice,
		DefenseDice:           defenseDice,
		BodyPoints:            bodyPoints,
		CurrentBody:           currentBody,
		MovementUsed:          0,
		MovementRemaining:     fixedMovement,
		TurnFlags:             make(map[string]bool),
		SpecialAbilities:      make([]MonsterAbility, 0),
		SpecialAbilitiesUsed:  make(map[string]int),
		QuestAbilityUsageLeft: make(map[string]int),
		ActiveEffects:         make([]MonsterActiveEffect, 0),
		TurnEvents:            make([]MonsterTurnEvent, 0),
		MovementPath:          make([]protocol.TileAddress, 0),
		TurnStartPosition:     startPosition,
		CurrentPosition:       startPosition,
		TurnStartedAt:         time.Now(),
		LastActivityAt:        time.Now(),
	}
}

// CanMove validates whether the monster can move based on turn state
func (mts *MonsterTurnState) CanMove() (bool, string) {
	if mts.MovementRemaining <= 0 {
		return false, "no movement remaining"
	}

	// Monsters can move before or after action (no strict ordering like heroes)
	// However, once both movement and action are done, no more moves
	// Note: Monsters cannot split movement like heroes with special abilities

	return true, ""
}

// CanTakeAction validates whether the monster can take an action
func (mts *MonsterTurnState) CanTakeAction() (bool, string) {
	if mts.ActionTaken {
		// Check for special "extra attack" abilities
		if mts.TurnFlags["can_make_extra_attack"] {
			return true, ""
		}
		return false, "action already taken this turn"
	}

	// Can take action if not taken yet
	return true, ""
}

// CanUseAbility checks if monster can use a specific special ability
func (mts *MonsterTurnState) CanUseAbility(abilityID string) (bool, string) {
	// Find the ability
	var ability *MonsterAbility
	for i := range mts.SpecialAbilities {
		if mts.SpecialAbilities[i].ID == abilityID {
			ability = &mts.SpecialAbilities[i]
			break
		}
	}

	if ability == nil {
		return false, "ability not found"
	}

	// Check per-turn usage limits
	if ability.UsesPerTurn > 0 {
		usedThisTurn := mts.SpecialAbilitiesUsed[abilityID]
		if usedThisTurn >= ability.UsesPerTurn {
			return false, "ability already used maximum times this turn"
		}
	}

	// Check per-quest usage limits
	if ability.UsesPerQuest > 0 {
		remaining := mts.QuestAbilityUsageLeft[abilityID]
		if remaining <= 0 {
			return false, "ability has no uses remaining this quest"
		}
	}

	// Check if ability requires an action
	if ability.RequiresAction {
		canAct, reason := mts.CanTakeAction()
		if !canAct {
			return false, reason
		}
	}

	return true, ""
}

// RecordMovement records a movement step for the monster
func (mts *MonsterTurnState) RecordMovement(to protocol.TileAddress) error {
	canMove, reason := mts.CanMove()
	if !canMove {
		return &GameError{Code: "cannot_move", Message: reason}
	}

	// Mark that movement has started
	mts.HasMoved = true

	// Add to movement path
	mts.MovementPath = append(mts.MovementPath, to)
	mts.CurrentPosition = to

	// Consume movement
	mts.MovementUsed++
	mts.MovementRemaining--

	// Log event
	mts.RecordTurnEvent("moved", "", map[string]interface{}{
		"position":           to,
		"movement_remaining": mts.MovementRemaining,
	})

	mts.LastActivityAt = time.Now()
	return nil
}

// RecordAction records the main action for this monster's turn
func (mts *MonsterTurnState) RecordAction(action MonsterActionRecord) error {
	canAct, reason := mts.CanTakeAction()
	if !canAct {
		return &GameError{Code: "cannot_act", Message: reason}
	}

	action.Timestamp = time.Now()
	mts.Action = &action
	mts.ActionTaken = true

	// Log event
	mts.RecordTurnEvent(action.ActionType, action.TargetID, action.Details)

	mts.LastActivityAt = time.Now()
	return nil
}

// UseAbility records usage of a special ability
func (mts *MonsterTurnState) UseAbility(abilityID string, targetID string, targetPosition *protocol.TileAddress, success bool, details map[string]interface{}) error {
	canUse, reason := mts.CanUseAbility(abilityID)
	if !canUse {
		return &GameError{Code: "cannot_use_ability", Message: reason}
	}

	// Find the ability
	var ability *MonsterAbility
	for i := range mts.SpecialAbilities {
		if mts.SpecialAbilities[i].ID == abilityID {
			ability = &mts.SpecialAbilities[i]
			break
		}
	}

	if ability == nil {
		return &GameError{Code: "ability_not_found", Message: "ability not found"}
	}

	// Track usage
	mts.SpecialAbilitiesUsed[abilityID]++
	if ability.UsesPerQuest > 0 {
		mts.QuestAbilityUsageLeft[abilityID]--
	}

	// If ability requires action, record it as the main action
	if ability.RequiresAction {
		action := MonsterActionRecord{
			ActionType:     "use_ability",
			TargetID:       targetID,
			TargetPosition: targetPosition,
			Success:        success,
			Details: map[string]interface{}{
				"ability_id":   abilityID,
				"ability_name": ability.Name,
			},
			Timestamp: time.Now(),
		}

		if details != nil {
			for k, v := range details {
				action.Details[k] = v
			}
		}

		return mts.RecordAction(action)
	}

	// Otherwise just log the event
	mts.RecordTurnEvent("used_ability", targetID, map[string]interface{}{
		"ability_id":   abilityID,
		"ability_name": ability.Name,
		"success":      success,
	})

	return nil
}

// AddActiveEffect adds a buff/debuff to the monster
func (mts *MonsterTurnState) AddActiveEffect(effect MonsterActiveEffect) {
	effect.CreatedAt = time.Now()
	effect.Applied = false
	mts.ActiveEffects = append(mts.ActiveEffects, effect)

	// Set turn flags for certain effects
	switch effect.EffectType {
	case "extra_attack":
		mts.TurnFlags["can_make_extra_attack"] = true
	case "passable":
		mts.TurnFlags["passable_by_heroes"] = true
	case "movement_halved":
		mts.MovementRemaining = mts.MovementRemaining / 2
	}
}

// TriggerEffects finds and marks effects with matching trigger, returns them
func (mts *MonsterTurnState) TriggerEffects(trigger string) []MonsterActiveEffect {
	triggered := make([]MonsterActiveEffect, 0)

	for i := range mts.ActiveEffects {
		effect := &mts.ActiveEffects[i]
		if effect.Trigger == trigger && !effect.Applied {
			effect.Applied = true
			triggered = append(triggered, *effect)
		}
	}

	return triggered
}

// RecordTurnEvent logs a turn event
func (mts *MonsterTurnState) RecordTurnEvent(eventType string, targetID string, details map[string]interface{}) {
	event := MonsterTurnEvent{
		EventType: eventType,
		TargetID:  targetID,
		Details:   details,
		Timestamp: time.Now(),
	}
	mts.TurnEvents = append(mts.TurnEvents, event)
}

// RecordDamage updates monster's current body points
func (mts *MonsterTurnState) RecordDamage(damage int) {
	oldBody := mts.CurrentBody
	mts.CurrentBody -= damage
	if mts.CurrentBody < 0 {
		mts.CurrentBody = 0
	}

	mts.RecordTurnEvent("damaged", "", map[string]interface{}{
		"damage_taken": damage,
		"old_body":     oldBody,
		"new_body":     mts.CurrentBody,
		"is_dead":      mts.CurrentBody == 0,
	})
}

// IsAlive returns whether the monster is still alive
func (mts *MonsterTurnState) IsAlive() bool {
	return mts.CurrentBody > 0
}

// ResetForNewTurn resets state for a new GM turn
func (mts *MonsterTurnState) ResetForNewTurn(newTurnNumber int) {
	mts.TurnNumber = newTurnNumber
	mts.TurnStartPosition = mts.CurrentPosition

	// Reset movement
	mts.MovementPath = make([]protocol.TileAddress, 0)
	mts.MovementUsed = 0
	mts.MovementRemaining = mts.FixedMovement
	mts.HasMoved = false

	// Reset action
	mts.Action = nil
	mts.ActionTaken = false

	// Reset turn flags (but keep quest-long flags)
	newFlags := make(map[string]bool)
	for key, value := range mts.TurnFlags {
		// Keep quest-long flags (add more as needed)
		if key == "quest_ability_unlocked" {
			newFlags[key] = value
		}
	}
	mts.TurnFlags = newFlags

	// Reset events
	mts.TurnEvents = make([]MonsterTurnEvent, 0)

	// Reset per-turn ability usage
	mts.SpecialAbilitiesUsed = make(map[string]int)

	// Clear expired active effects
	newEffects := make([]MonsterActiveEffect, 0)
	for _, effect := range mts.ActiveEffects {
		if effect.ExpiresOn != "end_of_gm_turn" {
			newEffects = append(newEffects, effect)
		}
	}
	mts.ActiveEffects = newEffects

	// Update timestamps
	mts.TurnStartedAt = time.Now()
	mts.LastActivityAt = time.Now()
}

// GetTurnSummary returns a summary string for logging
func (mts *MonsterTurnState) GetTurnSummary() string {
	if mts.HasMoved && mts.ActionTaken {
		return "moved and acted"
	} else if mts.HasMoved {
		return "moved only"
	} else if mts.ActionTaken {
		return "acted only"
	}
	return "no actions"
}
