package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// TurnType defines who has the current turn
type TurnType string

const (
	HeroTurn       TurnType = "hero"
	GameMasterTurn TurnType = "gamemaster"
)

// TurnPhase defines the current phase within a turn
type TurnPhase string

const (
	// Hero turn phases
	MovementPhase TurnPhase = "movement"
	ActionPhase   TurnPhase = "action"
	EndPhase      TurnPhase = "end"

	// GameMaster turn phases
	MonsterMovementPhase TurnPhase = "monster_movement"
	MonsterActionPhase   TurnPhase = "monster_action"
	EnvironmentPhase     TurnPhase = "environment"
	GMEndPhase           TurnPhase = "gm_end"
)

// MovementSegmentType defines the type of movement segment
type MovementSegmentType string

const (
	ManualMovement  MovementSegmentType = "manual"
	PlannedMovement MovementSegmentType = "planned"
)

// MovementStep represents a single movement step
type MovementStep struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// MovementSegment represents a sequence of related movement steps
type MovementSegment struct {
	Type          MovementSegmentType `json:"type"`
	StartPosition MovementStep        `json:"startPosition"`
	Path          []MovementStep      `json:"path"`
	StartTime     time.Time           `json:"startTime"`
	EndTime       *time.Time          `json:"endTime,omitempty"`
	Executed      bool                `json:"executed"`
	ExecutedTime  *time.Time          `json:"executedTime,omitempty"`
}

// TurnState represents the current state of the turn system
type TurnState struct {
	TurnNumber         int           `json:"turnNumber"`
	CurrentTurn        TurnType      `json:"currentTurn"`
	CurrentPhase       TurnPhase     `json:"currentPhase"`
	ActivePlayerID     string        `json:"activePlayerId"`
	ActionsLeft        int           `json:"actionsLeft"`
	MovementLeft       int           `json:"movementLeft"`
	MovementDiceRolled bool          `json:"movementDiceRolled"` // Whether movement dice have been rolled
	MovementRolls      []int         `json:"movementRolls"`      // The actual dice rolls for movement
	HasMoved           bool          `json:"hasMoved"`           // Whether player has used their movement this turn
	MovementStarted    bool          `json:"movementStarted"`    // Whether movement action has been started
	MovementAction     string        `json:"movementAction"`     // Which movement action was used (move_before/move_after)
	ActionTaken        bool          `json:"actionTaken"`        // Whether player has used their main action
	TurnStartTime      time.Time     `json:"turnStartTime"`
	TurnTimeLimit        time.Duration      `json:"turnTimeLimit"`
	CanEndTurn           bool               `json:"canEndTurn"`
	NextTurnType         TurnType           `json:"nextTurnType"`
	MovementHistory      []MovementSegment  `json:"movementHistory"`      // Complete movement history for this turn
	CurrentSegment       *MovementSegment   `json:"currentSegment"`       // Currently active movement segment
	InitialHeroPosition  *MovementStep      `json:"initialHeroPosition"`  // Hero position at start of turn
}

// TurnManager handles turn progression and validation
type TurnManager struct {
	state            *TurnState
	players          map[string]*Player // Player ID -> Player
	broadcaster      Broadcaster
	logger           Logger
	diceSystem       *DiceSystem       // For rolling movement dice
	turnStateManager *TurnStateManager // Manages per-hero turn state
	lock             sync.RWMutex
}

// Player represents a game player
type Player struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	EntityID  string         `json:"entityId"`
	Class     HeroClass      `json:"class"`
	Character *HeroCharacter `json:"character"`
	IsActive  bool           `json:"isActive"`
}

// NewPlayer creates a new player with initialized hero character
func NewPlayer(id, name, entityID string, class HeroClass) *Player {
	baseStats := GetBaseStatsForClass(class)
	character := &HeroCharacter{
		BaseStats:     baseStats,
		CurrentBody:   baseStats.BodyPoints,
		CurrentMind:   baseStats.MindPoints,
		EquipmentMods: StatMods{}, // No equipment bonuses initially
	}

	return &Player{
		ID:        id,
		Name:      name,
		EntityID:  entityID,
		Class:     class,
		Character: character,
		IsActive:  true,
	}
}

