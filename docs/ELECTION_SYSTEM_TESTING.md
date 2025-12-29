# Hero Turn Election System - Testing Guide

## Overview

This document provides step-by-step testing scenarios for the automatic hero turn election system. The system allows players to self-elect when to take their turn, with automatic selection for single-hero scenarios and last-remaining-hero cases.

## Prerequisites

- Server running at `http://localhost:8080/lobby`
- Two browser windows/tabs for testing multiple player scenarios
- One window for GM, one (or more) for hero players

---

## Test Scenario 1: Single Hero - Auto Start

**Objective**: Verify that a single hero automatically starts their turn after quest setup without needing election.

### Setup Steps

1. **Open GM Window**
   - Navigate to `http://localhost:8080/lobby`
   - Enter name: "Dungeon Daddy"
   - Select role: Game Master
   - Click "Ready"

2. **Open Hero Window**
   - Navigate to `http://localhost:8080/lobby` in a new tab/window
   - Enter name: "Solo Hero"
   - Select role: Hero (choose any class, e.g., Wizard)
   - Click "Ready"

3. **Start Game**
   - GM clicks "Start Game"
   - Quest loads successfully

### Test Steps

1. **Hero Selects Starting Position**
   - Hero window: Click on the stairwell tile (bottom-left area of board)
   - Modal should appear: "Choose Your Starting Position"
   - Click "Confirm Position"

2. **Verify Automatic Turn Start**
   - ✅ **Expected**: Hero turn should start automatically without election phase
   - ✅ **Expected**: Phase indicator shows "Hero Turn" or similar active state
   - ✅ **Expected**: "Roll for Movement" button is enabled and visible
   - ✅ **Expected**: No election panel appears (no "I'll Go Next!" button)

3. **Verify Hero Can Roll Movement**
   - Hero window: Click "Roll for Movement"
   - ✅ **Expected**: Dice roll animation/result appears
   - ✅ **Expected**: Movement counter shows rolled value (e.g., "Movement: 5/9")
   - ✅ **Expected**: Hero can move on the board

4. **GM View Verification**
   - GM window: Check right panel "Turn Phase Status"
   - ✅ **Expected**: Shows "Active Hero: Wizard" (or chosen class)
   - ✅ **Expected**: Shows movement state (rolled/not rolled)
   - ✅ **Expected**: No election controls visible

### Success Criteria

- [ ] Single hero skips election phase entirely
- [ ] Hero turn starts immediately after position confirmation
- [ ] Hero can roll for movement without any additional clicks
- [ ] No "I'll Go Next!" or election UI appears
- [ ] Phase transitions correctly from quest_setup → hero_active

---

## Test Scenario 2: Multiple Heroes - Election Flow

**Objective**: Verify that multiple heroes use the election system correctly, with automatic confirmation when clicking "I'll Go Next!"

### Setup Steps

1. **Open GM Window**
   - Navigate to `http://localhost:8080/lobby`
   - Enter name: "Dungeon Daddy"
   - Select role: Game Master
   - Click "Ready"

2. **Open Hero Window 1**
   - Navigate to `http://localhost:8080/lobby` in a new tab/window
   - Enter name: "Wizard Hero"
   - Select role: Hero → Wizard
   - Click "Ready"

3. **Open Hero Window 2**
   - Navigate to `http://localhost:8080/lobby` in a new tab/window
   - Enter name: "Barbarian Hero"
   - Select role: Hero → Barbarian
   - Click "Ready"

4. **Start Game**
   - GM clicks "Start Game"
   - Quest loads successfully

### Test Steps - Phase 1: Quest Setup

1. **Hero 1 Selects Starting Position**
   - Wizard window: Click stairwell, confirm position
   - ✅ **Expected**: Wizard appears on board at chosen position
   - ✅ **Expected**: Phase remains in "Quest Setup"

2. **Hero 2 Selects Starting Position**
   - Barbarian window: Click adjacent to stairwell, confirm position
   - ✅ **Expected**: Barbarian appears on board at chosen position
   - ✅ **Expected**: Phase transitions to "Hero Election"

### Test Steps - Phase 2: Election Phase

3. **Verify Election UI Appears**
   - Both hero windows should show:
     - ✅ **Expected**: "I'll Go Next!" button is visible and enabled
     - ✅ **Expected**: Panel shows "Choose next hero to act"
     - ✅ **Expected**: "Roll for Movement" is disabled/hidden
     - ✅ **Expected**: No cancel button visible yet

