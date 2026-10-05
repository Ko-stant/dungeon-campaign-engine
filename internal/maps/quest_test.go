package maps

import (
	"slices"
	"strings"
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

// trapSizes is the trap catalog stub: long_pit is 1x2; other kinds are markers.
func trapSizes(kind string) (int, int, bool) {
	if kind == "long_pit" {
		return 1, 2, true
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
		Teleports:      []Teleport{{ID: "tp1", X: 3, Y: 3, Label: "1"}},
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
	if issues := q.Check(b, sizes, trapSizes); len(issues) != 0 {
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
		"trap rotation":     func(q *Quest) { q.Traps[0].Rotation = 45 },
		"trap label":        func(q *Quest) { q.Traps[0].Label = "this label is far too long" },
		"removed trap":      func(q *Quest) { q.Traps[0].State = TrapRemoved }, // removed is live-only
		"empty blocked":     func(q *Quest) { q.BlockedSquares[0].W = 0 },
		"duplicate id":      func(q *Quest) { q.Monsters[0].ID = "d1" },
		"missing id":        func(q *Quest) { q.Notes[0].ID = "" },
		"teleport id":       func(q *Quest) { q.Teleports[0].ID = "d1" },
		"long label":        func(q *Quest) { q.Teleports[0].Label = "this label is far too long" },
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
	q.Traps = append(q.Traps,
		Trap{ID: "t-void", Kind: "pit", X: 5, Y: 1, State: TrapHidden},
		Trap{ID: "t-long-ok", Kind: "long_pit", X: 3, Y: 1, State: TrapHidden},                 // 1x2 up the corridor: fine
		Trap{ID: "t-long-void", Kind: "long_pit", X: 4, Y: 1, Rotation: 90, State: TrapHidden}, // turned 2x1 reaches the void (5,1)
		Trap{ID: "t-long-off", Kind: "long_pit", X: 3, Y: 3, State: TrapHidden},                // 1x2 from the top row leaves the board
	)
	q.Teleports = append(q.Teleports, Teleport{ID: "tp-void", X: 5, Y: 2}, Teleport{ID: "tp-off", X: 0, Y: 2})
	q.StartTiles = append(q.StartTiles, Tile{X: 5, Y: 2})
	q.ExitTiles = append(q.ExitTiles, Tile{X: 5, Y: 3}, Tile{X: 6, Y: 1})
	q.BlockedSquares = append(q.BlockedSquares, Rect{ID: "b-off", X: 5, Y: 1, W: 2, H: 1})

	want := []string{
		"blocked-off-board:b-off",
		"door-inside-room:d-same",
		"door-into-void:d-void",
		"door-off-board:d-far",
		"door-off-board:d-zero",
		"door-on-board-edge:d-edge",
		"exit-off-board:",
		"exit-on-void:",
		"furniture-on-void:f-void",
		"furniture-unknown-type:f-unknown",
		"piece-off-board:m-off",
		"piece-off-board:m-zero",
		"piece-off-board:t-long-off",
		"piece-off-board:tp-off",
		"piece-on-void:m-void",
		"piece-on-void:t-long-void",
		"piece-on-void:t-void",
		"piece-on-void:tp-void",
		"start-on-void:",
	}
	if got := issueCodes(q.Check(b, sizes, trapSizes)); !slices.Equal(got, want) {
		t.Fatalf("issues:\n got  %v\n want %v", got, want)
	}
}

func TestQuestCheckFlagsEditedBoard(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	b.Regions[4] = Corridor // repaint after the quest was saved
	if got := issueCodes(q.Check(b, sizes, trapSizes)); !slices.Contains(got, "board-changed:") {
		t.Fatalf("expected board-changed issue, got %v", got)
	}
}

func TestNewQuestIsEmptyAndBoundToBoard(t *testing.T) {
	b := testBoard()
	q := NewQuest(b)
	if q.BoardChecksum != b.Checksum() || q.Version != CurrentVersion {
		t.Fatalf("unexpected quest: %+v", q)
	}
	if q.Doors == nil || q.Furniture == nil || q.Monsters == nil || q.Traps == nil || q.Notes == nil || q.StartTiles == nil || q.ExitTiles == nil || q.Teleports == nil || q.BlockedSquares == nil {
		t.Fatal("layers should be empty slices, not nil, so they encode as []")
	}
	if err := q.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDoorsAndGatesAcrossACorridorAreFine(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	// Column x3 is corridor from y1 to y3: a gate across it and a secret door
	// behind rubble are normal HeroQuest layouts, not misplaced doors.
	q.Doors = append(q.Doors,
		Door{ID: "d-gate", Edge: Edge{X: 3, Y: 2, Orientation: Horizontal}, Kind: DoorGate, State: DoorClosed, Locked: true},
		Door{ID: "d-rubble", Edge: Edge{X: 3, Y: 3, Orientation: Horizontal}, Kind: DoorSecret, State: DoorClosed},
	)
	if issues := q.Check(b, sizes, trapSizes); len(issues) != 0 {
		t.Fatalf("Check: %+v", issues)
	}
}

func TestDoorInsideARoomIsFlaggedUnlessAWallIsDrawnThere(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	q.Doors = append(q.Doors, Door{ID: "d-in", Edge: Edge{X: 2, Y: 2, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed})
	issues := q.Check(b, sizes, trapSizes)
	if got := issueCodes(issues); !slices.Equal(got, []string{"door-inside-room:d-in"}) {
		t.Fatalf("issues: %v", got)
	}
	if want := "d-in is inside West at (2,2), not on a wall"; !strings.Contains(issues[0].Message, want) {
		t.Fatalf("message %q should contain %q", issues[0].Message, want)
	}
	b.DrawnWalls = []Edge{{X: 2, Y: 2, Orientation: Vertical}}
	q.BoardChecksum = b.Checksum()
	if issues := q.Check(b, sizes, trapSizes); len(issues) != 0 {
		t.Fatalf("a door on a drawn wall inside a room is fine: %+v", issues)
	}
}

func TestQuestObjectivesAreValidated(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	q.Objectives = []Objective{{Kind: ObjectiveKill}, {Kind: ObjectiveCollect, Item: "Soul Gem"}, {Kind: ObjectiveEscape}}
	if err := q.Validate(); err != nil {
		t.Fatalf("valid objectives: %v", err)
	}
	q.Objectives = []Objective{{Kind: "find"}}
	if err := q.Validate(); err == nil || !strings.Contains(err.Error(), "objective") {
		t.Errorf("an unknown kind: %v", err)
	}
	q.Objectives = []Objective{{Kind: ObjectiveKill, Monsters: []string{"nobody"}}}
	if err := q.Validate(); err == nil || !strings.Contains(err.Error(), "nobody") {
		t.Errorf("an unknown monster: %v", err)
	}
	q.Objectives = []Objective{{Kind: ObjectiveEscape, Monsters: []string{q.Monsters[0].ID}}}
	if err := q.Validate(); err == nil {
		t.Error("an escape names no monsters")
	}
	q.Objectives = []Objective{{Kind: ObjectiveCollect}}
	if err := q.Validate(); err == nil || !strings.Contains(err.Error(), "item") {
		t.Errorf("a collect objective names its item: %v", err)
	}
	q.Objectives = nil
	q.Goal = strings.Repeat("x", MaxGoal+1)
	if err := q.Validate(); err == nil || !strings.Contains(err.Error(), "goal") {
		t.Errorf("a long goal: %v", err)
	}
	q.Goal = ""
	q.Doors[0].Key = strings.Repeat("k", MaxKeyName+1)
	if err := q.Validate(); err == nil || !strings.Contains(err.Error(), "key") {
		t.Errorf("a long key name: %v", err)
	}
}

func TestCheckFlagsAnEscapeWithoutExits(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	q.ExitTiles = nil
	q.Objectives = []Objective{{Kind: ObjectiveEscape}}
	if codes := issueCodes(q.Check(b, sizes, trapSizes)); !slices.Contains(codes, "escape-without-exit:") {
		t.Errorf("issues %v", codes)
	}
	q.ExitTiles = []Tile{{X: 1, Y: 1}}
	if codes := issueCodes(q.Check(b, sizes, trapSizes)); slices.Contains(codes, "escape-without-exit:") {
		t.Errorf("with an exit: %v", codes)
	}
}
