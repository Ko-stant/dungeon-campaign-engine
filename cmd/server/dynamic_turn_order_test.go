package main

import (
	"testing"
)

func TestNewDynamicTurnOrderManager(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	if dtom.GetCurrentPhase() != QuestSetupPhase {
		t.Errorf("Expected initial phase to be QuestSetupPhase, got %s", dtom.GetCurrentPhase())
	}
	if dtom.GetCycleNumber() != 0 {
		t.Errorf("Expected initial cycle number to be 0, got %d", dtom.GetCycleNumber())
	}
}

func TestDynamicTurnOrderManager_QuestSetup(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Register players
	dtom.RegisterPlayer("player-1")
	dtom.RegisterPlayer("player-2")
	dtom.RegisterPlayer("player-3")

	// Select starting positions
	pos1 := Position{X: 1, Y: 1}
	err := dtom.SelectStartingPosition("player-1", pos1)
	if err != nil {
		t.Fatalf("Failed to select starting position: %v", err)
	}

	pos2 := Position{X: 2, Y: 1}
	err = dtom.SelectStartingPosition("player-2", pos2)
	if err != nil {
		t.Fatalf("Failed to select starting position: %v", err)
	}

	pos3 := Position{X: 3, Y: 1}
	err = dtom.SelectStartingPosition("player-3", pos3)
	if err != nil {
		t.Fatalf("Failed to select starting position: %v", err)
	}

	// Set players ready
	dtom.SetPlayerReady("player-1", true)
	dtom.SetPlayerReady("player-2", true)

	// Not all ready yet
	if dtom.AreAllPlayersReady() {
		t.Error("Expected not all players to be ready")
	}

	// Set last player ready
	dtom.SetPlayerReady("player-3", true)

	// All ready now
	if !dtom.AreAllPlayersReady() {
		t.Error("Expected all players to be ready")
	}

	// Start quest
	err = dtom.StartQuestAfterSetup()
	if err != nil {
		t.Fatalf("Failed to start quest: %v", err)
	}

	// Should now be in hero election phase
	if dtom.GetCurrentPhase() != HeroPhaseElection {
		t.Errorf("Expected phase to be HeroPhaseElection, got %s", dtom.GetCurrentPhase())
	}
	if dtom.GetCycleNumber() != 1 {
		t.Errorf("Expected cycle number to be 1, got %d", dtom.GetCycleNumber())
	}
}

func TestDynamicTurnOrderManager_StartQuestBeforeReady(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	dtom.RegisterPlayer("player-1")
	dtom.SetPlayerReady("player-1", false)

	// Try to start quest before ready
	err := dtom.StartQuestAfterSetup()
	if err == nil {
		t.Error("Expected error when starting quest before all players ready")
	}
}

func TestDynamicTurnOrderManager_PlayerElection(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Setup and start quest
	dtom.RegisterPlayer("player-1")
	dtom.RegisterPlayer("player-2")
	dtom.SetPlayerReady("player-1", true)
	dtom.SetPlayerReady("player-2", true)
	dtom.StartQuestAfterSetup()

	// Player 1 elects themselves
	err := dtom.ElectSelfAsNextPlayer("player-1")
	if err != nil {
		t.Fatalf("Failed to elect player: %v", err)
	}

	electedPlayer := dtom.GetElectedPlayer()
	if electedPlayer != "player-1" {
		t.Errorf("Expected elected player 'player-1', got '%s'", electedPlayer)
	}

	// Player 2 tries to elect themselves (should fail)
	err = dtom.ElectSelfAsNextPlayer("player-2")
	if err == nil {
		t.Error("Expected error when second player tries to elect while another is elected")
	}

	// Player 1 cancels election
	err = dtom.CancelPlayerElection("player-1")
	if err != nil {
		t.Fatalf("Failed to cancel election: %v", err)
	}

	electedPlayer = dtom.GetElectedPlayer()
	if electedPlayer != "" {
		t.Errorf("Expected no elected player after cancellation, got '%s'", electedPlayer)
	}

	// Now player 2 can elect
	err = dtom.ElectSelfAsNextPlayer("player-2")
	if err != nil {
		t.Fatalf("Failed to elect player 2: %v", err)
	}
}