4. **GM View During Election**
   - GM window: Check right panel
   - ✅ **Expected**: Shows "Turn Phase Status"
   - ✅ **Expected**: Shows "Eligible Heroes" list with both heroes
   - ✅ **Expected**: Phase description: "Heroes are electing next player"

### Test Steps - Phase 3: First Hero Elects

5. **Wizard Elects Self**
   - Wizard window: Click "I'll Go Next!"
   - ✅ **Expected**: Button immediately changes to "Cancel My Election"
   - ✅ **Expected**: Phase transitions to "Hero Turn" automatically (NO GM CONFIRMATION)
   - ✅ **Expected**: "Roll for Movement" button appears and is enabled
   - ✅ **Expected**: Cancel button is enabled (can cancel since no actions taken)

6. **Barbarian View After Election**
   - Barbarian window:
   - ✅ **Expected**: "I'll Go Next!" button is disabled/hidden
   - ✅ **Expected**: Shows message indicating Wizard is active
   - ✅ **Expected**: Cannot elect self while another hero is active

7. **GM View After Election**
   - GM window:
   - ✅ **Expected**: Shows "Active Hero: Wizard"
   - ✅ **Expected**: Shows hero turn state (movement not rolled yet)
   - ✅ **Expected**: No confirm button visible (election was automatic)

### Test Steps - Phase 4: Cancel Before Actions

8. **Wizard Cancels Election (Before Rolling)**
   - Wizard window: Click "Cancel My Election"
   - ✅ **Expected**: Returns to election phase
   - ✅ **Expected**: "I'll Go Next!" button reappears for both heroes
   - ✅ **Expected**: "Roll for Movement" button disappears
   - ✅ **Expected**: Wizard turn state is cleared

9. **Verify Both Heroes Can Elect Again**
   - Both hero windows:
   - ✅ **Expected**: "I'll Go Next!" button is enabled for both
   - ✅ **Expected**: No hero is shown as active

### Test Steps - Phase 5: Election with Actions Taken

10. **Barbarian Elects Self**
    - Barbarian window: Click "I'll Go Next!"
    - ✅ **Expected**: Immediately becomes active hero
    - ✅ **Expected**: "Roll for Movement" appears
    - ✅ **Expected**: Cancel button shows "Cancel My Election" (enabled)

11. **Barbarian Rolls Movement**
    - Barbarian window: Click "Roll for Movement"
    - ✅ **Expected**: Dice roll succeeds
    - ✅ **Expected**: Movement value appears
    - ✅ **Expected**: Cancel button now shows "Cannot Cancel (Actions Taken)" and is disabled

12. **Verify Cancel is Blocked After Rolling**
    - Barbarian window: Try to click cancel button
    - ✅ **Expected**: Button is disabled with visual feedback
    - ✅ **Expected**: Tooltip or text explains: "Cannot cancel after taking actions"

13. **Barbarian Moves and Completes Turn**
    - Barbarian window: Move hero to a new tile
    - Barbarian window: Click "End Turn" (or complete action)
    - ✅ **Expected**: Phase returns to "Hero Election"
    - ✅ **Expected**: Barbarian's "I'll Go Next!" button is disabled/hidden
    - ✅ **Expected**: Wizard's "I'll Go Next!" button is enabled

### Test Steps - Phase 6: Auto-Election of Last Hero

14. **Verify Auto-Election Trigger**
    - State: Only Wizard hasn't acted this cycle, Barbarian has completed turn
    - ✅ **Expected**: Wizard is AUTOMATICALLY elected (no button click needed)
    - ✅ **Expected**: Wizard's turn starts immediately
    - ✅ **Expected**: "Roll for Movement" appears automatically
    - ✅ **Expected**: No "Cancel" button (auto-elected as last hero)

15. **GM View During Auto-Election**
    - GM window:
    - ✅ **Expected**: Shows "Active Hero: Wizard" immediately
    - ✅ **Expected**: Console log shows: "Auto-electing last remaining hero: {playerID}"
    - ✅ **Expected**: Shows hero turn state initialized

16. **Complete Cycle**
    - Wizard window: Roll movement, take action, end turn
    - ✅ **Expected**: New cycle begins
    - ✅ **Expected**: Both heroes' "I'll Go Next!" buttons become available again
    - ✅ **Expected**: "Heroes Acted" list is cleared for new cycle

### Success Criteria

