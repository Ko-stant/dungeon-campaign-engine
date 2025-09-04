package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

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

	var sequence uint64

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

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			seq := atomic.AddUint64(&sequence, 1)
			patch := protocol.PatchEnvelope{
				Sequence: seq,
				EventID:  0,
				Type:     "VariablesChanged",
				Payload:  protocol.VariablesChanged{Entries: map[string]interface{}{"debugTick": seq}},
			}
			bytes, err := json.Marshal(patch)
			if err != nil {
				continue
			}
			hub.Broadcast(bytes)
		}
	}()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := protocol.Snapshot{
			MapID:           "dev-map",
			PackID:          "dev-pack@v1",
			Turn:            1,
			LastEventID:     0,
			DoorStates:      []byte{},
			RevealedRegions: []byte{},
			Entities:        []protocol.EntityLite{},
			Variables:       map[string]any{"ui.debug": true},
			ProtocolVersion: "v0",
		}
		err := views.IndexPage(s).Render(r.Context(), w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return context.Background()
		},
	}

	log.Printf("listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("server error: %v", err)
	}
}
