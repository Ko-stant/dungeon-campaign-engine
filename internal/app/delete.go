package app

import (
	"context"
	"log"
	"net/http"
	"slices"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// Deletes from the pages. Each button asks Yes/No first (data-confirm, see
// internal/web/src/ui/confirm.ts) and names what goes with the item.
func (s *Server) registerDeletes(mux *http.ServeMux) {
	mux.HandleFunc("POST /campaigns/{id}/delete", s.deleteCampaignForm)
	mux.HandleFunc("POST /campaigns/{id}/sessions/{sessionId}/delete", s.deleteSessionForm)
	mux.HandleFunc("POST /maps/{id}/delete", s.deleteBoardForm)
}

// deleteCampaignForm deletes a campaign with its chapters, sessions and audio
// clips. The maps stay.
func (s *Server) deleteCampaignForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteCampaign(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	if s.audio != nil {
		// The campaign is gone either way; a folder left behind only costs disk.
		if err := s.audio.RemoveCampaign(id); err != nil {
			log.Printf("app: remove the audio clips of deleted campaign %s: %v", id, err)
		}
	}
	http.Redirect(w, r, "/campaigns", http.StatusSeeOther)
}

// deleteSessionForm deletes one of the campaign's sessions and its log.
func (s *Server) deleteSessionForm(w http.ResponseWriter, r *http.Request) {
	id, sessionID := r.PathValue("id"), r.PathValue("sessionId")
	if err := s.store.DeleteSession(r.Context(), id, sessionID); err != nil {
		writeStoreError(w, err)
		return
	}
	s.sessionLocks.Delete(sessionID)
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

// deleteBoardForm deletes a board with its quests (and so takes them out of
// any campaign's chapters). Sessions keep their frozen copy of the map.
func (s *Server) deleteBoardForm(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteBoardWithQuests(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/maps", http.StatusSeeOther)
}

// boardDeleteDetails fills in each board's quests and the campaigns that have
// one of them as a chapter, for the board's delete warning.
func (s *Server) boardDeleteDetails(ctx context.Context, items []views.MapListItem) error {
	chapters, err := s.store.ListAllChapters(ctx)
	if err != nil {
		return err
	}
	for i := range items {
		b := &items[i]
		quests, err := s.store.ListQuests(ctx, b.ID)
		if err != nil {
			return err
		}
		for _, q := range quests {
			b.Quests = append(b.Quests, q.Name)
		}
		for _, ch := range chapters {
			if ch.BoardID == b.ID && !slices.Contains(b.Campaigns, ch.CampaignName) {
				b.Campaigns = append(b.Campaigns, ch.CampaignName)
			}
		}
	}
	return nil
}
