package store_test

import (
	"context"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func TestOpenSessionsAreListedForEveryone(t *testing.T) {
	st, _ := storetest.New(t)
	bg := context.Background()
	gm, _ := st.SignIn(bg, store.Identity{Provider: "dev", Subject: "gm", DisplayName: "The GM"})
	friend, _ := st.SignIn(bg, store.Identity{Provider: "dev", Subject: "sam", DisplayName: "Sam"})
	asGM := store.WithViewer(bg, store.Viewer{UserID: gm.ID})
	camp, _ := st.CreateCampaign(asGM, "Three Plagues", nil)
	closed, _ := st.CreateSession(asGM, camp.ID, "", "Prep", []byte(`{}`))
	night, _ := st.CreateSession(asGM, camp.ID, "", "Game night", []byte(`{"rules": {"phase": "heroes"}}`))
	if closed.Open || night.Open {
		t.Fatal("sessions start closed to players")
	}
	if err := st.SetSessionOpen(bg, night.ID, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.GetSession(bg, night.ID); !got.Open {
		t.Error("the session should be open now")
	}

	// A friend who owns nothing still sees open sessions, with who runs them.
	list, err := st.ListOpenSessions(store.WithViewer(bg, store.Viewer{UserID: friend.ID}))
	if err != nil {
		t.Fatal(err)
	}
	want := store.OpenSession{ID: night.ID, Name: "Game night", CampaignID: camp.ID, CampaignName: "Three Plagues", GMName: "The GM", Started: true}
	if len(list) != 1 || list[0].ID != want.ID || list[0].Name != want.Name || list[0].CampaignID != want.CampaignID ||
		list[0].CampaignName != want.CampaignName || list[0].GMName != want.GMName || !list[0].Started {
		t.Fatalf("open sessions %+v", list)
	}

	if err := st.SetSessionStatus(bg, night.ID, store.StatusCompleted); err != nil {
		t.Fatal(err)
	}
	if list, _ := st.ListOpenSessions(bg); len(list) != 0 {
		t.Errorf("a completed session is not listed: %+v", list)
	}
	summaries, _ := st.ListSessions(bg, camp.ID)
	for _, ss := range summaries {
		if ss.ID == night.ID && !ss.Open {
			t.Error("the campaign's session list shows the open flag")
		}
	}
}
