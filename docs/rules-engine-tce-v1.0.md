# Rules Engine (TCE) — Spec v1.0
Status: Draft
Last updated: 2025-09-03

## 1) Scope & Goals
Provide a visual, data-driven system to author mechanics as **T**riggers → **C**onditions → **E**ffects, without editing engine code. Support base game rules, expansions, and custom content.

## 2) Execution Model
- A **Trigger** produces a **Job** with deterministic ID.
- The Job evaluates a node graph: Conditions gate execution; Effects emit engine events.
- Effects are the **only** way to change state.
- Jobs may **pause** on prompts, timers, or waits; resumed deterministically.
- Concurrency: multiple Jobs may run; Effects apply in the order they are emitted.
- Determinism: all random outcomes (dice) enter as inputs/events, not hidden RNG.

## 3) Variables & Scopes
- Namespaces: `quest.*`, `room.*`, `zone.*`, `entity.*`, `team.*`.
- Types: `int`, `float`, `bool`, `string`, `enum`, `set<string>`, `map<string,*>`.
- Lifetime: `quest.*` persists across the quest; others scoped as named.
- Watching: `OnVariableChanged(name)` triggers when a variable changes.

## 4) Triggers (non-exhaustive)
- **Lifecycle**: `OnQuestStart`, `OnTurnStart(faction|entity)`, `OnTurnEnd`.
- **Map/Movement**: `OnEnterTile(tile)`, `OnEnterRegion(region)`, `OnEnterZone(zone)`, `OnApproachThreshold(threshold)`, `OnAttemptOpen(threshold)`, `OnInteractThreshold(threshold)`, `OnEnterLink(link)`.
- **Combat/Status**: `OnAttackDeclared`, `OnAttackResolved`, `OnEntityDefeated(tag|id)`, `OnStatusApplied(tag)`.
- **Inventory**: `OnItemAcquired(item)`, `OnItemUsed(item)`.
- **Variables & timers**: `OnVariableChanged(name)`, `OnTimer(name)`.
- **Dice**: `OnDiceRolled(source, spec)`.

## 5) Conditions (examples)
- `Compare(left, op, right)` where `op ∈ {==, !=, <, <=, >, >=}`
- `HasTag(entity, tag)`
- `InLOS(a, b)`
- `Count(selector, op, value)` e.g., entities in zone
- `IsDoorState(threshold, state)`
- `RandomChance(p)` (only valid when seeded by explicit dice result or a declared random gate that emits a DiceRoll request)

## 6) Effects (examples)
- **Map**: `RevealRegion(region)`, `RevealZone(zone)`, `OpenDoor(threshold)`, `CloseDoor(threshold)`, `LockDoor(threshold)`, `ActivateLink(link)`, `DeactivateLink(link)`, `Teleport(entity, tile)`
- **Entities**: `Spawn(kind, at)`, `Despawn(id)`, `MoveForced(entity, toTile)`
- **Combat/Status**: `ApplyDamage(entity, amount|rollRef)`, `ApplyStatus(entity, status, duration)`, `RemoveStatus(entity, status)`
- **Inventory**: `GiveItem(entity, item)`, `TakeItem(entity, item)`
- **Variables**: `SetVariable(name, value)`, `Increment(name, delta)`
- **Flow/UI**: `PromptDM(text, options)`, `PromptPlayers(text, options)`, `WaitForDice(source, spec)`, `Reroll(rollRef)`, `ClampMovement(entity, minDistance)`, `GrantFreeStep(entity, steps)`
- **Control**: `Sequence([...])`, `Parallel([...])`, `RepeatUntil(cond, body)`, `Gate(cond, then, else)`

*Editor should present these as a node library. Content packs may extend with new node IDs.*

## 7) Hazards & Thresholds
**Hazard entity**
- Fields: `shape (tile|edge|area)`, `state ∈ {hidden, revealed, armed, disarmed, spent}`, `detectionRule`, `disarmRule`, `tags`.
- Typical Triggers: `OnEnterTile`, `OnSearchNear`, `OnInteract`.
- Typical Effects: `RevealHazard`, `ApplyDamage`, `ApplyStatus`, `DisarmSelf`, `OpenDoor`.

**Threshold interactions**
- `OnApproachThreshold` (entity steps adjacent), `OnAttemptOpen`, `OnInteractThreshold`.
- `SealedThreshold` never grants passability; may still prompt or modify variables.

## 8) Ingress Policy (one-way entry patterns)
When heroes start **outside**:
- Policy options (content-chosen Effects):
  - `RerollOnce`
  - `ClampMovement(min=4)` for first turn
  - `GrantFreeStep(1..k)` and mark `entity.hasEntered = true`
  - Queue-based corridor outside the gate that advances automatically

## 9) Determinism & Randomness
- Dice results are inputs (`OnDiceRolled`) or results of `WaitForDice`; all downstream Effects must reference those outcomes.
- No hidden RNG inside Conditions/Effects.

## 10) Linting & Safety
- Cycles must contain a pause (`Prompt*`, `WaitForDice`, `OnTimer`) or a bounded counter.
- All Threshold references must exist and match kind.
- All Zones non-empty unless flagged `optional`.
- Variables must be declared with type before first write.
- Effects that move entities must verify passability (or be explicitly `MoveForced`).

## 11) Extensibility
- Node IDs are versioned. New nodes may be added; existing node semantics must remain backward compatible across a minor version.
- Content declares the minimum engine/node-library version it needs.

## 12) Examples (brief)
- **Story door**: `OnInteractThreshold(sealedDoor) → PromptDM(NPC text) → Increment(quest.counter,1)`
- **Pit trap**: `OnEnterTile(T) ∧ !HasTag(entity,"Flying") → RevealHazard → ApplyDamage(rollRef) → ApplyStatus("Prone")`
- **Annex unlock**: `OnVariableChanged(puzzleSolved=true) → ActivateLink(annexLink) → RevealZone(annexEntry)`