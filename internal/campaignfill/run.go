package campaignfill

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/tracker"
)

// classIDPrefix starts a custom class's catalog id, as the app's catalog
// names custom classes ("custom-<uuid>").
const classIDPrefix = "custom-"

// Run fills the campaign (a name or id) with d: the custom hero classes
// (matched by name; missing ones are created), the campaign's monster stat
// lines and each hero's starting kit (skipped while a quest is running, as
// completing it would overwrite the heroes). With apply false nothing is
// saved. It returns a report of what it did or would do.
func Run(ctx context.Context, st *store.Store, d Data, campaign string, apply bool) ([]string, error) {
	list, err := st.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	c, err := store.FindCampaign(list, campaign)
	if err != nil {
		return nil, err
	}
	var report []string
	if !apply {
		report = append(report, "Dry run: nothing is saved. Run again with -apply to save these changes.")
	}
	section := func(title string, lines []string) {
		report = append(report, "", title)
		if len(lines) == 0 {
			lines = []string{"up to date"}
		}
		for _, l := range lines {
			report = append(report, "  "+l)
		}
	}

	// Classes are shared by every campaign.
	classes, err := st.ListCustomHeroClasses(ctx)
	if err != nil {
		return nil, err
	}
	classIDs := map[string]string{}
	var lines []string
	for _, spec := range d.Classes {
		var rec *store.CustomHeroClass
		for i := range classes {
			if strings.EqualFold(classes[i].Name, spec.Name) {
				rec = &classes[i]
			}
		}
		var old json.RawMessage
		if rec != nil {
			old = rec.Doc
		}
		doc, changes, err := MergeClass(old, spec)
		if err != nil {
			return nil, fmt.Errorf("class %s: %w", spec.Name, err)
		}
		switch {
		case rec == nil:
			lines = append(lines, spec.Name+": new class")
			if apply {
				created, err := st.CreateCustomHeroClass(ctx, spec.Name, doc)
				if err != nil {
					return nil, err
				}
				classIDs[spec.Name] = classIDPrefix + created.ID
			}
		case len(changes) > 0:
			lines = append(lines, spec.Name+": "+strings.Join(changes, ", "))
			if apply {
				if _, err := st.UpdateCustomHeroClass(ctx, rec.ID, rec.Name, doc); err != nil {
					return nil, err
				}
			}
			classIDs[spec.Name] = classIDPrefix + rec.ID
		default:
			classIDs[spec.Name] = classIDPrefix + rec.ID
		}
	}
	section("Hero classes (shared by every campaign)", lines)

	raw, err := st.GetCampaignMonsterStats(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	var cur map[string]content.MonsterStats
	if err := json.Unmarshal(raw, &cur); err != nil {
		return nil, err
	}
	stats, changes := MergeMonsterStats(cur, d.Monsters)
	if apply && len(changes) > 0 {
		data, err := json.Marshal(stats)
		if err != nil {
			return nil, err
		}
		if err := st.SetCampaignMonsterStats(ctx, c.ID, data); err != nil {
			return nil, err
		}
	}
	section(fmt.Sprintf("Monster stats (%s)", c.Name), changes)

	sessions, err := st.ListSessions(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	for _, s := range sessions {
		if s.Status == store.StatusActive {
			section(fmt.Sprintf("Starting kits (%s)", c.Name), []string{fmt.Sprintf("skipped: the heroes are in a quest (%q); complete it, then run again", s.Name)})
			return report, nil
		}
	}
	var heroes []tracker.CampaignHero
	if err := json.Unmarshal(c.Heroes, &heroes); err != nil {
		return nil, err
	}
	kitted, changes, err := GiveKits(heroes, classIDs, d.StartingKit)
	if err != nil {
		return nil, err
	}
	before, _ := json.Marshal(heroes)
	data, err := json.Marshal(kitted)
	if err != nil {
		return nil, err
	}
	if apply && string(data) != string(before) {
		if _, err := st.UpdateCampaign(ctx, c.ID, c.Name, data); err != nil {
			return nil, err
		}
	}
	section(fmt.Sprintf("Starting kits (%s)", c.Name), changes)
	return report, nil
}
