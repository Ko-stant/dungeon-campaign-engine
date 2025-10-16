package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// Test fixtures for hero actions
func createTestHeroActionSystem() *HeroActionSystem {
	gameState := &GameState{
		Entities: map[string]protocol.TileAddress{
			"hero-1": {X: 5, Y: 5},
		},
		Doors:              make(map[string]*DoorInfo),
		RevealedRegions:    make(map[int]bool),
		KnownRegions:       make(map[int]bool),
		KnownDoors:         make(map[string]bool),
		KnownBlockingWalls: make(map[string]bool),
	}

	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}

	debugConfig := DebugConfig{
		Enabled:            true,
		AllowStateChanges:  true,
		AllowTeleportation: true,
		AllowMapReveal:     true,
		AllowDiceOverride:  true,
		LogDebugActions:    true,
	}
	debugSystem := NewDebugSystem(debugConfig, gameState, broadcaster, logger)
	diceSystem := NewDiceSystem(debugSystem)
	turnManager := NewTurnManager(broadcaster, logger, diceSystem)

	// Add test player
	player := NewPlayer("player-1", "Test Hero", "hero-1", Barbarian)
	turnManager.AddPlayer(player)

	has := NewHeroActionSystem(gameState, turnManager, broadcaster, logger, debugSystem)

	// Add mock monster system for attack tests
	monsterSystem := createTestMonsterSystem(logger)
	has.SetMonsterSystem(monsterSystem)

	return has
}

func createTestMonsterSystem(logger Logger) *MonsterSystem {
	gameState := &GameState{
		Entities:      make(map[string]protocol.TileAddress),
		KnownMonsters: make(map[string]bool),
	}
	broadcaster := &MockBroadcaster{}

	ms := NewMonsterSystem(gameState, nil, nil, broadcaster, logger)

	// Add a test monster for combat tests
	testMonster := &Monster{
		ID:               "monster-1",
		Type:             Goblin,
		Position:         protocol.TileAddress{X: 6, Y: 6},
		Body:             3,
		MaxBody:          3,
		AttackDice:       2,
		DefenseDice:      1,
		MovementRange:    3,
		IsVisible:        true,
		IsAlive:          true,
		SpecialAbilities: []string{},
		SpawnedTurn:      1,
		LastMovedTurn:    0,
	}
	ms.monsters["monster-1"] = testMonster

	return ms
}

// Use existing MockLogger from engine_test.go

// Test the 6 official HeroQuest actions

func TestHeroActions_Attack_Success(t *testing.T) {
	has := createTestHeroActionSystem()

	request := ActionRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": "monster-1",
		},
	}

	result, err := has.ProcessAction(request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful attack, got: %s", result.Message)
	}

	if result.Action != AttackAction {
		t.Errorf("Expected action %s, got %s", AttackAction, result.Action)
	}

	if len(result.AttackRolls) == 0 {
		t.Error("Expected attack rolls for attack")
	}
}

func TestHeroActions_SearchTreasure_FindsItems(t *testing.T) {
	t.Skip("Requires full treasure system integration - use TestTreasureSystem_* tests instead")
}

func TestHeroActions_SearchTraps_RequiresHighRoll(t *testing.T) {
	has := createTestHeroActionSystem()

	// Enable actions for testing
	has.turnManager.RestoreActions()

	// Test with low roll (should fail)
	debugSystem := has.debugSystem
	debugSystem.SetDiceOverride("search_traps", 3)

	request := ActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     SearchTrapsAction,
		Parameters: map[string]any{},
	}

	result, err := has.ProcessAction(request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful action, got: %s", result.Message)
	}

	if result.Message != "No traps found" {
		t.Errorf("Expected 'No traps found' with roll 3, got: %s", result.Message)
	}

	// Restore actions for second test
	has.turnManager.RestoreActions()

	// Test with high roll (should succeed)
	debugSystem.SetDiceOverride("search_traps", 6)

	result, err = has.ProcessAction(request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Message != "Found a trap!" {
		t.Errorf("Expected 'Found a trap!' with roll 6, got: %s", result.Message)
	}
}

