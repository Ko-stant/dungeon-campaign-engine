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
	"github.com/Ko-stant/dungeon-campaign-engine/internal/dice"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/maps"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/store"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/web/views"
)

// Custom hero class limits.
const (
	maxClassAbilities   = 30
	maxAbilityName      = 80
	maxAbilityText      = 1000
	maxClassDescription = 2000
)

// abilityIDPrefix starts every ability id ("ability-1"); ids are never
// reused within a class, so sessions can refer to an ability by id.
const abilityIDPrefix = "ability-"

// customClassDoc is what the custom_hero_class table stores in doc.
type customClassDoc struct {
	Color       string            `json:"color"`
	Description string            `json:"description,omitempty"`
	Body        int               `json:"body"`
	Mind        int               `json:"mind"`
	Attack      string            `json:"attack"`
	Defense     string            `json:"defense"`
	Movement    string            `json:"movement"`
	Accuracy    int               `json:"accuracy"`
	Mana        int               `json:"mana"`
	Exclusives  []string          `json:"exclusives,omitempty"`
	Abilities   []content.Ability `json:"abilities,omitempty"`
	// NextAbility is the number the next new ability's id gets.
	NextAbility int `json:"nextAbility,omitempty"`
	// Combat (see content.HeroDef); Attack and Defense above are the hit and defense dice.
	CritFrom   int `json:"critFrom,omitempty"`
	Damage     int `json:"damage,omitempty"`
	Avoidance  int `json:"avoidance,omitempty"`
	Mitigation int `json:"mitigation,omitempty"`
	ManaRegen  int `json:"manaRegen,omitempty"`
	// Reach is what the basic attack reaches (content.Reach*).
	Reach string `json:"reach,omitempty"`
}

func (s *Server) registerClassPages(mux routeMux) {
	mux.HandleFunc("GET /classes", s.classesPage)
	mux.HandleFunc("GET /classes/new", s.newClassPage)
	mux.HandleFunc("POST /classes", s.createClassForm)
	mux.HandleFunc("GET /classes/{id}", s.classPage)
	mux.HandleFunc("POST /classes/{id}", s.updateClassForm)
	mux.HandleFunc("POST /classes/{id}/deactivate", s.classActiveForm(false))
	mux.HandleFunc("POST /classes/{id}/reactivate", s.classActiveForm(true))
}

func decodeClassDoc(rec store.CustomHeroClass) (customClassDoc, error) {
	var doc customClassDoc
	if err := json.Unmarshal(rec.Doc, &doc); err != nil {
		return doc, fmt.Errorf("custom hero class %s: %w", rec.ID, err)
	}
	return doc, nil
}

// customHeroClassDef is a stored class as a catalog entry: a base game class
// keeps its catalog id ("barbarian") and its stored content.HeroDef; the GM's
// own classes are "custom-<uuid>".
func customHeroClassDef(rec store.CustomHeroClass) (content.HeroDef, error) {
	if rec.CatalogID != "" {
		var def content.HeroDef
		if err := json.Unmarshal(rec.Doc, &def); err != nil {
			return def, fmt.Errorf("base game class %s: %w", rec.CatalogID, err)
		}
		def.ID, def.Name, def.Custom, def.Inactive = rec.CatalogID, rec.Name, false, !rec.Active
		return def, nil
	}
	doc, err := decodeClassDoc(rec)
	if err != nil {
		return content.HeroDef{}, err
	}
	return content.HeroDef{
		ID: customPrefix + rec.ID, Name: rec.Name, Description: doc.Description,
		Body: doc.Body, Mind: doc.Mind, Custom: true, Color: doc.Color,
		AttackDice: doc.Attack, DefenseDice: doc.Defense, Movement: doc.Movement,
		Accuracy: doc.Accuracy, Mana: doc.Mana, Exclusives: doc.Exclusives, Abilities: doc.Abilities,
		CritFrom: doc.CritFrom, Damage: doc.Damage, Avoidance: doc.Avoidance, Mitigation: doc.Mitigation, ManaRegen: doc.ManaRegen,
		Reach:    doc.Reach,
		Inactive: !rec.Active,
	}, nil
}

// formInt reads a whole number from lo to hi; empty means empty.
func formInt(label, value string, lo, hi, empty int) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return empty, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < lo || n > hi {
		return 0, fmt.Errorf("%s must be a whole number from %d to %d", label, lo, hi)
	}
	return n, nil
}

// formDice reads a dice expression and returns its canonical form.
func formDice(label, value string) (string, error) {
	e, err := dice.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%s: %v", label, err)
	}
	return e.String(), nil
}

