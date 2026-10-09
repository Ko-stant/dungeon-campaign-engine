-- +goose Up

-- The base game's monsters are kept in custom_monster too, imported from
-- content/monsters by make import-content: catalog_id is the monster type's
-- catalog id ("orc"), which quests, sessions and campaign stat lines already
-- use; the GM's own monsters have none.
ALTER TABLE custom_monster ADD COLUMN catalog_id text UNIQUE;

-- The base game's furniture and trap kinds (content/furniture, content/traps),
-- imported the same way. doc is the catalog entry (size, artwork, ...);
-- quests refer to them by catalog_id.
CREATE TABLE catalog_piece (
  kind        text NOT NULL CHECK (kind IN ('furniture', 'trap')),
  catalog_id  text NOT NULL,
  name        text NOT NULL,
  doc         jsonb NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (kind, catalog_id)
);

-- +goose Down
DROP TABLE catalog_piece;
ALTER TABLE custom_monster DROP COLUMN catalog_id;
