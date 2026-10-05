package store

import (
	"context"
	"encoding/json"
)

// GetCampaignLoot returns a campaign's loot list (a JSON array of items).
func (s *Store) GetCampaignLoot(ctx context.Context, id string) (json.RawMessage, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	var raw json.RawMessage
	err := s.pool.QueryRow(ctx, `SELECT loot FROM campaign WHERE id = $1`, id).Scan(&raw)
	return raw, notFoundIfNoRows(err)
}

// SetCampaignLoot replaces a campaign's loot list.
func (s *Store) SetCampaignLoot(ctx context.Context, id string, loot json.RawMessage) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE campaign SET loot = $2, updated_at = now() WHERE id = $1`, id, loot)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