// parseClassForm reads and validates a class form. Ability rows with nothing
// typed are dropped; abilities keep their ids and new ones get the next id
// after next (the stored class's NextAbility, or 1).
func parseClassForm(r *http.Request, next int) (views.ClassForm, string, customClassDoc, error) {
	pf := r.PostForm
	form := views.ClassForm{
		Name: pf.Get("name"), Color: pf.Get("color"), Description: pf.Get("description"),
		Body: pf.Get("body"), Mind: pf.Get("mind"), Attack: pf.Get("attack"), Defense: pf.Get("defense"),
		Movement: pf.Get("movement"), Accuracy: pf.Get("accuracy"), Mana: pf.Get("mana"),
		Exclusives: pf["exclusives"],
		CritFrom:   pf.Get("crit_from"), Damage: pf.Get("damage"), Avoidance: pf.Get("avoidance"),
		Mitigation: pf.Get("mitigation"), ManaRegen: pf.Get("mana_regen"), Reach: pf.Get("reach"),
	}
	ids, names, kinds, manas, cooldowns, texts := pf["ability_id"], pf["ability_name"], pf["ability_kind"], pf["ability_mana"], pf["ability_cooldown"], pf["ability_text"]
	at := func(list []string, i int) string {
		if i < len(list) {
			return list[i]
		}
		return ""
	}
	for i := range names {
		row := views.AbilityRow{ID: at(ids, i), Name: names[i], Kind: at(kinds, i), Mana: at(manas, i), Cooldown: at(cooldowns, i), Text: at(texts, i)}
		if strings.TrimSpace(row.Name+row.Mana+row.Cooldown+row.Text) == "" {
			continue
		}
		form.Abilities = append(form.Abilities, row)
	}

	fail := func(err error) (views.ClassForm, string, customClassDoc, error) {
		return form, "", customClassDoc{}, err
	}
	name, err := cleanName(form.Name)
	if err != nil {
		return fail(err)
	}
	doc := customClassDoc{Color: strings.TrimSpace(form.Color), Description: strings.TrimSpace(form.Description)}
	if !maps.IsHexColor(doc.Color) {
		return fail(errors.New("color must be a #rrggbb color"))
	}
	if len(doc.Description) > maxClassDescription {
		return fail(fmt.Errorf("description must be at most %d characters", maxClassDescription))
	}
	for _, f := range []struct {
		label, value    string
		lo, hi, missing int
		dst             *int
	}{
		{"body", form.Body, 1, 999, 1, &doc.Body},
		{"mind", form.Mind, 0, 999, 0, &doc.Mind},
		{"accuracy", form.Accuracy, 0, 99, 0, &doc.Accuracy},
		{"mana", form.Mana, 0, 999, 0, &doc.Mana},
		{"crit range (lowest d20 roll that crits)", form.CritFrom, 2, 20, 20, &doc.CritFrom},
		{"damage", form.Damage, 0, 99, 0, &doc.Damage},
		{"avoidance", form.Avoidance, 0, 99, 0, &doc.Avoidance},
		{"mitigation", form.Mitigation, 0, 99, 0, &doc.Mitigation},
		{"mana per fight round", form.ManaRegen, 0, 99, 0, &doc.ManaRegen},
	} {
		if *f.dst, err = formInt(f.label, f.value, f.lo, f.hi, f.missing); err != nil {
			return fail(err)
		}
	}
	for _, f := range []struct {
		label, value string
		dst          *string
	}{
		{"hit dice", form.Attack, &doc.Attack},
		{"defense dice", form.Defense, &doc.Defense},
		{"movement", form.Movement, &doc.Movement},
	} {
		if *f.dst, err = formDice(f.label, f.value); err != nil {
			return fail(err)
		}
	}
	switch doc.Reach = form.Reach; {
	case doc.Reach == "":
		doc.Reach = content.ReachAdjacent
	case !slices.Contains(content.Reaches, doc.Reach):
		return fail(fmt.Errorf("reach must be one of %s", strings.Join(content.Reaches, ", ")))
	}
	for _, tag := range form.Exclusives {
		if !slices.Contains(content.Exclusives, tag) {
			return fail(fmt.Errorf("unknown class exclusive %q", tag))
		}
	}
	for _, tag := range content.Exclusives {
		if slices.Contains(form.Exclusives, tag) {
			doc.Exclusives = append(doc.Exclusives, tag)
		}
	}

	if len(form.Abilities) > maxClassAbilities {
		return fail(fmt.Errorf("a class may have at most %d abilities", maxClassAbilities))
	}
	doc.NextAbility = max(next, 1)
	for _, row := range form.Abilities {
		if n, err := strconv.Atoi(strings.TrimPrefix(row.ID, abilityIDPrefix)); err == nil && strings.HasPrefix(row.ID, abilityIDPrefix) && n >= doc.NextAbility {
			doc.NextAbility = n + 1
		}
	}
	used := map[string]bool{}
	for _, row := range form.Abilities {
		a := content.Ability{ID: row.ID, Name: strings.TrimSpace(row.Name), Kind: row.Kind, Text: strings.TrimSpace(row.Text)}
		if a.Name == "" {
			if row.ID != "" {
				// Clearing an existing ability's name removes it.
				continue
			}
			return fail(errors.New("every ability needs a name"))
		}
		label := "ability " + strconv.Quote(a.Name)
		if len(a.Name) > maxAbilityName {
			return fail(fmt.Errorf("%s: the name must be at most %d characters", label, maxAbilityName))
		}
		if len(a.Text) > maxAbilityText {
			return fail(fmt.Errorf("%s: the text must be at most %d characters", label, maxAbilityText))
		}
		if !slices.Contains(content.AbilityKinds, a.Kind) {
			return fail(fmt.Errorf("%s: kind must be one of %s", label, strings.Join(content.AbilityKinds, ", ")))
		}
		if a.ManaCost, err = formInt(label+" mana cost", row.Mana, 0, 999, 0); err != nil {
			return fail(err)
		}
		if a.Cooldown, err = formInt(label+" cooldown", row.Cooldown, 0, 99, 0); err != nil {
			return fail(err)
		}
		if !strings.HasPrefix(a.ID, abilityIDPrefix) || used[a.ID] {
			a.ID = abilityIDPrefix + strconv.Itoa(doc.NextAbility)
			doc.NextAbility++
		}
		used[a.ID] = true
		doc.Abilities = append(doc.Abilities, a)
	}
	return form, name, doc, nil
}

