package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func TestUpsertCatalogHeroClass(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	doc := json.RawMessage(`{"body":8,"mind":2,"attack":3,"defense":2,"movementDice":2}`)

	got, err := st.UpsertCatalogHeroClass(ctx, "barbarian", "Barbarian", doc)
	if err != nil || got != store.ClassCreated {
		t.Fatalf("first import: %v %v", got, err)
	}
	if got, err = st.UpsertCatalogHeroClass(ctx, "barbarian", "Barbarian", doc); err != nil || got != store.ClassUnchanged {
		t.Fatalf("same again: %v %v", got, err)
	}
	if got, err = st.UpsertCatalogHeroClass(ctx, "barbarian", "Barbarian", json.RawMessage(`{"body":9}`)); err != nil || got != store.ClassUpdated {
		t.Fatalf("changed stats: %v %v", got, err)
	}

	list, err := st.ListCustomHeroClasses(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %+v %v", list, err)
	}
	rec := list[0]
	if rec.CatalogID != "barbarian" || rec.Name != "Barbarian" || !rec.Active {
		t.Fatalf("catalog class: %+v", rec)
	}
	jsonEqual(t, rec.Doc, json.RawMessage(`{"body":9}`))

	// A deactivated base class stays deactivated when imported again.
	if err := st.SetCustomHeroClassActive(ctx, rec.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertCatalogHeroClass(ctx, "barbarian", "Barbarian", doc); err != nil {
		t.Fatal(err)
	}
	if again, _ := st.GetCustomHeroClass(ctx, rec.ID); again.Active {
		t.Error("importing again should keep the class deactivated")
	}

	// Base classes are read-only: the class form's update does not touch them.
	if _, err := st.UpdateCustomHeroClass(ctx, rec.ID, "Brute", json.RawMessage(`{}`)); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("update a base class: err = %v, want ErrNotFound", err)
	}
	if _, err := st.UpsertCatalogHeroClass(ctx, "", "Nameless", doc); err == nil {
		t.Error("a catalog class needs its catalog id")
	}
}

func TestCatalogHeroClassesAreListedForEveryone(t *testing.T) {
	st, _ := storetest.New(t)
	bg := context.Background()
	gm, _ := st.SignIn(bg, store.Identity{Provider: "dev", Subject: "gm", DisplayName: "The GM"})
	friend, _ := st.SignIn(bg, store.Identity{Provider: "dev", Subject: "sam", DisplayName: "Sam"})
	asGM := store.WithViewer(bg, store.Viewer{UserID: gm.ID})
	if _, err := st.CreateCustomHeroClass(asGM, "Rogue", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertCatalogHeroClass(bg, "elf", "Elf", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}

	// The GM sees their own class and the base game's; a friend sees only the base game's.
	names := func(ctx context.Context) []string {
		t.Helper()
		list, err := st.ListCustomHeroClasses(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, c := range list {
			out = append(out, c.Name)
		}
		return out
	}
	if got := names(asGM); len(got) != 2 || got[0] != "Elf" || got[1] != "Rogue" {
		t.Errorf("the GM's classes: %v", got)
	}
	if got := names(store.WithViewer(bg, store.Viewer{UserID: friend.ID})); len(got) != 1 || got[0] != "Elf" {
		t.Errorf("a friend's classes: %v", got)
	}
}
