package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// Integration test for complete game flow scenarios

func createTestGameManager() *GameManager {
	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}
	sequenceGen := &MockSequenceGenerator{}

	debugConfig := DebugConfig{
		Enabled:            true,
		AllowStateChanges:  true,
		AllowTeleportation: true,
		AllowMapReveal:     true,
		AllowDiceOverride:  true,
		LogDebugActions:    true,
	}

	gameManager, err := NewGameManager(broadcaster, logger, sequenceGen, debugConfig)
	if err != nil {
		panic("Failed to create test game manager: " + err.Error())
	}

	return gameManager
}

type MockSequenceGenerator struct {
	counter uint64
}

func (sg *MockSequenceGenerator) Next() uint64 {
	sg.counter++
	return sg.counter
}

// Helper function to roll movement dice for tests
func rollMovementDiceForTest(t *testing.T, gm *GameManager) {
	t.Helper()
	_, err := gm.turnManager.RollMovementDice()
	if err != nil {
		t.Fatalf("Failed to roll movement dice: %v", err)
	}
}

func TestGameFlow_CompleteHeroTurn(t *testing.T) {

	gm := createTestGameManager()

	// Get initial state
	initialTurnState := gm.GetTurnState()
	if initialTurnState.CurrentTurn != "hero" {
		t.Fatalf("Expected hero turn, got: %s", initialTurnState.CurrentTurn)
	}

	// Spawn a monster for combat testing
	monster, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 5, Y: 14})
	if err != nil {
		t.Fatalf("Failed to spawn monster: %v", err)
	}

	// Roll movement dice before moving
	rollMovementDiceForTest(t, gm)

	// 1. Move before action
	moveReq := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       1,
		DY:       0,
	}

	err = gm.ProcessMovement(moveReq)
	if err != nil {
		t.Fatalf("Movement failed: %v", err)
	}

	// Check movement state
	turnState := gm.GetTurnState()
	if !turnState.HasMoved {
		t.Error("Expected HasMoved to be true after movement")
	}

	// 2. Perform main action (attack)
	actionReq := ActionRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": monster.ID,
		},
	}

	result, err := gm.ProcessHeroAction(actionReq)
	if err != nil {
		t.Fatalf("Hero action failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful action, got: %s", result.Message)
	}

	// Check action state
	turnState = gm.GetTurnState()
	if !turnState.ActionTaken {
		t.Error("Expected ActionTaken to be true after action")
	}

	// 3. Use instant action (potion)
	instantReq := InstantActionRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   UsePotionInstant,
		Parameters: map[string]any{
			"potionId": "healing-potion",
		},
	}

	instantResult, err := gm.heroActions.ProcessInstantAction(instantReq)
	if err != nil {
		t.Fatalf("Instant action failed: %v", err)
	}

	if !instantResult.Success {
		t.Fatalf("Expected successful instant action, got: %s", instantResult.Message)
	}

	// 4. End turn
	err = gm.EndTurn()
	if err != nil {
		t.Fatalf("End turn failed: %v", err)
	}

	// Check turn advancement
	finalTurnState := gm.GetTurnState()
	if finalTurnState.CurrentTurn != "gamemaster" {
		t.Errorf("Expected gamemaster turn after hero turn, got: %s", finalTurnState.CurrentTurn)
	}
}

func TestGameFlow_MovementAndActionOrder(t *testing.T) {

	gm := createTestGameManager()

	// Roll movement dice first
	rollMovementDiceForTest(t, gm)

	// Test: Action before movement
	actionReq := ActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     SearchTreasureAction,
		Parameters: map[string]any{},
	}

	_, err := gm.ProcessHeroAction(actionReq)
	if err != nil {
		t.Fatalf("Action failed: %v", err)
	}

	// Now movement should still work (movement after action)
	moveReq := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       1,
		DY:       0,
	}

	err = gm.ProcessMovement(moveReq)
	if err != nil {
		t.Fatalf("Movement after action failed: %v", err)
	}

	turnState := gm.GetTurnState()
	if !turnState.HasMoved || !turnState.ActionTaken {
		t.Error("Expected both movement and action to be completed")
	}
}

func TestGameFlow_MovementOncePerTurnEnforcement(t *testing.T) {
	t.Skip("Test obsolete - we now use dice-based movement system where players can move multiple times until points run out. See TestMovement_FivePointConsumption for dice-based movement testing.")
}

func TestGameFlow_ActionOncePerTurnEnforcement(t *testing.T) {

	gm := createTestGameManager()

	// Spawn a monster for attack testing
	monster, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 4, Y: 14})
	if err != nil {
		t.Fatalf("Failed to spawn monster: %v", err)
	}

	// First action
	actionReq1 := ActionRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": monster.ID,
		},
	}

	_, err = gm.ProcessHeroAction(actionReq1)
	if err != nil {
		t.Fatalf("First action failed: %v", err)
	}

	// Second action should fail
	actionReq2 := ActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     SearchTreasureAction,
		Parameters: map[string]any{},
	}

	_, err = gm.ProcessHeroAction(actionReq2)
	if err == nil {
		t.Fatal("Expected second action to fail")
	}
}

