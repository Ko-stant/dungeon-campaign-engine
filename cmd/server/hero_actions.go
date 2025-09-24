package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// HeroAction represents the 6 core HeroQuest actions
type HeroAction string

const (
	AttackAction         HeroAction = "attack"
	CastSpellAction      HeroAction = "cast_spell"
	SearchTreasureAction HeroAction = "search_treasure"
	SearchTrapsAction    HeroAction = "search_traps"
	SearchSecretAction   HeroAction = "search_secret"
	DisarmTrapAction     HeroAction = "disarm_trap"
)

// InstantAction represents actions that don't consume the hero's main action
type InstantAction string

const (
	OpenDoorInstant    InstantAction = "open_door"
	UsePotionInstant   InstantAction = "use_potion"
	UseItemInstant     InstantAction = "use_item"
	TradeItemInstant   InstantAction = "trade_item"
	PassTurnInstant    InstantAction = "pass_turn"
	RollMovementInstant InstantAction = "roll_movement"
)

// MovementAction represents the special movement action (once per turn, before or after main action)
type MovementAction string

const (
	MoveBeforeAction MovementAction = "move_before"
	MoveAfterAction  MovementAction = "move_after"
)

// ActionRequest represents a request to perform a main action (consumes the turn action)
type ActionRequest struct {
	PlayerID   string         `json:"playerId"`
	EntityID   string         `json:"entityId"`
	Action     HeroAction     `json:"action"`
	Parameters map[string]any `json:"parameters"`
}

// InstantActionRequest represents a request for an instant action (doesn't consume turn action)
type InstantActionRequest struct {
	PlayerID   string         `json:"playerId"`
	EntityID   string         `json:"entityId"`
	Action     InstantAction  `json:"action"`
	Parameters map[string]any `json:"parameters"`
}

// MovementRequest represents a request for movement (once per turn, before or after main action)
type MovementRequest struct {
	PlayerID   string         `json:"playerId"`
	EntityID   string         `json:"entityId"`
	Action     MovementAction `json:"action"`
	Parameters map[string]any `json:"parameters"`
}

// ActionResult contains the results of performing an action
type ActionResult struct {
	Success        bool          `json:"success"`
	Action         HeroAction    `json:"action"`
	PlayerID       string        `json:"playerId"`
	EntityID       string        `json:"entityId"`
	AttackRolls    []DiceRoll    `json:"attackRolls,omitempty"`   // Hero's attack dice
	DefenseRolls   []DiceRoll    `json:"defenseRolls,omitempty"`  // Monster's defense dice
	SearchRolls    []DiceRoll    `json:"searchRolls,omitempty"`   // Search action dice
	MovementRolls  []DiceRoll    `json:"movementRolls,omitempty"` // Movement dice rolls
	Damage         int           `json:"damage,omitempty"`
	ItemsFound     []Item        `json:"itemsFound,omitempty"`
	SecretRevealed *SecretDoor   `json:"secretRevealed,omitempty"`
	SpellEffect    *SpellEffect  `json:"spellEffect,omitempty"`
	Message        string        `json:"message"`
	StateChanges   []StateChange `json:"stateChanges,omitempty"`
	Timestamp      time.Time     `json:"timestamp"`
}

// DiceRoll represents a single dice roll
type DiceRoll struct {
	Die          Die             `json:"die"`
	Result       int             `json:"result"`
	Type         string          `json:"type"`         // "attack", "defense", "movement", "search"
	CombatResult CombatDiceResult `json:"combatResult"` // For combat dice results
	IsDefended   bool            `json:"isDefended"`   // For attack dice that were blocked
	IsCritical   bool            `json:"isCritical"`   // For special results
}

// CombatDiceResult represents the result of a combat die
type CombatDiceResult string

const (
	Skull        CombatDiceResult = "skull"         // Attack success (appears on 3 faces)
	WhiteShield  CombatDiceResult = "white_shield"  // Defense success (appears on 2 faces)
	BlackShield  CombatDiceResult = "black_shield"  // Defense success (appears on 1 face)
	CombatMiss   CombatDiceResult = "miss"          // No effect (never appears in HeroQuest)
)

