package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

func main() {
	// Check if we want lobby mode (default) or direct game mode
	useLobby := os.Getenv("USE_LOBBY")
	if useLobby == "" || useLobby == "true" {
		log.Printf("=== Starting in LOBBY mode ===")
		mainWithLobby()
	} else {
		log.Printf("=== Starting in DIRECT GAME mode (legacy) ===")
		mainWithGameManager()
	}
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
	log.Printf("DEBUG: Loading furniture definitions from content...")
	if err := furnitureSystem.LoadFurnitureDefinitions("content"); err != nil {
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

		// Send initial messages with turn state and blocking walls
		turnState := gameManager.GetTurnState()

		// Send turn state
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

		// Send blocking walls for refreshed connections
		currentGameState := gameManager.GetGameState()
		currentGameState.Lock.Lock()
		hero := currentGameState.Entities["hero-1"]
		currentGameState.Lock.Unlock()

		blockingWalls, _ := getVisibleBlockingWalls(currentGameState, hero, quest)
		if len(blockingWalls) > 0 {
			blockingWallsMessage, _ := json.Marshal(protocol.PatchEnvelope{
				Sequence: 1,
				EventID:  1,
				Type:     "BlockingWallsVisible",
				Payload:  protocol.BlockingWallsVisible{BlockingWalls: blockingWalls},
			})
			_ = conn.Write(context.Background(), websocket.MessageText, blockingWallsMessage)
		}

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
		// Get HP from turn manager's player character
		heroHP := &protocol.HP{Current: 0, Max: 0}
		heroMindPoints := &protocol.HP{Current: 0, Max: 0}
		if player := gameManager.turnManager.GetPlayer("player-1"); player != nil && player.Character != nil {
			heroHP.Current = player.Character.CurrentBody
			heroHP.Max = player.Character.BaseStats.BodyPoints
			heroMindPoints.Current = player.Character.CurrentMind
			heroMindPoints.Max = player.Character.BaseStats.MindPoints
		}

		entities := []protocol.EntityLite{
			{
				ID:         "hero-1",
				Kind:       "hero",
				Tile:       hero,
				HP:         heroHP,
				MindPoints: heroMindPoints,
				Tags:       []string{"barbarian"}, // TODO: Get actual class from player
			},
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

		// Get hero turn states for snapshot
		heroTurnStates := gameManager.GetHeroTurnStatesForSnapshot()
		log.Printf("DEBUG: Snapshot generation - got %d hero turn states", len(heroTurnStates))

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

	case "MovementRequest":
		// New turn-based movement system
		var req MovementRequest
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			log.Printf("Failed to parse movement request: %v", err)
			return
		}
		log.Printf("DEBUG: Parsed MovementRequest: %+v", req)

		result, err := gameManager.ProcessMovementRequest(req)
		if err != nil {
			log.Printf("Movement request failed: %v", err)
			// Send error response to UI
			errorResult := map[string]interface{}{
				"success": false,
				"action":  "movement",
				"message": err.Error(),
			}
			patch := protocol.PatchEnvelope{
				Sequence: sequenceGen.Next(),
				EventID:  int64(sequenceGen.Next()),
				Type:     "HeroActionResult",
				Payload:  errorResult,
			}
			data, _ := json.Marshal(patch)
			hub.Broadcast(data)
			return
		}
		log.Printf("DEBUG: ProcessMovement returned result: %+v", result)

		// Broadcast the movement result
		patch := protocol.PatchEnvelope{
			Sequence: sequenceGen.Next(),
			EventID:  int64(sequenceGen.Next()),
			Type:     "HeroActionResult",
			Payload:  result,
		}
		data, _ := json.Marshal(patch)
		hub.Broadcast(data)

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

	case "InstantActionRequest":
		// Instant action system
		var req InstantActionRequest
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			log.Printf("Failed to parse instant action: %v", err)
			return
		}
		log.Printf("DEBUG: Parsed InstantActionRequest: %+v", req)

		result, err := gameManager.ProcessInstantAction(req)
		if err != nil {
			log.Printf("Instant action failed: %v", err)
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
			log.Printf("DEBUG: Broadcasting InstantActionError: %s", string(data))
			hub.Broadcast(data)
			return
		}
		log.Printf("DEBUG: ProcessInstantAction returned result: %+v", result)

		// Broadcast the action result
		patch := protocol.PatchEnvelope{
			Sequence: sequenceGen.Next(),
			EventID:  int64(sequenceGen.Next()),
			Type:     "HeroActionResult",
			Payload:  result,
		}

		data, _ := json.Marshal(patch)
		log.Printf("DEBUG: Broadcasting InstantActionResult: %s", string(data))
		hub.Broadcast(data)

	case "RequestJoinLobby":
		// Lobby: Player joins
		log.Printf("RequestJoinLobby not yet implemented")

	case "RequestSelectRole":
		// Lobby: Player selects role
		log.Printf("RequestSelectRole not yet implemented")

	case "RequestToggleReady":
		// Lobby: Player toggles ready status
		log.Printf("RequestToggleReady not yet implemented")

	case "RequestStartGame":
		// Lobby: Start the game
		log.Printf("RequestStartGame not yet implemented")

	default:
		// Unknown message type - fall back to legacy handler
		seqPtr := &sequenceGen.counter
		monsterSystem := gameManager.GetMonsterSystem()
		// Use GameManager's state to ensure consistency
		gameManagerState := gameManager.GetGameState()
		handleWebSocketMessage(data, gameManagerState, hub, seqPtr, quest, furnitureSystem, monsterSystem)
	}
}

