package main

import (
	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// VisibilityCalculatorImpl implements VisibilityCalculator
type VisibilityCalculatorImpl struct {
	logger Logger
}

func NewVisibilityCalculator(logger Logger) *VisibilityCalculatorImpl {
	return &VisibilityCalculatorImpl{logger: logger}
}

func (vc *VisibilityCalculatorImpl) ComputeVisibleRegions(state *GameState, from protocol.TileAddress, corridorRegion int) []int {
	return computeVisibleRoomRegionsNow(state, from, corridorRegion)
}

func (vc *VisibilityCalculatorImpl) CheckNewlyVisibleDoors(state *GameState, hero protocol.TileAddress) []protocol.ThresholdLite {
	return checkForNewlyVisibleDoors(state, hero)
}

func (vc *VisibilityCalculatorImpl) GetVisibleBlockingWalls(state *GameState, hero protocol.TileAddress, quest *geometry.QuestDefinition) ([]protocol.BlockingWallLite, []protocol.BlockingWallLite) {
	return getVisibleBlockingWalls(state, hero, quest)
}

func (vc *VisibilityCalculatorImpl) IsEdgeVisible(state *GameState, fromX, fromY int, target geometry.EdgeAddress) bool {
	return isEdgeVisible(state, fromX, fromY, target)
}

func (vc *VisibilityCalculatorImpl) IsTileCenterVisible(state *GameState, fromX, fromY, toX, toY int) bool {
	return isTileCenterVisible(state, fromX, fromY, toX, toY)
}