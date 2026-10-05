-- +goose Up

-- A campaign's own stat lines for monster types (The Three Plagues combat):
-- monster type id -> {body, avoidance, hitDice, damage, traits}. Sessions in the
-- campaign use them when monsters are set up or added.
ALTER TABLE campaign ADD COLUMN monster_stats jsonb NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE campaign DROP COLUMN monster_stats;
