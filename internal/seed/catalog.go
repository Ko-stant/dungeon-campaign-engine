package seed

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// Counts is what an import did to one kind of catalog entry.
type Counts struct {
	Created, Updated, Unchanged int
}

func (c *Counts) add(got store.Imported) {
	switch got {
	case store.ImportCreated:
		c.Created++
	case store.ImportUpdated:
		c.Updated++
	default:
		c.Unchanged++
	}
}

// CatalogResult counts what ImportCatalog did, by kind.
type CatalogResult struct {
	Classes, Monsters, Furniture, Traps Counts
}

// ByKind is the result keyed by kind name, in the order the import reports them.
func (r CatalogResult) ByKind() map[string]Counts {
	return map[string]Counts{"classes": r.Classes, "monsters": r.Monsters, "furniture": r.Furniture, "traps": r.Traps}
}

// ImportCatalog stores the base game's catalog (loaded from content/) in the
// database: hero classes, monster types, furniture and trap kinds, keyed by
// their catalog ids, so the server reads the catalog from the database. Each
// entry is stored as its content definition. Safe to run repeatedly; a class
// the GM deactivated stays so.
func ImportCatalog(ctx context.Context, st *store.Store, cat *content.Catalog) (CatalogResult, error) {
	var res CatalogResult
	put := func(counts *Counts, what, id, name string, def any, upsert func(id, name string, doc json.RawMessage) (store.Imported, error)) error {
		doc, err := json.Marshal(def)
		if err != nil {
			return fmt.Errorf("%s %s: %w", what, id, err)
		}
		got, err := upsert(id, name, doc)
		if err != nil {
			return fmt.Errorf("%s %s: %w", what, id, err)
		}
		counts.add(got)
		return nil
	}
	upsertPiece := func(kind store.PieceKind) func(id, name string, doc json.RawMessage) (store.Imported, error) {
		return func(id, name string, doc json.RawMessage) (store.Imported, error) {
			return st.UpsertCatalogPiece(ctx, kind, id, name, doc)
		}
	}
	upsertClass := func(id, name string, doc json.RawMessage) (store.Imported, error) {
		return st.UpsertCatalogHeroClass(ctx, id, name, doc)
	}
	upsertMonster := func(id, name string, doc json.RawMessage) (store.Imported, error) {
		return st.UpsertCatalogMonster(ctx, id, name, doc)
	}
	for _, h := range cat.Heroes {
		if err := put(&res.Classes, "class", h.ID, h.Name, h, upsertClass); err != nil {
			return res, err
		}
	}
	for _, m := range cat.Monsters {
		if err := put(&res.Monsters, "monster", m.ID, m.Name, m, upsertMonster); err != nil {
			return res, err
		}
	}
	for _, f := range cat.Furniture {
		if err := put(&res.Furniture, "furniture", f.ID, f.Name, f, upsertPiece(store.PieceFurniture)); err != nil {
			return res, err
		}
	}
	for _, tr := range cat.Traps {
		if err := put(&res.Traps, "trap", tr.ID, tr.Name, tr, upsertPiece(store.PieceTrap)); err != nil {
			return res, err
		}
	}
	return res, nil
}
