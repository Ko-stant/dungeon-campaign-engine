package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// Membership (docs/ONLINE_AND_RULES_PLAN.md, Phase 6): anyone may sign in,
// but only members get past the guard. A new user waits on /waiting until an
// admin lets them in on /members.

func (s *Server) registerMembers(mux routeMux) {
	mux.HandleFunc("GET /members", s.membersPage)
	mux.HandleFunc("POST /members/{id}", s.setMember)
}

// waitingPage is what a signed-in user who is not (yet) a member sees.
// Registered outside the guard, which sends them here.
func (s *Server) waitingPage(w http.ResponseWriter, r *http.Request) {
	if !s.auth.On() {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	user, viewer, ok := s.signedIn(r)
	if !ok {
		s.notSignedIn(w, r)
		return
	}
	if s.isMember(user, viewer) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	render(w, r, http.StatusOK, views.WaitingPage(user.DisplayName, user.Status == store.MemberRefused))
}

func (s *Server) membersPage(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	rows := make([]views.MemberRow, 0, len(users))
	for _, u := range users {
		ids, err := s.store.UserIdentities(r.Context(), u.ID)
		if err != nil {
			log.Printf("app: identities of %s: %v", u.ID, err)
		}
		row := views.MemberRow{ID: u.ID, Name: u.DisplayName, Avatar: u.AvatarURL, Status: u.Status, Admin: s.auth.IsAdmin(ids), Since: u.CreatedAt.Format("Jan 2, 2006")}
		if len(ids) > 0 {
			row.SignsInAs = fmt.Sprintf("%s %s", ids[0].Provider, ids[0].Subject)
		}
		rows = append(rows, row)
	}
	render(w, r, http.StatusOK, views.MembersPage(rows))
}

func (s *Server) setMember(w http.ResponseWriter, r *http.Request) {
	status := r.PostFormValue("status")
	switch status {
	case store.MemberApproved, store.MemberRefused, store.MemberPending:
	default:
		http.Error(w, "Unknown member status.", http.StatusBadRequest)
		return
	}
	if err := s.store.SetMemberStatus(r.Context(), r.PathValue("id"), status); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/members", http.StatusSeeOther)
}
