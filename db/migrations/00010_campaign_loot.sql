-- +goose Up

-- A campaign's loot list: items the GM hands out from chests and finds, with
-- their kind and stats ready ([{id, name, kind, stats, healBody, ...}]).
ALTER TABLE campaign ADD COLUMN loot jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE campaign DROP COLUMN loot;
