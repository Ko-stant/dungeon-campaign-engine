package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func TestImportAudioSavesNewAndChangedClips(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	camp, _ := st.CreateCampaign(ctx, "Three Plagues", nil)
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(camp.ID+"/Q2-03.mp3", "ID3 one")
	write(camp.ID+"/P0-01.ogg", "OggS")
	write("0190c6a0-0000-7000-8000-000000000000/Q1-01.mp3", "elsewhere") // no such campaign here

	var out strings.Builder
	got, err := importAudio(ctx, st, dir, &out)
	if err != nil {
		t.Fatal(err)
	}
	if got.saved != 2 || got.unchanged != 0 || got.noCampaign != 1 {
		t.Fatalf("first import %+v\n%s", got, out.String())
	}
	if !strings.Contains(out.String(), "0190c6a0-0000-7000-8000-000000000000") {
		t.Errorf("the missing campaign is reported: %s", out.String())
	}

	// Again: nothing changed. Then one clip is re-recorded.
	if got, _ = importAudio(ctx, st, dir, &out); got.saved != 0 || got.unchanged != 2 {
		t.Fatalf("second import %+v", got)
	}
	write(camp.ID+"/Q2-03.mp3", "ID3 two")
	if got, _ = importAudio(ctx, st, dir, &out); got.saved != 1 || got.unchanged != 1 {
		t.Fatalf("after a new take %+v", got)
	}
	if _, data, err := st.GetAudioClip(ctx, camp.ID, "Q2-03"); err != nil || string(data) != "ID3 two" {
		t.Errorf("the new take %q %v", data, err)
	}
}
