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

	"github.com/Ko-stant/dungeon-campaign-engine/internal/dotenv"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web"
)

type routesConfig struct {
	staticDir  string
	assetsDir  string
	contentDir string
}

// newMux builds every route. It returns a cleanup function for the database.
func newMux(cfg routesConfig) (*http.ServeMux, func()) {
	mux := http.NewServeMux()
	mux.Handle("/static/", web.NoCache(http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.staticDir)))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(cfg.assetsDir))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/campaigns", http.StatusSeeOther)
	})
	cleanup := mountApp(mux, cfg.contentDir)
	return mux, cleanup
}

func main() {
	if err := dotenv.Load(".env"); err != nil {
		log.Printf("warning: reading .env: %v", err)
	}
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	mux, cleanup := newMux(routesConfig{staticDir: "internal/web/static", assetsDir: "assets", contentDir: "content"})
	defer cleanup()

	srv := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
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
