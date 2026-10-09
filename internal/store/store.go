// Package store persists boards, quests, campaigns, game sessions and the
// session event log in Postgres.
//
// Documents (board layouts, quest layers, session state, campaign heroes) are
// stored as jsonb and passed through as json.RawMessage; the packages that own
// those shapes marshal them. The store only guarantees persistence, ordering
// and atomicity.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver for goose
	"github.com/pressly/goose/v3"

	"github.com/Ko-stant/dungeon-campaign-engine/db"
)

var (
	// ErrNotFound is returned when a row does not exist or the id is malformed.
	ErrNotFound = errors.New("store: not found")
	// ErrInUse is returned when deleting a row that others still reference.
	ErrInUse = errors.New("store: still in use")
)

// Session statuses.
const (
	StatusActive    = "active"
	StatusCompleted = "completed"
)

const defaultEventLimit = 500

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validID(id string) bool {
	return uuidPattern.MatchString(id)
}

// Store is safe for concurrent use.
type Store struct {
	pool *pgxpool.Pool
	url  string
}

// Open connects to Postgres and verifies the connection.
func Open(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{pool: pool, url: url}, nil
}

// Close releases all connections.
// Ping checks the database answers.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *Store) Close() {
	s.pool.Close()
}

// Migrate applies any pending embedded migrations.
func Migrate(ctx context.Context, url string) error {
	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("store: migrate: open: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	migrations, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
	if err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	return nil
}

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

const (
	fkViolation       = "23503"
	uniqueViolation   = "23505"
	restrictViolation = "23001"
)

func notFoundIfNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func orEmptyObject(doc json.RawMessage) json.RawMessage {
	if len(doc) == 0 {
		return json.RawMessage(`{}`)
	}
	return doc
}

// --- Boards ---

// Board is a stored board layout.
type Board struct {
	ID        string
	Name      string
	Width     int
	Height    int
	Doc       json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BoardSummary is a board without its layout document.
type BoardSummary struct {
	ID        string
	Name      string
	Width     int
	Height    int
	UpdatedAt time.Time
}

const boardColumns = `id::text, name, width, height, doc, created_at, updated_at`

func scanBoard(row pgx.Row) (Board, error) {
	var b Board
	err := row.Scan(&b.ID, &b.Name, &b.Width, &b.Height, &b.Doc, &b.CreatedAt, &b.UpdatedAt)
	return b, notFoundIfNoRows(err)
}

// CreateBoard stores a new board.
func (s *Store) CreateBoard(ctx context.Context, name string, width, height int, doc json.RawMessage) (Board, error) {
	return scanBoard(s.pool.QueryRow(ctx,
		`INSERT INTO board (name, width, height, doc, owner_id) VALUES ($1, $2, $3, $4, $5) RETURNING `+boardColumns,
		name, width, height, orEmptyObject(doc), ownerParam(ctx)))
}

// GetBoard loads a board by id.
func (s *Store) GetBoard(ctx context.Context, id string) (Board, error) {
	if !validID(id) {
		return Board{}, ErrNotFound
	}
	return scanBoard(s.pool.QueryRow(ctx, `SELECT `+boardColumns+` FROM board WHERE id = $1`, id))
}

// ListBoards returns every board, most recently updated first.
func (s *Store) ListBoards(ctx context.Context) ([]BoardSummary, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, name, width, height, updated_at FROM board
		WHERE $1::uuid IS NULL OR owner_id = $1 ORDER BY updated_at DESC, id`, filterParam(ctx))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BoardSummary, error) {
		var b BoardSummary
		err := row.Scan(&b.ID, &b.Name, &b.Width, &b.Height, &b.UpdatedAt)
		return b, err
	})
}

// UpdateBoard replaces a board's name, size and layout.
func (s *Store) UpdateBoard(ctx context.Context, id, name string, width, height int, doc json.RawMessage) (Board, error) {
	if !validID(id) {
		return Board{}, ErrNotFound
	}
	return scanBoard(s.pool.QueryRow(ctx,
		`UPDATE board SET name = $2, width = $3, height = $4, doc = $5, updated_at = now() WHERE id = $1 RETURNING `+boardColumns,
		id, name, width, height, orEmptyObject(doc)))
}

// DeleteBoard removes a board. It fails with ErrInUse while quests use it.
func (s *Store) DeleteBoard(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM board WHERE id = $1`, id)
	if code := pgCode(err); code == fkViolation || code == restrictViolation {
		return ErrInUse
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Quests ---

// Quest is a stored set of layers placed on a board.
type Quest struct {
	ID        string
	BoardID   string
	Name      string
	Doc       json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// QuestSummary is a quest without its layer document.
type QuestSummary struct {
	ID        string
	BoardID   string
	Name      string
	UpdatedAt time.Time
}

const questColumns = `id::text, board_id::text, name, doc, created_at, updated_at`

func scanQuest(row pgx.Row) (Quest, error) {
	var q Quest
	err := row.Scan(&q.ID, &q.BoardID, &q.Name, &q.Doc, &q.CreatedAt, &q.UpdatedAt)
	return q, notFoundIfNoRows(err)
}

// CreateQuest stores a new quest on an existing board.
func (s *Store) CreateQuest(ctx context.Context, boardID, name string, doc json.RawMessage) (Quest, error) {
	if !validID(boardID) {
		return Quest{}, ErrNotFound
	}
	q, err := scanQuest(s.pool.QueryRow(ctx,
		`INSERT INTO quest (board_id, name, doc) VALUES ($1, $2, $3) RETURNING `+questColumns,
		boardID, name, orEmptyObject(doc)))
	if pgCode(err) == fkViolation {
		return Quest{}, ErrNotFound
	}
	return q, err
}

// GetQuest loads a quest by id.
func (s *Store) GetQuest(ctx context.Context, id string) (Quest, error) {
	if !validID(id) {
		return Quest{}, ErrNotFound
	}
	return scanQuest(s.pool.QueryRow(ctx, `SELECT `+questColumns+` FROM quest WHERE id = $1`, id))
}

// ListQuests returns quests, optionally only those on one board ("" for all).
func (s *Store) ListQuests(ctx context.Context, boardID string) ([]QuestSummary, error) {
	// Like boards, quests (every board's, when boardID is "") are kept to the
	// viewer's own.
	query := `SELECT q.id::text, q.board_id::text, q.name, q.updated_at FROM quest q JOIN board b ON b.id = q.board_id
		WHERE ($1::uuid IS NULL OR b.owner_id = $1)`
	args := []any{filterParam(ctx)}
	if boardID != "" {
		if !validID(boardID) {
			return []QuestSummary{}, nil
		}
		query += ` AND q.board_id = $2`
		args = append(args, boardID)
	}
	rows, err := s.pool.Query(ctx, query+` ORDER BY q.updated_at DESC, q.id`, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (QuestSummary, error) {
		var q QuestSummary
		err := row.Scan(&q.ID, &q.BoardID, &q.Name, &q.UpdatedAt)
		return q, err
	})
}

// UpdateQuest replaces a quest's name and layers.
func (s *Store) UpdateQuest(ctx context.Context, id, name string, doc json.RawMessage) (Quest, error) {
	if !validID(id) {
		return Quest{}, ErrNotFound
	}
	return scanQuest(s.pool.QueryRow(ctx,
		`UPDATE quest SET name = $2, doc = $3, updated_at = now() WHERE id = $1 RETURNING `+questColumns,
		id, name, orEmptyObject(doc)))
}

// DeleteQuest removes a quest. Sessions keep their frozen copy of it.
func (s *Store) DeleteQuest(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM quest WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Campaigns ---

// Campaign groups heroes and the sessions played with them.
type Campaign struct {
	ID     string
	Name   string
	Heroes json.RawMessage
	// Gold is the party's purse between quests.
	Gold      int
	CreatedAt time.Time
	UpdatedAt time.Time
}

const campaignColumns = `id::text, name, heroes, gold, created_at, updated_at`

func scanCampaign(row pgx.Row) (Campaign, error) {
	var c Campaign
	err := row.Scan(&c.ID, &c.Name, &c.Heroes, &c.Gold, &c.CreatedAt, &c.UpdatedAt)
	return c, notFoundIfNoRows(err)
}

func orEmptyArray(doc json.RawMessage) json.RawMessage {
	if len(doc) == 0 {
		return json.RawMessage(`[]`)
	}
	return doc
}

// CreateCampaign stores a new campaign.
func (s *Store) CreateCampaign(ctx context.Context, name string, heroes json.RawMessage) (Campaign, error) {
	return scanCampaign(s.pool.QueryRow(ctx,
		`INSERT INTO campaign (name, heroes, owner_id) VALUES ($1, $2, $3) RETURNING `+campaignColumns,
		name, orEmptyArray(heroes), ownerParam(ctx)))
}

// GetCampaign loads a campaign by id.
func (s *Store) GetCampaign(ctx context.Context, id string) (Campaign, error) {
	if !validID(id) {
		return Campaign{}, ErrNotFound
	}
	return scanCampaign(s.pool.QueryRow(ctx, `SELECT `+campaignColumns+` FROM campaign WHERE id = $1`, id))
}

// ListCampaigns returns every campaign, most recently updated first.
func (s *Store) ListCampaigns(ctx context.Context) ([]Campaign, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+campaignColumns+` FROM campaign
		WHERE $1::uuid IS NULL OR owner_id = $1 ORDER BY updated_at DESC, id`, filterParam(ctx))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Campaign, error) {
		return scanCampaign(row)
	})
}

// UpdateCampaign replaces a campaign's name and heroes.
func (s *Store) UpdateCampaign(ctx context.Context, id, name string, heroes json.RawMessage) (Campaign, error) {
	if !validID(id) {
		return Campaign{}, ErrNotFound
	}
	return scanCampaign(s.pool.QueryRow(ctx,
		`UPDATE campaign SET name = $2, heroes = $3, updated_at = now() WHERE id = $1 RETURNING `+campaignColumns,
		id, name, orEmptyArray(heroes)))
}

// SetCampaignGold replaces the party's purse.
func (s *Store) SetCampaignGold(ctx context.Context, id string, gold int) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE campaign SET gold = $2, updated_at = now() WHERE id = $1`, id, gold)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Campaign chapters ---

// Chapter is one quest in a campaign's play order, with its board.
type Chapter struct {
	CampaignID   string
	CampaignName string
	QuestID      string
	QuestName    string
	BoardID      string
	BoardName    string
	Position     int
}

const chapterQuery = `SELECT c.id::text, c.name, q.id::text, q.name, b.id::text, b.name, cc.position
	FROM campaign_chapter cc
	JOIN campaign c ON c.id = cc.campaign_id
	JOIN quest q ON q.id = cc.quest_id
	JOIN board b ON b.id = q.board_id`

func collectChapters(rows pgx.Rows) ([]Chapter, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Chapter, error) {
		var ch Chapter
		err := row.Scan(&ch.CampaignID, &ch.CampaignName, &ch.QuestID, &ch.QuestName, &ch.BoardID, &ch.BoardName, &ch.Position)
		return ch, err
	})
}

// ListChapters returns a campaign's chapters in play order.
func (s *Store) ListChapters(ctx context.Context, campaignID string) ([]Chapter, error) {
	if !validID(campaignID) {
		return []Chapter{}, nil
	}
	rows, err := s.pool.Query(ctx, chapterQuery+` WHERE cc.campaign_id = $1 ORDER BY cc.position`, campaignID)
	if err != nil {
		return nil, err
	}
	return collectChapters(rows)
}

// ListAllChapters returns every campaign's chapters, grouped by campaign
// (by name) and in play order within each.
func (s *Store) ListAllChapters(ctx context.Context) ([]Chapter, error) {
	rows, err := s.pool.Query(ctx, chapterQuery+` WHERE $1::uuid IS NULL OR c.owner_id = $1 ORDER BY lower(c.name), c.id, cc.position`, filterParam(ctx))
	if err != nil {
		return nil, err
	}
	return collectChapters(rows)
}

// SetChapters replaces a campaign's chapters with questIDs, in that order. It
// returns ErrNotFound (and changes nothing) if a quest or the campaign does
// not exist.
func (s *Store) SetChapters(ctx context.Context, campaignID string, questIDs []string) error {
	if !validID(campaignID) {
		return ErrNotFound
	}
	for _, id := range questIDs {
		if !validID(id) {
			return ErrNotFound
		}
	}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM campaign_chapter WHERE campaign_id = $1`, campaignID); err != nil {
			return err
		}
		for i, id := range questIDs {
			if _, err := tx.Exec(ctx,
				`INSERT INTO campaign_chapter (campaign_id, quest_id, position) VALUES ($1, $2, $3)`,
				campaignID, id, i+1); err != nil {
				return err
			}
		}
		return nil
	})
	switch pgCode(err) {
	case fkViolation:
		return ErrNotFound
	case uniqueViolation:
		return errors.New("store: a quest can only be one chapter of a campaign")
	}
	return err
}

