-- +goose Up

-- The base game's hero classes (Barbarian, Elf...) are kept here too,
-- imported from content/heroes by make import-content. catalog_id is the
-- class's catalog id ("barbarian"), which campaign and session heroes already
-- use; the GM's own classes have none.
ALTER TABLE custom_hero_class ADD COLUMN catalog_id text UNIQUE;

-- +goose Down
ALTER TABLE custom_hero_class DROP COLUMN catalog_id;
