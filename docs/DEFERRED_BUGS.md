# Deferred Bug Fixes

**Last Updated**: 2025-10-16 13:36

This document tracks bugs and improvements that have been identified but are not immediately blocking core gameplay.


## Go server warnings
- monster_turn_state.go: Line 282: unnecessary nil check around range
- monster_turn_state.go: Line 284: Replace m[k]=v loop with maps.Copy
- lobby_server.go: Line 22: field mutex is unused
- handlers_turn_order.go: Line 253: this result of append is never used, except maybe in other appends


## Combat & Actions

### 1. Hero can move after taking an action ✅ COMPLETED
**Priority**: High
**Description**: Heroes were able to move after they'd already taken their main action AND moved, which violates HeroQuest rules. Movement should only be allowed before or after the main action, but not both.

**Status**: COMPLETED - Fixed validation and flag management logic

**Root Cause**: The original bug was that `HasMoved` was set immediately when movement started (even on first tile), causing the validation to trigger too early. The fix required changing WHEN `HasMoved` gets set.

**HeroQuest Rule Clarification**:
- You can move BEFORE your action (then no movement after)
- OR you can take your action FIRST, then move your ENTIRE movement allowance after
- You CANNOT split: Move → Action → Move

**Final Implementation**:
- `cmd/server/turn_system.go:421-425` - Validation check:
  - Simple check: if both `HasMoved` and `ActionTaken` are true → block movement
  - No exceptions for "continuing" movement (action interrupts the movement phase)
- `cmd/server/turn_system.go:442-446` - Movement start tracking:
  - Removed immediate setting of `HasMoved=true`
  - Only sets `MovementStarted=true` to track that movement is in progress
- `cmd/server/turn_system.go:553-557` - Action consumption logic:
  - When action is taken, if `MovementStarted=true` → sets `HasMoved=true`
  - This locks the movement phase, preventing further movement after action
- This fix properly prevents: Move → Act → Move
- Correctly allows: Act → Move (all tiles), Move (all tiles) → Act

---

## Monster System

### 2. Monster detail info shows incorrect stats ✅ COMPLETED
**Priority**: Medium
**Description**: When clicking on a monster to view its details, the UI shows:
- 0/0 body points (should show current/max)
- 0 mind points (should show actual value)

**Status**: COMPLETED - Fixed JSON serialization tag causing 0 values

**Root Cause**: The Monster struct had an incorrect JSON tag `MaxBody` (capital M) instead of `maxBody` (lowercase m), causing JSON serialization to fail and return 0 values to the client.

**Implementation**:
- `cmd/server/monster_system.go:44` - Fixed `json:"MaxBody"` to `json:"maxBody"` in Monster struct
- `cmd/server/monster_system.go:76` - Fixed same issue in MonsterTemplate struct
- The entityModal.js was already correctly reading the fields

### 3. Monsters show "0d6 movement" instead of fixed movement ✅ COMPLETED
**Priority**: Low
**Description**: Monster cards display "0d6" for movement instead of their fixed movement value (e.g., "6 squares")

**Status**: COMPLETED - Already implemented correctly

**Implementation**:
- `internal/web/static/js/ui/entityModal.js:108-112` - Reads `fixedMovement` from `monsterTurnState`
- `internal/web/static/js/ui/entityModal.js:139` - Displays as "X squares" or "Unknown" if not available
- The fixed movement is correctly populated from monster data in turn state

---

## UI/UX Improvements

### 4. Display combat dice results as icons/symbols ✅ COMPLETED
**Priority**: Medium
**Description**: Attack and defense dice rolls currently show numerical values (1-6). Should show symbolic representations of the dice faces:
- Attack dice: Skull (4-6) or Miss (1-3)
- Defense dice: Black Shield (6), White Shield (4-5), or Miss (1-3)

**Status**: COMPLETED - Dice are now displayed as emoji icons (💀 for skull, 🛡️ for shield, ⚪ for blank)

**Implementation**:
- `internal/web/static/js/ui/detailPane.js:120-131` - formatDiceRolls() already uses emoji icons