// NewPlayerFromContent creates a new player from hero content definition
// and equips starting items
func NewPlayerFromContent(id, entityID string, heroCard *HeroCard, contentMgr *ContentManager, inventoryMgr *InventoryManager) (*Player, error) {
	character := &HeroCharacter{
		BaseStats:     heroCard.Stats,
		CurrentBody:   heroCard.Stats.BodyPoints,
		CurrentMind:   heroCard.Stats.MindPoints,
		EquipmentMods: StatMods{},
	}

	player := &Player{
		ID:        id,
		Name:      heroCard.Name,
		EntityID:  entityID,
		Class:     HeroClass(heroCard.Class),
		Character: character,
		IsActive:  true,
	}

	// Equip starting equipment
	if inventoryMgr != nil {
		// Add starting weapons
		for _, weaponID := range heroCard.StartingEquipment.Weapons {
			if _, ok := contentMgr.GetEquipmentCard(weaponID); ok {
				if err := inventoryMgr.AddItem(entityID, weaponID); err == nil {
					// Auto-equip starting weapons
					if err := inventoryMgr.EquipItem(entityID, weaponID); err != nil {
						// Log but don't fail if auto-equip doesn't work
						continue
					}
				}
			}
		}

		// Add starting armor
		for _, armorID := range heroCard.StartingEquipment.Armor {
			if _, ok := contentMgr.GetEquipmentCard(armorID); ok {
				if err := inventoryMgr.AddItem(entityID, armorID); err == nil {
					// Auto-equip starting armor
					inventoryMgr.EquipItem(entityID, armorID)
				}
			}
		}

		// Add starting items
		for _, itemID := range heroCard.StartingEquipment.Items {
			if _, ok := contentMgr.GetEquipmentCard(itemID); ok {
				inventoryMgr.AddItem(entityID, itemID)
			}
		}
	}

	return player, nil
}

// HeroClass defines the hero class types
type HeroClass string

const (
	Barbarian HeroClass = "barbarian"
	Dwarf     HeroClass = "dwarf"
	Elf       HeroClass = "elf"
	Wizard    HeroClass = "wizard"
)

// HeroStats represents the base stats for a hero class
type HeroStats struct {
	BodyPoints    int `json:"bodyPoints"`
	MindPoints    int `json:"mindPoints"`
	AttackDice    int `json:"attackDice"`
	DefenseDice   int `json:"defenseDice"`
	MovementDice  int `json:"movementDice"`
}

// GetBaseStatsForClass returns the base stats for each hero class
func GetBaseStatsForClass(class HeroClass) HeroStats {
	switch class {
	case Barbarian:
		return HeroStats{
			BodyPoints:   8,
			MindPoints:   2,
			AttackDice:   3,
			DefenseDice:  2,
			MovementDice: 2,
		}
	case Dwarf:
		return HeroStats{
			BodyPoints:   7,
			MindPoints:   3,
			AttackDice:   2,
			DefenseDice:  2,
			MovementDice: 2,
		}
	case Elf:
		return HeroStats{
			BodyPoints:   6,
			MindPoints:   4,
			AttackDice:   2,
			DefenseDice:  2,
			MovementDice: 2,
		}
	case Wizard:
		return HeroStats{
			BodyPoints:   4,
			MindPoints:   6,
			AttackDice:   1,
			DefenseDice:  2,
			MovementDice: 2,
		}
	default:
		// Default stats if unknown class
		return HeroStats{
			BodyPoints:   6,
			MindPoints:   4,
			AttackDice:   2,
			DefenseDice:  2,
			MovementDice: 2,
		}
	}
}

// HeroCharacter represents a hero with current stats and equipment
type HeroCharacter struct {
	BaseStats     HeroStats `json:"baseStats"`
	CurrentBody   int       `json:"currentBody"`
	CurrentMind   int       `json:"currentMind"`
	EquipmentMods StatMods  `json:"equipmentMods"`
}

