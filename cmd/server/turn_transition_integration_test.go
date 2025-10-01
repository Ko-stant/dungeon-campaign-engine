package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

func TestTurnTransition_HeroToGameMaster(t *testing.T) {

	gm := createTestGameManager()

	// Verify starting state
	initialState := gm.GetTurnState()
	if initialState.CurrentTurn != "hero" {
		t.Fatalf("Expected hero turn, got: %s", initialState.CurrentTurn)
	}
	if initialState.TurnNumber != 1 {
		t.Errorf("Expected turn number 1, got: %d", initialState.TurnNumber)
	}

	// Complete hero turn
	err := gm.EndTurn()
	if err != nil {
		t.Fatalf("End hero turn failed: %v", err)
	}

	// Should now be GameMaster turn
	gmState := gm.GetTurnState()
	if gmState.CurrentTurn != "gamemaster" {
		t.Errorf("Expected gamemaster turn, got: %s", gmState.CurrentTurn)
	}
	if gmState.TurnNumber != 1 {
		t.Errorf("Expected same turn number during GM phase, got: %d", gmState.TurnNumber)
	}
}

func TestTurnTransition_GameMasterToHero(t *testing.T) {

	gm := createTestGameManager()

	// Move to GameMaster turn first
	err := gm.EndTurn()
	if err != nil {
		t.Fatalf("End hero turn failed: %v", err)
	}

	// Verify we're in GM turn
	gmState := gm.GetTurnState()
	if gmState.CurrentTurn != "gamemaster" {
		t.Fatalf("Expected gamemaster turn, got: %s", gmState.CurrentTurn)
	}

	// End GameMaster turn
	err = gm.EndTurn()
	if err != nil {
		t.Fatalf("End GM turn failed: %v", err)
	}

	// Should be back to hero turn with incremented turn number
	heroState := gm.GetTurnState()
	if heroState.CurrentTurn != "hero" {
		t.Errorf("Expected hero turn after GM, got: %s", heroState.CurrentTurn)
	}
	if heroState.TurnNumber != 2 {
		t.Errorf("Expected turn number 2, got: %d", heroState.TurnNumber)
	}
}

func TestTurnTransition_StateReset(t *testing.T) {

	gm := createTestGameManager()

	// Roll movement dice before moving
	rollMovementDiceForTest(t, gm)

	// Use movement and action in first turn
	moveReq := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       1,
		DY:       0,
	}

	err := gm.ProcessMovement(moveReq)
	if err != nil {
		t.Fatalf("Movement failed: %v", err)
	}

	actionReq := ActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     SearchTreasureAction,
		Parameters: map[string]any{},
	}

	_, err = gm.ProcessHeroAction(actionReq)
	if err != nil {
		t.Fatalf("Action failed: %v", err)
	}

	// Verify state is used
	midTurnState := gm.GetTurnState()
	if !midTurnState.HasMoved {
		t.Error("Expected HasMoved to be true")
	}
	if !midTurnState.ActionTaken {
		t.Error("Expected ActionTaken to be true")
	}

	// Complete turn cycle (Hero → GM → Hero)
	err = gm.EndTurn() // End hero turn
	if err != nil {
		t.Fatalf("End hero turn failed: %v", err)
	}

	err = gm.EndTurn() // End GM turn
	if err != nil {
		t.Fatalf("End GM turn failed: %v", err)
	}

	// Verify state is reset for new hero turn
	newTurnState := gm.GetTurnState()
	if newTurnState.HasMoved {
		t.Error("Expected HasMoved to be false in new turn")
	}
	if newTurnState.ActionTaken {
		t.Error("Expected ActionTaken to be false in new turn")
	}
	if newTurnState.ActionsLeft != 1 {
		t.Errorf("Expected 1 action left, got %d", newTurnState.ActionsLeft)
	}
	// Movement starts at 0 and requires rolling dice
	if newTurnState.MovementLeft != 0 {
		t.Errorf("Expected 0 movement before rolling dice, got %d", newTurnState.MovementLeft)
	}

	// Roll dice and verify movement is available
	rollMovementDiceForTest(t, gm)
	rolledState := gm.GetTurnState()
	if rolledState.MovementLeft <= 0 {
		t.Error("Expected movement points after rolling dice")
	}
}