func TestGameFlow_InstantActionsUnlimited(t *testing.T) {

	gm := createTestGameManager()

	// Multiple instant actions should all work
	instantActions := []InstantActionRequest{
		{
			PlayerID: "player-1",
			EntityID: "hero-1",
			Action:   UsePotionInstant,
			Parameters: map[string]any{
				"potionId": "healing-potion-1",
			},
		},
		{
			PlayerID: "player-1",
			EntityID: "hero-1",
			Action:   UseItemInstant,
			Parameters: map[string]any{
				"itemId": "torch",
			},
		},
	}

	for i, req := range instantActions {
		result, err := gm.heroActions.ProcessInstantAction(req)
		if err != nil {
			t.Fatalf("Instant action %d failed: %v", i, err)
		}

		if !result.Success {
			t.Errorf("Instant action %d not successful: %s", i, result.Message)
		}
	}

	// Main action should still be available
	turnState := gm.GetTurnState()
	if turnState.ActionTaken {
		t.Error("Expected main action to still be available after instant actions")
	}
}

func TestGameFlow_TurnTransition(t *testing.T) {

	gm := createTestGameManager()

	// Complete hero turn using instant action
	instantReq := InstantActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     PassTurnInstant, // This should end the turn
		Parameters: map[string]any{},
	}

	_, err := gm.heroActions.ProcessInstantAction(instantReq)
	if err != nil {
		t.Fatalf("Pass turn failed: %v", err)
	}

	// Should now be GameMaster turn
	turnState := gm.GetTurnState()
	if turnState.CurrentTurn != "gamemaster" {
		t.Errorf("Expected gamemaster turn, got: %s", turnState.CurrentTurn)
	}

	// End GameMaster turn
	err = gm.EndTurn()
	if err != nil {
		t.Fatalf("GM end turn failed: %v", err)
	}

	// Should be back to hero turn with reset state
	turnState = gm.GetTurnState()
	if turnState.CurrentTurn != "hero" {
		t.Errorf("Expected hero turn after GM, got: %s", turnState.CurrentTurn)
	}

	if turnState.HasMoved || turnState.ActionTaken {
		t.Error("Expected clean turn state after turn transition")
	}

	if turnState.ActionsLeft != 1 {
		t.Errorf("Expected 1 action point, got actions=%d", turnState.ActionsLeft)
	}

	// Movement starts at 0 and must be rolled with dice
	if turnState.MovementLeft != 0 {
		t.Errorf("Expected 0 movement points before rolling dice, got movement=%d", turnState.MovementLeft)
	}

	// Verify we can roll movement dice in new turn
	rollMovementDiceForTest(t, gm)
	turnState = gm.GetTurnState()
	if turnState.MovementLeft <= 0 {
		t.Error("Expected movement points after rolling dice")
	}
}

func TestGameFlow_ErrorHandling(t *testing.T) {

	gm := createTestGameManager()

	// Test invalid player
	invalidReq := ActionRequest{
		PlayerID: "invalid-player",
		EntityID: "hero-1",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": "monster-1",
		},
	}

	_, err := gm.ProcessHeroAction(invalidReq)
	if err == nil {
		t.Fatal("Expected error for invalid player")
	}

	// Test invalid entity
	invalidEntityReq := ActionRequest{
		PlayerID: "player-1",
		EntityID: "invalid-hero",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": "monster-1",
		},
	}

	_, err = gm.ProcessHeroAction(invalidEntityReq)
	if err == nil {
		t.Fatal("Expected error for invalid entity")
	}

	// Test invalid action parameters
	invalidParamsReq := ActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     AttackAction,
		Parameters: map[string]any{
			// Missing targetId
		},
	}

	result, err := gm.ProcessHeroAction(invalidParamsReq)
	if err == nil && result.Success {
		t.Fatal("Expected error for missing attack target")
	}
}

func TestGameFlow_DebugIntegration(t *testing.T) {

	gm := createTestGameManager()

	// Test debug teleportation
	err := gm.DebugTeleportHero("hero-1", 8, 8)
	if err != nil {
		t.Fatalf("Debug teleport failed: %v", err)
	}

	// Verify hero position
	gameState := gm.GetGameState()
	heroPos := gameState.Entities["hero-1"]
	if heroPos.X != 8 || heroPos.Y != 8 {
		t.Errorf("Expected hero at (8,8), got (%d,%d)", heroPos.X, heroPos.Y)
	}

	// Test debug map reveal
	err = gm.DebugRevealMap()
	if err != nil {
		t.Fatalf("Debug map reveal failed: %v", err)
	}

	// Test debug monster spawn
	monster, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 3, Y: 3})
	if err != nil {
		t.Fatalf("Debug monster spawn failed: %v", err)
	}

	if monster.Type != Goblin {
		t.Errorf("Expected Goblin, got %s", monster.Type)
	}

	monsters := gm.GetMonsters()
	if len(monsters) == 0 {
		t.Error("Expected monster to be in game state")
	}
}

func TestGameFlow_CombatIntegration(t *testing.T) {

	gm := createTestGameManager()

	// Spawn a monster for combat
	monster, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 6, Y: 5}) // Adjacent to hero
	if err != nil {
		t.Fatalf("Failed to spawn monster: %v", err)
	}

	// Hero attacks monster
	attackReq := ActionRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": monster.ID,
		},
	}

	result, err := gm.ProcessHeroAction(attackReq)
	if err != nil {
		t.Fatalf("Attack failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful attack, got: %s", result.Message)
	}

	if len(result.AttackRolls) == 0 {
		t.Error("Expected attack rolls for attack")
	}

	if result.Damage < 0 {
		t.Error("Expected non-negative damage")
	}
}
