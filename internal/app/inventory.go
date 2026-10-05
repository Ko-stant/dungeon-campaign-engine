package app

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// Campaign inventory forms: the party's gold and each hero's items between
// quests. During a quest the tracker's item and gold commands change the
// session's copy, which is saved back to the campaign when the quest is
// completed.
func (s *Server) registerInventoryPages(mux routeMux) {
	mux.HandleFunc("POST /campaigns/{id}/gold", s.partyGoldForm)
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/items", s.addItemForm)
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/items/{itemId}", s.updateItemForm)
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/items/{itemId}/equip", s.equipItemForm)
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/items/{itemId}/delete", s.deleteItemForm)
}

func (s *Server) partyGoldForm(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	c, err := s.store.GetCampaign(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	gold, err := tracker.GoldChange(c.Gold, r.PostFormValue("gold"))
	if err != nil {
		s.renderCampaignPage(w, r, http.StatusBadRequest, id, err.Error())
		return
	}
	if err := s.store.SetCampaignGold(r.Context(), id, gold); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id+"#inventory", http.StatusSeeOther)
}

// formItemStats reads an item's kind and stats; blank stats are 0.
func formItemStats(r *http.Request) (string, tracker.ItemStats, error) {
	var st tracker.ItemStats
	for _, f := range []struct {
		name, label string
		into        *int
	}{
		{"damage", "Damage", &st.Damage}, {"accuracy", "Accuracy", &st.Accuracy}, {"avoidance", "Avoidance", &st.Avoidance},
		{"mitigation", "Mitigation", &st.Mitigation}, {"mana", "Mana", &st.Mana}, {"mana_regen", "Mana regen", &st.ManaRegen},
	} {
		n, err := formInt(f.label, r.PostFormValue(f.name), -tracker.MaxItemStat, tracker.MaxItemStat, 0)
		if err != nil {
			return "", st, err
		}
		*f.into = n
	}
	return r.PostFormValue("kind"), st, nil
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

func (s *Server) addItemForm(w http.ResponseWriter, r *http.Request) {
	s.changeHero(w, r, func(h *tracker.CampaignHero) (bool, error) {
		qty, err := formQuantity(r.PostFormValue("quantity"))
		if err != nil {
			return false, err
		}
		kind, stats, err := formItemStats(r)
		if err != nil {
			return false, err
		}
		items, _, err := tracker.AddItem(h.Items, tracker.Item{
			Name: r.PostFormValue("name"), Quantity: qty, Notes: r.PostFormValue("notes"),
			Kind: kind, Equipped: r.PostFormValue("equipped") != "", ItemStats: stats,
		})
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
		kind, stats, err := formItemStats(r)
		if err != nil {
			return false, err
		}
		name, notes := r.PostFormValue("name"), r.PostFormValue("notes")
		items, _, err := tracker.UpdateItem(h.Items, itemID, tracker.ItemPatch{Name: &name, Quantity: &qty, Notes: &notes, Kind: &kind, Stats: &stats})
		if err != nil {
			return false, err
		}
		h.Items = items
		return false, nil
	})
}

func (s *Server) equipItemForm(w http.ResponseWriter, r *http.Request) {
	s.changeHero(w, r, func(h *tracker.CampaignHero) (bool, error) {
		i := slices.IndexFunc(h.Items, func(it tracker.Item) bool { return it.ID == r.PathValue("itemId") })
		if i < 0 {
			return true, nil
		}
		on, err := strconv.ParseBool(r.PostFormValue("equipped"))
		if err != nil {
			return false, errors.New("equipped must be true or false")
		}
		h.Items[i].Equipped = on
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
