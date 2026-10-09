package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func TestUpsertCatalogMonster(t *testing.T) {
	st, _ := storetest.New(t)
	bg := context.Background()
	doc := json.RawMessage(`{"body":1,"mind":0,"attack":2,"defense":1,"movement":10,"image":"assets/tiles_cleaned/monsters/orc.png"}`)

	if got, err := st.UpsertCatalogMonster(bg, "orc", "Orc", doc); err != nil || got != store.ImportCreated {
		t.Fatalf("first import: %v %v", got, err)
	}
	if got, err := st.UpsertCatalogMonster(bg, "orc", "Orc", doc); err != nil || got != store.ImportUnchanged {
		t.Fatalf("same again: %v %v", got, err)
	}
	if got, err := st.UpsertCatalogMonster(bg, "orc", "Orc", json.RawMessage(`{"body":2}`)); err != nil || got != store.ImportUpdated {
		t.Fatalf("changed stats: %v %v", got, err)
	}

	// Everyone sees the base game's monsters, next to their own.
	gm, _ := st.SignIn(bg, store.Identity{Provider: "dev", Subject: "gm", DisplayName: "The GM"})
	friend, _ := st.SignIn(bg, store.Identity{Provider: "dev", Subject: "sam", DisplayName: "Sam"})
	asGM := store.WithViewer(bg, store.Viewer{UserID: gm.ID})
	if _, err := st.CreateCustomMonster(asGM, "Cave Ogre", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if list, _ := st.ListCustomMonsters(asGM); len(list) != 2 {
		t.Errorf("the GM's monsters: %+v", list)
	}
	list, _ := st.ListCustomMonsters(store.WithViewer(bg, store.Viewer{UserID: friend.ID}))
	if len(list) != 1 || list[0].CatalogID != "orc" || list[0].Name != "Orc" {
		t.Fatalf("a friend's monsters: %+v", list)
	}

	if got, err := st.GetCustomMonster(bg, list[0].ID); err != nil || got.CatalogID != "orc" {
		t.Fatalf("get a base monster: %+v %v", got, err)
	}
	if _, err := st.GetCustomMonster(bg, "nope"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("bad id: err = %v, want ErrNotFound", err)
	}

	// Base monsters are read-only for the monster forms.
	if _, err := st.UpdateCustomMonster(bg, list[0].ID, "Big Orc", json.RawMessage(`{}`)); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("update a base monster: err = %v, want ErrNotFound", err)
	}
	if err := st.DeleteCustomMonster(bg, list[0].ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("delete a base monster: err = %v, want ErrNotFound", err)
	}
	if _, err := st.UpsertCatalogMonster(bg, "", "Nameless", doc); err == nil {
		t.Error("a base monster needs its catalog id")
	}
}

func TestCatalogPieces(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	table := json.RawMessage(`{"width":3,"height":2,"blocksMovement":true}`)

	if got, err := st.UpsertCatalogPiece(ctx, store.PieceFurniture, "table", "Table", table); err != nil || got != store.ImportCreated {
		t.Fatalf("first import: %v %v", got, err)
	}
	if got, err := st.UpsertCatalogPiece(ctx, store.PieceFurniture, "table", "Table", table); err != nil || got != store.ImportUnchanged {
		t.Fatalf("same again: %v %v", got, err)
	}
	if got, err := st.UpsertCatalogPiece(ctx, store.PieceFurniture, "table", "Long Table", table); err != nil || got != store.ImportUpdated {
		t.Fatalf("renamed: %v %v", got, err)
	}
	// A trap may share an id with a piece of furniture: the kinds are apart.
	if _, err := st.UpsertCatalogPiece(ctx, store.PieceTrap, "table", "Table Trap", json.RawMessage(`{"width":1,"height":1}`)); err != nil {
		t.Fatal(err)
	}

	furniture, err := st.ListCatalogPieces(ctx, store.PieceFurniture)
	if err != nil || len(furniture) != 1 || furniture[0].CatalogID != "table" || furniture[0].Name != "Long Table" {
		t.Fatalf("furniture: %+v %v", furniture, err)
	}
	jsonEqual(t, furniture[0].Doc, table)
	if traps, _ := st.ListCatalogPieces(ctx, store.PieceTrap); len(traps) != 1 || traps[0].Name != "Table Trap" {
		t.Fatalf("traps: %+v", traps)
	}
	if _, err := st.UpsertCatalogPiece(ctx, "door", "x", "X", table); err == nil {
		t.Error("an unknown kind should be refused")
	}
	if _, err := st.UpsertCatalogPiece(ctx, store.PieceTrap, "", "X", table); err == nil {
		t.Error("a piece needs its catalog id")
	}
}
