package app

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/audio"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// maxAudioUpload bounds one upload request (several clips at once).
const maxAudioUpload = 500 << 20

var audioTypes = map[string]string{
	".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".ogg": "audio/ogg", ".opus": "audio/ogg",
	".wav": "audio/wav", ".webm": "audio/webm", ".flac": "audio/flac",
}

// SetAudioDir sets the folder holding read-aloud audio clips. Without it the
// app lists no clips and refuses uploads.
func (s *Server) SetAudioDir(dir string) {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	s.audio = audio.New(dir)
}

func (s *Server) registerAudio(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/campaigns/{id}/audio", s.listAudio)
	mux.HandleFunc("GET /audio/{campaign}/{file}", s.serveAudio)
	mux.HandleFunc("POST /campaigns/{id}/audio", s.uploadAudioForm)
	mux.HandleFunc("POST /campaigns/{id}/audio/{clip}/delete", s.deleteAudioForm)
}

func clipURL(campaign, file string) string {
	return "/audio/" + campaign + "/" + file
}

// campaignClips maps clip ids to URLs; empty when no audio folder is set.
func (s *Server) campaignClips(campaign string) (map[string]string, error) {
	out := map[string]string{}
	if s.audio == nil {
		return out, nil
	}
	files, err := s.audio.List(campaign)
	if err != nil {
		return nil, err
	}
	for id, file := range files {
		out[id] = clipURL(campaign, file)
	}
	return out, nil
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
func (s *Server) addAudioPageData(d *views.CampaignPageData, titles map[string]string) error {
	d.AudioEnabled = s.audio != nil
	if s.audio == nil {
		return nil
	}
	d.AudioDir = filepath.Join(s.audio.Dir, d.ID)
	files, err := s.audio.List(d.ID)
	if err != nil {
		return err
	}
	for id, file := range files {
		d.AudioClips = append(d.AudioClips, views.AudioClipRow{ID: id, File: file, URL: clipURL(d.ID, file), Passage: clipPassage(id, titles)})
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
	clips, err := s.campaignClips(id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"clips": clips})
}

func (s *Server) serveAudio(w http.ResponseWriter, r *http.Request) {
	if s.audio == nil {
		http.NotFound(w, r)
		return
	}
	name := r.PathValue("file")
	file, err := s.audio.Path(r.PathValue("campaign"), name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if t, ok := audioTypes[strings.ToLower(path.Ext(name))]; ok {
		w.Header().Set("Content-Type", t)
	}
	http.ServeFile(w, r, file)
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
	if s.audio == nil {
		http.Error(w, "no audio folder is set (AUDIO_DIR)", http.StatusServiceUnavailable)
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
		if err := s.saveClip(id, f); err != nil {
			s.renderAudioError(w, r, id, "Upload stopped at "+f.Filename+": "+err.Error())
			return
		}
	}
	http.Redirect(w, r, "/campaigns/"+id+"#audio", http.StatusSeeOther)
}

func (s *Server) saveClip(campaign string, f *multipart.FileHeader) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := src.Close(); err != nil {
			log.Printf("app: close upload: %v", err)
		}
	}()
	return s.audio.Save(campaign, f.Filename, io.Reader(src))
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
	if s.audio == nil {
		http.NotFound(w, r)
		return
	}
	if err := s.audio.Delete(id, r.PathValue("clip")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/campaigns/"+id+"#audio", http.StatusSeeOther)
}
