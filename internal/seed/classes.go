package seed

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// ClassesResult counts what ImportClasses did.
type ClassesResult struct {
	Created, Updated, Unchanged int
}

// ImportClasses stores the base game's hero classes (content/heroes) in the
// database, keyed by their catalog id, so the server reads classes from the
// database. Safe to run repeatedly; a class the GM deactivated stays so.
func ImportClasses(ctx context.Context, st *store.Store, heroes []content.HeroDef) (ClassesResult, error) {
	var res ClassesResult
	for _, h := range heroes {
		doc, err := json.Marshal(h)
		if err != nil {
			return res, fmt.Errorf("class %s: %w", h.ID, err)
		}
		got, err := st.UpsertCatalogHeroClass(ctx, h.ID, h.Name, doc)
		if err != nil {
			return res, fmt.Errorf("class %s: %w", h.ID, err)
		}
		switch got {
		case store.ClassCreated:
			res.Created++
		case store.ClassUpdated:
			res.Updated++
		default:
			res.Unchanged++
		}
	}
	return res, nil
}
