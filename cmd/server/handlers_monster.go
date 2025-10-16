package main

import (
	"github.com/Ko-stant/dungeon-campaign-engine/internal/protocol"
	"github.com/Ko-stant/dungeon-campaign-engine/internal/ws"
)

// handleRequestSelectMonster handles GM selecting a monster to control
func handleRequestSelectMonster(req protocol.RequestSelectMonster, gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	turnStateManager := gameManager.GetTurnStateManager()
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()

	// Only allow during GM phase
	if dynamicTurnOrder.GetCurrentPhase() != GMPhase {
		gameManager.logger.Printf("Cannot select monster outside GM phase")
		return
	}

	// Select the monster (empty string to deselect)
	if err := turnStateManager.SelectMonster(req.MonsterID); err != nil {
		gameManager.logger.Printf("Failed to select monster %s: %v", req.MonsterID, err)
		return
	}

	if req.MonsterID != "" {
		gameManager.logger.Printf("GM selected monster: %s", req.MonsterID)
	} else {
		gameManager.logger.Printf("GM deselected monster")
	}

	// Broadcast monster selection change
	patch := protocol.MonsterSelectionChanged{
		SelectedMonsterID: req.MonsterID,
	}
	broadcastEvent(hub, sequence, "MonsterSelectionChanged", patch)
}

// handleRequestMoveMonster handles GM moving a monster
func handleRequestMoveMonster(req protocol.RequestMoveMonster, gameManager *GameManager, hub *ws.Hub, sequence *uint64, state *GameState, furnitureSystem *FurnitureSystem, monsterSystem *MonsterSystem) {
	turnStateManager := gameManager.GetTurnStateManager()
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()

	// Only allow during GM phase
	if dynamicTurnOrder.GetCurrentPhase() != GMPhase {
		gameManager.logger.Printf("Cannot move monster outside GM phase")
		return
	}

	// Get monster state
	monsterState := turnStateManager.GetMonsterTurnState(req.MonsterID)
	if monsterState == nil {
		gameManager.logger.Printf("Monster %s has no turn state", req.MonsterID)
		return
	}

	// Get monster from monster system
	monster, err := monsterSystem.GetMonsterByID(req.MonsterID)
	if err != nil {
		gameManager.logger.Printf("Monster %s not found: %v", req.MonsterID, err)
		return
	}

	// Validate movement
	currentX, currentY := monster.Position.X, monster.Position.Y
	targetX, targetY := req.ToX, req.ToY

	// Calculate distance (Manhattan distance for simplicity)
	distance := absInt(targetX-currentX) + absInt(targetY-currentY)

	// Check if monster has enough movement remaining
	if distance > monsterState.MovementRemaining {
		gameManager.logger.Printf("Monster %s does not have enough movement (needs %d, has %d)",
			req.MonsterID, distance, monsterState.MovementRemaining)
		return
	}

	// Validate target position is valid
	if targetX < 0 || targetY < 0 || targetX >= state.Segment.Width || targetY >= state.Segment.Height {
		gameManager.logger.Printf("Target position (%d,%d) out of bounds", targetX, targetY)
		return
	}

	// Check for blocking walls, doors, furniture
	// TODO: Implement proper pathfinding and validation
	// For now, only allow adjacent moves

	if distance != 1 {
		gameManager.logger.Printf("Only adjacent moves currently supported")
		return
	}

	// Check edge blocking
	dx := targetX - currentX
	dy := targetY - currentY
	edge := edgeForStep(currentX, currentY, dx, dy)

	state.Lock.Lock()
	blocked := state.BlockedWalls[edge]
	doorID, hasDoor := state.DoorByEdge[edge]
	var doorClosed bool
	if hasDoor {
		door := state.Doors[doorID]
		doorClosed = door != nil && door.State != "open"
	}
	state.Lock.Unlock()

	if blocked || doorClosed {
		gameManager.logger.Printf("Movement blocked by wall or closed door")
		return
	}

	// Check furniture blocking
	if furnitureSystem.BlocksMovement(targetX, targetY) {
		gameManager.logger.Printf("Movement blocked by furniture at (%d,%d)", targetX, targetY)
		return
	}

	// Check if target is occupied by another monster
	if monsterSystem.IsMonsterAt(targetX, targetY) {
		gameManager.logger.Printf("Target position (%d,%d) occupied by another monster", targetX, targetY)
		return
	}

	// Perform movement
	newPos := protocol.TileAddress{
		SegmentID: monster.Position.SegmentID,
		X:         targetX,
		Y:         targetY,
	}

	// Update monster position in monster system
	monster.Position = newPos

	// Record movement in turn state
	if err := turnStateManager.RecordMonsterMovement(req.MonsterID, newPos); err != nil {
		gameManager.logger.Printf("Failed to record monster movement: %v", err)
		return
	}

	// Also record in dynamic turn order manager
	if err := dynamicTurnOrder.SetMonsterMoved(req.MonsterID, true); err != nil {
		gameManager.logger.Printf("Failed to record monster moved state in turn order: %v", err)
	}

	gameManager.logger.Printf("Monster %s moved from (%d,%d) to (%d,%d)",
		req.MonsterID, currentX, currentY, targetX, targetY)

	// Broadcast entity update
	broadcastEvent(hub, sequence, "EntityUpdated", protocol.EntityUpdated{
		ID:   monster.ID,
		Tile: newPos,
	})

	// Broadcast updated monster turn state
	broadcastMonsterTurnState(monsterState, hub, sequence)
}

