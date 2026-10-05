package maps

import (
	"reflect"
	"testing"
)

func openTerrain(t *testing.T, b *Board) *Terrain {
	t.Helper()
	return NewTerrain(b, TerrainSpec{})
}

func reachSet(b *Board, got map[Tile]int) string {
	return tileSet(b, func(t Tile) bool { _, ok := got[t]; return ok })
}

func TestReachableCountsOrthogonalSteps(t *testing.T) {
	b := asciiBoard(t,
		".....",
		".....",
		".....",
		".....",
		".....",
	)
	got := Reachable(openTerrain(t, b), Move{From: Tile{3, 3}, Budget: 2})
	want := rows(
		"..x..",
		".xxx.",
		"xxxxx",
		".xxx.",
		"..x..",
	)
	if s := reachSet(b, got); s != want {
		t.Errorf("reachable:\n%s\nwant:\n%s", s, want)
	}
	if got[Tile{3, 3}] != 0 || got[Tile{5, 3}] != 2 || got[Tile{4, 4}] != 2 {
		t.Errorf("costs: start %d, (5,3) %d, (4,4) %d", got[Tile{3, 3}], got[Tile{5, 3}], got[Tile{4, 4}])
	}
}

func TestReachableGoesThroughOpenDoorsOnly(t *testing.T) {
	b, open, _ := twoRooms(t)
	got := Reachable(NewTerrain(b, TerrainSpec{OpenDoors: []Door{open}}), Move{From: Tile{1, 1}, Budget: 10})
	want := rows(
		"xx...",
		"xxx..",
		"xxx..",
	)
	if s := reachSet(b, got); s != want {
		t.Errorf("reachable:\n%s\nwant:\n%s", s, want)
	}
}

func TestReachablePassesAlliesButNeverStopsOnThem(t *testing.T) {
	b := asciiBoard(t, ".....")
	ally := Tile{2, 1}
	monster := Tile{4, 1}
	m := Move{
		From:     Tile{1, 1},
		Budget:   5,
		Occupied: func(t Tile) bool { return t == ally || t == monster },
		Through:  func(t Tile) bool { return t == ally },
	}
	got := Reachable(openTerrain(t, b), m)
	if s, want := reachSet(b, got), rows("x.x.."); s != want {
		t.Errorf("reachable:\n%s\nwant:\n%s", s, want)
	}
}

func TestPathIsShortestWithFixedTieBreaks(t *testing.T) {
	b := asciiBoard(t,
		"...",
		"...",
		"...",
	)
	tr := openTerrain(t, b)
	// East is tried before north, so the path goes along the bottom first.
	got, ok := Path(tr, Move{From: Tile{1, 1}, Budget: 4}, Tile{3, 3})
	want := []Tile{{2, 1}, {3, 1}, {3, 2}, {3, 3}}
	if !ok || !reflect.DeepEqual(got, want) {
		t.Errorf("Path = %v, %v; want %v", got, ok, want)
	}
	if _, ok := Path(tr, Move{From: Tile{1, 1}, Budget: 3}, Tile{3, 3}); ok {
		t.Error("a path longer than the budget should fail")
	}
	if got, ok := Path(tr, Move{From: Tile{2, 2}, Budget: 3}, Tile{2, 2}); !ok || len(got) != 0 {
		t.Errorf("a path to the start square is empty: %v, %v", got, ok)
	}
}

func TestPathAroundAWall(t *testing.T) {
	b := asciiBoard(t,
		"...",
		".#.",
		"...",
	)
	got, ok := Path(openTerrain(t, b), Move{From: Tile{2, 1}, Budget: 10}, Tile{2, 3})
	want := []Tile{{3, 1}, {3, 2}, {3, 3}, {2, 3}}
	if !ok || !reflect.DeepEqual(got, want) {
		t.Errorf("Path = %v, %v; want %v", got, ok, want)
	}
}

func TestLargePiecesNeedRoomForTheirWholeFootprint(t *testing.T) {
	b := asciiBoard(t,
		"1111.",
		"1111.",
		"1111.",
	)
	tr := openTerrain(t, b)
	got := Reachable(tr, Move{From: Tile{1, 1}, Width: 2, Height: 2, Budget: 10})
	// The anchor (bottom-left square) can reach any spot where 2x2 fits inside the room.
	want := rows(
		".....",
		"xxx..",
		"xxx..",
	)
	if s := reachSet(b, got); s != want {
		t.Errorf("reachable anchors:\n%s\nwant:\n%s", s, want)
	}
}

func TestWalkDistancesMatchesTheEncounterPlanner(t *testing.T) {
	b, open, closed := twoRooms(t)
	q := NewQuest(b)
	// The planner walks through every door, open or closed, and through
	// blocks that hide a secret door, but not through plain blocked squares.
	q.Doors = []Door{open, closed}
	q.BlockedSquares = []Rect{{ID: "r1", X: 5, Y: 1, W: 1, H: 1}, {ID: "r2", X: 4, Y: 3, W: 1, H: 1, HiddenDoor: true}}
	q.StartTiles = []Tile{{1, 1}}
	got := WalkDistances(b, q)
	want := map[Tile]int{
		{1, 1}: 0, {2, 1}: 1, {1, 2}: 1, {2, 2}: 2, {1, 3}: 2, {2, 3}: 3,
		{3, 1}: 2, {3, 2}: 3, {4, 2}: 4, {4, 1}: 5, {5, 2}: 5, {4, 3}: 5, {5, 3}: 6,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("WalkDistances = %v\nwant %v", got, want)
	}
}
