package main

import (
	"fmt"
	"log"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

func loadGameContent() (*geometry.BoardDefinition, *geometry.QuestDefinition, error) {
	// Load the static HeroQuest board
	board, err := geometry.LoadBoardFromFile("content/base/board.json")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load board: %v", err)
	}

	// Load the quest
	quest, err := geometry.LoadQuestFromFile("content/base/quests/quest-01.json")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load quest: %v", err)
	}

	return board, quest, nil
}

func createGameSegment(board *geometry.BoardDefinition, quest *geometry.QuestDefinition) (geometry.Segment, geometry.RegionMap) {
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

	return segment, regionMap
}

func createDoorsFromQuest(quest *geometry.QuestDefinition, segment geometry.Segment, regionMap geometry.RegionMap) (map[string]*DoorInfo, map[geometry.EdgeAddress]string) {
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

	return doors, doorByEdge
}

func initializeGameState(board *geometry.BoardDefinition, quest *geometry.QuestDefinition) (*GameState, protocol.TileAddress, error) {
	segment, regionMap := createGameSegment(board, quest)
	doors, _ := createDoorsFromQuest(quest, segment, regionMap)

	// Start hero in the quest's starting room
	startX, startY, err := geometry.FindStartingTileInRoom(board, quest.StartingRoom)
	if err != nil {
		return nil, protocol.TileAddress{}, fmt.Errorf("failed to find starting position: %v", err)
	}
	log.Printf("hero starting in room %d at position (%d,%d)", quest.StartingRoom, startX, startY)

	state := NewGameState(segment, regionMap, quest)

	// Add doors from quest configuration
	for id, door := range doors {
		state.AddDoor(id, door)
	}

	// Set hero starting position
	state.SetHeroPosition("hero-1", startX, startY)

	// Get hero position for initial discovery
	hero := state.Entities["hero-1"]
	heroIdx := hero.Y*state.Segment.Width + hero.X
	heroRegion := state.RegionMap.TileRegionIDs[heroIdx]

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

	return state, hero, nil
}
