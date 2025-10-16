package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

// connectionPlayerMap tracks which player ID owns each WebSocket connection
var connectionPlayerMap = struct {
	sync.RWMutex
	conns map[*websocket.Conn]string
}{conns: make(map[*websocket.Conn]string)}

// Helper functions for player ID tracking
func getPlayerIDFromRequest(r *http.Request) string {
	// Check session cookie
	cookie, err := r.Cookie("playerID")
	if err == nil {
		return cookie.Value
	}

	// Check query param as fallback
	return r.URL.Query().Get("playerID")
}

// isPlayerGameMaster checks if a player is the game master
func isPlayerGameMaster(playerID string, lobbyServer *LobbyServer) bool {
	if lobbyServer == nil {
		return false
	}

	player, exists := lobbyServer.lobby.GetPlayer(playerID)
	if !exists {
		return false
	}

	return player.Role == RoleGameMaster
}

func setPlayerIDCookie(w http.ResponseWriter, playerID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "playerID",
		Value:    playerID,
		Path:     "/",
		MaxAge:   3600 * 24, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// getStartingPositionsFromQuest extracts valid starting positions from the quest's starting room
func getStartingPositionsFromQuest(quest *geometry.QuestDefinition, board *geometry.BoardDefinition) []protocol.TileAddress {
	if quest == nil || board == nil {
		return []protocol.TileAddress{}
	}

	// Find the starting room from board definition
	startingRoomID := quest.StartingRoom
	for _, room := range board.Rooms {
		if room.ID == startingRoomID {
			// Convert all room tiles to starting positions
			positions := make([]protocol.TileAddress, 0, len(room.Tiles))
			for _, tile := range room.Tiles {
				positions = append(positions, protocol.TileAddress{
					SegmentID: "",
					X:         tile.X,
					Y:         tile.Y,
				})
			}
			return positions
		}
	}

	// Fallback if starting room not found
	return []protocol.TileAddress{}
}

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
	var board *geometry.BoardDefinition
	var furnitureSystem *FurnitureSystem

	// Flag to track if game has started
	gameStarted := false
	var gameMasterPlayerID string // Track GM player ID for reliable role checking

	// Set game start handler
	lobbyServer.SetGameStartHandler(func(gameMasterID string, heroPlayers map[string]string) error {
		log.Printf("Initializing game with GM=%s and heroes=%v", gameMasterID, heroPlayers)
		gameMasterPlayerID = gameMasterID // Store GM ID for persistent role checking

		// Load game content
		loadedBoard, loadedQuest, err := loadGameContent()
		if err != nil {
			return fmt.Errorf("failed to load game content: %w", err)
		}
		board = loadedBoard
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

		// Get dynamic turn order manager from game manager (already initialized)
		dynamicTurnOrder := gameManager.GetDynamicTurnOrder()
		log.Printf("Using DynamicTurnOrderManager from GameManager (starting in QuestSetup phase)")

		// Note: GM is not registered in turn order because they don't participate in quest setup
		// or hero election phases. They only participate during GM phase which is managed separately.
		log.Printf("GM player %s will control monsters during GM phase", gameMasterID)

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

			// Register player in dynamic turn order
			dynamicTurnOrder.RegisterPlayer(playerID)
			log.Printf("Registered hero player in turn order: %s", playerID)

			// Note: Hero entities will be spawned at positions chosen during quest setup phase
			// Do NOT add to game state yet - position selection happens first

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

		// Get player ID from cookie
		viewerPlayerID := getPlayerIDFromRequest(r)
		log.Printf("REFRESH DEBUG: Hero page requested by player %s", viewerPlayerID)
		if viewerPlayerID == "" {
			// No player ID, show error or redirect to lobby
			http.Error(w, "No player ID found", http.StatusUnauthorized)
			return
		}

		// Check if this player is the game master - if so, redirect to GM page
		if viewerPlayerID == gameMasterPlayerID {
			log.Printf("REFRESH DEBUG: Redirecting GM %s to /gm page", viewerPlayerID)
			http.Redirect(w, r, "/gm", http.StatusSeeOther)
			return
		}

		// Serve hero game page
		currentGameState := gameManager.GetGameState()
		currentGameState.Lock.Lock()

		// Get this player's hero entity ID and position
		viewerPlayer := gameManager.turnManager.GetPlayer(viewerPlayerID)
		viewerEntityID := ""
		var hero protocol.TileAddress
		if viewerPlayer != nil {
			viewerEntityID = viewerPlayer.EntityID
			hero = currentGameState.Entities[viewerEntityID]
			log.Printf("REFRESH DEBUG: Found viewerPlayer for %s, entityID=%s, position=(%d,%d)", viewerPlayerID, viewerEntityID, hero.X, hero.Y)
		} else {
			log.Printf("REFRESH DEBUG: No viewerPlayer found for %s in turnManager", viewerPlayerID)
			// Fallback: get first hero for initial view
			for entityID := range currentGameState.Entities {
				if len(entityID) >= 4 && entityID[:4] == "hero" {
					hero = currentGameState.Entities[entityID]
					break
				}
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
		// Get player list from turnManager (source of truth for game players)
		for _, player := range gameManager.turnManager.GetHeroPlayers() {
			if player == nil {
				continue
			}
			playerID := player.ID

			heroHP := &protocol.HP{Current: 0, Max: 0}
			heroMindPoints := &protocol.HP{Current: 0, Max: 0}
			if player.Character != nil {
				heroHP.Current = player.Character.CurrentBody
				heroHP.Max = player.Character.BaseStats.BodyPoints
				heroMindPoints.Current = player.Character.CurrentMind
				heroMindPoints.Max = player.Character.BaseStats.MindPoints
			}

			currentGameState.Lock.Lock()
			entityPos, entityExists := currentGameState.Entities[player.EntityID]
			currentGameState.Lock.Unlock()

			if !entityExists {
				log.Printf("WARNING: Entity %s not found in game state for player %s", player.EntityID, playerID)
				// During quest setup, heroes may not have positions yet - use zero position
				entityPos = protocol.TileAddress{SegmentID: "", X: 0, Y: 0}
			}

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

		// Get dynamic turn order state
		var turnPhase string
		var cycleNumber int
		var activeHeroPlayerID string
		var electedPlayerID string
		var heroesActedIDs []string

		dynamicTurnOrder := gameManager.GetDynamicTurnOrder()
		playersReady := make(map[string]bool)
		playerStartPositions := make(map[string]protocol.StartPositionInfo)

		if dynamicTurnOrder != nil {
			turnPhase = string(dynamicTurnOrder.GetCurrentPhase())
			cycleNumber = dynamicTurnOrder.GetCycleNumber()
			activeHeroPlayerID = dynamicTurnOrder.GetActiveHeroPlayerID()
			electedPlayerID = dynamicTurnOrder.GetElectedPlayer()

			// Convert heroes acted map to slice
			heroesActedMap := dynamicTurnOrder.GetHeroesActedThisCycle()
			heroesActedIDs = make([]string, 0, len(heroesActedMap))
			for playerID, acted := range heroesActedMap {
				if acted {
					heroesActedIDs = append(heroesActedIDs, playerID)
				}
			}

			// Get quest setup state
			playersReady = dynamicTurnOrder.GetPlayersReady()
			startPositions := dynamicTurnOrder.GetPlayerStartPositions()
			for playerID, pos := range startPositions {
				playerStartPositions[playerID] = protocol.StartPositionInfo{
					X: pos.X,
					Y: pos.Y,
				}
			}
		}

		// Extract starting positions for quest setup phase
		startingPositions := getStartingPositionsFromQuest(quest, board)

		// Build player names map from lobby data
		playerNames := make(map[string]string)
		for playerID := range lobbyServer.lobby.players {
			if player, exists := lobbyServer.lobby.GetPlayer(playerID); exists {
				playerNames[playerID] = player.Name
			}
		}

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
			ProtocolVersion:      "v0",
			Thresholds:           thresholds,
			BlockingWalls:        blockingWalls,
			Furniture:            furniture,
			Monsters:             monsters,
			HeroTurnStates:       heroTurnStates,
			PlayerNames:          playerNames,
			VisibleRegionIDs:     visibleNow,
			CorridorRegionID:     state.CorridorRegion,
			KnownRegionIDs:       known,
			ViewerPlayerID:       viewerPlayerID,
			ViewerRole:           "hero",
			ViewerEntityID:       viewerEntityID,
			StartingPositions:    startingPositions,
			PlayersReady:         playersReady,
			PlayerStartPositions: playerStartPositions,
			TurnPhase:            turnPhase,
			CycleNumber:          cycleNumber,
			ActiveHeroPlayerID:   activeHeroPlayerID,
			ElectedPlayerID:      electedPlayerID,
			HeroesActedIDs:       heroesActedIDs,
		}
		log.Printf("REFRESH DEBUG: Sending snapshot to %s with ViewerEntityID=%s, %d entities, turnPhase=%s", viewerPlayerID, viewerEntityID, len(entities), turnPhase)

		if err := views.IndexPage(s).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// GM page handler (only for game master, shows full visibility)
	mux.HandleFunc("/gm", func(w http.ResponseWriter, r *http.Request) {
		if !gameStarted {
			// Redirect to lobby if game hasn't started
			http.Redirect(w, r, "/lobby", http.StatusSeeOther)
			return
		}

		// Get player ID from cookie
		playerID := getPlayerIDFromRequest(r)
		if playerID == "" {
			// No player ID, redirect to root (which will redirect to lobby if needed)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Check if this player is the game master
		if playerID != gameMasterPlayerID {
			// Not the GM, redirect to hero view
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Build GM-specific snapshot with full visibility
		currentGameState := gameManager.GetGameState()
		currentGameState.Lock.Lock()

		// Build entities list (all heroes)
		entities := []protocol.EntityLite{}
		// Get player list from turnManager (source of truth for game players)
		for _, player := range gameManager.turnManager.GetHeroPlayers() {
			if player == nil {
				continue
			}
			pID := player.ID

			heroHP := &protocol.HP{Current: 0, Max: 0}
			heroMindPoints := &protocol.HP{Current: 0, Max: 0}
			if player.Character != nil {
				heroHP.Current = player.Character.CurrentBody
				heroHP.Max = player.Character.BaseStats.BodyPoints
				heroMindPoints.Current = player.Character.CurrentMind
				heroMindPoints.Max = player.Character.BaseStats.MindPoints
			}

			entityPos, entityExists := currentGameState.Entities[player.EntityID]

			if !entityExists {
				log.Printf("WARNING (GM): Entity %s not found in game state for player %s", player.EntityID, pID)
				// During quest setup, heroes may not have positions yet - use zero position
				entityPos = protocol.TileAddress{SegmentID: "", X: 0, Y: 0}
			}

			entities = append(entities, protocol.EntityLite{
				ID:         player.EntityID,
				Kind:       "hero",
				Tile:       entityPos,
				HP:         heroHP,
				MindPoints: heroMindPoints,
				Tags:       []string{string(player.Class)},
			})
		}

		// Include ALL doors (GM sees everything)
		thresholds := make([]protocol.ThresholdLite, 0, len(currentGameState.Doors))
		for id, info := range currentGameState.Doors {
			thresholds = append(thresholds, protocol.ThresholdLite{
				ID:          id,
				X:           info.Edge.X,
				Y:           info.Edge.Y,
				Orientation: string(info.Edge.Orientation),
				Kind:        "DoorSocket",
				State:       info.State,
			})
		}
		currentGameState.Lock.Unlock()

		// GM sees all blocking walls
		blockingWalls := []protocol.BlockingWallLite{}
		if quest != nil {
			for _, wall := range quest.BlockingWalls {
				blockingWalls = append(blockingWalls, protocol.BlockingWallLite{
					ID:          wall.ID,
					X:           wall.X,
					Y:           wall.Y,
					Orientation: wall.Orientation,
					Size:        wall.Size,
				})
			}
		}

		// GM sees all regions as revealed and visible
		allRegions := make([]int, 0, state.RegionMap.RegionsCount)
		for i := 0; i < state.RegionMap.RegionsCount; i++ {
			allRegions = append(allRegions, i)
		}

		turnState := gameManager.GetTurnState()
		furniture := gameManager.GetFurnitureForSnapshot()
		monsters := gameManager.GetMonstersForSnapshot()
		heroTurnStates := gameManager.GetHeroTurnStatesForSnapshot()

		// Extract quest data
		questName := ""
		questDescription := ""
		questNotes := ""
		questGMNotes := ""
		questObjectives := []string{}
		startingPositions := []protocol.TileAddress{}

		if quest != nil {
			questName = quest.Name
			questDescription = quest.Description
			// Extract objectives from quest.Objectives field
			for _, obj := range quest.Objectives {
				questObjectives = append(questObjectives, obj.Description)
			}
			// Extract starting positions from quest starting room
			startingPositions = getStartingPositionsFromQuest(quest, board)
		}

		// Get dynamic turn order state
		var turnPhase string
		var cycleNumber int
		var activeHeroPlayerID string
		var electedPlayerID string
		var heroesActedIDs []string

		dynamicTurnOrder := gameManager.GetDynamicTurnOrder()
		playersReady := make(map[string]bool)
		playerStartPositions := make(map[string]protocol.StartPositionInfo)

		if dynamicTurnOrder != nil {
			turnPhase = string(dynamicTurnOrder.GetCurrentPhase())
			cycleNumber = dynamicTurnOrder.GetCycleNumber()
			activeHeroPlayerID = dynamicTurnOrder.GetActiveHeroPlayerID()
			electedPlayerID = dynamicTurnOrder.GetElectedPlayer()

			// Convert heroes acted map to slice
			heroesActedMap := dynamicTurnOrder.GetHeroesActedThisCycle()
			heroesActedIDs = make([]string, 0, len(heroesActedMap))
			for playerID, acted := range heroesActedMap {
				if acted {
					heroesActedIDs = append(heroesActedIDs, playerID)
				}
			}

			// Get quest setup state
			playersReady = dynamicTurnOrder.GetPlayersReady()
			startPositions := dynamicTurnOrder.GetPlayerStartPositions()
			for playerID, pos := range startPositions {
				playerStartPositions[playerID] = protocol.StartPositionInfo{
					X: pos.X,
					Y: pos.Y,
				}
			}
		}

		// Build player names map from lobby data
		playerNames := make(map[string]string)
		for pID := range lobbyServer.lobby.players {
			if player, exists := lobbyServer.lobby.GetPlayer(pID); exists {
				playerNames[pID] = player.Name
			}
		}

		s := protocol.Snapshot{
			MapID:             "dev-map",
			PackID:            "dev-pack@v1",
			Turn:              turnState.TurnNumber,
			LastEventID:       0,
			MapWidth:          state.Segment.Width,
			MapHeight:         state.Segment.Height,
			RegionsCount:      state.RegionMap.RegionsCount,
			TileRegionIDs:     state.RegionMap.TileRegionIDs,
			RevealedRegionIDs: allRegions, // GM sees everything
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
			ProtocolVersion:      "v0",
			Thresholds:           thresholds,
			BlockingWalls:        blockingWalls,
			Furniture:            furniture,
			Monsters:             monsters,
			HeroTurnStates:       heroTurnStates,
			PlayerNames:          playerNames,
			VisibleRegionIDs:     allRegions, // GM sees everything
			CorridorRegionID:     state.CorridorRegion,
			KnownRegionIDs:       allRegions, // GM sees everything
			ViewerPlayerID:       playerID,
			ViewerRole:           "gm",
			ViewerEntityID:       "",
			QuestName:            questName,
			QuestDescription:     questDescription,
			QuestNotes:           questNotes,
			QuestGMNotes:         questGMNotes,
			QuestObjectives:      questObjectives,
			StartingPositions:    startingPositions,
			PlayersReady:         playersReady,
			PlayerStartPositions: playerStartPositions,
			TurnPhase:            turnPhase,
			CycleNumber:          cycleNumber,
			ActiveHeroPlayerID:   activeHeroPlayerID,
			ElectedPlayerID:      electedPlayerID,
			HeroesActedIDs:       heroesActedIDs,
		}

		if err := views.GMPage(s).Render(r.Context(), w); err != nil {
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

		// Get player ID from cookie/query param, or generate new one
		playerID := getPlayerIDFromRequest(r)
		if playerID == "" {
			playerID = generatePlayerID()
		}

		// Register connection with lobby server using existing player ID
		lobbyServer.HandleNewConnectionWithID(conn, playerID)
		log.Printf("WebSocket connected: %s", playerID)

		// Store player ID in connection map
		connectionPlayerMap.Lock()
		connectionPlayerMap.conns[conn] = playerID
		connectionPlayerMap.Unlock()

		// Set player ID cookie for HTTP routing
		setPlayerIDCookie(w, playerID)

		// Send player ID to client
		playerIDMessage, _ := json.Marshal(protocol.PatchEnvelope{
			Sequence: 0,
			EventID:  0,
			Type:     "PlayerIDAssigned",
			Payload:  protocol.PlayerIDAssigned{PlayerID: playerID},
		})
		_ = conn.Write(context.Background(), websocket.MessageText, playerIDMessage)

		// If game has started, mark connection as not in lobby and send initial game state
		if gameStarted && gameManager != nil {
			// Mark this connection as not in lobby so messages route to game handler
			lobbyServer.GetConnectionManager().SetInLobby(conn, false)
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
			defer func() {
				// Clean up connection map
				connectionPlayerMap.Lock()
				delete(connectionPlayerMap.conns, c)
				connectionPlayerMap.Unlock()
			}()

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
					// Handle game messages - get player ID from connection map
					connectionPlayerMap.RLock()
					playerID := connectionPlayerMap.conns[c]
					connectionPlayerMap.RUnlock()
					handleEnhancedWebSocketMessage(data, gameManager, state, hub, sequenceGen, quest, furnitureSystem, playerID)
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
