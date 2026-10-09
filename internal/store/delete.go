package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// DeleteCampaign removes a campaign with its chapter list and its sessions
// (and their events). The boards and quests stay.
func (s *Store) DeleteCampaign(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM campaign WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSession removes one of a campaign's sessions and its events. A
// session of another campaign is not found.
func (s *Store) DeleteSession(ctx context.Context, campaignID, id string) error {
	if !validID(campaignID) || !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM game_session WHERE id = $1 AND campaign_id = $2`, id, campaignID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteBoardWithQuests removes a board and every quest on it, which also
// takes those quests out of any campaign's chapters. Sessions keep their
// frozen copy of the map. (DeleteBoard refuses a board that still has quests.)
func (s *Store) DeleteBoardWithQuests(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM quest WHERE board_id = $1`, id); err != nil {
			return fmt.Errorf("delete the board's quests: %w", err)
		}
		tag, err := tx.Exec(ctx, `DELETE FROM board WHERE id = $1`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}
