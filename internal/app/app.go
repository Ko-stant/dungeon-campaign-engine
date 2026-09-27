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

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// maxBodyBytes bounds request bodies; a 200x200 board is well under 1 MB.
const maxBodyBytes = 5 << 20

const maxNameLength = 120

// Server holds the app's dependencies.
type Server struct {
	store   *store.Store
	catalog *content.Catalog
}

// New creates the app server.
func New(st *store.Store, catalog *content.Catalog) *Server {
	return &Server{store: st, catalog: catalog}
}

// Register mounts the app's routes.
func (s *Server) Register(mux *http.ServeMux) {
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

	s.registerPages(mux)
}

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

func (s *Server) getCatalog(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalog)
}