func TestDynamicTurnOrderManager_ConfirmElectionAndStartHeroTurn(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Setup
	dtom.RegisterPlayer("player-1")
	dtom.RegisterPlayer("player-2")
	dtom.SetPlayerReady("player-1", true)
	dtom.SetPlayerReady("player-2", true)
	dtom.StartQuestAfterSetup()

	// Player 1 elects themselves
	dtom.ElectSelfAsNextPlayer("player-1")

	// Confirm election
	playerID, err := dtom.ConfirmElectionAndStartHeroTurn()
	if err != nil {
		t.Fatalf("Failed to confirm election: %v", err)
	}

	if playerID != "player-1" {
		t.Errorf("Expected confirmed player 'player-1', got '%s'", playerID)
	}

	// Should now be in hero phase active
	if dtom.GetCurrentPhase() != HeroPhaseActive {
		t.Errorf("Expected phase to be HeroPhaseActive, got %s", dtom.GetCurrentPhase())
	}

	activePlayer := dtom.GetActiveHeroPlayerID()
	if activePlayer != "player-1" {
		t.Errorf("Expected active player 'player-1', got '%s'", activePlayer)
	}

	// Election should be cleared
	if dtom.GetElectedPlayer() != "" {
		t.Error("Expected elected player to be cleared after confirmation")
	}
}

func TestDynamicTurnOrderManager_CompleteHeroTurn(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Setup with 2 players
	dtom.RegisterPlayer("player-1")
	dtom.RegisterPlayer("player-2")
	dtom.SetPlayerReady("player-1", true)
	dtom.SetPlayerReady("player-2", true)
	dtom.StartQuestAfterSetup()

	// Player 1 acts
	dtom.ElectSelfAsNextPlayer("player-1")
	dtom.ConfirmElectionAndStartHeroTurn()

	// Complete player 1's turn
	err := dtom.CompleteHeroTurn()
	if err != nil {
		t.Fatalf("Failed to complete hero turn: %v", err)
	}

	// Should be back in election phase
	if dtom.GetCurrentPhase() != HeroPhaseElection {
		t.Errorf("Expected phase to be HeroPhaseElection, got %s", dtom.GetCurrentPhase())
	}

	// Player 1 should be marked as acted
	heroesActed := dtom.GetHeroesActedThisCycle()
	if !heroesActed["player-1"] {
		t.Error("Expected player-1 to be marked as acted")
	}

	// Check eligible heroes
	eligible := dtom.GetEligibleHeroes([]string{"player-1", "player-2"})
	if len(eligible) != 1 {
		t.Errorf("Expected 1 eligible hero, got %d", len(eligible))
	}
	if eligible[0] != "player-2" {
		t.Errorf("Expected eligible hero 'player-2', got '%s'", eligible[0])
	}
}

func TestDynamicTurnOrderManager_AdvanceToGMPhase(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Setup with 2 players
	dtom.RegisterPlayer("player-1")
	dtom.RegisterPlayer("player-2")
	dtom.SetPlayerReady("player-1", true)
	dtom.SetPlayerReady("player-2", true)
	dtom.StartQuestAfterSetup()

	// Player 1 acts
	dtom.ElectSelfAsNextPlayer("player-1")
	dtom.ConfirmElectionAndStartHeroTurn()
	dtom.CompleteHeroTurn()

	// Player 2 acts
	dtom.ElectSelfAsNextPlayer("player-2")
	dtom.ConfirmElectionAndStartHeroTurn()
	err := dtom.CompleteHeroTurn()
	if err != nil {
		t.Fatalf("Failed to complete hero turn: %v", err)
	}

	// Should now advance to GM phase
	if dtom.GetCurrentPhase() != GMPhase {
		t.Errorf("Expected phase to be GMPhase, got %s", dtom.GetCurrentPhase())
	}

	// Cycle number should still be 1
	if dtom.GetCycleNumber() != 1 {
		t.Errorf("Expected cycle number 1, got %d", dtom.GetCycleNumber())
	}
}

