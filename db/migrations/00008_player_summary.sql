-- +goose Up

-- The player screen's line for each event (empty: the players hear nothing).
ALTER TABLE session_event ADD COLUMN player_summary text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE session_event DROP COLUMN player_summary;
