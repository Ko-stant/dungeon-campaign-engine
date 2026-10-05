package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// Chapter statuses shown on the campaign page.
const (
	chapterNotStarted = "not started"
	chapterInProgress = "in progress"
	chapterDone       = "done"
)

func (s *Server) registerChapterPages(mux *http.ServeMux) {
	mux.HandleFunc("POST /campaigns/{id}/chapters", s.addChapterForm)
	mux.HandleFunc("POST /campaigns/{id}/chapters/{questId}/{action}", s.chapterActionForm)
	mux.HandleFunc("POST /campaigns/{id}/maps", s.newCampaignMapForm)
}

// sessionQuests returns the quests a session has played: its starting quest
// plus any maps it traveled to.
func sessionQuests(ss store.SessionSummary) []string {
	if len(ss.VisitedQuestIDs) > 0 {
		return ss.VisitedQuestIDs
	}
	if ss.QuestID != nil {
		return []string{*ss.QuestID}
	}
	return nil
}

// chapterStatus is "in progress" (with the session to resume) when an active
// session has played the quest, "done" (with the latest completed session)
// when only completed ones have, and "not started" otherwise. Sessions come
// active first, then most recent first.
func chapterStatus(questID string, sessions []store.SessionSummary) (string, string) {
	for _, ss := range sessions {
		if slices.Contains(sessionQuests(ss), questID) {
			if ss.Status == store.StatusActive {
				return chapterInProgress, ss.ID
			}
			return chapterDone, ss.ID
		}
	}
	return chapterNotStarted, ""
}

func chapterIDs(chapters []store.Chapter) []string {
	ids := make([]string, 0, len(chapters))
	for _, ch := range chapters {
		ids = append(ids, ch.QuestID)
	}
	return ids
}

// addChapterPageData fills the chapters, the add-chapter options and the
// start-quest options (chapters first, the next unplayed one preselected).
func (s *Server) addChapterPageData(ctx context.Context, d *views.CampaignPageData, sessions []store.SessionSummary) error {
	chapters, err := s.store.ListChapters(ctx, d.ID)
	if err != nil {
		return err
	}
	boards, err := s.store.ListBoards(ctx)
	if err != nil {
		return err
	}
	boardNames := make(map[string]string, len(boards))
	for _, b := range boards {
		boardNames[b.ID] = b.Name
	}
	quests, err := s.store.ListQuests(ctx, "")
	if err != nil {
		return err
	}

	next := ""
	for i, ch := range chapters {
		status, sessionID := chapterStatus(ch.QuestID, sessions)
		if next == "" && status == chapterNotStarted {
			next = ch.QuestID
		}
		d.Chapters = append(d.Chapters, views.ChapterRow{
			Number: i + 1, QuestID: ch.QuestID, QuestName: ch.QuestName, BoardID: ch.BoardID, BoardName: ch.BoardName,
			Status: status, SessionID: sessionID, First: i == 0, Last: i == len(chapters)-1,
		})
		d.Quests = append(d.Quests, views.QuestOption{ID: ch.QuestID, Label: "Chapter " + strconv.Itoa(i+1) + ": " + ch.QuestName + " — " + ch.BoardName})
	}
	inCampaign := chapterIDs(chapters)
	for _, q := range quests {
		if slices.Contains(inCampaign, q.ID) {
			continue
		}
		opt := views.QuestOption{ID: q.ID, Label: q.Name + " — " + boardNames[q.BoardID]}
		d.Quests = append(d.Quests, opt)
		d.OtherQuests = append(d.OtherQuests, opt)
	}
	for i := range d.Quests {
		d.Quests[i].Selected = d.Quests[i].ID == next
	}
	return nil
}

