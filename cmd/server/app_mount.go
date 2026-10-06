package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/app"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/auth"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// mountApp registers the map creator and tracker. It needs DATABASE_URL;
// without it (or without a reachable database) the app's routes answer 503
// with instructions. It returns a cleanup function.
func mountApp(mux *http.ServeMux, contentDir, assetsDir string, authCfg auth.Config) func() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Printf("app: DATABASE_URL is not set; see .env and make db-up")
		mountUnavailable(mux, "DATABASE_URL is not set. Add it to .env and run make db-up.")
		return func() {}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := store.Migrate(ctx, dbURL); err != nil {
		log.Printf("app: migrate: %v (is the database running? make db-up)", err)
		mountUnavailable(mux, "The database is not reachable. Start it with make db-up and restart the server.")
		return func() {}
	}
	st, err := store.Open(ctx, dbURL)
	if err != nil {
		log.Printf("app: open store: %v", err)
		mountUnavailable(mux, "The database is not reachable. Start it with make db-up and restart the server.")
		return func() {}
	}
	catalog, err := content.Load(os.DirFS(contentDir))
	if err != nil {
		log.Printf("app: load content catalog: %v; continuing with an empty catalog", err)
		catalog = &content.Catalog{Furniture: []content.FurnitureDef{}, Monsters: []content.MonsterDef{}, Heroes: []content.HeroDef{}, Traps: []content.TrapDef{}}
	}

	server := app.New(st, catalog)
	server.SetAuth(authCfg)
	if authCfg.On() {
		if err := st.DeleteExpiredLoginSessions(ctx); err != nil {
			log.Printf("app: clearing expired sign-ins: %v", err)
		}
	}
	audioDir := os.Getenv("AUDIO_DIR")
	if audioDir == "" {
		audioDir = "audio"
	}
	server.SetAudioDir(audioDir)
	server.SetAssetsDir(assetsDir)
	server.Register(mux)
	log.Printf("app: read-aloud audio clips in %s", audioDir)
	if authCfg.On() {
		members := "approved by an admin"
		if authCfg.OpenMembership {
			members = "open to everyone who signs in"
		}
		log.Printf("app: membership %s", members)
	}
	log.Printf("app: ready (%d furniture, %d monsters, %d heroes, %d traps in catalog)",
		len(catalog.Furniture), len(catalog.Monsters), len(catalog.Heroes), len(catalog.Traps))
	return st.Close
}

func mountUnavailable(mux *http.ServeMux, reason string) {
	unavailable := func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Unavailable: "+reason, http.StatusServiceUnavailable)
	}
	for _, p := range []string{"/maps", "/maps/", "/campaigns", "/campaigns/", "/play/", "/api/"} {
		mux.HandleFunc(p, unavailable)
	}
}
