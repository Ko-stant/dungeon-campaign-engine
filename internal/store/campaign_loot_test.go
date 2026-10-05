package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func TestCampaignLoot(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.CreateCampaign(ctx, "Three Plagues", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := s.GetCampaignLoot(ctx, c.ID)
	if err != nil || string(raw) != "[]" {
		t.Fatalf("new campaign: %s %v", raw, err)
	}
	if err := s.SetCampaignLoot(ctx, c.ID, json.RawMessage(`[{"id": "loot-1", "name": "Healing Potion", "quantity": 1, "healBody": 8}]`)); err != nil {
		t.Fatal(err)
	}
	raw, err = s.GetCampaignLoot(ctx, c.ID)
	var got []map[string]any
	if err != nil || json.Unmarshal(raw, &got) != nil || len(got) != 1 || got[0]["name"] != "Healing Potion" {
		t.Fatalf("saved loot: %s %v", raw, err)
	}
	if err := s.SetCampaignLoot(ctx, "nope", json.RawMessage(`[]`)); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad id: %v", err)
	}
	if _, err := s.GetCampaignLoot(ctx, "01900000-0000-7000-8000-000000000000"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown campaign: %v", err)
	}
}
