// Package narration turns a read-aloud script into paste-ready text for a
// text-to-speech tool: one <passage id>.txt per passage holding only the
// spoken words, and a README.md with each passage's notes, its speaker turns
// in order (to assign voices paragraph by paragraph) and the pronunciation of
// the names that are spoken.
package narration

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/script"
)

// Cast is one row of a narrative's cast table.
type Cast struct {
	Name, SayIt, Voice string
}

// File is one exported file.
type File struct {
	Name, Text string
}

// ParseCast reads the table under the "## Cast" heading of a narrative
// (columns Name, Say it, Who they are, Voice).
func ParseCast(md string) []Cast {
	var out []Cast
	inCast := false
	for line := range strings.SplitSeq(md, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") {
			inCast = strings.TrimSpace(line[3:]) == "Cast"
			continue
		}
		if !inCast || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if len(cells) < 4 || cells[0] == "Name" || strings.HasPrefix(cells[0], "---") {
			continue
		}
		out = append(out, Cast{Name: cells[0], SayIt: cells[1], Voice: cells[3]})
	}
	return out
}

type turn struct {
	speaker    string
	paragraphs []string
}

type passage struct {
	script.Passage
	turns []turn
}

// Export returns README.md and one text file per passage whose id starts
// with one of the prefixes (all passages without any).
func Export(s script.Script, cast []Cast, only []string) []File {
	var passages []passage
	for _, sec := range s.Sections {
		for _, p := range sec.Passages {
			if wanted(p.ID, only) {
				passages = append(passages, passage{Passage: p, turns: turns(p)})
			}
		}
	}
	files := []File{{Name: "README.md"}}
	var spoken strings.Builder
	for _, p := range passages {
		text := spokenText(p.turns)
		spoken.WriteString(text)
		files = append(files, File{Name: p.ID + ".txt", Text: text})
	}
	files[0].Text = readme(passages, cast, spoken.String())
	return files
}

func wanted(id string, only []string) bool {
	if len(only) == 0 {
		return true
	}
	for _, prefix := range only {
		if prefix = strings.TrimSpace(prefix); prefix != "" && strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

// turns groups a passage's spoken parts by speaker; a part without a speaker
// is spoken by the passage's own speaker (its Speaker note, else the Narrator).
func turns(p script.Passage) []turn {
	fallback := "Narrator"
	for _, n := range p.Notes {
		if n.Label == "Speaker" && n.Text != "" {
			fallback = strings.TrimSuffix(n.Text, ".")
		}
	}
	var out []turn
	for _, part := range p.Parts {
		speaker := part.Speaker
		if speaker == "" {
			speaker = fallback
		}
		if n := len(out); n > 0 && out[n-1].speaker == speaker {
			out[n-1].paragraphs = append(out[n-1].paragraphs, part.Paragraphs...)
			continue
		}
		out = append(out, turn{speaker: speaker, paragraphs: append([]string(nil), part.Paragraphs...)})
	}
	return out
}

func spokenText(turns []turn) string {
	var paras []string
	for _, t := range turns {
		paras = append(paras, t.paragraphs...)
	}
	return strings.Join(paras, "\n\n") + "\n"
}

const maxWords = 10

// firstWords is the start of a paragraph, enough to find it in a pasted text.
func firstWords(paragraph string) string {
	words := strings.Fields(paragraph)
	if len(words) <= maxWords {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:maxWords], " ") + "…"
}

var titles = map[string]bool{"Lord": true, "Lady": true, "Sergeant": true, "Brother": true, "Sister": true, "The": true}

var nonLetters = regexp.MustCompile(`[^\p{L}]+`)

func words(s string) []string {
	return strings.Fields(nonLetters.ReplaceAllString(s, " "))
}

// castFor finds a speaker's cast row: every word of the speaker's name is in
// the row's name ("Sergeant Hale" is "Sergeant Corin Hale").
func castFor(speaker string, cast []Cast) (Cast, bool) {
	for _, c := range cast {
		names := map[string]bool{}
		for _, w := range words(c.Name) {
			names[w] = true
		}
		all := true
		for _, w := range words(speaker) {
			all = all && names[w]
		}
		if all {
			return c, true
		}
	}
	return Cast{}, false
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func readme(passages []passage, cast []Cast, spoken string) string {
	var b strings.Builder
	total := utf8.RuneCountInString(spoken)
	fmt.Fprintf(&b, "# Narration text\n\n")
	fmt.Fprintf(&b, "Written by `make narration-text` from the read-aloud script; edit the script, not these\n")
	fmt.Fprintf(&b, "files, and export again. Each `<id>.txt` holds only the spoken words of one passage:\n")
	fmt.Fprintf(&b, "paste it into one Studio chapter named after the id, assign the voices turn by turn as\n")
	fmt.Fprintf(&b, "listed below, export the chapter as `<id>.mp3` and upload it on the campaign page\n")
	fmt.Fprintf(&b, "(Audio clips).\n\n")
	fmt.Fprintf(&b, "%s, %d characters in all (about one credit per character for each take).\n",
		plural(len(passages), "passage"), total)

	var speakers []string
	seen := map[string]bool{}
	for _, p := range passages {
		for _, t := range p.turns {
			if !seen[t.speaker] {
				seen[t.speaker] = true
				speakers = append(speakers, t.speaker)
			}
		}
	}
	if len(speakers) > 0 {
		fmt.Fprintf(&b, "\n## Voices\n\n| Speaker | Voice |\n|---|---|\n")
		for _, s := range speakers {
			voice := "-"
			if c, ok := castFor(s, cast); ok {
				voice = c.Voice
			}
			fmt.Fprintf(&b, "| %s | %s |\n", s, voice)
		}
	}

	spokenWords := map[string]bool{}
	for _, w := range words(spoken) {
		spokenWords[w] = true
	}
	var say []Cast
	for _, c := range cast {
		if c.SayIt == "" || c.SayIt == "-" {
			continue
		}
		for _, w := range words(c.Name) {
			// Lower-case words ("the") and titles are not names.
			if first, _ := utf8.DecodeRuneInString(w); unicode.IsUpper(first) && !titles[w] && spokenWords[w] {
				say = append(say, c)
				break
			}
		}
	}
	if len(say) > 0 {
		fmt.Fprintf(&b, "\n## Pronunciation (names spoken in these passages)\n\n| Name | Say it |\n|---|---|\n")
		for _, c := range say {
			fmt.Fprintf(&b, "| %s | %s |\n", c.Name, c.SayIt)
		}
	}

	for _, p := range passages {
		fmt.Fprintf(&b, "\n## %s - %s\n\n", p.ID, p.Title)
		fmt.Fprintf(&b, "File `%s.txt`, %d characters.\n\n", p.ID, utf8.RuneCountInString(spokenText(p.turns)))
		for _, n := range p.Notes {
			fmt.Fprintf(&b, "- %s: %s\n", n.Label, n.Text)
		}
		if len(p.Notes) > 0 {
			b.WriteString("\n")
		}
		for i, t := range p.turns {
			first := ""
			if len(t.paragraphs) > 0 {
				first = firstWords(t.paragraphs[0])
			}
			fmt.Fprintf(&b, "%d. %s (%s): \"%s\"\n", i+1, t.speaker, plural(len(t.paragraphs), "paragraph"), first)
		}
	}
	return b.String()
}