// StatMods represents modifications from equipment
type StatMods struct {
	AttackBonus  int `json:"attackBonus"`  // Additional attack dice from weapons
	DefenseBonus int `json:"defenseBonus"` // Additional defense from armor
}

// GetEffectiveAttackDice returns total attack dice including equipment
func (hc *HeroCharacter) GetEffectiveAttackDice() int {
	total := hc.BaseStats.AttackDice + hc.EquipmentMods.AttackBonus
	if total < 1 {
		return 1 // Attack dice never falls below 1
	}
	return total
}

// GetEffectiveDefenseDice returns total defense dice including equipment
func (hc *HeroCharacter) GetEffectiveDefenseDice() int {
	return hc.BaseStats.DefenseDice + hc.EquipmentMods.DefenseBonus
}

// TakeDamage reduces current body points
func (hc *HeroCharacter) TakeDamage(damage int) {
	hc.CurrentBody -= damage
	if hc.CurrentBody < 0 {
		hc.CurrentBody = 0
	}
}

// IsUnconscious checks if hero is unconscious (0 body points)
func (hc *HeroCharacter) IsUnconscious() bool {
	return hc.CurrentBody <= 0
}

// Heal restores body points
func (hc *HeroCharacter) Heal(amount int) {
	hc.CurrentBody += amount
	if hc.CurrentBody > hc.BaseStats.BodyPoints {
		hc.CurrentBody = hc.BaseStats.BodyPoints
	}
}

// RestoreMind restores mind points
func (hc *HeroCharacter) RestoreMind(amount int) {
	hc.CurrentMind += amount
	if hc.CurrentMind > hc.BaseStats.MindPoints {
		hc.CurrentMind = hc.BaseStats.MindPoints
	}
}

// NewTurnManager creates a new turn manager
func NewTurnManager(broadcaster Broadcaster, logger Logger, diceSystem *DiceSystem) *TurnManager {
	return &TurnManager{
		state: &TurnState{
			TurnNumber:         1,
			CurrentTurn:        HeroTurn,
			CurrentPhase:       MovementPhase,
			ActionsLeft:        1, // Heroes get 1 action per turn
			MovementLeft:       0, // Will be set by dice roll
			MovementDiceRolled: false,
			MovementRolls:      []int{},
			TurnTimeLimit:      5 * time.Minute,
			CanEndTurn:         true,
			NextTurnType:       GameMasterTurn,
		},
		players:     make(map[string]*Player),
		broadcaster: broadcaster,
		logger:      logger,
		diceSystem:  diceSystem,
	}
}

// SetTurnStateManager sets the turn state manager reference
func (tm *TurnManager) SetTurnStateManager(tsm *TurnStateManager) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	tm.turnStateManager = tsm
}

// AddPlayer adds a player to the game
func (tm *TurnManager) AddPlayer(player *Player) error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if _, exists := tm.players[player.ID]; exists {
		return fmt.Errorf("player %s already exists", player.ID)
	}

	tm.players[player.ID] = player
	tm.logger.Printf("Added player %s (%s) to game", player.Name, player.Class)

	// If this is the first player, make them active
	if tm.state.ActivePlayerID == "" {
		tm.state.ActivePlayerID = player.ID
	}

	// Initialize hero turn state in the turn state manager
	if tm.turnStateManager != nil {
		// We don't have the hero position yet, so use a default (0,0)
		// The position will be updated when the game state is initialized
		defaultPos := protocol.TileAddress{X: 0, Y: 0}
		if err := tm.turnStateManager.StartHeroTurn(player.EntityID, player.ID, defaultPos); err != nil {
			tm.logger.Printf("Warning: Failed to initialize hero turn state for %s: %v", player.ID, err)
		}
	}

	tm.broadcastTurnState()
	return nil
}

// GetCurrentPlayer returns the currently active player
func (tm *TurnManager) GetCurrentPlayer() *Player {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	if tm.state.ActivePlayerID == "" {
		return nil
	}

	return tm.players[tm.state.ActivePlayerID]
}

// GetPlayer returns a specific player by ID
func (tm *TurnManager) GetPlayer(playerID string) *Player {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	return tm.players[playerID]
}

