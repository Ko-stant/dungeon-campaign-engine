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

// Enhanced main function using GameManager
func mainWithGameManager() {
	// Initialize profiling if enabled
	// InitializeProfiling() // Comment out until we need it

	// Get debug configuration from environment
	debugConfig := GetDebugConfigFromEnv()
	if debugConfig.Enabled {
		log.Printf("Debug mode enabled")
	}

	// Create basic dependencies
	hub := ws.NewHub()
	sequenceGen := NewSequenceGenerator()
	broadcaster := &BroadcasterImpl{hub: hub}
	logger := &LoggerImpl{}

	// Initialize game manager with all systems
	gameManager, err := NewGameManager(broadcaster, logger, sequenceGen, debugConfig)
	if err != nil {
		log.Fatalf("Failed to initialize game manager: %v", err)
	}
	defer gameManager.Shutdown()

	// Legacy initialization for compatibility (TODO: Remove when fully migrated)
	board, quest, err := loadGameContent()
	if err != nil {
		log.Fatalf("Failed to load game content: %v", err)
	}
	state, _, err := initializeGameState(board, quest)
	if err != nil {
		log.Fatalf("Failed to initialize game state: %v", err)
	}
	corridorRegion := state.CorridorRegion

	// Setup HTTP handlers
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

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

		// Send initial hello with turn state
		turnState := gameManager.GetTurnState()
		hello, _ := json.Marshal(protocol.PatchEnvelope{
			Sequence: 0,
			EventID:  0,
			Type:     "VariablesChanged",
			Payload: protocol.VariablesChanged{
				Entries: map[string]any{
					"hello":           "world",
					"turnNumber":      turnState.TurnNumber,
					"currentTurn":     turnState.CurrentTurn,
					"activePlayerID":  turnState.ActivePlayerID,
					"actionsLeft":     turnState.ActionsLeft,
					"movementLeft":    turnState.MovementLeft,
					"canEndTurn":      turnState.CanEndTurn,
				},
			},
		})
		_ = conn.Write(context.Background(), websocket.MessageText, hello)

		go func(c *websocket.Conn) {
			defer hub.Remove(c)
			defer c.Close(websocket.StatusNormalClosure, "")
			for {
				_, data, err := c.Read(context.Background())
				if err != nil {
					return
				}
				handleEnhancedWebSocketMessage(data, gameManager, state, hub, sequenceGen, quest)
			}
		}(conn)
	})

	// Main page handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hero := state.Entities["hero-1"]
		visibleNow := computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)

		var revealed []int
		for id := range state.RevealedRegions {
			revealed = append(revealed, id)
		}

		// Include monsters in entities
		entities := []protocol.EntityLite{
			{ID: "hero-1", Kind: "hero", Tile: state.Entities["hero-1"]},
		}

		// Add visible monsters
		monsters := gameManager.GetMonsters()
		for id, monster := range monsters {
			if monster.IsVisible {
				entities = append(entities, protocol.EntityLite{
					ID:   id,
					Kind: "monster",
					Tile: monster.Position,
				})
			}
		}

		// Include all known doors in initial snapshot
		thresholds := make([]protocol.ThresholdLite, 0, len(state.KnownDoors))
		for id := range state.KnownDoors {
			if info, exists := state.Doors[id]; exists {
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
				"ui.debug":        debugConfig.Enabled,
				"turn.number":     turnState.TurnNumber,
				"turn.current":    turnState.CurrentTurn,
				"turn.phase":      turnState.CurrentPhase,
				"turn.playerId":   turnState.ActivePlayerID,
				"turn.actions":    turnState.ActionsLeft,
				"turn.movement":   turnState.MovementLeft,
				"turn.canEnd":     turnState.CanEndTurn,
			},
			ProtocolVersion:   "v0",
			Thresholds:        thresholds,
			BlockingWalls:     blockingWalls,
			Furniture:         furniture,
			VisibleRegionIDs:  visibleNow,
			CorridorRegionID:  state.CorridorRegion,
			KnownRegionIDs:    known,
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
func handleEnhancedWebSocketMessage(data []byte, gameManager *GameManager, state *GameState, hub *ws.Hub, sequenceGen *SequenceGeneratorImpl, quest *geometry.QuestDefinition) {
	var env protocol.IntentEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return
	}

	switch env.Type {
	case "RequestMove":
		var req protocol.RequestMove
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}

		// Try new GameManager movement first, fall back to legacy
		if err := gameManager.ProcessMovement(req); err != nil {
			log.Printf("GameManager movement failed: %v, falling back to legacy", err)
			seqPtr := &sequenceGen.counter
			handleRequestMove(req, state, hub, seqPtr, quest)
		}

	case "RequestToggleDoor":
		var req protocol.RequestToggleDoor
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}

		// Try new GameManager door toggle first, fall back to legacy
		if err := gameManager.ProcessDoorToggle(req); err != nil {
			log.Printf("GameManager door toggle failed: %v, falling back to legacy", err)
			seqPtr := &sequenceGen.counter
			handleRequestToggleDoor(req, state, hub, seqPtr, quest)
		}

	case "HeroAction":
		// New hero action system
		var req ActionRequest
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			log.Printf("Failed to parse hero action: %v", err)
			return
		}

		result, err := gameManager.ProcessHeroAction(req)
		if err != nil {
			log.Printf("Hero action failed: %v", err)
			return
		}

		// Broadcast the action result
		patch := protocol.PatchEnvelope{
			Sequence: sequenceGen.Next(),
			EventID:  int64(sequenceGen.Next()),
			Type:     "HeroActionResult",
			Payload:  result,
		}

		data, _ := json.Marshal(patch)
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
		handleWebSocketMessage(data, state, hub, seqPtr, quest)
	}
}

