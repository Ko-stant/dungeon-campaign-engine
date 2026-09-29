// Package audio keeps the read-aloud audio clips of each campaign on disk:
// <dir>/<campaign id>/<passage id>.<ext>, e.g. audio/<uuid>/Q2-03.mp3. A clip
// named after a passage id plays with that passage; "Q3-09a" is an extra
// clip of passage Q3-09. Files copied into the folder by hand work too.
package audio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Extensions lists the audio formats a clip may use.
var Extensions = []string{".mp3", ".m4a", ".ogg", ".opus", ".wav", ".webm", ".flac"}

// DefaultMaxBytes bounds one clip.
const DefaultMaxBytes = 50 << 20

const maxIDLength = 40

var (
	clipID     = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9._-]*$`)
	campaignID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// ClipName reads a clip's id and lower-case extension from a file name (any
// folders in front of it are dropped).
func ClipName(filename string) (id, ext string, err error) {
	base := path.Base(strings.ReplaceAll(filename, `\`, "/"))
	ext = strings.ToLower(path.Ext(base))
	id = strings.TrimSuffix(base, path.Ext(base))
	known := false
	for _, e := range Extensions {
		known = known || e == ext
	}
	switch {
	case !known:
		return "", "", fmt.Errorf("%q is not an audio file (%s)", base, strings.Join(Extensions, ", "))
	case id == "" || len(id) > maxIDLength || !clipID.MatchString(id) || strings.Contains(id, ".."):
		return "", "", fmt.Errorf("%q: name the file after its passage id, e.g. Q2-03%s", base, ext)
	}
	return id, ext, nil
}

// Library is the folder holding every campaign's clips.
type Library struct {
	Dir      string
	MaxBytes int64
}

// New returns a library rooted at dir.
func New(dir string) *Library {
	return &Library{Dir: dir, MaxBytes: DefaultMaxBytes}
}

func (l *Library) folder(campaign string) (string, error) {
	if !campaignID.MatchString(campaign) {
		return "", fmt.Errorf("invalid campaign id %q", campaign)
	}
	return filepath.Join(l.Dir, campaign), nil
}

// List maps each clip id of a campaign to its file name.
func (l *Library) List(campaign string) (map[string]string, error) {
	dir, err := l.folder(campaign)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if id, _, err := ClipName(e.Name()); err == nil {
			out[id] = e.Name()
		}
	}
	return out, nil
}

// Save writes a clip from r, replacing any clip with the same id.
func (l *Library) Save(campaign, filename string, r io.Reader) error {
	dir, err := l.folder(campaign)
	if err != nil {
		return err
	}
	id, ext, err := ClipName(filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return err
	}
	// After a successful rename the temp file is gone; otherwise clean it up.
	defer func() { _ = os.Remove(tmp.Name()) }()
	n, err := io.Copy(tmp, io.LimitReader(r, l.MaxBytes+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n > l.MaxBytes {
		return fmt.Errorf("%s is larger than %d MB", filename, l.MaxBytes>>20)
	}
	if err := l.removeID(dir, id); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, id+ext))
}

func (l *Library) removeID(dir, id string) error {
	for _, ext := range Extensions {
		if err := os.Remove(filepath.Join(dir, id+ext)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// Delete removes a campaign's clip by id.
func (l *Library) Delete(campaign, id string) error {
	clips, err := l.List(campaign)
	if err != nil {
		return err
	}
	if _, ok := clips[id]; !ok {
		return fmt.Errorf("no clip %q: %w", id, os.ErrNotExist)
	}
	dir, _ := l.folder(campaign)
	return l.removeID(dir, id)
}

// Path returns the file path of one of a campaign's clips by file name.
func (l *Library) Path(campaign, name string) (string, error) {
	clips, err := l.List(campaign)
	if err != nil {
		return "", err
	}
	id, _, err := ClipName(name)
	if err != nil || clips[id] != name || path.Base(name) != name {
		return "", fmt.Errorf("no clip %q: %w", name, os.ErrNotExist)
	}
	dir, _ := l.folder(campaign)
	return filepath.Join(dir, name), nil
}