// GetCombatResult converts a dice roll result to combat result
func GetCombatResult(result int, isAttack bool) CombatDiceResult {
	if isAttack {
		// Attack dice: 3 skulls (results 4, 5, 6), 3 misses (results 1, 2, 3)
		if result >= 4 {
			return Skull
		}
		return CombatMiss
	} else {
		// Defense dice: 2 white shields (results 4, 5), 1 black shield (result 6), 3 misses (results 1, 2, 3)
		switch result {
		case 6:
			return BlackShield
		case 4, 5:
			return WhiteShield
		default:
			return CombatMiss
		}
	}
}

// Die represents different types of dice
type Die string

const (
	CombatDie   Die = "combat"   // White/red combat dice
	MovementDie Die = "movement" // Blue movement dice
	SearchDie   Die = "search"   // Special search dice
)

// Item represents treasure or equipment
type Item struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        ItemType `json:"type"`
	Description string   `json:"description"`
	Value       int      `json:"value"`
	Effects     []Effect `json:"effects,omitempty"`
}

type ItemType string

const (
	Weapon ItemType = "weapon"
	Armor  ItemType = "armor"
	Potion ItemType = "potion"
	Gold   ItemType = "gold"
	Spell  ItemType = "spell"
)

// SecretDoor represents a hidden door revealed by searching
type SecretDoor struct {
	ID       string               `json:"id"`
	Position protocol.TileAddress `json:"position"`
	LeadsTo  string               `json:"leadsTo"`
}

// SpellEffect represents the effect of casting a spell
type SpellEffect struct {
	SpellID    string   `json:"spellId"`
	Name       string   `json:"name"`
	Duration   int      `json:"duration"` // Turns remaining
	Effects    []Effect `json:"effects"`
	TargetType string   `json:"targetType"` // "self", "ally", "enemy", "area"
	TargetID   string   `json:"targetId,omitempty"`
}

// Effect represents a game effect
type Effect struct {
	Type        string `json:"type"` // "damage_bonus", "defense_bonus", "movement_bonus"
	Value       int    `json:"value"`
	Description string `json:"description"`
}

// HeroActionSystem handles hero action processing
type HeroActionSystem struct {
	gameState         *GameState
	turnManager       *TurnManager
	diceSystem        *DiceSystem
	broadcaster       Broadcaster
	logger            Logger
	debugSystem       *DebugSystem
	movementValidator MovementValidator
	monsterSystem     *MonsterSystem
	quest             *geometry.QuestDefinition
}

// NewHeroActionSystem creates a new hero action system
func NewHeroActionSystem(gameState *GameState, turnManager *TurnManager, broadcaster Broadcaster, logger Logger, debugSystem *DebugSystem) *HeroActionSystem {
	return &HeroActionSystem{
		gameState:         gameState,
		turnManager:       turnManager,
		diceSystem:        NewDiceSystem(debugSystem),
		broadcaster:       broadcaster,
		logger:            logger,
		debugSystem:       debugSystem,
		movementValidator: NewMovementValidator(logger), // Default validator without systems
	}
}

// SetMovementValidator sets the movement validator with systems
func (has *HeroActionSystem) SetMovementValidator(validator MovementValidator) {
	has.movementValidator = validator
}

// SetMonsterSystem sets the monster system for combat interactions
func (has *HeroActionSystem) SetMonsterSystem(monsterSystem *MonsterSystem) {
	has.monsterSystem = monsterSystem
}

// SetQuest sets the quest definition for visibility calculations
func (has *HeroActionSystem) SetQuest(quest *geometry.QuestDefinition) {
	has.quest = quest
}

