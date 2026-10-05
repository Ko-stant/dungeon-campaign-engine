-- +goose Up

-- One purse for the party (The Three Plagues): each hero's gold is added up
-- into campaign.gold and the session state's "gold", and taken off the heroes.
ALTER TABLE campaign ADD COLUMN gold integer NOT NULL DEFAULT 0 CHECK (gold >= 0);

UPDATE campaign SET
  gold = LEAST(999999, COALESCE((
    SELECT sum((h->>'gold')::int) FROM jsonb_array_elements(heroes) h
    WHERE jsonb_typeof(h->'gold') = 'number'), 0)),
  heroes = COALESCE((
    SELECT jsonb_agg(h - 'gold' ORDER BY n) FROM jsonb_array_elements(heroes) WITH ORDINALITY e(h, n)), '[]'::jsonb)
WHERE jsonb_typeof(heroes) = 'array';

UPDATE game_session SET state = jsonb_set(state, '{heroes}', COALESCE((
    SELECT jsonb_agg(h - 'gold' ORDER BY n) FROM jsonb_array_elements(state->'heroes') WITH ORDINALITY e(h, n)), '[]'::jsonb))
  || jsonb_build_object('gold', LEAST(999999, COALESCE((
    SELECT sum((h->>'gold')::int) FROM jsonb_array_elements(state->'heroes') h
    WHERE jsonb_typeof(h->'gold') = 'number'), 0)))
WHERE jsonb_typeof(state->'heroes') = 'array';

-- +goose Down

-- The purse goes to each party's first hero.
UPDATE campaign SET heroes = jsonb_set(heroes, '{0,gold}', to_jsonb(gold))
WHERE jsonb_typeof(heroes) = 'array' AND jsonb_array_length(heroes) > 0;

UPDATE game_session SET state = CASE
    WHEN jsonb_typeof(state->'heroes') = 'array' AND jsonb_array_length(state->'heroes') > 0
      THEN jsonb_set(state, '{heroes,0,gold}', COALESCE(state->'gold', '0'::jsonb))
    ELSE state
  END - 'gold';

ALTER TABLE campaign DROP COLUMN gold;
