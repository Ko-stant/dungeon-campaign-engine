package main

import (
	"sync"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
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
	KnownFurniture     map[string]bool
	CorridorRegion     int
}

func NewGameState(segment geometry.Segment, regionMap geometry.RegionMap, quest *geometry.QuestDefinition) *GameState {
	state := &GameState{
		Segment:            segment,
		RegionMap:          regionMap,
		BlockedWalls:       buildBlockedWalls(segment),
		BlockedTiles:       buildBlockedTiles(quest),
		Doors:              make(map[string]*DoorInfo),
		DoorByEdge:         make(map[geometry.EdgeAddress]string),
		Entities:           make(map[string]protocol.TileAddress),
		RevealedRegions:    make(map[int]bool),
		KnownRegions:       make(map[int]bool),
		KnownDoors:         make(map[string]bool),
		KnownBlockingWalls: make(map[string]bool),
		KnownFurniture:     make(map[string]bool),
		CorridorRegion:     0, // Corridor is always region 0
	}

	// Initialize known regions - all rooms are "known" (borders visible)
	for i := 0; i < regionMap.RegionsCount; i++ {
		state.KnownRegions[i] = true
	}

	return state
}

func (gs *GameState) AddDoor(id string, door *DoorInfo) {
	gs.Doors[id] = door
	gs.DoorByEdge[door.Edge] = id
}

func (gs *GameState) SetHeroPosition(heroID string, x, y int) {
	hero := protocol.TileAddress{SegmentID: gs.Segment.ID, X: x, Y: y}
	gs.Entities[heroID] = hero

	// Reveal the hero's starting region
	heroIdx := y*gs.Segment.Width + x
	heroRegion := gs.RegionMap.TileRegionIDs[heroIdx]
	gs.RevealedRegions[heroRegion] = true
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