// CanPlayerAct checks if a specific player can take actions
func (tm *TurnManager) CanPlayerAct(playerID string) bool {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	return tm.state.CurrentTurn == HeroTurn &&
		tm.state.ActivePlayerID == playerID &&
		(tm.state.ActionsLeft > 0 || tm.state.MovementLeft > 0)
}

// IsPlayersTurn checks if it's the specified player's turn (for instant actions)
func (tm *TurnManager) IsPlayersTurn(playerID string) bool {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	return tm.state.CurrentTurn == HeroTurn &&
		tm.state.ActivePlayerID == playerID
}

// ConsumeMovement reduces remaining movement points and marks movement as used
func (tm *TurnManager) ConsumeMovement(squares int, movementAction string) error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.state.CurrentTurn != HeroTurn {
		return fmt.Errorf("movement can only be consumed during hero turns")
	}

	if !tm.state.MovementDiceRolled {
		return fmt.Errorf("must roll movement dice before moving")
	}

	// Allow movement to continue within the same action type
	if tm.state.HasMoved && (!tm.state.MovementStarted || tm.state.MovementAction != movementAction) {
		return fmt.Errorf("movement already used this turn")
	}

	if tm.state.MovementLeft < squares {
		return fmt.Errorf("not enough movement left: need %d, have %d", squares, tm.state.MovementLeft)
	}

	tm.state.MovementLeft -= squares

	// Mark movement as started on first step and track action type
	if !tm.state.MovementStarted {
		tm.state.MovementStarted = true
		tm.state.MovementAction = movementAction
		tm.state.HasMoved = true // Set immediately when any movement action starts
	}

	tm.logger.Printf("Consumed %d movement, %d remaining", squares, tm.state.MovementLeft)

	tm.broadcastTurnState()
	return nil
}

// EndMovement marks movement as finished for the current turn
func (tm *TurnManager) EndMovement() error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.state.CurrentTurn != HeroTurn {
		return fmt.Errorf("movement can only be ended during hero turns")
	}

	if !tm.state.MovementDiceRolled {
		return fmt.Errorf("no movement to end - dice not rolled")
	}

	if tm.state.HasMoved {
		return fmt.Errorf("movement already ended this turn")
	}

	tm.state.HasMoved = true
	tm.state.MovementStarted = true
	tm.logger.Printf("Movement ended with %d points remaining", tm.state.MovementLeft)

	tm.broadcastTurnState()
	return nil
}

// CanMove checks if the player can still move this turn
func (tm *TurnManager) CanMove() bool {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	// Allow movement if dice rolled and either:
	// 1. No movement started yet, OR
	// 2. Movement started but still has points remaining
	return tm.state.CurrentTurn == HeroTurn &&
		tm.state.MovementDiceRolled &&
		tm.state.MovementLeft > 0 &&
		(!tm.state.HasMoved || tm.state.MovementStarted)
}

// RollMovementDice rolls movement dice for the current player and sets available movement
func (tm *TurnManager) RollMovementDice() ([]DiceRoll, error) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.state.CurrentTurn != HeroTurn {
		return nil, fmt.Errorf("movement dice can only be rolled during hero turns")
	}

	if tm.state.MovementDiceRolled {
		return nil, fmt.Errorf("movement dice already rolled this turn")
	}

	// Get current player's movement dice count
	player := tm.players[tm.state.ActivePlayerID]
	if player == nil {
		return nil, fmt.Errorf("no active player found")
	}

	movementDiceCount := player.Character.BaseStats.MovementDice

	// Roll the dice
	diceRolls := tm.diceSystem.RollDice(MovementDie, movementDiceCount, "movement")

	// Calculate total movement points
	totalMovement := 0
	movementValues := make([]int, len(diceRolls))
	for i, roll := range diceRolls {
		movementValues[i] = roll.Result
		totalMovement += roll.Result
	}

	// Update state
	tm.state.MovementDiceRolled = true
	tm.state.MovementRolls = movementValues
	tm.state.MovementLeft = totalMovement

	tm.logger.Printf("Player %s rolled movement dice: %v (total: %d)",
		tm.state.ActivePlayerID, movementValues, totalMovement)

	tm.broadcastTurnState()
	return diceRolls, nil
}

