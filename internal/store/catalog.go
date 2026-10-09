package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// The base game's catalog lives in the database, imported from content/ by
// make import-content: hero classes and monsters as rows of
// custom_hero_class and custom_monster with a catalog_id, furniture and trap
// kinds in catalog_piece. Imports are idempotent.

// Imported says what an import did to one catalog entry.
type Imported int

// The outcomes of an import.
const (
	ImportUnchanged Imported = iota
	ImportCreated
	ImportUpdated
)

// upsertCatalogRow inserts a catalog entry, or replaces its name and doc when
// they changed. query must insert ($1 name, $2 doc, $3 catalog id, then any
// more args), update only on a real change, and return xmax = 0.
func (s *Store) upsertCatalogRow(ctx context.Context, query, catalogID, name string, doc json.RawMessage, more ...any) (Imported, error) {
	if catalogID == "" {
		return ImportUnchanged, errors.New("store: a catalog entry needs its catalog id")
	}
	var inserted bool
	args := append([]any{name, orEmptyObject(doc), catalogID}, more...)
	err := s.pool.QueryRow(ctx, query, args...).Scan(&inserted)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ImportUnchanged, nil
	case err != nil:
		return ImportUnchanged, err
	case inserted:
		return ImportCreated, nil
	}
	return ImportUpdated, nil
}

// UpsertCatalogHeroClass stores a base game class by its catalog id. Whether
// it is active stays as the GM left it. Base classes have no owner (everyone
// sees them).
func (s *Store) UpsertCatalogHeroClass(ctx context.Context, catalogID, name string, doc json.RawMessage) (Imported, error) {
	return s.upsertCatalogRow(ctx,
		`INSERT INTO custom_hero_class (name, doc, catalog_id) VALUES ($1, $2, $3)
		 ON CONFLICT (catalog_id) DO UPDATE SET name = EXCLUDED.name, doc = EXCLUDED.doc, updated_at = now()
		 WHERE (custom_hero_class.name, custom_hero_class.doc) IS DISTINCT FROM (EXCLUDED.name, EXCLUDED.doc)
		 RETURNING xmax = 0`, catalogID, name, doc)
}

// UpsertCatalogMonster stores a base game monster type by its catalog id. Base
// monsters have no owner (everyone sees them).
func (s *Store) UpsertCatalogMonster(ctx context.Context, catalogID, name string, doc json.RawMessage) (Imported, error) {
	return s.upsertCatalogRow(ctx,
		`INSERT INTO custom_monster (name, doc, catalog_id) VALUES ($1, $2, $3)
		 ON CONFLICT (catalog_id) DO UPDATE SET name = EXCLUDED.name, doc = EXCLUDED.doc, updated_at = now()
		 WHERE (custom_monster.name, custom_monster.doc) IS DISTINCT FROM (EXCLUDED.name, EXCLUDED.doc)
		 RETURNING xmax = 0`, catalogID, name, doc)
}

// PieceKind is a kind of catalog_piece.
type PieceKind string

// The catalog_piece kinds.
const (
	PieceFurniture PieceKind = "furniture"
	PieceTrap      PieceKind = "trap"
)

// CatalogPiece is a base game furniture or trap kind; Doc is its catalog
// entry (content.FurnitureDef or content.TrapDef).
type CatalogPiece struct {
	Kind      PieceKind
	CatalogID string
	Name      string
	Doc       json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

func validPieceKind(kind PieceKind) bool {
	return kind == PieceFurniture || kind == PieceTrap
}

// UpsertCatalogPiece stores a base game furniture or trap kind by its catalog id.
func (s *Store) UpsertCatalogPiece(ctx context.Context, kind PieceKind, catalogID, name string, doc json.RawMessage) (Imported, error) {
	if !validPieceKind(kind) {
		return ImportUnchanged, fmt.Errorf("store: unknown catalog piece kind %q", kind)
	}
	return s.upsertCatalogRow(ctx,
		`INSERT INTO catalog_piece (name, doc, catalog_id, kind) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (kind, catalog_id) DO UPDATE SET name = EXCLUDED.name, doc = EXCLUDED.doc, updated_at = now()
		 WHERE (catalog_piece.name, catalog_piece.doc) IS DISTINCT FROM (EXCLUDED.name, EXCLUDED.doc)
		 RETURNING xmax = 0`, catalogID, name, doc, string(kind))
}

// ListCatalogPieces returns the furniture or trap kinds, by catalog id.
func (s *Store) ListCatalogPieces(ctx context.Context, kind PieceKind) ([]CatalogPiece, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT kind, catalog_id, name, doc, created_at, updated_at FROM catalog_piece WHERE kind = $1 ORDER BY catalog_id`, string(kind))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (CatalogPiece, error) {
		var p CatalogPiece
		var k string
		err := row.Scan(&k, &p.CatalogID, &p.Name, &p.Doc, &p.CreatedAt, &p.UpdatedAt)
		p.Kind = PieceKind(k)
		return p, err
	})
}
