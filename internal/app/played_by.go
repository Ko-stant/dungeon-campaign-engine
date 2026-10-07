package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// Who plays which hero online is the campaign hero's UserID. Players pick
// free heroes on the join page; the GM can also hand any hero to a member
// (a friend covering for someone who can't come) or free it again, on the
// campaign page. Open seats update at once.

func (s *Server) registerPlayedBy(mux routeMux) {
	mux.HandleFunc("POST /campaigns/{id}/heroes/{heroId}/player", s.setHeroPlayer)
}

// memberChoices lists who may be given a hero: members (and admins, who are
// members from their first sign-in), by name. With open membership, everyone
// who has signed in.
func (s *Server) memberChoices(ctx context.Context) ([]views.MemberChoice, error) {
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	var out []views.MemberChoice
	for _, u := range users {
		if s.auth.OpenMembership || u.Status == store.MemberApproved {
			out = append(out, views.MemberChoice{ID: u.ID, Name: u.DisplayName})
		}
	}
	slices.SortFunc(out, func(a, b views.MemberChoice) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}

func (s *Server) setHeroPlayer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	campaignID := r.PathValue("id")
	camp, heroes, err := s.loadCampaign(ctx, campaignID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	i := slices.IndexFunc(heroes, func(h tracker.CampaignHero) bool { return h.ID == r.PathValue("heroId") })
	if i < 0 {
		http.NotFound(w, r)
		return
	}
	userID := r.PostFormValue("user")
	if userID == "" {
		heroes[i].UserID = ""
	} else {
		u, err := s.store.GetUser(ctx, userID)
		if errors.Is(err, store.ErrNotFound) || (err == nil && !s.auth.OpenMembership && u.Status != store.MemberApproved) {
			http.Error(w, "Only a member can be given a hero.", http.StatusBadRequest)
			return
		}
		if err != nil {
			writeStoreError(w, err)
			return
		}
		// Who plays the hero now; Player (who usually does) stays as it is.
		heroes[i].UserID = u.ID
	}
	if _, err := s.saveCampaignHeroes(ctx, camp.ID, camp.Name, heroes); err != nil {
		writeStoreError(w, err)
		return
	}
	s.refreshSeats(ctx, camp.ID, heroes)
	http.Redirect(w, r, "/campaigns/"+camp.ID, http.StatusSeeOther)
}

// refreshSeats sends the open seats of a campaign's active sessions their
// heroes again, after who plays them changed, and who is here.
func (s *Server) refreshSeats(ctx context.Context, campaignID string, heroes []tracker.CampaignHero) {
	sessions, err := s.store.ListSessions(ctx, campaignID)
	if err != nil {
		log.Printf("app: seats of campaign %s: %v", campaignID, err)
		return
	}
	for _, sess := range sessions {
		if sess.Status != store.StatusActive || len(s.seats.list(sess.ID)) == 0 {
			continue
		}
		ss, state, err := s.loadSessionState(ctx, sess.ID)
		if err != nil {
			log.Printf("app: seats of %s: %v", sess.ID, err)
			continue
		}
		s.seats.setHeroes(sess.ID, func(userID string) []string {
			var names []string
			for _, h := range heroes {
				if h.UserID == userID {
					names = append(names, h.Name)
				}
			}
			return names
		})
		s.pushSeats(ctx, sess.ID, state, PlayerUpdate{State: tracker.PlayerView(state), EventSeq: ss.EventSeq})
		s.announcePresence(sess.ID)
	}
}
