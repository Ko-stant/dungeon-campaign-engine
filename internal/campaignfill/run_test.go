package campaignfill

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

func TestRun(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	d := load(t)

	rogue, err := st.CreateCustomHeroClass(ctx, "Rogue", json.RawMessage(`{"color": "#334455", "body": 20, "mind": 4, "attack": "1d6", "defense": "1d6", "movement": "2d6",
		"abilities": [{"id": "ability-1", "name": "Fan of Blades", "kind": "active", "cooldown": 3}], "nextAbility": 2}`))
	if err != nil {
		t.Fatal(err)
	}
	heroes, _ := json.Marshal([]tracker.CampaignHero{{ID: "hero-1", Name: "Vex", Class: "custom-" + rogue.ID}})
	c, err := st.CreateCampaign(ctx, "Three Plagues", heroes)
	if err != nil {
		t.Fatal(err)
	}

	// A dry run saves nothing.
	lines, err := Run(ctx, st, d, "three plagues", false)
	if err != nil {
		t.Fatal(err)
	}
	report := strings.Join(lines, "\n")
	for _, s := range []string{"Dry run", "Rogue: body 20 → 28", "Fan of Blades renamed Fan of Cards", "Cleric: new class", "goblin added", "Vex: Sword, Soft Boots (equipped)", "no Cleric hero"} {
		if !strings.Contains(report, s) {
			t.Errorf("dry run report should mention %q:\n%s", s, report)
		}
	}
	if list, _ := st.ListCustomHeroClasses(ctx); len(list) != 1 {
		t.Fatalf("a dry run creates no class: %d", len(list))
	}
	if raw, _ := st.GetCampaignMonsterStats(ctx, c.ID); string(raw) != "{}" {
		t.Fatalf("a dry run sets no monster stats: %s", raw)
	}

	if _, err := Run(ctx, st, d, "Three Plagues", true); err != nil {
		t.Fatal(err)
	}
	list, _ := st.ListCustomHeroClasses(ctx)
	if len(list) != 2 {
		t.Fatalf("classes after the fill: %d", len(list))
	}
	got, _ := st.GetCustomHeroClass(ctx, rogue.ID)
	var doc struct {
		Body      int               `json:"body"`
		Mind      int               `json:"mind"`
		Color     string            `json:"color"`
		Abilities []content.Ability `json:"abilities"`
	}
	_ = json.Unmarshal(got.Doc, &doc)
	if doc.Body != 28 || doc.Mind != 4 || doc.Color != "#334455" || len(doc.Abilities) != 3 || doc.Abilities[1].ID != "ability-1" {
		t.Fatalf("rogue after the fill: %+v", doc)
	}
	raw, _ := st.GetCampaignMonsterStats(ctx, c.ID)
	var stats map[string]content.MonsterStats
	_ = json.Unmarshal(raw, &stats)
	if stats["gargoyle"].Body != 109 || stats["gargoyle"].Line != 1 {
		t.Fatalf("monster stats: %s", raw)
	}
	camp, _ := st.GetCampaign(ctx, c.ID)
	var after []tracker.CampaignHero
	_ = json.Unmarshal(camp.Heroes, &after)
	if len(after[0].Items) != 2 || !after[0].Items[0].Equipped {
		t.Fatalf("Vex's kit: %+v", after[0].Items)
	}

	// A second run finds nothing to do.
	lines, err = Run(ctx, st, d, "Three Plagues", true)
	if err != nil {
		t.Fatal(err)
	}
	report = strings.Join(lines, "\n")
	if strings.Contains(report, "→") || strings.Contains(report, "added") || !strings.Contains(report, "Vex already carries starting gear") {
		t.Fatalf("second run:\n%s", report)
	}

	// During a quest the heroes' kit is left alone: the session would overwrite it.
	if _, err := st.CreateSession(ctx, c.ID, "", "Night 1", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	lines, _ = Run(ctx, st, d, "Three Plagues", false)
	if report = strings.Join(lines, "\n"); !strings.Contains(report, "in a quest") {
		t.Fatalf("in a quest:\n%s", report)
	}
	if _, err := Run(ctx, st, d, "Nope", false); err == nil {
		t.Fatal("an unknown campaign should fail")
	}
}