// ProcessAction processes a hero action request
func (has *HeroActionSystem) ProcessAction(request ActionRequest) (*ActionResult, error) {
	// Validate player can act
	if !has.turnManager.CanPlayerAct(request.PlayerID) {
		return nil, fmt.Errorf("player %s cannot act right now", request.PlayerID)
	}

	// Validate entity belongs to player
	player := has.turnManager.GetCurrentPlayer()
	if player == nil || player.EntityID != request.EntityID {
		return nil, fmt.Errorf("entity %s does not belong to player %s", request.EntityID, request.PlayerID)
	}

	result := &ActionResult{
		Action:    request.Action,
		PlayerID:  request.PlayerID,
		EntityID:  request.EntityID,
		Timestamp: time.Now(),
	}

	// Process specific action
	switch request.Action {
	case AttackAction:
		return has.processAttack(request, result)
	case CastSpellAction:
		return has.processCastSpell(request, result)
	case SearchTreasureAction:
		return has.processSearchTreasure(request, result)
	case SearchTrapsAction:
		return has.processSearchTraps(request, result)
	case SearchSecretAction:
		return has.processSearchSecret(request, result)
	case DisarmTrapAction:
		return has.processDisarmTrap(request, result)
	default:
		return nil, fmt.Errorf("unknown action: %s", request.Action)
	}
}

// Search for treasure action
func (has *HeroActionSystem) processSearchTreasure(request ActionRequest, result *ActionResult) (*ActionResult, error) {
	// Consume action
	if err := has.turnManager.ConsumeAction(); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	// TODO: Check if room is uninhabited by monsters
	// TODO: Check if room has already been searched for treasure

	// Roll search dice for treasure
	searchRolls := has.diceSystem.RollDice(SearchDie, 1, "search_treasure")
	searchResult := searchRolls[0].Result

	result.SearchRolls = searchRolls
	result.Success = true

	switch searchResult {
	case 1, 2:
		result.Message = "Found nothing"
	case 3, 4:
		// Found gold
		treasure := Item{
			ID:    fmt.Sprintf("gold_%d", time.Now().Unix()),
			Name:  "Gold Coins",
			Type:  Gold,
			Value: searchResult * 50, // 150 or 200 gold
		}
		result.ItemsFound = []Item{treasure}
		result.Message = fmt.Sprintf("Found %d gold coins!", treasure.Value)
	case 5, 6:
		// Found equipment or artifact
		equipment := Item{
			ID:   fmt.Sprintf("equipment_%d", time.Now().Unix()),
			Name: "Equipment",
			Type: Weapon, // Could be weapon, armor, etc.
		}
		result.ItemsFound = []Item{equipment}
		result.Message = "Found equipment!"
	}

	has.logger.Printf("Player %s searched for treasure and rolled %d: %s", request.PlayerID, searchResult, result.Message)
	return result, nil
}

