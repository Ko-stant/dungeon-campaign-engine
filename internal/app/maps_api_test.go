package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/seed"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	return testServerWith(t, nil)
}

// testServerWith is testServer with a hook to configure the app server.
func testServerWith(t *testing.T, configure func(*Server)) *httptest.Server {
	t.Helper()
	st, _ := storetest.New(t)
	catalog, err := content.Load(fstest.MapFS{
		"furniture/table.json": {Data: []byte(`{"id":"table","name":"Table","gridSize":{"width":2,"height":1}}`)},
		"monsters/orc.json":    {Data: []byte(`{"id":"orc","name":"Orc","stats":{"bodyPoints":1,"mindPoints":2}}`)},
		"heroes/elf.json":      {Data: []byte(`{"id":"elf","name":"Elf","stats":{"bodyPoints":6,"mindPoints":4}}`)},
		"traps/long_pit.json":  {Data: []byte(`{"id":"long_pit","name":"Long Pit Trap","gridSize":{"width":1,"height":2}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The catalog comes from the database, as after make import-content.
	if _, err := seed.ImportCatalog(context.Background(), st, catalog); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app := New(st)
	if configure != nil {
		configure(app)
	}
	app.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func call(t *testing.T, srv *httptest.Server, method, path string, body any) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, srv.URL+path, r)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func decode[T any](t *testing.T, data []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
	return v
}

func TestCatalogAPI(t *testing.T) {
	srv := testServer(t)
	code, data := call(t, srv, http.MethodGet, "/api/catalog", nil)
	if code != http.StatusOK {
		t.Fatalf("status %d: %s", code, data)
	}
	c := decode[content.Catalog](t, data)
	if len(c.Furniture) != 1 || c.Furniture[0].Width != 2 || len(c.Monsters) != 1 || len(c.Heroes) != 1 ||
		len(c.Traps) != 1 || c.Traps[0].Height != 2 {
		t.Fatalf("catalog: %+v", c)
	}
}

func TestBoardAPILifecycle(t *testing.T) {
	srv := testServer(t)

	code, data := call(t, srv, http.MethodPost, "/api/boards", map[string]any{"name": "Custom 24x30", "width": 24, "height": 30})
	if code != http.StatusCreated {
		t.Fatalf("create: %d %s", code, data)
	}
	created := decode[BoardResponse](t, data)
	if created.ID == "" || created.Name != "Custom 24x30" || created.Board.Width != 24 || created.Board.Height != 30 || len(created.Board.Regions) != 24*30 {
		t.Fatalf("created: %+v", created)
	}
	if created.Board.Regions[0] != maps.Void {
		t.Fatal("a new board should start as solid rock")
	}

	edited := created.Board
	edited.Regions[0] = maps.Corridor
	edited.Regions[1] = 1
	edited.Rooms = []maps.Room{{ID: 1, Name: "Guard Room"}}
	edited.DrawnWalls = []maps.Edge{{X: 5, Y: 3, Orientation: maps.Vertical}}
	code, data = call(t, srv, http.MethodPut, "/api/boards/"+created.ID, map[string]any{"name": "Winter Keep", "board": edited})
	if code != http.StatusOK {
		t.Fatalf("update: %d %s", code, data)
	}
	updated := decode[BoardResponse](t, data)
	if updated.Name != "Winter Keep" || updated.Board.Regions[1] != 1 || updated.Board.Rooms[0].Name != "Guard Room" || len(updated.Board.DrawnWalls) != 1 {
		t.Fatalf("updated: %+v", updated)
	}

	code, data = call(t, srv, http.MethodGet, "/api/boards", nil)
	list := decode[[]BoardSummary](t, data)
	if code != http.StatusOK || len(list) != 1 || list[0].Width != 24 || list[0].Name != "Winter Keep" {
		t.Fatalf("list: %d %s", code, data)
	}

	code, _ = call(t, srv, http.MethodDelete, "/api/boards/"+created.ID, nil)
	if code != http.StatusNoContent {
		t.Fatalf("delete: %d", code)
	}
	code, _ = call(t, srv, http.MethodGet, "/api/boards/"+created.ID, nil)
	if code != http.StatusNotFound {
		t.Fatalf("get after delete: %d", code)
	}
}

func TestBoardAPIRejectsInvalidInput(t *testing.T) {
	srv := testServer(t)
	for name, body := range map[string]map[string]any{
		"too big":    {"name": "x", "width": 500, "height": 10},
		"no name":    {"name": "  ", "width": 5, "height": 5},
		"zero width": {"name": "x", "width": 0, "height": 5},
	} {
		if code, data := call(t, srv, http.MethodPost, "/api/boards", body); code != http.StatusBadRequest {
			t.Errorf("%s: status %d %s", name, code, data)
		}
	}

	_, data := call(t, srv, http.MethodPost, "/api/boards", map[string]any{"name": "ok", "width": 2, "height": 2})
	b := decode[BoardResponse](t, data)
	bad := b.Board
	bad.Regions = bad.Regions[:1]
	if code, _ := call(t, srv, http.MethodPut, "/api/boards/"+b.ID, map[string]any{"name": "ok", "board": bad}); code != http.StatusBadRequest {
		t.Fatalf("invalid board update: %d", code)
	}
	onEdge := b.Board
	onEdge.DrawnWalls = []maps.Edge{{X: 1, Y: 1, Orientation: maps.Vertical}} // the board's outer edge
	if code, _ := call(t, srv, http.MethodPut, "/api/boards/"+b.ID, map[string]any{"name": "ok", "board": onEdge}); code != http.StatusBadRequest {
		t.Fatalf("drawn wall on the outer edge: %d", code)
	}
	if code, _ := call(t, srv, http.MethodGet, "/api/boards/not-an-id", nil); code != http.StatusNotFound {
		t.Fatalf("bad id: %d", code)
	}
}

func TestQuestAPILifecycle(t *testing.T) {
	srv := testServer(t)
	_, data := call(t, srv, http.MethodPost, "/api/boards", map[string]any{"name": "Board", "width": 4, "height": 2})
	b := decode[BoardResponse](t, data)
	board := b.Board
	for i := range board.Regions {
		board.Regions[i] = maps.Corridor
	}
	board.Regions[0], board.Regions[4] = 1, 1 // (1,1) and (1,2): the left column
	board.Rooms = []maps.Room{{ID: 1, Name: "Cell"}}
	call(t, srv, http.MethodPut, "/api/boards/"+b.ID, map[string]any{"name": "Board", "board": board})

	code, data := call(t, srv, http.MethodPost, "/api/boards/"+b.ID+"/quests", map[string]any{"name": "Rescue"})
	if code != http.StatusCreated {
		t.Fatalf("create quest: %d %s", code, data)
	}
	q := decode[QuestResponse](t, data)
	if q.BoardID != b.ID || q.Name != "Rescue" || len(q.Issues) != 0 {
		t.Fatalf("created quest: %+v", q)
	}

	quest := q.Quest
	quest.Doors = []maps.Door{{ID: "door-1", Edge: maps.Edge{X: 2, Y: 1, Orientation: maps.Vertical}, Kind: maps.DoorNormal, State: maps.DoorClosed}}
	quest.Furniture = []maps.Furniture{{ID: "furniture-1", Type: "table", X: 4, Y: 1}} // 2 wide: runs off the board
	code, data = call(t, srv, http.MethodPut, "/api/quests/"+q.ID, map[string]any{"name": "Rescue", "quest": quest})
	if code != http.StatusOK {
		t.Fatalf("update quest: %d %s", code, data)
	}
	saved := decode[QuestResponse](t, data)
	if len(saved.Quest.Doors) != 1 || len(saved.Issues) != 1 || saved.Issues[0].Code != "furniture-off-board" {
		t.Fatalf("saved quest should keep data and report advisory issues: %+v", saved)
	}

	code, data = call(t, srv, http.MethodGet, "/api/boards/"+b.ID+"/quests", nil)
	if code != http.StatusOK || !strings.Contains(string(data), `"Rescue"`) {
		t.Fatalf("list quests: %d %s", code, data)
	}

	quest.Doors[0].Kind = "portcullis"
	if code, _ := call(t, srv, http.MethodPut, "/api/quests/"+q.ID, map[string]any{"name": "Rescue", "quest": quest}); code != http.StatusBadRequest {
		t.Fatalf("malformed quest: %d", code)
	}

	if code, _ := call(t, srv, http.MethodDelete, "/api/boards/"+b.ID, nil); code != http.StatusConflict {
		t.Fatalf("deleting a board with quests should conflict, got %d", code)
	}
	if code, _ := call(t, srv, http.MethodDelete, "/api/quests/"+q.ID, nil); code != http.StatusNoContent {
		t.Fatalf("delete quest: %d", code)
	}
}

func TestQuestSaveRebindsToCurrentBoard(t *testing.T) {
	srv := testServer(t)
	_, data := call(t, srv, http.MethodPost, "/api/boards", map[string]any{"name": "Board", "width": 3, "height": 1})
	b := decode[BoardResponse](t, data)
	_, data = call(t, srv, http.MethodPost, "/api/boards/"+b.ID+"/quests", map[string]any{"name": "Q"})
	q := decode[QuestResponse](t, data)

	board := b.Board
	board.Regions = []int{maps.Corridor, maps.Corridor, maps.Corridor}
	call(t, srv, http.MethodPut, "/api/boards/"+b.ID, map[string]any{"name": "Board", "board": board})

	_, data = call(t, srv, http.MethodGet, "/api/quests/"+q.ID, nil)
	loaded := decode[QuestResponse](t, data)
	if len(loaded.Issues) != 1 || loaded.Issues[0].Code != "board-changed" {
		t.Fatalf("expected board-changed after editing the board: %+v", loaded.Issues)
	}

	_, data = call(t, srv, http.MethodPut, "/api/quests/"+q.ID, map[string]any{"name": "Q", "quest": loaded.Quest})
	resaved := decode[QuestResponse](t, data)
	if len(resaved.Issues) != 0 {
		t.Fatalf("saving the quest should accept the current board: %+v", resaved.Issues)
	}
}