func TestDynamicTurnOrderManager_CompleteGMTurn(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Setup and get to GM phase
	dtom.RegisterPlayer("player-1")
	dtom.SetPlayerReady("player-1", true)
	dtom.StartQuestAfterSetup()
	dtom.ElectSelfAsNextPlayer("player-1")
	dtom.ConfirmElectionAndStartHeroTurn()
	dtom.CompleteHeroTurn() // Should advance to GM phase

	// Complete GM turn
	err := dtom.CompleteGMTurn()
	if err != nil {
		t.Fatalf("Failed to complete GM turn: %v", err)
	}

	// Should be back in hero election phase
	if dtom.GetCurrentPhase() != HeroPhaseElection {
		t.Errorf("Expected phase to be HeroPhaseElection, got %s", dtom.GetCurrentPhase())
	}

	// Cycle number should be incremented
	if dtom.GetCycleNumber() != 2 {
		t.Errorf("Expected cycle number 2, got %d", dtom.GetCycleNumber())
	}

	// Heroes acted should be reset
	heroesActed := dtom.GetHeroesActedThisCycle()
	if len(heroesActed) != 0 {
		t.Errorf("Expected heroes acted to be reset, got %d heroes", len(heroesActed))
	}
}

func TestDynamicTurnOrderManager_CannotElectAfterActing(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Setup
	dtom.RegisterPlayer("player-1")
	dtom.RegisterPlayer("player-2")
	dtom.SetPlayerReady("player-1", true)
	dtom.SetPlayerReady("player-2", true)
	dtom.StartQuestAfterSetup()

	// Player 1 acts
	dtom.ElectSelfAsNextPlayer("player-1")
	dtom.ConfirmElectionAndStartHeroTurn()
	dtom.CompleteHeroTurn()

	// Player 1 tries to elect again
	err := dtom.ElectSelfAsNextPlayer("player-1")
	if err == nil {
		t.Error("Expected error when player tries to elect after already acting")
	}
}

func TestDynamicTurnOrderManager_PhaseValidation(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Try to elect during setup
	err := dtom.ElectSelfAsNextPlayer("player-1")
	if err == nil {
		t.Error("Expected error when electing during setup phase")
	}

	// Setup and start quest
	dtom.RegisterPlayer("player-1")
	dtom.SetPlayerReady("player-1", true)
	dtom.StartQuestAfterSetup()

	// Try to complete hero turn during election
	err = dtom.CompleteHeroTurn()
	if err == nil {
		t.Error("Expected error when completing hero turn during election phase")
	}

	// Try to complete GM turn during election
	err = dtom.CompleteGMTurn()
	if err == nil {
		t.Error("Expected error when completing GM turn during election phase")
	}
}

func TestDynamicTurnOrderManager_IsHelpers(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Initially in setup
	if !dtom.IsQuestSetup() {
		t.Error("Expected IsQuestSetup to be true initially")
	}
	if dtom.IsHeroTurn() {
		t.Error("Expected IsHeroTurn to be false during setup")
	}
	if dtom.IsGMTurn() {
		t.Error("Expected IsGMTurn to be false during setup")
	}

	// Start quest (hero election)
	dtom.RegisterPlayer("player-1")
	dtom.SetPlayerReady("player-1", true)
	dtom.StartQuestAfterSetup()

	if dtom.IsQuestSetup() {
		t.Error("Expected IsQuestSetup to be false after starting")
	}
	if !dtom.IsHeroTurn() {
		t.Error("Expected IsHeroTurn to be true during election")
	}
	if dtom.IsGMTurn() {
		t.Error("Expected IsGMTurn to be false during hero phase")
	}

	// Advance to GM phase
	dtom.ElectSelfAsNextPlayer("player-1")
	dtom.ConfirmElectionAndStartHeroTurn()
	dtom.CompleteHeroTurn()

	if dtom.IsQuestSetup() {
		t.Error("Expected IsQuestSetup to be false during GM phase")
	}
	if dtom.IsHeroTurn() {
		t.Error("Expected IsHeroTurn to be false during GM phase")
	}
	if !dtom.IsGMTurn() {
		t.Error("Expected IsGMTurn to be true during GM phase")
	}
}

func TestDynamicTurnOrderManager_CanPlayerAct(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Setup
	dtom.RegisterPlayer("player-1")
	dtom.RegisterPlayer("player-2")
	dtom.SetPlayerReady("player-1", true)
	dtom.SetPlayerReady("player-2", true)
	dtom.StartQuestAfterSetup()

	// During election, no one can act
	if dtom.CanPlayerAct("player-1") {
		t.Error("Expected player-1 to not be able to act during election")
	}

	// Player 1 confirmed as active
	dtom.ElectSelfAsNextPlayer("player-1")
	dtom.ConfirmElectionAndStartHeroTurn()

	// Player 1 can act
	if !dtom.CanPlayerAct("player-1") {
		t.Error("Expected player-1 to be able to act when active")
	}

	// Player 2 cannot act
	if dtom.CanPlayerAct("player-2") {
		t.Error("Expected player-2 to not be able to act when not active")
	}
}

