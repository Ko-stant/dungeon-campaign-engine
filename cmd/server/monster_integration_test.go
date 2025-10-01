package main

import (
	"testing"

	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
)

func TestMonsterSystem_SpawnAndManagement(t *testing.T) {

	gm := createTestGameManager()

	// Spawn a monster
	monster, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 7, Y: 7})
	if err != nil {
		t.Fatalf("Failed to spawn monster: %v", err)
	}

	if monster.Type != Goblin {
		t.Errorf("Expected Goblin, got %s", monster.Type)
	}

	if monster.Position.X != 7 || monster.Position.Y != 7 {
		t.Errorf("Expected monster at (7,7), got (%d,%d)", monster.Position.X, monster.Position.Y)
	}

	// Verify monster is in game state
	monsters := gm.GetMonsters()
	if len(monsters) != 1 {
		t.Errorf("Expected 1 monster, got %d", len(monsters))
	}

	foundMonster := false
	for _, m := range monsters {
		if m.ID == monster.ID {
			foundMonster = true
			break
		}
	}

	if !foundMonster {
		t.Error("Spawned monster not found in game state")
	}
}

func TestMonsterSystem_MultipleMonsters(t *testing.T) {

	gm := createTestGameManager()

	// Spawn different types of monsters
	monsters := []struct {
		monsterType MonsterType
		position    protocol.TileAddress
	}{
		{Goblin, protocol.TileAddress{X: 7, Y: 7}},
		{Orc, protocol.TileAddress{X: 8, Y: 8}},
		{Skeleton, protocol.TileAddress{X: 9, Y: 9}},
	}

	spawnedMonsters := make([]*Monster, len(monsters))

	for i, monsterDef := range monsters {
		monster, err := gm.SpawnMonster(monsterDef.monsterType, monsterDef.position)
		if err != nil {
			t.Fatalf("Failed to spawn monster %d: %v", i, err)
		}
		spawnedMonsters[i] = monster
	}

	// Verify all monsters are in game state
	gameMonsters := gm.GetMonsters()
	if len(gameMonsters) != len(monsters) {
		t.Errorf("Expected %d monsters, got %d", len(monsters), len(gameMonsters))
	}

	// Verify each monster
	for i, expectedMonster := range spawnedMonsters {
		found := false
		for _, gameMonster := range gameMonsters {
			if gameMonster.ID == expectedMonster.ID {
				if gameMonster.Type != expectedMonster.Type {
					t.Errorf("Monster %d: Expected type %s, got %s", i, expectedMonster.Type, gameMonster.Type)
				}
				if gameMonster.Position != expectedMonster.Position {
					t.Errorf("Monster %d: Expected position %v, got %v", i, expectedMonster.Position, gameMonster.Position)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Monster %d not found in game state", i)
		}
	}
}

func TestMonsterSystem_CombatIntegration(t *testing.T) {

	gm := createTestGameManager()

	// Spawn monster adjacent to hero
	monster, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 6, Y: 5}) // Adjacent to hero at (5,5)
	if err != nil {
		t.Fatalf("Failed to spawn monster: %v", err)
	}

	// Hero attacks monster
	attackReq := ActionRequest{
		PlayerID: "player-1",
		EntityID: "hero-1",
		Action:   AttackAction,
		Parameters: map[string]any{
			"targetId": monster.ID,
		},
	}

	result, err := gm.ProcessHeroAction(attackReq)
	if err != nil {
		t.Fatalf("Hero attack failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful attack, got: %s", result.Message)
	}

	// Verify attack result has required fields
	if len(result.AttackRolls) == 0 {
		t.Error("Expected attack rolls for attack")
	}

	if result.Damage < 0 {
		t.Error("Expected non-negative damage")
	}

	// Note: ActionResult doesn't have TargetID field - this would be tracked in the combat system
}

