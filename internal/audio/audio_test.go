package audio

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const campaign = "01a0ea8a-e8d5-7517-94a0-7176179e5bb3"

func TestClipName(t *testing.T) {
	for in, want := range map[string][2]string{
		"Q2-03.mp3":       {"Q2-03", ".mp3"},
		"Q3-09a.WAV":      {"Q3-09a", ".wav"},
		"P0-01.m4a":       {"P0-01", ".m4a"},
		"E-02.ogg":        {"E-02", ".ogg"},
		"G-01.opus":       {"G-01", ".opus"},
		"x.webm":          {"x", ".webm"},
		"dir/Q1-01.flac":  {"Q1-01", ".flac"},
		`C:\clips\Q1.mp3`: {"Q1", ".mp3"},
	} {
		id, ext, err := ClipName(in)
		if err != nil || id != want[0] || ext != want[1] {
			t.Errorf("ClipName(%q) = %q %q %v; want %q", in, id, ext, err, want)
		}
	}
	for _, in := range []string{"", "Q2-03", "Q2-03.txt", ".mp3", "a b.mp3", "..mp3", "x..y.mp3", strings.Repeat("x", 41) + ".mp3"} {
		if _, _, err := ClipName(in); err == nil {
			t.Errorf("ClipName(%q) should fail", in)
		}
	}
}

func TestContentType(t *testing.T) {
	for ext, want := range map[string]string{".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".ogg": "audio/ogg", ".opus": "audio/ogg", ".wav": "audio/wav", ".webm": "audio/webm", ".flac": "audio/flac"} {
		if got := ContentType(ext); got != want {
			t.Errorf("ContentType(%q) = %q, want %q", ext, got, want)
		}
	}
	for _, ext := range Extensions {
		if ContentType(ext) == "application/octet-stream" {
			t.Errorf("%s has no type", ext)
		}
	}
}

func TestReadClipHasALimit(t *testing.T) {
	if data, err := ReadClip(strings.NewReader("12345"), 5); err != nil || string(data) != "12345" {
		t.Errorf("at the limit: %q %v", data, err)
	}
	if _, err := ReadClip(strings.NewReader("123456"), 5); err == nil {
		t.Error("over the limit should fail")
	}
}

func TestScanDirFindsCampaignFolders(t *testing.T) {
	dir := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(campaign + "/Q2-03.mp3")
	write(campaign + "/Q3-09a.ogg")
	write(campaign + "/notes.txt")    // not audio: skipped
	write("not-a-campaign/Q1-01.mp3") // not a campaign folder: skipped
	write("Q1-02.mp3")                // loose file: skipped
	found, skipped, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range found {
		got = append(got, f.Campaign+"/"+f.ID+f.Ext)
	}
	if want := []string{campaign + "/Q2-03.mp3", campaign + "/Q3-09a.ogg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("found %v, want %v", got, want)
	}
	if len(skipped) != 3 {
		t.Errorf("skipped %v", skipped)
	}
}