// ConsumeAction reduces remaining action points and marks action as taken
func (tm *TurnManager) ConsumeAction() error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.state.CurrentTurn != HeroTurn {
		return fmt.Errorf("actions can only be consumed during hero turns")
	}

	if tm.state.ActionsLeft <= 0 {
		return fmt.Errorf("no actions remaining")
	}

	tm.state.ActionsLeft--
	tm.state.ActionTaken = true

	tm.logger.Printf("Consumed 1 action, %d remaining", tm.state.ActionsLeft)

	// Auto-advance to end phase if no actions left
	if tm.state.ActionsLeft == 0 {
		tm.state.CurrentPhase = EndPhase
		tm.state.CanEndTurn = true
	}

	tm.broadcastTurnState()
	return nil
}

// EndTurn advances to the next turn
func (tm *TurnManager) EndTurn() error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if !tm.state.CanEndTurn {
		return fmt.Errorf("cannot end turn in current state")
	}

	// Switch turn types
	switch tm.state.CurrentTurn {
	case HeroTurn:
		tm.advanceToNextHeroOrGameMaster()
	case GameMasterTurn:
		tm.advanceToNextHero()
	}

	tm.broadcastTurnState()
	return nil
}

// ForceAdvanceTurn forces turn advancement (debug function)
func (tm *TurnManager) ForceAdvanceTurn() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.state.CanEndTurn = true
	tm.advanceToNextHeroOrGameMaster()
	tm.broadcastTurnState()
}

// SetGameMasterTurn forces GameMaster turn (debug function)
func (tm *TurnManager) SetGameMasterTurn() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.state.CurrentTurn = GameMasterTurn
	tm.state.CurrentPhase = MonsterMovementPhase
	tm.state.ActivePlayerID = "gamemaster"
	tm.state.ActionsLeft = -1  // Unlimited for GM
	tm.state.MovementLeft = -1 // Unlimited for GM
	tm.state.CanEndTurn = true

	tm.logger.Printf("Forced GameMaster turn")
	tm.broadcastTurnState()
}

// RestoreActions restores actions for testing (debug function)
func (tm *TurnManager) RestoreActions() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.state.CurrentTurn == HeroTurn {
		tm.state.ActionsLeft = 1 // Reset to standard 1 action per turn
		tm.state.ActionTaken = false
		tm.state.CurrentPhase = ActionPhase
		tm.logger.Printf("DEBUG: Restored actions for testing - ActionsLeft: %d", tm.state.ActionsLeft)
		tm.broadcastTurnState()
	}
}

// Private methods

func (tm *TurnManager) advanceToNextHeroOrGameMaster() {
	if tm.state.CurrentTurn == HeroTurn {
		// Check if there are more heroes to play
		nextPlayer := tm.getNextActivePlayer()
		firstPlayer := tm.getFirstActivePlayer()

		// If next player is different from current AND not wrapping back to first, continue hero turns
		if nextPlayer != nil && nextPlayer.ID != tm.state.ActivePlayerID && nextPlayer.ID != firstPlayer.ID {
			// Next hero's turn
			tm.state.ActivePlayerID = nextPlayer.ID
			tm.resetHeroTurn()
		} else {
			// All heroes have played (wrapped back to first) or no more players, switch to GameMaster
			tm.state.CurrentTurn = GameMasterTurn
			tm.state.CurrentPhase = MonsterMovementPhase
			tm.state.ActivePlayerID = "gamemaster"
			tm.state.ActionsLeft = -1  // Unlimited for GM
			tm.state.MovementLeft = -1 // Unlimited for GM
			tm.logger.Printf("Turn %d: Advanced to GameMaster turn", tm.state.TurnNumber)
		}
	}
}

