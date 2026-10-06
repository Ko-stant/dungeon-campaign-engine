-- +goose Up

-- Membership (hosting, docs/ONLINE_AND_RULES_PLAN.md Phase 6): anyone may
-- sign in, but a new user waits until an admin approves them (member) or
-- turns them away (refused). Users who signed in before this were already
-- let in, so they start as members.
ALTER TABLE app_user ADD COLUMN member_status text NOT NULL DEFAULT 'pending'
  CHECK (member_status IN ('pending', 'member', 'refused'));
UPDATE app_user SET member_status = 'member';

-- +goose Down
ALTER TABLE app_user DROP COLUMN member_status;
