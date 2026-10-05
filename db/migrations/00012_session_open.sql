-- +goose Up

-- An open session is listed in the lobby: signed-in players can join it with
-- a hero (online play).
ALTER TABLE game_session ADD COLUMN open boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE game_session DROP COLUMN open;
