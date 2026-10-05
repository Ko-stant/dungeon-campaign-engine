package store

import (
	"context"
	"encoding/json"
)

// GetCampaignMonsterStats returns a campaign's monster stat lines as stored
// (a JSON object keyed by monster type; "{}" if none).
func (s *Store) GetCampaignMonsterStats(ctx context.Context, id string) (json.RawMessage, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	var raw json.RawMessage
	err := s.pool.QueryRow(ctx, `SELECT monster_stats FROM campaign WHERE id = $1`, id).Scan(&raw)
	return raw, notFoundIfNoRows(err)
}

// SetCampaignMonsterStats replaces a campaign's monster stat lines.
func (s *Store) SetCampaignMonsterStats(ctx context.Context, id string, stats json.RawMessage) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE campaign SET monster_stats = $2, updated_at = now() WHERE id = $1`, id, stats)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
