# Turn State Management System - Design Spike

## Problem Statement

Currently, turn state is partially tracked but not comprehensive enough to:
1. Survive page refreshes (movement rolled but not shown after refresh)
2. Support database persistence for save/load functionality
3. Track per-hero actions and restrictions (e.g., search once per room per hero)
4. Fully restore game state mid-turn
5. Handle non-action activities (potions, item passing, door opening)
6. Support reactive abilities (defensive potions, hero reactions during monster attacks)

## Current State Analysis

### Existing Turn State (server-side: `cmd/server/turn_system.go`)

The `TurnState` struct currently tracks:
```go
type TurnState struct {
    Phase            string  // "hero", "gm", "monster"
    CurrentPlayerID  string
    CurrentEntityID  string  // e.g., "hero-1"
    ActionTaken      bool
    MovementLeft     int
    // ... other fields
}
```

**Problems:**
- No history of actions taken
- No tracking of *which* movement dice were rolled
- No per-hero, per-room action tracking (e.g., searches)
- State resets on refresh because client doesn't receive full context
- No support for activities vs actions (potions, item passing, door opening)
- No reaction/interrupt phase for defensive abilities
- Movement/action relationship not clearly enforced

### Current Client State (`internal/web/static/js/movementPlanning.js`)

```javascript
const turnMovementState = {
    diceRolled: false,
    maxMovementForTurn: 0,
    movementUsedThisTurn: 0,
    initialHeroPosition: null,
    movementHistory: [],
    currentSegment: null,
}
```

**Problems:**
- Not synced from server on page load
- Movement history not persisted
- No concept of "has rolled dice" vs "dice result"
- No tracking of whether hero has moved or taken action yet

## HeroQuest Game Rules Context

### Movement and Action Rules
- Heroes must roll movement dice at the start of their turn
- **Movement Strategy (mutually exclusive):**
  - **Option 1:** Move (partial or full) → Action
  - **Option 2:** Action → Move (remaining movement)
  - Cannot split movement around action (move → action → move) unless special ability
- Partial movement is allowed (can stop moving at any point to take action)
- Once movement begins, hero is locked into "move-first" strategy
- Once action is taken, hero is locked into "act-first" strategy

### Activities vs Actions
**Actions** (one per turn, consumes the action slot):
- Attack a monster
- Cast a spell
- Search for treasure/traps/secret doors
- Disarm a trap

