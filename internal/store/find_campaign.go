package store

import (
	"fmt"
	"strings"
)

// FindCampaign picks a campaign by id, or by name ignoring case and
// surrounding spaces.
func FindCampaign(list []Campaign, key string) (Campaign, error) {
	key = strings.TrimSpace(key)
	names := make([]string, 0, len(list))
	for _, c := range list {
		names = append(names, fmt.Sprintf("%q", c.Name))
	}
	if key == "" {
		return Campaign{}, fmt.Errorf("which campaign? pass -campaign with one of: %s", strings.Join(names, ", "))
	}
	var found []Campaign
	for _, c := range list {
		if c.ID == key || strings.EqualFold(c.Name, key) {
			found = append(found, c)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return Campaign{}, fmt.Errorf("no campaign %q; campaigns: %s", key, strings.Join(names, ", "))
	default:
		return Campaign{}, fmt.Errorf("%d campaigns are called %q; pass the id instead", len(found), key)
	}
}
