package narration

import (
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/script"
)

const sampleScript = `## Prologue

### P0-01 - The square

- **When:** first scene.
- **Speakers:** Narrator, then Sergeant Hale.
- **Sound:** wagon wheels.

**Narrator**

> The wagon stops. Haldmere is
> quiet.
>
> A man crosses the square.

**Sergeant Hale**

> Welcome to Haldmere. I'm Corin.

### P0-02 - The gate

- **Speaker:** Maren Ashby.

> Keep your torches lit.

## Quest 1

### Q1-02 - The verse

- **Speaker:** Narrator.

> Three the plagues.\
> Three the stones.

**Quest goals**
- Cross the halls.

## Quest 2

### Q2-01 - Later

> Not exported.
`

const sampleCast = `# Narrative

## Cast

| Name | Say it | Who they are | Voice |
|---|---|---|---|
| Narrator | - | Tells the story | Warm, measured |
| Sergeant Corin Hale | KOR-in HAIL | Keep guard | Gruff but polite |
| Maren Ashby | MAIR-en ASH-bee | Steward | Brisk, dry |
| Tomas Reed | TOM-us REED | Lost guard | Young, hoarse |
| Brugg the Unbroken | BRUG | Champion | Silent |

## The Soul Gems

| Not | A cast | table | row |
`

func parsed(t *testing.T) script.Script {
	t.Helper()
	s, err := script.Parse(sampleScript)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseCast(t *testing.T) {
	cast := ParseCast(sampleCast)
	if len(cast) != 5 {
		t.Fatalf("got %d cast rows, want 5: %+v", len(cast), cast)
	}
	if got := cast[1]; got != (Cast{Name: "Sergeant Corin Hale", SayIt: "KOR-in HAIL", Voice: "Gruff but polite"}) {
		t.Errorf("row 1 = %+v", got)
	}
}

func TestExportWritesOnlySpokenText(t *testing.T) {
	files := Export(parsed(t), ParseCast(sampleCast), []string{"P0", "Q1"})
	byName := map[string]string{}
	var names []string
	for _, f := range files {
		byName[f.Name] = f.Text
		names = append(names, f.Name)
	}
	if strings.Join(names, ",") != "README.md,P0-01.txt,P0-02.txt,Q1-02.txt" {
		t.Fatalf("files = %v", names)
	}
	want := "The wagon stops. Haldmere is quiet.\n\nA man crosses the square.\n\nWelcome to Haldmere. I'm Corin.\n"
	if got := byName["P0-01.txt"]; got != want {
		t.Errorf("P0-01.txt = %q, want %q", got, want)
	}
	// A kept line break (the verse) stays; quest goals are table text, not spoken.
	if got := byName["Q1-02.txt"]; got != "Three the plagues.\nThree the stones.\n" {
		t.Errorf("Q1-02.txt = %q", got)
	}
}

func TestExportReadmeMapsTurnsAndPronunciation(t *testing.T) {
	readme := Export(parsed(t), ParseCast(sampleCast), []string{"P0", "Q1"})[0].Text
	for _, want := range []string{
		"## P0-01 - The square",
		"- Sound: wagon wheels.",
		`1. Narrator (2 paragraphs): "The wagon stops. Haldmere is quiet."`,
		`2. Sergeant Hale (1 paragraph): "Welcome to Haldmere. I'm Corin."`,
		// A one-speaker passage takes its speaker from the notes.
		`1. Maren Ashby (1 paragraph): "Keep your torches lit."`,
		// Voices of the speakers in the export, matched to the cast by name.
		"| Sergeant Hale | Gruff but polite |",
		"| Maren Ashby | Brisk, dry |",
		// Only names that are spoken get a pronunciation line.
		"| Sergeant Corin Hale | KOR-in HAIL |",
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("README is missing %q:\n%s", want, readme)
		}
	}
	for _, unwanted := range []string{"Q2-01", "TOM-us", "BRUG", "Cross the halls", "| Maren Ashby | MAIR-en"} {
		if strings.Contains(readme, unwanted) {
			t.Errorf("README should not contain %q", unwanted)
		}
	}
}

func TestExportWithoutFilterTakesEverything(t *testing.T) {
	if n := len(Export(parsed(t), nil, nil)); n != 5 {
		t.Errorf("got %d files, want README and 4 passages", n)
	}
}

func TestFirstWordsShortensLongParagraphs(t *testing.T) {
	long := "One two three four five six seven eight nine ten eleven twelve thirteen. More text."
	if got := firstWords(long); got != "One two three four five six seven eight nine ten…" {
		t.Errorf("firstWords = %q", got)
	}
}
