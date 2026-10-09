-- +goose Up

-- People who sign in (online play). A user may sign in several ways; each
-- is an identity: a provider ("discord", "dev") and that provider's id.
CREATE TABLE app_user (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  display_name  text NOT NULL,
  avatar_url    text NOT NULL DEFAULT '',
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_identity (
  provider    text NOT NULL,
  subject     text NOT NULL,
  user_id     uuid NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (provider, subject)
);
CREATE INDEX user_identity_user_idx ON user_identity (user_id);

-- A signed-in browser. Only a hash of the cookie's token is stored.
CREATE TABLE user_session (
  token_hash  bytea PRIMARY KEY,
  user_id     uuid NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
  created_at  timestamptz NOT NULL DEFAULT now(),
  expires_at  timestamptz NOT NULL
);
CREATE INDEX user_session_user_idx ON user_session (user_id);

-- Owners. Quests belong to their board's owner and sessions and chapters to
-- their campaign's; rows made before sign-in existed have none (only admins
-- reach them).
ALTER TABLE board ADD COLUMN owner_id uuid REFERENCES app_user (id) ON DELETE SET NULL;
ALTER TABLE campaign ADD COLUMN owner_id uuid REFERENCES app_user (id) ON DELETE SET NULL;
ALTER TABLE custom_monster ADD COLUMN owner_id uuid REFERENCES app_user (id) ON DELETE SET NULL;
ALTER TABLE custom_hero_class ADD COLUMN owner_id uuid REFERENCES app_user (id) ON DELETE SET NULL;
CREATE INDEX board_owner_idx ON board (owner_id);
CREATE INDEX campaign_owner_idx ON campaign (owner_id);
CREATE INDEX custom_monster_owner_idx ON custom_monster (owner_id);
CREATE INDEX custom_hero_class_owner_idx ON custom_hero_class (owner_id);

-- +goose Down
ALTER TABLE custom_hero_class DROP COLUMN owner_id;
ALTER TABLE custom_monster DROP COLUMN owner_id;
ALTER TABLE campaign DROP COLUMN owner_id;
ALTER TABLE board DROP COLUMN owner_id;
DROP TABLE user_session;
DROP TABLE user_identity;
DROP TABLE app_user;
