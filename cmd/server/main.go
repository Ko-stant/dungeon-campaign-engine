package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

type DoorInfo struct {
	Edge    geometry.EdgeAddress
	RegionA int
	RegionB int
	State   string
}

type GameState struct {
	Segment            geometry.Segment
	RegionMap          geometry.RegionMap
	BlockedWalls       map[geometry.EdgeAddress]bool
	BlockedTiles       map[protocol.TileAddress]bool
	Doors              map[string]*DoorInfo
	DoorByEdge         map[geometry.EdgeAddress]string
	Entities           map[string]protocol.TileAddress
	RevealedRegions    map[int]bool
	Lock               sync.Mutex
	KnownRegions       map[int]bool
	KnownDoors         map[string]bool
	KnownBlockingWalls map[string]bool
	CorridorRegion     int
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
		directions := []struct{dx, dy int}{
			{0, 1},   // down
			{0, -1},  // up
			{1, 0},   // right
			{-1, 0},  // left
		}

		for _, dir := range directions {
			// Find the corridor tile adjacent to the wall in this direction
			adjX, adjY := wallX + dir.dx, wallY + dir.dy

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

func isTileCenterVisible(state *GameState, fromX, fromY, toX, toY int) bool {
	// Check line-of-sight from center of fromTile to center of toTile
	sx, sy := float64(fromX)+0.5, float64(fromY)+0.5
	tx, ty := float64(toX)+0.5, float64(toY)+0.5
	dx, dy := tx-sx, ty-sy
	if dx == 0 && dy == 0 {
		return true
	}
	adx, ady := math.Abs(dx), math.Abs(dy)
	stepX, stepY := 0, 0
	if dx > 0 {
		stepX = 1
	} else if dx < 0 {
		stepX = -1
	}
	if dy > 0 {
		stepY = 1
	} else if dy < 0 {
		stepY = -1
	}

	tDeltaX, tDeltaY := math.Inf(1), math.Inf(1)
	if adx > 0 {
		tDeltaX = 1.0 / adx
	}
	if ady > 0 {
		tDeltaY = 1.0 / ady
	}

	xCell := int(math.Floor(sx))
	yCell := int(math.Floor(sy))

	var tMaxX, tMaxY float64
	if stepX > 0 {
		tMaxX = (float64(xCell+1) - sx) / adx
	} else if stepX < 0 {
		tMaxX = (sx - float64(xCell)) / adx
	} else {
		tMaxX = math.Inf(1)
	}
	if stepY > 0 {
		tMaxY = (float64(yCell+1) - sy) / ady
	} else if stepY < 0 {
		tMaxY = (sy - float64(yCell)) / ady
	} else {
		tMaxY = math.Inf(1)
	}

	for range 2048 {
		// Check if we've reached the target tile
		if xCell == toX && yCell == toY {
			return true
		}

		if tMaxX < tMaxY {
			var crossed geometry.EdgeAddress
			if stepX > 0 {
				crossed = geometry.EdgeAddress{X: xCell + 1, Y: yCell, Orientation: geometry.Vertical}
				xCell++
			} else {
				crossed = geometry.EdgeAddress{X: xCell, Y: yCell, Orientation: geometry.Vertical}
				xCell--
			}

			if state.BlockedWalls[crossed] {
				return false
			}
			if id, ok := state.DoorByEdge[crossed]; ok {
				if d := state.Doors[id]; d != nil && d.State != "open" {
					return false
				}
			}
			tMaxX += tDeltaX
		} else if tMaxY < tMaxX {
			var crossed geometry.EdgeAddress
			if stepY > 0 {
				crossed = geometry.EdgeAddress{X: xCell, Y: yCell + 1, Orientation: geometry.Horizontal}
				yCell++
			} else {
				crossed = geometry.EdgeAddress{X: xCell, Y: yCell, Orientation: geometry.Horizontal}
				yCell--
			}

			if state.BlockedWalls[crossed] {
				return false
			}
			if id, ok := state.DoorByEdge[crossed]; ok {
				if d := state.Doors[id]; d != nil && d.State != "open" {
					return false
				}
			}
			tMaxY += tDeltaY
		} else {
			oldX, oldY := xCell, yCell

			var vEdge, hEdge geometry.EdgeAddress
			var nextX, nextY = xCell, yCell

			if stepX > 0 {
				vEdge = geometry.EdgeAddress{X: oldX + 1, Y: oldY, Orientation: geometry.Vertical}
				nextX = oldX + 1
			} else if stepX < 0 {
				vEdge = geometry.EdgeAddress{X: oldX, Y: oldY, Orientation: geometry.Vertical}
				nextX = oldX - 1
			}

			if stepY > 0 {
				hEdge = geometry.EdgeAddress{X: oldX, Y: oldY + 1, Orientation: geometry.Horizontal}
				nextY = oldY + 1
			} else if stepY < 0 {
				hEdge = geometry.EdgeAddress{X: oldX, Y: oldY, Orientation: geometry.Horizontal}
				nextY = oldY - 1
			}

			// Corner rule: if EITHER of the two edges at the corner is blocking, LOS stops
			if state.BlockedWalls[vEdge] || state.BlockedWalls[hEdge] {
				return false
			}
			if id, ok := state.DoorByEdge[vEdge]; ok {
				if d := state.Doors[id]; d != nil && d.State != "open" {
					return false
				}
			}
			if id, ok := state.DoorByEdge[hEdge]; ok {
				if d := state.Doors[id]; d != nil && d.State != "open" {
					return false
				}
			}

			// advance from the corner
			xCell, yCell = nextX, nextY
			tMaxX += tDeltaX
			tMaxY += tDeltaY
		}
	}
	return false
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

func main() {
	// Load the static HeroQuest board
	board, err := geometry.LoadBoardFromFile("content/base/board.json")
	if err != nil {
		log.Fatalf("Failed to load board: %v", err)
	}

	// Load the quest
	quest, err := geometry.LoadQuestFromFile("content/base/quests/quest-01.json")
	if err != nil {
		log.Fatalf("Failed to load quest: %v", err)
	}

	segment := geometry.CreateSegmentFromBoard(board)
	regionMap := geometry.CreateRegionMapFromBoard(board)

	// Debug: Log walls around room 20 and regions
	log.Printf("=== Walls around room 20 area ===")
	for _, wall := range segment.WallsVertical {
		if wall.X >= 0 && wall.X <= 5 && wall.Y >= 13 && wall.Y <= 18 {
			log.Printf("Vertical wall at (%d,%d)", wall.X, wall.Y)
		}
	}
	for _, wall := range segment.WallsHorizontal {
		if wall.X >= 0 && wall.X <= 6 && wall.Y >= 12 && wall.Y <= 18 {
			log.Printf("Horizontal wall at (%d,%d)", wall.X, wall.Y)
		}
	}

	// Debug: Check regions around room 20
	log.Printf("=== Regions around room 20 ===")
	for y := 12; y <= 18; y++ {
		for x := 0; x <= 6; x++ {
			if x < segment.Width && y < segment.Height {
				idx := y*segment.Width + x
				region := regionMap.TileRegionIDs[idx]
				log.Printf("Tile (%d,%d) = region %d", x, y, region)
			}
		}
	}

	// Add quest doors to the segment
	questDoors := geometry.ConvertQuestDoorsToEdges(quest.Doors)
	segment.DoorSockets = questDoors

	// Blocking walls are now handled as blocked tiles, not edge walls

	// Create doors from quest configuration
	doors := make(map[string]*DoorInfo, len(quest.Doors))
	doorByEdge := make(map[geometry.EdgeAddress]string, len(quest.Doors))
	for _, questDoor := range quest.Doors {
		edge := geometry.EdgeAddress{
			X:           questDoor.X,
			Y:           questDoor.Y,
			Orientation: geometry.Vertical,
		}
		if questDoor.Orientation == "horizontal" {
			edge.Orientation = geometry.Horizontal
		}

		id := makeDoorID(segment.ID, edge)
		a, b := geometry.RegionsAcrossDoor(regionMap, segment, edge)
		doors[id] = &DoorInfo{Edge: edge, RegionA: a, RegionB: b, State: questDoor.State}
		doorByEdge[edge] = id
		log.Printf("loaded quest door %s at (%d,%d,%s) regions=%d|%d state=%s",
			id, edge.X, edge.Y, edge.Orientation, a, b, questDoor.State)
	}

	// Start hero in the quest's starting room
	startX, startY, err := geometry.FindStartingTileInRoom(board, quest.StartingRoom)
	if err != nil {
		log.Fatalf("Failed to find starting position: %v", err)
	}
	hero := protocol.TileAddress{SegmentID: segment.ID, X: startX, Y: startY}
	log.Printf("hero starting in room %d at position (%d,%d)", quest.StartingRoom, startX, startY)
	corridorRegion := regionMap.TileRegionIDs[startY*segment.Width+startX]

	// Initialize known regions - all rooms are "known" (borders visible) but only starting room is "revealed" (accessible)
	knownRegions := make(map[int]bool)
	for i := 0; i < regionMap.RegionsCount; i++ {
		knownRegions[i] = true
	}

	// Get the hero's starting region
	heroRegion := regionMap.TileRegionIDs[startY*segment.Width+startX]

	state := &GameState{
		Segment:            segment,
		RegionMap:          regionMap,
		BlockedWalls:       buildBlockedWalls(segment),
		BlockedTiles:       buildBlockedTiles(quest),
		Doors:              doors,
		DoorByEdge:         doorByEdge,
		Entities:           map[string]protocol.TileAddress{"hero-1": hero},
		RevealedRegions:    map[int]bool{heroRegion: true},
		KnownRegions:       knownRegions,
		KnownDoors:         map[string]bool{},
		KnownBlockingWalls: map[string]bool{},
		CorridorRegion:     0, // Corridor is always region 0
	}

	// Discover initially visible doors and blocking walls
	for id, info := range state.Doors {
		// Check if door is on the edge of hero's starting room
		doorOnCurrentRoom := (info.RegionA == heroRegion) || (info.RegionB == heroRegion)

		// Check line-of-sight visibility
		visible := isEdgeVisible(state, hero.X, hero.Y, info.Edge)

		// Door is visible if it has line-of-sight OR is on the edge of current room
		if visible || doorOnCurrentRoom {
			state.KnownDoors[id] = true
			log.Printf("Initially visible door %s at (%d,%d) - LOS: %v, OnCurrentRoom: %v",
				id, info.Edge.X, info.Edge.Y, visible, doorOnCurrentRoom)
		}
	}

	// Discover initially visible blocking walls
	_, _ = getVisibleBlockingWalls(state, hero, quest)

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	hub := ws.NewHub()
	var sequence uint64

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		hub.Add(conn)

		hello, _ := json.Marshal(protocol.PatchEnvelope{
			Sequence: 0,
			EventID:  0,
			Type:     "VariablesChanged",
			Payload:  protocol.VariablesChanged{Entries: map[string]any{"hello": "world"}},
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
				var env protocol.IntentEnvelope
				if err := json.Unmarshal(data, &env); err != nil {
					continue
				}
				switch env.Type {
				case "RequestMove":
					var req protocol.RequestMove
					if err := json.Unmarshal(env.Payload, &req); err != nil {
						continue
					}
					if (req.DX != 0 && req.DY != 0) || req.DX < -1 || req.DX > 1 || req.DY < -1 || req.DY > 1 {
						continue
					}
					if req.DX == 0 && req.DY == 0 {
						continue
					}
					state.Lock.Lock()
					tile, ok := state.Entities[req.EntityID]
					if !ok {
						state.Lock.Unlock()
						continue
					}
					nx := tile.X + req.DX
					ny := tile.Y + req.DY
					if nx < 0 || ny < 0 || nx >= state.Segment.Width || ny >= state.Segment.Height {
						log.Printf("DEBUG: Movement blocked by bounds check: from (%d,%d) to (%d,%d), bounds: %dx%d",
							tile.X, tile.Y, nx, ny, state.Segment.Width, state.Segment.Height)
						state.Lock.Unlock()
						continue
					}

					// Check if destination tile is blocked by a blocking wall
					destTile := protocol.TileAddress{X: nx, Y: ny}
					if state.BlockedTiles[destTile] {
						log.Printf("DEBUG: Movement blocked by blocking wall tile: from (%d,%d) to (%d,%d)",
							tile.X, tile.Y, nx, ny)
						state.Lock.Unlock()
						continue
					}

					edge := edgeForStep(tile.X, tile.Y, req.DX, req.DY)
					if state.BlockedWalls[edge] {
						log.Printf("DEBUG: Movement blocked by wall: from (%d,%d) to (%d,%d), blocked edge: %+v",
							tile.X, tile.Y, nx, ny, edge)
						state.Lock.Unlock()
						continue
					}
					if id, ok := state.DoorByEdge[edge]; ok {
						if d := state.Doors[id]; d != nil && d.State != "open" {
							state.Lock.Unlock()
							continue
						}
					}
					tile.X = nx
					tile.Y = ny
					state.Entities[req.EntityID] = tile
					state.Lock.Unlock()

					seq := atomic.AddUint64(&sequence, 1)
					out := protocol.PatchEnvelope{
						Sequence: seq,
						EventID:  0,
						Type:     "EntityUpdated",
						Payload:  protocol.EntityUpdated{ID: req.EntityID, Tile: tile},
					}
					b, _ := json.Marshal(out)
					hub.Broadcast(b)

					hero := state.Entities[req.EntityID]
					visible := computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)
					state.Lock.Lock()
					newlyKnown := addKnownRegions(state, visible)
					state.Lock.Unlock()
					seq2 := atomic.AddUint64(&sequence, 1)
					b2, _ := json.Marshal(protocol.PatchEnvelope{
						Sequence: seq2,
						EventID:  0,
						Type:     "VisibleNow",
						Payload:  protocol.VisibleNow{IDs: visible},
					})
					log.Printf("visibleNow (hero @ %d,%d): %v", hero.X, hero.Y, visible)
					hub.Broadcast(b2)

					if len(newlyKnown) > 0 {
						seq3 := atomic.AddUint64(&sequence, 1)
						b3, _ := json.Marshal(protocol.PatchEnvelope{
							Sequence: seq3,
							EventID:  0,
							Type:     "RegionsKnown",
							Payload:  protocol.RegionsKnown{IDs: newlyKnown},
						})
						hub.Broadcast(b3)
					}

					// Check for newly visible doors
					hero = state.Entities[req.EntityID]
					heroIdx := hero.Y*state.Segment.Width + hero.X
					heroRegion := state.RegionMap.TileRegionIDs[heroIdx]

					log.Printf("Hero moved to (%d,%d) in region %d", hero.X, hero.Y, heroRegion)

					newlyVisibleDoors := make([]protocol.ThresholdLite, 0)
					for id, info := range state.Doors {
						// Check if door is on the edge of hero's current room
						doorOnCurrentRoom := (info.RegionA == heroRegion) || (info.RegionB == heroRegion)

						// Check line-of-sight visibility
						visible := isEdgeVisible(state, hero.X, hero.Y, info.Edge)

						// Add distance check to prevent showing all corridor doors
						dx := hero.X - info.Edge.X
						dy := hero.Y - info.Edge.Y
						distance := dx*dx + dy*dy
						// nearbyDoor := distance <= 25 // Within 5 tile radius

						// Door visibility rules:
						// 1. If in a room: show doors on room edges OR doors with line-of-sight
						// 2. If in corridor: ONLY show doors with line-of-sight (no auto-discovery)
						shouldShow := false
						if heroRegion != state.CorridorRegion {
							// In a room: show room-edge doors or LOS doors
							shouldShow = visible || doorOnCurrentRoom
						} else {
							// In corridor: only show doors with line-of-sight
							shouldShow = visible
						}

						if shouldShow && !state.KnownDoors[id] {
							state.KnownDoors[id] = true
							newlyVisibleDoors = append(newlyVisibleDoors, protocol.ThresholdLite{
								ID:          id,
								X:           info.Edge.X,
								Y:           info.Edge.Y,
								Orientation: string(info.Edge.Orientation),
								Kind:        "DoorSocket",
								State:       info.State,
							})
							log.Printf("newly discovered door %s at (%d,%d,%s) - LOS: %v, OnCurrentRoom: %v, Distance: %d (hero region: %d, corridor region: %d, door regions: %d|%d)",
								id, info.Edge.X, info.Edge.Y, info.Edge.Orientation, visible, doorOnCurrentRoom, distance, heroRegion, state.CorridorRegion, info.RegionA, info.RegionB)
						}
					}

					// Send newly visible doors to client (client will add them to existing ones)
					if len(newlyVisibleDoors) > 0 {
						log.Printf("sending %d newly visible doors to client", len(newlyVisibleDoors))
						seq4 := atomic.AddUint64(&sequence, 1)
						envelope := protocol.PatchEnvelope{
							Sequence: seq4,
							EventID:  0,
							Type:     "DoorsVisible",
							Payload:  protocol.DoorsVisible{Doors: newlyVisibleDoors},
						}
						b4, err := json.Marshal(envelope)
						if err != nil {
							log.Printf("failed to marshal DoorsVisible: %v", err)
						} else {
							log.Printf("broadcasting DoorsVisible: %s", string(b4))
							hub.Broadcast(b4)
						}
					}

					// Check for newly visible blocking walls
					_, newlyVisibleBlockingWalls := getVisibleBlockingWalls(state, hero, quest)
					if len(newlyVisibleBlockingWalls) > 0 {
						log.Printf("sending %d newly visible blocking walls to client", len(newlyVisibleBlockingWalls))
						seq5 := atomic.AddUint64(&sequence, 1)
						envelope := protocol.PatchEnvelope{
							Sequence: seq5,
							EventID:  0,
							Type:     "BlockingWallsVisible",
							Payload:  protocol.BlockingWallsVisible{BlockingWalls: newlyVisibleBlockingWalls},
						}
						b5, err := json.Marshal(envelope)
						if err != nil {
							log.Printf("failed to marshal BlockingWallsVisible: %v", err)
						} else {
							log.Printf("broadcasting BlockingWallsVisible: %s", string(b5))
							hub.Broadcast(b5)
						}
					}

				case "RequestToggleDoor":
					var req protocol.RequestToggleDoor
					if err := json.Unmarshal(env.Payload, &req); err != nil {
						continue
					}

					state.Lock.Lock()
					info, ok := state.Doors[req.ThresholdID]
					if !ok || info == nil || info.State == "open" {
						state.Lock.Unlock()
						continue
					}
					info.State = "open"

					var toReveal []int
					a, b := info.RegionA, info.RegionB
					if state.RevealedRegions[a] && !state.RevealedRegions[b] {
						state.RevealedRegions[b] = true
						toReveal = append(toReveal, b)
					} else if state.RevealedRegions[b] && !state.RevealedRegions[a] {
						state.RevealedRegions[a] = true
						toReveal = append(toReveal, a)
					}
					state.Lock.Unlock()

					seq := atomic.AddUint64(&sequence, 1)
					b1, _ := json.Marshal(protocol.PatchEnvelope{
						Sequence: seq,
						EventID:  0,
						Type:     "DoorStateChanged",
						Payload:  protocol.DoorStateChanged{ThresholdID: req.ThresholdID, State: "open"},
					})
					hub.Broadcast(b1)

					if len(toReveal) > 0 {
						seq2 := atomic.AddUint64(&sequence, 1)
						b2, _ := json.Marshal(protocol.PatchEnvelope{
							Sequence: seq2,
							EventID:  0,
							Type:     "RegionsRevealed",
							Payload:  protocol.RegionsRevealed{IDs: toReveal},
						})
						hub.Broadcast(b2)
					}
					hero := state.Entities["hero-1"]
					visible := computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)
					state.Lock.Lock()
					newlyKnown := addKnownRegions(state, visible)
					state.Lock.Unlock()
					seq3 := atomic.AddUint64(&sequence, 1)
					b3, _ := json.Marshal(protocol.PatchEnvelope{
						Sequence: seq3,
						EventID:  0,
						Type:     "VisibleNow",
						Payload:  protocol.VisibleNow{IDs: visible},
					})
					hub.Broadcast(b3)
					if len(newlyKnown) > 0 {
						seq4 := atomic.AddUint64(&sequence, 1)
						b4, _ := json.Marshal(protocol.PatchEnvelope{
							Sequence: seq4,
							EventID:  0,
							Type:     "RegionsKnown",
							Payload:  protocol.RegionsKnown{IDs: newlyKnown},
						})
						hub.Broadcast(b4)
					}

					// Check for newly visible doors after opening door
					hero = state.Entities["hero-1"]
					heroIdx := hero.Y*state.Segment.Width + hero.X
					heroRegion := state.RegionMap.TileRegionIDs[heroIdx]

					newlyVisibleDoors := make([]protocol.ThresholdLite, 0)
					for id, info := range state.Doors {
						// Check if door is on the edge of hero's current room
						doorOnCurrentRoom := (info.RegionA == heroRegion) || (info.RegionB == heroRegion)

						// Check line-of-sight visibility
						visible := isEdgeVisible(state, hero.X, hero.Y, info.Edge)

						// Add distance check to prevent showing all corridor doors
						dx := hero.X - info.Edge.X
						dy := hero.Y - info.Edge.Y
						distance := dx*dx + dy*dy
						// nearbyDoor := distance <= 25 // Within 5 tile radius

						// Door visibility rules:
						// 1. If in a room: show doors on room edges OR doors with line-of-sight
						// 2. If in corridor: ONLY show doors with line-of-sight (no auto-discovery)
						shouldShow := false
						if heroRegion != state.CorridorRegion {
							// In a room: show room-edge doors or LOS doors
							shouldShow = visible || doorOnCurrentRoom
						} else {
							// In corridor: only show doors with line-of-sight
							shouldShow = visible
						}

						if shouldShow && !state.KnownDoors[id] {
							state.KnownDoors[id] = true
							newlyVisibleDoors = append(newlyVisibleDoors, protocol.ThresholdLite{
								ID:          id,
								X:           info.Edge.X,
								Y:           info.Edge.Y,
								Orientation: string(info.Edge.Orientation),
								Kind:        "DoorSocket",
								State:       info.State,
							})
							log.Printf("newly discovered door after opening %s at (%d,%d,%s) - LOS: %v, OnCurrentRoom: %v, Distance: %d (hero region: %d, corridor region: %d, door regions: %d|%d)",
								id, info.Edge.X, info.Edge.Y, info.Edge.Orientation, visible, doorOnCurrentRoom, distance, heroRegion, state.CorridorRegion, info.RegionA, info.RegionB)
						}
					}

					if len(newlyVisibleDoors) > 0 {
						seq5 := atomic.AddUint64(&sequence, 1)
						b5, _ := json.Marshal(protocol.PatchEnvelope{
							Sequence: seq5,
							EventID:  0,
							Type:     "DoorsVisible",
							Payload:  protocol.DoorsVisible{Doors: newlyVisibleDoors},
						})
						hub.Broadcast(b5)
					}

					// Check for newly visible blocking walls after door toggle
					_, newlyVisibleBlockingWalls := getVisibleBlockingWalls(state, hero, quest)
					if len(newlyVisibleBlockingWalls) > 0 {
						seq6 := atomic.AddUint64(&sequence, 1)
						b6, _ := json.Marshal(protocol.PatchEnvelope{
							Sequence: seq6,
							EventID:  0,
							Type:     "BlockingWallsVisible",
							Payload:  protocol.BlockingWallsVisible{BlockingWalls: newlyVisibleBlockingWalls},
						})
						hub.Broadcast(b6)
					}
				default:
				}
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

func edgeCenter(e geometry.EdgeAddress) (float64, float64) {
	if e.Orientation == geometry.Vertical {
		// Vertical door at (x,y) should be on the LEFT edge of tile (x,y)
		return float64(e.X), float64(e.Y) + 0.5
	}
	// Horizontal door at (x,y) should be on the TOP edge of tile (x,y)
	return float64(e.X) + 0.5, float64(e.Y)
}

func isEdgeVisible(state *GameState, fromX, fromY int, target geometry.EdgeAddress) bool {
	sx, sy := float64(fromX)+0.5, float64(fromY)+0.5
	tx, ty := edgeCenter(target)
	dx, dy := tx-sx, ty-sy
	if dx == 0 && dy == 0 {
		return true
	}
	adx, ady := math.Abs(dx), math.Abs(dy)
	stepX, stepY := 0, 0
	if dx > 0 {
		stepX = 1
	} else if dx < 0 {
		stepX = -1
	}
	if dy > 0 {
		stepY = 1
	} else if dy < 0 {
		stepY = -1
	}

	// Debug specific case that's failing
	isDebugCase := fromX == 2 && fromY == 14 && target.X == 1 && target.Y == 11 && target.Orientation == geometry.Vertical
	if isDebugCase {
		log.Printf("LOS DEBUG: Checking from (%d,%d) to door at (%d,%d,%s)", fromX, fromY, target.X, target.Y, target.Orientation)
		log.Printf("LOS DEBUG: Ray from (%.1f,%.1f) to (%.1f,%.1f), direction: (%.1f,%.1f)", sx, sy, tx, ty, dx, dy)
	}

	tDeltaX, tDeltaY := math.Inf(1), math.Inf(1)
	if adx > 0 {
		tDeltaX = 1.0 / adx
	}
	if ady > 0 {
		tDeltaY = 1.0 / ady
	}

	xCell := int(math.Floor(sx))
	yCell := int(math.Floor(sy))

	var tMaxX, tMaxY float64
	if stepX > 0 {
		tMaxX = (float64(xCell+1) - sx) / adx
	} else if stepX < 0 {
		tMaxX = (sx - float64(xCell)) / adx
	} else {
		tMaxX = math.Inf(1)
	}
	if stepY > 0 {
		tMaxY = (float64(yCell+1) - sy) / ady
	} else if stepY < 0 {
		tMaxY = (sy - float64(yCell)) / ady
	} else {
		tMaxY = math.Inf(1)
	}

	stepCount := 0
	for range 2048 {
		stepCount++
		if isDebugCase {
			log.Printf("LOS DEBUG: Step %d - at cell (%d,%d), tMaxX=%.3f, tMaxY=%.3f", stepCount, xCell, yCell, tMaxX, tMaxY)
		}

		if tMaxX < tMaxY {
			var crossed geometry.EdgeAddress
			if stepX > 0 {
				crossed = geometry.EdgeAddress{X: xCell + 1, Y: yCell, Orientation: geometry.Vertical}
				xCell++
			} else {
				crossed = geometry.EdgeAddress{X: xCell, Y: yCell, Orientation: geometry.Vertical}
				xCell--
			}
			if isDebugCase {
				log.Printf("LOS DEBUG: Crossing vertical edge at (%d,%d)", crossed.X, crossed.Y)
			}
			if crossed == target {
				if isDebugCase {
					log.Printf("LOS DEBUG: Found target!")
				}
				return true
			}
			if state.BlockedWalls[crossed] {
				if isDebugCase {
					log.Printf("LOS DEBUG: Blocked by vertical wall at (%d,%d)", crossed.X, crossed.Y)
				}
				return false
			}
			if id, ok := state.DoorByEdge[crossed]; ok {
				if d := state.Doors[id]; d != nil && d.State != "open" {
					if isDebugCase {
						log.Printf("LOS DEBUG: Blocked by closed door at (%d,%d)", crossed.X, crossed.Y)
					}
					return false
				}
			}
			tMaxX += tDeltaX
		} else if tMaxY < tMaxX {
			var crossed geometry.EdgeAddress
			if stepY > 0 {
				crossed = geometry.EdgeAddress{X: xCell, Y: yCell + 1, Orientation: geometry.Horizontal}
				yCell++
			} else {
				crossed = geometry.EdgeAddress{X: xCell, Y: yCell, Orientation: geometry.Horizontal}
				yCell--
			}
			if isDebugCase {
				log.Printf("LOS DEBUG: Crossing horizontal edge at (%d,%d)", crossed.X, crossed.Y)
			}
			if crossed == target {
				if isDebugCase {
					log.Printf("LOS DEBUG: Found target!")
				}
				return true
			}
			if state.BlockedWalls[crossed] {
				if isDebugCase {
					log.Printf("LOS DEBUG: Blocked by horizontal wall at (%d,%d)", crossed.X, crossed.Y)
				}
				return false
			}
			if id, ok := state.DoorByEdge[crossed]; ok {
				if d := state.Doors[id]; d != nil && d.State != "open" {
					if isDebugCase {
						log.Printf("LOS DEBUG: Blocked by closed door at (%d,%d)", crossed.X, crossed.Y)
					}
					return false
				}
			}
			tMaxY += tDeltaY
		} else {
			oldX, oldY := xCell, yCell

			var vEdge, hEdge geometry.EdgeAddress
			var nextX, nextY = xCell, yCell

			if stepX > 0 {
				vEdge = geometry.EdgeAddress{X: oldX + 1, Y: oldY, Orientation: geometry.Vertical}
				nextX = oldX + 1
			} else if stepX < 0 {
				vEdge = geometry.EdgeAddress{X: oldX, Y: oldY, Orientation: geometry.Vertical}
				nextX = oldX - 1
			}

			if stepY > 0 {
				hEdge = geometry.EdgeAddress{X: oldX, Y: oldY + 1, Orientation: geometry.Horizontal}
				nextY = oldY + 1
			} else if stepY < 0 {
				hEdge = geometry.EdgeAddress{X: oldX, Y: oldY, Orientation: geometry.Horizontal}
				nextY = oldY - 1
			}

			if isDebugCase {
				log.Printf("LOS DEBUG: Diagonal step from (%d,%d) to (%d,%d), checking edges v(%d,%d) h(%d,%d)",
					oldX, oldY, nextX, nextY, vEdge.X, vEdge.Y, hEdge.X, hEdge.Y)
			}

			// If either edge *is* the target, we can see it.
			if vEdge == target || hEdge == target {
				if isDebugCase {
					log.Printf("LOS DEBUG: Found target at diagonal!")
				}
				return true
			}

			// Corner rule: if EITHER of the two edges at the corner is blocking, LOS stops
			if state.BlockedWalls[vEdge] || state.BlockedWalls[hEdge] {
				if isDebugCase {
					vBlocked := state.BlockedWalls[vEdge]
					hBlocked := state.BlockedWalls[hEdge]
					log.Printf("LOS DEBUG: Blocked at corner - v(%d,%d)=%t h(%d,%d)=%t",
						vEdge.X, vEdge.Y, vBlocked, hEdge.X, hEdge.Y, hBlocked)
				}
				return false
			}
			if id, ok := state.DoorByEdge[vEdge]; ok {
				if d := state.Doors[id]; d != nil && d.State != "open" {
					if isDebugCase {
						log.Printf("LOS DEBUG: Blocked by closed door at v(%d,%d)", vEdge.X, vEdge.Y)
					}
					return false
				}
			}
			if id, ok := state.DoorByEdge[hEdge]; ok {
				if d := state.Doors[id]; d != nil && d.State != "open" {
					if isDebugCase {
						log.Printf("LOS DEBUG: Blocked by closed door at h(%d,%d)", hEdge.X, hEdge.Y)
					}
					return false
				}
			}

			// advance from the corner
			xCell, yCell = nextX, nextY
			tMaxX += tDeltaX
			tMaxY += tDeltaY
		}
	}
	if isDebugCase {
		log.Printf("LOS DEBUG: No obstruction found after %d steps, returning false", stepCount)
	}
	return false
}

func computeVisibleRoomRegionsNow(state *GameState, from protocol.TileAddress, corridorRegion int) []int {
	seen := make(map[int]struct{}, len(state.Doors))
	for _, info := range state.Doors {
		room := info.RegionA
		if room == corridorRegion {
			room = info.RegionB
		}
		if isEdgeVisible(state, from.X, from.Y, info.Edge) {
			seen[room] = struct{}{}
		}
	}
	out := make([]int, 0, len(seen))
	for rid := range seen {
		out = append(out, rid)
	}
	return out
}

func addKnownRegions(state *GameState, ids []int) (added []int) {
	for _, rid := range ids {
		if !state.KnownRegions[rid] {
			state.KnownRegions[rid] = true
			added = append(added, rid)
		}
	}
	return
}
