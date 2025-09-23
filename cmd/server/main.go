package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

func main() {
	// log.Printf("=====================================")
	// log.Printf("STARTING ENHANCED GAME MANAGER VERSION")
	// log.Printf("HeroAction support: ENABLED")
	// log.Printf("=====================================")
	mainWithGameManager()
}

// Enhanced main function using GameManager
func mainWithGameManager() {
	// Initialize profiling if enabled
	// InitializeProfiling() // Comment out until we need it

	// Get debug configuration from environment
	debugConfig := GetDebugConfigFromEnv()
	// if debugConfig.Enabled {
	// 	log.Printf("Debug mode enabled")
	// }

	// Create basic dependencies
	hub := ws.NewHub()
	sequenceGen := NewSequenceGenerator()
	broadcaster := NewBroadcaster(hub, sequenceGen)
	logger := NewLogger()

	// Legacy initialization for compatibility (TODO: Remove when fully migrated)
	board, quest, err := loadGameContent()
	if err != nil {
		log.Fatalf("Failed to load game content: %v", err)
	}

	// Initialize furniture system
	log.Printf("DEBUG: Initializing furniture system...")
	furnitureSystem := NewFurnitureSystem(log.New(os.Stdout, "", log.LstdFlags))

	// Load furniture definitions
	log.Printf("DEBUG: Loading furniture definitions from content/base...")
	if err := furnitureSystem.LoadFurnitureDefinitions("content/base"); err != nil {
		log.Printf("Warning: Failed to load furniture definitions: %v", err)
	}

	// Create furniture instances from quest
	log.Printf("DEBUG: Quest has %d furniture items", len(quest.Furniture))
	if err := furnitureSystem.CreateFurnitureInstancesFromQuest(quest); err != nil {
		log.Printf("Warning: Failed to create furniture instances: %v", err)
	}

	log.Printf("DEBUG: Furniture system initialized with %d instances", len(furnitureSystem.GetAllInstances()))

	// Initialize game manager with pre-loaded furniture system
	gameManager, err := NewGameManagerWithFurniture(broadcaster, logger, sequenceGen, debugConfig, furnitureSystem, quest)
	if err != nil {
		log.Fatalf("Failed to initialize game manager: %v", err)
	}
	defer gameManager.Shutdown()

	// Spawn monsters from quest definition
	if err := createMonstersFromQuest(quest, gameManager.GetMonsterSystem()); err != nil {
		log.Fatalf("Failed to create monsters from quest: %v", err)
	}

	state, _, err := initializeGameState(board, quest, furnitureSystem)
	if err != nil {
		log.Fatalf("Failed to initialize game state: %v", err)
	}
	corridorRegion := state.CorridorRegion

	log.Printf("DEBUG: Map dimensions - Width: %d, Height: %d", state.Segment.Width, state.Segment.Height)

	// Setup HTTP handlers
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Serve assets directory for furniture images and other game assets
	assetsServer := http.FileServer(http.Dir("."))
	mux.Handle("/assets/", http.StripPrefix("/", assetsServer))

	// Register debug endpoints if enabled
	if debugConfig.Enabled {
		gameManager.GetDebugSystem().RegisterDebugRoutes(mux)
		log.Printf("Debug API endpoints registered")
	}

	// WebSocket handler with enhanced game manager support
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		hub.Add(conn)

		// Send initial initMessage with turn state
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

		go func(c *websocket.Conn) {
			defer hub.Remove(c)
			defer c.Close(websocket.StatusNormalClosure, "")
			for {
				_, data, err := c.Read(context.Background())
				if err != nil {
					return
				}
				handleEnhancedWebSocketMessage(data, gameManager, state, hub, sequenceGen, quest, furnitureSystem)
			}
		}(conn)
	})

	// Main page handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get current game state for up-to-date entity positions
		currentGameState := gameManager.GetGameState()

		// Use current hero position, not stale initial position
		currentGameState.Lock.Lock()
		hero := currentGameState.Entities["hero-1"]

		// Use current game state for up-to-date revealed regions
		var revealed []int
		for id := range currentGameState.RevealedRegions {
			revealed = append(revealed, id)
		}
		currentGameState.Lock.Unlock()

		visibleNow := computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)

		// Include only heroes in entities (monsters are handled separately)
		// Use current entity position from game manager, not stale initial state
		entities := []protocol.EntityLite{
			{ID: "hero-1", Kind: "hero", Tile: hero},
		}

		// Include all known doors in initial snapshot
		// Use current game state for up-to-date door information
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
				log.Printf("known door %s at (%d,%d,%s) regions=%d|%d state=%s",
					id, info.Edge.X, info.Edge.Y, info.Edge.Orientation, info.RegionA, info.RegionB, info.State)
			}
		}
		currentGameState.Lock.Unlock()

		// Include visible blocking walls
		blockingWalls, _ := getVisibleBlockingWalls(state, hero, quest)

		log.Printf("corridorRegion=%d", corridorRegion)
		known := make([]int, 0, len(state.KnownRegions))
		for rid := range state.KnownRegions {
			known = append(known, rid)
		}

		// Get current turn state for UI
		turnState := gameManager.GetTurnState()

		// Get furniture data for snapshot
		furniture := gameManager.GetFurnitureForSnapshot()
		log.Printf("DEBUG: Snapshot generation - got %d furniture items", len(furniture))

		// Get monster data for snapshot
		monsters := gameManager.GetMonstersForSnapshot()
		log.Printf("DEBUG: Snapshot generation - got %d monster items", len(monsters))

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
				"ui.debug":      debugConfig.Enabled,
				"turn.number":   turnState.TurnNumber,
				"turn.current":  turnState.CurrentTurn,
				"turn.phase":    turnState.CurrentPhase,
				"turn.playerId": turnState.ActivePlayerID,
				"turn.actions":  turnState.ActionsLeft,
				"turn.movement": turnState.MovementLeft,
				"turn.canEnd":   turnState.CanEndTurn,
			},
			ProtocolVersion:  "v0",
			Thresholds:       thresholds,
			BlockingWalls:    blockingWalls,
			Furniture:        furniture,
			Monsters:         monsters,
			VisibleRegionIDs: visibleNow,
			CorridorRegionID: state.CorridorRegion,
			KnownRegionIDs:   known,
		}

		if err := views.IndexPage(s).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s with enhanced game manager", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// Enhanced WebSocket message handler supporting both legacy and new actions
