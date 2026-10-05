package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func TestSigningInCreatesOneUserPerIdentity(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	u, err := st.SignIn(ctx, store.Identity{Provider: "discord", Subject: "1001", DisplayName: "Sam", AvatarURL: "https://cdn/a.png"})
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == "" || u.DisplayName != "Sam" || u.AvatarURL != "https://cdn/a.png" {
		t.Fatalf("user %+v", u)
	}
	// The same identity again is the same user, with the latest name.
	again, err := st.SignIn(ctx, store.Identity{Provider: "discord", Subject: "1001", DisplayName: "Sammy"})
	if err != nil || again.ID != u.ID || again.DisplayName != "Sammy" || again.AvatarURL != "" {
		t.Fatalf("again %+v, %v", again, err)
	}
	other, err := st.SignIn(ctx, store.Identity{Provider: "discord", Subject: "1002", DisplayName: "Jo"})
	if err != nil || other.ID == u.ID {
		t.Fatalf("another identity is another user: %+v %v", other, err)
	}
	ids, err := st.UserIdentities(ctx, u.ID)
	if err != nil || len(ids) != 1 || ids[0] != (store.IdentityKey{Provider: "discord", Subject: "1001"}) {
		t.Fatalf("identities %+v %v", ids, err)
	}
}

func TestLoginSessions(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	u, _ := st.SignIn(ctx, store.Identity{Provider: "dev", Subject: "gm", DisplayName: "GM"})
	hash := []byte("0123456789abcdef0123456789abcdef")
	if err := st.CreateLoginSession(ctx, u.ID, hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := st.LoginSessionUser(ctx, hash)
	if err != nil || got.ID != u.ID || got.DisplayName != "GM" {
		t.Fatalf("session user %+v, %v", got, err)
	}
	if _, err := st.LoginSessionUser(ctx, []byte("nope")); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("an unknown token: %v", err)
	}
	if err := st.DeleteLoginSession(ctx, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := st.LoginSessionUser(ctx, hash); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("after signing out: %v", err)
	}

	old := []byte("expired-expired-expired-expired!")
	if err := st.CreateLoginSession(ctx, u.ID, old, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.LoginSessionUser(ctx, old); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("an expired session: %v", err)
	}
}

func TestOwnersAndViewers(t *testing.T) {
	st, _ := storetest.New(t)
	bg := context.Background()
	gm, _ := st.SignIn(bg, store.Identity{Provider: "dev", Subject: "gm", DisplayName: "GM"})
	friend, _ := st.SignIn(bg, store.Identity{Provider: "dev", Subject: "friend", DisplayName: "Friend"})
	asGM := store.WithViewer(bg, store.Viewer{UserID: gm.ID})
	asFriend := store.WithViewer(bg, store.Viewer{UserID: friend.ID})
	asAdmin := store.WithViewer(bg, store.Viewer{UserID: friend.ID, Admin: true})

	legacy, _ := st.CreateBoard(bg, "Old board", 2, 2, nil)
	mine, _ := st.CreateBoard(asGM, "GM board", 2, 2, nil)
	quest, _ := st.CreateQuest(asGM, mine.ID, "Q", nil)
	camp, _ := st.CreateCampaign(asGM, "GM campaign", nil)
	sess, _ := st.CreateSession(asGM, camp.ID, quest.ID, "Night", []byte(`{}`))
	monster, _ := st.CreateCustomMonster(asGM, "Ogre", nil)
	class, _ := st.CreateCustomHeroClass(asGM, "Ranger", []byte(`{}`))

	for _, c := range []struct {
		kind store.OwnedKind
		id   string
	}{
		{store.OwnedBoard, mine.ID}, {store.OwnedQuest, quest.ID}, {store.OwnedCampaign, camp.ID},
		{store.OwnedSession, sess.ID}, {store.OwnedMonster, monster.ID}, {store.OwnedClass, class.ID},
	} {
		owner, err := st.OwnerOf(bg, c.kind, c.id)
		if err != nil || owner != gm.ID {
			t.Errorf("%s %s owner %q, %v", c.kind, c.id, owner, err)
		}
	}
	if owner, err := st.OwnerOf(bg, store.OwnedBoard, legacy.ID); err != nil || owner != "" {
		t.Errorf("a board made without a viewer has no owner: %q %v", owner, err)
	}
	if _, err := st.OwnerOf(bg, store.OwnedBoard, "0190c6a0-0000-7000-8000-000000000000"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("a missing board: %v", err)
	}

	names := func(ctx context.Context) []string {
		boards, err := st.ListBoards(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, b := range boards {
			out = append(out, b.Name)
		}
		return out
	}
	if got := names(asGM); len(got) != 1 || got[0] != "GM board" {
		t.Errorf("the GM sees their own boards: %v", got)
	}
	if got := names(asFriend); len(got) != 0 {
		t.Errorf("a friend sees none of them: %v", got)
	}
	if got := names(asAdmin); len(got) != 2 {
		t.Errorf("an admin sees all: %v", got)
	}
	if got := names(bg); len(got) != 2 {
		t.Errorf("without sign-in (table mode) everything shows: %v", got)
	}
	for name, count := range map[string]func(context.Context) int{
		"campaigns": func(ctx context.Context) int { l, _ := st.ListCampaigns(ctx); return len(l) },
		"monsters":  func(ctx context.Context) int { l, _ := st.ListCustomMonsters(ctx); return len(l) },
		"classes":   func(ctx context.Context) int { l, _ := st.ListCustomHeroClasses(ctx); return len(l) },
		"chapters": func(ctx context.Context) int {
			_ = st.SetChapters(asGM, camp.ID, []string{quest.ID})
			l, _ := st.ListAllChapters(ctx)
			return len(l)
		},
	} {
		if count(asGM) != 1 || count(asFriend) != 0 {
			t.Errorf("%s: the GM sees %d, a friend %d", name, count(asGM), count(asFriend))
		}
	}
}
