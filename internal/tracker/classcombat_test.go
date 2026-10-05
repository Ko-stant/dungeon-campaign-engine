package tracker

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/content"
)

func TestClassCombatCarriesTheReach(t *testing.T) {
	class := content.HeroDef{Custom: true, AttackDice: "1d20", Reach: content.ReachSight}
	if c := ClassCombat(class); c == nil || c.Reach != content.ReachSight {
		t.Fatalf("combat: %+v", c)
	}
	class.Reach = ""
	if c := ClassCombat(class); c.Reach != content.ReachAdjacent {
		t.Errorf("no reach means adjacent: %q", c.Reach)
	}
}