func TestTurnTransition_MultiPlayer(t *testing.T) {

	gm := createTestGameManager()

	// Add second player
	player2 := &Player{
		ID:       "player-2",
		Name:     "Test Hero 2",
		EntityID: "hero-2",
		Class:    Wizard,
		IsActive: true,
	}
	gm.turnManager.AddPlayer(player2)

	// Add hero to game state
	gm.gameState.Lock.Lock()
	gm.gameState.Entities["hero-2"] = protocol.TileAddress{X: 6, Y: 6}
	gm.gameState.Lock.Unlock()

	// First player's turn
	initialState := gm.GetTurnState()
	if initialState.ActivePlayerID != "player-1" {
		t.Errorf("Expected player-1 active, got: %s", initialState.ActivePlayerID)
	}

	// Take an action to allow turn ending
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

	// Complete first player's turn
	err = gm.EndTurn()
	if err != nil {
		t.Fatalf("End first player turn failed: %v", err)
	}

	// Should go to second player's turn (all heroes play before GM)
	player2State := gm.GetTurnState()
	if player2State.CurrentTurn != "hero" {
		t.Errorf("Expected hero turn after player-1, got: %s", player2State.CurrentTurn)
	}
	if player2State.ActivePlayerID != "player-2" {
		t.Errorf("Expected player-2 active, got: %s", player2State.ActivePlayerID)
	}

	// Player 2 takes an action
	actionReq2 := ActionRequest{
		PlayerID:   "player-2",
		EntityID:   "hero-2",
		Action:     SearchTreasureAction,
		Parameters: map[string]any{},
	}
	_, err = gm.ProcessHeroAction(actionReq2)
	if err != nil {
		t.Fatalf("Player 2 action failed: %v", err)
	}

	// End player 2's turn
	err = gm.EndTurn()
	if err != nil {
		t.Fatalf("End player 2 turn failed: %v", err)
	}

	// Now should be GM turn (all heroes have played)
	gmState := gm.GetTurnState()
	if gmState.CurrentTurn != "gamemaster" {
		t.Errorf("Expected gamemaster turn after all heroes, got: %s", gmState.CurrentTurn)
	}
}

func TestTurnTransition_CannotEndTurnWithoutAction(t *testing.T) {
	t.Skip("Test obsolete - design changed to allow ending turn without action (via PassTurnInstant). CanEndTurn is now always true to allow passing.")

	gm := createTestGameManager()

	// Try to end turn without taking any action
	initialState := gm.GetTurnState()
	if initialState.CanEndTurn {
		t.Error("Expected CanEndTurn to be false without action")
	}

	// Take an action
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

	// Now should be able to end turn
	finalState := gm.GetTurnState()
	if !finalState.CanEndTurn {
		t.Error("Expected CanEndTurn to be true after action")
	}

	// End turn should succeed
	err = gm.EndTurn()
	if err != nil {
		t.Fatalf("End turn failed: %v", err)
	}
}

func TestTurnTransition_PassTurnInstantEndsImmediately(t *testing.T) {

	gm := createTestGameManager()

	// Pass turn using instant action
	passReq := InstantActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     PassTurnInstant,
		Parameters: map[string]any{},
	}

	result, err := gm.heroActions.ProcessInstantAction(passReq)
	if err != nil {
		t.Fatalf("Pass turn instant failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful pass turn, got: %s", result.Message)
	}

	// Should immediately transition to GM turn
	turnState := gm.GetTurnState()
	if turnState.CurrentTurn != "gamemaster" {
		t.Errorf("Expected immediate transition to gamemaster turn, got: %s", turnState.CurrentTurn)
	}
}