// --- Custom monsters ---

// CustomMonster is a monster type kept in the database: a GM-made one (Doc
// holds its color, size and stats) or a base game one (CatalogID set, e.g.
// "orc"; Doc is its content.MonsterDef) imported from content/monsters.
type CustomMonster struct {
	ID        string
	Name      string
	Doc       json.RawMessage
	CatalogID string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const customMonsterColumns = `id::text, name, doc, coalesce(catalog_id, ''), created_at, updated_at`

func scanCustomMonster(row pgx.Row) (CustomMonster, error) {
	var m CustomMonster
	err := row.Scan(&m.ID, &m.Name, &m.Doc, &m.CatalogID, &m.CreatedAt, &m.UpdatedAt)
	return m, notFoundIfNoRows(err)
}

// CreateCustomMonster stores a new custom monster.
func (s *Store) CreateCustomMonster(ctx context.Context, name string, doc json.RawMessage) (CustomMonster, error) {
	return scanCustomMonster(s.pool.QueryRow(ctx,
		`INSERT INTO custom_monster (name, doc, owner_id) VALUES ($1, $2, $3) RETURNING `+customMonsterColumns,
		name, orEmptyObject(doc), ownerParam(ctx)))
}

// GetCustomMonster loads a monster type kept in the database by its row id.
func (s *Store) GetCustomMonster(ctx context.Context, id string) (CustomMonster, error) {
	if !validID(id) {
		return CustomMonster{}, ErrNotFound
	}
	return scanCustomMonster(s.pool.QueryRow(ctx, `SELECT `+customMonsterColumns+` FROM custom_monster WHERE id = $1`, id))
}

// ListCustomMonsters returns the viewer's monsters and the base game's (which
// everyone sees), by name.
func (s *Store) ListCustomMonsters(ctx context.Context) ([]CustomMonster, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+customMonsterColumns+` FROM custom_monster
		WHERE $1::uuid IS NULL OR owner_id = $1 OR catalog_id IS NOT NULL ORDER BY lower(name), id`, filterParam(ctx))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (CustomMonster, error) {
		return scanCustomMonster(row)
	})
}

