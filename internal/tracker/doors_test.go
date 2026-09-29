package tracker

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
)

func TestExitDoorIsNamedInTheLog(t *testing.T) {
	b, q, cat := fixture()
	q.Doors = append(q.Doors, maps.Door{ID: "door-9", Edge: maps.Edge{X: 7, Y: 1, Orientation: maps.Vertical}, Kind: maps.DoorExit, State: maps.DoorClosed, Span: 2})
	s, err := NewSession(b, q, "The Test", party(), cat)
	if err != nil {
		t.Fatal(err)
	}
	if d := s.Doors[len(s.Doors)-1]; d.ID != "door-9" || !d.Found {
		t.Fatalf("an exit door starts found: %+v", d)
	}
	_, ev := apply(t, s, cmd(t, "door.set", map[string]any{"id": "door-9", "state": "open"}))
	if ev.Summary != "Opened exit door door-9" {
		t.Fatalf("summary: %q", ev.Summary)
	}
}