**Activities** (free, can do multiple, don't consume action):
- Open/close doors
- Drink potions (including during combat reactions)
- Pass items to adjacent heroes (only on your turn)
- Use non-consumable items (based on item rules)

### Search Rules
- **Treasure Search:** Each hero can search once per room (per-hero, per-room limit)
- **Trap Search:** Multiple searches allowed, reveals traps in LOS
  - In rooms: reveals all traps in that room
  - In corridors: reveals all traps within line of sight from search position
  - Some quests require multiple searches to find all traps
- **Secret Door Search:** Multiple searches allowed
- Corridors currently not segmented (future spike needed)

### Reactions and Interrupts
- Potions can be drunk during monster attacks (reactive defense)
- Some heroes have reaction abilities triggered by specific events:
  - Example: Knight's "Shield Block" - can cancel damage to adjacent hero
  - Can be triggered on another player's turn or monster's turn
- Need to differentiate between "whose turn it is" vs "whose action is resolving"

### Item System
- Items have usage timing constraints (see item table below)
- Some items usable "at any time", others "before attack", "during attack", etc.
- Some items have per-turn usage limits
- Item effects can persist across actions ("next time you attack")

#### Base Game Usable Items

| Item Name | Effect | Timing |
|-----------|--------|--------|
| Potion of Battle | Reroll 1 Attack die | During attack resolution |
| Potion of Dexterity | +5 movement or guarantee pit jump | Next movement roll |
| Potion of Restoration | Restore 1 Body and 1 Mind | Any time |
| Potion of Speed | Roll double movement dice | Next movement roll |
| Caltrops | Place trap on tile you moved through | During your movement |
| Smoke Bomb | Monster becomes passable for heroes | During your movement |
| Potion of Defense | +2 Defend dice next defense | Any time, before next defense |
| Potion of Healing | Restore 1d6 Body Points | Any time |
| Potion of Strength | +2 Attack dice next attack | Any time, before next attack |
| Heroic Brew | Make two attacks instead of one | Before attacking |

## Proposed Architecture

### 1. Hero Turn State (Server-Side)

Create a new comprehensive per-hero turn state structure:

**Location:** `cmd/server/hero_turn_state.go` (new file)

```go
type HeroTurnState struct {
    // Identity
    HeroID          string
    PlayerID        string
    TurnNumber      int  // Which turn in the quest (for historical tracking)

    // Movement Tracking
    MovementDice    MovementDiceState
    MovementPath    []TileAddress     // All tiles moved this turn

    // Movement/Action State (simplified flag-based model)
    HasMoved        bool    // True once any movement happens (locks into move-first)
    ActionTaken     bool    // True once action happens
    TurnFlags       map[string]bool  // Generic flags: "can_split_movement", "can_make_extra_attack", etc.

    // Action Tracking
    Action          *ActionRecord  // The single action taken this turn (nil if not taken)

    // Activities (non-action activities)
    Activities      []Activity       // Potions used, items passed, etc.

    // Active Effects (from items/abilities waiting to trigger)
    ActiveEffects   []ActiveEffect   // "Next attack +2 dice", "Next movement doubled", etc.

    // Item Usage Tracking (resets each turn)
    ItemUsageThisTurn map[string]int  // ItemID -> usage count

    // Per-Location Action Tracking
    LocationActions map[string]LocationActionHistory  // Location key -> actions

    // Turn Event Log (doors opened, etc.)
    TurnEvents      []TurnEvent

    // Position Tracking
    TurnStartPosition  TileAddress
    CurrentPosition    TileAddress

    // Timestamps
    TurnStartedAt   time.Time
    LastActivityAt  time.Time
}

type MovementDiceState struct {
    Rolled            bool
    DiceResults       []int     // Individual die results (e.g., [2, 3, 4])
    TotalMovement     int       // Sum of dice (e.g., 9)
    MovementUsed      int       // How much movement consumed so far
    MovementRemaining int       // TotalMovement - MovementUsed
}

type ActionRecord struct {
    ActionType      string         // "attack", "cast_spell", "search_treasure", "search_trap", etc.
    TargetID        string         // Entity ID if targeting something
    TargetPosition  *TileAddress   // Position if targeting a location
    LocationKey     string         // Room ID or corridor segment key
    Success         bool           // Whether action succeeded
    Details         map[string]any // Action-specific data (damage, spell name, search results, etc.)
    Timestamp       time.Time
}

type Activity struct {
    Type        string         // "use_item", "pass_item", "open_door", "close_door"
    ItemID      string         // For item-related activities
    ItemName    string
    Target      string         // For pass_item: recipient hero ID; for doors: door ID
    Context     string         // "on_turn", "during_monster_attack", "during_hero_defense"
    Details     map[string]any
    Timestamp   time.Time
}

type ActiveEffect struct {
    Source      string    // "potion_of_strength", "heroic_brew", "tidal_surge"
    EffectType  string    // "bonus_attack_dice", "extra_attack", "bonus_movement", "can_split_movement"
    Value       int       // e.g., 2 for "+2 dice"
    Trigger     string    // "next_attack", "next_defend", "next_movement", "immediate"
    ExpiresOn   string    // "end_of_turn", "after_trigger", "end_of_quest"
    Applied     bool      // Whether the effect has been consumed
    CreatedAt   time.Time
}

type LocationActionHistory struct {
    LocationKey     string                  // "room-17" or "corridor-seg-3"
    LocationType    string                  // "room" or "corridor"
    SearchesByHero  map[string]SearchHistory  // HeroID -> their searches in this location
    FirstEntered    time.Time
}

type SearchHistory struct {
    TreasureSearches   []SearchRecord  // Each hero limited to 1 per room
    TrapSearches       []SearchRecord  // Multiple allowed
    SecretDoorSearches []SearchRecord  // Multiple allowed
}

type SearchRecord struct {
    SearchType  string         // "treasure", "trap", "secret_door"
    Success     bool
    FoundItems  []string       // Item IDs found
    Position    TileAddress    // Where the search occurred
    Timestamp   time.Time
}

type TurnEvent struct {
    EventType   string         // "door_opened", "door_closed", "monster_spawned", "gm_narration"
    EntityID    string         // Door ID, monster ID, etc.
    Details     map[string]any
    Timestamp   time.Time
}
```

### 2. Movement/Action State Machine

The state machine is based on simple flags rather than explicit phases:

```
TURN START
    ↓
Roll Movement Dice (required)
    ↓
┌─────────────────────────────────────┐
│  HasMoved=false, ActionTaken=false  │  ← Can choose either path
└─────────────────────────────────────┘
         ↓                    ↓
    ┌────────┐          ┌────────┐
    │  MOVE  │          │ ACTION │
    └────────┘          └────────┘
         ↓                    ↓
  HasMoved=true        ActionTaken=true
  (locked into           (locked into
   move-first)           act-first)
         ↓                    ↓
    ┌────────┐          ┌────────┐
    │ ACTION │          │  MOVE  │
    │(if not │          │(if not │
    │ taken) │          │ moved) │
    └────────┘          └────────┘
         ↓                    ↓
    ActionTaken=true    HasMoved=true
         ↓                    ↓
         └──────┬─────────────┘
                ↓
           TURN COMPLETE
       (or pass turn early)

SPECIAL CASE: TurnFlags["can_split_movement"] = true
    → Can move → action → move
    → Not locked into strategy
```

**Validation Logic:**
```go
func (hts *HeroTurnState) CanMove() bool {
    if !hts.MovementDice.Rolled {
        return false // Must roll dice first
    }
    if hts.MovementDice.MovementRemaining <= 0 {
        return false // No movement left
    }
    // If action taken and movement not started, can still move (act-first strategy)
    if hts.ActionTaken && !hts.HasMoved {
        return true
    }
    // If movement started, can continue moving (but action will lock it)
    if hts.HasMoved && !hts.ActionTaken {
        return true
    }
    // If both done, can only move with split ability
    if hts.HasMoved && hts.ActionTaken {
        return hts.TurnFlags["can_split_movement"]
    }
    // Neither done yet, can move
    return true
}

func (hts *HeroTurnState) CanTakeAction() bool {
    if !hts.MovementDice.Rolled {
        return false // Must roll dice first
    }
    if hts.ActionTaken {
        // Special case: extra attacks from Heroic Brew, etc.
        return hts.TurnFlags["can_make_extra_attack"]
    }
    // Can take action if not taken yet (regardless of movement state)
    return true
}
```

### 3. Turn State Manager (Server-Side)

**Location:** `cmd/server/turn_state_manager.go` (new file)

```go
type TurnStateManager struct {
    currentTurn     int
    heroStates      map[string]*HeroTurnState  // Hero ID -> state
    reactionStack   []ReactionContext          // For handling interrupts/reactions
    turnHistory     []TurnHistoryEntry         // For replay/undo
    mutex           sync.RWMutex
}

type ReactionContext struct {
    TriggerEvent    string    // "monster_attack", "hero_damaged", "trap_triggered"
    ActiveHeroID    string    // Whose turn is being interrupted
    TargetHeroID    string    // Who is being attacked/affected
    AvailableReactions []AvailableReaction
    Timestamp       time.Time
}

type AvailableReaction struct {
    HeroID      string
    AbilityID   string
    AbilityName string
    CanUse      bool
    Reason      string  // Why can/can't use
}

// Key methods:
func (tsm *TurnStateManager) StartHeroTurn(heroID, playerID string) error
func (tsm *TurnStateManager) RollMovementDice(heroID string, diceResults []int) error
func (tsm *TurnStateManager) RecordMovement(heroID string, from, to TileAddress) error
func (tsm *TurnStateManager) RecordAction(heroID string, action ActionRecord) error
func (tsm *TurnStateManager) RecordActivity(heroID string, activity Activity) error
func (tsm *TurnStateManager) AddActiveEffect(heroID string, effect ActiveEffect) error
func (tsm *TurnStateManager) TriggerEffects(heroID string, trigger string) []ActiveEffect

// Validation methods
func (tsm *TurnStateManager) CanMove(heroID string) (bool, string)
func (tsm *TurnStateManager) CanTakeAction(heroID string, actionType string) (bool, string)
func (tsm *TurnStateManager) CanUseItem(heroID string, itemID string) (bool, string)
func (tsm *TurnStateManager) CanSearchTreasure(heroID string, locationKey string) (bool, string)

// Reaction handling
func (tsm *TurnStateManager) PushReactionContext(ctx ReactionContext)
func (tsm *TurnStateManager) PopReactionContext() *ReactionContext
func (tsm *TurnStateManager) GetAvailableReactions(triggerEvent string, targetHeroID string) []AvailableReaction

// Turn completion
func (tsm *TurnStateManager) CompleteHeroTurn(heroID string) error
func (tsm *TurnStateManager) ResetTurnState(heroID string) error

// State access
func (tsm *TurnStateManager) GetHeroTurnState(heroID string) *HeroTurnState
func (tsm *TurnStateManager) SerializeForClient(heroID string) map[string]any
func (tsm *TurnStateManager) SerializeForPersistence() ([]byte, error)
func (tsm *TurnStateManager) RestoreFromPersistence(data []byte) error
```

### 4. Integration Points

#### A. Game Manager Integration

**File:** `cmd/server/game_manager.go`

Add:
```go
type GameManager struct {
    // ... existing fields ...
    turnStateManager *TurnStateManager  // NEW
    itemManager      *ItemManager       // NEW - needed for item validation
}
```

#### B. Snapshot Enhancement

**File:** `internal/protocol/snapshot.go`

Extend snapshot to include full turn state:
```go
type Snapshot struct {
    // ... existing fields ...
    HeroTurnStates  map[string]HeroTurnStateLite  // NEW: HeroID -> state
}

type HeroTurnStateLite struct {
    HeroID              string
    PlayerID            string
    TurnNumber          int

    // Movement
    MovementDiceRolled  bool
    MovementDiceResults []int
    MovementTotal       int
    MovementUsed        int
    MovementRemaining   int

    // Action/Movement state
    HasMoved            bool
    ActionTaken         bool
    ActionType          string  // Type of action taken, if any
    TurnFlags           map[string]bool

    // Activities & Effects
    ActivitiesCount     int
    ActiveEffectsCount  int
    ActiveEffects       []ActiveEffectLite

    // Location-based tracking
    LocationSearches    map[string]LocationSearchSummary  // LocationKey -> summary

    // Positions
    TurnStartPosition   TileAddress
    CurrentPosition     TileAddress
}

type ActiveEffectLite struct {
    Source      string
    EffectType  string
    Value       int
    Trigger     string
    Applied     bool
}

type LocationSearchSummary struct {
    LocationKey        string
    TreasureSearchDone bool   // Has this hero searched for treasure here
}
```

#### C. WebSocket Patch Events

**File:** `internal/protocol/events.go`

New patch types:
```go
type HeroTurnStateChanged struct {
    HeroID    string
    TurnState HeroTurnStateLite
}

type MovementDiceRolled struct {
    HeroID      string
    DiceResults []int
    Total       int
}

type ActionRecorded struct {
    HeroID     string
    ActionType string
    Success    bool
    Location   string
}

type ActivityRecorded struct {
    HeroID       string
    ActivityType string
    ItemName     string
}

type EffectAdded struct {
    HeroID      string
    EffectType  string
    Description string
}

type ReactionAvailable struct {
    TriggerEvent string
    TargetHeroID string
    Reactions    []AvailableReaction
}
```

### 5. Client-Side State Management

#### A. Enhanced Turn State

**File:** `internal/web/static/js/heroTurnState.js` (new file)

```javascript
// Replace turnMovementState with server-synced heroTurnState
export const heroTurnState = {
    heroID: null,
    playerID: null,
    turnNumber: 0,

    // Movement
    movementDice: {
        rolled: false,
        diceResults: [],      // e.g., [2, 3, 4]
        total: 0,             // 9
        used: 0,              // 4
        remaining: 0,         // 5
    },

    // Movement/Action tracking (simple flags)
    hasMoved: false,
    actionTaken: false,
    turnFlags: {},           // { "can_split_movement": true }

    // Active effects
    activeEffects: [],       // [{source, effectType, value, trigger, applied}]

    // Location tracking
    locationSearches: {},    // { "room-17": { treasureSearchDone: false } }

    // Positions
    turnStartPosition: null,
    currentPosition: null,

    // Client-side planning state (not synced to server)
    isPlanning: false,
    plannedPath: [],
};

// Validation helpers (mirrors server logic)
export function canMove() {
    if (!heroTurnState.movementDice.rolled) return false;
    if (heroTurnState.movementDice.remaining <= 0) return false;

    // Act-first: can still move after action if haven't moved
    if (heroTurnState.actionTaken && !heroTurnState.hasMoved) return true;

    // Move-first: can continue moving before action
    if (heroTurnState.hasMoved && !heroTurnState.actionTaken) return true;

    // Both done: only if split movement allowed
    if (heroTurnState.hasMoved && heroTurnState.actionTaken) {
        return heroTurnState.turnFlags["can_split_movement"] || false;
    }

    // Neither done: can move
    return true;
}

export function canTakeAction() {
    if (!heroTurnState.movementDice.rolled) return false;
    if (heroTurnState.actionTaken) {
        return heroTurnState.turnFlags["can_make_extra_attack"] || false;
    }
    return true;
}

export function getTurnStrategy() {
    if (!heroTurnState.hasMoved && !heroTurnState.actionTaken) {
        return "choose"; // Can choose either
    }
    if (heroTurnState.hasMoved && !heroTurnState.actionTaken) {
        return "move_first"; // Locked into moving first
    }
    if (!heroTurnState.hasMoved && heroTurnState.actionTaken) {
        return "act_first"; // Locked into acting first
    }
    return "complete"; // Both done
}

// Sync from server
export function syncHeroTurnStateFromServer(serverState) {
    heroTurnState.heroID = serverState.heroID;
    heroTurnState.playerID = serverState.playerID;
    heroTurnState.turnNumber = serverState.turnNumber;

    heroTurnState.movementDice = {
        rolled: serverState.movementDiceRolled,
        diceResults: serverState.movementDiceResults || [],
        total: serverState.movementTotal,
        used: serverState.movementUsed,
        remaining: serverState.movementRemaining,
    };

    heroTurnState.hasMoved = serverState.hasMoved;
    heroTurnState.actionTaken = serverState.actionTaken;
    heroTurnState.turnFlags = serverState.turnFlags || {};
    heroTurnState.activeEffects = serverState.activeEffects || [];
    heroTurnState.locationSearches = serverState.locationSearches || {};

    heroTurnState.turnStartPosition = serverState.turnStartPosition;
    heroTurnState.currentPosition = serverState.currentPosition;

    // Restore movement planning if movement remaining
    if (canMove() && heroTurnState.movementDice.remaining > 0) {
        import('./movementPlanning.js').then(mp => {
            mp.startMovementPlanning();
        });
    }
}
```

#### B. State Sync Handler

**File:** `internal/web/static/js/patchSystem.js`

```javascript
import { syncHeroTurnStateFromServer, heroTurnState } from './heroTurnState.js';

function handleHeroTurnStateChanged(patch) {
    const state = patch.turnState;
    syncHeroTurnStateFromServer(state);

    // Update UI based on state
    updateTurnUI();
}

function handleEffectAdded(patch) {
    // Show notification: "Potion of Strength active: +2 attack dice on next attack"
    showEffectNotification(patch.effectType, patch.description);
}

function handleReactionAvailable(patch) {
    // Show reaction prompt: "Monster attacking! Use Shield Block?"
    showReactionPrompt(patch.reactions);
}
```

### 6. Item System Integration

**Note:** Full item system requires a separate content model spike. For now:

**Location:** `content/base/items/` (future implementation)

Items need:
```json
{
  "id": "potion_of_strength",
  "name": "Potion of Strength",
  "type": "consumable",
  "usageRules": {
    "timing": ["any_time", "before_attack"],
    "maxUsesPerTurn": 1,
    "maxUsesPerQuest": null,
    "consumed": true
  },
  "effect": {
    "type": "bonus_attack_dice",
    "value": 2,
    "trigger": "next_attack",
    "duration": "until_triggered"
  },
  "description": "Drink this strange liquid at any time, enabling you to roll 2 extra combat dice the next time you attack."
}
```

**Item Manager** (new):
```go
type ItemManager struct {
    definitions map[string]ItemDefinition
    mutex       sync.RWMutex
}

func (im *ItemManager) ValidateItemUsage(itemID string, heroID string, context string) (bool, string)
func (im *ItemManager) CreateEffect(itemID string) *ActiveEffect
func (im *ItemManager) GetItemTiming(itemID string) []string
```

### 7. Database Persistence (Conceptual)

**Future Scope:** Database persistence will be added once the in-memory model is stable.

**Conceptual approach:**
- Store `HeroTurnState` as JSONB in PostgreSQL
- Separate tables for turn history and audit logs
- Incremental saves on state changes
- Full restoration on game load

**Key considerations:**
- Schema will evolve as gameplay systems mature
- Performance: batch writes, async persistence
- Versioning: support schema migrations
- Compression: large turn histories may need compression

## Implementation Phases

### Phase 1: Core State Structure (No DB)
- Create `HeroTurnState` struct with flag-based model
- Create `TurnStateManager` with validation methods
- Integrate with `GameManager`
- Keep state in memory only
- **Goal:** Establish state model and validation logic

### Phase 2: Client-Server Sync
- Extend snapshot with `HeroTurnStates`
- Add WebSocket patch types
- Client-side `heroTurnState` module
- Test page refresh scenarios
- **Goal:** Page refresh preserves movement/action state

### Phase 3: Activities & Effects
- Track activities (potions, item passing)
- Track active effects (pending bonuses)
- Item usage validation
- Basic item timing system
- **Goal:** Non-action activities work correctly

### Phase 4: Search & Location Tracking
- Per-hero, per-location search tracking
- Treasure search validation (once per room)
- Trap/secret door search recording
- **Goal:** Search restrictions enforced

### Phase 5: Reactions & Interrupts
- Reaction context stack
- Available reaction detection
- Client UI for reaction prompts
- Potion drinking during combat
- **Goal:** Defensive abilities and reactive potions work

### Phase 6: Item System Integration
- Full item content model (separate spike)
- Item effect triggers
- Item timing validation
- UI for item usage at correct times
- **Goal:** All items work per their rules

### Phase 7: Database Persistence (Future)
- Schema design and migration
- Serialization/deserialization
- Save/load functionality
- Mid-turn save/restore
- **Goal:** Games can be saved and restored

## Open Questions & Future Spikes Needed

### 1. Corridor Segmentation
**Problem:** Corridors are not segmented, complicating location-based tracking (especially trap searches).

**Options:**
- Define corridor segments algorithmically (e.g., every 3 tiles)
- Pre-define segments in quest JSON
- Use tile-based tracking instead of segment-based

**Action:** Separate spike document for corridor segment model.

### 2. Trap Revelation Tracking
**Problem:** When a trap is found, it's revealed but not placed on board. Heroes must remember which tiles are trapped.

**Options:**
- Track revealed traps separately from search history
- Add "known traps" to game state (tile → trap type)
- Track "sprung traps" separately from "known traps"

**Action:** Separate spike for trap tracking system.

### 3. Item Content Model
**Problem:** Items have complex timing, effects, and restrictions that need structured representation.

**Topics to cover:**
- Item definition schema
- Effect types and triggers
- Usage validation rules
- Client UI integration (when to offer items)

**Action:** Separate spike: `ITEM_SYSTEM_SPIKE.md`

### 4. Reaction Priority & Chaining
**Problem:** Multiple reactions might be available simultaneously (e.g., target drinks potion AND adjacent hero uses Shield Block).

**Questions:**
- Can multiple reactions chain?
- Who decides the order?
- Can reactions interrupt each other?

**Action:** Design reaction resolution system once basic reactions are implemented.

## Risk Assessment

### High Risk
- **State sync complexity:** Keeping client/server flags in perfect sync
- **Item system scope:** Many items with unique behaviors
- **Corridor segmentation:** Affects multiple systems (search, traps, LOS)

### Medium Risk
- **Reaction timing:** Interrupt handling can be complex
- **Performance:** Large turn histories and effect lists
- **Backward compatibility:** Migrating existing games to new structure

### Low Risk
- **Flag-based model:** Simpler than phase-based, less error-prone
- **Testing:** State machine is easy to unit test
- **Code organization:** Clean separation of concerns

## Success Criteria

1. ✓ Page refresh during movement restores flood effect and movement planning
2. ✓ Movement dice roll results persist across refresh
3. ✓ Action state (taken/not taken) persists across refresh
4. ✓ Movement/action strategy enforced (move→action OR action→move)
5. ✓ Can track activities separately from actions
6. ✓ Active effects tracked and triggered correctly
7. ✓ Per-hero, per-room treasure search limit enforced
8. ✓ Potion drinking during combat works (reactive)
9. ✓ Item usage limits enforced (once per turn, etc.)
10. ✓ Full turn state can be serialized for future database storage
11. ✓ System supports future undo/replay features

## Next Steps

1. ✓ Review and finalize this spike document
2. Create implementation tasks for Phase 1
3. Begin with `HeroTurnState` struct definition in `cmd/server/hero_turn_state.go`
4. Implement `TurnStateManager` with validation methods
5. Integrate with `GameManager`
6. Create Phase 2 tasks for client-server sync
7. Create separate spikes:
   - `ITEM_SYSTEM_SPIKE.md`
   - `CORRIDOR_SEGMENTATION_SPIKE.md`
   - `TRAP_TRACKING_SPIKE.md`