func (s *Server) addChapterForm(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	chapters, err := s.store.ListChapters(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	questID := r.PostFormValue("questId")
	ids := chapterIDs(chapters)
	if !slices.Contains(ids, questID) {
		ids = append(ids, questID)
	}
	if err := s.store.SetChapters(r.Context(), id, ids); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.renderCampaignPage(w, r, http.StatusBadRequest, id, "That quest no longer exists.")
			return
		}
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

// chapterActionForm moves a chapter up or down, or removes it (the quest
// itself is kept).
func (s *Server) chapterActionForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	chapters, err := s.store.ListChapters(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	ids := chapterIDs(chapters)
	i := slices.Index(ids, r.PathValue("questId"))
	if i < 0 {
		http.NotFound(w, r)
		return
	}
	switch r.PathValue("action") {
	case "up":
		if i > 0 {
			ids[i-1], ids[i] = ids[i], ids[i-1]
		}
	case "down":
		if i < len(ids)-1 {
			ids[i+1], ids[i] = ids[i], ids[i+1]
		}
	case "remove":
		ids = slices.Delete(ids, i, i+1)
	default:
		http.NotFound(w, r)
		return
	}
	if err := s.store.SetChapters(r.Context(), id, ids); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

// newCampaignMapForm creates a board and a quest of the same name, adds the
// quest as the campaign's next chapter and opens it in the editor.
func (s *Server) newCampaignMapForm(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	chapters, err := s.store.ListChapters(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if _, err := s.store.GetCampaign(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	width, errW := strconv.Atoi(r.PostFormValue("width"))
	height, errH := strconv.Atoi(r.PostFormValue("height"))
	if errW != nil || errH != nil {
		s.renderCampaignPage(w, r, http.StatusBadRequest, id, "Columns and rows must be whole numbers.")
		return
	}
	boardRec, err := s.newBoard(r.Context(), r.PostFormValue("name"), width, height)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrInUse) {
			writeStoreError(w, err)
			return
		}
		s.renderCampaignPage(w, r, http.StatusBadRequest, id, err.Error())
		return
	}
	var board maps.Board
	if err := json.Unmarshal(boardRec.Doc, &board); err != nil {
		writeStoreError(w, err)
		return
	}
	doc, err := json.Marshal(maps.NewQuest(&board))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	quest, err := s.store.CreateQuest(r.Context(), boardRec.ID, boardRec.Name, doc)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.store.SetChapters(r.Context(), id, append(chapterIDs(chapters), quest.ID)); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/maps/"+boardRec.ID+"/edit?quest="+url.QueryEscape(quest.ID), http.StatusSeeOther)
}

// campaignMapGroups lists each campaign's chapters for the maps page.
func (s *Server) campaignMapGroups(ctx context.Context) ([]views.CampaignMapGroup, error) {
	all, err := s.store.ListAllChapters(ctx)
	if err != nil {
		return nil, err
	}
	var groups []views.CampaignMapGroup
	for _, ch := range all {
		if len(groups) == 0 || groups[len(groups)-1].CampaignID != ch.CampaignID {
			groups = append(groups, views.CampaignMapGroup{CampaignID: ch.CampaignID, Name: ch.CampaignName})
		}
		g := &groups[len(groups)-1]
		g.Chapters = append(g.Chapters, views.ChapterRow{
			Number: len(g.Chapters) + 1, QuestID: ch.QuestID, QuestName: ch.QuestName, BoardID: ch.BoardID, BoardName: ch.BoardName,
		})
	}
	return groups, nil
}

// ChapterResponse is one chapter of a campaign, for the tracker's travel menu.
type ChapterResponse struct {
	Number    int    `json:"number"`
	QuestID   string `json:"questId"`
	QuestName string `json:"questName"`
	BoardID   string `json:"boardId"`
	BoardName string `json:"boardName"`
}

func (s *Server) listChapters(w http.ResponseWriter, r *http.Request) {
	if _, err := s.store.GetCampaign(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	chapters, err := s.store.ListChapters(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	out := make([]ChapterResponse, 0, len(chapters))
	for i, ch := range chapters {
		out = append(out, ChapterResponse{Number: i + 1, QuestID: ch.QuestID, QuestName: ch.QuestName, BoardID: ch.BoardID, BoardName: ch.BoardName})
	}
	writeJSON(w, http.StatusOK, out)
}

// sessionTravel moves a running session's party to another map (a quest on
// any board). A map visited before is restored as it was left; a new one is
// copied from the quest now, like a session start.
func (s *Server) sessionTravel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		QuestID string `json:"questId"`
	}
	if !readJSON(w, r, &req) {
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
	dest := tracker.Destination{QuestID: req.QuestID}
	if !state.Visited(req.QuestID) {
		questRec, err := s.store.GetQuest(r.Context(), req.QuestID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		var quest maps.Quest
		if err := json.Unmarshal(questRec.Doc, &quest); err != nil {
			writeStoreError(w, err)
			return
		}
		_, board, err := s.loadBoard(r.Context(), questRec.BoardID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		dest = tracker.Destination{QuestID: questRec.ID, QuestName: questRec.Name, Board: board, Quest: &quest}
	}
	cat, err := s.campaignCatalog(r.Context(), ss.CampaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	next, ev, err := tracker.Travel(state, dest, cat)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	resp, err := s.record(r.Context(), id, next, store.NewEvent{Round: ev.Round, Kind: ev.Kind, Summary: ev.Summary, Payload: ev.Payload, PlayerSummary: ev.PlayerSummary})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
