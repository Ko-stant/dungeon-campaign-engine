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

func getVisibleBlockingWalls(state *GameState, hero protocol.TileAddress, quest *geometry.QuestDefinition) ([]protocol.BlockingWallLite, []protocol.BlockingWallLite) {
	log.Printf("=== Checking blocking wall visibility from hero at (%d,%d) ===", hero.X, hero.Y)
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

func firstCorridorTile(seg geometry.Segment, rm geometry.RegionMap) (int, int) {
	// Find the corridor region ID (region of tile at 0,0)
	corridorRegion := rm.TileRegionIDs[0]

	// Start from top-left and find the first corridor tile
	for y := 0; y < seg.Height; y++ {
		for x := 0; x < seg.Width; x++ {
			idx := y*seg.Width + x
			if rm.TileRegionIDs[idx] == corridorRegion {
				return x, y
			}
		}
	}
	return 1, 1
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

func makeDoorID(segmentID string, e geometry.EdgeAddress) string {
	return fmt.Sprintf("%s:%d:%d:%s", segmentID, e.X, e.Y, e.Orientation)
}

func filterVerticalEdges(edges []geometry.EdgeAddress) []geometry.EdgeAddress {
	var vertical []geometry.EdgeAddress
	for _, edge := range edges {
		if edge.Orientation == geometry.Vertical {
			vertical = append(vertical, edge)
		}
	}
	return vertical
}

func filterHorizontalEdges(edges []geometry.EdgeAddress) []geometry.EdgeAddress {
	var horizontal []geometry.EdgeAddress
	for _, edge := range edges {
		if edge.Orientation == geometry.Horizontal {
			horizontal = append(horizontal, edge)
		}
	}
	return horizontal
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

func main() {
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

	state, _, err := initializeGameState(board, quest, furnitureSystem)
	if err != nil {
		log.Fatalf("Failed to initialize game state: %v", err)
	}

	corridorRegion := state.CorridorRegion

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Serve assets directory for furniture images and other game assets
	assetsServer := http.FileServer(http.Dir("."))
	mux.Handle("/assets/", http.StripPrefix("/", assetsServer))

	hub := ws.NewHub()
	var sequence uint64

	// Initialize monster system for legacy main
	log.Printf("DEBUG: Initializing monster system...")
	broadcaster := &BroadcasterImpl{hub: hub}
	logger := &LoggerImpl{}
	diceSystem := &DiceSystem{}
	monsterSystem := NewMonsterSystem(state, nil, diceSystem, broadcaster, logger)

	// Create monsters from quest
	log.Printf("DEBUG: Quest has %d monsters", len(quest.Monsters))
	if err := createMonstersFromQuest(quest, monsterSystem); err != nil {
		log.Printf("Warning: Failed to create monsters: %v", err)
	} else {
		log.Printf("DEBUG: Monster system initialized")
	}

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		hub.Add(conn)

		go func(c *websocket.Conn) {
			defer hub.Remove(c)
			defer c.Close(websocket.StatusNormalClosure, "")
			for {
				_, data, err := c.Read(context.Background())
				if err != nil {
					return
				}
				handleWebSocketMessage(data, state, hub, &sequence, quest, furnitureSystem, monsterSystem)
			}
		}(conn)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hero := state.Entities["hero-1"]
		visibleNow := computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)
		var revealed []int
		for id := range state.RevealedRegions {
			revealed = append(revealed, id)
		}
		entities := []protocol.EntityLite{
			{ID: "hero-1", Kind: "hero", Tile: state.Entities["hero-1"]},
		}
		// Snapshot generation should only include already-known doors, not discover new ones

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

		// Get furniture data for snapshot
		log.Printf("DEBUG: Getting furniture for snapshot...")
		furnitureForSnapshot := getFurnitureForSnapshot(furnitureSystem, state)
		log.Printf("DEBUG: Snapshot will include %d furniture items", len(furnitureForSnapshot))

		// Get monsters for snapshot
		monstersForSnapshot := getMonstersForSnapshot(monsterSystem, state)
		log.Printf("DEBUG: Snapshot will include %d monster items", len(monstersForSnapshot))

		log.Printf("corridorRegion=%d", corridorRegion)
		known := make([]int, 0, len(state.KnownRegions))
		for rid := range state.KnownRegions {
			known = append(known, rid)
		}

		s := protocol.Snapshot{
			MapID:             "dev-map",
			PackID:            "dev-pack@v1",
			Turn:              1,
			LastEventID:       0,
			MapWidth:          state.Segment.Width,
			MapHeight:         state.Segment.Height,
			RegionsCount:      state.RegionMap.RegionsCount,
			TileRegionIDs:     state.RegionMap.TileRegionIDs,
			RevealedRegionIDs: revealed,
			DoorStates:        []byte{},
			Entities:          entities,
			Variables:         map[string]any{"ui.debug": true},
			ProtocolVersion:   "v0",
			Thresholds:        thresholds,
			BlockingWalls:     blockingWalls,
			Furniture:         furnitureForSnapshot,
			Monsters:          monstersForSnapshot,
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
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// getFurnitureForSnapshot converts furniture instances to protocol format, only including furniture in revealed regions
func getFurnitureForSnapshot(furnitureSystem *FurnitureSystem, state *GameState) []protocol.FurnitureLite {
	instances := furnitureSystem.GetAllInstances()
	furniture := make([]protocol.FurnitureLite, 0, len(instances))

	for _, instance := range instances {
		if instance.Definition == nil {
			log.Printf("Warning: Furniture instance %s has no definition", instance.ID)
			continue
		}

		// Check if furniture is in a revealed region
		furnitureIdx := instance.Position.Y*state.Segment.Width + instance.Position.X
		furnitureRegion := state.RegionMap.TileRegionIDs[furnitureIdx]
		if !state.RevealedRegions[furnitureRegion] {
			log.Printf("DEBUG: Furniture %s in region %d not revealed, skipping", instance.ID, furnitureRegion)
			continue
		}

		furnitureItem := protocol.FurnitureLite{
			ID:   instance.ID,
			Type: instance.Type,
			Tile: instance.Position,
			GridSize: struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			}{
				Width:  instance.Definition.GridSize.Width,
				Height: instance.Definition.GridSize.Height,
			},
			Rotation:           instance.Rotation,
			SwapAspectOnRotate: instance.SwapAspectOnRotate,
			TileImage:          instance.Definition.Rendering.TileImage,
			TileImageCleaned:   instance.Definition.Rendering.TileImageCleaned,
			PixelDimensions: struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			}{
				Width:  instance.Definition.Rendering.PixelDimensions.Width,
				Height: instance.Definition.Rendering.PixelDimensions.Height,
			},
			BlocksLineOfSight: instance.Definition.BlocksLineOfSight,
			BlocksMovement:    instance.Definition.BlocksMovement,
			Contains:          instance.Contains,
		}

		furniture = append(furniture, furnitureItem)
		log.Printf("DEBUG: Added furniture to snapshot: %s (%s) at (%d,%d)",
			instance.ID, instance.Type, instance.Position.X, instance.Position.Y)
	}

	return furniture
}

// getMonstersForSnapshot converts monster instances to protocol format, only including known monsters
func getMonstersForSnapshot(monsterSystem *MonsterSystem, state *GameState) []protocol.MonsterLite {
	allMonsters := monsterSystem.GetMonsters()
	monsters := make([]protocol.MonsterLite, 0, len(allMonsters))

	for _, monster := range allMonsters {
		// Only include monsters that have been discovered
		if state.KnownMonsters[monster.ID] {
			monsterItem := protocol.MonsterLite{
				ID:        monster.ID,
				Type:      string(monster.Type),
				Tile:      monster.Position,
				Body:      monster.Body,
				MaxBody:   monster.MaxBody,
				IsVisible: monster.IsVisible,
				IsAlive:   monster.IsAlive,
			}
			monsters = append(monsters, monsterItem)
			log.Printf("DEBUG: Added monster to snapshot: %s (%s) at (%d,%d) - visible: %v, alive: %v",
				monster.ID, monster.Type, monster.Position.X, monster.Position.Y, monster.IsVisible, monster.IsAlive)
		}
	}

	return monsters
}
