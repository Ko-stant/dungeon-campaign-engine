package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

func main() {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("internal/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

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

	log.Printf("listening on :%s", port)
	http.ListenAndServe(":"+port, mux)
}
