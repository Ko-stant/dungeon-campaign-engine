package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// CampaignResponse is a campaign with its heroes.
type CampaignResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Heroes    []tracker.CampaignHero `json:"heroes"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// SessionSummaryResponse is a session list entry.
type SessionSummaryResponse struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaignId"`
	QuestID    *string   `json:"questId"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	EventSeq   int64     `json:"eventSeq"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// SessionResponse is a session with its full state.
type SessionResponse struct {
	ID         string        `json:"id"`
	CampaignID string        `json:"campaignId"`
	Name       string        `json:"name"`
	Status     string        `json:"status"`
	State      tracker.State `json:"state"`
	EventSeq   int64         `json:"eventSeq"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

// EventResponse is one logged change.
type EventResponse struct {
	Seq       int64           `json:"seq"`
	Round     int             `json:"round"`
	Kind      string          `json:"kind"`
	Summary   string          `json:"summary"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"createdAt"`
}

// CommandResponse is the result of a change: the new state and its event. It
// is also what the session stream pushes to every open tab.
type CommandResponse struct {
	State    tracker.State `json:"state"`
	Event    EventResponse `json:"event"`
	EventSeq int64         `json:"eventSeq"`
}

func (s *Server) registerTracker(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/campaigns", s.listCampaigns)
	mux.HandleFunc("POST /api/campaigns", s.createCampaign)
	mux.HandleFunc("GET /api/campaigns/{id}", s.getCampaign)
	mux.HandleFunc("PUT /api/campaigns/{id}", s.updateCampaign)
	mux.HandleFunc("GET /api/campaigns/{id}/sessions", s.listSessions)
	mux.HandleFunc("POST /api/campaigns/{id}/sessions", s.startSession)

	mux.HandleFunc("GET /api/sessions/{id}", s.getSession)
	mux.HandleFunc("POST /api/sessions/{id}/commands", s.sessionCommand)
	mux.HandleFunc("POST /api/sessions/{id}/travel", s.sessionTravel)
	mux.HandleFunc("GET /api/campaigns/{id}/chapters", s.listChapters)
	mux.HandleFunc("GET /api/sessions/{id}/events", s.sessionEvents)
	mux.HandleFunc("POST /api/sessions/{id}/complete", s.completeSession)
	mux.HandleFunc("POST /api/sessions/{id}/reopen", s.reopenSession)
	mux.HandleFunc("GET /api/sessions/{id}/stream", s.sessionStream)
}

// --- Campaigns ---

func campaignResponse(c store.Campaign) (CampaignResponse, error) {
	resp := CampaignResponse{ID: c.ID, Name: c.Name, UpdatedAt: c.UpdatedAt}
	if err := json.Unmarshal(c.Heroes, &resp.Heroes); err != nil {
		return resp, err
	}
	if resp.Heroes == nil {
		resp.Heroes = []tracker.CampaignHero{}
	}
	return resp, nil
}

func (s *Server) listCampaigns(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListCampaigns(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	out := make([]CampaignResponse, 0, len(list))
	for _, c := range list {
		resp, err := campaignResponse(c)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		out = append(out, resp)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createCampaign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	c, err := s.newCampaign(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	resp, err := campaignResponse(c)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) newCampaign(ctx context.Context, name string) (store.Campaign, error) {
	name, err := cleanName(name)
	if err != nil {
		return store.Campaign{}, err
	}
	return s.store.CreateCampaign(ctx, name, json.RawMessage(`[]`))
}

func (s *Server) getCampaign(w http.ResponseWriter, r *http.Request) {
	c, err := s.store.GetCampaign(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	resp, err := campaignResponse(c)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// normalizeHeroes validates heroes against the catalog and assigns ids.
func (s *Server) normalizeHeroes(heroes []tracker.CampaignHero) ([]tracker.CampaignHero, error) {
	out := make([]tracker.CampaignHero, 0, len(heroes))
	used := map[string]bool{}
	highest := 0
	for _, h := range heroes {
		if n, err := strconv.Atoi(strings.TrimPrefix(h.ID, "hero-")); err == nil && strings.HasPrefix(h.ID, "hero-") && n > highest {
			highest = n
		}
	}
	for _, h := range heroes {
		h.Name = strings.TrimSpace(h.Name)
		h.Player = strings.TrimSpace(h.Player)
		if h.Name == "" {
			return nil, errors.New("every hero needs a name")
		}
		if _, ok := s.catalog.Hero(h.Class); !ok {
			return nil, fmt.Errorf("hero %q has unknown class %q", h.Name, h.Class)
		}
		if h.Gold < 0 {
			return nil, fmt.Errorf("hero %q: gold must not be negative", h.Name)
		}
		if h.ID == "" || used[h.ID] {
			highest++
			h.ID = fmt.Sprintf("hero-%d", highest)
		}
		used[h.ID] = true
		out = append(out, h)
	}
	return out, nil
}

func (s *Server) saveCampaignHeroes(ctx context.Context, id, name string, heroes []tracker.CampaignHero) (store.Campaign, error) {
	data, err := json.Marshal(heroes)
	if err != nil {
		return store.Campaign{}, err
	}
	return s.store.UpdateCampaign(ctx, id, name, data)
}

func (s *Server) updateCampaign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string                 `json:"name"`
		Heroes []tracker.CampaignHero `json:"heroes"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	name, err := cleanName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	heroes, err := s.normalizeHeroes(req.Heroes)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	c, err := s.saveCampaignHeroes(r.Context(), r.PathValue("id"), name, heroes)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	resp, err := campaignResponse(c)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Sessions ---

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	if _, err := s.store.GetCampaign(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	list, err := s.store.ListSessions(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	out := make([]SessionSummaryResponse, 0, len(list))
	for _, ss := range list {
		out = append(out, SessionSummaryResponse{ID: ss.ID, CampaignID: ss.CampaignID, QuestID: ss.QuestID, Name: ss.Name, Status: ss.Status, EventSeq: ss.EventSeq, UpdatedAt: ss.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

// errBadInput marks errors caused by the request rather than the server.
var errBadInput = errors.New("bad input")

// newSession creates a session for a campaign and quest and logs its start.
func (s *Server) newSession(ctx context.Context, campaignID, questID, name string) (store.Session, error) {
	camp, err := s.store.GetCampaign(ctx, campaignID)
	if err != nil {
		return store.Session{}, err
	}
	var heroes []tracker.CampaignHero
	if err := json.Unmarshal(camp.Heroes, &heroes); err != nil {
		return store.Session{}, err
	}
	questRec, err := s.store.GetQuest(ctx, questID)
	if err != nil {
		return store.Session{}, err
	}
	var quest maps.Quest
	if err := json.Unmarshal(questRec.Doc, &quest); err != nil {
		return store.Session{}, err
	}
	_, board, err := s.loadBoard(ctx, questRec.BoardID)
	if err != nil {
		return store.Session{}, err
	}
	cat, err := s.catalogFor(ctx)
	if err != nil {
		return store.Session{}, err
	}
	state, err := tracker.NewSession(board, &quest, questRec.Name, heroes, cat)
	if err != nil {
		return store.Session{}, fmt.Errorf("%w: %v", errBadInput, err)
	}
	state.QuestID = questRec.ID

	name = strings.TrimSpace(name)
	if name == "" {
		name = questRec.Name
	}
	if len(name) > maxNameLength {
		return store.Session{}, fmt.Errorf("%w: name must be at most %d characters", errBadInput, maxNameLength)
	}
	data, err := json.Marshal(state)
	if err != nil {
		return store.Session{}, err
	}
	sess, err := s.store.CreateSession(ctx, campaignID, questID, name, data)
	if err != nil {
		return store.Session{}, err
	}

	names := make([]string, 0, len(state.Heroes))
	for _, h := range state.Heroes {
		names = append(names, h.Name)
	}
	summary := fmt.Sprintf("Started %s", questRec.Name)
	if len(names) > 0 {
		summary += " with " + strings.Join(names, ", ")
	}
	if _, err := s.store.RecordEvent(ctx, sess.ID, data, store.NewEvent{Round: state.Round, Kind: "session.start", Summary: summary}); err != nil {
		return store.Session{}, err
	}
	return s.store.GetSession(ctx, sess.ID)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		QuestID string `json:"questId"`
		Name    string `json:"name"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	sess, err := s.newSession(r.Context(), r.PathValue("id"), req.QuestID, req.Name)
	if errors.Is(err, errBadInput) {
		writeError(w, http.StatusBadRequest, "%v", errors.Unwrap(err))
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	resp, err := sessionResponse(sess)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func sessionResponse(ss store.Session) (SessionResponse, error) {
	resp := SessionResponse{ID: ss.ID, CampaignID: ss.CampaignID, Name: ss.Name, Status: ss.Status, EventSeq: ss.EventSeq, UpdatedAt: ss.UpdatedAt}
	err := json.Unmarshal(ss.State, &resp.State)
	return resp, err
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	ss, err := s.store.GetSession(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	resp, err := sessionResponse(ss)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func eventResponse(e store.Event) EventResponse {
	return EventResponse{Seq: e.Seq, Round: e.Round, Kind: e.Kind, Summary: e.Summary, Payload: e.Payload, CreatedAt: e.CreatedAt}
}

// lockSession serialises changes to one session.
func (s *Server) lockSession(id string) func() {
	v, _ := s.sessionLocks.LoadOrStore(id, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// record saves a new state with its event and pushes the result to open tabs.
func (s *Server) record(ctx context.Context, sessionID string, state *tracker.State, ev store.NewEvent) (CommandResponse, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return CommandResponse{}, err
	}
	saved, err := s.store.RecordEvent(ctx, sessionID, data, ev)
	if err != nil {
		return CommandResponse{}, err
	}
	resp := CommandResponse{State: *state, Event: eventResponse(saved), EventSeq: saved.Seq}
	s.streams.broadcast(sessionID, resp)
	return resp, nil
}

func (s *Server) loadSessionState(ctx context.Context, id string) (store.Session, *tracker.State, error) {
	ss, err := s.store.GetSession(ctx, id)
	if err != nil {
		return store.Session{}, nil, err
	}
	var state tracker.State
	if err := json.Unmarshal(ss.State, &state); err != nil {
		return store.Session{}, nil, err
	}
	if state.QuestID == "" && ss.QuestID != nil {
		// Sessions started before travel between maps existed.
		state.QuestID = *ss.QuestID
	}
	return ss, &state, nil
}

func (s *Server) sessionCommand(w http.ResponseWriter, r *http.Request) {
	var cmd tracker.Command
	if !readJSON(w, r, &cmd) {
		return
	}
	id := r.PathValue("id")
	unlock := s.lockSession(id)
	defer unlock()

	ss, state, err := s.loadSessionState(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if ss.Status != store.StatusActive {
		writeError(w, http.StatusConflict, "this session is completed; reopen it to make changes")
		return
	}
	cat, err := s.catalogFor(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	next, ev, err := tracker.Apply(state, cmd, cat)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	resp, err := s.record(r.Context(), id, next, store.NewEvent{Round: ev.Round, Kind: ev.Kind, Summary: ev.Summary, Payload: ev.Payload})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) sessionEvents(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if _, err := s.store.GetSession(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	events, err := s.store.ListEvents(r.Context(), r.PathValue("id"), after, limit)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	out := make([]EventResponse, 0, len(events))
	for _, e := range events {
		out = append(out, eventResponse(e))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) completeSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	unlock := s.lockSession(id)
	defer unlock()

	ss, state, err := s.loadSessionState(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	camp, err := s.store.GetCampaign(r.Context(), ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	var heroes []tracker.CampaignHero
	if err := json.Unmarshal(camp.Heroes, &heroes); err != nil {
		writeStoreError(w, err)
		return
	}
	if _, err := s.saveCampaignHeroes(r.Context(), camp.ID, camp.Name, state.CarryOver(heroes)); err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.store.SetSessionStatus(r.Context(), id, store.StatusCompleted); err != nil {
		writeStoreError(w, err)
		return
	}
	resp, err := s.record(r.Context(), id, state, store.NewEvent{Round: state.Round, Kind: "session.complete", Summary: "Quest completed; gold, equipment and notes saved to the campaign"})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) reopenSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	unlock := s.lockSession(id)
	defer unlock()

	_, state, err := s.loadSessionState(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.store.SetSessionStatus(r.Context(), id, store.StatusActive); err != nil {
		writeStoreError(w, err)
		return
	}
	resp, err := s.record(r.Context(), id, state, store.NewEvent{Round: state.Round, Kind: "session.reopen", Summary: "Session reopened"})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Live stream ---

// streams fans session changes out to connected WebSocket clients.
type streams struct {
	mu    sync.Mutex
	conns map[string]map[*websocket.Conn]struct{}
}

func (st *streams) add(id string, c *websocket.Conn) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.conns == nil {
		st.conns = map[string]map[*websocket.Conn]struct{}{}
	}
	if st.conns[id] == nil {
		st.conns[id] = map[*websocket.Conn]struct{}{}
	}
	st.conns[id][c] = struct{}{}
}

func (st *streams) remove(id string, c *websocket.Conn) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.conns[id], c)
	if len(st.conns[id]) == 0 {
		delete(st.conns, id)
	}
}

func (st *streams) broadcast(id string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("app: stream encode: %v", err)
		return
	}
	st.mu.Lock()
	targets := make([]*websocket.Conn, 0, len(st.conns[id]))
	for c := range st.conns[id] {
		targets = append(targets, c)
	}
	st.mu.Unlock()

	for _, c := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := c.Write(ctx, websocket.MessageText, data); err != nil {
			st.remove(id, c)
			_ = c.CloseNow()
		}
		cancel()
	}
}

func (s *Server) sessionStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.store.GetSession(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	s.streams.add(id, conn)
	defer s.streams.remove(id, conn)
	// Clients only listen; read until they go away so close frames are handled.
	for {
		if _, _, err := conn.Read(r.Context()); err != nil {
			return
		}
	}
}
