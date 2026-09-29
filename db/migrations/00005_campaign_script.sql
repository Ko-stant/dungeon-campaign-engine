-- +goose Up

-- A campaign's read-aloud script, as the GM wrote it (Markdown in the format
-- of docs/campaigns/three-plagues/SCRIPT.md). It is parsed when served.
ALTER TABLE campaign ADD COLUMN script text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE campaign DROP COLUMN script;
