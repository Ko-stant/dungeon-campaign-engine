// Package app is the HTTP layer of the table-companion app: the map creator
// API and pages (and, later, the tracker). It is mounted on the server's mux.
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/auth"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// maxBodyBytes bounds request bodies; a 200x200 board is well under 1 MB.
const maxBodyBytes = 5 << 20

const maxNameLength = 120

// Server holds the app's dependencies.
type Server struct {
	store        *store.Store
	sessionLocks sync.Map // session id -> *sync.Mutex
	streams      streams
	// playerStreams carries the player screen's filtered view (see player_api.go).
	playerStreams streams
	// auth is how people sign in; the zero value (no mode) is treated as none.
	auth auth.Config
	// seats are the players connected online (see seat.go).
	seats seatHub
	// assetsDir holds the board art served under /assets/ (empty: none).
	assetsDir string
	// keepalive is how often idle WebSockets are pinged (0: defaultKeepalive).
	keepalive time.Duration
}

// New creates the app server.
// The catalog (classes, monsters, furniture, traps) comes from the database
// (see catalogFor); make import-content puts the base game's there.
func New(st *store.Store) *Server {
	return &Server{store: st, auth: auth.Config{Mode: auth.ModeNone}}
}

// Register mounts the app's routes.
func (s *Server) Register(serveMux *http.ServeMux) {
	serveMux.HandleFunc("GET /healthz", s.healthz)
	s.registerSignIn(serveMux)
	mux := guarded{s: s, mux: serveMux}
	mux.HandleFunc("GET /api/catalog", s.getCatalog)

	mux.HandleFunc("GET /api/boards", s.listBoards)
	mux.HandleFunc("POST /api/boards", s.createBoard)
	mux.HandleFunc("GET /api/boards/{id}", s.getBoard)
	mux.HandleFunc("PUT /api/boards/{id}", s.updateBoard)
	mux.HandleFunc("DELETE /api/boards/{id}", s.deleteBoard)
	mux.HandleFunc("GET /api/boards/{id}/quests", s.listQuests)
	mux.HandleFunc("POST /api/boards/{id}/quests", s.createQuest)

	mux.HandleFunc("GET /api/quests/{id}", s.getQuest)
	mux.HandleFunc("PUT /api/quests/{id}", s.updateQuest)
	mux.HandleFunc("DELETE /api/quests/{id}", s.deleteQuest)

	s.registerTracker(mux)
	s.registerPages(mux)
	s.registerTrackerPages(mux)
	s.registerMonsterPages(mux)
	s.registerClassPages(mux)
	s.registerInventoryPages(mux)
	s.registerCampaignMonsterPages(mux)
	s.registerScript(mux)
	s.registerLoot(mux)
	s.registerAudio(mux)
	s.registerChapterPages(mux)
	s.registerLobby(mux)
	s.registerMembers(mux)
	if s.assetsDir != "" {
		// The board art: HeroQuest material, for members only when hosted.
		mux.HandleFunc("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(s.assetsDir))).ServeHTTP)
	}
	s.registerDeletes(mux)
}

// SetAssetsDir serves the board art from dir under /assets/ (through the
// guard, so only members see it when sign-in is on).
func (s *Server) SetAssetsDir(dir string) { s.assetsDir = dir }

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("app: encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	writeJSON(w, status, errorResponse{Error: fmt.Sprintf(format, args...)})
}

// writeStoreError maps store errors to HTTP statuses.
func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrInUse):
		writeError(w, http.StatusConflict, "still in use")
	default:
		log.Printf("app: store error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return false
	}
	return true
}

// cleanName trims a user-supplied name and checks its length.
func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("name is required")
	}
	if len(name) > maxNameLength {
		return "", fmt.Errorf("name must be at most %d characters", maxNameLength)
	}
	return name, nil
}