// Attack action with dice rolling
func (has *HeroActionSystem) processAttack(request ActionRequest, result *ActionResult) (*ActionResult, error) {
	targetID, ok := request.Parameters["targetId"].(string)
	if !ok {
		result.Success = false
		result.Message = "No target specified"
		return result, fmt.Errorf("missing targetId parameter")
	}

	// Consume action
	if err := has.turnManager.ConsumeAction(); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	// Get attacking player's character
	player := has.turnManager.GetCurrentPlayer()
	if player == nil || player.Character == nil {
		result.Success = false
		result.Message = "No active player character"
		return result, fmt.Errorf("no active player character")
	}

	// Get target monster
	if has.monsterSystem == nil {
		result.Success = false
		result.Message = "Monster system not available"
		return result, fmt.Errorf("monster system not initialized")
	}

	targetMonster, err := has.monsterSystem.GetMonsterByID(targetID)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Target not found: %s", err.Error())
		return result, err
	}

	if !targetMonster.IsAlive {
		result.Success = false
		result.Message = "Target is already dead"
		return result, fmt.Errorf("target monster %s is already dead", targetID)
	}

	// Roll attack dice based on hero's effective attack dice
	attackDice := player.Character.GetEffectiveAttackDice()
	attackRolls := has.diceSystem.RollAttackDice(attackDice)

	// Roll defense dice for monster
	defenseRolls := has.diceSystem.RollDefenseDice(targetMonster.DefenseDice)

	// Calculate damage
	damage := CalculateCombatDamage(attackRolls, defenseRolls)

	// Apply damage to monster
	_, isDead, err := has.monsterSystem.ApplyDamageToMonster(targetID, damage)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to apply damage: %s", err.Error())
		return result, err
	}

	// Set separated dice rolls
	result.AttackRolls = attackRolls
	result.DefenseRolls = defenseRolls
	result.Damage = damage
	result.Success = true

	// Create appropriate message
	if damage > 0 {
		if isDead {
			result.Message = fmt.Sprintf("Dealt %d damage to %s (%s) - Monster killed!", damage, targetMonster.Type, targetID)
		} else {
			result.Message = fmt.Sprintf("Dealt %d damage to %s (%s) - %d body points remaining", damage, targetMonster.Type, targetID, targetMonster.Body)
		}
	} else {
		result.Message = fmt.Sprintf("Attack blocked! %s (%s) took no damage", targetMonster.Type, targetID)
	}

	has.logger.Printf("Player %s (%s) attacked %s (%s): %d skulls vs %d shields = %d damage (Monster %s: %d/%d body)",
		player.Name, player.Class, targetMonster.Type, targetID,
		countSkulls(attackRolls), countShields(defenseRolls), damage,
		targetMonster.Type, targetMonster.Body, targetMonster.MaxBody)

	// Broadcast entity update for hero to ensure frontend keeps track of position
	has.gameState.Lock.Lock()
	heroTile, exists := has.gameState.Entities[request.EntityID]
	has.gameState.Lock.Unlock()

	if exists {
		has.broadcaster.BroadcastEvent("EntityUpdated", protocol.EntityUpdated{
			ID:   request.EntityID,
			Tile: heroTile,
		})
	}

	// Auto-restore actions for testing when debug mode is enabled
	if has.debugSystem != nil && has.debugSystem.config.Enabled {
		has.turnManager.RestoreActions()
		has.logger.Printf("DEBUG: Auto-restored actions after attack for testing")
	}

	return result, nil
}

// Helper functions for combat logging
func countSkulls(rolls []DiceRoll) int {
	count := 0
	for _, roll := range rolls {
		if roll.CombatResult == Skull {
			count++
		}
	}
	return count
}

func countShields(rolls []DiceRoll) int {
	count := 0
	for _, roll := range rolls {
		if roll.CombatResult == WhiteShield || roll.CombatResult == BlackShield {
			count++
		}
	}
	return count
}

// Search for traps action
func (has *HeroActionSystem) processSearchTraps(request ActionRequest, result *ActionResult) (*ActionResult, error) {
	// Consume action
	if err := has.turnManager.ConsumeAction(); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	// TODO: Check if room/corridor is uninhabited by monsters

	// Roll search dice for traps
	searchRolls := has.diceSystem.RollDice(SearchDie, 1, "search_traps")
	searchResult := searchRolls[0].Result

	result.SearchRolls = searchRolls
	result.Success = true

	if searchResult >= 5 { // Success on 5-6
		// TODO: Reveal trap in current room/corridor
		result.Message = "Found a trap!"
		has.logger.Printf("Player %s found a trap with roll %d", request.PlayerID, searchResult)
	} else {
		result.Message = "No traps found"
		has.logger.Printf("Player %s searched for traps and rolled %d: no traps found", request.PlayerID, searchResult)
	}

	return result, nil
}