func TestHeroActions_SearchSecret_RequiresSix(t *testing.T) {
	has := createTestHeroActionSystem()

	// Enable actions for testing
	has.turnManager.RestoreActions()

	// Test with roll 5 (should fail)
	debugSystem := has.debugSystem
	debugSystem.SetDiceOverride("search_secret", 5)

	request := ActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     SearchSecretAction,
		Parameters: map[string]any{},
	}

	result, err := has.ProcessAction(request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Message != "No secret doors found" {
		t.Errorf("Expected 'No secret doors found' with roll 5, got: %s", result.Message)
	}

	// Restore actions for second test
	has.turnManager.RestoreActions()

	// Test with roll 6 (should succeed)
	debugSystem.SetDiceOverride("search_secret", 6)

	result, err = has.ProcessAction(request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Message != "Found a secret door!" {
		t.Errorf("Expected 'Found a secret door!' with roll 6, got: %s", result.Message)
	}

	if result.SecretRevealed == nil {
		t.Error("Expected secret door to be revealed")
	}
}

func TestHeroActions_DisarmTrap_RequiresTrapID(t *testing.T) {
	has := createTestHeroActionSystem()

	// Enable actions for testing
	has.turnManager.RestoreActions()

	// Test without trap ID (should fail)
	request := ActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     DisarmTrapAction,
		Parameters: map[string]any{},
	}

	result, err := has.ProcessAction(request)

	if err == nil {
		t.Fatal("Expected error for missing trap ID")
	}

	if result.Success {
		t.Error("Expected failure for missing trap ID")
	}

	// Restore actions for second test
	has.turnManager.RestoreActions()

	// Test with trap ID
	request.Parameters["trapId"] = "trap-1"

	result, err = has.ProcessAction(request)

	if err != nil {
		t.Fatalf("Expected no error with trap ID, got: %v", err)
	}

	if len(result.SearchRolls) == 0 {
		t.Error("Expected search rolls for disarm attempt")
	}
}

func TestHeroActions_CastSpell_RequiresSpellID(t *testing.T) {
	has := createTestHeroActionSystem()

	// Test without spell ID (should fail)
	request := ActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     CastSpellAction,
		Parameters: map[string]any{},
	}

	result, err := has.ProcessAction(request)

	if err == nil {
		t.Fatal("Expected error for missing spell ID")
	}

	if result.Success {
		t.Error("Expected failure for missing spell ID")
	}

	// Test with spell ID
	request.Parameters["spellId"] = "fireball"

	result, err = has.ProcessAction(request)

	if err != nil {
		t.Fatalf("Expected no error with spell ID, got: %v", err)
	}

	if result.SpellEffect == nil {
		t.Error("Expected spell effect to be created")
	}
}

func TestHeroActions_ConsumesActionPoint(t *testing.T) {
	has := createTestHeroActionSystem()

	// Start hero turn and enable movement/actions
	has.turnManager.RestoreActions()

	// Check initial action points
	turnState := has.turnManager.GetTurnState()
	initialActions := turnState.ActionsLeft
	t.Logf("Initial state: ActionsLeft=%d, ActionTaken=%v", turnState.ActionsLeft, turnState.ActionTaken)

	request := ActionRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": "monster-1",
		},
	}

	result, err := has.ProcessAction(request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	t.Logf("Attack result: Success=%v, Message=%s", result.Success, result.Message)

	// Check that action point was consumed
	newTurnState := has.turnManager.GetTurnState()
	t.Logf("After attack: ActionsLeft=%d, ActionTaken=%v", newTurnState.ActionsLeft, newTurnState.ActionTaken)

	if newTurnState.ActionsLeft != initialActions-1 {
		t.Errorf("Expected actions to decrease by 1, got %d -> %d", initialActions, newTurnState.ActionsLeft)
	}

	if !newTurnState.ActionTaken {
		t.Error("Expected ActionTaken to be true after action")
	}
}

func TestHeroActions_InvalidPlayer(t *testing.T) {
	has := createTestHeroActionSystem()

	request := ActionRequest{
		PlayerID: "invalid-player",
		EntityID: "hero-1",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": "monster-1",
		},
	}

	_, err := has.ProcessAction(request)

	if err == nil {
		t.Fatal("Expected error for invalid player")
	}
}

func TestHeroActions_WrongEntity(t *testing.T) {
	has := createTestHeroActionSystem()

	request := ActionRequest{
		PlayerID: "player-1",
		EntityID: "wrong-hero",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": "monster-1",
		},
	}

	_, err := has.ProcessAction(request)

	if err == nil {
		t.Fatal("Expected error for wrong entity")
	}
}
