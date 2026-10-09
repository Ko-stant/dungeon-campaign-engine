package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// CustomHeroClass is a GM-made hero class. Doc holds its stats and abilities.
// A deactivated class (Active false) is left out of new-hero pickers; heroes
// who already have it keep it.
type CustomHeroClass struct {
	ID        string
	Name      string
	Doc       json.RawMessage
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

const customHeroClassColumns = `id::text, name, doc, active, created_at, updated_at`

func scanCustomHeroClass(row pgx.Row) (CustomHeroClass, error) {
	var c CustomHeroClass
	err := row.Scan(&c.ID, &c.Name, &c.Doc, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	return c, notFoundIfNoRows(err)
}

// CreateCustomHeroClass stores a new custom hero class.
func (s *Store) CreateCustomHeroClass(ctx context.Context, name string, doc json.RawMessage) (CustomHeroClass, error) {
	return scanCustomHeroClass(s.pool.QueryRow(ctx,
		`INSERT INTO custom_hero_class (name, doc) VALUES ($1, $2) RETURNING `+customHeroClassColumns,
		name, orEmptyObject(doc)))
}

// GetCustomHeroClass loads a custom hero class by id.
func (s *Store) GetCustomHeroClass(ctx context.Context, id string) (CustomHeroClass, error) {
	if !validID(id) {
		return CustomHeroClass{}, ErrNotFound
	}
	return scanCustomHeroClass(s.pool.QueryRow(ctx,
		`SELECT `+customHeroClassColumns+` FROM custom_hero_class WHERE id = $1`, id))
}

// ListCustomHeroClasses returns every custom hero class by name.
func (s *Store) ListCustomHeroClasses(ctx context.Context) ([]CustomHeroClass, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+customHeroClassColumns+` FROM custom_hero_class ORDER BY lower(name), id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (CustomHeroClass, error) {
		return scanCustomHeroClass(row)
	})
}

// UpdateCustomHeroClass replaces a custom hero class's name and doc.
func (s *Store) UpdateCustomHeroClass(ctx context.Context, id, name string, doc json.RawMessage) (CustomHeroClass, error) {
	if !validID(id) {
		return CustomHeroClass{}, ErrNotFound
	}
	return scanCustomHeroClass(s.pool.QueryRow(ctx,
		`UPDATE custom_hero_class SET name = $2, doc = $3, updated_at = now() WHERE id = $1 RETURNING `+customHeroClassColumns,
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
