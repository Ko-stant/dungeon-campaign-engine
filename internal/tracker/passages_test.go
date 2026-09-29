package tracker

import (
	"strings"
	"testing"
)

func TestPassageRead(t *testing.T) {
	s := newState(t)

	s, ev := apply(t, s, cmd(t, "passage.read", map[string]any{"id": "Q2-03", "title": "The tithe barn"}))
	if len(s.ReadPassages) != 1 || s.ReadPassages[0] != "Q2-03" || ev.Summary != "Read aloud: Q2-03 The tithe barn" {
		t.Fatalf("read: %v %q", s.ReadPassages, ev.Summary)
	}
	// Reading it again (a replay at the table) is recorded too.
	s, ev = apply(t, s, cmd(t, "passage.read", map[string]any{"id": "Q2-03", "title": "The tithe barn"}))
	if len(s.ReadPassages) != 1 || ev.Summary != "Read aloud again: Q2-03 The tithe barn" {
		t.Fatalf("read again: %v %q", s.ReadPassages, ev.Summary)
	}
	s, ev = apply(t, s, cmd(t, "passage.read", map[string]any{"id": "Q2-03", "title": "The tithe barn", "read": false}))
	if len(s.ReadPassages) != 0 || ev.Summary != "Q2-03 The tithe barn marked as not read" {
		t.Fatalf("unread: %v %q", s.ReadPassages, ev.Summary)
	}

	_, _, cat := fixture()
	for name, c := range map[string]Command{
		"no id":         cmd(t, "passage.read", map[string]any{"title": "x"}),
		"long id":       cmd(t, "passage.read", map[string]any{"id": strings.Repeat("x", 41)}),
		"long title":    cmd(t, "passage.read", map[string]any{"id": "P-1", "title": strings.Repeat("x", 201)}),
		"not read yet":  cmd(t, "passage.read", map[string]any{"id": "P-1", "read": false}),
		"unknown field": cmd(t, "passage.read", map[string]any{"id": "P-1", "text": "x"}),
	} {
		if _, _, err := Apply(s, c, cat); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestReadPassagesSurviveTravel(t *testing.T) {
	s := newState(t)
	s, _ = apply(t, s, cmd(t, "passage.read", map[string]any{"id": "P0-01", "title": "The request"}))
	b, q, cat := fixture()
	next, _, err := Travel(s, Destination{QuestID: "quest-2", QuestName: "Next map", Board: b, Quest: q}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.ReadPassages) != 1 {
		t.Fatalf("read passages after travel: %v", next.ReadPassages)
	}
}
