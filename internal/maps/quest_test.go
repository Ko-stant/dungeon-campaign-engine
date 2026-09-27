package maps

import (
	"slices"
	"testing"
)

// testBoard is 5x3:
//
//	y0:  C  C  C  C  V
//	y1:  1  1  C  2  V
//	y2:  1  1  C  2  V
//
// C = corridor, V = void, 1/2 = rooms.
func testBoard() *Board {
	return &Board{
		Version: CurrentVersion,
		Width:   5,
		Height:  3,
		Regions: []int{
			0, 0, 0, 0, -1,
			1, 1, 0, 2, -1,
			1, 1, 0, 2, -1,
		},
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
			{ID: "d1", Edge: Edge{X: 2, Y: 1, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},   // room 1 | corridor
			{ID: "d2", Edge: Edge{X: 3, Y: 1, Orientation: Horizontal}, Kind: DoorSecret, State: DoorClosed}, // corridor | room 2
		},
		BlockedSquares: []Rect{{ID: "b1", X: 0, Y: 0, W: 2, H: 1}},
		Furniture:      []Furniture{{ID: "f1", Type: "table", X: 0, Y: 1, Rotation: 0}},
		Monsters:       []Monster{{ID: "m1", Type: "orc", X: 3, Y: 2}},
		Traps:          []Trap{{ID: "t1", Kind: "pit", X: 2, Y: 2, State: TrapHidden}},
		Notes:          []Note{{ID: "n1", Label: "A", X: 1, Y: 2, Text: "A chest of gold."}},
		StartTiles:     []Tile{{X: 1, Y: 2}},
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

func TestQuestCheckReportsAdvisoryIssues(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	q.Doors = append(q.Doors,
		Door{ID: "d-same", Edge: Edge{X: 1, Y: 1, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},  // room 1 | room 1
		Door{ID: "d-void", Edge: Edge{X: 4, Y: 1, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},  // room 2 | void
		Door{ID: "d-edge", Edge: Edge{X: 0, Y: 1, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed},  // off-board | room 1
		Door{ID: "d-far", Edge: Edge{X: 9, Y: 9, Orientation: Horizontal}, Kind: DoorNormal, State: DoorClosed}, // off the board
	)
	q.Monsters = append(q.Monsters,
		Monster{ID: "m-void", Type: "orc", X: 4, Y: 0},
		Monster{ID: "m-off", Type: "orc", X: 7, Y: 0},
	)
	q.Furniture = append(q.Furniture,
		Furniture{ID: "f-void", Type: "table", X: 3, Y: 0, Rotation: 0},    // 2x1 reaches the void tile (4,0)
		Furniture{ID: "f-turned", Type: "table", X: 3, Y: 1, Rotation: 90}, // 1x2 inside room 2: fine
		Furniture{ID: "f-unknown", Type: "harpsichord", X: 2, Y: 0},
	)
	q.Traps = append(q.Traps, Trap{ID: "t-void", Kind: "pit", X: 4, Y: 2, State: TrapHidden})
	q.StartTiles = append(q.StartTiles, Tile{X: 4, Y: 1})
	q.BlockedSquares = append(q.BlockedSquares, Rect{ID: "b-off", X: 4, Y: 2, W: 2, H: 1})

	want := []string{
		"blocked-off-board:b-off",
		"door-into-void:d-void",
		"door-off-board:d-far",
		"door-on-board-edge:d-edge",
		"door-same-region:d-same",
		"furniture-on-void:f-void",
		"furniture-unknown-type:f-unknown",
		"piece-off-board:m-off",
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
	if q.Doors == nil || q.Furniture == nil || q.Monsters == nil || q.Traps == nil || q.Notes == nil || q.StartTiles == nil || q.BlockedSquares == nil {
		t.Fatal("layers should be empty slices, not nil, so they encode as []")
	}
	if err := q.Validate(); err != nil {
		t.Fatal(err)
	}
}
