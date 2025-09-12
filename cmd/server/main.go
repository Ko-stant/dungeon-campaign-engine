package main

import (
	"context"
	"encoding/json"
	"log"
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

type GameState struct {
	Segment         geometry.Segment
	RegionMap       geometry.RegionMap
	BlockedEdges    map[geometry.EdgeAddress]bool
	Entities        map[string]protocol.TileAddress
	RevealedRegions map[int]bool
	Lock            sync.Mutex
}

func buildBlockedEdges(seg geometry.Segment) map[geometry.EdgeAddress]bool {
	m := make(map[geometry.EdgeAddress]bool, len(seg.WallsVertical)+len(seg.WallsHorizontal)+len(seg.DoorSockets))
	for _, e := range seg.WallsVertical {
		m[e] = true
	}
	for _, e := range seg.WallsHorizontal {
		m[e] = true
	}
	for _, e := range seg.DoorSockets {
		m[e] = true
	}
	return m
}

func firstCorridorTile(seg geometry.Segment, rm geometry.RegionMap) (int, int) {
	for y := 0; y < seg.Height; y++ {
		for x := 0; x < seg.Width; x++ {
			idx := y*seg.Width + x
			rid := rm.TileRegionIDs[idx]
			if rid == 0 {
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

func main() {
	segment := geometry.CorridorsAndRoomsSegment(26, 19)
	regionMap := geometry.BuildRegionMap(segment)
	blocked := buildBlockedEdges(segment)

	corridorRegion := regionMap.TileRegionIDs[0]
	for i, rid := range regionMap.TileRegionIDs {
		if rid < corridorRegion {
			corridorRegion = rid
		}
		_ = i
	}

	startX, startY := firstCorridorTile(segment, regionMap)
	hero := protocol.TileAddress{SegmentID: segment.ID, X: startX, Y: startY}

	state := &GameState{
		Segment:         segment,
		RegionMap:       regionMap,
		BlockedEdges:    blocked,
		Entities:        map[string]protocol.TileAddress{"hero-1": hero},
		RevealedRegions: map[int]bool{corridorRegion: true},
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
					if req.DX == 0 && req.DY == 0 {
						continue
					}
					if (req.DX != 0 && req.DY != 0) || (req.DX < -1 || req.DX > 1) || (req.DY < -1 || req.DY > 1) {
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
					if state.BlockedEdges[edge] {
						state.Lock.Unlock()
						continue
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
				default:
				}
			}
		}(conn)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var revealed []int
		for id := range state.RevealedRegions {
			revealed = append(revealed, id)
		}
		entities := []protocol.EntityLite{
			{ID: "hero-1", Kind: "hero", Tile: state.Entities["hero-1"]},
		}
		s := protocol.Snapshot{
			MapID:             "dev-map",
			PackID:            "dev-pack@v1",
			Turn:              1,
			LastEventID:       0,
			MapWidth:          segment.Width,
			MapHeight:         segment.Height,
			RegionsCount:      regionMap.RegionsCount,
			TileRegionIDs:     regionMap.TileRegionIDs,
			RevealedRegionIDs: revealed,
			DoorStates:        []byte{},
			Entities:          entities,
			Variables:         map[string]any{"ui.debug": true},
			ProtocolVersion:   "v0",
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
