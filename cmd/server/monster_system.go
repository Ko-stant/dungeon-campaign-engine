package main

import (
	"fmt"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// MonsterActionRequest represents a request to perform a monster action
type MonsterActionRequest struct {
	MonsterID  string            `json:"monsterId"`
	Action     MonsterActionType `json:"action"`
	Parameters map[string]any    `json:"parameters"`
}

// MonsterActionResult contains the results of performing a monster action
type MonsterActionResult struct {
	Success   bool              `json:"success"`
	Action    MonsterActionType `json:"action"`
	MonsterID string            `json:"monsterId"`
	DiceRolls []DiceRoll        `json:"diceRolls,omitempty"`
	Damage    int               `json:"damage,omitempty"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
}

// MonsterActionType represents actions a monster can take
type MonsterActionType string

const (
	MonsterMoveAction    MonsterActionType = "move"
	MonsterAttackAction  MonsterActionType = "attack"
	MonsterSpecialAction MonsterActionType = "special"
	MonsterWaitAction    MonsterActionType = "wait"
)

// Monster represents a monster entity in the game
type Monster struct {
	ID               string               `json:"id"`
	Type             MonsterType          `json:"type"`
	Position         protocol.TileAddress `json:"position"`
	Body             int                  `json:"body"`
	MaxBody          int                  `json:"MaxBody"`
	Mind             int                  `json:"mind"`
	MaxMind          int                  `json:"maxMind"`
	AttackDice       int                  `json:"attackDice"`
	DefenseDice      int                  `json:"defenseDice"`
	MovementRange    int                  `json:"movementRange"`
	IsVisible        bool                 `json:"isVisible"`
	IsAlive          bool                 `json:"isAlive"`
	SpecialAbilities []string             `json:"specialAbilities,omitempty"`
	SpawnedTurn      int                  `json:"spawnedTurn"`
	LastMovedTurn    int                  `json:"lastMovedTurn"`
	SubType          string               `json:"subType,omitempty"` // e.g., "undead" for skeletons
}

// MonsterType defines different monster types
type MonsterType string

const (
	Goblin       MonsterType = "goblin"
	Orc          MonsterType = "orc"
	Skeleton     MonsterType = "skeleton"
	Zombie       MonsterType = "zombie"
	Gargoyle     MonsterType = "gargoyle"
	Mummy        MonsterType = "mummy"
	DreadWarrior MonsterType = "dread_warrior"
	Abomination  MonsterType = "abomination"
)

// MonsterTemplate defines monster stats and behavior
type MonsterTemplate struct {
	Type             MonsterType `json:"type"`
	Name             string      `json:"name"`
	MaxBody          int         `json:"MaxBody"`
	MaxMind          int         `json:"maxMind"`
	AttackDice       int         `json:"attackDice"`
	DefenseDice      int         `json:"defenseDice"`
	MovementRange    int         `json:"movementRange"`
	SpecialAbilities []string    `json:"specialAbilities,omitempty"`
	Description      string      `json:"description"`
	SubType          string      `json:"subType,omitempty"` // e.g., "undead" for skeletons
}

// MonsterAction represents an action a monster can take
type MonsterAction struct {
	Type       MonsterActionType     `json:"type"`
	MonsterID  string                `json:"monsterId"`
	TargetID   string                `json:"targetId,omitempty"`
	Position   *protocol.TileAddress `json:"position,omitempty"`
	Parameters map[string]any        `json:"parameters,omitempty"`
}

// Remove duplicate - already defined above

// MonsterSystem handles monster management and AI
type MonsterSystem struct {
	monsters      map[string]*Monster
	templates     map[MonsterType]*MonsterTemplate
	gameState     *GameState
	turnManager   *TurnManager
	diceSystem    *DiceSystem
	broadcaster   Broadcaster
	logger        Logger
	nextMonsterID int
}

// NewMonsterSystem creates a new monster system
func NewMonsterSystem(gameState *GameState, turnManager *TurnManager, diceSystem *DiceSystem, broadcaster Broadcaster, logger Logger) *MonsterSystem {
	ms := &MonsterSystem{
		monsters:      make(map[string]*Monster),
		templates:     make(map[MonsterType]*MonsterTemplate),
		gameState:     gameState,
		turnManager:   turnManager,
		diceSystem:    diceSystem,
		broadcaster:   broadcaster,
		logger:        logger,
		nextMonsterID: 1,
	}

	ms.initializeMonsterTemplates()
	return ms
}

// Initialize monster templates with HeroQuest stats
func (ms *MonsterSystem) initializeMonsterTemplates() {
	ms.templates[Goblin] = &MonsterTemplate{
		Type:          Goblin,
		Name:          "Goblin",
		MaxBody:       1,
		MaxMind:       1,
		AttackDice:    2,
		DefenseDice:   1,
		MovementRange: 10,
		Description:   "Weak but numerous creatures",
	}

	ms.templates[Orc] = &MonsterTemplate{
		Type:          Orc,
		Name:          "Orc",
		MaxBody:       1,
		MaxMind:       2,
		AttackDice:    3,
		DefenseDice:   2,
		MovementRange: 8,
		Description:   "Stronger than goblins, more aggressive",
	}

	ms.templates[Skeleton] = &MonsterTemplate{
		Type:          Skeleton,
		Name:          "Skeleton",
		MaxBody:       1,
		MaxMind:       0,
		AttackDice:    2,
		DefenseDice:   2,
		MovementRange: 6,
		Description:   "Undead creatures that guard areas",
		SubType:       "undead",
	}

	ms.templates[Zombie] = &MonsterTemplate{
		Type:          Zombie,
		Name:          "Zombie",
		MaxBody:       1,
		MaxMind:       0,
		AttackDice:    2,
		DefenseDice:   3,
		MovementRange: 5,
		Description:   "Slow but tough undead",
		SubType:       "undead",
	}

	ms.templates[Mummy] = &MonsterTemplate{
		Type:          Mummy,
		Name:          "Mummy",
		MaxBody:       2,
		MaxMind:       0,
		AttackDice:    3,
		DefenseDice:   4,
		MovementRange: 4,
		Description:   "Ancient undead guardians wrapped in bandages",
		SubType:       "undead",
	}

	ms.templates[Gargoyle] = &MonsterTemplate{
		Type:          Gargoyle,
		Name:          "Gargoyle",
		MaxBody:       3,
		MaxMind:       4,
		AttackDice:    4,
		DefenseDice:   5,
		MovementRange: 6,
		Description:   "Stone creatures that guard important areas",
	}

	ms.templates[DreadWarrior] = &MonsterTemplate{
		Type:          DreadWarrior,
		Name:          "Dread Warrior",
		MaxBody:       3,
		MaxMind:       3,
		AttackDice:    4,
		DefenseDice:   4,
		MovementRange: 7,
		Description:   "Heavily armored undead warriors",
	}

	ms.templates[Abomination] = &MonsterTemplate{
		Type:          Abomination,
		Name:          "Abomination",
		MaxBody:       2,
		MaxMind:       3,
		AttackDice:    3,
		DefenseDice:   3,
		MovementRange: 6,
		Description:   "Twisted creatures of chaos",
	}
}

// SpawnMonster creates a new monster at the specified location
func (ms *MonsterSystem) SpawnMonster(monsterType MonsterType, position protocol.TileAddress) (*Monster, error) {
	template, exists := ms.templates[monsterType]
	if !exists {
		return nil, fmt.Errorf("unknown monster type: %s", monsterType)
	}

	// Check if position is valid and unoccupied
	if err := ms.validatePosition(position); err != nil {
		return nil, fmt.Errorf("invalid spawn position: %w", err)
	}

	monsterID := fmt.Sprintf("monster_%d", ms.nextMonsterID)
	ms.nextMonsterID++

	monster := &Monster{
		ID:               monsterID,
		Type:             monsterType,
		Position:         position,
		Body:             template.MaxBody,
		MaxBody:          template.MaxBody,
		Mind:             template.MaxMind,
		MaxMind:          template.MaxMind,
		AttackDice:       template.AttackDice,
		DefenseDice:      template.DefenseDice,
		MovementRange:    template.MovementRange,
		IsVisible:        false, // Monsters start hidden until revealed
		IsAlive:          true,
		SpecialAbilities: template.SpecialAbilities,
		SpawnedTurn:      ms.getTurnNumber(),
		LastMovedTurn:    0,
	}

	ms.monsters[monsterID] = monster

	// Add to game state entities
	ms.gameState.Lock.Lock()
	ms.gameState.Entities[monsterID] = position
	ms.gameState.Lock.Unlock()

	ms.logger.Printf("Spawned %s at (%d,%d) with ID %s", template.Name, position.X, position.Y, monsterID)

	// Broadcast monster spawn (only if visible)
	if monster.IsVisible {
		ms.broadcastMonsterUpdate(monster)
	}

	return monster, nil
}

// MoveMonster moves a monster to a new position
func (ms *MonsterSystem) MoveMonster(monsterID string, destination protocol.TileAddress) error {
	monster, exists := ms.monsters[monsterID]
	if !exists {
		return fmt.Errorf("monster %s not found", monsterID)
	}

	if !monster.IsAlive {
		return fmt.Errorf("monster %s is dead", monsterID)
	}

	// Validate movement distance
	distance := ms.calculateDistance(monster.Position, destination)
	if distance > monster.MovementRange {
		return fmt.Errorf("movement distance %d exceeds range %d", distance, monster.MovementRange)
	}

	// Validate destination
	if err := ms.validatePosition(destination); err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	// Update position
	oldPosition := monster.Position
	monster.Position = destination
	monster.LastMovedTurn = ms.getTurnNumber()

	// Update game state
	ms.gameState.Lock.Lock()
	ms.gameState.Entities[monsterID] = destination
	ms.gameState.Lock.Unlock()

	ms.logger.Printf("Moved %s from (%d,%d) to (%d,%d)", monsterID, oldPosition.X, oldPosition.Y, destination.X, destination.Y)

	// Broadcast update if visible
	if monster.IsVisible {
		ms.broadcaster.BroadcastEvent("EntityUpdated", protocol.EntityUpdated{
			ID:   monsterID,
			Tile: destination,
		})
	}

	return nil
}

// GetVisibleMonsters returns all monsters that are currently visible
func (ms *MonsterSystem) GetVisibleMonsters() []*Monster {
	var visible []*Monster
	for _, monster := range ms.monsters {
		if monster.IsVisible && monster.IsAlive {
			visible = append(visible, monster)
		}
	}
	return visible
}

// RevealMonster makes a monster visible
func (ms *MonsterSystem) RevealMonster(monsterID string) error {
	monster, exists := ms.monsters[monsterID]
	if !exists {
		return fmt.Errorf("monster %s not found", monsterID)
	}

	if !monster.IsVisible {
		monster.IsVisible = true
		ms.broadcastMonsterUpdate(monster)
		ms.logger.Printf("Revealed monster %s", monsterID)
	}

	return nil
}

// KillMonster removes a monster from the game
func (ms *MonsterSystem) KillMonster(monsterID string) error {
	monster, exists := ms.monsters[monsterID]
	if !exists {
		return fmt.Errorf("monster %s not found", monsterID)
	}

	monster.IsAlive = false
	monster.Body = 0

	// Remove from game state entities
	ms.gameState.Lock.Lock()
	delete(ms.gameState.Entities, monsterID)
	ms.gameState.Lock.Unlock()

	// Broadcast monster death
	ms.broadcaster.BroadcastEvent("MonsterKilled", map[string]any{
		"monsterId": monsterID,
	})

	ms.logger.Printf("Killed monster %s", monsterID)
	return nil
}

func (ms *MonsterSystem) broadcastMonsterUpdate(monster *Monster) {
	ms.broadcaster.BroadcastEvent("MonsterUpdate", map[string]any{
		"monster": monster,
	})
}

// ProcessAction processes a monster action request
func (ms *MonsterSystem) ProcessAction(request MonsterActionRequest) (*MonsterActionResult, error) {
	result := &MonsterActionResult{
		Action:    request.Action,
		MonsterID: request.MonsterID,
		Timestamp: time.Now(),
	}

	switch request.Action {
	case MonsterMoveAction:
		return ms.processMonsterMoveAction(request, result)
	case MonsterAttackAction:
		return ms.processMonsterAttackAction(request, result)
	case MonsterSpecialAction:
		return ms.processMonsterSpecial(request, result)
	case MonsterWaitAction:
		return ms.processMonsterWaitAction(request, result)
	default:
		return nil, fmt.Errorf("unknown monster action: %s", request.Action)
	}
}

// GetMonsters returns all monsters
func (ms *MonsterSystem) GetMonsters() map[string]*Monster {
	return ms.monsters
}

// GetMonsterByID returns a specific monster by ID
func (ms *MonsterSystem) GetMonsterByID(monsterID string) (*Monster, error) {
	monster, exists := ms.monsters[monsterID]
	if !exists {
		return nil, fmt.Errorf("monster %s not found", monsterID)
	}
	return monster, nil
}

// ApplyDamageToMonster applies damage to a monster and handles death
func (ms *MonsterSystem) ApplyDamageToMonster(monsterID string, damage int) (*Monster, bool, error) {
	monster, exists := ms.monsters[monsterID]
	if !exists {
		return nil, false, fmt.Errorf("monster %s not found", monsterID)
	}

	if !monster.IsAlive {
		return monster, false, fmt.Errorf("monster %s is already dead", monsterID)
	}

	// Apply damage
	monster.Body -= damage
	isDead := false

	// Check if monster dies
	if monster.Body <= 0 {
		monster.Body = 0
		monster.IsAlive = false
		isDead = true
		ms.logger.Printf("Monster %s (%s) has been killed", monster.ID, monster.Type)
	}

	// Broadcast monster update
	ms.broadcastMonsterUpdate(monster)

	return monster, isDead, nil
}

// IsMonsterAt checks if there is an alive monster at the specified position
func (ms *MonsterSystem) IsMonsterAt(x, y int) bool {
	for _, monster := range ms.monsters {
		if monster.IsAlive && monster.Position.X == x && monster.Position.Y == y {
			return true
		}
	}
	return false
}

// Monster action processors

func (ms *MonsterSystem) processMonsterMoveAction(request MonsterActionRequest, result *MonsterActionResult) (*MonsterActionResult, error) {
	targetX, ok1 := request.Parameters["x"].(float64)
	targetY, ok2 := request.Parameters["y"].(float64)

	if !ok1 || !ok2 {
		result.Success = false
		result.Message = "Invalid move parameters"
		return result, fmt.Errorf("missing x/y parameters")
	}

	newPos := protocol.TileAddress{
		X: int(targetX),
		Y: int(targetY),
	}

	if err := ms.MoveMonster(request.MonsterID, newPos); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	result.Success = true
	result.Message = fmt.Sprintf("Monster moved to (%d,%d)", newPos.X, newPos.Y)
	return result, nil
}

func (ms *MonsterSystem) processMonsterAttackAction(request MonsterActionRequest, result *MonsterActionResult) (*MonsterActionResult, error) {
	targetID, ok := request.Parameters["targetId"].(string)
	if !ok {
		result.Success = false
		result.Message = "No target specified"
		return result, fmt.Errorf("missing targetId parameter")
	}

	// Get the monster
	monster, exists := ms.monsters[request.MonsterID]
	if !exists {
		result.Success = false
		result.Message = "Monster not found"
		return result, fmt.Errorf("monster %s not found", request.MonsterID)
	}

	// Roll attack dice
	attackRolls := ms.diceSystem.RollAttackDice(monster.AttackDice)
	skulls := 0
	for _, roll := range attackRolls {
		if roll.CombatResult == Skull {
			skulls++
		}
		result.DiceRolls = append(result.DiceRolls, roll)
	}

	result.Damage = skulls
	result.Success = true
	result.Message = fmt.Sprintf("Monster attacked %s for %d damage", targetID, skulls)

	ms.logger.Printf("Monster %s attacks %s - rolled %d skulls", request.MonsterID, targetID, skulls)

	// TODO: Actually apply damage to target when hero damage system is ready

	return result, nil
}

func (ms *MonsterSystem) processMonsterSpecial(request MonsterActionRequest, result *MonsterActionResult) (*MonsterActionResult, error) {
	abilityID, ok := request.Parameters["abilityId"].(string)
	if !ok {
		result.Success = false
		result.Message = "No ability specified"
		return result, fmt.Errorf("missing abilityId parameter")
	}

	// TODO: Implement special abilities
	result.Success = true
	result.Message = fmt.Sprintf("Monster used %s", abilityID)
	return result, nil
}

func (ms *MonsterSystem) processMonsterWaitAction(request MonsterActionRequest, result *MonsterActionResult) (*MonsterActionResult, error) {
	result.Success = true
	result.Message = "Monster is waiting"
	return result, nil
}

// getTurnNumber safely gets the current turn number, handling nil TurnManager
func (ms *MonsterSystem) getTurnNumber() int {
	if ms.turnManager == nil {
		return 1 // Default to turn 1 for initialization
	}
	return ms.turnManager.GetTurnState().TurnNumber
}

// Utility functions needed by core monster system
func (ms *MonsterSystem) validatePosition(position protocol.TileAddress) error {
	// Basic bounds checking (should be enhanced with actual game board bounds)
	if position.X < 0 || position.Y < 0 {
		return fmt.Errorf("position out of bounds: (%d, %d)", position.X, position.Y)
	}
	return nil
}

func (ms *MonsterSystem) calculateDistance(pos1, pos2 protocol.TileAddress) int {
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy // Manhattan distance
}

// executeMonsterAttackAction - Used by ProcessAction for GM-controlled monster attacks
func (ms *MonsterSystem) executeMonsterAttackAction(monsterID, targetID string) error {
	_, exists := ms.monsters[monsterID]
	if !exists {
		return fmt.Errorf("monster %s not found", monsterID)
	}

	// For now, this is a placeholder for GM-controlled attacks
	// TODO: Implement proper hero damage system when hero damage system is ready
	ms.logger.Printf("Monster %s attacks %s (GM-controlled action)", monsterID, targetID)
	return nil
}
