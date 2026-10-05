-- +goose Up

-- The monsters each event showed the players ([{id, name, map}]), kept apart
-- from player_summary so removing a monster added by mistake can take its
-- sighting back.
ALTER TABLE session_event ADD COLUMN player_spotted jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE session_event DROP COLUMN player_spotted;