// Search for secret doors action
func (has *HeroActionSystem) processSearchSecret(request ActionRequest, result *ActionResult) (*ActionResult, error) {
	// Consume action
	if err := has.turnManager.ConsumeAction(); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	// TODO: Check if room/corridor is uninhabited by monsters

	// Roll search dice for secret doors
	searchRolls := has.diceSystem.RollDice(SearchDie, 1, "search_secret")
	searchResult := searchRolls[0].Result

	result.SearchRolls = searchRolls
	result.Success = true

	if searchResult == 6 { // Success only on 6
		// Found secret door
		secret := &SecretDoor{
			ID:       fmt.Sprintf("secret_%d", time.Now().Unix()),
			Position: has.gameState.Entities[request.EntityID],
			LeadsTo:  "Unknown",
		}
		result.SecretRevealed = secret
		result.Message = "Found a secret door!"
		has.logger.Printf("Player %s found a secret door with roll %d", request.PlayerID, searchResult)
	} else {
		result.Message = "No secret doors found"
		has.logger.Printf("Player %s searched for secret doors and rolled %d: no secrets found", request.PlayerID, searchResult)
	}

	return result, nil
}

// Cast spell action
func (has *HeroActionSystem) processCastSpell(request ActionRequest, result *ActionResult) (*ActionResult, error) {
	spellID, ok := request.Parameters["spellId"].(string)
	if !ok {
		result.Success = false
		result.Message = "No spell specified"
		return result, fmt.Errorf("missing spellId parameter")
	}

	// Consume action
	if err := has.turnManager.ConsumeAction(); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	// TODO: Implement spell system
	// For now, simple placeholder
	spell := &SpellEffect{
		SpellID:    spellID,
		Name:       "Test Spell",
		Duration:   3,
		TargetType: "self",
		TargetID:   request.EntityID,
	}

	result.Success = true
	result.SpellEffect = spell
	result.Message = fmt.Sprintf("Cast %s", spell.Name)

	has.logger.Printf("Player %s cast spell %s", request.PlayerID, spellID)
	return result, nil
}

// Disarm trap action
func (has *HeroActionSystem) processDisarmTrap(request ActionRequest, result *ActionResult) (*ActionResult, error) {
	// Consume action
	if err := has.turnManager.ConsumeAction(); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	trapID, ok := request.Parameters["trapId"].(string)
	if !ok {
		result.Success = false
		result.Message = "No trap specified"
		return result, fmt.Errorf("missing trapId parameter")
	}

	// TODO: Check if player has tool kit (except for Dwarf)
	// TODO: Check if trap exists and is known
	// TODO: Implement actual trap disarming mechanics

	// For now, simple success/failure
	disarmRolls := has.diceSystem.RollDice(SearchDie, 1, "disarm_trap")
	disarmResult := disarmRolls[0].Result

	result.SearchRolls = disarmRolls
	result.Success = disarmResult >= 4 // Success on 4-6

	if result.Success {
		result.Message = fmt.Sprintf("Successfully disarmed trap %s", trapID)
		has.logger.Printf("Player %s disarmed trap %s with roll %d", request.PlayerID, trapID, disarmResult)
	} else {
		result.Message = "Failed to disarm trap"
		has.logger.Printf("Player %s failed to disarm trap %s with roll %d", request.PlayerID, trapID, disarmResult)
		// TODO: Trigger trap effect
	}

	return result, nil
}

// ProcessInstantAction processes instant actions that don't consume the main action
func (has *HeroActionSystem) ProcessInstantAction(request InstantActionRequest) (*ActionResult, error) {
	// Validate player can act (but don't consume action)
	if !has.turnManager.CanPlayerAct(request.PlayerID) {
		return nil, fmt.Errorf("player %s cannot act right now", request.PlayerID)
	}

	// Validate entity belongs to player
	player := has.turnManager.GetCurrentPlayer()
	if player == nil || player.EntityID != request.EntityID {
		return nil, fmt.Errorf("entity %s does not belong to player %s", request.EntityID, request.PlayerID)
	}

	result := &ActionResult{
		Action:    HeroAction(request.Action), // Cast to HeroAction for compatibility
		PlayerID:  request.PlayerID,
		EntityID:  request.EntityID,
		Timestamp: time.Now(),
	}

	// Process specific instant action
	switch request.Action {
	case OpenDoorInstant:
		return has.processOpenDoor(request, result)
	case UsePotionInstant:
		return has.processUsePotion(request, result)
	case UseItemInstant:
		return has.processUseItem(request, result)
	case TradeItemInstant:
		return has.processTradeItem(request, result)
	case PassTurnInstant:
		return has.processPassTurn(request, result)
	case RollMovementInstant:
		return has.processRollMovement(request, result)
	default:
		return nil, fmt.Errorf("unknown instant action: %s", request.Action)
	}
}