func edgeForStep(x, y, dx, dy int) geometry.EdgeAddress {
	if dx == 1 && dy == 0 {
		// Moving right: cross the right edge of current tile = left edge of tile (x+1,y)
		return geometry.EdgeAddress{X: x + 1, Y: y, Orientation: geometry.Vertical}
	}
	if dx == -1 && dy == 0 {
		// Moving left: cross the left edge of current tile = left edge of tile (x,y)
		return geometry.EdgeAddress{X: x, Y: y, Orientation: geometry.Vertical}
	}
	if dx == 0 && dy == 1 {
		// Moving down: cross the bottom edge of current tile = top edge of tile (x,y+1)
		return geometry.EdgeAddress{X: x, Y: y + 1, Orientation: geometry.Horizontal}
	}
	// Moving up: cross the top edge of current tile = top edge of tile (x,y)
	return geometry.EdgeAddress{X: x, Y: y, Orientation: geometry.Horizontal}
}

func buildBlockedWalls(seg geometry.Segment) map[geometry.EdgeAddress]bool {
	m := make(map[geometry.EdgeAddress]bool, len(seg.WallsVertical)+len(seg.WallsHorizontal))

	// Create a set of door socket edges to exclude from walls
	doorEdges := make(map[geometry.EdgeAddress]bool)
	for _, e := range seg.DoorSockets {
		doorEdges[e] = true
	}

	// Add walls, but exclude any that have door sockets
	for _, e := range seg.WallsVertical {
		if !doorEdges[e] {
			m[e] = true
		}
	}
	for _, e := range seg.WallsHorizontal {
		if !doorEdges[e] {
			m[e] = true
		}
	}
	return m
}

func buildBlockedTiles(quest *geometry.QuestDefinition) map[protocol.TileAddress]bool {
	blockedTiles := make(map[protocol.TileAddress]bool)

	log.Printf("=== Building blocked tiles ===")
	for _, wall := range quest.BlockingWalls {
		// Handle multi-tile walls
		size := wall.Size
		if size <= 0 {
			size = 1 // Default to single tile
		}

		for i := 0; i < size; i++ {
			tileX := wall.X
			tileY := wall.Y

			// Offset for multi-tile walls
			if wall.Orientation == "horizontal" {
				tileX += i
			} else {
				tileY += i
			}

			tile := protocol.TileAddress{X: tileX, Y: tileY}
			blockedTiles[tile] = true
			log.Printf("Blocked tile at (%d,%d) from wall %s", tileX, tileY, wall.ID)
		}
	}

	return blockedTiles
}

