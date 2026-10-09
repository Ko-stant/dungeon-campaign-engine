// Command server runs the Dungeon Campaign Engine: the map creator (/maps) and
// the table-companion tracker (/campaigns, /play).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/auth"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/dotenv"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web"
)

type routesConfig struct {
	staticDir string
	assetsDir string
	auth      auth.Config
}

// newMux builds every route. It returns a cleanup function for the database.
func newMux(cfg routesConfig) (*http.ServeMux, func()) {
	mux := http.NewServeMux()
	mux.Handle("/static/", web.NoCache(http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.staticDir)))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/campaigns", http.StatusSeeOther)
	})
	cleanup := mountApp(mux, cfg.assetsDir, cfg.auth)
	return mux, cleanup
}

// listenPort is the port to serve on: PORT when the host sets it (Render
// does), else APP_PORT (.env), else 8080.
func listenPort(getenv func(string) string) string {
	for _, k := range []string{"PORT", "APP_PORT"} {
		if p := getenv(k); p != "" {
			return p
		}
	}
	return "8080"
}

// protect blocks cross-site form posts and API calls (another site making a
// signed-in browser change things here) whenever sign-in is on.
func protect(h http.Handler, cfg auth.Config) http.Handler {
	if !cfg.On() {
		return h
	}
	return http.NewCrossOriginProtection().Handler(h)
}

func main() {
	if err := dotenv.Load(".env"); err != nil {
		log.Printf("warning: reading .env: %v", err)
	}
	port := listenPort(os.Getenv)

	authCfg, err := auth.ConfigFromEnv(os.Getenv)
	if err != nil {
		log.Fatalf("sign-in settings: %v", err)
	}
	log.Printf("sign-in: %s (%d admins)", authCfg.Mode, len(authCfg.Admins))

	mux, cleanup := newMux(routesConfig{staticDir: "internal/web/static", assetsDir: "assets", auth: authCfg})
	defer cleanup()

	srv := &http.Server{Addr: ":" + port, Handler: protect(mux, authCfg), ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Dungeon Campaign Engine on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