func TestDynamicTurnOrderManager_GetStateString(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Setup phase
	dtom.RegisterPlayer("player-1")
	stateStr := dtom.GetStateString()
	if stateStr == "" {
		t.Error("Expected non-empty state string during setup")
	}

	// Election phase
	dtom.SetPlayerReady("player-1", true)
	dtom.StartQuestAfterSetup()
	stateStr = dtom.GetStateString()
	if stateStr == "" {
		t.Error("Expected non-empty state string during election")
	}

	// Active hero phase
	dtom.ElectSelfAsNextPlayer("player-1")
	dtom.ConfirmElectionAndStartHeroTurn()
	stateStr = dtom.GetStateString()
	if stateStr == "" {
		t.Error("Expected non-empty state string during hero turn")
	}

	// GM phase
	dtom.CompleteHeroTurn()
	stateStr = dtom.GetStateString()
	if stateStr == "" {
		t.Error("Expected non-empty state string during GM turn")
	}
}

func TestDynamicTurnOrderManager_GetStateSummary(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	dtom.RegisterPlayer("player-1")
	dtom.SetPlayerReady("player-1", true)
	dtom.StartQuestAfterSetup()

	summary := dtom.GetStateSummary()
	if summary == nil {
		t.Fatal("Expected non-nil state summary")
	}

	if summary["current_phase"] != string(HeroPhaseElection) {
		t.Errorf("Expected current_phase to be HeroPhaseElection, got %v", summary["current_phase"])
	}
	if summary["cycle_number"] != 1 {
		t.Errorf("Expected cycle_number to be 1, got %v", summary["cycle_number"])
	}
}

func TestDynamicTurnOrderManager_MultipleHeroCycles(t *testing.T) {
	logger := &MockLogger{messages: []string{}}
	dtom := NewDynamicTurnOrderManager(logger)

	// Setup with 2 players
	dtom.RegisterPlayer("player-1")
	dtom.RegisterPlayer("player-2")
	dtom.SetPlayerReady("player-1", true)
	dtom.SetPlayerReady("player-2", true)
	dtom.StartQuestAfterSetup()

	// Cycle 1
	dtom.ElectSelfAsNextPlayer("player-1")
	dtom.ConfirmElectionAndStartHeroTurn()
	dtom.CompleteHeroTurn()

	dtom.ElectSelfAsNextPlayer("player-2")
	dtom.ConfirmElectionAndStartHeroTurn()
	dtom.CompleteHeroTurn()

	if dtom.GetCycleNumber() != 1 {
		t.Errorf("Expected cycle 1, got %d", dtom.GetCycleNumber())
	}
	if dtom.GetCurrentPhase() != GMPhase {
		t.Errorf("Expected GM phase, got %s", dtom.GetCurrentPhase())
	}

	// Complete GM turn - start cycle 2
	dtom.CompleteGMTurn()

	if dtom.GetCycleNumber() != 2 {
		t.Errorf("Expected cycle 2, got %d", dtom.GetCycleNumber())
	}
	if dtom.GetCurrentPhase() != HeroPhaseElection {
		t.Errorf("Expected hero election, got %s", dtom.GetCurrentPhase())
	}

	// Cycle 2
	dtom.ElectSelfAsNextPlayer("player-2")
	dtom.ConfirmElectionAndStartHeroTurn()
	dtom.CompleteHeroTurn()

	dtom.ElectSelfAsNextPlayer("player-1")
	dtom.ConfirmElectionAndStartHeroTurn()
	dtom.CompleteHeroTurn()

	if dtom.GetCycleNumber() != 2 {
		t.Errorf("Expected cycle 2, got %d", dtom.GetCycleNumber())
	}
	if dtom.GetCurrentPhase() != GMPhase {
		t.Errorf("Expected GM phase, got %s", dtom.GetCurrentPhase())
	}
}
