// Package script parses a campaign's read-aloud script: Markdown written in
// the format of docs/campaigns/three-plagues/script/ (see Assemble for script
// folders). The GM pastes it on the campaign page or loads it with
// cmd/load-script, and the tracker shows its passages to read (or play) at
// the table.
//
// The format, line by line:
//
//	## Section title                    a group of passages (sections without passages are dropped)
//	### ID - Title                      a passage
//	- **Label:** text                   a note (When, Speaker, Voice, Sound, ...); indented lines continue it
//	**Speaker** (aside)                 the next quoted text is spoken by Speaker
//	> text                              spoken text; "> " alone ends a paragraph; a trailing \ keeps the line break
//	**Quest goals** then "- item"s      table text shown with the passage
//
// Anything else inside a passage is an error, so a typo is caught when the
// script is saved rather than silently dropped.
package script

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Limits.
const (
	MaxBytes       = 200_000
	MaxIDLength    = 40
	MaxTitleLength = 200
)

// Script is a parsed read-aloud script.
type Script struct {
	Sections []Section `json:"sections"`
}

// Section groups passages, e.g. "Quest 2 - The Bloated Fields".
type Section struct {
	Title    string    `json:"title"`
	Intro    string    `json:"intro,omitempty"`
	Passages []Passage `json:"passages"`
}

// Passage is one piece of read-aloud text (one audio clip).
type Passage struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Notes      []Note   `json:"notes,omitempty"`
	Parts      []Part   `json:"parts"`
	GoalsTitle string   `json:"goalsTitle,omitempty"`
	Goals      []string `json:"goals,omitempty"`
}

// Note is a GM note on a passage: when to read it, who speaks, how.
type Note struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

// Part is text spoken by one speaker (empty: the passage's speaker).
type Part struct {
	Speaker    string   `json:"speaker,omitempty"`
	Aside      string   `json:"aside,omitempty"`
	Paragraphs []string `json:"paragraphs"`
}

// Count is the number of passages.
func (s Script) Count() int {
	n := 0
	for _, sec := range s.Sections {
		n += len(sec.Passages)
	}
	return n
}

var (
	noteLine  = regexp.MustCompile(`^- \*\*(.+?):\*\*\s*(.*)$`)
	boldLine  = regexp.MustCompile(`^\*\*(.+?)\*\*\s*(?:\((.+)\))?$`)
	emphasis  = regexp.MustCompile("[*`]+") // bold, italic and code marks
	spaceRuns = regexp.MustCompile(`[ \t]+`)
)

func plain(s string) string {
	return strings.TrimSpace(spaceRuns.ReplaceAllString(emphasis.ReplaceAllString(s, ""), " "))
}

type parser struct {
	out     Script
	sec     *Section
	psg     *Passage
	label   string // a bold line waiting to see whether speech or goals follow
	para    *strings.Builder
	keepNL  bool // the last quoted line ended with a backslash
	cont    *string
	inGoals bool
}

func (p *parser) endParagraph() {
	if p.para != nil && p.psg != nil {
		if text := strings.TrimSpace(p.para.String()); text != "" {
			part := &p.psg.Parts[len(p.psg.Parts)-1]
			part.Paragraphs = append(part.Paragraphs, text)
		}
	}
	p.para = nil
	p.keepNL = false
}

