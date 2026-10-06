// Package audio names and reads the read-aloud audio clips of a campaign.
// A clip is named after a script passage id ("Q2-03.mp3" plays with passage
// Q2-03; "Q3-09a" is an extra clip of passage Q3-09). The clips themselves
// are kept in the database (internal/store, audio_clip).
package audio

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Extensions lists the audio formats a clip may use.
var Extensions = []string{".mp3", ".m4a", ".ogg", ".opus", ".wav", ".webm", ".flac"}

var contentTypes = map[string]string{
	".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".ogg": "audio/ogg", ".opus": "audio/ogg",
	".wav": "audio/wav", ".webm": "audio/webm", ".flac": "audio/flac",
}

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
	_, known := contentTypes[ext]
	switch {
	case !known:
		return "", "", fmt.Errorf("%q is not an audio file (%s)", base, strings.Join(Extensions, ", "))
	case id == "" || len(id) > maxIDLength || !clipID.MatchString(id) || strings.Contains(id, ".."):
		return "", "", fmt.Errorf("%q: name the file after its passage id, e.g. Q2-03%s", base, ext)
	}
	return id, ext, nil
}

// ContentType is the media type of a clip extension.
func ContentType(ext string) string {
	if t, ok := contentTypes[strings.ToLower(ext)]; ok {
		return t
	}
	return "application/octet-stream"
}

// ReadClip reads a clip of at most maxBytes.
func ReadClip(r io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("larger than %d MB", maxBytes>>20)
	}
	return data, nil
}

// Found is a clip file found by ScanDir.
type Found struct {
	Campaign string
	ID       string
	Ext      string
	Path     string
}

// ScanDir finds clip files laid out as <dir>/<campaign id>/<clip file>,
// the folder layout of AUDIO_DIR before clips moved into the database. It
// returns the clips by campaign and id, and the paths it skipped.
func ScanDir(dir string) (found []Found, skipped []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if !e.IsDir() || !campaignID.MatchString(e.Name()) {
			skipped = append(skipped, p)
			continue
		}
		files, err := os.ReadDir(p)
		if err != nil {
			return nil, nil, err
		}
		for _, f := range files {
			fp := filepath.Join(p, f.Name())
			id, ext, err := ClipName(f.Name())
			if f.IsDir() || err != nil {
				skipped = append(skipped, fp)
				continue
			}
			found = append(found, Found{Campaign: e.Name(), ID: id, Ext: ext, Path: fp})
		}
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].Campaign != found[j].Campaign {
			return found[i].Campaign < found[j].Campaign
		}
		return found[i].ID < found[j].ID
	})
	return found, skipped, nil
}