### 5. Remove separated "skulls" and "shields" from attack results ✅ COMPLETED
**Priority**: Low
**Description**: Attack result details currently show separate counts like "2 skulls, 1 shield". Once dice are displayed as symbols (issue #4), this redundant text should be removed.

**Status**: COMPLETED - Redundant skull/shield counts removed from attack results display

**Implementation**:
- `internal/web/static/js/ui/detailPane.js:87-100` - Removed "Skulls: X" and "Shields: X" text, keeping only emoji dice display

---

## Toolbar & Display Issues

### 6. GM toolbar always shows "Quest Setup" phase ✅ COMPLETED (Already Implemented)
**Priority**: Medium
**Description**: The Game Master's toolbar continues to display "Quest Setup" even after the game has progressed to hero turns.

**Status**: COMPLETED - Already correctly implemented

**Implementation**:
- `internal/web/static/js/ui/gmControls.js:635-709` - `updateTurnPhase()` method correctly reads `snapshot.turnPhase` and updates display
- `internal/web/static/js/ui/gmControls.js:649-670` - Switch statement handles all phases: quest_setup, hero_election, hero_turn/hero_active, gm_phase
- `internal/web/static/js/patchSystem.js:558,694` - GM controls are updated when patches arrive
- The system correctly shows different phases with appropriate colors and text

### 7. Player toolbar should show hero's chosen name ✅ COMPLETED
**Priority**: Medium
**Description**: Player toolbars currently show only the hero's class (e.g., "Wizard") but not the player's chosen character name.

**Status**: COMPLETED - Both backend and frontend implementation complete

**Backend Implementation**:
- `internal/protocol/snapshot.go:126` - Added PlayerNames field to Snapshot struct
- `cmd/server/main_lobby.go:417-423` - Populates PlayerNames for hero view snapshots
- `cmd/server/main_lobby.go:639-645` - Populates PlayerNames for GM view snapshots

**Frontend Implementation**:
- `internal/web/static/js/ui/heroTurnControls.js:595-604` - Displays player name alongside class name
- Format: "Wizard - Gandalf" (class - player name)
- Falls back to class name only if player name not available

### 8. Browser refresh loses game state ✅ COMPLETED
**Priority**: High
**Description**: When the browser is refreshed, multiple UI elements lose their state:
- Player icon disappears from gameboard
- Hero status panel shows "no hero selected"
- Toolbar shows "Active: unknown"
- Toolbar stats (body, mind, gold) are all zeroed out
- GM sees the starting position modal during quest setup
- GM gets redirected to hero view after refresh

**Status**: COMPLETED - Fixed lobby state dependency and added persistent GM tracking

**Root Causes**:
1. **Entities disappearing**: When building the entities list for the snapshot, the code iterated over `lobbyServer.lobby.players` which becomes unreliable after game starts (players disconnect/reconnect from lobby WebSocket)
2. **GM redirect issue**: GM role checking used `lobbyServer.lobby.GetPlayer()` which fails after game starts due to lobby state instability
3. **Quest setup modal**: GM was seeing the quest setup UI because there was no role check in `questSetupControls.js`

**Implementation**:

**Hero State Fix** (`cmd/server/main_lobby.go`):
- Lines 186-219 (Hero page): Changed from iterating over `lobbyServer.lobby.players` to using `gameManager.turnManager.GetHeroPlayers()`
  - TurnManager is the source of truth for active players during gameplay
  - Added defensive check: if entity not found in game state, use zero position (0,0)
  - Still include entity in snapshot so UI can display it during quest setup
- Lines 249-280 (GM page): Applied same fix for GM page snapshot generation

**GM Role Tracking** (`cmd/server/main_lobby.go`):
- Line 120: Added `var gameMasterPlayerID string` to persistently track GM player ID
- Line 125: Set `gameMasterPlayerID = gameMasterID` when game starts
- Line 279: Changed hero page GM redirect from lobby lookup to direct comparison: `if viewerPlayerID == gameMasterPlayerID`
- Line 236: Changed GM page access control from lobby lookup to direct comparison: `if playerID != gameMasterPlayerID`

**Quest Setup Modal Fix** (`internal/web/static/js/ui/questSetupControls.js`):
- Line 74: Added role check: `if (snapshot.turnPhase === 'quest_setup' && snapshot.viewerRole !== 'gm')`
- GM no longer sees quest setup UI elements (status panel, position selection modal)

### 9. Turn counter never increments ✅ COMPLETED
**Priority**: Medium
**Description**: The turn counter in the UI remained at 0 and never incremented as the game progressed through turns.

**Status**: COMPLETED - UI now reads from correct turn counter field

**Root Cause**: The game uses two different turn tracking systems:
1. **TurnManager** (old system) - Has `TurnNumber` field that only increments in `advanceToNextHero()` at `turn_system.go:648`
2. **DynamicTurnOrderManager** (new system) - Has `cycleNumber` field that increments when GM phase completes at `dynamic_turn_order.go:374`

When using the dynamic turn order system, `TurnManager.TurnNumber` never gets incremented, but `cycleNumber` does. The UI was reading the wrong field from the snapshot.

**Implementation**:
- `internal/web/static/js/ui/turnCounter.js:22-23` - Changed from reading `snapshot.turn` to `snapshot.cycleNumber`
- The snapshot protocol (`internal/protocol/snapshot.go`) contains both fields:
  - Line 111: `Turn int` (from old TurnManager)
  - Line 139: `CycleNumber int` (from DynamicTurnOrderManager)
- Turn counter now correctly increments each time all heroes act and GM phase completes

---

## Implementation Notes

- These issues should be tackled after core multiplayer functionality is stable
- Issues #4 and #5 are related and can be implemented together
- Issue #1 is the highest priority as it affects game balance
- Issue #8 (browser refresh) is high priority as it affects basic usability
