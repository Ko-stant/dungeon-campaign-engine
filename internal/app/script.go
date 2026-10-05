package app

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/script"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// ScriptResponse is a campaign's parsed read-aloud script.
type ScriptResponse struct {
	Sections []script.Section `json:"sections"`
}

func (s *Server) registerScript(mux routeMux) {
	mux.HandleFunc("GET /api/campaigns/{id}/script", s.getScript)
	mux.HandleFunc("POST /campaigns/{id}/script", s.saveScriptForm)
}

// loadScript returns a campaign's script text and its parsed form. A stored
// script always parsed when it was saved.
func (s *Server) loadScript(r *http.Request, id string) (string, script.Script, error) {
	text, err := s.store.GetCampaignScript(r.Context(), id)
	if err != nil {
		return "", script.Script{}, err
	}
	sc, err := script.Parse(text)
	return text, sc, err
}

func (s *Server) getScript(w http.ResponseWriter, r *http.Request) {
	_, sc, err := s.loadScript(r, r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ScriptResponse{Sections: sc.Sections})
}

// saveScriptForm replaces the campaign's script. A script that does not
// parse is not saved; the page shows the error with the text still in the box.
func (s *Server) saveScriptForm(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	if _, err := s.store.GetCampaign(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		writeStoreError(w, err)
		return
	}
	text := strings.ReplaceAll(r.PostFormValue("script"), "\r\n", "\n")
	if strings.TrimSpace(text) == "" {
		text = ""
	}
	if _, err := script.Parse(text); err != nil {
		d, derr := s.campaignPageData(r.Context(), id)
		if derr != nil {
			writeStoreError(w, derr)
			return
		}
		d.ScriptText, d.ScriptError = text, "Script not saved: "+err.Error()
		render(w, r, http.StatusBadRequest, views.CampaignPage(d))
		return
	}
	if err := s.store.SetCampaignScript(r.Context(), id, text); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id+"#script", http.StatusSeeOther)
}
