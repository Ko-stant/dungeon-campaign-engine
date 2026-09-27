// Package storetest provides a Store bound to a throwaway, migrated schema for
// tests that need a real database.
package storetest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

// New returns a Store bound to a fresh, migrated schema that is dropped when
// the test ends, plus that schema's connection URL. It skips the test unless
// TEST_DATABASE_URL (or DATABASE_URL) is set, e.g. via `make test-db`.
func New(t testing.TB) (*store.Store, string) {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		base = os.Getenv("DATABASE_URL")
	}
	if base == "" {
		t.Skip("set TEST_DATABASE_URL (or run make test-db) to run database tests")
	}
	ctx := context.Background()

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	schema := "test_" + hex.EncodeToString(suffix)

	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop schema: %v", err)
		}
		_ = admin.Close(context.Background())
	})

	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	schemaURL := u.String()

	if err := store.Migrate(ctx, schemaURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := store.Open(ctx, schemaURL)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	return s, schemaURL
}
