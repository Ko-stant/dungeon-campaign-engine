# ADR-0003: Snapshot Cadence & Size Budget
Date: 2025-09-03
Status: Proposed

## Context
Clients join/resume frequently. Replaying a full event log is wasteful. We need a compact, authoritative “now” to render SSR and resume streams reliably.

## What is a Snapshot?
A **Snapshot** is a compact, deterministic serialization of the current authoritative game state sufficient for a client to render and resume without replaying all past events.

### Included
- `mapId`, `packId@version`, `protocolVersion`
- `turnNumber`, `lastEventId`
- `doorStates` (packed)
- `revealedRegions` (bitset) and/or revealed `zones`
- **Active** entities (minimal fields to render):
  - id, kind, tile (segmentId,x,y), hp (current,max), tags, visible statuses
- Key quest variables needed for UI/rules (whitelist)
- Optional: UI hints (ambiguity markers) that don’t affect rules

### Not Included
- Full event history
- Non-render-critical variables or logs
- Transient UI local state

### Hash
Include `stateHash` (32-bit or 64-bit) for divergence checks.

## Decision
**Cadence**
- Always snapshot at **start of each turn**.
- Also snapshot when either:
  - `structuralChange=true` (activation/deactivation of Links, segment added, mass reveal), or
  - More than **64 events** have occurred since last snapshot.

**Size Budget**
- Target **≤ 2 KB** typical; hard cap **≤ 4 KB**.
- Breakdown targets:
  - doorStates: ≤ 64 B
  - revealedRegions/zones: ≤ 64 B
  - entities: ≤ 1.2 KB (≤ 40 entities × ~30 B each)
  - variables: ≤ 512 B

**Retention**
- Persist one snapshot per turn; keep last **64 turns**.
- Each snapshot references `lastEventId` so replay tail is bounded.

**Join/Resume Flow**
1. HTTP SSR sends latest Snapshot + initial HTML.
2. Client opens WS/SSE with `lastEventId`; server streams patches from there.
3. On gaps or checksum mismatch, server re-sends Snapshot + tail.

## Consequences
- Fast, bounded rejoins; predictable storage.
- Minimal over-the-wire data during normal play.
- Slight write amplification on turns with many small events (mitigated by the 64-event rule).

## Alternatives Considered
- Snapshot every N events only: can misalign with turn UI.
- No snapshots: replay time grows with session length.

## Validation
- Measure Snapshot size across representative scenarios.
- Reconnect tests: SSR → patches → identical state vs. from-scratch join.
- State hash comparison in CI golden replays.