func TestTurnTransition_ActionLimitsEnforced(t *testing.T) {

	gm := createTestGameManager()

	// Spawn a monster for the attack
	monster, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 4, Y: 14})
	if err != nil {
		t.Fatalf("Failed to spawn monster: %v", err)
	}

	// Take the main action
	actionReq := ActionRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": monster.ID,
		},
	}

	_, err = gm.ProcessHeroAction(actionReq)
	if err != nil {
		t.Fatalf("First action failed: %v", err)
	}

	// Try to take another main action (should fail)
	secondActionReq := ActionRequest{
		PlayerID:   "player-1",
		EntityID:   "hero-1",
		Action:     SearchTreasureAction,
		Parameters: map[string]any{},
	}

	_, err = gm.ProcessHeroAction(secondActionReq)
	if err == nil {
		t.Fatal("Expected second main action to fail")
	}

	// But instant actions should still work
	instantReq := InstantActionRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   UsePotionInstant,
		Parameters: map[string]any{
			"potionId": "healing-potion",
		},
	}

	result, err := gm.heroActions.ProcessInstantAction(instantReq)
	if err != nil {
		t.Fatalf("Instant action after main action failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful instant action, got: %s", result.Message)
	}
}

func TestTurnTransition_MovementLimitsEnforced(t *testing.T) {
	t.Skip("Test obsolete - we now use dice-based movement system where players can move multiple times until points run out. Movement is no longer limited to once per turn.")

	gm := createTestGameManager()

	// Roll movement dice before moving
	rollMovementDiceForTest(t, gm)

	// Take movement
	moveReq := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       1,
		DY:       0,
	}

	err := gm.ProcessMovement(moveReq)
	if err != nil {
		t.Fatalf("First movement failed: %v", err)
	}

	// Try to move again (should fail)
	secondMoveReq := protocol.RequestMove{
		EntityID: "hero-1",
		DX:       0,
		DY:       1,
	}

	err = gm.ProcessMovement(secondMoveReq)
	if err == nil {
		t.Fatal("Expected second movement to fail")
	}

	// Verify hero is still at first moved position
	gameState := gm.GetGameState()
	heroPos := gameState.Entities["hero-1"]
	if heroPos.X != 4 || heroPos.Y != 14 { // Original (3,14) + (1,0)
		t.Errorf("Expected hero at (4,14), got (%d,%d)", heroPos.X, heroPos.Y)
	}
}

func TestTurnTransition_CompleteGameCycle(t *testing.T) {

	gm := createTestGameManager()

	// Track multiple complete turn cycles (only 2 to avoid hitting walls)
	for cycle := 1; cycle <= 2; cycle++ {
		// Roll movement dice before moving
		rollMovementDiceForTest(t, gm)

		// Hero turn: movement + action
		// Alternate movement direction to avoid hitting walls
		dx := 1
		if cycle == 2 {
			dx = -1 // Move back left in cycle 2
		}
		moveReq := protocol.RequestMove{
			EntityID: "hero-1",
			DX:       dx,
			DY:       0,
		}

		err := gm.ProcessMovement(moveReq)
		if err != nil {
			t.Fatalf("Cycle %d movement failed: %v", cycle, err)
		}

		actionReq := ActionRequest{
			PlayerID:   "player-1",
			EntityID:   "hero-1",
			Action:     SearchTreasureAction,
			Parameters: map[string]any{},
		}

		_, err = gm.ProcessHeroAction(actionReq)
		if err != nil {
			t.Fatalf("Cycle %d action failed: %v", cycle, err)
		}

		// End hero turn
		err = gm.EndTurn()
		if err != nil {
			t.Fatalf("Cycle %d end hero turn failed: %v", cycle, err)
		}

		// Verify GM turn
		gmState := gm.GetTurnState()
		if gmState.CurrentTurn != "gamemaster" {
			t.Errorf("Cycle %d: Expected gamemaster turn, got: %s", cycle, gmState.CurrentTurn)
		}

		// End GM turn
		err = gm.EndTurn()
		if err != nil {
			t.Fatalf("Cycle %d end GM turn failed: %v", cycle, err)
		}

		// Verify back to hero turn with correct turn number
		heroState := gm.GetTurnState()
		if heroState.CurrentTurn != "hero" {
			t.Errorf("Cycle %d: Expected hero turn, got: %s", cycle, heroState.CurrentTurn)
		}
		expectedTurnNumber := cycle + 1
		if heroState.TurnNumber != expectedTurnNumber {
			t.Errorf("Cycle %d: Expected turn number %d, got %d", cycle, expectedTurnNumber, heroState.TurnNumber)
		}

		// Verify fresh state
		if heroState.HasMoved || heroState.ActionTaken {
			t.Errorf("Cycle %d: Expected fresh turn state", cycle)
		}
	}
}
