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

// TurnState represents the current state of the turn system
type TurnState struct {
	TurnNumber      int           `json:"turnNumber"`
	CurrentTurn     TurnType      `json:"currentTurn"`
	CurrentPhase    TurnPhase     `json:"currentPhase"`
	ActivePlayerID  string        `json:"activePlayerId"`
	ActionsLeft     int           `json:"actionsLeft"`
	MovementLeft    int           `json:"movementLeft"`
	HasMoved        bool          `json:"hasMoved"`        // Whether player has used their movement this turn
	ActionTaken     bool          `json:"actionTaken"`     // Whether player has used their main action
	TurnStartTime   time.Time     `json:"turnStartTime"`
	TurnTimeLimit   time.Duration `json:"turnTimeLimit"`
	CanEndTurn      bool          `json:"canEndTurn"`
	NextTurnType    TurnType      `json:"nextTurnType"`
}

// TurnManager handles turn progression and validation
type TurnManager struct {
	state       *TurnState
	players     map[string]*Player // Player ID -> Player
	broadcaster Broadcaster
	logger      Logger
	lock        sync.RWMutex
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
func NewTurnManager(broadcaster Broadcaster, logger Logger) *TurnManager {
	return &TurnManager{
		state: &TurnState{
			TurnNumber:    1,
			CurrentTurn:   HeroTurn,
			CurrentPhase:  MovementPhase,
			ActionsLeft:   1, // Heroes get 1 action per turn
			MovementLeft:  2, // Heroes can move up to 2 squares
			TurnTimeLimit: 5 * time.Minute,
			CanEndTurn:    true,
			NextTurnType:  GameMasterTurn,
		},
		players:     make(map[string]*Player),
		broadcaster: broadcaster,
		logger:      logger,
	}
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

// CanPlayerAct checks if a specific player can take actions
func (tm *TurnManager) CanPlayerAct(playerID string) bool {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	return tm.state.CurrentTurn == HeroTurn &&
		tm.state.ActivePlayerID == playerID &&
		(tm.state.ActionsLeft > 0 || tm.state.MovementLeft > 0)
}

// ConsumeMovement reduces remaining movement points and marks movement as used
func (tm *TurnManager) ConsumeMovement(squares int) error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.state.CurrentTurn != HeroTurn {
		return fmt.Errorf("movement can only be consumed during hero turns")
	}

	if tm.state.HasMoved {
		return fmt.Errorf("movement already used this turn")
	}

	if tm.state.MovementLeft < squares {
		return fmt.Errorf("not enough movement left: need %d, have %d", squares, tm.state.MovementLeft)
	}

	tm.state.MovementLeft -= squares
	tm.state.HasMoved = true
	tm.logger.Printf("Consumed %d movement, %d remaining", squares, tm.state.MovementLeft)

	tm.broadcastTurnState()
	return nil
}

// CanMove checks if the player can still move this turn
func (tm *TurnManager) CanMove() bool {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	return tm.state.CurrentTurn == HeroTurn && !tm.state.HasMoved && tm.state.MovementLeft > 0
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
		if nextPlayer != nil && nextPlayer.ID != tm.state.ActivePlayerID {
			// Next hero's turn
			tm.state.ActivePlayerID = nextPlayer.ID
			tm.resetHeroTurn()
		} else {
			// All heroes have played, switch to GameMaster
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
}

func (tm *TurnManager) resetHeroTurn() {
	tm.state.CurrentPhase = MovementPhase
	tm.state.ActionsLeft = 1
	tm.state.MovementLeft = 2
	tm.state.HasMoved = false
	tm.state.ActionTaken = false
	tm.state.TurnStartTime = time.Now()
	tm.state.CanEndTurn = true
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
	for _, player := range tm.players {
		if player.IsActive {
			return player
		}
	}
	return nil
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