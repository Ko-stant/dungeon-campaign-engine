package app

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// Campaign inventory forms: a hero's gold and items between quests. During a
// quest the tracker's item and gold commands change the session's copy,
// which is saved back to the campaign when the quest is completed.
func (s *Server) registerInventoryPages(mux *http.ServeMux) {
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/gold", s.heroGoldForm)
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/items", s.addItemForm)
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/items/{itemId}", s.updateItemForm)
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/items/{itemId}/delete", s.deleteItemForm)
}

// changeHero applies change to one campaign hero and saves the campaign.
// change returns an error to show on the page; notFound true means the item
// (or hero) does not exist.
func (s *Server) changeHero(w http.ResponseWriter, r *http.Request, change func(h *tracker.CampaignHero) (notFound bool, err error)) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	c, heroes, err := s.loadCampaign(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	i := slices.IndexFunc(heroes, func(h tracker.CampaignHero) bool { return h.ID == r.PathValue("heroId") })
	if i < 0 {
		http.NotFound(w, r)
		return
	}
	notFound, err := change(&heroes[i])
	if notFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.renderCampaignPage(w, r, http.StatusBadRequest, id, heroes[i].Name+": "+err.Error())
		return
	}
	if _, err := s.saveCampaignHeroes(r.Context(), id, c.Name, heroes); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id+"#inventory", http.StatusSeeOther)
}

func formQuantity(value string) (int, error) {
	return formInt("quantity", value, 0, tracker.MaxItemQuantity, 0)
}

func (s *Server) heroGoldForm(w http.ResponseWriter, r *http.Request) {
	s.changeHero(w, r, func(h *tracker.CampaignHero) (bool, error) {
		gold, err := tracker.GoldChange(h.Gold, r.PostFormValue("gold"))
		if err != nil {
			return false, err
		}
		h.Gold = gold
		return false, nil
	})
}

func (s *Server) addItemForm(w http.ResponseWriter, r *http.Request) {
	s.changeHero(w, r, func(h *tracker.CampaignHero) (bool, error) {
		qty, err := formQuantity(r.PostFormValue("quantity"))
		if err != nil {
			return false, err
		}
		items, _, err := tracker.AddItem(h.Items, r.PostFormValue("name"), qty, r.PostFormValue("notes"))
		if err != nil {
			return false, err
		}
		h.Items = items
		return false, nil
	})
}

func (s *Server) updateItemForm(w http.ResponseWriter, r *http.Request) {
	s.changeHero(w, r, func(h *tracker.CampaignHero) (bool, error) {
		itemID := r.PathValue("itemId")
		if !slices.ContainsFunc(h.Items, func(it tracker.Item) bool { return it.ID == itemID }) {
			return true, nil
		}
		qty, err := strconv.Atoi(strings.TrimSpace(r.PostFormValue("quantity")))
		if err != nil {
			qty = -1 // UpdateItem explains the allowed range
		}
		name, notes := r.PostFormValue("name"), r.PostFormValue("notes")
		items, _, err := tracker.UpdateItem(h.Items, itemID, &name, &qty, &notes)
		if err != nil {
			return false, err
		}
		h.Items = items
		return false, nil
	})
}

func (s *Server) deleteItemForm(w http.ResponseWriter, r *http.Request) {
	s.changeHero(w, r, func(h *tracker.CampaignHero) (bool, error) {
		items, _, _, err := tracker.RemoveItem(h.Items, r.PathValue("itemId"), 0)
		if err != nil {
			return true, nil
		}
		h.Items = items
		return false, nil
	})
}
