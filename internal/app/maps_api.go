package app

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// BoardSummary is a board list entry.
type BoardSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BoardResponse is a board with its layout document.
type BoardResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Board     maps.Board `json:"board"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// QuestSummary is a quest list entry.
type QuestSummary struct {
	ID        string    `json:"id"`
	BoardID   string    `json:"boardId"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// QuestResponse is a quest with its layers and advisory placement issues.
type QuestResponse struct {
	ID        string       `json:"id"`
	BoardID   string       `json:"boardId"`
	Name      string       `json:"name"`
	Quest     maps.Quest   `json:"quest"`
	Issues    []maps.Issue `json:"issues"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

func boardResponse(b store.Board) (BoardResponse, error) {
	resp := BoardResponse{ID: b.ID, Name: b.Name, UpdatedAt: b.UpdatedAt}
	err := json.Unmarshal(b.Doc, &resp.Board)
	return resp, err
}

// loadBoard fetches and decodes a board document.
func (s *Server) loadBoard(ctx context.Context, id string) (store.Board, *maps.Board, error) {
	rec, err := s.store.GetBoard(ctx, id)
	if err != nil {
		return store.Board{}, nil, err
	}
	var b maps.Board
	if err := json.Unmarshal(rec.Doc, &b); err != nil {
		return store.Board{}, nil, err
	}
	return rec, &b, nil
}

func (s *Server) listBoards(w http.ResponseWriter, r *http.Request) {
	boards, err := s.store.ListBoards(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	out := make([]BoardSummary, 0, len(boards))
	for _, b := range boards {
		out = append(out, BoardSummary{ID: b.ID, Name: b.Name, Width: b.Width, Height: b.Height, UpdatedAt: b.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createBoard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	created, err := s.newBoard(r.Context(), req.Name, req.Width, req.Height)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	resp, err := boardResponse(created)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// newBoard validates and stores a blank board. Errors are the caller's fault
// unless they wrap a store error.
func (s *Server) newBoard(ctx context.Context, name string, width, height int) (store.Board, error) {
	name, err := cleanName(name)
	if err != nil {
		return store.Board{}, err
	}
	board, err := maps.NewBoard(width, height)
	if err != nil {
		return store.Board{}, err
	}
	doc, err := json.Marshal(board)
	if err != nil {
		return store.Board{}, err
	}
	return s.store.CreateBoard(ctx, name, board.Width, board.Height, doc)
}

func (s *Server) getBoard(w http.ResponseWriter, r *http.Request) {
	rec, err := s.store.GetBoard(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	resp, err := boardResponse(rec)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) updateBoard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string     `json:"name"`
		Board maps.Board `json:"board"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	name, err := cleanName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	req.Board.Version = maps.CurrentVersion
	if req.Board.Rooms == nil {
		req.Board.Rooms = []maps.Room{}
	}
	if err := req.Board.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	doc, err := json.Marshal(req.Board)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	rec, err := s.store.UpdateBoard(r.Context(), r.PathValue("id"), name, req.Board.Width, req.Board.Height, doc)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	resp, err := boardResponse(rec)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) deleteBoard(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteBoard(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listQuests(w http.ResponseWriter, r *http.Request) {
	boardID := r.PathValue("id")
	if _, err := s.store.GetBoard(r.Context(), boardID); err != nil {
		writeStoreError(w, err)
		return
	}
	quests, err := s.store.ListQuests(r.Context(), boardID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	out := make([]QuestSummary, 0, len(quests))
	for _, q := range quests {
		out = append(out, QuestSummary{ID: q.ID, BoardID: q.BoardID, Name: q.Name, UpdatedAt: q.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createQuest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	name, err := cleanName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	_, board, err := s.loadBoard(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	quest := maps.NewQuest(board)
	doc, err := json.Marshal(quest)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	rec, err := s.store.CreateQuest(r.Context(), r.PathValue("id"), name, doc)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.questResponse(rec, quest, board))
}

func (s *Server) questResponse(rec store.Quest, quest *maps.Quest, board *maps.Board) QuestResponse {
	issues := quest.Check(board, s.catalog.FurnitureSize, s.catalog.TrapSize)
	if issues == nil {
		issues = []maps.Issue{}
	}
	return QuestResponse{ID: rec.ID, BoardID: rec.BoardID, Name: rec.Name, Quest: *quest, Issues: issues, UpdatedAt: rec.UpdatedAt}
}

func (s *Server) getQuest(w http.ResponseWriter, r *http.Request) {
	rec, err := s.store.GetQuest(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	var quest maps.Quest
	if err := json.Unmarshal(rec.Doc, &quest); err != nil {
		writeStoreError(w, err)
		return
	}
	_, board, err := s.loadBoard(r.Context(), rec.BoardID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.questResponse(rec, &quest, board))
}

func (s *Server) updateQuest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string     `json:"name"`
		Quest maps.Quest `json:"quest"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	name, err := cleanName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if err := req.Quest.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	existing, err := s.store.GetQuest(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	_, board, err := s.loadBoard(r.Context(), existing.BoardID)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	// Saving a quest while looking at the current board accepts that board.
	quest := req.Quest
	quest.Version = maps.CurrentVersion
	quest.BoardChecksum = board.Checksum()
	normalizeQuest(&quest)

	doc, err := json.Marshal(quest)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	rec, err := s.store.UpdateQuest(r.Context(), existing.ID, name, doc)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.questResponse(rec, &quest, board))
}

func (s *Server) deleteQuest(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteQuest(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// normalizeQuest replaces nil layers with empty ones so they encode as [].
func normalizeQuest(q *maps.Quest) {
	if q.Doors == nil {
		q.Doors = []maps.Door{}
	}
	if q.BlockedSquares == nil {
		q.BlockedSquares = []maps.Rect{}
	}
	if q.Furniture == nil {
		q.Furniture = []maps.Furniture{}
	}
	if q.Monsters == nil {
		q.Monsters = []maps.Monster{}
	}
	if q.Traps == nil {
		q.Traps = []maps.Trap{}
	}
	if q.Notes == nil {
		q.Notes = []maps.Note{}
	}
	if q.StartTiles == nil {
		q.StartTiles = []maps.Tile{}
	}
}
