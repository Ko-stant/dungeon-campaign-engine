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

func TestLibrary(t *testing.T) {
	dir := t.TempDir()
	lib := New(dir)

	clips, err := lib.List(campaign)
	if err != nil || len(clips) != 0 {
		t.Fatalf("empty: %v %v", clips, err)
	}
	if err := lib.Save(campaign, "Q2-03.mp3", strings.NewReader("mp3 data")); err != nil {
		t.Fatal(err)
	}
	if err := lib.Save(campaign, "Q3-09a.ogg", strings.NewReader("ogg")); err != nil {
		t.Fatal(err)
	}
	// Saving an id again with another format replaces the old file.
	if err := lib.Save(campaign, "Q2-03.wav", strings.NewReader("wav data")); err != nil {
		t.Fatal(err)
	}
	clips, err = lib.List(campaign)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(clips, map[string]string{"Q2-03": "Q2-03.wav", "Q3-09a": "Q3-09a.ogg"}) {
		t.Fatalf("clips: %v", clips)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, campaign, "Q2-03.wav")); string(data) != "wav data" {
		t.Fatalf("file on disk: %q", data)
	}

	// Files copied into the folder by hand count too; other files are ignored.
	if err := os.WriteFile(filepath.Join(dir, campaign, "E-01.m4a"), []byte("m4a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, campaign, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if clips, _ = lib.List(campaign); len(clips) != 3 || clips["E-01"] != "E-01.m4a" {
		t.Fatalf("hand-copied clip: %v", clips)
	}

	path, err := lib.Path(campaign, "Q2-03.wav")
	if err != nil || path != filepath.Join(dir, campaign, "Q2-03.wav") {
		t.Fatalf("path: %q %v", path, err)
	}
	for _, name := range []string{"../Q2-03.wav", "notes.txt", "Q9-99.mp3"} {
		if _, err := lib.Path(campaign, name); err == nil {
			t.Errorf("Path(%q) should fail", name)
		}
	}

	if err := lib.Delete(campaign, "Q2-03"); err != nil {
		t.Fatal(err)
	}
	if clips, _ = lib.List(campaign); len(clips) != 2 {
		t.Fatalf("after delete: %v", clips)
	}
	if err := lib.Delete(campaign, "Q2-03"); err == nil {
		t.Fatal("deleting a missing clip should fail")
	}

	for _, bad := range []string{"", "..", "../other", "not-a-uuid"} {
		if _, err := lib.List(bad); err == nil {
			t.Errorf("List(%q) should fail", bad)
		}
		if err := lib.Save(bad, "Q1.mp3", strings.NewReader("x")); err == nil {
			t.Errorf("Save(%q) should fail", bad)
		}
	}
}

func TestSaveLimit(t *testing.T) {
	lib := New(t.TempDir())
	lib.MaxBytes = 4
	if err := lib.Save(campaign, "Q1.mp3", strings.NewReader("12345")); err == nil {
		t.Fatal("a file over the limit should be refused")
	}
	if clips, _ := lib.List(campaign); len(clips) != 0 {
		t.Fatalf("a refused file must not be left behind: %v", clips)
	}
}

func TestRemoveCampaign(t *testing.T) {
	dir := t.TempDir()
	lib := New(dir)
	const other = "01a0ea8a-e8d5-7517-94a0-7176179e5bb4"
	for _, c := range []string{campaign, other} {
		if err := lib.Save(c, "Q1.mp3", strings.NewReader("x")); err != nil {
			t.Fatal(err)
		}
	}

	if err := lib.RemoveCampaign(campaign); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, campaign)); !os.IsNotExist(err) {
		t.Errorf("the campaign's folder should be gone: %v", err)
	}
	if clips, _ := lib.List(other); len(clips) != 1 {
		t.Errorf("another campaign's clips must stay: %v", clips)
	}
	// A campaign without clips has no folder; removing it is fine.
	if err := lib.RemoveCampaign(campaign); err != nil {
		t.Errorf("removing a missing folder: %v", err)
	}
	for _, bad := range []string{"", "..", "../other", "not-a-uuid"} {
		if err := lib.RemoveCampaign(bad); err == nil {
			t.Errorf("RemoveCampaign(%q) should fail", bad)
		}
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("the audio folder itself must stay: %v", err)
	}
}
