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
		`INSERT INTO board (name, width, height, doc) VALUES ($1, $2, $3, $4) RETURNING `+boardColumns,
		name, width, height, orEmptyObject(doc)))
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
	rows, err := s.pool.Query(ctx, `SELECT id::text, name, width, height, updated_at FROM board ORDER BY updated_at DESC, id`)
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
	query := `SELECT id::text, board_id::text, name, updated_at FROM quest`
	var args []any
	if boardID != "" {
		if !validID(boardID) {
			return []QuestSummary{}, nil
		}
		query += ` WHERE board_id = $1`
		args = append(args, boardID)
	}
	rows, err := s.pool.Query(ctx, query+` ORDER BY updated_at DESC, id`, args...)
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
	ID        string
	Name      string
	Heroes    json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

const campaignColumns = `id::text, name, heroes, created_at, updated_at`

func scanCampaign(row pgx.Row) (Campaign, error) {
	var c Campaign
	err := row.Scan(&c.ID, &c.Name, &c.Heroes, &c.CreatedAt, &c.UpdatedAt)
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
		`INSERT INTO campaign (name, heroes) VALUES ($1, $2) RETURNING `+campaignColumns,
		name, orEmptyArray(heroes)))
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
	rows, err := s.pool.Query(ctx, `SELECT `+campaignColumns+` FROM campaign ORDER BY updated_at DESC, id`)
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

// --- Custom monsters ---

// CustomMonster is a GM-made monster type. Doc holds its color, size and stats.
type CustomMonster struct {
	ID        string
	Name      string
	Doc       json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

const customMonsterColumns = `id::text, name, doc, created_at, updated_at`

func scanCustomMonster(row pgx.Row) (CustomMonster, error) {
	var m CustomMonster
	err := row.Scan(&m.ID, &m.Name, &m.Doc, &m.CreatedAt, &m.UpdatedAt)
	return m, notFoundIfNoRows(err)
}

// CreateCustomMonster stores a new custom monster.
func (s *Store) CreateCustomMonster(ctx context.Context, name string, doc json.RawMessage) (CustomMonster, error) {
	return scanCustomMonster(s.pool.QueryRow(ctx,
		`INSERT INTO custom_monster (name, doc) VALUES ($1, $2) RETURNING `+customMonsterColumns,
		name, orEmptyObject(doc)))
}

// ListCustomMonsters returns every custom monster by name.
func (s *Store) ListCustomMonsters(ctx context.Context) ([]CustomMonster, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+customMonsterColumns+` FROM custom_monster ORDER BY lower(name), id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (CustomMonster, error) {
		return scanCustomMonster(row)
	})
}

// UpdateCustomMonster replaces a custom monster's name and doc.
func (s *Store) UpdateCustomMonster(ctx context.Context, id, name string, doc json.RawMessage) (CustomMonster, error) {
	if !validID(id) {
		return CustomMonster{}, ErrNotFound
	}
	return scanCustomMonster(s.pool.QueryRow(ctx,
		`UPDATE custom_monster SET name = $2, doc = $3, updated_at = now() WHERE id = $1 RETURNING `+customMonsterColumns,
		id, name, orEmptyObject(doc)))
}

// DeleteCustomMonster removes a custom monster. Quests that placed it keep
// the placement, and running sessions keep their copy of its size and color.
func (s *Store) DeleteCustomMonster(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM custom_monster WHERE id = $1`, id)
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
}

// SessionSummary is a session without its state document.
type SessionSummary struct {
	ID         string
	CampaignID string
	QuestID    *string
	Name       string
	Status     string
	EventSeq   int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const sessionColumns = `id::text, campaign_id::text, quest_id::text, name, status, state, event_seq, created_at, updated_at`

func scanSession(row pgx.Row) (Session, error) {
	var ss Session
	err := row.Scan(&ss.ID, &ss.CampaignID, &ss.QuestID, &ss.Name, &ss.Status, &ss.State, &ss.EventSeq, &ss.CreatedAt, &ss.UpdatedAt)
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
		`SELECT id::text, campaign_id::text, quest_id::text, name, status, event_seq, created_at, updated_at
		 FROM game_session WHERE campaign_id = $1
		 ORDER BY (status = 'active') DESC, updated_at DESC, id`, campaignID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (SessionSummary, error) {
		var ss SessionSummary
		err := row.Scan(&ss.ID, &ss.CampaignID, &ss.QuestID, &ss.Name, &ss.Status, &ss.EventSeq, &ss.CreatedAt, &ss.UpdatedAt)
		return ss, err
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
		return tx.QueryRow(ctx,
			`INSERT INTO session_event (session_id, seq, round, kind, summary, payload)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING session_id::text, seq, round, kind, summary, payload, created_at`,
			sessionID, seq, ev.Round, ev.Kind, ev.Summary, orEmptyObject(ev.Payload)).
			Scan(&out.SessionID, &out.Seq, &out.Round, &out.Kind, &out.Summary, &out.Payload, &out.CreatedAt)
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
		`SELECT session_id::text, seq, round, kind, summary, payload, created_at
		 FROM session_event WHERE session_id = $1 AND seq > $2 ORDER BY seq LIMIT $3`,
		sessionID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Event, error) {
		var e Event
		err := row.Scan(&e.SessionID, &e.Seq, &e.Round, &e.Kind, &e.Summary, &e.Payload, &e.CreatedAt)
		return e, err
	})
}
