package app

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/a-h/templ"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

func (s *Server) registerPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /maps", s.mapsPage)
	mux.HandleFunc("POST /maps", s.createBoardForm)
	mux.HandleFunc("GET /maps/{id}/edit", s.mapEditorPage)
}

func render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := c.Render(r.Context(), w); err != nil {
		log.Printf("app: render: %v", err)
	}
}

func (s *Server) renderMapsPage(w http.ResponseWriter, r *http.Request, status int, form views.MapCreateForm) {
	boards, err := s.store.ListBoards(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	items := make([]views.MapListItem, 0, len(boards))
	for _, b := range boards {
		items = append(items, views.MapListItem{ID: b.ID, Name: b.Name, Width: b.Width, Height: b.Height, UpdatedAt: b.UpdatedAt})
	}
	if err := s.boardDeleteDetails(r.Context(), items); err != nil {
		writeStoreError(w, err)
		return
	}
	groups, err := s.campaignMapGroups(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	render(w, r, status, views.MapsPage(items, groups, form))
}

func (s *Server) mapsPage(w http.ResponseWriter, r *http.Request) {
	s.renderMapsPage(w, r, http.StatusOK, views.MapCreateForm{Width: "26", Height: "19"})
}

func (s *Server) createBoardForm(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	form := views.MapCreateForm{Name: r.PostFormValue("name"), Width: r.PostFormValue("width"), Height: r.PostFormValue("height")}
	width, errW := strconv.Atoi(form.Width)
	height, errH := strconv.Atoi(form.Height)
	if errW != nil || errH != nil {
		form.Error = "Width and height must be whole numbers."
		s.renderMapsPage(w, r, http.StatusBadRequest, form)
		return
	}

	created, err := s.newBoard(r.Context(), form.Name, width, height)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrInUse) {
			writeStoreError(w, err)
			return
		}
		form.Error = err.Error()
		s.renderMapsPage(w, r, http.StatusBadRequest, form)
		return
	}
	http.Redirect(w, r, "/maps/"+created.ID+"/edit", http.StatusSeeOther)
}

func (s *Server) mapEditorPage(w http.ResponseWriter, r *http.Request) {
	b, err := s.store.GetBoard(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	render(w, r, http.StatusOK, views.MapEditorPage(b.ID, b.Name))
}