// ProcessMovement processes movement requests (once per turn, before or after main action)
func (has *HeroActionSystem) ProcessMovement(request MovementRequest) (*ActionResult, error) {
	// Validate player can move
	if !has.turnManager.CanMove() {
		return nil, fmt.Errorf("player cannot move right now")
	}

	// Validate entity belongs to player
	player := has.turnManager.GetCurrentPlayer()
	if player == nil || player.EntityID != request.EntityID {
		return nil, fmt.Errorf("entity %s does not belong to player %s", request.EntityID, request.PlayerID)
	}

	result := &ActionResult{
		Action:    HeroAction("movement"), // Special action type
		PlayerID:  request.PlayerID,
		EntityID:  request.EntityID,
		Timestamp: time.Now(),
	}

	return has.processMovement(request, result)
}

// Instant action processors

func (has *HeroActionSystem) processMovement(request MovementRequest, result *ActionResult) (*ActionResult, error) {
	// Check if we're switching movement action types mid-turn
	turnState := has.turnManager.GetTurnState()
	requestAction := string(request.Action)

	if turnState.MovementStarted && turnState.MovementAction != "" && turnState.MovementAction != requestAction {
		result.Success = false
		result.Message = "Player cannot move right now"
		return result, fmt.Errorf("player cannot move right now")
	}

	dx, ok1 := request.Parameters["dx"].(float64)
	dy, ok2 := request.Parameters["dy"].(float64)

	if !ok1 || !ok2 {
		result.Success = false
		result.Message = "Invalid movement parameters"
		return result, fmt.Errorf("missing or invalid dx/dy parameters")
	}

	// Calculate movement distance
	distance := int(abs(dx) + abs(dy))
	if distance == 0 {
		result.Success = false
		result.Message = "No movement specified"
		return result, fmt.Errorf("no movement")
	}

	// Validate movement first BEFORE consuming movement points
	newTile, err := has.movementValidator.ValidateMove(has.gameState, request.EntityID, int(dx), int(dy))
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Movement blocked: %s", err.Error())
		return result, err
	}

	// Only consume movement points if the move is valid
	if err := has.turnManager.ConsumeMovement(distance, requestAction); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	// Update entity position
	has.gameState.Lock.Lock()
	has.gameState.Entities[request.EntityID] = *newTile
	has.gameState.Lock.Unlock()

	// Broadcast entity position update
	has.broadcaster.BroadcastEvent("EntityUpdated", protocol.EntityUpdated{
		ID:   request.EntityID,
		Tile: *newTile,
	})

	// Check for newly visible doors after movement (only if game state is fully initialized)
	hero := *newTile
	heroIdx := hero.Y*has.gameState.Segment.Width + hero.X
	if len(has.gameState.RegionMap.TileRegionIDs) > heroIdx {
		if newlyVisibleDoors := checkForNewlyVisibleDoors(has.gameState, hero); len(newlyVisibleDoors) > 0 {
			has.logger.Printf("Movement revealed %d new doors", len(newlyVisibleDoors))
			has.broadcaster.BroadcastEvent("DoorsVisible", protocol.DoorsVisible{Doors: newlyVisibleDoors})
		}
	}

	// Check for newly visible blocking walls after movement
	if has.quest != nil {
		_, newlyVisibleWalls := getVisibleBlockingWalls(has.gameState, hero, has.quest)
		if len(newlyVisibleWalls) > 0 {
			has.logger.Printf("Movement revealed %d new blocking walls", len(newlyVisibleWalls))
			has.broadcaster.BroadcastEvent("BlockingWallsVisible", protocol.BlockingWallsVisible{BlockingWalls: newlyVisibleWalls})
		}
	}

	result.Success = true
	result.Message = fmt.Sprintf("Moved to (%d,%d)", newTile.X, newTile.Y)
	return result, nil
}

