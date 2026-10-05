package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

// Campaign monster stats: a campaign's own stat lines for monster types (The
// Three Plagues combat). Sessions in the campaign use them when monsters are
// set up or added; monsters without a line keep their catalog stats.
func (s *Server) registerCampaignMonsterPages(mux *http.ServeMux) {
	mux.HandleFunc("POST /campaigns/{id}/monsters", s.saveMonsterStatsForm)
	mux.HandleFunc("POST /campaigns/{id}/monsters/{type}/delete", s.deleteMonsterStatsForm)
}

// campaignMonsterStats returns a campaign's monster stat lines by monster type.
func (s *Server) campaignMonsterStats(ctx context.Context, campaignID string) (map[string]content.MonsterStats, error) {
	raw, err := s.store.GetCampaignMonsterStats(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	stats := map[string]content.MonsterStats{}
	if err := json.Unmarshal(raw, &stats); err != nil {
		return nil, fmt.Errorf("campaign %s monster stats: %w", campaignID, err)
	}
	return stats, nil
}

// campaignCatalog is the catalog (with custom monsters and classes) with a
// campaign's monster stat lines laid over its monsters.
func (s *Server) campaignCatalog(ctx context.Context, campaignID string) (*content.Catalog, error) {
	cat, err := s.catalogFor(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := s.campaignMonsterStats(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return cat.WithMonsterStats(stats), nil
}

// parseMonsterStatsForm reads one monster stat line; the type must be in cat.
// maxMonsterAbilities bounds a stat line's abilities text.
const maxMonsterAbilities = 500

func parseMonsterStatsForm(r *http.Request, cat *content.Catalog) (string, content.MonsterStats, error) {
	pf := r.PostForm
	var st content.MonsterStats
	typ := strings.TrimSpace(pf.Get("type"))
	if _, ok := cat.Monster(typ); !ok {
		return "", st, fmt.Errorf("unknown monster type %q", typ)
	}
	for _, f := range []struct {
		label, name string
		lo, hi      int
		dst         *int
	}{
		{"body", "body", 1, 999, &st.Body},
		{"avoidance", "avoidance", 0, 99, &st.Avoidance},
		{"damage", "damage", 0, 99, &st.Damage},
		{"line (extra heroes struck)", "line", 0, 3, &st.Line},
		{"splash damage", "splash_damage", 0, 99, &st.SplashDamage},
		{"splash targets", "splash_targets", 0, 8, &st.SplashTargets},
	} {
		n, err := formInt(f.label, pf.Get(f.name), f.lo, f.hi, f.lo)
		if err != nil {
			return "", st, err
		}
		*f.dst = n
	}
	hit, err := formDice("hit dice", pf.Get("hit_dice"))
	if err != nil {
		return "", st, err
	}
	st.HitDice = hit
	st.Ranged, st.Reach, st.Undead = pf.Get("ranged") != "", pf.Get("reach") != "", pf.Get("undead") != ""
	st.Abilities = strings.TrimSpace(pf.Get("abilities"))
	if len(st.Abilities) > maxMonsterAbilities {
		return "", st, fmt.Errorf("abilities must be at most %d characters", maxMonsterAbilities)
	}
	return typ, st, nil
}

// changeMonsterStats applies change to a campaign's stat lines and saves them.
// change returns an error to show on the page; notFound true means a 404.
func (s *Server) changeMonsterStats(w http.ResponseWriter, r *http.Request, change func(stats map[string]content.MonsterStats, cat *content.Catalog) (notFound bool, err error)) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	stats, err := s.campaignMonsterStats(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	cat, err := s.catalogFor(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	notFound, err := change(stats, cat)
	if notFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.renderCampaignPage(w, r, http.StatusBadRequest, id, "Monster stats: "+err.Error())
		return
	}
	raw, err := json.Marshal(stats)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.store.SetCampaignMonsterStats(r.Context(), id, raw); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id+"#monster-stats", http.StatusSeeOther)
}

func (s *Server) saveMonsterStatsForm(w http.ResponseWriter, r *http.Request) {
	s.changeMonsterStats(w, r, func(stats map[string]content.MonsterStats, cat *content.Catalog) (bool, error) {
		typ, st, err := parseMonsterStatsForm(r, cat)
		if err != nil {
			return false, err
		}
		stats[typ] = st
		return false, nil
	})
}

func (s *Server) deleteMonsterStatsForm(w http.ResponseWriter, r *http.Request) {
	s.changeMonsterStats(w, r, func(stats map[string]content.MonsterStats, _ *content.Catalog) (bool, error) {
		typ := r.PathValue("type")
		if _, ok := stats[typ]; !ok {
			return true, nil
		}
		delete(stats, typ)
		return false, nil
	})
}
