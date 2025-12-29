# Hero Turn Election and UI Updates
**Last Updated**: 2025-10-15 17:30

## Overview
Implement hero turn election system for multiplayer games and clean up toolbar UI by removing debug info and ensuring proper phase transitions after quest setup.

## Current Issues
1. **Bug**: After all heroes select starting positions, GM screen shows "waiting for game to start" but hero cannot roll for movement
2. **Missing**: Hero turn election panel for choosing which hero goes first (when multiple heroes)
3. **UI Clutter**: Map and Pack info still showing in both GM and player toolbars (debug info)
4. **Phase Transition**: Need to transition from quest_setup → hero_election (if multiple heroes) or quest_setup → hero_active (if single hero)

## Goals
- [ ] Implement hero turn election system in top right panel
- [ ] Auto-select single hero when only one player
- [ ] Remove Map/Pack info from toolbars
- [ ] Fix phase transition after starting position selection
- [ ] Ensure hero can roll for movement after quest setup completes

## Tasks

### 1. Backend: Phase Transition Logic
**File**: `cmd/server/dynamic_turn_order.go` or `cmd/server/handlers_turn_order.go`

- [ ] Add logic to detect when all heroes have confirmed starting positions
- [ ] Implement phase transition after quest setup:
  - Count total hero players (exclude GM)
  - If `heroCount == 1`: transition to `hero_active` phase with that hero
  - If `heroCount > 1`: transition to `hero_election` phase
- [ ] Create `hero_election` phase handler
  - Allow heroes to vote/select who goes first
  - Track votes or implement election logic
  - Transition to `hero_active` with elected hero
- [ ] Ensure turn state manager properly initializes hero turn after election
- [ ] Broadcast phase changes to all connected clients

**Files to Review**:
- `cmd/server/dynamic_turn_order.go` - Turn order management
- `cmd/server/handlers_turn_order.go` - Turn order handlers
- `cmd/server/game_manager.go` - Quest setup completion detection
- `internal/protocol/snapshot.go` - Phase state in snapshot

### 2. Frontend: Hero Election UI Component
**File**: `internal/web/views/components/heroTurnElectionPanel.templ` (already exists?)

- [ ] Review existing election panel component
- [ ] Update election panel for top-right placement
- [ ] Show list of all hero players
- [ ] Allow heroes to select/vote for who goes first
- [ ] Show voting status (who has voted, current leader)
- [ ] Auto-hide when only 1 hero player
- [ ] Send election vote via WebSocket intent

**JavaScript File**: `internal/web/static/js/ui/heroTurnControls.js`

- [ ] Add `showElectionPanel()` method (may already exist - verify)
- [ ] Implement election voting UI logic
- [ ] Handle `hero_election` phase in snapshot handler
- [ ] Auto-select single hero without showing panel
- [ ] Send election intent to backend

### 3. UI Cleanup: Remove Map/Pack Info
**File**: `internal/web/views/components/header.templ`

- [ ] Remove Map and Pack display from lines 110-119
- [ ] Optionally: Add to a debug panel accessible via keyboard shortcut
- [ ] Update both `Header()` and `HeaderWithSnapshot()` templates
- [ ] Regenerate templ files

### 4. UI Update: Player Toolbar Enhancement
**Already Complete** (based on previous work):
- [x] Body Points display
- [x] Mind Points display
- [x] Gold display
- [x] Character name
- [x] Character class

**Remaining**:
- [ ] Verify backend populates `mindPoints` in EntityLite
- [ ] Verify backend sends gold/treasure count in snapshot
- [ ] Test that all values update during gameplay

### 5. Top Right Panel: Election UI Integration
**File**: `internal/web/views/index.templ` or `internal/web/views/gm.templ`

- [ ] Ensure top-right panel can show election UI
- [ ] Position election panel in top-right area
- [ ] Style election panel to match toolbar aesthetic
- [ ] Show election panel during `hero_election` phase
- [ ] Hide election panel during other phases

### 6. Testing & Validation

**Single Hero Scenario**:
- [ ] Start game with 1 GM + 1 Hero
- [ ] Complete quest setup (select starting position)
- [ ] Verify: Immediately transitions to hero's turn (no election)
- [ ] Verify: Hero can roll for movement
- [ ] Verify: Movement actions work correctly

**Multiple Hero Scenario**:
- [ ] Start game with 1 GM + 2+ Heroes
- [ ] Complete quest setup (all select starting positions)
- [ ] Verify: Transitions to `hero_election` phase
- [ ] Verify: Election panel shows in top-right
- [ ] Verify: All heroes can vote/select
- [ ] Verify: After election, transitions to elected hero's turn
- [ ] Verify: Elected hero can roll for movement

**UI Verification**:
- [ ] Map/Pack info removed from both GM and player views
- [ ] Toolbar shows correct character info (body, mind, gold)
- [ ] Turn info updates correctly through phases
- [ ] Election panel only shows when appropriate

## Technical Notes

### Phase Flow
```
quest_setup (heroes select starting positions)
    ↓
[All heroes confirmed?]
    ↓
    ├─→ [1 hero] → hero_active (auto-select)
    └─→ [2+ heroes] → hero_election
                         ↓
                    [Vote/Select]
                         ↓
                    hero_active (elected hero)
```

### WebSocket Intent for Election
```go
// Example intent structure
{
  "type": "elect_hero",
  "data": {
    "votedFor": "hero-id" or "player-id"
  }
}
```

### Phase State Management
- Current phases: `quest_setup`, `hero_election`, `hero_active`, `gm_phase`
- Need to ensure phase state is persisted in game state
- Broadcast phase changes via patch system

## Files to Modify

### Backend (Go)
- [ ] `cmd/server/dynamic_turn_order.go`
- [ ] `cmd/server/handlers_turn_order.go`
- [ ] `cmd/server/game_manager.go`
- [ ] `internal/protocol/snapshot.go` (verify phase field)
- [ ] `internal/protocol/intent.go` (add election intent)

### Frontend (Templates)
- [ ] `internal/web/views/components/header.templ`
- [ ] `internal/web/views/components/heroTurnElectionPanel.templ`
- [ ] `internal/web/views/index.templ` (hero view)

### Frontend (JavaScript)
- [ ] `internal/web/static/js/ui/heroTurnControls.js`
- [ ] `internal/web/static/js/patchSystem.js` (if needed for phase handling)

### Frontend (CSS)
- [ ] `internal/web/static/styles/index.css` (if new styles needed)

## Success Criteria
1. ✅ Single hero games: No election panel, immediate turn transition
2. ✅ Multiple hero games: Election panel appears, heroes can vote
3. ✅ After election: Correct hero gets turn, can perform actions
4. ✅ UI: Map/Pack removed from toolbars
5. ✅ UI: Character stats showing correctly (body, mind, gold)
6. ✅ No "waiting for game to start" bug after quest setup

## Next Steps After Completion
1. Test with 3-4 hero games to ensure scalability
2. Add election timeout/default selection logic
3. Consider adding "ready" button for heroes instead of automatic election
4. Implement turn order rotation for subsequent turns