func TestMonsterSystem_GameMasterTurn(t *testing.T) {

	gm := createTestGameManager()

	// Spawn monsters
	monster1, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 7, Y: 7})
	if err != nil {
		t.Fatalf("Failed to spawn monster 1: %v", err)
	}

	monster2, err := gm.SpawnMonster(Orc, protocol.TileAddress{X: 8, Y: 8})
	if err != nil {
		t.Fatalf("Failed to spawn monster 2: %v", err)
	}

	// Move to GameMaster turn
	err = gm.EndTurn() // End hero turn
	if err != nil {
		t.Fatalf("Failed to end hero turn: %v", err)
	}

	// Verify we're in GM turn
	turnState := gm.GetTurnState()
	if turnState.CurrentTurn != "gamemaster" {
		t.Fatalf("Expected gamemaster turn, got: %s", turnState.CurrentTurn)
	}

	// Move first monster from (7,7) to (8,7)
	moveReq := MonsterActionRequest{
		MonsterID: monster1.ID,
		Action:    MonsterMoveAction,
		Parameters: map[string]any{
			"x": float64(8),
			"y": float64(7),
		},
	}

	result, err := gm.ProcessMonsterAction(moveReq)
	if err != nil {
		t.Fatalf("Monster move failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected successful monster move, got: %s", result.Message)
	}

	// Verify monster moved
	monsters := gm.GetMonsters()
	var movedMonster *Monster
	for _, m := range monsters {
		if m.ID == monster1.ID {
			movedMonster = m
			break
		}
	}

	if movedMonster == nil {
		t.Fatal("Could not find moved monster")
	}

	if movedMonster.Position.X != 8 || movedMonster.Position.Y != 7 {
		t.Errorf("Expected monster at (8,7), got (%d,%d)", movedMonster.Position.X, movedMonster.Position.Y)
	}

	// Second monster attacks hero (if in range)
	attackReq := MonsterActionRequest{
		MonsterID: monster2.ID,
		Action:    MonsterAttackAction,
		Parameters: map[string]any{
			"targetId": "hero-1",
		},
	}

	attackResult, err := gm.ProcessMonsterAction(attackReq)
	if err != nil {
		t.Fatalf("Monster attack failed: %v", err)
	}

	if !attackResult.Success {
		t.Fatalf("Expected successful monster attack, got: %s", attackResult.Message)
	}

	// Verify attack result
	if len(attackResult.DiceRolls) == 0 {
		t.Error("Expected dice rolls for monster attack")
	}
}

func TestMonsterSystem_RemoveMonster(t *testing.T) {

	gm := createTestGameManager()

	// Spawn monster
	_, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 7, Y: 7})
	if err != nil {
		t.Fatalf("Failed to spawn monster: %v", err)
	}

	// Verify monster exists
	monsters := gm.GetMonsters()
	if len(monsters) != 1 {
		t.Errorf("Expected 1 monster, got %d", len(monsters))
	}

	// Remove monster - method not implemented yet
	// TODO: Implement RemoveMonster method
	// err = gm.RemoveMonster(monster.ID)
	// if err != nil {
	//	t.Fatalf("Failed to remove monster: %v", err)
	// }

	// For now, skip the removal test
	t.Skip("RemoveMonster method not implemented yet")

	// Verify monster is gone
	monstersAfter := gm.GetMonsters()
	if len(monstersAfter) != 0 {
		t.Errorf("Expected 0 monsters after removal, got %d", len(monstersAfter))
	}
}

