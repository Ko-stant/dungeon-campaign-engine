package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func TestCampaignMonsterStats(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.CreateCampaign(ctx, "Three Plagues", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := s.GetCampaignMonsterStats(ctx, c.ID)
	if err != nil || string(raw) != "{}" {
		t.Fatalf("new campaign: %s %v", raw, err)
	}
	want := `{"orc": {"body": 22, "avoidance": 8, "hitDice": "2d8", "damage": 9}}`
	if err := s.SetCampaignMonsterStats(ctx, c.ID, json.RawMessage(want)); err != nil {
		t.Fatal(err)
	}
	raw, err = s.GetCampaignMonsterStats(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	var got, exp map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal([]byte(want), &exp)
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("saved stats: %s", raw)
	}
	if err := s.SetCampaignMonsterStats(ctx, "nope", json.RawMessage(`{}`)); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad id: %v", err)
	}
	if _, err := s.GetCampaignMonsterStats(ctx, "0190c6a0-0000-7000-8000-000000000000"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown campaign: %v", err)
	}
}