- [ ] Election phase appears after all heroes select positions
- [ ] "I'll Go Next!" immediately starts hero turn (no GM confirmation)
- [ ] Cancel works ONLY if no movement rolled and no action taken
- [ ] Cancel button is disabled with explanation after actions taken
- [ ] Last remaining hero is automatically elected
- [ ] Auto-elected hero cannot cancel
- [ ] All heroes get fresh election opportunity each cycle

---

## Test Scenario 3: Three Heroes - Last Hero Auto-Election

**Objective**: Verify auto-election works correctly with more than 2 heroes.

### Setup Steps

1. Start game with GM + 3 heroes (Wizard, Barbarian, Dwarf)
2. All heroes select starting positions
3. Election phase begins

### Test Steps

1. **First Hero Volunteers**
   - Wizard clicks "I'll Go Next!"
   - ✅ **Expected**: Wizard's turn starts immediately
   - Wizard completes turn
   - ✅ **Expected**: Returns to election phase
   - ✅ **Expected**: Wizard's button is disabled (already acted)

2. **Second Hero Volunteers**
   - Barbarian clicks "I'll Go Next!"
   - ✅ **Expected**: Barbarian's turn starts immediately
   - Barbarian completes turn
   - ✅ **Expected**: Returns to election phase
   - ✅ **Expected**: Only Dwarf's button remains enabled

3. **Third Hero Auto-Elected**
   - ✅ **Expected**: Dwarf is AUTOMATICALLY elected without clicking
   - ✅ **Expected**: Dwarf's turn starts immediately
   - ✅ **Expected**: Console log shows auto-election
   - ✅ **Expected**: Dwarf cannot cancel (auto-elected)

### Success Criteria

- [ ] Auto-election triggers when only 1 hero remains in cycle
- [ ] Works correctly with 3+ heroes
- [ ] Each hero acts exactly once per cycle
- [ ] New cycle resets all eligibility

---

## Test Scenario 4: Cancel Edge Cases

**Objective**: Verify cancel validation works correctly in various situations.

### Test Cases

#### 4A: Cancel After Movement (No Action)

1. Hero elects self and becomes active
2. Hero rolls movement (dice appear)
3. Hero clicks cancel button
4. ✅ **Expected**: Cancel is BLOCKED (dice rolled counts as action taken)
5. ✅ **Expected**: Button shows "Cannot Cancel (Actions Taken)"

#### 4B: Cancel After Moving (No Action Yet)

1. Hero elects self and becomes active
2. Hero rolls movement
3. Hero moves to a new tile
4. Hero clicks cancel button
5. ✅ **Expected**: Cancel is BLOCKED (movement used)

#### 4C: Cancel Immediately After Election

1. Hero elects self and becomes active
2. Hero immediately clicks cancel (before rolling)
3. ✅ **Expected**: Cancel SUCCEEDS
4. ✅ **Expected**: Returns to election phase
5. ✅ **Expected**: Turn state is cleared

#### 4D: Cancel During Auto-Election

1. Setup scenario where only 1 hero remains
2. Hero is auto-elected automatically
3. Check cancel button state
4. ✅ **Expected**: Cancel button is hidden or permanently disabled
5. ✅ **Expected**: Cannot cancel when auto-elected

### Success Criteria

- [ ] Cancel blocked after MovementDice.Rolled = true
- [ ] Cancel blocked after ActionTaken = true
- [ ] Cancel succeeds before any actions
- [ ] Auto-elected heroes cannot cancel

---

## Console Log Validation

### Key Log Messages to Watch For

During testing, monitor the server console for these log messages:

**Quest Setup Complete (Single Hero)**:
```
Only 1 hero in game - starting their turn directly
Auto-started turn for single hero: {playerID}
```

**Quest Setup Complete (Multiple Heroes)**:
```
Multiple heroes (2) - entering hero election phase
Phase transition: quest_setup → hero_election
```

**Hero Self-Election**:
```
Player {playerID} elected themselves to go next
Auto-started turn for player {playerID} after election
```

**Cancel Election**:
```
Player {playerID} cancelled their election
After cancel: X heroes remaining who haven't acted
```

**Cancel Blocked**:
```
Cannot cancel election: player {playerID} has already taken actions (rolled: true, action: false)
```

**Auto-Election of Last Hero**:
```
After hero turn complete: 1 heroes remaining who haven't acted
Auto-electing last remaining hero: {playerID}
Auto-started turn for last remaining hero {playerID}
```

---

## UI State Verification Checklist

### Hero Election Panel (Hero View)

