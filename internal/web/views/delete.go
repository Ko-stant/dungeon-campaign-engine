package views

import (
	"fmt"
	"strings"
)

// The questions the Yes/No dialog asks before a delete (data-confirm), naming
// what goes with the item.

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// joinNames lists names as "A", "A and B" or "A, B and C".
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

func campaignDeleteWarning(d CampaignPageData) string {
	gone := []string{"its heroes and their items", "the chapter list", "the loot list", "the monster stat lines", "the read-aloud script"}
	if len(d.Sessions) > 0 {
		gone = append(gone, plural(len(d.Sessions), "session", "sessions")+" with their logs")
	}
	if len(d.AudioClips) > 0 {
		gone = append(gone, plural(len(d.AudioClips), "audio clip", "audio clips"))
	}
	return fmt.Sprintf("Delete the campaign %s for good? This deletes %s. The maps stay in Maps.", d.Name, joinNames(gone))
}

func sessionDeleteWarning(s SessionLink) string {
	return fmt.Sprintf("Delete the session %s and its log (%s) for good? The campaign's heroes keep what earlier completed quests gave them.",
		s.Name, plural(int(s.Events), "event", "events"))
}

func boardDeleteWarning(b MapListItem) string {
	var msg strings.Builder
	fmt.Fprintf(&msg, "Delete the board %s for good?", b.Name)
	if len(b.Quests) > 0 {
		fmt.Fprintf(&msg, " Its %s (%s) go with it.", map[bool]string{true: "quest", false: "quests"}[len(b.Quests) == 1], joinNames(b.Quests))
	}
	if len(b.Campaigns) > 0 {
		fmt.Fprintf(&msg, " They are removed from the chapters of %s.", joinNames(b.Campaigns))
	}
	msg.WriteString(" Sessions already started keep their copy of the map.")
	return msg.String()
}
