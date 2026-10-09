package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func TestCustomHeroClassLifecycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	doc := json.RawMessage(`{"color":"#aa3300","body":30,"attack":"1d8","abilities":[{"id":"ability-1","name":"Fan of Blades"}]}`)
	c, err := s.CreateCustomHeroClass(ctx, "Rogue", doc)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == "" || c.Name != "Rogue" {
		t.Fatalf("created: %+v", c)
	}
	jsonEqual(t, c.Doc, doc)

	got, err := s.GetCustomHeroClass(ctx, c.ID)
	if err != nil || got.Name != "Rogue" {
		t.Fatalf("get: %+v %v", got, err)
	}
	jsonEqual(t, got.Doc, doc)

	more := json.RawMessage(`{"color":"#aa3300","body":35}`)
	upd, err := s.UpdateCustomHeroClass(ctx, c.ID, "Shadow Rogue", more)
	if err != nil || upd.Name != "Shadow Rogue" {
		t.Fatalf("update: %+v %v", upd, err)
	}
	jsonEqual(t, upd.Doc, more)

	if _, err := s.CreateCustomHeroClass(ctx, "barbarian", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("second create: %v", err)
	}
	list, err := s.ListCustomHeroClasses(ctx)
	if err != nil || len(list) != 2 || list[0].Name != "barbarian" || list[1].ID != c.ID {
		t.Fatalf("list (by name, case-insensitive): %+v %v", list, err)
	}

	if _, err := s.UpdateCustomHeroClass(ctx, "nope", "x", doc); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad id: %v", err)
	}
}

func TestCustomHeroClassDeactivateAndReactivate(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.CreateCustomHeroClass(ctx, "Rogue", json.RawMessage(`{"body":30}`))
	if err != nil {
		t.Fatal(err)
	}
	if !c.Active {
		t.Fatal("a new class should be active")
	}

	if err := s.SetCustomHeroClassActive(ctx, c.ID, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetCustomHeroClass(ctx, c.ID); got.Active {
		t.Fatal("the class should be deactivated")
	}
	if list, _ := s.ListCustomHeroClasses(ctx); len(list) != 1 || list[0].Active {
		t.Fatalf("a deactivated class is still listed, as deactivated: %+v", list)
	}

	// Saving the class (its form, or the campaign fill) keeps it deactivated.
	upd, err := s.UpdateCustomHeroClass(ctx, c.ID, "Rogue", json.RawMessage(`{"body":35}`))
	if err != nil {
		t.Fatal(err)
	}
	if upd.Active {
		t.Fatal("saving the class should keep it deactivated")
	}
	if err := s.SetCustomHeroClassActive(ctx, c.ID, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetCustomHeroClass(ctx, c.ID); !got.Active {
		t.Fatal("the class should be active again")
	}

	if err := s.SetCustomHeroClassActive(ctx, "01a0ea8a-e8d5-7517-94a0-7176179e5bb3", false); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("missing class: err = %v, want ErrNotFound", err)
	}
	if err := s.SetCustomHeroClassActive(ctx, "nope", false); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("bad id: err = %v, want ErrNotFound", err)
	}
}
