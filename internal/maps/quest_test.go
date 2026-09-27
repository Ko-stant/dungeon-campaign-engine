package maps

import (
	"slices"
	"testing"
)

// rowsTopFirst builds row-major regions (bottom row first) from rows written
// the way the board looks: top row first.
func rowsTopFirst(rows ...[]int) []int {
	var out []int
	for i := len(rows) - 1; i >= 0; i-- {
		out = append(out, rows[i]...)
	}
	return out
}

// testBoard is 5x3:
//
//	y3:  C  C  C  C  V
//	y2:  1  1  C  2  V
//	y1:  1  1  C  2  V
//	    x1 x2 x3 x4 x5
//
// C = corridor, V = void, 1/2 = rooms.
func testBoard() *Board {
	return &Board{
		Version: CurrentVersion,
		Width:   5,
		Height:  3,
		Regions: rowsTopFirst(
			[]int{0, 0, 0, 0, -1},
			[]int{1, 1, 0, 2, -1},
			[]int{1, 1, 0, 2, -1},
		),
		Rooms: []Room{{ID: 1, Name: "West"}, {ID: 2, Name: "East"}},
	}
}

func sizes(kind string) (int, int, bool) {
	switch kind {
	case "table":
		return 2, 1, true
	case "chest":
		return 1, 1, true
	}
	return 0, 0, false
}

func validQuest(b *Board) *Quest {
	return &Quest{
		Version:       CurrentVersion,
		BoardChecksum: b.Checksum(),
		Doors: []Door{
			{ID: "d1", Edge: Edge{X: 3, Y: 2, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},   // room 1 | corridor
			{ID: "d2", Edge: Edge{X: 4, Y: 3, Orientation: Horizontal}, Kind: DoorSecret, State: DoorClosed}, // room 2 below | corridor above
		},
		BlockedSquares: []Rect{{ID: "b1", X: 1, Y: 3, W: 2, H: 1}},
		Furniture:      []Furniture{{ID: "f1", Type: "table", X: 1, Y: 2, Rotation: 0}},
		Monsters:       []Monster{{ID: "m1", Type: "orc", X: 4, Y: 1}},
		Traps:          []Trap{{ID: "t1", Kind: "pit", X: 3, Y: 1, State: TrapHidden}},
		Notes:          []Note{{ID: "n1", Label: "A", X: 2, Y: 1, Text: "A chest of gold."}},
		StartTiles:     []Tile{{X: 2, Y: 1}},
		ExitTiles:      []Tile{{X: 1, Y: 3}},
	}
}

func issueCodes(issues []Issue) []string {
	codes := make([]string, 0, len(issues))
	for _, i := range issues {
		codes = append(codes, i.Code+":"+i.ItemID)
	}
	slices.Sort(codes)
	return codes
}

func TestValidQuestHasNoIssues(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	if err := q.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if issues := q.Check(b, sizes); len(issues) != 0 {
		t.Fatalf("Check: %+v", issues)
	}
}

func TestQuestValidateRejectsMalformedData(t *testing.T) {
	cases := map[string]func(q *Quest){
		"door orientation":  func(q *Quest) { q.Doors[0].Edge.Orientation = "diagonal" },
		"door kind":         func(q *Quest) { q.Doors[0].Kind = "portcullis" },
		"door state":        func(q *Quest) { q.Doors[0].State = "ajar" },
		"rotation":          func(q *Quest) { q.Furniture[0].Rotation = 45 },
		"trap state":        func(q *Quest) { q.Traps[0].State = "armed" },
		"empty blocked":     func(q *Quest) { q.BlockedSquares[0].W = 0 },
		"duplicate id":      func(q *Quest) { q.Monsters[0].ID = "d1" },
		"missing id":        func(q *Quest) { q.Notes[0].ID = "" },
		"furniture type":    func(q *Quest) { q.Furniture[0].Type = "" },
		"monster type":      func(q *Quest) { q.Monsters[0].Type = "" },
		"negative override": func(q *Quest) { body := -1; q.Monsters[0].Body = &body },
		"future version":    func(q *Quest) { q.Version = CurrentVersion + 1 },
		"top-left version":  func(q *Quest) { q.Version = 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			q := validQuest(testBoard())
			mutate(q)
			if err := q.Validate(); err == nil {
				t.Fatal("expected a validation error")
			}
		})
	}
}

