package store

import "context"

// GetCampaignScript returns a campaign's read-aloud script text ("" if none).
func (s *Store) GetCampaignScript(ctx context.Context, id string) (string, error) {
	if !validID(id) {
		return "", ErrNotFound
	}
	var text string
	err := s.pool.QueryRow(ctx, `SELECT script FROM campaign WHERE id = $1`, id).Scan(&text)
	return text, notFoundIfNoRows(err)
}

// SetCampaignScript replaces a campaign's read-aloud script text.
func (s *Store) SetCampaignScript(ctx context.Context, id, text string) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE campaign SET script = $2, updated_at = now() WHERE id = $1`, id, text)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
