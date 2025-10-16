package main

import (
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

// handleRequestSelectStartingPosition handles a player selecting their starting position during quest setup
func handleRequestSelectStartingPosition(req protocol.RequestSelectStartingPosition, playerID string, gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()

	// Create position from request
	pos := Position{X: req.X, Y: req.Y}

	// Attempt to select starting position
	if err := dynamicTurnOrder.SelectStartingPosition(playerID, pos); err != nil {
		gameManager.logger.Printf("Failed to select starting position for player %s: %v", playerID, err)
		return
	}

	gameManager.logger.Printf("Player %s selected starting position (%d, %d)", playerID, req.X, req.Y)

	// Broadcast quest setup state update
	broadcastQuestSetupState(dynamicTurnOrder, hub, sequence)
}

// handleRequestQuestSetupToggleReady handles a player toggling ready status during quest setup
func handleRequestQuestSetupToggleReady(playerID string, ready bool, gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()

	// Set player ready status
	if err := dynamicTurnOrder.SetPlayerReady(playerID, ready); err != nil {
		gameManager.logger.Printf("Failed to set player ready for %s: %v", playerID, err)
		return
	}

	gameManager.logger.Printf("Player %s set ready status: %t", playerID, ready)

	// Broadcast quest setup state update
	broadcastQuestSetupState(dynamicTurnOrder, hub, sequence)

	// Check if all players are ready and can start quest
	if dynamicTurnOrder.AreAllPlayersReady() {
		// Spawn hero entities at selected positions FIRST
		if err := spawnHeroesAtStartingPositions(gameManager, hub, sequence); err != nil {
			gameManager.logger.Printf("Failed to spawn heroes at starting positions: %v", err)
			return
		}

		// Count hero players (exclude GM)
		turnManager := gameManager.turnManager
		heroPlayers := turnManager.GetHeroPlayers()
		heroCount := len(heroPlayers)

		gameManager.logger.Printf("All players ready, found %d hero player(s)", heroCount)

		if heroCount == 1 {
			// Single hero: auto-select and start their turn immediately
			gameManager.logger.Printf("Single hero detected, auto-starting their turn")

			// Transition from quest setup to hero election (required for state machine)
			if err := dynamicTurnOrder.StartQuestAfterSetup(); err != nil {
				gameManager.logger.Printf("Failed to start quest after setup: %v", err)
				return
			}

			// Get the single hero player
			singlePlayerID := heroPlayers[0].ID

			// Auto-elect the single player
			if err := dynamicTurnOrder.ElectSelfAsNextPlayer(singlePlayerID); err != nil {
				gameManager.logger.Printf("Failed to auto-elect single hero: %v", err)
				return
			}

			// Immediately confirm and start their turn
			playerID, err := dynamicTurnOrder.ConfirmElectionAndStartHeroTurn()
			if err != nil {
				gameManager.logger.Printf("Failed to start single hero turn: %v", err)
				return
			}

			// Start hero turn state
			player := turnManager.GetPlayer(playerID)
			if player != nil {
				turnStateManager := gameManager.GetTurnStateManager()
				gameState := gameManager.GetGameState()
				gameState.Lock.Lock()
				heroPos := gameState.Entities[player.EntityID]
				gameState.Lock.Unlock()

				if err := turnStateManager.StartHeroTurn(player.EntityID, playerID, heroPos); err != nil {
					gameManager.logger.Printf("Failed to start hero turn state: %v", err)
					return
				}

				// Broadcast hero turn state
				heroState := turnStateManager.GetHeroTurnState(player.EntityID)
				if heroState != nil {
					broadcastHeroTurnState(heroState, hub, sequence)
				}
			}

			gameManager.logger.Printf("Single hero %s turn started automatically", singlePlayerID)
		} else {
			// Multiple heroes: transition to election phase
			gameManager.logger.Printf("Multiple heroes detected, transitioning to hero election")

			if err := dynamicTurnOrder.StartQuestAfterSetup(); err != nil {
				gameManager.logger.Printf("Failed to start quest after setup: %v", err)
				return
			}
		}

		// Broadcast turn phase change
		broadcastTurnPhaseState(dynamicTurnOrder, hub, sequence)
	}
}

// handleRequestElectSelfAsNextPlayer handles a player electing themselves to go next
func handleRequestElectSelfAsNextPlayer(playerID string, gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()
	turnStateManager := gameManager.GetTurnStateManager()

	if err := dynamicTurnOrder.ElectSelfAsNextPlayer(playerID); err != nil {
		gameManager.logger.Printf("Failed to elect player %s: %v", playerID, err)
		return
	}

	gameManager.logger.Printf("Player %s elected themselves to go next", playerID)

	// Automatically start the hero's turn after election
	confirmedPlayerID, err := dynamicTurnOrder.ConfirmElectionAndStartHeroTurn()
	if err != nil {
		gameManager.logger.Printf("Failed to auto-start hero turn for player %s: %v", playerID, err)
		return
	}

	// Get player and start their turn state
	turnManager := gameManager.turnManager
	player := turnManager.GetPlayer(confirmedPlayerID)
	if player != nil {
		gameState := gameManager.GetGameState()
		gameState.Lock.Lock()
		heroPos := gameState.Entities[player.EntityID]
		gameState.Lock.Unlock()

		if err := turnStateManager.StartHeroTurn(player.EntityID, confirmedPlayerID, heroPos); err != nil {
			gameManager.logger.Printf("Failed to start hero turn state: %v", err)
			return
		}

		// Broadcast hero turn state
		heroState := turnStateManager.GetHeroTurnState(player.EntityID)
		if heroState != nil {
			broadcastHeroTurnState(heroState, hub, sequence)
		}
	}

	gameManager.logger.Printf("Auto-started turn for player %s after election", confirmedPlayerID)

	// Broadcast turn phase update
	broadcastTurnPhaseState(dynamicTurnOrder, hub, sequence)
}

// handleRequestCancelPlayerElection handles a player canceling their election
func handleRequestCancelPlayerElection(playerID string, gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()
	turnStateManager := gameManager.GetTurnStateManager()
	turnManager := gameManager.turnManager

	// Get player's entity ID
	player := turnManager.GetPlayer(playerID)
	if player == nil {
		gameManager.logger.Printf("Cannot cancel election: player %s not found", playerID)
		return
	}

	// Check if player has taken any actions (rolled movement or done action)
	heroState := turnStateManager.GetHeroTurnState(player.EntityID)
	if heroState != nil {
		if heroState.MovementDice.Rolled || heroState.ActionTaken {
			gameManager.logger.Printf("Cannot cancel election: player %s has already taken actions (rolled: %t, action: %t)",
				playerID, heroState.MovementDice.Rolled, heroState.ActionTaken)
			return
		}
	}

	if err := dynamicTurnOrder.CancelPlayerElection(playerID); err != nil {
		gameManager.logger.Printf("Failed to cancel election for player %s: %v", playerID, err)
		return
	}

	gameManager.logger.Printf("Player %s cancelled their election", playerID)

	// Remove hero turn state since they're canceling
	turnStateManager.RemoveHeroState(player.EntityID)

	// Check if only one hero remains - if so, auto-elect them
	heroPlayers := turnManager.GetHeroPlayers()
	heroesActed := dynamicTurnOrder.GetHeroesActedThisCycle()
	heroesRemaining := 0
	var lastHeroID string

	for _, heroPlayer := range heroPlayers {
		if !heroesActed[heroPlayer.ID] {
			heroesRemaining++
			lastHeroID = heroPlayer.ID
		}
	}

	gameManager.logger.Printf("After cancel: %d heroes remaining who haven't acted", heroesRemaining)

	if heroesRemaining == 1 {
		// Auto-elect the last remaining hero
		gameManager.logger.Printf("Auto-electing last remaining hero: %s", lastHeroID)

		if err := dynamicTurnOrder.ElectSelfAsNextPlayer(lastHeroID); err != nil {
			gameManager.logger.Printf("Failed to auto-elect last hero %s: %v", lastHeroID, err)
			return
		}

		// Start their turn
		confirmedPlayerID, err := dynamicTurnOrder.ConfirmElectionAndStartHeroTurn()
		if err != nil {
			gameManager.logger.Printf("Failed to auto-start last hero turn: %v", err)
			return
		}

		lastPlayer := turnManager.GetPlayer(confirmedPlayerID)
		if lastPlayer != nil {
			gameState := gameManager.GetGameState()
			gameState.Lock.Lock()
			heroPos := gameState.Entities[lastPlayer.EntityID]
			gameState.Lock.Unlock()

			if err := turnStateManager.StartHeroTurn(lastPlayer.EntityID, confirmedPlayerID, heroPos); err != nil {
				gameManager.logger.Printf("Failed to start last hero turn state: %v", err)
				return
			}

			// Broadcast hero turn state
			heroState := turnStateManager.GetHeroTurnState(lastPlayer.EntityID)
			if heroState != nil {
				broadcastHeroTurnState(heroState, hub, sequence)
			}
		}

		gameManager.logger.Printf("Auto-started turn for last remaining hero %s", confirmedPlayerID)
	}

	// Broadcast turn phase update
	broadcastTurnPhaseState(dynamicTurnOrder, hub, sequence)
}

// handleRequestConfirmElectionAndStartTurn handles confirming the election and starting the elected hero's turn
func handleRequestConfirmElectionAndStartTurn(gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()
	turnStateManager := gameManager.GetTurnStateManager()

	// Confirm election and get the elected player ID
	playerID, err := dynamicTurnOrder.ConfirmElectionAndStartHeroTurn()
	if err != nil {
		gameManager.logger.Printf("Failed to confirm election: %v", err)
		return
	}

	gameManager.logger.Printf("Confirmed election, starting turn for player %s", playerID)

	// Initialize hero turn state if needed
	// TODO: Get heroID from playerID mapping
	heroID := "hero-1" // Placeholder - need player→hero mapping
	if turnStateManager.GetHeroTurnState(heroID) == nil {
		// Get hero position from game state
		gameState := gameManager.GetGameState()
		gameState.Lock.Lock()
		heroPos := gameState.Entities[heroID]
		gameState.Lock.Unlock()

		// Start hero turn with dice roll
		if err := turnStateManager.StartHeroTurn(heroID, playerID, heroPos); err != nil {
			gameManager.logger.Printf("Failed to start hero turn: %v", err)
			return
		}

		// TODO: Roll movement dice through dice system
		// Dice rolling should be handled by the TurnManager/DiceSystem
		// For now, skip automatic dice rolling - let client request it
	}

	// Broadcast turn phase update
	broadcastTurnPhaseState(dynamicTurnOrder, hub, sequence)

	// Broadcast hero turn state update
	heroState := turnStateManager.GetHeroTurnState(heroID)
	if heroState != nil {
		broadcastHeroTurnState(heroState, hub, sequence)
	}
}

// handleRequestCompleteHeroTurn handles a hero completing their turn
func handleRequestCompleteHeroTurn(playerID string, gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()
	turnManager := gameManager.turnManager
	turnStateManager := gameManager.GetTurnStateManager()

	if err := dynamicTurnOrder.CompleteHeroTurn(); err != nil {
		gameManager.logger.Printf("Failed to complete hero turn for player %s: %v", playerID, err)
		return
	}

	gameManager.logger.Printf("Player %s completed their turn", playerID)

	// Check if only one hero remains - if so, auto-elect them
	heroPlayers := turnManager.GetHeroPlayers()
	heroesActed := dynamicTurnOrder.GetHeroesActedThisCycle()
	heroesRemaining := 0
	var lastHeroID string

	for _, heroPlayer := range heroPlayers {
		if !heroesActed[heroPlayer.ID] {
			heroesRemaining++
			lastHeroID = heroPlayer.ID
		}
	}

	gameManager.logger.Printf("After hero turn complete: %d heroes remaining who haven't acted", heroesRemaining)

	if heroesRemaining == 1 && dynamicTurnOrder.GetCurrentPhase() == "hero_election" {
		// Auto-elect the last remaining hero
		gameManager.logger.Printf("Auto-electing last remaining hero: %s", lastHeroID)

		if err := dynamicTurnOrder.ElectSelfAsNextPlayer(lastHeroID); err != nil {
			gameManager.logger.Printf("Failed to auto-elect last hero %s: %v", lastHeroID, err)
			return
		}

		// Start their turn
		confirmedPlayerID, err := dynamicTurnOrder.ConfirmElectionAndStartHeroTurn()
		if err != nil {
			gameManager.logger.Printf("Failed to auto-start last hero turn: %v", err)
			return
		}

		lastPlayer := turnManager.GetPlayer(confirmedPlayerID)
		if lastPlayer != nil {
			gameState := gameManager.GetGameState()
			gameState.Lock.Lock()
			heroPos := gameState.Entities[lastPlayer.EntityID]
			gameState.Lock.Unlock()

			if err := turnStateManager.StartHeroTurn(lastPlayer.EntityID, confirmedPlayerID, heroPos); err != nil {
				gameManager.logger.Printf("Failed to start last hero turn state: %v", err)
				return
			}

			// Broadcast hero turn state
			heroState := turnStateManager.GetHeroTurnState(lastPlayer.EntityID)
			if heroState != nil {
				broadcastHeroTurnState(heroState, hub, sequence)
			}
		}

		gameManager.logger.Printf("Auto-started turn for last remaining hero %s", confirmedPlayerID)
	}

	// Broadcast turn phase update
	broadcastTurnPhaseState(dynamicTurnOrder, hub, sequence)
}

// handleRequestCompleteGMTurn handles the GM completing their turn
func handleRequestCompleteGMTurn(gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()
	turnStateManager := gameManager.GetTurnStateManager()

	if err := dynamicTurnOrder.CompleteGMTurn(); err != nil {
		gameManager.logger.Printf("Failed to complete GM turn: %v", err)
		return
	}

	gameManager.logger.Printf("GM turn completed, starting new hero cycle %d", dynamicTurnOrder.GetCycleNumber())

	// Reset all monster turn states for new cycle
	monsterStates := turnStateManager.GetAllMonsterStates()
	for monsterID := range monsterStates {
		turnStateManager.RemoveMonsterState(monsterID)
	}

	// Broadcast turn phase update
	broadcastTurnPhaseState(dynamicTurnOrder, hub, sequence)

	// Broadcast monster states cleared
	broadcastEvent(hub, sequence, "AllMonsterStatesSync", protocol.AllMonsterStatesSync{
		MonsterStates: make(map[string]*protocol.MonsterTurnStateChanged),
	})
}

// broadcastTurnPhaseState broadcasts the current turn phase state to all clients
func broadcastTurnPhaseState(dynamicTurnOrder *DynamicTurnOrderManager, hub *ws.Hub, sequence *uint64) {
	heroesActed := dynamicTurnOrder.GetHeroesActedThisCycle()
	heroesActedIDs := make([]string, 0, len(heroesActed))
	for playerID := range heroesActed {
		heroesActedIDs = append(heroesActedIDs, playerID)
	}

	// TODO: Get eligible heroes from actual player list
	eligibleHeroIDs := []string{} // Placeholder

	patch := protocol.TurnPhaseChanged{
		CurrentPhase:       string(dynamicTurnOrder.GetCurrentPhase()),
		CycleNumber:        dynamicTurnOrder.GetCycleNumber(),
		ActiveHeroPlayerID: dynamicTurnOrder.GetActiveHeroPlayerID(),
		ElectedPlayerID:    dynamicTurnOrder.GetElectedPlayer(),
		HeroesActedIDs:     heroesActedIDs,
		EligibleHeroIDs:    eligibleHeroIDs,
	}

	broadcastEvent(hub, sequence, "TurnPhaseChanged", patch)
}

// broadcastQuestSetupState broadcasts the quest setup state to all clients
func broadcastQuestSetupState(dynamicTurnOrder *DynamicTurnOrderManager, hub *ws.Hub, sequence *uint64) {
	// Get players ready map
	playersReady := dynamicTurnOrder.GetPlayersReady()
	if playersReady == nil {
		playersReady = make(map[string]bool)
	}

	// Get starting positions
	startPositions := dynamicTurnOrder.GetPlayerStartPositions()
	playerStartPositions := make(map[string]protocol.StartPositionInfo)
	for playerID, pos := range startPositions {
		playerStartPositions[playerID] = protocol.StartPositionInfo{
			X: pos.X,
			Y: pos.Y,
		}
	}

	patch := protocol.QuestSetupStateChanged{
		PlayersReady:         playersReady,
		PlayerStartPositions: playerStartPositions,
		AllPlayersReady:      dynamicTurnOrder.AreAllPlayersReady(),
	}

	broadcastEvent(hub, sequence, "QuestSetupStateChanged", patch)
}

// broadcastHeroTurnState broadcasts a hero turn state update
func broadcastHeroTurnState(state *HeroTurnState, hub *ws.Hub, sequence *uint64) {
	// TODO: Add ActiveEffects and LocationSearches to TurnStateChanged protocol
	// Currently these fields are tracked in HeroTurnState but not transmitted to clients

	// Broadcast TurnStateChanged with hero turn state details
	broadcastEvent(hub, sequence, "TurnStateChanged", protocol.TurnStateChanged{
		HeroID:              state.HeroID,
		PlayerID:            state.PlayerID,
		TurnNumber:          state.TurnNumber,
		CurrentTurn:         state.PlayerID,
		CurrentPhase:        "hero_active",
		ActivePlayerID:      state.PlayerID,
		ActionsLeft:         boolToInt(!state.ActionTaken),
		MovementLeft:        state.MovementDice.MovementRemaining,
		HasMoved:            state.HasMoved,
		ActionTaken:         state.ActionTaken,
		CanEndTurn:          state.ActionTaken || state.MovementDice.MovementRemaining == 0,
		MovementDiceRolled:  state.MovementDice.Rolled,
		MovementDiceResults: state.MovementDice.DiceResults,
		MovementTotal:       state.MovementDice.TotalMovement,
		MovementUsed:        state.MovementDice.MovementUsed,
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// spawnHeroesAtStartingPositions spawns hero entities at their selected starting positions
func spawnHeroesAtStartingPositions(gameManager *GameManager, hub *ws.Hub, sequence *uint64) error {
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()
	gameState := gameManager.GetGameState()
	turnManager := gameManager.turnManager

	// Get selected starting positions from dynamic turn order
	startingPositions := dynamicTurnOrder.GetPlayerStartPositions()

	// Iterate through all players and spawn them at their positions
	gameState.Lock.Lock()
	defer gameState.Lock.Unlock()

	for playerID, pos := range startingPositions {
		// Get player from turn manager
		player := turnManager.GetPlayer(playerID)
		if player == nil {
			gameManager.logger.Printf("Warning: Player %s not found in turn manager", playerID)
			continue
		}

		// Create tile address for entity
		tileAddr := protocol.TileAddress{
			SegmentID: "",
			X:         pos.X,
			Y:         pos.Y,
		}

		// Add entity to game state
		gameState.Entities[player.EntityID] = tileAddr
		gameManager.logger.Printf("Spawned hero %s (player %s) at (%d, %d)", player.EntityID, playerID, pos.X, pos.Y)

		// Broadcast entity update to notify clients
		broadcastEvent(hub, sequence, "EntityUpdated", protocol.EntityUpdated{
			ID:   player.EntityID,
			Tile: tileAddr,
		})
	}

	return nil
}
