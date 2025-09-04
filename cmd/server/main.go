package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/geometry"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
	"github.com/coder/websocket"
)

func main() {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	hub := ws.NewHub()
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		hub.Add(conn)
		go func(c *websocket.Conn) {
			defer hub.Remove(c)
			defer c.Close(websocket.StatusNormalClosure, "")
			for {
				_, _, err := c.Read(r.Context())
				if err != nil {
					return
				}
			}
		}(conn)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		seg := geometry.DevSegment()
		rm := geometry.BuildRegionMap(seg)

		s := protocol.Snapshot{
			MapID:           "dev-map",
			PackID:          "dev-pack@v1",
			Turn:            1,
			LastEventID:     0,
			MapWidth:        seg.Width,
			MapHeight:       seg.Height,
			RegionsCount:    rm.RegionsCount,
			DoorStates:      []byte{},
			RevealedRegions: []byte{},
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
