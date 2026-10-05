package store_test

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func TestFindCampaign(t *testing.T) {
	list := []store.Campaign{
		{ID: "01a0ea8a-e8d5-7517-94a0-7176179e5bb3", Name: "Three Plagues"},
		{ID: "01a0e40a-2bb2-73f1-bab5-bc73f1a4e347", Name: "Test Campaign"},
		{ID: "01a0e41a-e653-79f2-b4ef-61083f6fe5c8", Name: "test campaign"},
	}
	for key, want := range map[string]string{
		"Three Plagues":                        "01a0ea8a-e8d5-7517-94a0-7176179e5bb3",
		"  three plagues ":                     "01a0ea8a-e8d5-7517-94a0-7176179e5bb3",
		"01a0e40a-2bb2-73f1-bab5-bc73f1a4e347": "01a0e40a-2bb2-73f1-bab5-bc73f1a4e347",
	} {
		c, err := store.FindCampaign(list, key)
		if err != nil || c.ID != want {
			t.Errorf("store.FindCampaign(%q) = %+v, %v; want %s", key, c, err, want)
		}
	}
	for key, msg := range map[string]string{
		"":              "which campaign",
		"Nope":          "Three Plagues",
		"test campaign": "2 campaigns",
	} {
		if _, err := store.FindCampaign(list, key); err == nil || !strings.Contains(err.Error(), msg) {
			t.Errorf("store.FindCampaign(%q) error %v should mention %q", key, err, msg)
		}
	}
}
