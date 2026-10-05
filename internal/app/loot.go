package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// The campaign's loot list: items with their kind and stats ready, for the
// GM to hand out during play (the tracker's loot picker). make fill-campaign
// adds the campaign's finds from combat.json.

func (s *Server) registerLoot(mux routeMux) {
	mux.HandleFunc("GET /api/campaigns/{id}/loot", s.getLoot)
	mux.HandleFunc("POST /campaigns/{id}/loot", s.saveLootForm)
	mux.HandleFunc("POST /campaigns/{id}/loot/{lootId}/delete", s.deleteLootForm)
}

func (s *Server) campaignLoot(ctx context.Context, campaignID string) ([]tracker.Item, error) {
	raw, err := s.store.GetCampaignLoot(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	loot := []tracker.Item{}
	if err := json.Unmarshal(raw, &loot); err != nil {
		return nil, fmt.Errorf("campaign %s loot: %w", campaignID, err)
	}
	return loot, nil
}

func (s *Server) getLoot(w http.ResponseWriter, r *http.Request) {
	loot, err := s.campaignLoot(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loot)
}

// lootRows is the loot list as the campaign page shows it.
func lootRows(loot []tracker.Item) []views.LootRow {
	out := make([]views.LootRow, 0, len(loot))
	for _, it := range loot {
		var tag []string
		if it.Kind != "" {
			tag = append(tag, it.Kind)
		}
		if sum := it.Summary(); sum != "" {
			tag = append(tag, sum)
		}
		out = append(out, views.LootRow{ID: it.ID, Name: it.Name, Notes: it.Notes, Tag: strings.Join(tag, " · ")})
	}
	return out
}

// changeLoot applies change to a campaign's loot list and saves it, then
// returns to the list; change's error is shown on the page.
func (s *Server) changeLoot(w http.ResponseWriter, r *http.Request, change func(loot []tracker.Item) ([]tracker.Item, error)) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	loot, err := s.campaignLoot(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		writeStoreError(w, err)
		return
	}
	loot, err = change(loot)
	if err != nil {
		s.renderCampaignPage(w, r, http.StatusBadRequest, id, "Loot: "+err.Error())
		return
	}
	data, err := json.Marshal(loot)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.store.SetCampaignLoot(r.Context(), id, data); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id+"#loot", http.StatusSeeOther)
}

// saveLootForm adds an item to the loot list, or replaces the one with the
// same name (keeping its id).
func (s *Server) saveLootForm(w http.ResponseWriter, r *http.Request) {
	s.changeLoot(w, r, func(loot []tracker.Item) ([]tracker.Item, error) {
		kind, stats, err := formItemStats(r)
		if err != nil {
			return nil, err
		}
		heal, err := formInt("Heals Body", r.PostFormValue("heal_body"), 0, tracker.MaxItemStat, 0)
		if err != nil {
			return nil, err
		}
		restore, err := formInt("Restores mana", r.PostFormValue("restore_mana"), 0, tracker.MaxItemStat, 0)
		if err != nil {
			return nil, err
		}
		checked, err := tracker.NormalizeItems([]tracker.Item{{
			Name: r.PostFormValue("name"), Notes: r.PostFormValue("notes"), Kind: kind, ItemStats: stats, HealBody: heal, RestoreMana: restore,
		}})
		if err != nil {
			return nil, err
		}
		it := checked[0]
		out := slices.Clone(loot)
		if i := slices.IndexFunc(out, func(c tracker.Item) bool { return strings.EqualFold(c.Name, it.Name) }); i >= 0 {
			it.ID = out[i].ID
			out[i] = it
			return out, nil
		}
		it.ID = nextLootID(out)
		return append(out, it), nil
	})
}

func (s *Server) deleteLootForm(w http.ResponseWriter, r *http.Request) {
	s.changeLoot(w, r, func(loot []tracker.Item) ([]tracker.Item, error) {
		id := r.PathValue("lootId")
		return slices.DeleteFunc(slices.Clone(loot), func(it tracker.Item) bool { return it.ID == id }), nil
	})
}

// nextLootID is "loot-N", one past the highest in use.
func nextLootID(loot []tracker.Item) string {
	n := 0
	for _, it := range loot {
		if v, err := strconv.Atoi(strings.TrimPrefix(it.ID, "loot-")); err == nil && v > n {
			n = v
		}
	}
	return fmt.Sprintf("loot-%d", n+1)
}
