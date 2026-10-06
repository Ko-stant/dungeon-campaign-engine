package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// --- Users and sign-in ---

// User is someone who signs in.
type User struct {
	ID          string
	DisplayName string
	AvatarURL   string
	// Status is MemberPending, MemberApproved or MemberRefused.
	Status    string
	CreatedAt time.Time
}

// Membership statuses: a new user waits until an admin approves or refuses them.
const (
	MemberPending  = "pending"
	MemberApproved = "member"
	MemberRefused  = "refused"
)

// IdentityKey names one way a user signs in.
type IdentityKey struct {
	Provider string
	Subject  string
}

// Identity is what a sign-in provider says about someone.
type Identity struct {
	Provider    string
	Subject     string
	DisplayName string
	AvatarURL   string
}

const userColumns = `u.id::text, u.display_name, u.avatar_url, u.member_status, u.created_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.DisplayName, &u.AvatarURL, &u.Status, &u.CreatedAt)
	return u, notFoundIfNoRows(err)
}

// SetMemberStatus approves, refuses or puts back a user.
func (s *Store) SetMemberStatus(ctx context.Context, userID, status string) error {
	if status != MemberPending && status != MemberApproved && status != MemberRefused {
		return fmt.Errorf("unknown member status %q", status)
	}
	if !validID(userID) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE app_user SET member_status = $2, updated_at = now() WHERE id = $1`, userID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListUsers lists everyone who has signed in: those waiting first, then
// members, then the refused, each oldest first.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+userColumns+` FROM app_user u
		ORDER BY CASE u.member_status WHEN 'pending' THEN 0 WHEN 'member' THEN 1 ELSE 2 END, u.created_at, u.id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) { return scanUser(row) })
}

// CountWaiting counts the users waiting for approval.
func (s *Store) CountWaiting(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM app_user WHERE member_status = 'pending'`).Scan(&n)
	return n, err
}

// SignIn returns the user for an identity, creating both the first time,
// and keeps the user's name and avatar up to date with the provider's.
func (s *Store) SignIn(ctx context.Context, id Identity) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	u, err := scanUser(tx.QueryRow(ctx,
		`UPDATE app_user u SET display_name = $3, avatar_url = $4, updated_at = now()
		 FROM user_identity i WHERE i.user_id = u.id AND i.provider = $1 AND i.subject = $2
		 RETURNING `+userColumns,
		id.Provider, id.Subject, id.DisplayName, id.AvatarURL))
	if errors.Is(err, ErrNotFound) {
		u, err = scanUser(tx.QueryRow(ctx,
			`INSERT INTO app_user AS u (display_name, avatar_url) VALUES ($1, $2) RETURNING `+userColumns,
			id.DisplayName, id.AvatarURL))
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO user_identity (provider, subject, user_id) VALUES ($1, $2, $3)`, id.Provider, id.Subject, u.ID)
		}
	}
	if err != nil {
		return User{}, err
	}
	return u, tx.Commit(ctx)
}

// UserIdentities lists the ways a user signs in.
func (s *Store) UserIdentities(ctx context.Context, userID string) ([]IdentityKey, error) {
	if !validID(userID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT provider, subject FROM user_identity WHERE user_id = $1 ORDER BY provider, subject`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (IdentityKey, error) {
		var k IdentityKey
		err := row.Scan(&k.Provider, &k.Subject)
		return k, err
	})
}

// CreateLoginSession records a signed-in browser by the hash of its token.
func (s *Store) CreateLoginSession(ctx context.Context, userID string, tokenHash []byte, expires time.Time) error {
	if !validID(userID) {
		return ErrNotFound
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO user_session (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`, tokenHash, userID, expires)
	return err
}

// LoginSessionUser returns the user of an unexpired login session.
func (s *Store) LoginSessionUser(ctx context.Context, tokenHash []byte) (User, error) {
	return scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM user_session t JOIN app_user u ON u.id = t.user_id
		 WHERE t.token_hash = $1 AND t.expires_at > now()`, tokenHash))
}

// DeleteLoginSession signs a browser out.
func (s *Store) DeleteLoginSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM user_session WHERE token_hash = $1`, tokenHash)
	return err
}

// DeleteExpiredLoginSessions clears old login sessions.
func (s *Store) DeleteExpiredLoginSessions(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM user_session WHERE expires_at <= now()`)
	return err
}
