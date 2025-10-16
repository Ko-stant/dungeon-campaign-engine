package main

import (
	"encoding/json"
	"log"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

func handleRequestMove(req protocol.RequestMove, state *GameState, hub *ws.Hub, sequence *uint64, quest *geometry.QuestDefinition, furnitureSystem *FurnitureSystem, monsterSystem *MonsterSystem) {
	log.Printf("DEBUG: handleRequestMove called - entity=%s dx=%d dy=%d", req.EntityID, req.DX, req.DY)

	if (req.DX != 0 && req.DY != 0) || req.DX < -1 || req.DX > 1 || req.DY < -1 || req.DY > 1 {
		log.Printf("DEBUG: Movement rejected - invalid dx/dy values")
		return
	}
	if req.DX == 0 && req.DY == 0 {
		log.Printf("DEBUG: Movement rejected - zero movement")
		return
	}

	state.Lock.Lock()
	tile, ok := state.Entities[req.EntityID]
	if !ok {
		log.Printf("DEBUG: Movement rejected - entity %s not found", req.EntityID)
		state.Lock.Unlock()
		return
	}
	log.Printf("DEBUG: Current entity position: (%d,%d), moving to (%d,%d)", tile.X, tile.Y, tile.X+req.DX, tile.Y+req.DY)
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

	// Check if destination tile is blocked by furniture
	if furnitureSystem.BlocksMovement(nx, ny) {
		log.Printf("DEBUG: Movement blocked by furniture: from (%d,%d) to (%d,%d)",
			tile.X, tile.Y, nx, ny)
		state.Lock.Unlock()
		return
	}

	// Check if destination tile is blocked by a monster (heroes cannot move onto monster tiles)
	if monsterSystem.IsMonsterAt(nx, ny) {
		log.Printf("DEBUG: Movement blocked by monster: from (%d,%d) to (%d,%d)",
			tile.X, tile.Y, nx, ny)
		state.Lock.Unlock()
		return
	}

	edge := edgeForStep(tile.X, tile.Y, req.DX, req.DY)
	log.Printf("DEBUG: Checking edge for movement: %+v", edge)
	if state.BlockedWalls[edge] {
		log.Printf("DEBUG: Movement blocked by wall: from (%d,%d) to (%d,%d), blocked edge: %+v",
			tile.X, tile.Y, nx, ny, edge)
		state.Lock.Unlock()
		return
	}
	if id, ok := state.DoorByEdge[edge]; ok {
		log.Printf("DEBUG: Found door %s at edge %+v, state: %s", id, edge, state.Doors[id].State)
		if d := state.Doors[id]; d != nil && d.State != "open" {
			log.Printf("DEBUG: Movement blocked by closed door: %s (state: %s)", id, d.State)
			state.Lock.Unlock()
			return
		}
	}
	tile.X = nx
	tile.Y = ny
	state.Entities[req.EntityID] = tile
	state.Lock.Unlock()

	log.Printf("DEBUG: Movement successful - entity %s moved to (%d,%d)", req.EntityID, nx, ny)
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

func handleRequestToggleDoor(req protocol.RequestToggleDoor, state *GameState, hub *ws.Hub, sequence *uint64, quest *geometry.QuestDefinition, furnitureSystem *FurnitureSystem, monsterSystem *MonsterSystem) {
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

	// Check for newly visible furniture after door toggle
	newlyVisibleFurniture := checkForNewlyVisibleFurniture(state, furnitureSystem)
	if len(newlyVisibleFurniture) > 0 {
		broadcastEvent(hub, sequence, "FurnitureVisible", protocol.FurnitureVisible{Furniture: newlyVisibleFurniture})
	}

	// Check for newly visible monsters after door toggle
	newlyVisibleMonsters := checkForNewlyVisibleMonsters(state, monsterSystem)
	if len(newlyVisibleMonsters) > 0 {
		broadcastEvent(hub, sequence, "MonstersVisible", protocol.MonstersVisible{Monsters: newlyVisibleMonsters})
	}
}

func checkForNewlyVisibleFurniture(state *GameState, furnitureSystem *FurnitureSystem) []protocol.FurnitureLite {
	var newlyVisible []protocol.FurnitureLite

	instances := furnitureSystem.GetAllInstances()
	for _, instance := range instances {
		if instance.Definition == nil {
			continue
		}

		// Skip if furniture is already known
		if state.KnownFurniture[instance.ID] {
			continue
		}

		// Check if furniture is in a revealed region
		furnitureIdx := instance.Position.Y*state.Segment.Width + instance.Position.X
		furnitureRegion := state.RegionMap.TileRegionIDs[furnitureIdx]

		if state.RevealedRegions[furnitureRegion] {
			// Mark furniture as known
			state.KnownFurniture[instance.ID] = true

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
			newlyVisible = append(newlyVisible, furnitureItem)
			log.Printf("DEBUG: Newly visible furniture %s in region %d", instance.ID, furnitureRegion)
		}
	}

	return newlyVisible
}

func checkForNewlyVisibleMonsters(state *GameState, monsterSystem *MonsterSystem) []protocol.MonsterLite {
	var newlyVisible []protocol.MonsterLite

	// If no monster system provided, return empty list
	if monsterSystem == nil {
		return newlyVisible
	}

	monsters := monsterSystem.GetMonsters()
	for _, monster := range monsters {
		// Skip if monster is already known
		if state.KnownMonsters[monster.ID] {
			continue
		}

		// Check if monster is in a revealed region
		monsterIdx := monster.Position.Y*state.Segment.Width + monster.Position.X
		monsterRegion := state.RegionMap.TileRegionIDs[monsterIdx]

		if state.RevealedRegions[monsterRegion] {
			// Mark monster as known and visible
			state.KnownMonsters[monster.ID] = true
			monster.IsVisible = true

			monsterItem := protocol.MonsterLite{
				ID:          monster.ID,
				Type:        string(monster.Type),
				Tile:        monster.Position,
				Body:        monster.Body,
				MaxBody:     monster.MaxBody,
				Mind:        monster.Mind,
				MaxMind:     monster.MaxMind,
				AttackDice:  monster.AttackDice,
				DefenseDice: monster.DefenseDice,
				IsVisible:   monster.IsVisible,
				IsAlive:     monster.IsAlive,
			}
			newlyVisible = append(newlyVisible, monsterItem)
			log.Printf("DEBUG: Newly visible monster %s (%s) in region %d at (%d,%d)",
				monster.ID, monster.Type, monsterRegion, monster.Position.X, monster.Position.Y)
		}
	}

	return newlyVisible
}

func handleWebSocketMessage(data []byte, state *GameState, hub *ws.Hub, sequence *uint64, quest *geometry.QuestDefinition, furnitureSystem *FurnitureSystem, monsterSystem *MonsterSystem, gameManager *GameManager, playerID string) {
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
		handleRequestMove(req, state, hub, sequence, quest, furnitureSystem, monsterSystem)

	case "RequestToggleDoor":
		var req protocol.RequestToggleDoor
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}
		handleRequestToggleDoor(req, state, hub, sequence, quest, furnitureSystem, monsterSystem)

	// Quest Setup Phase
	case "RequestSelectStartingPosition":
		var req protocol.RequestSelectStartingPosition
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}
		handleRequestSelectStartingPosition(req, playerID, gameManager, hub, sequence)

	case "RequestQuestSetupToggleReady":
		var req protocol.RequestToggleReady
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}
		handleRequestQuestSetupToggleReady(playerID, req.IsReady, gameManager, hub, sequence)

	// Dynamic Turn Order
	case "RequestElectSelfAsNextPlayer":
		handleRequestElectSelfAsNextPlayer(playerID, gameManager, hub, sequence)

	case "RequestCancelPlayerElection":
		handleRequestCancelPlayerElection(playerID, gameManager, hub, sequence)

	case "RequestConfirmElectionAndStartTurn":
		handleRequestConfirmElectionAndStartTurn(gameManager, hub, sequence)

	case "RequestCompleteHeroTurn":
		handleRequestCompleteHeroTurn(playerID, gameManager, hub, sequence)

	case "RequestCompleteGMTurn":
		handleRequestCompleteGMTurn(gameManager, hub, sequence)

	// Monster Management
	case "RequestSelectMonster":
		var req protocol.RequestSelectMonster
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}
		handleRequestSelectMonster(req, gameManager, hub, sequence)

	case "RequestMoveMonster":
		var req protocol.RequestMoveMonster
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}
		handleRequestMoveMonster(req, gameManager, hub, sequence, state, furnitureSystem, monsterSystem)

	case "RequestMonsterAttack":
		var req protocol.RequestMonsterAttack
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}
		handleRequestMonsterAttack(req, gameManager, hub, sequence)

	case "RequestUseMonsterAbility":
		var req protocol.RequestUseMonsterAbility
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}
		handleRequestUseMonsterAbility(req, gameManager, hub, sequence)

	default:
		// Unknown message type
	}
}
