package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store/storetest"
)

func TestAudioClips(t *testing.T) {
	st, _ := storetest.New(t)
	ctx := context.Background()
	camp, _ := st.CreateCampaign(ctx, "Three Plagues", nil)
	other, _ := st.CreateCampaign(ctx, "Other", nil)

	if clips, err := st.ListAudioClips(ctx, camp.ID); err != nil || len(clips) != 0 {
		t.Fatalf("no clips yet: %v %v", clips, err)
	}
	if err := st.SaveAudioClip(ctx, camp.ID, store.AudioClip{ID: "Q2-03", Ext: ".mp3", ContentType: "audio/mpeg"}, []byte("ID3 one")); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveAudioClip(ctx, camp.ID, store.AudioClip{ID: "P0-01", Ext: ".ogg", ContentType: "audio/ogg"}, []byte("OggS")); err != nil {
		t.Fatal(err)
	}
	clips, err := st.ListAudioClips(ctx, camp.ID)
	if err != nil || len(clips) != 2 || clips[0].ID != "P0-01" || clips[1].ID != "Q2-03" || clips[1].Size != 7 || clips[1].File() != "Q2-03.mp3" || len(clips[1].ETag) == 0 {
		t.Fatalf("clips (by id, without their data) %+v %v", clips, err)
	}

	// Saving an id again replaces the clip, format and all.
	if err := st.SaveAudioClip(ctx, camp.ID, store.AudioClip{ID: "Q2-03", Ext: ".m4a", ContentType: "audio/mp4"}, []byte("new take")); err != nil {
		t.Fatal(err)
	}
	clip, data, err := st.GetAudioClip(ctx, camp.ID, "Q2-03")
	if err != nil || clip.Ext != ".m4a" || clip.ContentType != "audio/mp4" || string(data) != "new take" || clip.ETag == clips[1].ETag {
		t.Fatalf("replaced %+v %q %v", clip, data, err)
	}
	if _, _, err := st.GetAudioClip(ctx, other.ID, "Q2-03"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("another campaign's clip: %v", err)
	}
	if _, _, err := st.GetAudioClip(ctx, "not-a-uuid", "Q2-03"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("a bad campaign id: %v", err)
	}

	if err := st.DeleteAudioClip(ctx, camp.ID, "P0-01"); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteAudioClip(ctx, camp.ID, "P0-01"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("deleting twice: %v", err)
	}
	if err := st.SaveAudioClip(ctx, "0190c6a0-0000-7000-8000-000000000000", store.AudioClip{ID: "X", Ext: ".mp3", ContentType: "audio/mpeg"}, []byte("x")); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("a missing campaign: %v", err)
	}

}