func handleEnhancedWebSocketMessage(data []byte, gameManager *GameManager, state *GameState, hub *ws.Hub, sequenceGen *SequenceGeneratorImpl, quest *geometry.QuestDefinition, furnitureSystem *FurnitureSystem) {
	log.Printf("DEBUG: Received WebSocket message: %s", string(data))
	var env protocol.IntentEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		log.Printf("DEBUG: Failed to unmarshal IntentEnvelope: %v", err)
		return
	}
	log.Printf("DEBUG: Message type: %s", env.Type)

	switch env.Type {
	case "RequestMove":
		var req protocol.RequestMove
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}

		// Use legacy movement system for now (unlimited movement compatibility)
		seqPtr := &sequenceGen.counter
		monsterSystem := gameManager.GetMonsterSystem()
		// Use GameManager's state to ensure consistency with door toggles
		gameManagerState := gameManager.GetGameState()
		handleRequestMove(req, gameManagerState, hub, seqPtr, quest, furnitureSystem, monsterSystem)

	case "RequestToggleDoor":
		var req protocol.RequestToggleDoor
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}

		// Use new GameManager door toggle exclusively (no fallback to avoid nil pointer issues)
		if err := gameManager.ProcessDoorToggle(req); err != nil {
			log.Printf("GameManager door toggle failed: %v", err)
			// Don't fallback to legacy handler to avoid nil pointer issues
		}

	case "HeroAction":
		log.Printf("DEBUG: Received HeroAction message, raw payload: %s", string(env.Payload))
		// New hero action system
		var req ActionRequest
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			log.Printf("Failed to parse hero action: %v", err)
			return
		}
		log.Printf("DEBUG: Parsed ActionRequest: %+v", req)

		result, err := gameManager.ProcessHeroAction(req)
		if err != nil {
			log.Printf("Hero action failed: %v", err)
			// Send error response to UI
			errorResult := map[string]interface{}{
				"success": false,
				"action":  req.Action,
				"message": err.Error(),
			}
			patch := protocol.PatchEnvelope{
				Sequence: sequenceGen.Next(),
				EventID:  int64(sequenceGen.Next()),
				Type:     "HeroActionResult",
				Payload:  errorResult,
			}
			data, _ := json.Marshal(patch)
			log.Printf("DEBUG: Broadcasting HeroActionError: %s", string(data))
			hub.Broadcast(data)
			return
		}
		log.Printf("DEBUG: ProcessHeroAction returned result: %+v", result)

		// Broadcast the action result
		patch := protocol.PatchEnvelope{
			Sequence: sequenceGen.Next(),
			EventID:  int64(sequenceGen.Next()),
			Type:     "HeroActionResult",
			Payload:  result,
		}

		data, _ := json.Marshal(patch)
		log.Printf("DEBUG: Broadcasting HeroActionResult: %s", string(data))
		hub.Broadcast(data)

	case "MonsterAction":
		// New monster action system (GameMaster only)
		var req MonsterActionRequest
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			log.Printf("Failed to parse monster action: %v", err)
			return
		}

		result, err := gameManager.ProcessMonsterAction(req)
		if err != nil {
			log.Printf("Monster action failed: %v", err)
			return
		}

		// Broadcast the action result
		patch := protocol.PatchEnvelope{
			Sequence: sequenceGen.Next(),
			EventID:  int64(sequenceGen.Next()),
			Type:     "MonsterActionResult",
			Payload:  result,
		}

		data, _ := json.Marshal(patch)
		hub.Broadcast(data)

	case "PassGMTurn":
		// Debug function to pass GM turn and return to hero turn
		log.Printf("DEBUG: Received PassGMTurn request")

		if err := gameManager.PassGMTurn(); err != nil {
			log.Printf("PassGMTurn failed: %v", err)
			return
		}

		// Broadcast new turn state
		turnState := gameManager.GetTurnState()
		patch := protocol.PatchEnvelope{
			Sequence: sequenceGen.Next(),
			EventID:  int64(sequenceGen.Next()),
			Type:     "TurnStateChanged",
			Payload:  turnState,
		}

		data, _ := json.Marshal(patch)
		log.Printf("DEBUG: Broadcasting TurnStateChanged after PassGMTurn: %s", string(data))
		hub.Broadcast(data)

	case "EndTurn":
		// End turn request
		if err := gameManager.EndTurn(); err != nil {
			log.Printf("End turn failed: %v", err)
			return
		}

		// Broadcast new turn state
		turnState := gameManager.GetTurnState()
		patch := protocol.PatchEnvelope{
			Sequence: sequenceGen.Next(),
			EventID:  int64(sequenceGen.Next()),
			Type:     "TurnStateChanged",
			Payload:  turnState,
		}

		data, _ := json.Marshal(patch)
		hub.Broadcast(data)

	default:
		// Unknown message type - fall back to legacy handler
		seqPtr := &sequenceGen.counter
		monsterSystem := gameManager.GetMonsterSystem()
		// Use GameManager's state to ensure consistency
		gameManagerState := gameManager.GetGameState()
		handleWebSocketMessage(data, gameManagerState, hub, seqPtr, quest, furnitureSystem, monsterSystem)
	}
}
