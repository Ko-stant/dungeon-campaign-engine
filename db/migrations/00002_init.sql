-- +goose Up
CREATE TABLE game (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v1mc(),
  map_id text NOT NULL,
  quest_id text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  dm_user_id text NOT NULL
);

CREATE TABLE player (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v1mc(),
  game_id uuid NOT NULL REFERENCES game(id) ON DELETE CASCADE,
  user_id text NOT NULL,
  hero_type text NOT NULL,
  name text NOT NULL
);

CREATE TABLE event (
  event_id bigserial PRIMARY KEY,
  game_id uuid NOT NULL REFERENCES game(id) ON DELETE CASCADE,
  turn_number integer NOT NULL,
  index_in_turn integer NOT NULL,
  type text NOT NULL,
  payload_json jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX event_game_event_id_idx ON event (game_id, event_id);
CREATE INDEX event_game_turn_idx ON event (game_id, turn_number, index_in_turn);
CREATE INDEX event_type_idx ON event (type);
CREATE INDEX event_payload_gin ON event USING gin (payload_json);

CREATE TABLE snapshot (
  game_id uuid NOT NULL REFERENCES game(id) ON DELETE CASCADE,
  turn_number integer NOT NULL,
  state_binary bytea NOT NULL,
  last_event_id bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (game_id, turn_number)
);

CREATE TABLE content_pack (
  id text NOT NULL,
  version text NOT NULL,
  manifest_json jsonb NOT NULL,
  bundle_binary bytea NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id, version)
);

-- +goose Down
DROP TABLE content_pack;
DROP TABLE snapshot;
DROP INDEX event_payload_gin;
DROP INDEX event_type_idx;
DROP INDEX event_game_turn_idx;
DROP INDEX event_game_event_id_idx;
DROP TABLE event;
DROP TABLE player;
DROP TABLE game;
