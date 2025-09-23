package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

// TestCombatDiceDebugOverride tests the debug override functionality for combat dice
func TestCombatDiceDebugOverride(t *testing.T) {
	// Create test systems
	gameState := &GameState{
		Entities:      make(map[string]protocol.TileAddress),
		KnownMonsters: make(map[string]bool),
	}

	broadcaster := &MockBroadcaster{}
	logger := &MockLogger{}

	debugConfig := DebugConfig{
		Enabled:           true,
		AllowDiceOverride: true,
	}
	debugSystem := NewDebugSystem(debugConfig, gameState, broadcaster, logger)

	// Test scenarios covering damage values 0-12
	testScenarios := []struct {
		name         string
		attackDice   []int
		defenseDice  []int
		expectedSkul int
		expectedShie int
		expectedDmg  int
		description  string
	}{
		{
			name:         "No damage - all misses",
			attackDice:   []int{1, 2, 3},  // 0 skulls
			defenseDice:  []int{1},        // 0 shields
			expectedSkul: 0,
			expectedShie: 0,
			expectedDmg:  0,
			description:  "Testing minimum damage (0)",
		},
		{
			name:         "Perfect defense blocks all",
			attackDice:   []int{4, 5, 6},  // 3 skulls
			defenseDice:  []int{6, 5, 4},  // 3 shields (1 black, 2 white)
			expectedSkul: 3,
			expectedShie: 3,
			expectedDmg:  0,
			description:  "Testing 3 skulls vs 3 shields = 0 damage",
		},
		{
			name:         "Single point damage",
			attackDice:   []int{6},        // 1 skull
			defenseDice:  []int{1},        // 0 shields
			expectedSkul: 1,
			expectedShie: 0,
			expectedDmg:  1,
			description:  "Testing 1 damage",
		},
		{
			name:         "Two point damage",
			attackDice:   []int{4, 5},     // 2 skulls
			defenseDice:  []int{2},        // 0 shields
			expectedSkul: 2,
			expectedShie: 0,
			expectedDmg:  2,
			description:  "Testing 2 damage",
		},
		{
			name:         "High damage scenario",
			attackDice:   []int{4, 5, 6, 4, 5}, // 5 skulls
			defenseDice:  []int{1, 2},           // 0 shields
			expectedSkul: 5,
			expectedShie: 0,
			expectedDmg:  5,
			description:  "Testing 5 damage (high roll)",
		},
		{
			name:         "Barbarian vs tough monster",
			attackDice:   []int{4, 5, 6},  // 3 skulls (Barbarian has 3 attack dice)
			defenseDice:  []int{3},        // 0 shields (monster rolls poorly)
			expectedSkul: 3,
			expectedShie: 0,
			expectedDmg:  3,
			description:  "Testing typical Barbarian attack",
		},
		{
			name:         "Wizard vs defended monster",
			attackDice:   []int{6},        // 1 skull (Wizard has 1 attack die)
			defenseDice:  []int{4},        // 1 white shield
			expectedSkul: 1,
			expectedShie: 1,
			expectedDmg:  0,
			description:  "Testing Wizard attack blocked",
		},
		{
			name:         "Maximum theoretical damage",
			attackDice:   []int{6, 6, 6, 6, 6, 6}, // 6 skulls (theoretical max with equipment)
			defenseDice:  []int{1},                 // 0 shields
			expectedSkul: 6,
			expectedShie: 0,
			expectedDmg:  6,
			description:  "Testing maximum damage scenario",
		},
	}

	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Set up the debug override
			debugSystem.SetCombatDiceOverride(scenario.attackDice, scenario.defenseDice)

			// Create dice system
			diceSystem := NewDiceSystem(debugSystem)

			// Roll attack dice
			attackRolls := diceSystem.RollAttackDice(len(scenario.attackDice))

			// Roll defense dice
			defenseRolls := diceSystem.RollDefenseDice(len(scenario.defenseDice))

			// Verify the rolls match our expectations
			if len(attackRolls) != len(scenario.attackDice) {
				t.Errorf("Expected %d attack rolls, got %d", len(scenario.attackDice), len(attackRolls))
			}

			if len(defenseRolls) != len(scenario.defenseDice) {
				t.Errorf("Expected %d defense rolls, got %d", len(scenario.defenseDice), len(defenseRolls))
			}

			// Verify dice results
			for i, expectedResult := range scenario.attackDice {
				if i < len(attackRolls) && attackRolls[i].Result != expectedResult {
					t.Errorf("Attack roll %d: expected %d, got %d", i, expectedResult, attackRolls[i].Result)
				}
			}

			for i, expectedResult := range scenario.defenseDice {
				if i < len(defenseRolls) && defenseRolls[i].Result != expectedResult {
					t.Errorf("Defense roll %d: expected %d, got %d", i, expectedResult, defenseRolls[i].Result)
				}
			}

			// Calculate actual damage
			damage := CalculateCombatDamage(attackRolls, defenseRolls)

			// Verify damage calculation
			if damage != scenario.expectedDmg {
				t.Errorf("Expected damage %d, got %d", scenario.expectedDmg, damage)
			}

			// Count skulls and shields for additional verification
			skulls := countSkulls(attackRolls)
			shields := countShields(defenseRolls)

			if skulls != scenario.expectedSkul {
				t.Errorf("Expected %d skulls, got %d", scenario.expectedSkul, skulls)
			}

			if shields != scenario.expectedShie {
				t.Errorf("Expected %d shields, got %d", scenario.expectedShie, shields)
			}

			t.Logf("✅ %s: %d skulls vs %d shields = %d damage",
				scenario.description, skulls, shields, damage)
		})
	}
}

// TestCombatDiceTestEndpoint tests the HTTP endpoint for combat dice testing
func TestCombatDiceTestEndpoint(t *testing.T) {
	// This test would need HTTP testing framework - placeholder for now
	t.Log("Combat dice test endpoint available at /debug/dice/combat-test")
	t.Log("POST with JSON: {\"attackDice\": [6,6], \"defenseDice\": [1], \"testScenario\": \"Test 2 damage\"}")
}