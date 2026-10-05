package store_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/Ko-stant/dungeon-campaign-engine/db"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func TestCampaignGold(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.CreateCampaign(ctx, "Three Plagues", nil)
	if err != nil || c.Gold != 0 {
		t.Fatalf("new campaign: %+v %v", c, err)
	}
	if err := s.SetCampaignGold(ctx, c.ID, 84); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetCampaign(ctx, c.ID); got.Gold != 84 {
		t.Fatalf("gold: %d", got.Gold)
	}
	if err := s.SetCampaignGold(ctx, c.ID, -1); err == nil {
		t.Fatal("negative gold should fail")
	}
	if err := s.SetCampaignGold(ctx, "nope", 1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad id: %v", err)
	}
}

// The party-gold migration adds up each hero's gold into the purse, for
// campaigns and sessions, and takes it off the heroes.
func TestPartyGoldMigration(t *testing.T) {
	s, url := storetest.New(t)
	ctx := context.Background()
	b, err := s.CreateBoard(ctx, "Board", 2, 2, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	q, err := s.CreateQuest(ctx, b.ID, "Quest", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CreateCampaign(ctx, "Old", nil)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := s.CreateCampaign(ctx, "Empty", nil)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := s.CreateSession(ctx, c.ID, q.ID, "Night 1", json.RawMessage(`{"round": 1, "heroes": []}`))
	if err != nil {
		t.Fatal(err)
	}

	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	migrations, _ := fs.Sub(db.Migrations, "migrations")
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 6); err != nil {
		t.Fatal(err)
	}
	// Before the purse: gold on each hero.
	if _, err := sqlDB.ExecContext(ctx, `UPDATE campaign SET heroes = '[{"id":"hero-1","name":"Grom","gold":30},{"id":"hero-2","name":"Ilsa","gold":12},{"id":"hero-3","name":"Vex"}]' WHERE id = $1`, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `UPDATE game_session SET state = '{"round":1,"heroes":[{"id":"hero-1","gold":50},{"id":"hero-2","gold":7}]}' WHERE id = $1`, sess.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 7); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetCampaign(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Gold != 42 {
		t.Fatalf("campaign purse: %d", got.Gold)
	}
	jsonEqual(t, got.Heroes, json.RawMessage(`[{"id":"hero-1","name":"Grom"},{"id":"hero-2","name":"Ilsa"},{"id":"hero-3","name":"Vex"}]`))
	if e, _ := s.GetCampaign(ctx, empty.ID); e.Gold != 0 || string(e.Heroes) != "[]" {
		t.Fatalf("empty campaign: %+v", e)
	}
	ss, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	jsonEqual(t, ss.State, json.RawMessage(`{"round":1,"gold":57,"heroes":[{"id":"hero-1"},{"id":"hero-2"}]}`))

	// Down gives the purse to the first hero.
	if _, err := provider.DownTo(ctx, 6); err != nil {
		t.Fatal(err)
	}
	var heroes, state string
	_ = sqlDB.QueryRowContext(ctx, `SELECT heroes::text FROM campaign WHERE id = $1`, c.ID).Scan(&heroes)
	_ = sqlDB.QueryRowContext(ctx, `SELECT state::text FROM game_session WHERE id = $1`, sess.ID).Scan(&state)
	jsonEqual(t, json.RawMessage(heroes), json.RawMessage(`[{"id":"hero-1","name":"Grom","gold":42},{"id":"hero-2","name":"Ilsa"},{"id":"hero-3","name":"Vex"}]`))
	jsonEqual(t, json.RawMessage(state), json.RawMessage(`{"round":1,"heroes":[{"id":"hero-1","gold":57},{"id":"hero-2"}]}`))
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
}