// UpdateCustomMonster replaces a GM-made monster's name and doc. Base game
// monsters are read-only here (not found); UpsertCatalogMonster imports them.
func (s *Store) UpdateCustomMonster(ctx context.Context, id, name string, doc json.RawMessage) (CustomMonster, error) {
	if !validID(id) {
		return CustomMonster{}, ErrNotFound
	}
	return scanCustomMonster(s.pool.QueryRow(ctx,
		`UPDATE custom_monster SET name = $2, doc = $3, updated_at = now()
		 WHERE id = $1 AND catalog_id IS NULL RETURNING `+customMonsterColumns,
		id, name, orEmptyObject(doc)))
}

// DeleteCustomMonster removes a GM-made monster (base game monsters are not
// found). Quests that placed it keep the placement, and running sessions keep
// their copy of its size and color.
func (s *Store) DeleteCustomMonster(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM custom_monster WHERE id = $1 AND catalog_id IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Sessions and events ---

// Session is one quest being played, with its complete current state.
type Session struct {
	ID         string
	CampaignID string
	QuestID    *string
	Name       string
	Status     string
	State      json.RawMessage
	EventSeq   int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	// Open sessions are listed in the lobby for players to join.
	Open bool
}

// SessionSummary is a session without its state document.
type SessionSummary struct {
	ID         string
	CampaignID string
	QuestID    *string
	// VisitedQuestIDs lists every quest (map) the session has played, when it
	// traveled between maps; empty for single-map sessions.
	VisitedQuestIDs []string
	Name            string
	Status          string
	EventSeq        int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Open            bool
	// Started: online play has started (the rules are on).
	Started bool
}

const sessionColumns = `id::text, campaign_id::text, quest_id::text, name, status, state, event_seq, created_at, updated_at, open`

func scanSession(row pgx.Row) (Session, error) {
	var ss Session
	err := row.Scan(&ss.ID, &ss.CampaignID, &ss.QuestID, &ss.Name, &ss.Status, &ss.State, &ss.EventSeq, &ss.CreatedAt, &ss.UpdatedAt, &ss.Open)
	return ss, notFoundIfNoRows(err)
}

// CreateSession starts a session. questID may be "" when there is no quest.
func (s *Store) CreateSession(ctx context.Context, campaignID, questID, name string, state json.RawMessage) (Session, error) {
	if !validID(campaignID) || (questID != "" && !validID(questID)) {
		return Session{}, ErrNotFound
	}
	var quest *string
	if questID != "" {
		quest = &questID
	}
	ss, err := scanSession(s.pool.QueryRow(ctx,
		`INSERT INTO game_session (campaign_id, quest_id, name, state) VALUES ($1, $2, $3, $4) RETURNING `+sessionColumns,
		campaignID, quest, name, orEmptyObject(state)))
	if pgCode(err) == fkViolation {
		return Session{}, ErrNotFound
	}
	return ss, err
}

// GetSession loads a session and its current state.
func (s *Store) GetSession(ctx context.Context, id string) (Session, error) {
	if !validID(id) {
		return Session{}, ErrNotFound
	}
	return scanSession(s.pool.QueryRow(ctx, `SELECT `+sessionColumns+` FROM game_session WHERE id = $1`, id))
}

// ListSessions returns a campaign's sessions, active first, then most recent.
func (s *Store) ListSessions(ctx context.Context, campaignID string) ([]SessionSummary, error) {
	if !validID(campaignID) {
		return []SessionSummary{}, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, campaign_id::text, quest_id::text,
		        jsonb_path_query_array(state, '$.questId') || jsonb_path_query_array(state, '$.otherMaps[*].questId'),
		        name, status, event_seq, created_at, updated_at, open, state ? 'rules'
		 FROM game_session WHERE campaign_id = $1
		 ORDER BY (status = 'active') DESC, updated_at DESC, id`, campaignID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (SessionSummary, error) {
		var ss SessionSummary
		err := row.Scan(&ss.ID, &ss.CampaignID, &ss.QuestID, &ss.VisitedQuestIDs, &ss.Name, &ss.Status, &ss.EventSeq, &ss.CreatedAt, &ss.UpdatedAt, &ss.Open, &ss.Started)
		return ss, err
	})
}

// SetSessionOpen opens a session to players (it shows in the lobby) or
// closes it.
func (s *Store) SetSessionOpen(ctx context.Context, id string, open bool) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE game_session SET open = $2, updated_at = now() WHERE id = $1`, id, open)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// OpenSession is a game in the lobby.
type OpenSession struct {
	ID           string
	Name         string
	CampaignID   string
	CampaignName string
	// GMName is the campaign owner's name ("" for a campaign from before
	// sign-in).
	GMName string
	// Started: the GM has started online play (the rules are on).
	Started   bool
	UpdatedAt time.Time
}

// ListOpenSessions lists the active sessions open to players, for everyone
// signed in, most recently updated first.
func (s *Store) ListOpenSessions(ctx context.Context) ([]OpenSession, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT g.id::text, g.name, c.id::text, c.name, coalesce(u.display_name, ''), g.state ? 'rules', g.updated_at
		 FROM game_session g JOIN campaign c ON c.id = g.campaign_id LEFT JOIN app_user u ON u.id = c.owner_id
		 WHERE g.open AND g.status = 'active'
		 ORDER BY g.updated_at DESC, g.id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (OpenSession, error) {
		var o OpenSession
		err := row.Scan(&o.ID, &o.Name, &o.CampaignID, &o.CampaignName, &o.GMName, &o.Started, &o.UpdatedAt)
		return o, err
	})
}

// SetSessionStatus marks a session active or completed.
func (s *Store) SetSessionStatus(ctx context.Context, id, status string) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE game_session SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// NewEvent describes a change to record.
type NewEvent struct {
	Round   int
	Kind    string
	Summary string
	Payload json.RawMessage
	// PlayerSummary is the player screen's line ("" for none).
	PlayerSummary string
	// PlayerSpotted lists the monsters the players just saw (a JSON array of
	// {id, name, map}; nil for none).
	PlayerSpotted json.RawMessage
	// RetractSpotted takes a monster's sightings out of earlier events (it
	// was removed, never really there).
	RetractSpotted *SpottedRef
}

// SpottedRef names one monster's sightings: its id on the map of quest Map.
type SpottedRef struct {
	ID  string
	Map string
}

// Event is a recorded, ordered change.
type Event struct {
	SessionID string
	Seq       int64
	Round     int
	Kind      string
	Summary   string
	Payload   json.RawMessage
	CreatedAt time.Time
	// PlayerSummary is the player screen's line ("" for none).
	PlayerSummary string
	// PlayerSpotted lists the monsters the players saw ("[]" for none).
	PlayerSpotted json.RawMessage
}

const eventColumns = `session_id::text, seq, round, kind, summary, payload, created_at, player_summary, player_spotted`

func scanEvent(row pgx.Row) (Event, error) {
	var e Event
	err := row.Scan(&e.SessionID, &e.Seq, &e.Round, &e.Kind, &e.Summary, &e.Payload, &e.CreatedAt, &e.PlayerSummary, &e.PlayerSpotted)
	return e, err
}

// RecordEvent atomically replaces the session's state and appends the event
// that produced it. Either both happen or neither does.
func (s *Store) RecordEvent(ctx context.Context, sessionID string, state json.RawMessage, ev NewEvent) (Event, error) {
	if !validID(sessionID) {
		return Event{}, ErrNotFound
	}
	var out Event
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var seq int64
		err := tx.QueryRow(ctx,
			`UPDATE game_session SET state = $2, event_seq = event_seq + 1, updated_at = now()
			 WHERE id = $1 RETURNING event_seq`, sessionID, orEmptyObject(state)).Scan(&seq)
		if err != nil {
			return notFoundIfNoRows(err)
		}
		if r := ev.RetractSpotted; r != nil {
			// Only the GM's log is history; the players' lines follow what was really there.
			_, err = tx.Exec(ctx,
				`UPDATE session_event SET player_spotted = (
				   SELECT coalesce(jsonb_agg(m ORDER BY i), '[]') FROM jsonb_array_elements(player_spotted) WITH ORDINALITY AS x(m, i)
				   WHERE NOT (m->>'id' = $2 AND coalesce(m->>'map', '') = $3))
				 WHERE session_id = $1 AND player_spotted @> jsonb_build_array(jsonb_build_object('id', $2::text))`,
				sessionID, r.ID, r.Map)
			if err != nil {
				return err
			}
		}
		out, err = scanEvent(tx.QueryRow(ctx,
			`INSERT INTO session_event (session_id, seq, round, kind, summary, payload, player_summary, player_spotted)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING `+eventColumns,
			sessionID, seq, ev.Round, ev.Kind, ev.Summary, orEmptyObject(ev.Payload), ev.PlayerSummary, orEmptyArray(ev.PlayerSpotted)))
		return err
	})
	if err != nil {
		return Event{}, err
	}
	return out, nil
}

// ListEvents returns a session's events with seq > afterSeq, oldest first.
func (s *Store) ListEvents(ctx context.Context, sessionID string, afterSeq int64, limit int) ([]Event, error) {
	if !validID(sessionID) {
		return []Event{}, nil
	}
	if limit <= 0 {
		limit = defaultEventLimit
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+eventColumns+`
		 FROM session_event WHERE session_id = $1 AND seq > $2 ORDER BY seq LIMIT $3`,
		sessionID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Event, error) {
		return scanEvent(row)
	})
}

// ListPlayerEvents returns a session's latest events that have a player
// screen line or sighting (at most limit), oldest first.
func (s *Store) ListPlayerEvents(ctx context.Context, sessionID string, limit int) ([]Event, error) {
	if !validID(sessionID) {
		return []Event{}, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT * FROM (SELECT `+eventColumns+`
		 FROM session_event WHERE session_id = $1 AND (player_summary <> '' OR player_spotted <> '[]') ORDER BY seq DESC LIMIT $2) latest
		 ORDER BY seq`,
		sessionID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Event, error) {
		return scanEvent(row)
	})
}
