-- +goose Up

-- A custom monster is a GM-made monster type (name, color, size in squares and
-- optional stats) that any quest can place alongside the catalog monsters.
CREATE TABLE custom_monster (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  name        text NOT NULL,
  doc         jsonb NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE custom_monster;