func broadcastEvent(hub *ws.Hub, sequence *uint64, eventType string, payload any) {
	seq := atomic.AddUint64(sequence, 1)
	envelope := protocol.PatchEnvelope{
		Sequence: seq,
		EventID:  0,
		Type:     eventType,
		Payload:  payload,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		log.Printf("failed to marshal %s: %v", eventType, err)
		return
	}
	log.Printf("broadcasting %s", eventType)
	hub.Broadcast(data)
}

func getVisibleBlockingWalls(state *GameState, hero protocol.TileAddress, quest *geometry.QuestDefinition) ([]protocol.BlockingWallLite, []protocol.BlockingWallLite) {
	log.Printf("=== Checking blocking wall visibility from hero at (%d,%d) ===", hero.X, hero.Y)

	// Handle nil quest gracefully
	if quest == nil {
		log.Printf("Quest is nil, no blocking walls to check")
		return []protocol.BlockingWallLite{}, []protocol.BlockingWallLite{}
	}

	log.Printf("Total blocking walls to check: %d", len(quest.BlockingWalls))

	// Track newly discovered walls
	var newlyDiscovered []protocol.BlockingWallLite

	// First, check for newly visible blocking walls and add them to known walls
	for _, wall := range quest.BlockingWalls {
		if state.KnownBlockingWalls[wall.ID] {
			continue // Already known
		}

		// Check if any tile of this blocking wall is visible from hero position
		hasLOS := false
		size := wall.Size
		if size <= 0 {
			size = 1
		}

		for i := 0; i < size; i++ {
			tileX := wall.X
			tileY := wall.Y

			// Offset for multi-tile walls
			if wall.Orientation == "horizontal" {
				tileX += i
			} else {
				tileY += i
			}

			// Check line-of-sight to the center of this blocking wall tile
			if isTileCenterVisible(state, hero.X, hero.Y, tileX, tileY) {
				log.Printf("Blocking wall %s tile (%d,%d) has line-of-sight from hero", wall.ID, tileX, tileY)
				hasLOS = true
				break
			} else {
				log.Printf("Blocking wall %s tile (%d,%d) blocked from hero", wall.ID, tileX, tileY)
			}
		}

		// Check if blocking wall is on the same room as hero (like doors)
		heroIdx := hero.Y*state.Segment.Width + hero.X
		heroRegion := state.RegionMap.TileRegionIDs[heroIdx]

		// For wall-on-room check, check the first tile of the wall
		wallIdx := wall.Y*state.Segment.Width + wall.X
		wallRegion := state.RegionMap.TileRegionIDs[wallIdx]
		wallOnCurrentRoom := wallRegion == heroRegion

		// Check corridor segment visibility (dynamic calculation)
		onSameCorridorAxis := false
		if heroRegion == state.CorridorRegion && wallRegion == state.CorridorRegion {
			onSameCorridorAxis = isInCorridorSegmentWithWall(state, hero.X, hero.Y, wall.X, wall.Y)
		}

		// Blocking wall visibility rules:
		// 1. If in a room: show walls on room tiles OR walls with line-of-sight
		// 2. If in corridor: show walls with line-of-sight OR walls on same corridor axis
		shouldShow := false
		if heroRegion != state.CorridorRegion {
			// In a room: show room walls or LOS walls
			shouldShow = hasLOS || wallOnCurrentRoom
		} else {
			// In corridor: show walls with line-of-sight OR same corridor axis
			shouldShow = hasLOS || onSameCorridorAxis
		}

		isVisible := shouldShow
		if isVisible {
			state.KnownBlockingWalls[wall.ID] = true
			wallLite := protocol.BlockingWallLite{
				ID:          wall.ID,
				X:           wall.X,
				Y:           wall.Y,
				Orientation: wall.Orientation,
				Size:        wall.Size,
			}
			newlyDiscovered = append(newlyDiscovered, wallLite)
			log.Printf("Newly discovered blocking wall %s at (%d,%d) orientation=%s size=%d - LOS: %v, OnCurrentRoom: %v, OnSameCorridorAxis: %v (hero region: %d, wall region: %d, corridor region: %d)",
				wall.ID, wall.X, wall.Y, wall.Orientation, wall.Size, hasLOS, wallOnCurrentRoom, onSameCorridorAxis, heroRegion, wallRegion, state.CorridorRegion)
		}
	}

	// Return all known blocking walls
	var allVisible []protocol.BlockingWallLite
	for _, wall := range quest.BlockingWalls {
		if state.KnownBlockingWalls[wall.ID] {
			allVisible = append(allVisible, protocol.BlockingWallLite{
				ID:          wall.ID,
				X:           wall.X,
				Y:           wall.Y,
				Orientation: wall.Orientation,
				Size:        wall.Size,
			})
		}
	}

	return allVisible, newlyDiscovered
}