func (p *parser) endPassage() error {
	p.endParagraph()
	if p.psg == nil {
		return nil
	}
	psg := p.psg
	p.psg, p.label, p.cont, p.inGoals = nil, "", nil, false
	var parts []Part
	for _, part := range psg.Parts {
		if len(part.Paragraphs) > 0 {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return fmt.Errorf("passage %s has no text to read (quote it with > )", psg.ID)
	}
	psg.Parts = parts
	p.sec.Passages = append(p.sec.Passages, *psg)
	return nil
}

func (p *parser) endSection() error {
	if err := p.endPassage(); err != nil {
		return err
	}
	if p.sec != nil && len(p.sec.Passages) > 0 {
		p.out.Sections = append(p.out.Sections, *p.sec)
	}
	p.sec = nil
	return nil
}

// Parse reads a script. An empty text is an empty script.
func Parse(text string) (Script, error) {
	if len(text) > MaxBytes {
		return Script{}, fmt.Errorf("the script is %d bytes; the limit is %d", len(text), MaxBytes)
	}
	if strings.TrimSpace(text) == "" {
		return Script{Sections: []Section{}}, nil
	}
	p := &parser{}
	ids := map[string]bool{}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, raw := range lines {
		lineErr := func(format string, args ...any) error {
			return fmt.Errorf("line %d: %s", i+1, fmt.Sprintf(format, args...))
		}
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "### "):
			if p.sec == nil {
				return Script{}, lineErr("passage %q is not inside a ## section", trimmed)
			}
			if err := p.endPassage(); err != nil {
				return Script{}, lineErr("%v", err)
			}
			id, title, ok := strings.Cut(strings.TrimSpace(line[4:]), " - ")
			id, title = strings.TrimSpace(id), plain(title)
			if !ok || id == "" || title == "" {
				return Script{}, lineErr("a passage heading needs an id and a title: ### P0-01 - The request")
			}
			if len(id) > MaxIDLength || strings.ContainsAny(id, " \t") {
				return Script{}, lineErr("passage id %q must be one word of at most %d characters", id, MaxIDLength)
			}
			if len(title) > MaxTitleLength {
				return Script{}, lineErr("passage %s: the title must be at most %d characters", id, MaxTitleLength)
			}
			if ids[id] {
				return Script{}, lineErr("passage id %s is used twice", id)
			}
			ids[id] = true
			p.psg = &Passage{ID: id, Title: title, Parts: []Part{{}}}

		case strings.HasPrefix(line, "## "):
			if err := p.endSection(); err != nil {
				return Script{}, lineErr("%v", err)
			}
			p.sec = &Section{Title: plain(line[3:])}

		case strings.HasPrefix(line, "# "), p.sec == nil:
			// The document title and anything before the first section.

		case p.psg == nil:
			// Section text before its first passage.
			if trimmed != "" && trimmed != "---" && !strings.HasPrefix(trimmed, "|") {
				p.sec.Intro = strings.TrimSpace(p.sec.Intro + " " + plain(trimmed))
			}

		case strings.HasPrefix(trimmed, ">"):
			p.cont = nil
			body := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			if p.label != "" {
				p.endParagraph()
				speaker, aside := p.label, ""
				if m := boldLine.FindStringSubmatch(speaker); m != nil {
					speaker, aside = m[1], m[2]
				}
				p.psg.Parts = append(p.psg.Parts, Part{Speaker: plain(speaker), Aside: plain(aside)})
				p.label = ""
			}
			if body == "" {
				p.endParagraph()
				continue
			}
			keep := strings.HasSuffix(body, `\`)
			body = plain(strings.TrimSuffix(body, `\`))
			if p.para == nil {
				p.para = &strings.Builder{}
			} else if p.keepNL {
				p.para.WriteString("\n")
			} else {
				p.para.WriteString(" ")
			}
			p.para.WriteString(body)
			p.keepNL = keep

		case trimmed == "", trimmed == "---":
			p.endParagraph()
			if trimmed == "" {
				p.cont = nil
			}

		case strings.HasPrefix(line, "  ") && p.cont != nil:
			*p.cont = plain(*p.cont + " " + trimmed)

		case boldLine.MatchString(trimmed):
			p.endParagraph()
			p.label = trimmed
			p.inGoals = false

		case strings.HasPrefix(trimmed, "- "):
			p.endParagraph()
			if m := noteLine.FindStringSubmatch(trimmed); m != nil && !p.inGoals && p.label == "" {
				p.psg.Notes = append(p.psg.Notes, Note{Label: plain(m[1]), Text: plain(m[2])})
				p.cont = &p.psg.Notes[len(p.psg.Notes)-1].Text
				continue
			}
			if p.label != "" {
				title := p.label
				if m := boldLine.FindStringSubmatch(title); m != nil {
					title = m[1]
				}
				p.psg.GoalsTitle, p.label, p.inGoals = plain(title), "", true
			}
			p.psg.Goals = append(p.psg.Goals, plain(trimmed[2:]))
			p.cont = &p.psg.Goals[len(p.psg.Goals)-1]

		default:
			return Script{}, lineErr("passage %s: %q is not spoken text (start it with > ), a note (- **When:** ...) or a speaker (**Name**)", p.psg.ID, trimmed)
		}
	}
	if err := p.endSection(); err != nil {
		return Script{}, err
	}
	if len(p.out.Sections) == 0 {
		return Script{}, errors.New("no passages found: start each with a heading like ### P0-01 - The request inside a ## section")
	}
	return p.out, nil
}
