-- +goose Up

-- Read-aloud audio clips (hosting, docs/ONLINE_AND_RULES_PLAN.md Phase 6):
-- kept in the database so one backup covers everything and no disk is
-- needed. A clip is named after a script passage id ("Q2-03"; "Q3-09a" is
-- an extra clip of Q3-09); saving an id again replaces it.
CREATE TABLE audio_clip (
  campaign_id   uuid NOT NULL REFERENCES campaign (id) ON DELETE CASCADE,
  clip_id       text NOT NULL,
  ext           text NOT NULL,
  content_type  text NOT NULL,
  data          bytea NOT NULL,
  sha256        bytea NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (campaign_id, clip_id)
);

-- +goose Down
DROP TABLE audio_clip;
