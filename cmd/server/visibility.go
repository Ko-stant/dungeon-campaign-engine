package main

import (
	"log"
	"math"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

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

func checkForNewlyVisibleDoors(state *GameState, hero protocol.TileAddress) []protocol.ThresholdLite {
	heroIdx := hero.Y*state.Segment.Width + hero.X
	heroRegion := state.RegionMap.TileRegionIDs[heroIdx]

	log.Printf("Hero at (%d,%d) in region %d", hero.X, hero.Y, heroRegion)

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
	return newlyVisibleDoors
}
