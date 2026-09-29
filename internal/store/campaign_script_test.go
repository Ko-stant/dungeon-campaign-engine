package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func TestCampaignScript(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.CreateCampaign(ctx, "Three Plagues", nil)
	if err != nil {
		t.Fatal(err)
	}
	if text, err := s.GetCampaignScript(ctx, c.ID); err != nil || text != "" {
		t.Fatalf("new campaign script: %q %v", text, err)
	}
	if err := s.SetCampaignScript(ctx, c.ID, "## Prologue\n"); err != nil {
		t.Fatal(err)
	}
	if text, err := s.GetCampaignScript(ctx, c.ID); err != nil || text != "## Prologue\n" {
		t.Fatalf("saved script: %q %v", text, err)
	}
	if err := s.SetCampaignScript(ctx, "nope", "x"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad id: %v", err)
	}
	if _, err := s.GetCampaignScript(ctx, "0190c6a0-0000-7000-8000-000000000000"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown campaign: %v", err)
	}
}
