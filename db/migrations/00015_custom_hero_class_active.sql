-- +goose Up

-- Classes are deactivated instead of deleted: a deactivated class is left out
-- of the new-hero pickers, and heroes who already have it keep it.
ALTER TABLE custom_hero_class ADD COLUMN active boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE custom_hero_class DROP COLUMN active;
