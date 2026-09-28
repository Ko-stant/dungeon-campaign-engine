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

	if err := s.DeleteCustomHeroClass(ctx, c.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.DeleteCustomHeroClass(ctx, c.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
	if _, err := s.GetCustomHeroClass(ctx, c.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("get deleted: %v", err)
	}
	if _, err := s.UpdateCustomHeroClass(ctx, "nope", "x", doc); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad id: %v", err)
	}
}

func TestCampaignsUsingClass(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	heroes := json.RawMessage(`[{"id":"hero-1","name":"Vex","class":"custom-abc"},{"id":"hero-2","name":"Bram","class":"elf"}]`)
	if _, err := s.CreateCampaign(ctx, "Three Plagues", heroes); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCampaign(ctx, "another", json.RawMessage(`[{"id":"hero-1","name":"Ana","class":"custom-abc"}]`)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCampaign(ctx, "Unrelated", json.RawMessage(`[{"id":"hero-1","name":"Ana","class":"elf"}]`)); err != nil {
		t.Fatal(err)
	}
	names, err := s.CampaignsUsingClass(ctx, "custom-abc")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "another" || names[1] != "Three Plagues" {
		t.Fatalf("campaigns using class: %v", names)
	}
	if names, err = s.CampaignsUsingClass(ctx, "custom-none"); err != nil || len(names) != 0 {
		t.Fatalf("unused class: %v %v", names, err)
	}
}