**During Election Phase**:
- [ ] "I'll Go Next!" button visible
- [ ] Button enabled if hero hasn't acted
- [ ] Button disabled if hero has acted
- [ ] No cancel button visible

**After Electing Self**:
- [ ] "Cancel My Election" button replaces "I'll Go Next!"
- [ ] "Roll for Movement" button appears
- [ ] Cancel button is enabled (if no actions taken)
- [ ] Cancel button disabled with explanation (if actions taken)

**When Another Hero is Active**:
- [ ] "I'll Go Next!" button is disabled
- [ ] Shows which hero is currently active
- [ ] "Roll for Movement" not visible

### GM Turn Phase Panel

**During Quest Setup**:
- [ ] Shows "Quest Setup in Progress"
- [ ] Lists heroes who haven't chosen positions

**During Hero Election**:
- [ ] Shows "Heroes Electing Next Player"
- [ ] Lists eligible heroes (who haven't acted)
- [ ] Shows heroes who have acted this cycle
- [ ] No "Confirm" button visible

**During Hero Turn**:
- [ ] Shows "Active Hero: {HeroClass}"
- [ ] Shows movement status (rolled/not rolled)
- [ ] Shows action status (taken/not taken)
- [ ] No election controls visible

---

## Common Issues and Debugging

### Issue: Election Phase Never Starts

**Symptoms**: After heroes select positions, phase shows "Quest Setup"

**Check**:
- All heroes have confirmed starting positions
- Console shows "All heroes have selected starting positions"
- Check `handleQuestSetupComplete` was called

**Expected Fix**: Ensure all hero position confirmations are received

### Issue: "I'll Go Next!" Doesn't Start Turn

**Symptoms**: Button click does nothing, or requires GM confirmation

**Check**:
- Console shows "Player {id} elected themselves"
- Console shows "Auto-started turn for player"
- Check `handleRequestElectSelfAsNextPlayer` calls `ConfirmElectionAndStartHeroTurn`

**Expected Fix**: Should auto-start without GM action

### Issue: Cancel Button Always Disabled

**Symptoms**: Cannot cancel even without rolling dice

**Check**:
- Console log for hero turn state
- Check `heroState.MovementDice.Rolled` value
- Check `heroState.ActionTaken` value

**Expected Fix**: Both should be false before any actions

### Issue: Last Hero Not Auto-Elected

**Symptoms**: Election phase continues with only 1 hero eligible

**Check**:
- Console shows hero count: "X heroes remaining who haven't acted"
- Check `handleRequestCompleteHeroTurn` auto-election logic
- Check `GetHeroPlayers()` returns correct count

**Expected Fix**: Should auto-elect when count = 1

---

## Testing Completion Checklist

### Single Hero Scenario
- [ ] Auto-starts turn after quest setup
- [ ] No election phase shown
- [ ] Can roll movement immediately

### Multiple Hero Scenario
- [ ] Election phase appears after quest setup
- [ ] "I'll Go Next!" auto-starts turn
- [ ] No GM confirmation required

### Cancel Functionality
- [ ] Can cancel before rolling/acting
- [ ] Cannot cancel after rolling dice
- [ ] Cannot cancel after taking action
- [ ] Button shows appropriate state/message

### Auto-Election
- [ ] Last hero auto-elected (2 hero game)
- [ ] Last hero auto-elected (3+ hero game)
- [ ] Auto-elected hero cannot cancel
- [ ] Works after turn complete
- [ ] Works after election cancel

### UI Consistency
- [ ] GM panel shows correct phase
- [ ] Hero panel shows correct buttons
- [ ] Button states update correctly
- [ ] Phase indicators accurate

### Server Logs
- [ ] All expected log messages appear
- [ ] No error messages in console
- [ ] Sequence of events is correct

---

## Reporting Issues

If you find issues during testing, please note:

1. **Scenario**: Which test scenario you were running
2. **Expected**: What should have happened
3. **Actual**: What actually happened
4. **Console Logs**: Relevant server log output
5. **Browser Console**: Any JavaScript errors
6. **Steps to Reproduce**: Exact sequence that triggered the issue

---

## Summary

This testing guide covers all major flows of the automatic hero turn election system:

- ✅ Single hero auto-selection
- ✅ Multiple hero election with auto-confirmation
- ✅ Cancel validation (only if no actions taken)
- ✅ Auto-election of last remaining hero
- ✅ Proper UI state management across all phases

Complete all test scenarios to ensure the system works as designed.
