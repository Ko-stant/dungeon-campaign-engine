// Command import-audio copies read-aloud audio clips from a folder into a
// database. The folder is laid out as AUDIO_DIR was before clips moved into
// the database: <dir>/<campaign id>/<passage id>.<ext>. Clips already there
// with the same sound are left alone, so it can be run again after
// recording new takes.
//
//	go run ./cmd/import-audio -dir ../dungeon-campaign-engine/audio
//	go run ./cmd/import-audio -dir audio -db "$HOSTED_DATABASE_URL"
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/audio"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
)

func main() {
	dir := flag.String("dir", "audio", "folder of <campaign id>/<clip> files")
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "Postgres URL (defaults to $DATABASE_URL)")
	flag.Parse()
	log.SetFlags(0)
	if *dbURL == "" {
		log.Fatal("no database URL: set DATABASE_URL (see .env) or pass -db")
	}
	ctx := context.Background()
	if err := store.Migrate(ctx, *dbURL); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(ctx, *dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	got, err := importAudio(ctx, st, *dir, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved %d clips, %d unchanged, %d for campaigns not in this database.\n", got.saved, got.unchanged, got.noCampaign)
}

type counts struct {
	saved, unchanged, noCampaign int
}

func importAudio(ctx context.Context, st *store.Store, dir string, out io.Writer) (counts, error) {
	var c counts
	found, skipped, err := audio.ScanDir(dir)
	if err != nil {
		return c, err
	}
	for _, p := range skipped {
		_, _ = fmt.Fprintf(out, "skipped %s (not <campaign id>/<passage id>.<audio ext>)\n", p)
	}
	existing := map[string]map[string]string{} // campaign -> clip id -> ETag
	for _, f := range found {
		etags, ok := existing[f.Campaign]
		if !ok {
			clips, err := st.ListAudioClips(ctx, f.Campaign)
			if err != nil {
				return c, err
			}
			etags = map[string]string{}
			for _, cl := range clips {
				etags[cl.ID] = cl.Ext + " " + cl.ETag
			}
			existing[f.Campaign] = etags
		}
		file, err := os.Open(f.Path)
		if err != nil {
			return c, err
		}
		data, err := audio.ReadClip(file, audio.DefaultMaxBytes)
		_ = file.Close()
		if err != nil {
			return c, fmt.Errorf("%s: %w", f.Path, err)
		}
		if etags[f.ID] == f.Ext+" "+store.AudioETag(data) {
			c.unchanged++
			continue
		}
		err = st.SaveAudioClip(ctx, f.Campaign, store.AudioClip{ID: f.ID, Ext: f.Ext, ContentType: audio.ContentType(f.Ext)}, data)
		if errors.Is(err, store.ErrNotFound) {
			c.noCampaign++
			_, _ = fmt.Fprintf(out, "no campaign %s here: skipped %s\n", f.Campaign, f.Path)
			continue
		}
		if err != nil {
			return c, fmt.Errorf("%s: %w", f.Path, err)
		}
		c.saved++
		_, _ = fmt.Fprintf(out, "saved %s/%s%s\n", f.Campaign, f.ID, f.Ext)
	}
	return c, nil
}
