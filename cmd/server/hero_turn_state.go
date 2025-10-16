package main

import (
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// HeroTurnState tracks comprehensive turn state for a single hero
type HeroTurnState struct {
	// Identity
	HeroID     string
	PlayerID   string
	TurnNumber int // Which turn in the quest (for historical tracking)

	// Movement Tracking
	MovementDice MovementDiceState
	MovementPath []protocol.TileAddress // All tiles moved this turn

	// Movement/Action State (simplified flag-based model)
	HasMoved    bool            // True once any movement happens (locks into move-first)
	ActionTaken bool            // True once action happens
	TurnFlags   map[string]bool // Generic flags: "can_split_movement", "can_make_extra_attack", etc.

	// Action Tracking
	Action *ActionRecord // The single action taken this turn (nil if not taken)

	// Activities (non-action activities)
	Activities []Activity // Potions used, items passed, etc.

	// Active Effects (from items/abilities waiting to trigger)
	ActiveEffects []ActiveEffect // "Next attack +2 dice", "Next movement doubled", etc.

	// Item Usage Tracking (resets each turn)
	ItemUsageThisTurn map[string]int // ItemID -> usage count

	// Per-Location Action Tracking
	LocationActions map[string]*LocationActionHistory // Location key -> actions

	// Turn Event Log (doors opened, etc.)
	TurnEvents []TurnEvent

	// Position Tracking
	TurnStartPosition protocol.TileAddress
	CurrentPosition   protocol.TileAddress

	// Timestamps
	TurnStartedAt  time.Time
	LastActivityAt time.Time
}

// MovementDiceState tracks movement dice roll and usage
type MovementDiceState struct {
	Rolled            bool
	DiceResults       []int // Individual die results (e.g., [2, 3, 4])
	TotalMovement     int   // Sum of dice (e.g., 9)
	MovementUsed      int   // How much movement consumed so far
	MovementRemaining int   // TotalMovement - MovementUsed
}

// ActionRecord represents the main action taken this turn
type ActionRecord struct {
	ActionType     string                 // "attack", "cast_spell", "search_treasure", "search_trap", etc.
	TargetID       string                 // Entity ID if targeting something
	TargetPosition *protocol.TileAddress  // Position if targeting a location
	LocationKey    string                 // Room ID or corridor segment key
	Success        bool                   // Whether action succeeded
	Details        map[string]interface{} // Action-specific data (damage, spell name, search results, etc.)
	Timestamp      time.Time
}

// Activity represents non-action activities (potions, item passing, door opening)
type Activity struct {
	Type      string // "use_item", "pass_item", "open_door", "close_door"
	ItemID    string // For item-related activities
	ItemName  string
	Target    string // For pass_item: recipient hero ID; for doors: door ID
	Context   string // "on_turn", "during_monster_attack", "during_hero_defense"
	Details   map[string]interface{}
	Timestamp time.Time
}

// ActiveEffect represents a pending bonus from items/abilities
type ActiveEffect struct {
	Source     string // "potion_of_strength", "heroic_brew", "tidal_surge"
	EffectType string // "bonus_attack_dice", "extra_attack", "bonus_movement", "can_split_movement"
	Value      int    // e.g., 2 for "+2 dice"
	Trigger    string // "next_attack", "next_defend", "next_movement", "immediate"
	ExpiresOn  string // "end_of_turn", "after_trigger", "end_of_quest"
	Applied    bool   // Whether the effect has been consumed
	CreatedAt  time.Time
}

// LocationActionHistory tracks all actions performed at a specific location
type LocationActionHistory struct {
	LocationKey    string                    // "room-17" or "corridor-seg-3"
	LocationType   string                    // "room" or "corridor"
	SearchesByHero map[string]*SearchHistory // HeroID -> their searches in this location
	FirstEntered   time.Time
}

// SearchHistory tracks a hero's searches at a specific location
type SearchHistory struct {
	TreasureSearches   []SearchRecord // Each hero limited to 1 per room
	TrapSearches       []SearchRecord // Multiple allowed
	SecretDoorSearches []SearchRecord // Multiple allowed
}

// SearchRecord represents a single search action
type SearchRecord struct {
	SearchType string // "treasure", "trap", "secret_door"
	Success    bool
	FoundItems []string             // Item IDs found
	Position   protocol.TileAddress // Where the search occurred
	Timestamp  time.Time
}

// TurnEvent represents a logged event that doesn't affect game rules
type TurnEvent struct {
	EventType string // "door_opened", "door_closed", "monster_spawned", "gm_narration"
	EntityID  string // Door ID, monster ID, etc.
	Details   map[string]interface{}
	Timestamp time.Time
}

// NewHeroTurnState creates a new hero turn state
func NewHeroTurnState(heroID, playerID string, turnNumber int, startPosition protocol.TileAddress) *HeroTurnState {
	return &HeroTurnState{
		HeroID:            heroID,
		PlayerID:          playerID,
		TurnNumber:        turnNumber,
		TurnFlags:         make(map[string]bool),
		ItemUsageThisTurn: make(map[string]int),
		LocationActions:   make(map[string]*LocationActionHistory),
		Activities:        make([]Activity, 0),
		ActiveEffects:     make([]ActiveEffect, 0),
		TurnEvents:        make([]TurnEvent, 0),
		MovementPath:      make([]protocol.TileAddress, 0),
		TurnStartPosition: startPosition,
		CurrentPosition:   startPosition,
		TurnStartedAt:     time.Now(),
		LastActivityAt:    time.Now(),
	}
}

// CanMove validates whether the hero can move based on turn state
func (hts *HeroTurnState) CanMove() (bool, string) {
	if !hts.MovementDice.Rolled {
		return false, "must roll movement dice first"
	}
	if hts.MovementDice.MovementRemaining <= 0 {
		return false, "no movement remaining"
	}

	// If both action and movement are done, no more movement allowed
	// Exception: can_split_movement ability (e.g., from special items)
	if hts.HasMoved && hts.ActionTaken {
		if hts.TurnFlags["can_split_movement"] {
			return true, ""
		}
		return false, "cannot move after both moving and taking an action"
	}

	// If action taken but no movement yet, can move once (act-first strategy)
	if hts.ActionTaken && !hts.HasMoved {
		return true, ""
	}

	// If movement started but no action, can continue moving
	if hts.HasMoved && !hts.ActionTaken {
		return true, ""
	}

	// Neither done yet, can move
	return true, ""
}

// CanTakeAction validates whether the hero can take an action
func (hts *HeroTurnState) CanTakeAction() (bool, string) {
	if !hts.MovementDice.Rolled {
		return false, "must roll movement dice first"
	}

	if hts.ActionTaken {
		// Special case: extra attacks from Heroic Brew, etc.
		if hts.TurnFlags["can_make_extra_attack"] {
			return true, ""
		}
		return false, "action already taken this turn"
	}

	// Can take action if not taken yet (regardless of movement state)
	return true, ""
}

// GetTurnStrategy returns the current turn strategy state
func (hts *HeroTurnState) GetTurnStrategy() string {
	if !hts.HasMoved && !hts.ActionTaken {
		return "choose" // Can choose either move or action
	}
	if hts.HasMoved && !hts.ActionTaken {
		return "move_first" // Locked into moving first
	}
	if !hts.HasMoved && hts.ActionTaken {
		return "act_first" // Locked into acting first
	}
	return "complete" // Both done
}

// RollMovementDice records a movement dice roll
func (hts *HeroTurnState) RollMovementDice(diceResults []int) error {
	if hts.MovementDice.Rolled {
		return &GameError{Code: "already_rolled", Message: "movement dice already rolled this turn"}
	}

	total := 0
	for _, result := range diceResults {
		total += result
	}

	hts.MovementDice = MovementDiceState{
		Rolled:            true,
		DiceResults:       diceResults,
		TotalMovement:     total,
		MovementUsed:      0,
		MovementRemaining: total,
	}

	hts.LastActivityAt = time.Now()
	return nil
}

// RecordMovement records a movement step
func (hts *HeroTurnState) RecordMovement(to protocol.TileAddress) error {
	canMove, reason := hts.CanMove()
	if !canMove {
		return &GameError{Code: "cannot_move", Message: reason}
	}

	// Mark that movement has started (locks into move-first if action not taken)
	hts.HasMoved = true

	// Add to movement path
	hts.MovementPath = append(hts.MovementPath, to)
	hts.CurrentPosition = to

	// Consume movement
	hts.MovementDice.MovementUsed++
	hts.MovementDice.MovementRemaining--

	hts.LastActivityAt = time.Now()
	return nil
}

// RecordAction records the main action for this turn
func (hts *HeroTurnState) RecordAction(action ActionRecord) error {
	canAct, reason := hts.CanTakeAction()
	if !canAct {
		return &GameError{Code: "cannot_act", Message: reason}
	}

	action.Timestamp = time.Now()
	hts.Action = &action
	hts.ActionTaken = true
	hts.LastActivityAt = time.Now()

	return nil
}

// RecordActivity records a non-action activity
func (hts *HeroTurnState) RecordActivity(activity Activity) {
	activity.Timestamp = time.Now()
	hts.Activities = append(hts.Activities, activity)
	hts.LastActivityAt = time.Now()

	// Track item usage
	if activity.ItemID != "" {
		hts.ItemUsageThisTurn[activity.ItemID]++
	}
}

// AddActiveEffect adds a pending effect from an item or ability
func (hts *HeroTurnState) AddActiveEffect(effect ActiveEffect) {
	effect.CreatedAt = time.Now()
	effect.Applied = false
	hts.ActiveEffects = append(hts.ActiveEffects, effect)

	// Set turn flags for certain effects
	switch effect.EffectType {
	case "can_split_movement":
		hts.TurnFlags["can_split_movement"] = true
	case "extra_attack":
		hts.TurnFlags["can_make_extra_attack"] = true
	}
}

// TriggerEffects finds and marks effects with matching trigger, returns them
func (hts *HeroTurnState) TriggerEffects(trigger string) []ActiveEffect {
	triggered := make([]ActiveEffect, 0)

	for i := range hts.ActiveEffects {
		effect := &hts.ActiveEffects[i]
		if effect.Trigger == trigger && !effect.Applied {
			effect.Applied = true
			triggered = append(triggered, *effect)
		}
	}

	return triggered
}

// CanSearchTreasure checks if hero can search for treasure at this location
func (hts *HeroTurnState) CanSearchTreasure(locationKey string) (bool, string) {
	// Get location history
	locHistory := hts.LocationActions[locationKey]
	if locHistory == nil {
		return true, "" // First search at this location
	}

	// Get this hero's search history at this location
	searchHistory := locHistory.SearchesByHero[hts.HeroID]
	if searchHistory == nil {
		return true, "" // Hero hasn't searched here yet
	}

	// Check if treasure search already done
	if len(searchHistory.TreasureSearches) > 0 {
		return false, "already searched for treasure at this location"
	}

	return true, ""
}

// RecordSearch records a search action at a location
func (hts *HeroTurnState) RecordSearch(searchType string, locationKey string, locationTyp string, position protocol.TileAddress, success bool, foundItems []string) {
	// Ensure location history exists
	if hts.LocationActions[locationKey] == nil {
		hts.LocationActions[locationKey] = &LocationActionHistory{
			LocationKey:    locationKey,
			LocationType:   locationTyp,
			SearchesByHero: make(map[string]*SearchHistory),
			FirstEntered:   time.Now(),
		}
	}

	locHistory := hts.LocationActions[locationKey]

	// Ensure this hero's search history exists
	if locHistory.SearchesByHero[hts.HeroID] == nil {
		locHistory.SearchesByHero[hts.HeroID] = &SearchHistory{
			TreasureSearches:   make([]SearchRecord, 0),
			TrapSearches:       make([]SearchRecord, 0),
			SecretDoorSearches: make([]SearchRecord, 0),
		}
	}

	searchHistory := locHistory.SearchesByHero[hts.HeroID]

	// Create search record
	record := SearchRecord{
		SearchType: searchType,
		Success:    success,
		FoundItems: foundItems,
		Position:   position,
		Timestamp:  time.Now(),
	}

	// Add to appropriate list
	switch searchType {
	case "treasure":
		searchHistory.TreasureSearches = append(searchHistory.TreasureSearches, record)
	case "trap":
		searchHistory.TrapSearches = append(searchHistory.TrapSearches, record)
	case "secret_door":
		searchHistory.SecretDoorSearches = append(searchHistory.SecretDoorSearches, record)
	}
}

// RecordTurnEvent logs a turn event
func (hts *HeroTurnState) RecordTurnEvent(eventType string, entityID string, details map[string]interface{}) {
	event := TurnEvent{
		EventType: eventType,
		EntityID:  entityID,
		Details:   details,
		Timestamp: time.Now(),
	}
	hts.TurnEvents = append(hts.TurnEvents, event)
}

// ResetForNewTurn resets state for a new turn (called when turn ends)
func (hts *HeroTurnState) ResetForNewTurn(newTurnNumber int) {
	hts.TurnNumber = newTurnNumber
	hts.TurnStartPosition = hts.CurrentPosition

	// Reset movement
	hts.MovementDice = MovementDiceState{}
	hts.MovementPath = make([]protocol.TileAddress, 0)
	hts.HasMoved = false

	// Reset action
	hts.Action = nil
	hts.ActionTaken = false

	// Reset turn flags (but keep quest-long flags)
	newFlags := make(map[string]bool)
	for key, value := range hts.TurnFlags {
		// Keep quest-long flags (add more as needed)
		if key == "quest_ability_used" {
			newFlags[key] = value
		}
	}
	hts.TurnFlags = newFlags

	// Reset activities and events
	hts.Activities = make([]Activity, 0)
	hts.TurnEvents = make([]TurnEvent, 0)

	// Reset item usage
	hts.ItemUsageThisTurn = make(map[string]int)

	// Clear expired active effects
	newEffects := make([]ActiveEffect, 0)
	for _, effect := range hts.ActiveEffects {
		if effect.ExpiresOn != "end_of_turn" {
			newEffects = append(newEffects, effect)
		}
	}
	hts.ActiveEffects = newEffects

	// Update timestamps
	hts.TurnStartedAt = time.Now()
	hts.LastActivityAt = time.Now()
}
