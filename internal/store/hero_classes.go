package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// CustomHeroClass is a hero class kept in the database: a GM-made class, or
// a base game class (CatalogID set, e.g. "barbarian") imported from
// content/heroes. Doc holds its stats and abilities (a base class's is its
// content.HeroDef). A deactivated class (Active false) is left out of
// new-hero pickers; heroes who already have it keep it.
type CustomHeroClass struct {
	ID        string
	Name      string
	Doc       json.RawMessage
	Active    bool
	CatalogID string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const customHeroClassColumns = `id::text, name, doc, active, coalesce(catalog_id, ''), created_at, updated_at`

func scanCustomHeroClass(row pgx.Row) (CustomHeroClass, error) {
	var c CustomHeroClass
	err := row.Scan(&c.ID, &c.Name, &c.Doc, &c.Active, &c.CatalogID, &c.CreatedAt, &c.UpdatedAt)
	return c, notFoundIfNoRows(err)
}

// CreateCustomHeroClass stores a new custom hero class.
func (s *Store) CreateCustomHeroClass(ctx context.Context, name string, doc json.RawMessage) (CustomHeroClass, error) {
	return scanCustomHeroClass(s.pool.QueryRow(ctx,
		`INSERT INTO custom_hero_class (name, doc, owner_id) VALUES ($1, $2, $3) RETURNING `+customHeroClassColumns,
		name, orEmptyObject(doc), ownerParam(ctx)))
}

// GetCustomHeroClass loads a custom hero class by id.
func (s *Store) GetCustomHeroClass(ctx context.Context, id string) (CustomHeroClass, error) {
	if !validID(id) {
		return CustomHeroClass{}, ErrNotFound
	}
	return scanCustomHeroClass(s.pool.QueryRow(ctx,
		`SELECT `+customHeroClassColumns+` FROM custom_hero_class WHERE id = $1`, id))
}

// ListCustomHeroClasses returns the viewer's classes and the base game's
// (which everyone sees), by name.
func (s *Store) ListCustomHeroClasses(ctx context.Context) ([]CustomHeroClass, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+customHeroClassColumns+` FROM custom_hero_class
		WHERE $1::uuid IS NULL OR owner_id = $1 OR catalog_id IS NOT NULL ORDER BY lower(name), id`, filterParam(ctx))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (CustomHeroClass, error) {
		return scanCustomHeroClass(row)
	})
}

// UpdateCustomHeroClass replaces a GM-made class's name and doc. Base game
// classes are read-only here (not found); UpsertCatalogHeroClass imports them.
func (s *Store) UpdateCustomHeroClass(ctx context.Context, id, name string, doc json.RawMessage) (CustomHeroClass, error) {
	if !validID(id) {
		return CustomHeroClass{}, ErrNotFound
	}
	return scanCustomHeroClass(s.pool.QueryRow(ctx,
		`UPDATE custom_hero_class SET name = $2, doc = $3, updated_at = now()
		 WHERE id = $1 AND catalog_id IS NULL RETURNING `+customHeroClassColumns,
		id, name, orEmptyObject(doc)))
}

// SetCustomHeroClassActive deactivates or reactivates a custom hero class.
func (s *Store) SetCustomHeroClassActive(ctx context.Context, id string, active bool) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE custom_hero_class SET active = $2, updated_at = now() WHERE id = $1`, id, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ClassImport says what UpsertCatalogHeroClass did.
type ClassImport int

// The outcomes of UpsertCatalogHeroClass.
const (
	ClassUnchanged ClassImport = iota
	ClassCreated
	ClassUpdated
)

// UpsertCatalogHeroClass stores a base game class by its catalog id: it is
// created, or its name and doc are replaced when they changed. Whether it is
// active stays as the GM left it. Base classes have no owner (everyone sees
// them).
func (s *Store) UpsertCatalogHeroClass(ctx context.Context, catalogID, name string, doc json.RawMessage) (ClassImport, error) {
	if catalogID == "" {
		return ClassUnchanged, errors.New("store: a base game class needs its catalog id")
	}
	var inserted bool
	err := s.pool.QueryRow(ctx,
		`INSERT INTO custom_hero_class (name, doc, catalog_id) VALUES ($1, $2, $3)
		 ON CONFLICT (catalog_id) DO UPDATE SET name = EXCLUDED.name, doc = EXCLUDED.doc, updated_at = now()
		 WHERE (custom_hero_class.name, custom_hero_class.doc) IS DISTINCT FROM (EXCLUDED.name, EXCLUDED.doc)
		 RETURNING xmax = 0`,
		name, orEmptyObject(doc), catalogID).Scan(&inserted)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ClassUnchanged, nil
	case err != nil:
		return ClassUnchanged, err
	case inserted:
		return ClassCreated, nil
	}
	return ClassUpdated, nil
}