func (s *Server) renderClassesPage(w http.ResponseWriter, r *http.Request, status int, pageError string) {
	recs, err := s.store.ListCustomHeroClasses(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	items := make([]views.ClassItem, 0, len(recs))
	for _, rec := range recs {
		def, err := customHeroClassDef(rec)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		items = append(items, views.ClassItem{ID: rec.ID, Class: def})
	}
	render(w, r, status, views.ClassesPage(items, pageError))
}

func (s *Server) classesPage(w http.ResponseWriter, r *http.Request) {
	s.renderClassesPage(w, r, http.StatusOK, "")
}

func (s *Server) newClassPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, views.ClassEditPage("", views.DefaultClassForm(), true, ""))
}

func (s *Server) classPage(w http.ResponseWriter, r *http.Request) {
	rec, err := s.store.GetCustomHeroClass(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if rec.CatalogID != "" {
		// Base game classes are read-only: their stats show on the list.
		http.Redirect(w, r, "/classes", http.StatusSeeOther)
		return
	}
	def, err := customHeroClassDef(rec)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	render(w, r, http.StatusOK, views.ClassEditPage(rec.ID, views.ClassFormFromDef(def), rec.Active, ""))
}

func (s *Server) createClassForm(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	form, name, doc, err := parseClassForm(r, 1)
	if err != nil {
		render(w, r, http.StatusBadRequest, views.ClassEditPage("", form, true, err.Error()))
		return
	}
	s.saveClass(w, r, doc, func(data json.RawMessage) error {
		_, err := s.store.CreateCustomHeroClass(r.Context(), name, data)
		return err
	})
}

func (s *Server) updateClassForm(w http.ResponseWriter, r *http.Request) {
	if !parseForm(w, r) {
		return
	}
	id := r.PathValue("id")
	rec, err := s.store.GetCustomHeroClass(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if rec.CatalogID != "" {
		http.Error(w, "The base game's classes can't be edited; deactivate one instead.", http.StatusBadRequest)
		return
	}
	old, err := decodeClassDoc(rec)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	form, name, doc, err := parseClassForm(r, old.NextAbility)
	if err != nil {
		render(w, r, http.StatusBadRequest, views.ClassEditPage(id, form, rec.Active, err.Error()))
		return
	}
	s.saveClass(w, r, doc, func(data json.RawMessage) error {
		_, err := s.store.UpdateCustomHeroClass(r.Context(), id, name, data)
		return err
	})
}

func (s *Server) saveClass(w http.ResponseWriter, r *http.Request, doc customClassDoc, save func(json.RawMessage) error) {
	data, err := json.Marshal(doc)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := save(data); err != nil {
		writeStoreError(w, err)
		return
	}
	http.Redirect(w, r, "/classes", http.StatusSeeOther)
}

// classActiveForm deactivates or reactivates a class. Classes are never
// deleted: a deactivated one is left out of the new-hero picker, and heroes
// who already have it keep it.
func (s *Server) classActiveForm(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.store.SetCustomHeroClassActive(r.Context(), r.PathValue("id"), active); err != nil {
			writeStoreError(w, err)
			return
		}
		http.Redirect(w, r, "/classes", http.StatusSeeOther)
	}
}

// customClasses returns the classes kept in the database (the base game's
// and the GM's own) as catalog entries.
func (s *Server) customClasses(ctx context.Context) ([]content.HeroDef, error) {
	recs, err := s.store.ListCustomHeroClasses(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]content.HeroDef, 0, len(recs))
	for _, rec := range recs {
		def, err := customHeroClassDef(rec)
		if err != nil {
			return nil, err
		}
		out = append(out, def)
	}
	return out, nil
}
