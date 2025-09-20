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
	Health           int                  `json:"health"`
	MaxHealth        int                  `json:"maxHealth"`
	AttackDice       int                  `json:"attackDice"`
	DefenseDice      int                  `json:"defenseDice"`
	MovementRange    int                  `json:"movementRange"`
	IsVisible        bool                 `json:"isVisible"`
	IsAlive          bool                 `json:"isAlive"`
	AIBehavior       AIBehavior           `json:"aiBehavior"`
	SpecialAbilities []string             `json:"specialAbilities,omitempty"`
	SpawnedTurn      int                  `json:"spawnedTurn"`
	LastMovedTurn    int                  `json:"lastMovedTurn"`
}

// MonsterType defines different monster types
type MonsterType string

const (
	Goblin    MonsterType = "goblin"
	Orc       MonsterType = "orc"
	Skeleton  MonsterType = "skeleton"
	Zombie    MonsterType = "zombie"
	Fimir     MonsterType = "fimir"
	ChaosMage MonsterType = "chaos_mage"
	Gargoyle  MonsterType = "gargoyle"
	Mummy     MonsterType = "mummy"
)

// AIBehavior defines monster AI patterns
type AIBehavior string

const (
	Aggressive AIBehavior = "aggressive" // Always attack if possible
	Defensive  AIBehavior = "defensive"  // Prefer defense, guard locations
	Patrol     AIBehavior = "patrol"     // Move in patterns
	GuardRoom  AIBehavior = "guard"      // Stay in specific room
	Wandering  AIBehavior = "wandering"  // Random movement
	Hunter     AIBehavior = "hunter"     // Seek heroes aggressively
)