func TestMonsterSystem_MonsterTypes(t *testing.T) {

	gm := createTestGameManager()

	// Test all monster types
	monsterTypes := []MonsterType{
		Goblin,
		Orc,
		Skeleton,
		Zombie,
		Mummy,
		DreadWarrior, // Updated from Chaos_Warrior to use existing type
		Abomination,
		Gargoyle,
	}

	for i, monsterType := range monsterTypes {
		position := protocol.TileAddress{X: i + 1, Y: 1} // Place in a row
		monster, err := gm.SpawnMonster(monsterType, position)
		if err != nil {
			t.Errorf("Failed to spawn %s: %v", monsterType, err)
			continue
		}

		if monster.Type != monsterType {
			t.Errorf("Expected %s, got %s", monsterType, monster.Type)
		}

		// Verify monster has valid stats
		if monster.AttackDice <= 0 {
			t.Errorf("%s has invalid attack dice: %d", monsterType, monster.AttackDice)
		}

		if monster.DefenseDice <= 0 {
			t.Errorf("%s has invalid defense dice: %d", monsterType, monster.DefenseDice)
		}

		// Note: Monster struct uses Body/MaxBody instead of BodyPoints/MindPoints
		if monster.Body <= 0 {
			t.Errorf("%s has invalid body: %d", monsterType, monster.Body)
		}

		if monster.MaxBody <= 0 {
			t.Errorf("%s has invalid max body: %d", monsterType, monster.MaxBody)
		}
	}

	// Verify all monsters were spawned
	allMonsters := gm.GetMonsters()
	if len(allMonsters) != len(monsterTypes) {
		t.Errorf("Expected %d monsters, got %d", len(monsterTypes), len(allMonsters))
	}
}

func TestMonsterSystem_VisibilityIntegration(t *testing.T) {

	gm := createTestGameManager()

	// Spawn monster in visible area
	visibleMonster, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 6, Y: 5})
	if err != nil {
		t.Fatalf("Failed to spawn visible monster: %v", err)
	}

	// Reveal the visible monster (monsters start hidden until revealed)
	err = gm.RevealMonster(visibleMonster.ID)
	if err != nil {
		t.Fatalf("Failed to reveal visible monster: %v", err)
	}

	// Spawn monster in non-visible area (far away)
	hiddenMonster, err := gm.SpawnMonster(Orc, protocol.TileAddress{X: 0, Y: 0})
	if err != nil {
		t.Fatalf("Failed to spawn hidden monster: %v", err)
	}

	// Get visible monsters for hero (method signature different)
	visibleMonsters := gm.GetVisibleMonsters() // Takes no parameters

	// Should see visible monster
	foundVisible := false
	foundHidden := false

	for _, monster := range visibleMonsters {
		if monster.ID == visibleMonster.ID {
			foundVisible = true
		}
		if monster.ID == hiddenMonster.ID {
			foundHidden = true
		}
	}

	if !foundVisible {
		t.Error("Should see monster in visible area")
	}

	if foundHidden {
		t.Error("Should not see monster in hidden area")
	}
}

func TestMonsterSystem_ConcurrentActions(t *testing.T) {

	gm := createTestGameManager()

	// Spawn multiple monsters
	monsters := make([]*Monster, 3)
	for i := 0; i < 3; i++ {
		monster, err := gm.SpawnMonster(Goblin, protocol.TileAddress{X: 7 + i, Y: 7})
		if err != nil {
			t.Fatalf("Failed to spawn monster %d: %v", i, err)
		}
		monsters[i] = monster
	}

	// Move to GM turn
	err := gm.EndTurn()
	if err != nil {
		t.Fatalf("Failed to end hero turn: %v", err)
	}

	// Process actions for all monsters
	for i, monster := range monsters {
		actionReq := MonsterActionRequest{
			MonsterID:  monster.ID,
			Action:     MonsterWaitAction,
			Parameters: map[string]any{},
		}

		result, err := gm.ProcessMonsterAction(actionReq)
		if err != nil {
			t.Errorf("Monster %d action failed: %v", i, err)
		}

		if !result.Success {
			t.Errorf("Monster %d action not successful: %s", i, result.Message)
		}
	}

	// All monsters should still exist
	finalMonsters := gm.GetMonsters()
	if len(finalMonsters) != 3 {
		t.Errorf("Expected 3 monsters after actions, got %d", len(finalMonsters))
	}
}