func makeDoorID(segmentID string, e geometry.EdgeAddress) string {
	return fmt.Sprintf("%s:%d:%d:%s", segmentID, e.X, e.Y, e.Orientation)
}

func isInCorridorSegmentWithWall(state *GameState, heroX, heroY, wallX, wallY int) bool {
	// Check if hero and wall are aligned on same axis and in same corridor segment

	if heroX == wallX {
		// Vertical alignment - check if there's a clear corridor path between hero and wall
		minY, maxY := heroY, wallY
		if minY > maxY {
			minY, maxY = maxY, minY
		}

		// Check for uninterrupted corridor path
		for y := minY; y <= maxY; y++ {
			if y >= 0 && y < state.Segment.Height {
				idx := y*state.Segment.Width + heroX
				if state.RegionMap.TileRegionIDs[idx] != state.CorridorRegion {
					return false
				}
			}
		}
		return true

	} else if heroY == wallY {
		// Horizontal alignment - check if there's a clear corridor path between hero and wall
		minX, maxX := heroX, wallX
		if minX > maxX {
			minX, maxX = maxX, minX
		}

		// Check for uninterrupted corridor path
		for x := minX; x <= maxX; x++ {
			if x >= 0 && x < state.Segment.Width {
				idx := heroY*state.Segment.Width + x
				if state.RegionMap.TileRegionIDs[idx] != state.CorridorRegion {
					return false
				}
			}
		}
		return true

	} else {
		// Not aligned - check if hero is in a corridor that can see the wall

		// Check all four directions from the wall to find corridor connections
		directions := []struct{ dx, dy int }{
			{0, 1},  // down
			{0, -1}, // up
			{1, 0},  // right
			{-1, 0}, // left
		}

		for _, dir := range directions {
			// Find the corridor tile adjacent to the wall in this direction
			adjX, adjY := wallX+dir.dx, wallY+dir.dy

			if adjX >= 0 && adjX < state.Segment.Width && adjY >= 0 && adjY < state.Segment.Height {
				adjIdx := adjY*state.Segment.Width + adjX
				if adjIdx < len(state.RegionMap.TileRegionIDs) &&
					state.RegionMap.TileRegionIDs[adjIdx] == state.CorridorRegion {

					// Check if hero can reach this corridor tile from their position
					if dir.dy == 0 {
						// Horizontal direction from wall (left/right) creates vertical corridor - check if hero is in same column
						if heroX == adjX {
							// Check if there's a clear corridor path between hero and the wall's adjacent tile
							minY, maxY := heroY, adjY
							if minY > maxY {
								minY, maxY = maxY, minY
							}
							pathClear := true
							for y := minY; y <= maxY; y++ {
								if y >= 0 && y < state.Segment.Height {
									idx := y*state.Segment.Width + heroX
									region := state.RegionMap.TileRegionIDs[idx]
									if region != state.CorridorRegion {
										pathClear = false
										break
									}
								}
							}
							if pathClear {
								return true
							}
						}
					} else {
						// Vertical direction from wall (up/down) creates horizontal corridor - check if hero is in same row
						if heroY == adjY {
							// Check if there's a clear corridor path between hero and the wall's adjacent tile
							minX, maxX := heroX, adjX
							if minX > maxX {
								minX, maxX = maxX, minX
							}
							pathClear := true
							for x := minX; x <= maxX; x++ {
								if x >= 0 && x < state.Segment.Width {
									idx := heroY*state.Segment.Width + x
									region := state.RegionMap.TileRegionIDs[idx]
									if region != state.CorridorRegion {
										pathClear = false
										break
									}
								}
							}
							if pathClear {
								return true
							}
						}
					}
				}
			}
		}

		return false
	}
}
