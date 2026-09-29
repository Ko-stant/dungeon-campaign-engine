package script

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Source is one file of an assembled script: where its part starts in the
// joined text (1-based line) and how many header lines were dropped above it.
type Source struct {
	Name  string
	Start int
	Skip  int
}

// scriptFile matches the numbered files of a script folder: "01-prologue.md".
var scriptFile = regexp.MustCompile(`^\d+-.*\.md$`)

// Assemble joins a script folder into one script text: its numbered Markdown
// files ("01-prologue.md", "02-quest-1.md", ...) in name order, each without
// the header above its first "## " section (a title, a Last Updated line,
// notes). Other files, such as README.md, and sub-folders are left out. It
// returns the text and the files used, for Locate.
func Assemble(fsys fs.FS) (string, []Source, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return "", nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && scriptFile.MatchString(e.Name()) {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		return "", nil, errors.New("no numbered script files (like 01-prologue.md) found")
	}
	sort.Strings(files)
	parts := make([]string, 0, len(files))
	sources := make([]Source, 0, len(files))
	start := 1
	for _, name := range files {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return "", nil, err
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		i := strings.Index(text, "\n## ")
		switch {
		case strings.HasPrefix(text, "## "):
			i = 0
		case i >= 0:
			i++
		default:
			return "", nil, fmt.Errorf("%s has no ## section heading", name)
		}
		part := strings.TrimSpace(text[i:])
		parts = append(parts, part)
		sources = append(sources, Source{Name: name, Start: start, Skip: strings.Count(text[:i], "\n")})
		start += strings.Count(part, "\n") + 2 // the part's lines and the blank line after it
	}
	return strings.Join(parts, "\n\n") + "\n", sources, nil
}

var lineRef = regexp.MustCompile(`\bline (\d+)\b`)

// Locate rewrites "line N" of the joined script in a Parse error as the file
// and its own line ("02-quest-1.md line 9"). Other errors are unchanged.
func Locate(err error, sources []Source) error {
	m := lineRef.FindStringSubmatchIndex(err.Error())
	if m == nil || len(sources) == 0 {
		return err
	}
	msg := err.Error()
	n, _ := strconv.Atoi(msg[m[2]:m[3]])
	src := sources[0]
	for _, s := range sources {
		if s.Start <= n {
			src = s
		}
	}
	where := fmt.Sprintf("%s line %d", src.Name, n-src.Start+1+src.Skip)
	return errors.New(msg[:m[0]] + where + msg[m[1]:])
}
