package store

import (
	"context"
	"fmt"
)

// --- Owners and viewers ---
//
// With sign-in on, the HTTP layer puts the signed-in Viewer in the request's
// context. Things it creates get that user as owner, and lists show only the
// viewer's own (an admin sees everything). Without a viewer (the table
// companion, tools) nothing is filtered and new things have no owner.

// Viewer is the signed-in user a request acts for.
type Viewer struct {
	UserID string
	Admin  bool
}

type viewerKey struct{}

// WithViewer returns a context acting for a signed-in user.
func WithViewer(ctx context.Context, v Viewer) context.Context {
	return context.WithValue(ctx, viewerKey{}, v)
}

// ViewerFrom returns the context's viewer, if any.
func ViewerFrom(ctx context.Context) (Viewer, bool) {
	v, ok := ctx.Value(viewerKey{}).(Viewer)
	return v, ok
}

// ownerParam is the owner for new rows: the viewer, or NULL.
func ownerParam(ctx context.Context) any {
	if v, ok := ViewerFrom(ctx); ok && validID(v.UserID) {
		return v.UserID
	}
	return nil
}

// filterParam is the owner a list keeps to, or NULL for everything (no
// viewer, or an admin).
func filterParam(ctx context.Context) any {
	if v, ok := ViewerFrom(ctx); ok && !v.Admin {
		if validID(v.UserID) {
			return v.UserID
		}
		return "00000000-0000-0000-0000-000000000000"
	}
	return nil
}

// OwnedKind names what OwnerOf looks up.
type OwnedKind string

// The owned things: quests belong to their board's owner, sessions to
// their campaign's.
const (
	OwnedBoard    OwnedKind = "board"
	OwnedQuest    OwnedKind = "quest"
	OwnedCampaign OwnedKind = "campaign"
	OwnedSession  OwnedKind = "session"
	OwnedMonster  OwnedKind = "custom monster"
	OwnedClass    OwnedKind = "custom class"
)

var ownerQueries = map[OwnedKind]string{
	OwnedBoard:    `SELECT owner_id::text FROM board WHERE id = $1`,
	OwnedQuest:    `SELECT b.owner_id::text FROM quest q JOIN board b ON b.id = q.board_id WHERE q.id = $1`,
	OwnedCampaign: `SELECT owner_id::text FROM campaign WHERE id = $1`,
	OwnedSession:  `SELECT c.owner_id::text FROM game_session g JOIN campaign c ON c.id = g.campaign_id WHERE g.id = $1`,
	OwnedMonster:  `SELECT owner_id::text FROM custom_monster WHERE id = $1`,
	OwnedClass:    `SELECT owner_id::text FROM custom_hero_class WHERE id = $1`,
}

// OwnerOf returns the owner's user id ("" when it has none), or ErrNotFound.
func (s *Store) OwnerOf(ctx context.Context, kind OwnedKind, id string) (string, error) {
	q, ok := ownerQueries[kind]
	if !ok {
		return "", fmt.Errorf("unknown owned kind %q", kind)
	}
	if !validID(id) {
		return "", ErrNotFound
	}
	var owner *string
	if err := notFoundIfNoRows(s.pool.QueryRow(ctx, q, id).Scan(&owner)); err != nil {
		return "", err
	}
	if owner == nil {
		return "", nil
	}
	return *owner, nil
}
