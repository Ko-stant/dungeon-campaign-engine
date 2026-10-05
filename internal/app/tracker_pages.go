package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/script"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

func (s *Server) registerTrackerPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /campaigns", s.campaignsPage)
	mux.HandleFunc("POST /campaigns", s.createCampaignForm)
	mux.HandleFunc("GET /campaigns/{id}", s.campaignPage)
	mux.HandleFunc("POST /campaigns/{id}/heroes", s.addHeroForm)
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/delete", s.removeHeroForm)
	mux.HandleFunc("POST /campaigns/{id}/sessions", s.startSessionForm)
	mux.HandleFunc("GET /play/{id}", s.playPage)
}

func sessionLinks(list []store.SessionSummary) []views.SessionLink {
	out := make([]views.SessionLink, 0, len(list))
	for _, ss := range list {
		out = append(out, views.SessionLink{ID: ss.ID, Name: ss.Name, Status: ss.Status, Events: ss.EventSeq, UpdatedAt: ss.UpdatedAt})
	}
	return out
}

func (s *Server) renderCampaignsPage(w http.ResponseWriter, r *http.Request, status int, formError string) {
	campaigns, err := s.store.ListCampaigns(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	items := make([]views.CampaignListItem, 0, len(campaigns))
	for _, c := range campaigns {
		var heroes []tracker.CampaignHero
		if err := json.Unmarshal(c.Heroes, &heroes); err != nil {
			writeStoreError(w, err)
			return
		}
		sessions, err := s.store.ListSessions(r.Context(), c.ID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		items = append(items, views.CampaignListItem{ID: c.ID, Name: c.Name, Heroes: len(heroes), Sessions: sessionLinks(sessions)})
	}
	render(w, r, status, views.CampaignsPage(items, formError))
}

func (s *Server) campaignsPage(w http.ResponseWriter, r *http.Request) {
	s.renderCampaignsPage(w, r, http.StatusOK, "")
}

func (s *Server) createCampaignForm(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	c, err := s.newCampaign(r.Context(), r.PostFormValue("name"))
	if err != nil {
		s.renderCampaignsPage(w, r, http.StatusBadRequest, err.Error())
		return
	}
	http.Redirect(w, r, "/campaigns/"+c.ID, http.StatusSeeOther)
}

func parseForm(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return false
	}
	return true
}

func (s *Server) loadCampaign(ctx context.Context, id string) (store.Campaign, []tracker.CampaignHero, error) {
	c, err := s.store.GetCampaign(ctx, id)
	if err != nil {
		return store.Campaign{}, nil, err
	}
	var heroes []tracker.CampaignHero
	if err := json.Unmarshal(c.Heroes, &heroes); err != nil {
		return store.Campaign{}, nil, err
	}
	return c, heroes, nil
}

func (s *Server) campaignPageData(ctx context.Context, id string) (views.CampaignPageData, error) {
	c, heroes, err := s.loadCampaign(ctx, id)
	if err != nil {
		return views.CampaignPageData{}, err
	}
	cat, err := s.catalogFor(ctx)
	if err != nil {
		return views.CampaignPageData{}, err
	}
	d := views.CampaignPageData{ID: c.ID, Name: c.Name, Gold: c.Gold}
	for _, h := range heroes {
		row := views.HeroRow{ID: h.ID, Name: h.Name, Player: h.Player, Class: h.Class, Equipment: h.Equipment, Notes: h.Notes}
		if def, ok := cat.Hero(h.Class); ok {
			row.Class = def.Name
			// The totals the hero would start a quest with.
			hero := tracker.Hero{Combat: tracker.ClassCombat(def), Items: h.Items}
			if tot := hero.CombatTotals(); tot != nil {
				row.Combat = tracker.CombatLine(*tot)
			}
		}
		for _, it := range h.Items {
			row.Items = append(row.Items, views.ItemRow{
				ID: it.ID, Name: it.Name, Quantity: it.Quantity, Notes: it.Notes, Kind: it.Kind, Equipped: it.Equipped,
				Damage: it.Damage, Accuracy: it.Accuracy, Avoidance: it.Avoidance, Mitigation: it.Mitigation, Mana: it.Mana, ManaRegen: it.ManaRegen,
				StatsLine: it.Summary(),
			})
		}
		d.Heroes = append(d.Heroes, row)
	}
	for _, def := range cat.Heroes {
		d.Classes = append(d.Classes, views.ClassOption{ID: def.ID, Name: def.Name})
	}

	stats, err := s.campaignMonsterStats(ctx, c.ID)
	if err != nil {
		return views.CampaignPageData{}, err
	}
	for _, m := range cat.Monsters {
		d.MonsterOptions = append(d.MonsterOptions, views.MonsterOption{ID: m.ID, Name: m.Name})
		if st, ok := stats[m.ID]; ok {
			d.MonsterStats = append(d.MonsterStats, views.MonsterStatsRow{Type: m.ID, Name: m.Name, Stats: st})
		}
	}

	text, err := s.store.GetCampaignScript(ctx, c.ID)
	if err != nil {
		return views.CampaignPageData{}, err
	}
	d.ScriptText = text
	titles := map[string]string{}
	if sc, err := script.Parse(text); err != nil {
		d.ScriptError = "The saved script no longer parses: " + err.Error()
	} else {
		d.ScriptSections, d.ScriptPassages = len(sc.Sections), sc.Count()
		for _, sec := range sc.Sections {
			for _, p := range sec.Passages {
				titles[p.ID] = p.Title
			}
		}
	}
	if err := s.addAudioPageData(&d, titles); err != nil {
		return views.CampaignPageData{}, err
	}

	sessions, err := s.store.ListSessions(ctx, c.ID)
	if err != nil {
		return views.CampaignPageData{}, err
	}
	d.Sessions = sessionLinks(sessions)
	if err := s.addChapterPageData(ctx, &d, sessions); err != nil {
		return views.CampaignPageData{}, err
	}
	return d, nil
}

func (s *Server) renderCampaignPage(w http.ResponseWriter, r *http.Request, status int, id, formError string) {
	d, err := s.campaignPageData(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	d.Error = formError
	render(w, r, status, views.CampaignPage(d))
}

func (s *Server) campaignPage(w http.ResponseWriter, r *http.Request) {
	s.renderCampaignPage(w, r, http.StatusOK, r.PathValue("id"), "")
}

func (s *Server) addHeroForm(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	c, heroes, err := s.loadCampaign(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	heroes = append(heroes, tracker.CampaignHero{Name: r.PostFormValue("name"), Player: r.PostFormValue("player"), Class: r.PostFormValue("class")})
	cat, err := s.catalogFor(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	heroes, err = normalizeHeroes(cat, heroes)
	if err != nil {
		s.renderCampaignPage(w, r, http.StatusBadRequest, id, err.Error())
		return
	}
	if _, err := s.saveCampaignHeroes(r.Context(), id, c.Name, heroes); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

func (s *Server) removeHeroForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, heroes, err := s.loadCampaign(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	kept := make([]tracker.CampaignHero, 0, len(heroes))
	for _, h := range heroes {
		if h.ID != r.PathValue("heroId") {
			kept = append(kept, h)
		}
	}
	if _, err := s.saveCampaignHeroes(r.Context(), id, c.Name, kept); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

func (s *Server) startSessionForm(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	sess, err := s.newSession(r.Context(), id, r.PostFormValue("questId"), r.PostFormValue("name"))
	if errors.Is(err, errBadInput) {
		s.renderCampaignPage(w, r, http.StatusBadRequest, id, errors.Unwrap(err).Error())
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		s.renderCampaignPage(w, r, http.StatusNotFound, id, "That quest no longer exists.")
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/play/"+sess.ID, http.StatusSeeOther)
}

func (s *Server) playPage(w http.ResponseWriter, r *http.Request) {
	ss, err := s.store.GetSession(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	render(w, r, http.StatusOK, views.PlayPage(ss.ID, ss.Name))
}
