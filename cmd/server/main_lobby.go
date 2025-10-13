package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

// mainWithLobby starts the server in lobby mode, where players join before the game starts
func mainWithLobby() {
	log.Printf("=== Starting HeroQuest Server in Lobby Mode ===")

	// Get debug configuration from environment
	debugConfig := GetDebugConfigFromEnv()

	// Create basic dependencies
	hub := ws.NewHub()
	sequenceGen := NewSequenceGenerator()
	logger := NewLogger()

	// Load content manager for hero data
	contentManager := NewContentManager(logger)
	if err := contentManager.LoadCampaign("base"); err != nil {
		log.Fatalf("Failed to load campaign content: %v", err)
	}

	// Create lobby server
	lobbyServer := NewLobbyServer(contentManager, sequenceGen)

	// Game state (will be initialized after lobby phase)
	var gameManager *GameManager
	var state *GameState
	var quest *geometry.QuestDefinition
	var furnitureSystem *FurnitureSystem

	// Flag to track if game has started
	gameStarted := false

	// Set game start handler
	lobbyServer.SetGameStartHandler(func(gameMasterID string, heroPlayers map[string]string) error {
		log.Printf("Initializing game with GM=%s and heroes=%v", gameMasterID, heroPlayers)

		// Load game content
		board, loadedQuest, err := loadGameContent()
		if err != nil {
			return fmt.Errorf("failed to load game content: %w", err)
		}
		quest = loadedQuest

		// Initialize furniture system
		furnitureSystem = NewFurnitureSystem(log.New(os.Stdout, "", log.LstdFlags))
		if err := furnitureSystem.LoadFurnitureDefinitions("content"); err != nil {
			log.Printf("Warning: Failed to load furniture definitions: %v", err)
		}
		if err := furnitureSystem.CreateFurnitureInstancesFromQuest(quest); err != nil {
			log.Printf("Warning: Failed to create furniture instances: %v", err)
		}

		// Create broadcaster
		broadcaster := NewBroadcaster(hub, sequenceGen)

		// Initialize game manager
		gameManager, err = NewGameManagerWithFurniture(broadcaster, logger, sequenceGen, debugConfig, furnitureSystem, quest)
		if err != nil {
			return fmt.Errorf("failed to initialize game manager: %w", err)
		}

		// Initialize game state
		state, _, err = initializeGameState(board, quest, furnitureSystem)
		if err != nil {
			return fmt.Errorf("failed to initialize game state: %w", err)
		}

		// Create players from lobby selections
		inventoryManager := NewInventoryManager(contentManager, logger)
		entityIDCounter := 1

		for playerID, heroClassID := range heroPlayers {
			// Load hero card
			heroCard, ok := contentManager.GetHeroCard(heroClassID)
			if !ok {
				return fmt.Errorf("hero class not found: %s", heroClassID)
			}

			// Create entity ID
			entityID := fmt.Sprintf("hero-%d", entityIDCounter)
			entityIDCounter++

			// Initialize inventory
			if err := inventoryManager.InitializeHeroInventory(entityID); err != nil {
				return fmt.Errorf("failed to initialize inventory for %s: %w", entityID, err)
			}

			// Create player from content
			player, err := NewPlayerFromContent(playerID, entityID, heroCard, contentManager, inventoryManager)
			if err != nil {
				return fmt.Errorf("failed to create player %s: %w", playerID, err)
			}

			// Add to turn manager
			if err := gameManager.turnManager.AddPlayer(player); err != nil {
				return fmt.Errorf("failed to add player to turn manager: %w", err)
			}

			// Add hero entity to game state at starting position
			// TODO: Get starting positions from quest
			startPos := protocol.TileAddress{X: 7, Y: 11} // Default starting position
			state.Lock.Lock()
			state.Entities[entityID] = startPos
			state.Lock.Unlock()

			log.Printf("Created player %s as %s (%s)", playerID, heroCard.Name, entityID)
		}

		// Spawn monsters from quest definition
		if err := createMonstersFromQuest(quest, gameManager.GetMonsterSystem()); err != nil {
			return fmt.Errorf("failed to create monsters: %w", err)
		}

		// Mark all connections as no longer in lobby
		for _, conn := range lobbyServer.GetConnectionManager().GetAllConnections() {
			lobbyServer.GetConnectionManager().SetInLobby(conn, false)
		}

		gameStarted = true
		log.Printf("Game initialization complete!")

		// Broadcast game snapshot to all players
		// This will be handled by the WebSocket handler when clients reconnect or refresh

		return nil
	})

	// Setup HTTP handlers
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	assetsServer := http.FileServer(http.Dir("."))
	mux.Handle("/assets/", http.StripPrefix("/", assetsServer))

	// Register debug endpoints if enabled
	if debugConfig.Enabled {
		// Debug endpoints will be registered once game starts
		log.Printf("Debug mode enabled (endpoints will be available after game starts)")
	}

	// Lobby page handler
	mux.HandleFunc("/lobby", func(w http.ResponseWriter, r *http.Request) {
		if gameStarted {
			// Redirect to game if already started
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Get available heroes from content
		heroes := contentManager.GetAllHeroes()
		heroIDs := make([]string, 0, len(heroes))
		for id := range heroes {
			heroIDs = append(heroIDs, id)
		}

		if err := views.LobbyPage(heroIDs).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Main page handler (game or redirect to lobby)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !gameStarted {
			// Redirect to lobby if game hasn't started
			http.Redirect(w, r, "/lobby", http.StatusSeeOther)
			return
		}

		// Serve game page (similar to original mainWithGameManager)
		currentGameState := gameManager.GetGameState()
		currentGameState.Lock.Lock()

		// Get first hero for initial view (TODO: support multiple heroes properly)
		var hero protocol.TileAddress
		for entityID := range currentGameState.Entities {
			if len(entityID) >= 4 && entityID[:4] == "hero" {
				hero = currentGameState.Entities[entityID]
				break
			}
		}

		var revealed []int
		for id := range currentGameState.RevealedRegions {
			revealed = append(revealed, id)
		}
		currentGameState.Lock.Unlock()

		visibleNow := computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)

		// Build entities list (only hero players, not GM)
		entities := []protocol.EntityLite{}
		for playerID := range lobbyServer.lobby.players {
			// Skip GM players - they don't have entities
			lobbyPlayer, _ := lobbyServer.lobby.GetPlayer(playerID)
			if lobbyPlayer != nil && lobbyPlayer.Role == RoleGameMaster {
				continue
			}

			player := gameManager.turnManager.GetPlayer(playerID)
			if player == nil {
				continue
			}

			heroHP := &protocol.HP{Current: 0, Max: 0}
			heroMindPoints := &protocol.HP{Current: 0, Max: 0}
			if player.Character != nil {
				heroHP.Current = player.Character.CurrentBody
				heroHP.Max = player.Character.BaseStats.BodyPoints
				heroMindPoints.Current = player.Character.CurrentMind
				heroMindPoints.Max = player.Character.BaseStats.MindPoints
			}

			currentGameState.Lock.Lock()
			entityPos := currentGameState.Entities[player.EntityID]
			currentGameState.Lock.Unlock()

			entities = append(entities, protocol.EntityLite{
				ID:         player.EntityID,
				Kind:       "hero",
				Tile:       entityPos,
				HP:         heroHP,
				MindPoints: heroMindPoints,
				Tags:       []string{string(player.Class)},
			})
		}

		// Include doors
		currentGameState.Lock.Lock()
		thresholds := make([]protocol.ThresholdLite, 0, len(currentGameState.KnownDoors))
		for id := range currentGameState.KnownDoors {
			if info, exists := currentGameState.Doors[id]; exists {
				thresholds = append(thresholds, protocol.ThresholdLite{
					ID:          id,
					X:           info.Edge.X,
					Y:           info.Edge.Y,
					Orientation: string(info.Edge.Orientation),
					Kind:        "DoorSocket",
					State:       info.State,
				})
			}
		}
		currentGameState.Lock.Unlock()

		blockingWalls, _ := getVisibleBlockingWalls(state, hero, quest)

		known := make([]int, 0, len(state.KnownRegions))
		for rid := range state.KnownRegions {
			known = append(known, rid)
		}

		turnState := gameManager.GetTurnState()
		furniture := gameManager.GetFurnitureForSnapshot()
		monsters := gameManager.GetMonstersForSnapshot()
		heroTurnStates := gameManager.GetHeroTurnStatesForSnapshot()

		s := protocol.Snapshot{
			MapID:             "dev-map",
			PackID:            "dev-pack@v1",
			Turn:              turnState.TurnNumber,
			LastEventID:       0,
			MapWidth:          state.Segment.Width,
			MapHeight:         state.Segment.Height,
			RegionsCount:      state.RegionMap.RegionsCount,
			TileRegionIDs:     state.RegionMap.TileRegionIDs,
			RevealedRegionIDs: revealed,
			DoorStates:        []byte{},
			Entities:          entities,
			Variables: map[string]any{
				"ui.debug":             debugConfig.Enabled,
				"turn.number":          turnState.TurnNumber,
				"turn.current":         turnState.CurrentTurn,
				"turn.phase":           turnState.CurrentPhase,
				"turn.playerId":        turnState.ActivePlayerID,
				"turn.actions":         turnState.ActionsLeft,
				"turn.movement":        turnState.MovementLeft,
				"turn.canEnd":          turnState.CanEndTurn,
				"turn.movementRolled":  turnState.MovementDiceRolled,
				"turn.movementRolls":   turnState.MovementRolls,
				"turn.hasMoved":        turnState.HasMoved,
				"turn.movementStarted": turnState.MovementStarted,
				"turn.movementAction":  turnState.MovementAction,
				"turn.actionTaken":     turnState.ActionTaken,
			},
			ProtocolVersion:  "v0",
			Thresholds:       thresholds,
			BlockingWalls:    blockingWalls,
			Furniture:        furniture,
			Monsters:         monsters,
			HeroTurnStates:   heroTurnStates,
			VisibleRegionIDs: visibleNow,
			CorridorRegionID: state.CorridorRegion,
			KnownRegionIDs:   known,
		}

		if err := views.IndexPage(s).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// WebSocket handler
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		hub.Add(conn)

		// Register connection with lobby server
		playerID := lobbyServer.HandleNewConnection(conn)
		log.Printf("WebSocket connected: %s", playerID)

		// Send player ID to client
		playerIDMessage, _ := json.Marshal(protocol.PatchEnvelope{
			Sequence: 0,
			EventID:  0,
			Type:     "PlayerIDAssigned",
			Payload:  protocol.PlayerIDAssigned{PlayerID: playerID},
		})
		_ = conn.Write(context.Background(), websocket.MessageText, playerIDMessage)

		// If game has started, send initial game state
		if gameStarted && gameManager != nil {
			turnState := gameManager.GetTurnState()
			initMessage, _ := json.Marshal(protocol.PatchEnvelope{
				Sequence: 0,
				EventID:  0,
				Type:     "VariablesChanged",
				Payload: protocol.VariablesChanged{
					Entries: map[string]any{
						"turnNumber":     turnState.TurnNumber,
						"currentTurn":    turnState.CurrentTurn,
						"activePlayerID": turnState.ActivePlayerID,
						"actionsLeft":    turnState.ActionsLeft,
						"movementLeft":   turnState.MovementLeft,
						"canEndTurn":     turnState.CanEndTurn,
					},
				},
			})
			_ = conn.Write(context.Background(), websocket.MessageText, initMessage)
		}

		go func(c *websocket.Conn) {
			defer hub.Remove(c)
			defer lobbyServer.HandleDisconnection(c)
			defer c.Close(websocket.StatusNormalClosure, "")

			for {
				_, data, err := c.Read(context.Background())
				if err != nil {
					return
				}

				// Check if we're in lobby or game phase
				if lobbyServer.GetConnectionManager().IsInLobby(c) {
					// Handle lobby messages
					if err := lobbyServer.HandleMessage(c, data); err != nil {
						log.Printf("Lobby message error for %s: %v", playerID, err)
					}
				} else if gameStarted && gameManager != nil {
					// Handle game messages
					handleEnhancedWebSocketMessage(data, gameManager, state, hub, sequenceGen, quest, furnitureSystem)
				}
			}
		}(conn)
	})

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server listening on :%s in lobby mode", port)
	log.Printf("Visit http://localhost:%s/lobby to join", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
