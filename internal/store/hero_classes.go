package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// CustomHeroClass is a GM-made hero class. Doc holds its stats and abilities.
type CustomHeroClass struct {
	ID        string
	Name      string
	Doc       json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

const customHeroClassColumns = `id::text, name, doc, created_at, updated_at`

func scanCustomHeroClass(row pgx.Row) (CustomHeroClass, error) {
	var c CustomHeroClass
	err := row.Scan(&c.ID, &c.Name, &c.Doc, &c.CreatedAt, &c.UpdatedAt)
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

// DeleteCustomHeroClass removes a custom hero class. Running sessions keep
// their heroes' stats; callers check CampaignsUsingClass first.
func (s *Store) DeleteCustomHeroClass(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM custom_hero_class WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CampaignsUsingClass returns the names of campaigns with a hero of the
// given class id (a catalog id or "custom-<uuid>"), by name.
func (s *Store) CampaignsUsingClass(ctx context.Context, classID string) ([]string, error) {
	match, err := json.Marshal([]map[string]string{{"class": classID}})
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT name FROM campaign WHERE heroes @> $1::jsonb ORDER BY lower(name), id`, match)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
