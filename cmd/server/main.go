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
	Segment         geometry.Segment
	RegionMap       geometry.RegionMap
	BlockedWalls    map[geometry.EdgeAddress]bool
	Doors           map[string]*DoorInfo
	DoorByEdge      map[geometry.EdgeAddress]string
	Entities        map[string]protocol.TileAddress
	RevealedRegions map[int]bool
	Lock            sync.Mutex
	KnownRegions    map[int]bool
	CorridorRegion  int
}

func buildBlockedWalls(seg geometry.Segment) map[geometry.EdgeAddress]bool {
	m := make(map[geometry.EdgeAddress]bool, len(seg.WallsVertical)+len(seg.WallsHorizontal))
	for _, e := range seg.WallsVertical {
		m[e] = true
	}
	for _, e := range seg.WallsHorizontal {
		m[e] = true
	}
	return m
}

func firstCorridorTile(seg geometry.Segment, rm geometry.RegionMap) (int, int) {
	for y := 0; y < seg.Height; y++ {
		for x := 0; x < seg.Width; x++ {
			idx := y*seg.Width + x
			if rm.TileRegionIDs[idx] == rm.TileRegionIDs[0] {
				return x, y
			}
		}
	}
	return 1, 1
}

func edgeForStep(x, y, dx, dy int) geometry.EdgeAddress {
	if dx == 1 && dy == 0 {
		return geometry.EdgeAddress{X: x, Y: y, Orientation: geometry.Vertical}
	}
	if dx == -1 && dy == 0 {
		return geometry.EdgeAddress{X: x - 1, Y: y, Orientation: geometry.Vertical}
	}
	if dx == 0 && dy == 1 {
		return geometry.EdgeAddress{X: x, Y: y, Orientation: geometry.Horizontal}
	}
	return geometry.EdgeAddress{X: x, Y: y - 1, Orientation: geometry.Horizontal}
}

func makeDoorID(segmentID string, e geometry.EdgeAddress) string {
	return fmt.Sprintf("%s:%d:%d:%s", segmentID, e.X, e.Y, e.Orientation)
}

func main() {
	segment := geometry.CorridorsAndRoomsWithDoorsSegment(26, 19)
	regionMap := geometry.BuildRegionMap(segment)

	doors := make(map[string]*DoorInfo, len(segment.DoorSockets))
	doorByEdge := make(map[geometry.EdgeAddress]string, len(segment.DoorSockets))
	for _, e := range segment.DoorSockets {
		id := makeDoorID(segment.ID, e)
		a, b := geometry.RegionsAcrossDoor(regionMap, segment, e)
		doors[id] = &DoorInfo{Edge: e, RegionA: a, RegionB: b, State: "closed"}
		doorByEdge[e] = id
	}

	startX, startY := firstCorridorTile(segment, regionMap)
	hero := protocol.TileAddress{SegmentID: segment.ID, X: startX, Y: startY}
	corridorRegion := regionMap.TileRegionIDs[startY*segment.Width+startX]

	state := &GameState{
		Segment:         segment,
		RegionMap:       regionMap,
		BlockedWalls:    buildBlockedWalls(segment),
		Doors:           doors,
		DoorByEdge:      doorByEdge,
		Entities:        map[string]protocol.TileAddress{"hero-1": hero},
		RevealedRegions: map[int]bool{corridorRegion: true},
		KnownRegions:    map[int]bool{corridorRegion: true},
		CorridorRegion:  corridorRegion,
	}

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
						state.Lock.Unlock()
						continue
					}
					edge := edgeForStep(tile.X, tile.Y, req.DX, req.DY)
					if state.BlockedWalls[edge] {
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
		thresholds := make([]protocol.ThresholdLite, 0, len(state.Doors))
		for id, info := range state.Doors {
			thresholds = append(thresholds, protocol.ThresholdLite{
				ID:          id,
				X:           info.Edge.X,
				Y:           info.Edge.Y,
				Orientation: string(info.Edge.Orientation),
				Kind:        "DoorSocket",
				State:       info.State,
			})
			log.Printf("door %s at (%d,%d,%s) regions=%d|%d state=%s",
				id, info.Edge.X, info.Edge.Y, info.Edge.Orientation, info.RegionA, info.RegionB, info.State)
		}
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
		return float64(e.X + 1), float64(e.Y) + 0.5
	}
	return float64(e.X) + 0.5, float64(e.Y + 1)
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
		if tMaxX < tMaxY {
			var crossed geometry.EdgeAddress
			if stepX > 0 {
				crossed = geometry.EdgeAddress{X: xCell, Y: yCell, Orientation: geometry.Vertical}
				xCell++
			} else {
				crossed = geometry.EdgeAddress{X: xCell - 1, Y: yCell, Orientation: geometry.Vertical}
				xCell--
			}
			if crossed == target {
				return true
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
				crossed = geometry.EdgeAddress{X: xCell, Y: yCell, Orientation: geometry.Horizontal}
				yCell++
			} else {
				crossed = geometry.EdgeAddress{X: xCell, Y: yCell - 1, Orientation: geometry.Horizontal}
				yCell--
			}
			if crossed == target {
				return true
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
				vEdge = geometry.EdgeAddress{X: oldX, Y: oldY, Orientation: geometry.Vertical}
				nextX = oldX + 1
			} else if stepX < 0 {
				vEdge = geometry.EdgeAddress{X: oldX - 1, Y: oldY, Orientation: geometry.Vertical}
				nextX = oldX - 1
			}

			if stepY > 0 {
				hEdge = geometry.EdgeAddress{X: oldX, Y: oldY, Orientation: geometry.Horizontal}
				nextY = oldY + 1
			} else if stepY < 0 {
				hEdge = geometry.EdgeAddress{X: oldX, Y: oldY - 1, Orientation: geometry.Horizontal}
				nextY = oldY - 1
			}

			// If either edge *is* the target, we can see it.
			if vEdge == target || hEdge == target {
				return true
			}

			// Corner rule: if EITHER of the two edges at the corner is blocking,
			// LOS stops. This prevents "peeking" around corners.
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
