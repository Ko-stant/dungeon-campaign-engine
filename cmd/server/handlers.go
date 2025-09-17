package main

import (
	"encoding/json"
	"log"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

func handleRequestMove(req protocol.RequestMove, state *GameState, hub *ws.Hub, sequence *uint64, quest *geometry.QuestDefinition) {
	if (req.DX != 0 && req.DY != 0) || req.DX < -1 || req.DX > 1 || req.DY < -1 || req.DY > 1 {
		return
	}
	if req.DX == 0 && req.DY == 0 {
		return
	}

	state.Lock.Lock()
	tile, ok := state.Entities[req.EntityID]
	if !ok {
		state.Lock.Unlock()
		return
	}
	nx := tile.X + req.DX
	ny := tile.Y + req.DY
	if nx < 0 || ny < 0 || nx >= state.Segment.Width || ny >= state.Segment.Height {
		log.Printf("DEBUG: Movement blocked by bounds check: from (%d,%d) to (%d,%d), bounds: %dx%d",
			tile.X, tile.Y, nx, ny, state.Segment.Width, state.Segment.Height)
		state.Lock.Unlock()
		return
	}

	// Check if destination tile is blocked by a blocking wall
	destTile := protocol.TileAddress{X: nx, Y: ny}
	if state.BlockedTiles[destTile] {
		log.Printf("DEBUG: Movement blocked by blocking wall tile: from (%d,%d) to (%d,%d)",
			tile.X, tile.Y, nx, ny)
		state.Lock.Unlock()
		return
	}

	edge := edgeForStep(tile.X, tile.Y, req.DX, req.DY)
	if state.BlockedWalls[edge] {
		log.Printf("DEBUG: Movement blocked by wall: from (%d,%d) to (%d,%d), blocked edge: %+v",
			tile.X, tile.Y, nx, ny, edge)
		state.Lock.Unlock()
		return
	}
	if id, ok := state.DoorByEdge[edge]; ok {
		if d := state.Doors[id]; d != nil && d.State != "open" {
			state.Lock.Unlock()
			return
		}
	}
	tile.X = nx
	tile.Y = ny
	state.Entities[req.EntityID] = tile
	state.Lock.Unlock()

	broadcastEvent(hub, sequence, "EntityUpdated", protocol.EntityUpdated{ID: req.EntityID, Tile: tile})

	hero := state.Entities[req.EntityID]
	visible := computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)
	state.Lock.Lock()
	newlyKnown := addKnownRegions(state, visible)
	state.Lock.Unlock()
	log.Printf("visibleNow (hero @ %d,%d): %v", hero.X, hero.Y, visible)
	broadcastEvent(hub, sequence, "VisibleNow", protocol.VisibleNow{IDs: visible})

	if len(newlyKnown) > 0 {
		broadcastEvent(hub, sequence, "RegionsKnown", protocol.RegionsKnown{IDs: newlyKnown})
	}

	// Check for newly visible doors
	hero = state.Entities[req.EntityID]
	newlyVisibleDoors := checkForNewlyVisibleDoors(state, hero)

	// Send newly visible doors to client (client will add them to existing ones)
	if len(newlyVisibleDoors) > 0 {
		log.Printf("sending %d newly visible doors to client", len(newlyVisibleDoors))
		broadcastEvent(hub, sequence, "DoorsVisible", protocol.DoorsVisible{Doors: newlyVisibleDoors})
	}

	// Check for newly visible blocking walls
	_, newlyVisibleBlockingWalls := getVisibleBlockingWalls(state, hero, quest)
	if len(newlyVisibleBlockingWalls) > 0 {
		log.Printf("sending %d newly visible blocking walls to client", len(newlyVisibleBlockingWalls))
		broadcastEvent(hub, sequence, "BlockingWallsVisible", protocol.BlockingWallsVisible{BlockingWalls: newlyVisibleBlockingWalls})
	}
}

func handleRequestToggleDoor(req protocol.RequestToggleDoor, state *GameState, hub *ws.Hub, sequence *uint64, quest *geometry.QuestDefinition) {
	state.Lock.Lock()
	info, ok := state.Doors[req.ThresholdID]
	if !ok || info == nil || info.State == "open" {
		state.Lock.Unlock()
		return
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

	broadcastEvent(hub, sequence, "DoorStateChanged", protocol.DoorStateChanged{ThresholdID: req.ThresholdID, State: "open"})

	if len(toReveal) > 0 {
		broadcastEvent(hub, sequence, "RegionsRevealed", protocol.RegionsRevealed{IDs: toReveal})
	}
	hero := state.Entities["hero-1"]
	visible := computeVisibleRoomRegionsNow(state, hero, state.CorridorRegion)
	state.Lock.Lock()
	newlyKnown := addKnownRegions(state, visible)
	state.Lock.Unlock()
	broadcastEvent(hub, sequence, "VisibleNow", protocol.VisibleNow{IDs: visible})
	if len(newlyKnown) > 0 {
		broadcastEvent(hub, sequence, "RegionsKnown", protocol.RegionsKnown{IDs: newlyKnown})
	}

	// Check for newly visible doors after opening door
	hero = state.Entities["hero-1"]
	newlyVisibleDoors := checkForNewlyVisibleDoors(state, hero)

	if len(newlyVisibleDoors) > 0 {
		broadcastEvent(hub, sequence, "DoorsVisible", protocol.DoorsVisible{Doors: newlyVisibleDoors})
	}

	// Check for newly visible blocking walls after door toggle
	_, newlyVisibleBlockingWalls := getVisibleBlockingWalls(state, hero, quest)
	if len(newlyVisibleBlockingWalls) > 0 {
		broadcastEvent(hub, sequence, "BlockingWallsVisible", protocol.BlockingWallsVisible{BlockingWalls: newlyVisibleBlockingWalls})
	}
}

func handleWebSocketMessage(data []byte, state *GameState, hub *ws.Hub, sequence *uint64, quest *geometry.QuestDefinition) {
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
		handleRequestMove(req, state, hub, sequence, quest)

	case "RequestToggleDoor":
		var req protocol.RequestToggleDoor
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}
		handleRequestToggleDoor(req, state, hub, sequence, quest)

	default:
		// Unknown message type
	}
}
