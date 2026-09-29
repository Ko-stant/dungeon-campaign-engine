package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Passage limits, matching internal/script.
const (
	maxPassageID    = 40
	maxPassageTitle = 200
)

// passageRead records that a read-aloud passage was read at the table (read
// true, the default; reading it again is logged too) or marks it as not read.
// The script lives on the campaign, so the command carries the title for the
// log.
func (a *applier) passageRead(payload json.RawMessage) (string, error) {
	p, err := decode[struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Read  *bool  `json:"read"`
	}](payload)
	if err != nil {
		return "", err
	}
	id, title := strings.TrimSpace(p.ID), strings.TrimSpace(p.Title)
	if id == "" {
		return "", errors.New("a passage id is required")
	}
	if len(id) > maxPassageID || len(title) > maxPassageTitle {
		return "", fmt.Errorf("passage ids are at most %d characters and titles %d", maxPassageID, maxPassageTitle)
	}
	label := strings.TrimSpace(id + " " + title)
	i := slices.Index(a.s.ReadPassages, id)
	if p.Read != nil && !*p.Read {
		if i < 0 {
			return "", fmt.Errorf("%s has not been read", id)
		}
		a.s.ReadPassages = slices.Delete(a.s.ReadPassages, i, i+1)
		return label + " marked as not read", nil
	}
	if i >= 0 {
		return "Read aloud again: " + label, nil
	}
	a.s.ReadPassages = append(a.s.ReadPassages, id)
	return "Read aloud: " + label, nil
}