func (has *HeroActionSystem) processOpenDoor(request InstantActionRequest, result *ActionResult) (*ActionResult, error) {
	doorID, ok := request.Parameters["doorId"].(string)
	if !ok {
		result.Success = false
		result.Message = "No door specified"
		return result, fmt.Errorf("missing doorId parameter")
	}

	// TODO: Implement door opening logic
	result.Success = true
	result.Message = fmt.Sprintf("Opened door %s", doorID)
	has.logger.Printf("Player %s opened door %s", request.PlayerID, doorID)
	return result, nil
}

func (has *HeroActionSystem) processUsePotion(request InstantActionRequest, result *ActionResult) (*ActionResult, error) {
	potionID, ok := request.Parameters["potionId"].(string)
	if !ok {
		result.Success = false
		result.Message = "No potion specified"
		return result, fmt.Errorf("missing potionId parameter")
	}

	// TODO: Implement potion usage logic
	result.Success = true
	result.Message = fmt.Sprintf("Used potion %s", potionID)
	has.logger.Printf("Player %s used potion %s", request.PlayerID, potionID)
	return result, nil
}

func (has *HeroActionSystem) processUseItem(request InstantActionRequest, result *ActionResult) (*ActionResult, error) {
	itemID, ok := request.Parameters["itemId"].(string)
	if !ok {
		result.Success = false
		result.Message = "No item specified"
		return result, fmt.Errorf("missing itemId parameter")
	}

	// TODO: Implement item usage logic
	result.Success = true
	result.Message = fmt.Sprintf("Used item %s", itemID)
	has.logger.Printf("Player %s used item %s", request.PlayerID, itemID)
	return result, nil
}

func (has *HeroActionSystem) processTradeItem(request InstantActionRequest, result *ActionResult) (*ActionResult, error) {
	targetPlayerID, ok1 := request.Parameters["targetPlayerId"].(string)
	itemID, ok2 := request.Parameters["itemId"].(string)

	if !ok1 || !ok2 {
		result.Success = false
		result.Message = "Missing trade parameters"
		return result, fmt.Errorf("missing targetPlayerId or itemId parameter")
	}

	// TODO: Check if players are adjacent
	// TODO: Implement item trading logic
	result.Success = true
	result.Message = fmt.Sprintf("Traded item %s to %s", itemID, targetPlayerID)
	has.logger.Printf("Player %s traded item %s to %s", request.PlayerID, itemID, targetPlayerID)
	return result, nil
}

func (has *HeroActionSystem) processPassTurn(request InstantActionRequest, result *ActionResult) (*ActionResult, error) {
	// Force end turn
	has.turnManager.EndTurn()

	result.Success = true
	result.Message = "Passed turn"
	has.logger.Printf("Player %s passed their turn", request.PlayerID)
	return result, nil
}

