package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// AudioClip is a read-aloud clip of a campaign, without its sound.
type AudioClip struct {
	// ID is the clip's name: a passage id, or one with a letter after it.
	ID string
	// Ext is the file extension, with its dot (".mp3").
	Ext         string
	ContentType string
	Size        int
	// ETag changes whenever the clip is replaced.
	ETag      string
	UpdatedAt time.Time
}

// File is the clip's file name, e.g. "Q2-03.mp3".
func (c AudioClip) File() string { return c.ID + c.Ext }

const audioColumns = `clip_id, ext, content_type, octet_length(data), encode(sha256, 'hex'), updated_at`

func scanAudioClip(row pgx.Row, extra ...any) (AudioClip, error) {
	var c AudioClip
	err := row.Scan(append([]any{&c.ID, &c.Ext, &c.ContentType, &c.Size, &c.ETag, &c.UpdatedAt}, extra...)...)
	return c, notFoundIfNoRows(err)
}

// ListAudioClips lists a campaign's clips by id.
func (s *Store) ListAudioClips(ctx context.Context, campaignID string) ([]AudioClip, error) {
	if !validID(campaignID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT `+audioColumns+` FROM audio_clip WHERE campaign_id = $1 ORDER BY clip_id`, campaignID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (AudioClip, error) { return scanAudioClip(row) })
}

// GetAudioClip returns one of a campaign's clips and its sound.
func (s *Store) GetAudioClip(ctx context.Context, campaignID, clipID string) (AudioClip, []byte, error) {
	if !validID(campaignID) {
		return AudioClip{}, nil, ErrNotFound
	}
	var data []byte
	c, err := scanAudioClip(s.pool.QueryRow(ctx,
		`SELECT `+audioColumns+`, data FROM audio_clip WHERE campaign_id = $1 AND clip_id = $2`, campaignID, clipID), &data)
	return c, data, err
}

// SaveAudioClip saves a clip (its ID, Ext and ContentType) with its sound,
// replacing any clip with the same id.
func (s *Store) SaveAudioClip(ctx context.Context, campaignID string, c AudioClip, data []byte) error {
	if !validID(campaignID) {
		return ErrNotFound
	}
	sum := sha256.Sum256(data)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audio_clip (campaign_id, clip_id, ext, content_type, data, sha256) VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (campaign_id, clip_id) DO UPDATE
		   SET ext = EXCLUDED.ext, content_type = EXCLUDED.content_type, data = EXCLUDED.data, sha256 = EXCLUDED.sha256, updated_at = now()`,
		campaignID, c.ID, c.Ext, c.ContentType, data, sum[:])
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign key: no such campaign
		return ErrNotFound
	}
	return err
}

// DeleteAudioClip removes one of a campaign's clips.
func (s *Store) DeleteAudioClip(ctx context.Context, campaignID, clipID string) error {
	if !validID(campaignID) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM audio_clip WHERE campaign_id = $1 AND clip_id = $2`, campaignID, clipID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AudioETag is the ETag of some sound, as ListAudioClips reports it.
func AudioETag(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
