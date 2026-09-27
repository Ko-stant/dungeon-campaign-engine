package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// customMonsterPrefix marks custom monster ids in the catalog, keeping them
// apart from the content catalog's ids.
const customMonsterPrefix = "custom-"

// maxMonsterSize bounds a custom monster's footprint in squares.
const maxMonsterSize = 4

// customMonsterDoc is what the custom_monster table stores in doc.
type customMonsterDoc struct {
	Color    string `json:"color"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Body     int    `json:"body"`
	Mind     int    `json:"mind"`
	Attack   int    `json:"attack"`
	Defense  int    `json:"defense"`
	Movement int    `json:"movement"`
	Notes    string `json:"notes,omitempty"`
}

func (s *Server) registerMonsterPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /monsters", s.monstersPage)
	mux.HandleFunc("POST /monsters", s.createMonsterForm)
	mux.HandleFunc("POST /monsters/{id}", s.updateMonsterForm)
	mux.HandleFunc("POST /monsters/{id}/delete", s.deleteMonsterForm)
}

func customMonsterDef(rec store.CustomMonster) (content.MonsterDef, error) {
	var doc customMonsterDoc
	if err := json.Unmarshal(rec.Doc, &doc); err != nil {
		return content.MonsterDef{}, fmt.Errorf("custom monster %s: %w", rec.ID, err)
	}
	return content.MonsterDef{
		ID: customMonsterPrefix + rec.ID, Name: rec.Name,
		Body: doc.Body, Mind: doc.Mind, Attack: doc.Attack, Defense: doc.Defense, Movement: doc.Movement,
		Width: doc.Width, Height: doc.Height, Color: doc.Color, Notes: doc.Notes, Custom: true,
	}, nil
}

// catalogFor returns the content catalog plus the GM's custom monsters.
func (s *Server) catalogFor(ctx context.Context) (*content.Catalog, error) {
	recs, err := s.store.ListCustomMonsters(ctx)
	if err != nil {
		return nil, err
	}
	out := *s.catalog
	out.Monsters = slices.Clone(s.catalog.Monsters)
	for _, rec := range recs {
		def, err := customMonsterDef(rec)
		if err != nil {
			return nil, err
		}
		out.Monsters = append(out.Monsters, def)
	}
	return &out, nil
}

func (s *Server) getCatalog(w http.ResponseWriter, r *http.Request) {
	cat, err := s.catalogFor(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cat)
}

// parseMonsterForm reads and validates a custom monster form.
func parseMonsterForm(r *http.Request) (views.MonsterForm, string, customMonsterDoc, error) {
	form := views.MonsterForm{
		Name: r.PostFormValue("name"), Color: r.PostFormValue("color"),
		Width: r.PostFormValue("width"), Height: r.PostFormValue("height"),
		Body: r.PostFormValue("body"), Mind: r.PostFormValue("mind"), Attack: r.PostFormValue("attack"),
		Defense: r.PostFormValue("defense"), Movement: r.PostFormValue("movement"), Notes: r.PostFormValue("notes"),
	}
	name, err := cleanName(form.Name)
	if err != nil {
		return form, "", customMonsterDoc{}, err
	}
	doc := customMonsterDoc{Color: strings.TrimSpace(form.Color), Notes: strings.TrimSpace(form.Notes)}
	if !maps.IsHexColor(doc.Color) {
		return form, "", doc, errors.New("color must be a #rrggbb color")
	}
	number := func(label, value string, lo, hi int, dst *int) error {
		value = strings.TrimSpace(value)
		if value == "" {
			value = strconv.Itoa(lo)
		}
		n, err := strconv.Atoi(value)
		if err != nil || n < lo || n > hi {
			return fmt.Errorf("%s must be a whole number from %d to %d", label, lo, hi)
		}
		*dst = n
		return nil
	}
	for _, f := range []struct {
		label, value string
		lo, hi       int
		dst          *int
	}{
		{"width", form.Width, 1, maxMonsterSize, &doc.Width},
		{"height", form.Height, 1, maxMonsterSize, &doc.Height},
		{"body", form.Body, 0, 99, &doc.Body},
		{"mind", form.Mind, 0, 99, &doc.Mind},
		{"attack", form.Attack, 0, 99, &doc.Attack},
		{"defend", form.Defense, 0, 99, &doc.Defense},
		{"move", form.Movement, 0, 99, &doc.Movement},
	} {
		if err := number(f.label, f.value, f.lo, f.hi, f.dst); err != nil {
			return form, "", doc, err
		}
	}
	if len(doc.Notes) > 500 {
		return form, "", doc, errors.New("notes must be at most 500 characters")
	}
	return form, name, doc, nil
}

func (s *Server) renderMonstersPage(w http.ResponseWriter, r *http.Request, status int, form views.MonsterForm, formError string) {
	recs, err := s.store.ListCustomMonsters(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	items := make([]views.MonsterItem, 0, len(recs))
	for _, rec := range recs {
		def, err := customMonsterDef(rec)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		items = append(items, views.MonsterItem{ID: rec.ID, Form: views.MonsterFormFromDef(def)})
	}
	render(w, r, status, views.MonstersPage(items, form, formError))
}

func (s *Server) monstersPage(w http.ResponseWriter, r *http.Request) {
	s.renderMonstersPage(w, r, http.StatusOK, views.DefaultMonsterForm(), "")
}

func (s *Server) createMonsterForm(w http.ResponseWriter, r *http.Request) {
	form, name, doc, err := parseMonsterForm(r)
	if err != nil {
		s.renderMonstersPage(w, r, http.StatusBadRequest, form, err.Error())
		return
	}
	data, err := json.Marshal(doc)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if _, err := s.store.CreateCustomMonster(r.Context(), name, data); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/monsters", http.StatusSeeOther)
}

func (s *Server) updateMonsterForm(w http.ResponseWriter, r *http.Request) {
	form, name, doc, err := parseMonsterForm(r)
	if err != nil {
		s.renderMonstersPage(w, r, http.StatusBadRequest, views.DefaultMonsterForm(), fmt.Sprintf("%s: %v", form.Name, err))
		return
	}
	data, err := json.Marshal(doc)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if _, err := s.store.UpdateCustomMonster(r.Context(), r.PathValue("id"), name, data); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/monsters", http.StatusSeeOther)
}

func (s *Server) deleteMonsterForm(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteCustomMonster(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/monsters", http.StatusSeeOther)
}