func TestGatesAndLockedDoorsAreValid(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	q.Doors[0].Kind = DoorGate
	q.Doors[1].Locked = true
	q.BlockedSquares[0].HiddenDoor = true
	if err := q.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestQuestCheckReportsAdvisoryIssues(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	q.Doors = append(q.Doors,
		Door{ID: "d-same", Edge: Edge{X: 2, Y: 2, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},    // room 1 | room 1
		Door{ID: "d-void", Edge: Edge{X: 5, Y: 2, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},    // room 2 | void
		Door{ID: "d-edge", Edge: Edge{X: 1, Y: 2, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},    // off-board | room 1
		Door{ID: "d-far", Edge: Edge{X: 10, Y: 10, Orientation: Horizontal}, Kind: DoorNormal, State: DoorClosed}, // off the board
		Door{ID: "d-zero", Edge: Edge{X: 0, Y: 2, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},    // 0 is off a 1-based board
	)
	q.Monsters = append(q.Monsters,
		Monster{ID: "m-void", Type: "orc", X: 5, Y: 3},
		Monster{ID: "m-off", Type: "orc", X: 8, Y: 3},
		Monster{ID: "m-zero", Type: "orc", X: 1, Y: 0},
	)
	q.Furniture = append(q.Furniture,
		Furniture{ID: "f-void", Type: "table", X: 4, Y: 3, Rotation: 0},    // 2x1 reaches the void tile (5,3)
		Furniture{ID: "f-turned", Type: "table", X: 4, Y: 1, Rotation: 90}, // 1x2 up from (4,1) inside room 2: fine
		Furniture{ID: "f-unknown", Type: "harpsichord", X: 3, Y: 3},
	)
	q.Traps = append(q.Traps, Trap{ID: "t-void", Kind: "pit", X: 5, Y: 1, State: TrapHidden})
	q.StartTiles = append(q.StartTiles, Tile{X: 5, Y: 2})
	q.ExitTiles = append(q.ExitTiles, Tile{X: 5, Y: 3}, Tile{X: 6, Y: 1})
	q.BlockedSquares = append(q.BlockedSquares, Rect{ID: "b-off", X: 5, Y: 1, W: 2, H: 1})

	want := []string{
		"blocked-off-board:b-off",
		"door-into-void:d-void",
		"door-off-board:d-far",
		"door-off-board:d-zero",
		"door-on-board-edge:d-edge",
		"door-same-region:d-same",
		"exit-off-board:",
		"exit-on-void:",
		"furniture-on-void:f-void",
		"furniture-unknown-type:f-unknown",
		"piece-off-board:m-off",
		"piece-off-board:m-zero",
		"piece-on-void:m-void",
		"piece-on-void:t-void",
		"start-on-void:",
	}
	if got := issueCodes(q.Check(b, sizes)); !slices.Equal(got, want) {
		t.Fatalf("issues:\n got  %v\n want %v", got, want)
	}
}

func TestQuestCheckFlagsEditedBoard(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	b.Regions[4] = Corridor // repaint after the quest was saved
	if got := issueCodes(q.Check(b, sizes)); !slices.Contains(got, "board-changed:") {
		t.Fatalf("expected board-changed issue, got %v", got)
	}
}

func TestNewQuestIsEmptyAndBoundToBoard(t *testing.T) {
	b := testBoard()
	q := NewQuest(b)
	if q.BoardChecksum != b.Checksum() || q.Version != CurrentVersion {
		t.Fatalf("unexpected quest: %+v", q)
	}
	if q.Doors == nil || q.Furniture == nil || q.Monsters == nil || q.Traps == nil || q.Notes == nil || q.StartTiles == nil || q.ExitTiles == nil || q.BlockedSquares == nil {
		t.Fatal("layers should be empty slices, not nil, so they encode as []")
	}
	if err := q.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDoorOnADrawnCorridorWallIsFine(t *testing.T) {
	b := testBoard()
	// Column x3 is corridor from y1 to y3; draw a wall across it between y1 and y2.
	b.DrawnWalls = []Edge{{X: 3, Y: 2, Orientation: Horizontal}}
	q := validQuest(b)
	q.Doors = append(q.Doors, Door{ID: "d-drawn", Edge: Edge{X: 3, Y: 2, Orientation: Horizontal}, Kind: DoorNormal, State: DoorClosed})
	if issues := q.Check(b, sizes); len(issues) != 0 {
		t.Fatalf("Check: %+v", issues)
	}
	b.DrawnWalls = nil
	q.BoardChecksum = b.Checksum()
	if got := issueCodes(q.Check(b, sizes)); !slices.Equal(got, []string{"door-same-region:d-drawn"}) {
		t.Fatalf("without the drawn wall: %v", got)
	}
}
