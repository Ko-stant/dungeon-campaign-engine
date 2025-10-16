package protocol

import (
	"encoding/json"
	"testing"
)

func TestRequestSelectStartingPosition_Serialization(t *testing.T) {
	req := RequestSelectStartingPosition{
		X: 5,
		Y: 10,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RequestSelectStartingPosition
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.X != 5 || decoded.Y != 10 {
		t.Errorf("Expected position (5, 10), got (%d, %d)", decoded.X, decoded.Y)
	}
}

func TestRequestSelectMonster_Serialization(t *testing.T) {
	req := RequestSelectMonster{
		MonsterID: "monster-orc-1",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RequestSelectMonster
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.MonsterID != "monster-orc-1" {
		t.Errorf("Expected monster ID 'monster-orc-1', got '%s'", decoded.MonsterID)
	}
}

func TestRequestMoveMonster_Serialization(t *testing.T) {
	req := RequestMoveMonster{
		MonsterID: "monster-orc-1",
		ToX:       5,
		ToY:       10,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RequestMoveMonster
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.MonsterID != "monster-orc-1" {
		t.Errorf("Expected monster ID 'monster-orc-1', got '%s'", decoded.MonsterID)
	}
	if decoded.ToX != 5 || decoded.ToY != 10 {
		t.Errorf("Expected position (5, 10), got (%d, %d)", decoded.ToX, decoded.ToY)
	}
}

func TestRequestMonsterAttack_Serialization(t *testing.T) {
	req := RequestMonsterAttack{
		MonsterID: "monster-orc-1",
		TargetID:  "hero-barbarian",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RequestMonsterAttack
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.MonsterID != "monster-orc-1" {
		t.Errorf("Expected monster ID 'monster-orc-1', got '%s'", decoded.MonsterID)
	}
	if decoded.TargetID != "hero-barbarian" {
		t.Errorf("Expected target ID 'hero-barbarian', got '%s'", decoded.TargetID)
	}
}

func TestRequestUseMonsterAbility_Serialization(t *testing.T) {
	targetX := 10
	targetY := 15

	req := RequestUseMonsterAbility{
		MonsterID: "monster-chaos-wizard",
		AbilityID: "dread_spell_fireball",
		TargetID:  "hero-barbarian",
		TargetX:   &targetX,
		TargetY:   &targetY,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RequestUseMonsterAbility
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.MonsterID != "monster-chaos-wizard" {
		t.Errorf("Expected monster ID 'monster-chaos-wizard', got '%s'", decoded.MonsterID)
	}
	if decoded.AbilityID != "dread_spell_fireball" {
		t.Errorf("Expected ability ID 'dread_spell_fireball', got '%s'", decoded.AbilityID)
	}
	if decoded.TargetID != "hero-barbarian" {
		t.Errorf("Expected target ID 'hero-barbarian', got '%s'", decoded.TargetID)
	}
	if decoded.TargetX == nil || *decoded.TargetX != 10 {
		t.Error("Expected TargetX to be 10")
	}
	if decoded.TargetY == nil || *decoded.TargetY != 15 {
		t.Error("Expected TargetY to be 15")
	}
}

func TestTurnPhaseChanged_Serialization(t *testing.T) {
	patch := TurnPhaseChanged{
		CurrentPhase:       "hero_phase_active",
		CycleNumber:        2,
		ActiveHeroPlayerID: "player-1",
		ElectedPlayerID:    "",
		HeroesActedIDs:     []string{"player-1", "player-2"},
		EligibleHeroIDs:    []string{"player-3", "player-4"},
	}

	data, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded TurnPhaseChanged
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.CurrentPhase != "hero_phase_active" {
		t.Errorf("Expected phase 'hero_phase_active', got '%s'", decoded.CurrentPhase)
	}
	if decoded.CycleNumber != 2 {
		t.Errorf("Expected cycle 2, got %d", decoded.CycleNumber)
	}
	if decoded.ActiveHeroPlayerID != "player-1" {
		t.Errorf("Expected active player 'player-1', got '%s'", decoded.ActiveHeroPlayerID)
	}
	if len(decoded.HeroesActedIDs) != 2 {
		t.Errorf("Expected 2 heroes acted, got %d", len(decoded.HeroesActedIDs))
	}
	if len(decoded.EligibleHeroIDs) != 2 {
		t.Errorf("Expected 2 eligible heroes, got %d", len(decoded.EligibleHeroIDs))
	}
}

func TestQuestSetupStateChanged_Serialization(t *testing.T) {
	patch := QuestSetupStateChanged{
		PlayersReady: map[string]bool{
			"player-1": true,
			"player-2": false,
			"player-3": true,
		},
		PlayerStartPositions: map[string]StartPositionInfo{
			"player-1": {X: 5, Y: 10},
			"player-3": {X: 6, Y: 10},
		},
		AllPlayersReady: false,
	}

	data, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded QuestSetupStateChanged
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.PlayersReady) != 3 {
		t.Errorf("Expected 3 players, got %d", len(decoded.PlayersReady))
	}
	if !decoded.PlayersReady["player-1"] {
		t.Error("Expected player-1 to be ready")
	}
	if decoded.PlayersReady["player-2"] {
		t.Error("Expected player-2 to not be ready")
	}
	if len(decoded.PlayerStartPositions) != 2 {
		t.Errorf("Expected 2 start positions, got %d", len(decoded.PlayerStartPositions))
	}
	if decoded.AllPlayersReady {
		t.Error("Expected AllPlayersReady to be false")
	}
}

func TestMonsterTurnStateChanged_Serialization(t *testing.T) {
	patch := MonsterTurnStateChanged{
		MonsterID:         "monster-orc-1",
		EntityID:          "entity-orc-1",
		TurnNumber:        1,
		CurrentPosition:   TileAddress{SegmentID: "seg-1", X: 5, Y: 10},
		FixedMovement:     8,
		MovementRemaining: 5,
		MovementUsed:      3,
		HasMoved:          true,
		ActionTaken:       false,
		ActionType:        "",
		AttackDice:        3,
		DefenseDice:       2,
		BodyPoints:        5,
		CurrentBody:       4,
		SpecialAbilities: []MonsterAbilityLite{
			{
				ID:             "dread_spell_fireball",
				Name:           "Fireball",
				Type:           "dread_spell",
				UsesPerTurn:    1,
				UsesPerQuest:   3,
				UsesLeftQuest:  2,
				RequiresAction: true,
				Range:          6,
				Description:    "Cast a fireball",
				EffectDetails:  map[string]interface{}{"damage": "2d6"},
			},
		},
		AbilityUsageThisTurn: map[string]int{
			"dread_spell_fireball": 1,
		},
		ActiveEffectsCount: 2,
	}

	data, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MonsterTurnStateChanged
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.MonsterID != "monster-orc-1" {
		t.Errorf("Expected monster ID 'monster-orc-1', got '%s'", decoded.MonsterID)
	}
	if decoded.FixedMovement != 8 {
		t.Errorf("Expected fixed movement 8, got %d", decoded.FixedMovement)
	}
	if decoded.MovementRemaining != 5 {
		t.Errorf("Expected movement remaining 5, got %d", decoded.MovementRemaining)
	}
	if !decoded.HasMoved {
		t.Error("Expected HasMoved to be true")
	}
	if decoded.ActionTaken {
		t.Error("Expected ActionTaken to be false")
	}
	if len(decoded.SpecialAbilities) != 1 {
		t.Errorf("Expected 1 special ability, got %d", len(decoded.SpecialAbilities))
	}
	if decoded.SpecialAbilities[0].ID != "dread_spell_fireball" {
		t.Errorf("Expected ability ID 'dread_spell_fireball', got '%s'", decoded.SpecialAbilities[0].ID)
	}
	if decoded.AbilityUsageThisTurn["dread_spell_fireball"] != 1 {
		t.Error("Expected ability usage count to be 1")
	}
	if decoded.ActiveEffectsCount != 2 {
		t.Errorf("Expected 2 active effects, got %d", decoded.ActiveEffectsCount)
	}
}

func TestMonsterSelectionChanged_Serialization(t *testing.T) {
	patch := MonsterSelectionChanged{
		SelectedMonsterID: "monster-orc-1",
	}

	data, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MonsterSelectionChanged
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.SelectedMonsterID != "monster-orc-1" {
		t.Errorf("Expected selected monster 'monster-orc-1', got '%s'", decoded.SelectedMonsterID)
	}
}

func TestSnapshotWithExtensions_Serialization(t *testing.T) {
	snapshot := Snapshot{
		MapID:        "test-map",
		PackID:       "base-pack",
		Turn:         5,
		LastEventID:  123,
		MapWidth:     20,
		MapHeight:    20,
		RegionsCount: 10,

		// Dynamic turn order state
		TurnPhase:          "gm_phase",
		CycleNumber:        2,
		ActiveHeroPlayerID: "",
		ElectedPlayerID:    "",
		HeroesActedIDs:     []string{"player-1", "player-2", "player-3"},

		// Quest setup state
		PlayersReady: map[string]bool{
			"player-1": true,
			"player-2": true,
		},
		PlayerStartPositions: map[string]StartPositionInfo{
			"player-1": {X: 5, Y: 10},
			"player-2": {X: 6, Y: 10},
		},

		// Monster turn states
		MonsterTurnStates: map[string]MonsterTurnStateLite{
			"monster-orc-1": {
				MonsterID:         "monster-orc-1",
				EntityID:          "entity-orc-1",
				TurnNumber:        1,
				CurrentPosition:   TileAddress{SegmentID: "seg-1", X: 5, Y: 10},
				FixedMovement:     8,
				MovementRemaining: 5,
				HasMoved:          true,
				ActionTaken:       false,
				AttackDice:        3,
				DefenseDice:       2,
				BodyPoints:        5,
				CurrentBody:       4,
			},
		},
		SelectedMonsterID: "monster-orc-1",
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Snapshot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.TurnPhase != "gm_phase" {
		t.Errorf("Expected phase 'gm_phase', got '%s'", decoded.TurnPhase)
	}
	if decoded.CycleNumber != 2 {
		t.Errorf("Expected cycle 2, got %d", decoded.CycleNumber)
	}
	if len(decoded.HeroesActedIDs) != 3 {
		t.Errorf("Expected 3 heroes acted, got %d", len(decoded.HeroesActedIDs))
	}
	if len(decoded.PlayersReady) != 2 {
		t.Errorf("Expected 2 players ready, got %d", len(decoded.PlayersReady))
	}
	if len(decoded.PlayerStartPositions) != 2 {
		t.Errorf("Expected 2 start positions, got %d", len(decoded.PlayerStartPositions))
	}
	if len(decoded.MonsterTurnStates) != 1 {
		t.Errorf("Expected 1 monster turn state, got %d", len(decoded.MonsterTurnStates))
	}
	if decoded.SelectedMonsterID != "monster-orc-1" {
		t.Errorf("Expected selected monster 'monster-orc-1', got '%s'", decoded.SelectedMonsterID)
	}

	// Verify nested monster state
	monsterState := decoded.MonsterTurnStates["monster-orc-1"]
	if monsterState.FixedMovement != 8 {
		t.Errorf("Expected fixed movement 8, got %d", monsterState.FixedMovement)
	}
	if !monsterState.HasMoved {
		t.Error("Expected HasMoved to be true")
	}
}

func TestAllMonsterStatesSync_Serialization(t *testing.T) {
	patch := AllMonsterStatesSync{
		MonsterStates: map[string]*MonsterTurnStateChanged{
			"monster-orc-1": {
				MonsterID:         "monster-orc-1",
				EntityID:          "entity-orc-1",
				TurnNumber:        1,
				CurrentPosition:   TileAddress{SegmentID: "seg-1", X: 5, Y: 10},
				FixedMovement:     8,
				MovementRemaining: 8,
				HasMoved:          false,
				ActionTaken:       false,
				AttackDice:        3,
				DefenseDice:       2,
				BodyPoints:        5,
				CurrentBody:       5,
			},
			"monster-goblin-1": {
				MonsterID:         "monster-goblin-1",
				EntityID:          "entity-goblin-1",
				TurnNumber:        1,
				CurrentPosition:   TileAddress{SegmentID: "seg-1", X: 6, Y: 10},
				FixedMovement:     10,
				MovementRemaining: 10,
				HasMoved:          false,
				ActionTaken:       false,
				AttackDice:        2,
				DefenseDice:       2,
				BodyPoints:        3,
				CurrentBody:       3,
			},
		},
	}

	data, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded AllMonsterStatesSync
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.MonsterStates) != 2 {
		t.Errorf("Expected 2 monster states, got %d", len(decoded.MonsterStates))
	}

	orcState := decoded.MonsterStates["monster-orc-1"]
	if orcState == nil {
		t.Fatal("Expected orc state to exist")
	}
	if orcState.FixedMovement != 8 {
		t.Errorf("Expected orc fixed movement 8, got %d", orcState.FixedMovement)
	}

	goblinState := decoded.MonsterStates["monster-goblin-1"]
	if goblinState == nil {
		t.Fatal("Expected goblin state to exist")
	}
	if goblinState.FixedMovement != 10 {
		t.Errorf("Expected goblin fixed movement 10, got %d", goblinState.FixedMovement)
	}
}
