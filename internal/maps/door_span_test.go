package maps

import (
	"slices"
	"testing"
)

func TestDoorEdges(t *testing.T) {
	v := Door{Edge: Edge{X: 3, Y: 1, Orientation: Vertical}, Span: 2}
	if got := v.Edges(); !slices.Equal(got, []Edge{{X: 3, Y: 1, Orientation: Vertical}, {X: 3, Y: 2, Orientation: Vertical}}) {
		t.Fatalf("vertical span 2 runs upward: %v", got)
	}
	h := Door{Edge: Edge{X: 1, Y: 3, Orientation: Horizontal}, Span: 2}
	if got := h.Edges(); !slices.Equal(got, []Edge{{X: 1, Y: 3, Orientation: Horizontal}, {X: 2, Y: 3, Orientation: Horizontal}}) {
		t.Fatalf("horizontal span 2 runs right: %v", got)
	}
	if got := (Door{Edge: Edge{X: 1, Y: 1, Orientation: Vertical}}).Edges(); len(got) != 1 {
		t.Fatalf("no span means one edge: %v", got)
	}
}

func TestDoorSpanValidation(t *testing.T) {
	for _, span := range []int{-1, 3} {
		q := validQuest(testBoard())
		q.Doors[0].Span = span
		if err := q.Validate(); err == nil {
			t.Errorf("span %d should be refused", span)
		}
	}
	q := validQuest(testBoard())
	q.Doors[0].Span = 2
	q.Doors[0].Kind = DoorExit
	if err := q.Validate(); err != nil {
		t.Fatalf("a 2-wide exit door is valid: %v", err)
	}
}

func TestWideAndExitDoorChecks(t *testing.T) {
	b := testBoard()
	q := validQuest(b)
	q.Doors = []Door{
		// Two wall edges between room 1 and the corridor: fine.
		{ID: "d-wide", Edge: Edge{X: 3, Y: 1, Orientation: Vertical}, Kind: DoorGate, State: DoorClosed, Span: 2},
		// The second half leaves the top of the board.
		{ID: "d-wide-off", Edge: Edge{X: 4, Y: 3, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed, Span: 2},
		// Both halves are in the middle of room 1.
		{ID: "d-wide-in", Edge: Edge{X: 2, Y: 1, Orientation: Vertical}, Kind: DoorNormal, State: DoorClosed, Span: 2},
		// Exit doors belong on the board's edge or against solid rock: no warning.
		{ID: "x-edge", Edge: Edge{X: 1, Y: 1, Orientation: Vertical}, Kind: DoorExit, State: DoorClosed, Span: 2},
		{ID: "x-top", Edge: Edge{X: 1, Y: 4, Orientation: Horizontal}, Kind: DoorExit, State: DoorClosed},
		{ID: "x-rock", Edge: Edge{X: 5, Y: 2, Orientation: Vertical}, Kind: DoorExit, State: DoorClosed},
		// Off the board is still off the board.
		{ID: "x-off", Edge: Edge{X: 9, Y: 9, Orientation: Vertical}, Kind: DoorExit, State: DoorClosed},
	}
	want := []string{"door-inside-room:d-wide-in", "door-off-board:d-wide-off", "door-off-board:x-off"}
	if got := issueCodes(q.Check(b, sizes, trapSizes)); !slices.Equal(got, want) {
		t.Fatalf("issues:\n got  %v\n want %v", got, want)
	}
}