func (tm *TurnManager) advanceToNextHero() {
	tm.state.TurnNumber++
	tm.state.CurrentTurn = HeroTurn
	tm.state.ActivePlayerID = tm.getFirstActivePlayer().ID
	tm.resetHeroTurn()
	tm.logger.Printf("Turn %d: Advanced to Hero turn (Player: %s)", tm.state.TurnNumber, tm.state.ActivePlayerID)

	// Advance TurnStateManager to clear old hero states
	if tm.turnStateManager != nil {
		tm.turnStateManager.AdvanceTurn()
	}
}

func (tm *TurnManager) resetHeroTurn() {
	tm.resetHeroTurnWithPosition(nil)
}

func (tm *TurnManager) resetHeroTurnWithPosition(heroPos *protocol.TileAddress) {
	tm.state.CurrentPhase = MovementPhase
	tm.state.ActionsLeft = 1
	tm.state.MovementLeft = 0 // Will be set by dice roll
	tm.state.MovementDiceRolled = false
	tm.state.MovementRolls = []int{}
	tm.state.HasMoved = false
	tm.state.MovementStarted = false
	tm.state.MovementAction = ""
	tm.state.ActionTaken = false
	tm.state.TurnStartTime = time.Now()
	tm.state.CanEndTurn = true

	// Reset movement history for new turn
	if heroPos != nil {
		tm.state.MovementHistory = []MovementSegment{}
		tm.state.CurrentSegment = nil
		tm.state.InitialHeroPosition = &MovementStep{X: heroPos.X, Y: heroPos.Y}
	}
}

func (tm *TurnManager) getNextActivePlayer() *Player {
	// Simple round-robin for now
	// In a real implementation, this would handle turn order, death, etc.
	playerIDs := make([]string, 0, len(tm.players))
	for id, player := range tm.players {
		if player.IsActive {
			playerIDs = append(playerIDs, id)
		}
	}

	if len(playerIDs) == 0 {
		return nil
	}

	// Sort player IDs for consistent ordering
	for i := 0; i < len(playerIDs); i++ {
		for j := i + 1; j < len(playerIDs); j++ {
			if playerIDs[j] < playerIDs[i] {
				playerIDs[i], playerIDs[j] = playerIDs[j], playerIDs[i]
			}
		}
	}

	// Find current player index
	currentIndex := -1
	for i, id := range playerIDs {
		if id == tm.state.ActivePlayerID {
			currentIndex = i
			break
		}
	}

	// Get next player (wrap around)
	nextIndex := (currentIndex + 1) % len(playerIDs)
	return tm.players[playerIDs[nextIndex]]
}

func (tm *TurnManager) getFirstActivePlayer() *Player {
	// Get all active player IDs and sort them for consistent ordering
	playerIDs := make([]string, 0, len(tm.players))
	for id, player := range tm.players {
		if player.IsActive {
			playerIDs = append(playerIDs, id)
		}
	}

	if len(playerIDs) == 0 {
		return nil
	}

	// Sort to get consistent first player
	// In practice, player-1 will be first, then player-2, etc.
	minID := playerIDs[0]
	for _, id := range playerIDs[1:] {
		if id < minID {
			minID = id
		}
	}

	return tm.players[minID]
}

func (tm *TurnManager) broadcastTurnState() {
	tm.broadcaster.BroadcastEvent("TurnStateChanged", protocol.TurnStateChanged{
		TurnNumber:     tm.state.TurnNumber,
		CurrentTurn:    string(tm.state.CurrentTurn),
		CurrentPhase:   string(tm.state.CurrentPhase),
		ActivePlayerID: tm.state.ActivePlayerID,
		ActionsLeft:    tm.state.ActionsLeft,
		MovementLeft:   tm.state.MovementLeft,
		CanEndTurn:     tm.state.CanEndTurn,
	})
}

// GetTurnState returns the current turn state (read-only)
func (tm *TurnManager) GetTurnState() TurnState {
	tm.lock.RLock()
	defer tm.lock.RUnlock()
	return *tm.state
}

// IsGameMasterTurn checks if it's currently the GameMaster's turn
func (tm *TurnManager) IsGameMasterTurn() bool {
	tm.lock.RLock()
	defer tm.lock.RUnlock()
	return tm.state.CurrentTurn == GameMasterTurn
}

