-- +goose Up

-- A campaign's chapters: its quests in play order. Each quest sits on its own
-- board, so a campaign can span several maps. Deleting a quest drops its
-- chapter; deleting a campaign drops all of them.
CREATE TABLE campaign_chapter (
  campaign_id  uuid NOT NULL REFERENCES campaign (id) ON DELETE CASCADE,
  quest_id     uuid NOT NULL REFERENCES quest (id) ON DELETE CASCADE,
  position     integer NOT NULL,
  PRIMARY KEY (campaign_id, quest_id)
);
CREATE INDEX campaign_chapter_quest_idx ON campaign_chapter (quest_id);

-- +goose Down
DROP TABLE campaign_chapter;
