package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/app"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// mountApp registers the table-companion app (map creator, and later the
// tracker) on the legacy server's mux. It needs DATABASE_URL; without it the
// app's routes answer 503 with instructions, and the legacy game still runs.
// It returns a cleanup function.
func mountApp(mux *http.ServeMux) func() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Printf("app: DATABASE_URL is not set; map creator disabled (see .env and make db-up)")
		mountUnavailable(mux, "DATABASE_URL is not set. Add it to .env and run make db-up.")
		return func() {}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := store.Migrate(ctx, dbURL); err != nil {
		log.Printf("app: migrate: %v; map creator disabled (is the database running? make db-up)", err)
		mountUnavailable(mux, "The database is not reachable. Start it with make db-up and restart the server.")
		return func() {}
	}
	st, err := store.Open(ctx, dbURL)
	if err != nil {
		log.Printf("app: open store: %v", err)
		mountUnavailable(mux, "The database is not reachable. Start it with make db-up and restart the server.")
		return func() {}
	}
	catalog, err := content.Load(os.DirFS("content"))
	if err != nil {
		log.Printf("app: load content catalog: %v; continuing with an empty catalog", err)
		catalog = &content.Catalog{Furniture: []content.FurnitureDef{}, Monsters: []content.MonsterDef{}, Heroes: []content.HeroDef{}}
	}

	app.New(st, catalog).Register(mux)
	log.Printf("app: map creator at /maps (%d furniture, %d monsters, %d heroes in catalog)",
		len(catalog.Furniture), len(catalog.Monsters), len(catalog.Heroes))
	return st.Close
}

func mountUnavailable(mux *http.ServeMux, reason string) {
	unavailable := func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Map creator unavailable: "+reason, http.StatusServiceUnavailable)
	}
	mux.HandleFunc("/maps", unavailable)
	mux.HandleFunc("/maps/", unavailable)
	mux.HandleFunc("/api/", unavailable)
}