func (has *HeroActionSystem) processRollMovement(request InstantActionRequest, result *ActionResult) (*ActionResult, error) {
	// Roll movement dice for the current player
	diceRolls, err := has.turnManager.RollMovementDice()
	if err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	// Convert dice rolls to expected format
	movementRolls := make([]DiceRoll, len(diceRolls))
	totalMovement := 0
	for i, roll := range diceRolls {
		movementRolls[i] = roll
		totalMovement += roll.Result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Rolled movement dice: %d points", totalMovement)
	result.MovementRolls = movementRolls
	has.logger.Printf("Player %s rolled movement dice: total %d points", request.PlayerID, totalMovement)

	return result, nil
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// DiceSystem handles dice rolling with debug overrides
type DiceSystem struct {
	debugSystem *DebugSystem
	random      *rand.Rand
}

func NewDiceSystem(debugSystem *DebugSystem) *DiceSystem {
	return &DiceSystem{
		debugSystem: debugSystem,
		random:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RollDice rolls multiple dice of the same type
func (ds *DiceSystem) RollDice(dieType Die, count int, rollType string) []DiceRoll {
	rolls := make([]DiceRoll, count)

	// Check for debug sequence override first (for multiple dice)
	if ds.debugSystem != nil {
		if overrideResults, exists := ds.debugSystem.GetDiceOverrideSequence(rollType, count); exists {
			for i, result := range overrideResults {
				roll := DiceRoll{
					Die:    dieType,
					Result: result,
					Type:   rollType,
				}
				// Set combat result for combat dice
				if dieType == CombatDie {
					roll.CombatResult = GetCombatResult(result, rollType == "attack")
				}
				rolls[i] = roll
			}

			// Fill any remaining rolls with normal dice if sequence was shorter
			for i := len(overrideResults); i < count; i++ {
				result := ds.random.Intn(6) + 1
				roll := DiceRoll{
					Die:    dieType,
					Result: result,
					Type:   rollType,
				}
				if dieType == CombatDie {
					roll.CombatResult = GetCombatResult(result, rollType == "attack")
				}
				rolls[i] = roll
			}
			return rolls
		}
	}

	// Fall back to individual die overrides or normal rolling
	for i := 0; i < count; i++ {
		// Check for debug override (single die)
		if ds.debugSystem != nil {
			if override, exists := ds.debugSystem.GetDiceOverride(rollType); exists {
				roll := DiceRoll{
					Die:    dieType,
					Result: override,
					Type:   rollType,
				}
				// Set combat result for combat dice
				if dieType == CombatDie {
					roll.CombatResult = GetCombatResult(override, rollType == "attack")
				}
				rolls[i] = roll
				continue
			}
		}

		// Normal dice roll
		result := ds.random.Intn(6) + 1
		roll := DiceRoll{
			Die:    dieType,
			Result: result,
			Type:   rollType,
		}

		// Set combat result for combat dice
		if dieType == CombatDie {
			roll.CombatResult = GetCombatResult(result, rollType == "attack")
		}

		rolls[i] = roll
	}

	return rolls
}

// RollAttackDice rolls attack dice for a character
func (ds *DiceSystem) RollAttackDice(attackDice int) []DiceRoll {
	return ds.RollDice(CombatDie, attackDice, "attack")
}

// RollDefenseDice rolls defense dice for a character
func (ds *DiceSystem) RollDefenseDice(defenseDice int) []DiceRoll {
	return ds.RollDice(CombatDie, defenseDice, "defense")
}

// CalculateCombatDamage calculates damage from attack and defense rolls
// Note: This function assumes monster defense (only black shields count)
// For hero defense, use CalculateHeroCombatDamage instead
func CalculateCombatDamage(attackRolls, defenseRolls []DiceRoll) int {
	skulls := 0
	blackShields := 0

	// Count skulls from attack rolls
	for _, roll := range attackRolls {
		if roll.CombatResult == Skull {
			skulls++
		}
	}

	// Count only black shields from monster defense rolls
	// In HeroQuest, monsters can only defend with black shields (1/6 chance)
	for _, roll := range defenseRolls {
		if roll.CombatResult == BlackShield {
			blackShields++
		}
	}

	// Net damage = skulls - black shields (minimum 0)
	damage := skulls - blackShields
	if damage < 0 {
		damage = 0
	}

	return damage
}

// CalculateHeroCombatDamage calculates damage when a hero is defending
// Heroes can defend with both white and black shields (3/6 chance total)
func CalculateHeroCombatDamage(attackRolls, defenseRolls []DiceRoll) int {
	skulls := 0
	shields := 0

	// Count skulls from attack rolls
	for _, roll := range attackRolls {
		if roll.CombatResult == Skull {
			skulls++
		}
	}

	// Count both white and black shields from hero defense rolls
	for _, roll := range defenseRolls {
		if roll.CombatResult == WhiteShield || roll.CombatResult == BlackShield {
			shields++
		}
	}

	// Net damage = skulls - shields (minimum 0)
	damage := skulls - shields
	if damage < 0 {
		damage = 0
	}

	return damage
}