// MonsterTemplate defines monster stats and behavior
type MonsterTemplate struct {
	Type             MonsterType `json:"type"`
	Name             string      `json:"name"`
	MaxHealth        int         `json:"maxHealth"`
	AttackDice       int         `json:"attackDice"`
	DefenseDice      int         `json:"defenseDice"`
	MovementRange    int         `json:"movementRange"`
	DefaultBehavior  AIBehavior  `json:"defaultBehavior"`
	SpecialAbilities []string    `json:"specialAbilities,omitempty"`
	Description      string      `json:"description"`
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
		Type:            Goblin,
		Name:            "Goblin",
		MaxHealth:       1,
		AttackDice:      2,
		DefenseDice:     2,
		MovementRange:   2,
		DefaultBehavior: Aggressive,
		Description:     "Weak but numerous creatures",
	}

	ms.templates[Orc] = &MonsterTemplate{
		Type:            Orc,
		Name:            "Orc",
		MaxHealth:       2,
		AttackDice:      3,
		DefenseDice:     2,
		MovementRange:   2,
		DefaultBehavior: Aggressive,
		Description:     "Stronger than goblins, more aggressive",
	}

	ms.templates[Skeleton] = &MonsterTemplate{
		Type:            Skeleton,
		Name:            "Skeleton",
		MaxHealth:       1,
		AttackDice:      2,
		DefenseDice:     2,
		MovementRange:   2,
		DefaultBehavior: Patrol,
		Description:     "Undead creatures that guard areas",
	}

	ms.templates[Zombie] = &MonsterTemplate{
		Type:            Zombie,
		Name:            "Zombie",
		MaxHealth:       2,
		AttackDice:      2,
		DefenseDice:     3,
		MovementRange:   1,
		DefaultBehavior: Wandering,
		Description:     "Slow but tough undead",
	}

	ms.templates[Fimir] = &MonsterTemplate{
		Type:             Fimir,
		Name:             "Fimir",
		MaxHealth:        3,
		AttackDice:       3,
		DefenseDice:      3,
		MovementRange:    2,
		DefaultBehavior:  Hunter,
		SpecialAbilities: []string{"tail_attack"},
		Description:      "Dangerous reptilian creatures with tail attacks",
	}

	ms.templates[ChaosMage] = &MonsterTemplate{
		Type:             ChaosMage,
		Name:             "Chaos Mage",
		MaxHealth:        2,
		AttackDice:       2,
		DefenseDice:      2,
		MovementRange:    2,
		DefaultBehavior:  Defensive,
		SpecialAbilities: []string{"chaos_spell", "teleport"},
		Description:      "Spell-casting enemies with magical abilities",
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
		Health:           template.MaxHealth,
		MaxHealth:        template.MaxHealth,
		AttackDice:       template.AttackDice,
		DefenseDice:      template.DefenseDice,
		MovementRange:    template.MovementRange,
		IsVisible:        false, // Monsters start hidden until revealed
		IsAlive:          true,
		AIBehavior:       template.DefaultBehavior,
		SpecialAbilities: template.SpecialAbilities,
		SpawnedTurn:      ms.turnManager.GetTurnState().TurnNumber,
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
	monster.LastMovedTurn = ms.turnManager.GetTurnState().TurnNumber

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

// ProcessMonsterTurn handles AI for all monsters during GameMaster turn
func (ms *MonsterSystem) ProcessMonsterTurn() {
	if !ms.turnManager.IsGameMasterTurn() {
		ms.logger.Printf("Attempted to process monster turn during non-GM turn")
		return
	}

	visibleMonsters := ms.GetVisibleMonsters()
	ms.logger.Printf("Processing turn for %d visible monsters", len(visibleMonsters))

	for _, monster := range visibleMonsters {
		if !monster.IsAlive {
			continue
		}

		action := ms.calculateAIAction(monster)
		if err := ms.executeMonsterAction(action); err != nil {
			ms.logger.Printf("Error executing action for monster %s: %v", monster.ID, err)
		}
	}
}

// Calculate AI action based on monster behavior
func (ms *MonsterSystem) calculateAIAction(monster *Monster) MonsterAction {
	// Get nearby heroes
	nearbyHeroes := ms.getNearbyHeroes(monster.Position, 3) // Within 3 squares

	switch monster.AIBehavior {
	case Aggressive:
		// Attack if adjacent hero, otherwise move toward nearest hero
		for _, heroPos := range nearbyHeroes {
			if ms.calculateDistance(monster.Position, heroPos.position) == 1 {
				return MonsterAction{
					Type:      MonsterAttackAction,
					MonsterID: monster.ID,
					TargetID:  heroPos.entityID,
				}
			}
		}
		// Move toward nearest hero
		if len(nearbyHeroes) > 0 {
			target := ms.findBestMoveToward(monster, nearbyHeroes[0].position)
			return MonsterAction{
				Type:      MonsterMoveAction,
				MonsterID: monster.ID,
				Position:  &target,
			}
		}

	case Defensive:
		// Only attack if hero is adjacent
		for _, heroPos := range nearbyHeroes {
			if ms.calculateDistance(monster.Position, heroPos.position) == 1 {
				return MonsterAction{
					Type:      MonsterAttackAction,
					MonsterID: monster.ID,
					TargetID:  heroPos.entityID,
				}
			}
		}

	case Hunter:
		// Aggressively seek heroes
		if len(nearbyHeroes) > 0 {
			nearest := nearbyHeroes[0]
			if ms.calculateDistance(monster.Position, nearest.position) == 1 {
				return MonsterAction{
					Type:      MonsterAttackAction,
					MonsterID: monster.ID,
					TargetID:  nearest.entityID,
				}
			} else {
				target := ms.findBestMoveToward(monster, nearest.position)
				return MonsterAction{
					Type:      MonsterMoveAction,
					MonsterID: monster.ID,
					Position:  &target,
				}
			}
		}

	case Patrol:
		// Move in patterns (simplified - just random movement for now)
		possibleMoves := ms.getPossibleMoves(monster)
		if len(possibleMoves) > 0 {
			target := possibleMoves[0] // Take first available move
			return MonsterAction{
				Type:      MonsterMoveAction,
				MonsterID: monster.ID,
				Position:  &target,
			}
		}
	}

	// Default: wait
	return MonsterAction{
		Type:      MonsterWaitAction,
		MonsterID: monster.ID,
	}
}

// Execute a monster action
func (ms *MonsterSystem) executeMonsterAction(action MonsterAction) error {
	switch action.Type {
	case MonsterMoveAction:
		if action.Position != nil {
			return ms.MoveMonster(action.MonsterID, *action.Position)
		}
	case MonsterAttackAction:
		return ms.executeMonsterAttackAction(action.MonsterID, action.TargetID)
	case MonsterWaitAction:
		ms.logger.Printf("Monster %s is waiting", action.MonsterID)
		return nil
	}
	return nil
}

// Execute monster attack
func (ms *MonsterSystem) executeMonsterAttackAction(monsterID, targetID string) error {
	monster, exists := ms.monsters[monsterID]
	if !exists {
		return fmt.Errorf("monster %s not found", monsterID)
	}

	// Roll attack dice
	attackRolls := ms.diceSystem.RollDice(CombatDie, monster.AttackDice, "attack")

	// Calculate damage (simplified)
	damage := 0
	for _, roll := range attackRolls {
		if roll.Result >= 3 { // Skulls on 3+
			damage++
		}
	}

	ms.logger.Printf("Monster %s attacked %s for %d damage", monsterID, targetID, damage)

	// Broadcast attack result
	ms.broadcaster.BroadcastEvent("MonsterAttackAction", map[string]any{
		"monsterId": monsterID,
		"targetId":  targetID,
		"damage":    damage,
		"diceRolls": attackRolls,
	})

	return nil
}

// Helper methods

type heroPosition struct {
	entityID string
	position protocol.TileAddress
}

func (ms *MonsterSystem) getNearbyHeroes(center protocol.TileAddress, maxDistance int) []heroPosition {
	var heroes []heroPosition

	ms.gameState.Lock.Lock()
	defer ms.gameState.Lock.Unlock()

	for entityID, position := range ms.gameState.Entities {
		// Check if this is a hero entity (simple check for now)
		if entityID == "hero-1" || (len(entityID) > 4 && entityID[:4] == "hero") {
			distance := ms.calculateDistance(center, position)
			if distance <= maxDistance {
				heroes = append(heroes, heroPosition{
					entityID: entityID,
					position: position,
				})
			}
		}
	}

	return heroes
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

func (ms *MonsterSystem) validatePosition(position protocol.TileAddress) error {
	// Check bounds
	if position.X < 0 || position.Y < 0 ||
		position.X >= ms.gameState.Segment.Width ||
		position.Y >= ms.gameState.Segment.Height {
		return fmt.Errorf("position out of bounds")
	}

	// Check if occupied (simplified)
	ms.gameState.Lock.Lock()
	defer ms.gameState.Lock.Unlock()

	for _, entityPos := range ms.gameState.Entities {
		if entityPos.X == position.X && entityPos.Y == position.Y {
			return fmt.Errorf("position occupied")
		}
	}

	return nil
}

func (ms *MonsterSystem) findBestMoveToward(monster *Monster, target protocol.TileAddress) protocol.TileAddress {
	possibleMoves := ms.getPossibleMoves(monster)

	bestMove := monster.Position
	bestDistance := ms.calculateDistance(monster.Position, target)

	for _, move := range possibleMoves {
		distance := ms.calculateDistance(move, target)
		if distance < bestDistance {
			bestDistance = distance
			bestMove = move
		}
	}

	return bestMove
}

func (ms *MonsterSystem) getPossibleMoves(monster *Monster) []protocol.TileAddress {
	var moves []protocol.TileAddress

	// Check all adjacent positions
	deltas := []struct{ dx, dy int }{
		{0, 1}, {0, -1}, {1, 0}, {-1, 0}, // Cardinal directions
	}

	for _, delta := range deltas {
		newPos := protocol.TileAddress{
			SegmentID: monster.Position.SegmentID,
			X:         monster.Position.X + delta.dx,
			Y:         monster.Position.Y + delta.dy,
		}

		if ms.validatePosition(newPos) == nil {
			moves = append(moves, newPos)
		}
	}

	return moves
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
	monster.Health = 0

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

	if err := ms.executeMonsterAttackAction(request.MonsterID, targetID); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	result.Success = true
	result.Message = fmt.Sprintf("Monster attacked %s", targetID)
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
