package main

import (
	"context"
	"encoding/json"
	"fmt"
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

type DoorInfo struct {
	Edge    geometry.EdgeAddress
	RegionA int
	RegionB int
	State   string
}

type GameState struct {
	Segment         geometry.Segment
	RegionMap       geometry.RegionMap
	Doors           map[string]*DoorInfo
	RevealedRegions map[int]bool
	Lock            sync.Mutex
}

func makeDoorID(segmentID string, e geometry.EdgeAddress) string {
	return fmt.Sprintf("%s:%d:%d:%s", segmentID, e.X, e.Y, e.Orientation)
}

func main() {
	segment := geometry.DevSegment()
	regionMap := geometry.BuildRegionMap(segment)

	doorEdge := segment.DoorSockets[0]
	doorID := makeDoorID(segment.ID, doorEdge)
	a, b := geometry.RegionsAcrossDoor(regionMap, segment, doorEdge)

	state := &GameState{
		Segment:         segment,
		RegionMap:       regionMap,
		Doors:           map[string]*DoorInfo{doorID: {Edge: doorEdge, RegionA: a, RegionB: b, State: "closed"}},
		RevealedRegions: map[int]bool{a: true},
	}
	fmt.Printf("STATE: %v\n\n", state.Doors)
	fmt.Printf("doorID: %v\n\n", doorID)

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
				case "RequestToggleDoor":
					var req protocol.RequestToggleDoor
					if err := json.Unmarshal(env.Payload, &req); err != nil {
						fmt.Printf("Err with unmarshal %s\n", err.Error())
						continue
					}
					state.Lock.Lock()
					info, ok := state.Doors[req.ThresholdID]
					if !ok {
						fmt.Printf("Err, id not in state %s\n", req.ThresholdID)
						state.Lock.Unlock()
						continue
					}
					fmt.Println(info.State)
					next := "open"
					if info.State == "open" {
						next = "closed"
					}
					info.State = next
					var toReveal []int
					if next == "open" {
						if state.RevealedRegions[info.RegionA] && !state.RevealedRegions[info.RegionB] {
							state.RevealedRegions[info.RegionB] = true
							toReveal = append(toReveal, info.RegionB)
						} else if state.RevealedRegions[info.RegionB] && !state.RevealedRegions[info.RegionA] {
							state.RevealedRegions[info.RegionA] = true
							toReveal = append(toReveal, info.RegionA)
						}
					}
					state.Lock.Unlock()

					seq := atomic.AddUint64(&sequence, 1)
					b1, _ := json.Marshal(protocol.PatchEnvelope{
						Sequence: seq,
						EventID:  0,
						Type:     "DoorStateChanged",
						Payload:  protocol.DoorStateChanged{ThresholdID: req.ThresholdID, State: next},
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
			Entities:          []protocol.EntityLite{},
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
