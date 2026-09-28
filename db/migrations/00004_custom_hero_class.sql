-- +goose Up

-- A custom hero class is a GM-made class (stats, dice, abilities) that
-- campaign heroes can take alongside the catalog classes.
CREATE TABLE custom_hero_class (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  name        text NOT NULL,
  doc         jsonb NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE custom_hero_class;
