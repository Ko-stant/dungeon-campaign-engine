# Testing & Validation — v1.0
Status: Living document
Last updated: 2025-09-03

## 1) Principles
- Deterministic replay: same inputs → same state.
- Small payloads: region-level reveal and tiny deltas.
- Strict separation: engine state changes only via events; rules only emit effects.

## 2) Test Layers
### Unit (pure functions)
- RegionMap generation: stable for same Segment.
- neighbors/pathExists/los with controlled door states and links.
- TCE node evaluation (Condition/Effect primitives).

### Property tests
- Opening a DoorSocket never decreases reachability within a Segment.
- Inactive Links do not allow cross-segment paths.
- Every DoorSocket connects exactly two distinct Regions within one Segment.
- No Zone references an invalid tile.

### Scenario tests (golden replays)
- Story door interaction increments counter; no movement through it.
- Pit trap: hidden → reveal → damage → status → optional disarm; replay stable.
- Annex unlock: variable flip → link activation → reveal; reachable after activation only.
- Iron door ingress: entry guaranteed per policy; replays identical given the same dice input.

### Fuzz
- Random placements of walls and door sockets within bounds; ensure invariants hold.
- Random TCE graphs with linter enabled; engine rejects invalid graphs.

## 3) Linting (Editor + CI)
- Geometry: DoorSocket on wall? SealedThreshold on boundary? Two thresholds on same edge?
- Rules: cycles without pause, undeclared variables, missing node references.
- Packs: missing assets, dangling ids, dependency unsatisfied.

## 4) Deterministic Replay Harness
- Input: Snapshot S, Events [N..M].
- Output: State hash H.
- Golden: commit pair (S hash, H) per scenario; CI compares.

## 5) Performance Budgets
- SSR snapshot: < 2KB baseline maps.
- Patch: < 200B common ops (door toggle, reveal region, entity update).
- Event append: < 100µs average on local dev DB (SQLite/Turso), excluding I/O.

## 6) Failure Modes & Handling
- IntentRejected: no side-effects; include reason code.
- Patch gaps: client resyncs from last eventId; idempotent apply.
- Content pack mismatch: refuse to load; show required engine/node-library version.

## 7) Observability
- Event log viewer: filter by type/entity/threshold.
- State hash at each event for quick divergence detection.
- Counters: patches/sec, bytes/sec, spectators connected.

## 8) Security & Trust
- DM authoritative; players send intents only.
- Validate all ids from clients; reject unknown/foreign ids.
- No eval of arbitrary script; TCE is declarative only.

## 9) Open Decisions (track via ADRs)
- Line-of-sight algorithm choice and parameters.
- Over-the-wire encoding (MessagePack vs Protobuf).
- Snapshot cadence and compression.
