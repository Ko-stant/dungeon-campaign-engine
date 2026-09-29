package script

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAssemble(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md":         {Data: []byte("# Guide\n\n## Not part of the script\n\n### X-1 - Ignored\n\n> Never read.\n")},
		"02-quest-1.md":     {Data: []byte("# Quest 1 file\n\n**Last Updated**: today\n\nNotes for the file.\n\n## Quest 1 - The Halls\n\n### Q1-01 - The Halls\n\n> The doors groan shut.\n")},
		"01-prologue.md":    {Data: []byte("# Prologue file\n\n**Last Updated**: today\n\n## Prologue\n\n### P0-01 - The request\n\n> Come in.\n")},
		"10-the-end.md":     {Data: []byte("## The End\n\n### E-01 - Done\n\n> The end.\n")},
		"notes.txt":         {Data: []byte("not markdown")},
		"drafts/03-idea.md": {Data: []byte("## Draft\n\n### D-1 - Draft\n\n> Not yet.\n")},
	}
	text, sources, err := Assemble(fsys)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, src := range sources {
		names = append(names, src.Name)
	}
	if strings.Join(names, ",") != "01-prologue.md,02-quest-1.md,10-the-end.md" {
		t.Fatalf("files in order: %v", names)
	}
	if strings.Contains(text, "Last Updated") || strings.Contains(text, "file") || strings.Contains(text, "Never read") {
		t.Fatalf("each file's header before its first ## section is dropped:\n%s", text)
	}
	s, err := Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	if s.Count() != 3 || s.Sections[0].Title != "Prologue" || s.Sections[2].Title != "The End" {
		t.Fatalf("assembled script: %+v", s.Sections)
	}
}

func TestAssembleErrors(t *testing.T) {
	if _, _, err := Assemble(fstest.MapFS{"README.md": {Data: []byte("# Guide")}}); err == nil {
		t.Fatal("a folder without numbered files is an error")
	}
	_, _, err := Assemble(fstest.MapFS{"01-a.md": {Data: []byte("# Only a header\n")}})
	if err == nil || !strings.Contains(err.Error(), "01-a.md") {
		t.Fatalf("a numbered file without a ## section should be named: %v", err)
	}
}

func TestLocate(t *testing.T) {
	fsys := fstest.MapFS{
		"01-a.md": {Data: []byte("# A\n\n**Last Updated**: today\n\n## A\n\n### A-1 - One\n\n> Hi.\n")},
		"02-b.md": {Data: []byte("# B\n\n## B\n\n### B-1 - Two\n\n> Hi.\n\nStray words.\n")},
	}
	text, sources, err := Assemble(fsys)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse(text)
	if err == nil {
		t.Fatal("want a parse error")
	}
	located := Locate(err, sources)
	if !strings.Contains(located.Error(), "02-b.md line 9:") {
		t.Fatalf("error should name the file and its own line: %v", located)
	}
	if got := Locate(errors.New("no line here"), sources).Error(); got != "no line here" {
		t.Fatalf("errors without a line are unchanged: %q", got)
	}
}