// IsPlayerTurn checks if it's currently a specific player's turn
func (tm *TurnManager) IsPlayerTurn(playerID string) bool {
	tm.lock.RLock()
	defer tm.lock.RUnlock()
	return tm.state.CurrentTurn == HeroTurn && tm.state.ActivePlayerID == playerID
}

// PassGMTurn skips the current GM turn and advances to the next hero turn (debug function)
func (tm *TurnManager) PassGMTurn() error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.state.CurrentTurn == HeroTurn && tm.state.CurrentPhase == EndPhase {
		// Hero is in end phase (no actions left), advance to GM turn first
		tm.logger.Printf("DEBUG: Hero in end phase, advancing to GM turn first")
		tm.advanceToNextHeroOrGameMaster()
	}

	if tm.state.CurrentTurn != GameMasterTurn {
		return fmt.Errorf("can only pass GM turn during GameMaster turn, current turn: %s", tm.state.CurrentTurn)
	}

	tm.logger.Printf("DEBUG: Passing GM turn, advancing to next hero turn")
	tm.advanceToNextHero()
	tm.broadcastTurnState()
	return nil
}

// Movement History Management Functions

// StartMovementSegment begins tracking a new movement segment
func (tm *TurnManager) StartMovementSegment(segmentType MovementSegmentType, startPos protocol.TileAddress) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	// Complete any existing segment first
	if tm.state.CurrentSegment != nil {
		tm.completeCurrentSegmentLocked()
	}

	now := time.Now()
	tm.state.CurrentSegment = &MovementSegment{
		Type:          segmentType,
		StartPosition: MovementStep{X: startPos.X, Y: startPos.Y},
		Path:          []MovementStep{},
		StartTime:     now,
		Executed:      false,
	}
}

// AddMovementStep adds a step to the current movement segment
func (tm *TurnManager) AddMovementStep(pos protocol.TileAddress) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.state.CurrentSegment != nil {
		tm.state.CurrentSegment.Path = append(tm.state.CurrentSegment.Path, MovementStep{X: pos.X, Y: pos.Y})
	}
}

// CompleteMovementSegment completes the current movement segment
func (tm *TurnManager) CompleteMovementSegment() {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	tm.completeCurrentSegmentLocked()
}

// completeCurrentSegmentLocked completes the current segment (must hold lock)
func (tm *TurnManager) completeCurrentSegmentLocked() {
	if tm.state.CurrentSegment != nil {
		now := time.Now()
		tm.state.CurrentSegment.EndTime = &now
		tm.state.MovementHistory = append(tm.state.MovementHistory, *tm.state.CurrentSegment)
		tm.state.CurrentSegment = nil
	}
}

// MarkSegmentExecuted marks a planned segment as executed
func (tm *TurnManager) MarkSegmentExecuted(segmentIndex int) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if segmentIndex >= 0 && segmentIndex < len(tm.state.MovementHistory) {
		now := time.Now()
		tm.state.MovementHistory[segmentIndex].Executed = true
		tm.state.MovementHistory[segmentIndex].ExecutedTime = &now
	}
}

// GetMovementHistory returns a copy of the current movement history
func (tm *TurnManager) GetMovementHistory() []MovementSegment {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	history := make([]MovementSegment, len(tm.state.MovementHistory))
	copy(history, tm.state.MovementHistory)
	return history
}

// ResetMovementHistory clears movement history for a new turn
func (tm *TurnManager) ResetMovementHistory(heroPos protocol.TileAddress) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.state.MovementHistory = []MovementSegment{}
	tm.state.CurrentSegment = nil
	tm.state.InitialHeroPosition = &MovementStep{X: heroPos.X, Y: heroPos.Y}
}

// GetMovementVisualizationData returns structured data for movement visualization
func (tm *TurnManager) GetMovementVisualizationData() map[string]interface{} {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	return map[string]interface{}{
		"history":           tm.state.MovementHistory,
		"currentSegment":    tm.state.CurrentSegment,
		"initialPosition":   tm.state.InitialHeroPosition,
		"movementLeft":      tm.state.MovementLeft,
		"movementDiceRolled": tm.state.MovementDiceRolled,
	}
}