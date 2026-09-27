-- +goose Up

-- A board is the dungeon layout: size plus per-tile regions (void, corridor,
-- rooms). The full layout lives in doc; width/height are copied out so lists
-- can show them without parsing JSON.
CREATE TABLE board (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  name        text NOT NULL,
  width       integer NOT NULL CHECK (width BETWEEN 1 AND 200),
  height      integer NOT NULL CHECK (height BETWEEN 1 AND 200),
  doc         jsonb NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

-- A quest is a set of layers placed on a board: doors, blocked squares,
-- furniture, monsters, traps, notes and start tiles.
CREATE TABLE quest (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  board_id    uuid NOT NULL REFERENCES board (id) ON DELETE RESTRICT,
  name        text NOT NULL,
  doc         jsonb NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX quest_board_id_idx ON quest (board_id);

-- A campaign groups the heroes (and what they carry between quests) and the
-- quest sessions played with them.
CREATE TABLE campaign (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  name        text NOT NULL,
  heroes      jsonb NOT NULL DEFAULT '[]'::jsonb,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

-- A game session is one quest being played. state is the complete current
-- state, including frozen copies of the board and quest taken at start, so a
-- resumed session never depends on later edits. event_seq counts events.
CREATE TABLE game_session (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  campaign_id  uuid NOT NULL REFERENCES campaign (id) ON DELETE CASCADE,
  quest_id     uuid REFERENCES quest (id) ON DELETE SET NULL,
  name         text NOT NULL,
  status       text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed')),
  state        jsonb NOT NULL,
  event_seq    bigint NOT NULL DEFAULT 0,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX game_session_campaign_idx ON game_session (campaign_id, status);

-- The readable event capture: one row per GM change, in order. Corrections are
-- just further events; nothing is edited or deleted.
CREATE TABLE session_event (
  id          bigserial PRIMARY KEY,
  session_id  uuid NOT NULL REFERENCES game_session (id) ON DELETE CASCADE,
  seq         bigint NOT NULL,
  round       integer NOT NULL,
  kind        text NOT NULL,
  summary     text NOT NULL,
  payload     jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at  timestamptz NOT NULL DEFAULT now(),
  UNIQUE (session_id, seq)
);

-- +goose Down
DROP TABLE session_event;
DROP TABLE game_session;
DROP TABLE campaign;
DROP TABLE quest;
DROP TABLE board;
