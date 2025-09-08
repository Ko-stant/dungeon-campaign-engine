package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

func main() {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	hub := ws.NewHub()
	var sequence uint64

	segment := geometry.DevSegment()
	regionMap := geometry.BuildRegionMap(segment)

	door := segment.DoorSockets[0]
	doorID := "dev-door-12-9-v"

	leftTileIndex := door.Y*segment.Width + door.X
	rightTileIndex := door.Y*segment.Width + (door.X + 1)
	leftRegionID := regionMap.TileRegionIDs[leftTileIndex]
	rightRegionID := regionMap.TileRegionIDs[rightTileIndex]

	var doorOpen atomic.Bool

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
				_, _, err := c.Read(context.Background())
				if err != nil {
					return
				}
			}
		}(conn)
	})

	mux.HandleFunc("/dev/toggle-door", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		newOpen := !doorOpen.Load()
		doorOpen.Store(newOpen)

		seq := atomic.AddUint64(&sequence, 1)
		ds := protocol.DoorStateChanged{
			ThresholdID: doorID,
			State:       map[bool]string{true: "open", false: "closed"}[newOpen],
		}
		b1, _ := json.Marshal(protocol.PatchEnvelope{
			Sequence: seq,
			EventID:  0,
			Type:     "DoorStateChanged",
			Payload:  ds,
		})
		hub.Broadcast(b1)

		if newOpen {
			seq2 := atomic.AddUint64(&sequence, 1)
			rr := protocol.RegionsRevealed{IDs: []int{rightRegionID}}
			b2, _ := json.Marshal(protocol.PatchEnvelope{
				Sequence: seq2,
				EventID:  0,
				Type:     "RegionsRevealed",
				Payload:  rr,
			})
			hub.Broadcast(b2)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := protocol.Snapshot{
			MapID:             "dev-map",
			PackID:            "dev-pack@v1",
			Turn:              1,
			LastEventID:       0,
			MapWidth:          segment.Width,
			MapHeight:         segment.Height,
			RegionsCount:      regionMap.RegionsCount,
			TileRegionIDs:     regionMap.TileRegionIDs,
			RevealedRegionIDs: []int{leftRegionID},
			DoorStates:        []byte{},
			DoorSockets: []protocol.DoorSocketLite{
				{ID: doorID, X: door.X, Y: door.Y, Orientation: "vertical", State: "closed"},
			},
			Entities:        []protocol.EntityLite{},
			Variables:       map[string]any{"ui.debug": true},
			ProtocolVersion: "v0",
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
