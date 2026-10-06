package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"sort"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/audio"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// Read-aloud audio clips, kept in the database (internal/store, audio_clip):
// uploaded on the campaign page, listed for the tracker's reader, served
// under /audio/<campaign>/<file>.

// maxAudioUpload bounds one upload request (several clips at once).
const maxAudioUpload = 500 << 20

func (s *Server) registerAudio(mux routeMux) {
	mux.HandleFunc("GET /api/campaigns/{id}/audio", s.listAudio)
	mux.HandleFunc("GET /audio/{campaign}/{file}", s.serveAudio)
	mux.HandleFunc("POST /campaigns/{id}/audio", s.uploadAudioForm)
	mux.HandleFunc("POST /campaigns/{id}/audio/{clip}/delete", s.deleteAudioForm)
}

func clipURL(campaign, file string) string {
	return "/audio/" + campaign + "/" + file
}

// clipPassage names the passage a clip belongs to: the passage with the
// clip's id, or for "Q3-09a" passage Q3-09 (clip "a"). Empty when none.
func clipPassage(clip string, titles map[string]string) string {
	if t, ok := titles[clip]; ok {
		return clip + " " + t
	}
	if n := len(clip); n > 1 {
		if last := clip[n-1]; last >= 'a' && last <= 'z' {
			if t, ok := titles[clip[:n-1]]; ok {
				return fmt.Sprintf("%s %s (%c)", clip[:n-1], t, last)
			}
		}
	}
	return ""
}

// addAudioPageData lists the campaign's clips on the campaign page.
func (s *Server) addAudioPageData(ctx context.Context, d *views.CampaignPageData, titles map[string]string) error {
	clips, err := s.store.ListAudioClips(ctx, d.ID)
	if err != nil {
		return err
	}
	for _, c := range clips {
		d.AudioClips = append(d.AudioClips, views.AudioClipRow{ID: c.ID, File: c.File(), URL: clipURL(d.ID, c.File()), Passage: clipPassage(c.ID, titles)})
	}
	sort.Slice(d.AudioClips, func(i, j int) bool { return d.AudioClips[i].ID < d.AudioClips[j].ID })
	return nil
}

func (s *Server) listAudio(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.store.GetCampaign(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	clips, err := s.store.ListAudioClips(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	out := make(map[string]string, len(clips))
	for _, c := range clips {
		out[c.ID] = clipURL(id, c.File())
	}
	writeJSON(w, http.StatusOK, map[string]any{"clips": out})
}

// serveAudio sends a clip, with ranges (seeking) and revalidation by ETag.
func (s *Server) serveAudio(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("file")
	id, ext, err := audio.ClipName(name)
	if err != nil || id+ext != name {
		http.NotFound(w, r)
		return
	}
	clip, data, err := s.store.GetAudioClip(r.Context(), r.PathValue("campaign"), id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && clip.Ext != ext) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	w.Header().Set("Content-Type", clip.ContentType)
	w.Header().Set("ETag", `"`+clip.ETag+`"`)
	w.Header().Set("Cache-Control", "private, no-cache")
	http.ServeContent(w, r, name, clip.UpdatedAt, bytes.NewReader(data))
}

// uploadAudioForm saves the uploaded clips. Every file name is checked first,
// so a bad name refuses the whole upload.
func (s *Server) uploadAudioForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.store.GetCampaign(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		writeStoreError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAudioUpload)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.renderAudioError(w, r, id, "Upload failed: "+err.Error())
		return
	}
	files := r.MultipartForm.File["clips"]
	if len(files) == 0 {
		s.renderAudioError(w, r, id, "Choose one or more audio files to upload.")
		return
	}
	for _, f := range files {
		if _, _, err := audio.ClipName(f.Filename); err != nil {
			s.renderAudioError(w, r, id, "Nothing uploaded: "+err.Error())
			return
		}
	}
	for _, f := range files {
		if err := s.saveClip(r, id, f); err != nil {
			s.renderAudioError(w, r, id, "Upload stopped at "+f.Filename+": "+err.Error())
			return
		}
	}
	http.Redirect(w, r, "/campaigns/"+id+"#audio", http.StatusSeeOther)
}

func (s *Server) saveClip(r *http.Request, campaign string, f *multipart.FileHeader) error {
	clipID, ext, err := audio.ClipName(f.Filename)
	if err != nil {
		return err
	}
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := src.Close(); err != nil {
			log.Printf("app: close upload: %v", err)
		}
	}()
	data, err := audio.ReadClip(src, audio.DefaultMaxBytes)
	if err != nil {
		return err
	}
	return s.store.SaveAudioClip(r.Context(), campaign, store.AudioClip{ID: clipID, Ext: ext, ContentType: audio.ContentType(ext)}, data)
}

func (s *Server) renderAudioError(w http.ResponseWriter, r *http.Request, id, msg string) {
	d, err := s.campaignPageData(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	d.AudioError = msg
	render(w, r, http.StatusBadRequest, views.CampaignPage(d))
}

func (s *Server) deleteAudioForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteAudioClip(r.Context(), id, r.PathValue("clip")); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id+"#audio", http.StatusSeeOther)
}
