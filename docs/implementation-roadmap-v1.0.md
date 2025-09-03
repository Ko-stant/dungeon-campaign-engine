# Implementation Roadmap — v1.0
Status: Planning
Last updated: 2025-09-03

This roadmap emphasizes deterministic behavior, small payloads, and future-proofing for expansions.

## Phase 0 — Repo & Policy (1 day)
- Public repo with MIT license
- Add NOTICE and "No Assets" policy; .gitignore excludes packs/assets/content.
- Docs committed (v1.0).

**Acceptance:** Repo passes a license/NOTICE check.

---

## Phase 1 — Geometry Core (Segments/Regions/Thresholds) (3–5 days)
- Data structures for MapDefinition with Segments, DoorSockets, Gates, SealedThresholds, PortalLinks, Links, Zones.
- RegionMap generator per Segment (all doors closed).
- Contracts for queries:
  - `neighbors(tile, doorStates, links)`
  - `regionOf(tile)`
  - `tilesInRegion(regionId)`
  - `adjacentRegionsAcrossDoors(regionId)`
  - `pathExists(a,b,doorStates,links)` (Dijkstra/A*)
  - `los(a,b,blockers)` decision deferred but pluggable.

**Acceptance:**
- Given a seed map and door states, `neighbors` and `pathExists` match expected outcomes.
- Region and zone invariants validated by lint (no DoorSocket crossing walls; inactive links do not create passability).

---

## Phase 2 — Event Log + SSR Snapshot & Patches (3–5 days)
- Append-only event journal; monotonic eventId per game.
- Snapshot format: `mapId`, `packId`, `turn`, `doorStates`, `revealedRegions`, `entities`, `variables`.
- Patch types: `DoorStateChanged`, `RegionsRevealed`, `EntitySpawned/Updated/Despawned`, `VariablesChanged`.
- Reconnect protocol: resume from last seen eventId.

**Acceptance:**
- Join → receive SSR snapshot under 2KB (base map).
- Door toggle and region reveal produce ≤200B patches; client state equals server after replay.
- Reconnect from eventId N yields the same state as fresh join.

---

## Phase 3 — Minimal TCE (Triggers–Conditions–Effects) (4–6 days)
- Implement trigger sources: `OnQuestStart`, `OnEnterRegion`, `OnInteractThreshold`, `OnAttemptOpen`, `OnVariableChanged`.
- Conditions: `Compare`, `HasTag`, `IsDoorState`.
- Effects: `RevealRegion/Zone`, `OpenDoor/CloseDoor/LockDoor`, `ActivateLink/DeactivateLink`, `Teleport`, `SetVariable`, `PromptDM`, `WaitForDice`, `ClampMovement`, `GrantFreeStep`.
- Job model with pause/resume (prompts/dice).

**Acceptance:**
- Story door (SealedThreshold) interaction increments a counter and shows prompt; no passability.
- Annex unlock via variable change activates a Link and reveals an entry zone.
- Iron-door ingress policy works (ClampMovement or GrantFreeStep).

---

## Phase 4 — Entities & Hazards (3–5 days)
- Entity templates (heroes, monsters, furniture); tags and minimal stats.
- Hazard entity with states: hidden, revealed, armed, disarmed, spent.
- Triggers: `OnEnterTile`, `OnSearchNear`, `OnInteract`.
- Effects: `RevealHazard`, `ApplyDamage`, `ApplyStatus`, `DisarmSelf`.

**Acceptance:**
- Pit trap: entering tile reveals hazard and applies damage/status; disarm flow works.
- Hazards linted: hidden hazards have at least one reveal path.

---

## Phase 5 — Authoring Pack Loader + Validation (3–4 days)
- Load ContentPack manifest, maps, rulesets, strings, assets (placeholder).
- Validation pipeline (schema + geometry + rules lint).
- Pack version compatibility check.

**Acceptance:**
- Invalid pack rejected with actionable errors.
- Valid pack loads; SSR snapshot references packId@version.

---

## Phase 6 — DM Controls & Player Intents (2–4 days)
- Roles: DM, Player, Spectator; capability masks.
- Intents: `RequestMove`, `RequestOpenDoor`, `SubmitDice`.
- Rejections include reason codes; no side-effects on reject.

**Acceptance:**
- Invalid move rejected; no event emitted.
- Valid move produces `Move` event and `EntityUpdated` patch.

---

## Phase 7 — Editor MVP (map + rules) (timeboxed spike)
- Map editor: paint walls/doors/links; auto-regions; zone creator.
- Rules editor: node palette for TCE; job trace panel.
- Export to ContentPack skeleton.

**Acceptance:**
- Author a small quest with annex and a sealed story door entirely via editor; play it through.

---

## Performance & Scalability Targets
- Join payload < 2KB; average patch < 200B during typical play.
- Event storage append ≤ a few KB per turn.
- 100 concurrent spectators per game via patch fan-out without backpressure (SSE or WS with broadcast).