// handleRequestMonsterAttack handles GM initiating a monster attack
func handleRequestMonsterAttack(req protocol.RequestMonsterAttack, gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	turnStateManager := gameManager.GetTurnStateManager()
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()

	// Only allow during GM phase
	if dynamicTurnOrder.GetCurrentPhase() != GMPhase {
		gameManager.logger.Printf("Cannot attack outside GM phase")
		return
	}

	// Get monster state
	monsterState := turnStateManager.GetMonsterTurnState(req.MonsterID)
	if monsterState == nil {
		gameManager.logger.Printf("Monster %s has no turn state", req.MonsterID)
		return
	}

	// Check if monster can take action
	if canAct, reason := monsterState.CanTakeAction(); !canAct {
		gameManager.logger.Printf("Monster %s cannot take action: %s", req.MonsterID, reason)
		return
	}

	// TODO: Implement actual combat resolution
	// For now, just record the action

	action := MonsterActionRecord{
		ActionType: "attack",
		TargetID:   req.TargetID,
		Success:    true,
		Details:    map[string]interface{}{"type": "melee_attack"},
	}

	if err := turnStateManager.RecordMonsterAction(req.MonsterID, action); err != nil {
		gameManager.logger.Printf("Failed to record monster action: %v", err)
		return
	}

	// Also record in dynamic turn order manager
	if err := dynamicTurnOrder.SetMonsterActionTaken(req.MonsterID, true); err != nil {
		gameManager.logger.Printf("Failed to record monster action taken in turn order: %v", err)
	}

	gameManager.logger.Printf("Monster %s attacked %s", req.MonsterID, req.TargetID)

	// Broadcast updated monster turn state
	broadcastMonsterTurnState(monsterState, hub, sequence)
}

// handleRequestUseMonsterAbility handles GM using a monster special ability
func handleRequestUseMonsterAbility(req protocol.RequestUseMonsterAbility, gameManager *GameManager, hub *ws.Hub, sequence *uint64) {
	turnStateManager := gameManager.GetTurnStateManager()
	dynamicTurnOrder := gameManager.GetDynamicTurnOrder()

	// Only allow during GM phase
	if dynamicTurnOrder.GetCurrentPhase() != GMPhase {
		gameManager.logger.Printf("Cannot use ability outside GM phase")
		return
	}

	// Get monster state
	monsterState := turnStateManager.GetMonsterTurnState(req.MonsterID)
	if monsterState == nil {
		gameManager.logger.Printf("Monster %s has no turn state", req.MonsterID)
		return
	}

	// Check if monster can use this ability
	if canUse, reason := monsterState.CanUseAbility(req.AbilityID); !canUse {
		gameManager.logger.Printf("Monster %s cannot use ability %s: %s", req.MonsterID, req.AbilityID, reason)
		return
	}

	// Prepare target position
	var targetPos *protocol.TileAddress
	if req.TargetX != nil && req.TargetY != nil {
		targetPos = &protocol.TileAddress{
			SegmentID: "", // TODO: Get from context
			X:         *req.TargetX,
			Y:         *req.TargetY,
		}
	}

	// Use the ability
	if err := turnStateManager.UseMonsterAbility(req.MonsterID, req.AbilityID, req.TargetID, targetPos, true, nil); err != nil {
		gameManager.logger.Printf("Failed to use monster ability: %v", err)
		return
	}

	gameManager.logger.Printf("Monster %s used ability %s", req.MonsterID, req.AbilityID)

	// Broadcast updated monster turn state
	broadcastMonsterTurnState(monsterState, hub, sequence)
}

// broadcastMonsterTurnState broadcasts a monster turn state update
func broadcastMonsterTurnState(state *MonsterTurnState, hub *ws.Hub, sequence *uint64) {
	// Convert special abilities to protocol format
	abilities := make([]protocol.MonsterAbilityLite, 0, len(state.SpecialAbilities))
	for _, ability := range state.SpecialAbilities {
		usesLeft := state.QuestAbilityUsageLeft[ability.ID]
		abilities = append(abilities, protocol.MonsterAbilityLite{
			ID:             ability.ID,
			Name:           ability.Name,
			Type:           ability.Type,
			UsesPerTurn:    ability.UsesPerTurn,
			UsesPerQuest:   ability.UsesPerQuest,
			UsesLeftQuest:  usesLeft,
			RequiresAction: ability.RequiresAction,
			Range:          ability.Range,
			Description:    ability.Description,
			EffectDetails:  ability.EffectDetails,
		})
	}

	actionType := ""
	if state.Action != nil {
		actionType = state.Action.ActionType
	}

	patch := protocol.MonsterTurnStateChanged{
		MonsterID:            state.MonsterID,
		EntityID:             state.EntityID,
		TurnNumber:           state.TurnNumber,
		CurrentPosition:      state.CurrentPosition,
		FixedMovement:        state.FixedMovement,
		MovementRemaining:    state.MovementRemaining,
		MovementUsed:         state.MovementUsed,
		HasMoved:             state.HasMoved,
		ActionTaken:          state.ActionTaken,
		ActionType:           actionType,
		AttackDice:           state.AttackDice,
		DefenseDice:          state.DefenseDice,
		BodyPoints:           state.BodyPoints,
		CurrentBody:          state.CurrentBody,
		SpecialAbilities:     abilities,
		AbilityUsageThisTurn: state.SpecialAbilitiesUsed,
		ActiveEffectsCount:   len(state.ActiveEffects),
	}

	broadcastEvent(hub, sequence, "MonsterTurnStateChanged", patch)
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
