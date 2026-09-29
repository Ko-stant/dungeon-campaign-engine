package script

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

const sample = `# A Campaign - Read-Aloud Script

Intro text that is not in a section.

## How this script is written

- **The quoted text is exactly what is spoken.**

| ID | Passage |
|---|---|
| P0-01 | The request |

---

## Prologue

### P0-01 - The request

- **When:** before the first quest.
- **Speaker:** Lord Voss.
- **Voice:** quiet and serious. Slows down
  for the last line.

> Come in. Close the door
> behind you.
>
> Three plagues walk my lands.

---

## Quest 1 - The Halls

Short intro for the section.

### Q1-01 - The Halls

- **When:** the quest begins.

> The doors groan shut.

**Quest goals**
- Find the eastern gate.
- *(Optional)* Search the tombs.

### Q1-02 - The verse

> Three the plagues that bar the way.\
> Three the stones that turn the key.

### Q3-05 - The great hall

- **Speakers:** Narrator, then Varnok.
- **Voice (Varnok):** calm and cold.

**Narrator**

> The hall is a war camp.

**Varnok** (optional)

> So. The little ones.

### Q3-09 - Taunts

> **a** (the fight begins) All of it mine.

> **b** (any time) I could almost thank you.
`

func TestParseSample(t *testing.T) {
	s, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Sections) != 2 {
		t.Fatalf("sections without passages are dropped: %+v", s.Sections)
	}
	pro, q1 := s.Sections[0], s.Sections[1]
	if pro.Title != "Prologue" || q1.Title != "Quest 1 - The Halls" || q1.Intro != "Short intro for the section." {
		t.Fatalf("section titles: %q %q %q", pro.Title, q1.Title, q1.Intro)
	}

	want := Passage{
		ID: "P0-01", Title: "The request",
		Notes: []Note{
			{Label: "When", Text: "before the first quest."},
			{Label: "Speaker", Text: "Lord Voss."},
			{Label: "Voice", Text: "quiet and serious. Slows down for the last line."},
		},
		Parts: []Part{{Paragraphs: []string{"Come in. Close the door behind you.", "Three plagues walk my lands."}}},
	}
	if !reflect.DeepEqual(pro.Passages[0], want) {
		t.Fatalf("P0-01:\n got  %+v\n want %+v", pro.Passages[0], want)
	}

	halls := q1.Passages[0]
	if halls.GoalsTitle != "Quest goals" || !reflect.DeepEqual(halls.Goals, []string{"Find the eastern gate.", "(Optional) Search the tombs."}) {
		t.Fatalf("goals: %q %+v", halls.GoalsTitle, halls.Goals)
	}
	if verse := q1.Passages[1].Parts[0].Paragraphs; len(verse) != 1 || verse[0] != "Three the plagues that bar the way.\nThree the stones that turn the key." {
		t.Fatalf("a trailing backslash keeps the line break: %q", verse)
	}

	hall := q1.Passages[2]
	if len(hall.Parts) != 2 || hall.Parts[0].Speaker != "Narrator" || hall.Parts[1].Speaker != "Varnok" ||
		hall.Parts[1].Aside != "optional" || hall.Parts[1].Paragraphs[0] != "So. The little ones." {
		t.Fatalf("speaker parts: %+v", hall.Parts)
	}
	if hall.Notes[1] != (Note{Label: "Voice (Varnok)", Text: "calm and cold."}) {
		t.Fatalf("labelled voice note: %+v", hall.Notes)
	}

	taunts := q1.Passages[3].Parts[0].Paragraphs
	if !reflect.DeepEqual(taunts, []string{"a (the fight begins) All of it mine.", "b (any time) I could almost thank you."}) {
		t.Fatalf("bold markers are dropped and separate quotes are paragraphs: %q", taunts)
	}
	if s.Count() != 5 {
		t.Fatalf("count %d", s.Count())
	}
}

func TestParseRejects(t *testing.T) {
	for name, text := range map[string]string{
		"duplicate id":  "## A\n\n### P-1 - One\n\n> Hi.\n\n### P-1 - Two\n\n> Hi.\n",
		"no text":       "## A\n\n### P-1 - One\n\n- **When:** now.\n",
		"no title":      "## A\n\n### P-1\n\n> Hi.\n",
		"long id":       "## A\n\n### " + strings.Repeat("x", MaxIDLength+1) + " - One\n\n> Hi.\n",
		"too big":       strings.Repeat("x", MaxBytes+1),
		"no passages":   "## A\n\nJust words.\n",
		"outside quote": "## A\n\n### P-1 - One\n\n> Hi.\n\nStray words that belong nowhere.\n",
	} {
		if _, err := Parse(text); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if s, err := Parse("  \n"); err != nil || len(s.Sections) != 0 {
		t.Fatalf("an empty script is allowed (no script yet): %+v %v", s, err)
	}
}

func TestParseErrorsNameTheLine(t *testing.T) {
	_, err := Parse("## A\n\n### P-1 - One\n\n> Hi.\n\nStray words.\n")
	if err == nil || !strings.Contains(err.Error(), "line 7") {
		t.Fatalf("error should name the line: %v", err)
	}
}

// The campaign's real script must always parse.
func TestParseThreePlaguesScript(t *testing.T) {
	data, err := os.ReadFile("../../docs/campaigns/three-plagues/SCRIPT.md")
	if err != nil {
		t.Skip("script not found:", err)
	}
	s, err := Parse(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if s.Count() != 26 {
		t.Fatalf("passages: %d", s.Count())
	}
	var titles []string
	for _, sec := range s.Sections {
		titles = append(titles, sec.Title)
	}
	want := []string{"Prologue", "Quest 1 - The Crumbling Halls", "Quest 2 - The Bloated Fields", "Quest 3 - The Wardens' Rise", "Soul Gem moments (optional)", "The End"}
	if !reflect.DeepEqual(titles, want) {
		t.Fatalf("sections: %q", titles)
	}
	verse := s.Sections[1].Passages[1].Parts[0].Paragraphs
	if last := verse[len(verse)-1]; !strings.Contains(last, "bar the way.\nThree the stones") {
		t.Fatalf("the verse keeps its lines: %q", last)
	}
}

func TestNotesDropCodeMarks(t *testing.T) {
	s, err := Parse("## A\n\n### P-1 - One\n\n- **Voice:** clips `P-1a` to `P-1e`.\n\n> Hi.\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Sections[0].Passages[0].Notes[0].Text; got != "clips P-1a to P-1e." {
		t.Fatalf("note: %q", got)
	}
}
