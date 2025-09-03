# Event Log & Patch Protocol — Spec v1.0
Status: Draft
Last updated: 2025-09-03

## 1) Purpose
Guarantee deterministic gameplay with server-authoritative state, SSR initial render, and compact incremental patches to all clients.

## 2) Roles & Authority
- **DM**: full control; may emit administrative Effects.
- **Player**: sends **Intents** (requests); server validates and emits authoritative **Events**.
- **Spectator**: receives patches only.

## 3) Event Envelope
```yaml
Event:
  id: int64                 # monotonically increasing per game
  gameId: string
  turn: int32
  type: string              # e.g., "Spawn", "OpenDoor", "RevealRegion"
  payload: object
  timestamp: int64
  causationId?: int64       # optional, for tracing
```

## 4) Canonical Event Types (baseline)

- Lifecycle: StartGame, Join, EndTurn

- Map: OpenDoor{thresholdId}, CloseDoor{thresholdId}, LockDoor{thresholdId}, RevealRegion{id}, RevealZone{id}, ActivateLink{id}, DeactivateLink{id}, Teleport{entityId,toTile}

- Entities: Spawn{entity}, Despawn{entityId}, Move{entityId,from,to}

- Combat/Status: AttackDeclared, AttackResolved, ApplyDamage, ApplyStatus, RemoveStatus

- Inventory: Pickup, Drop, Trade

- Variables: SetVariable{name,value}, Increment{name,delta}

- Dice: DiceRolled{source,spec,results}

## 5) Intents vs Events

- Clients send Intents (RequestMove, RequestOpenDoor, SubmitDice).

- Server validates against current state:

- Accept → emit one or more Events.

- Reject → reply with IntentRejected{code, reason}.

- Codes: OUT_OF_RANGE, BLOCKED, NO_ACTIONS_LEFT, INVALID_TARGET, NOT_YOUR_TURN, LOCKED, UNKNOWN_ID.

## 6) Join & SSR Handshake

- HTTP SSR response includes a Snapshot:
```yaml
Snapshot:
  mapId: string
  packId: string
  turn: int
  doorStates: binary      # packed states for known thresholds
  revealedRegions: binary # bitset by region
  entities: [EntityLite]
  variables: { key: value }
```
- After SSR, clients subscribe (WS/SSE) to Patch stream.

## 7) Patch Messages (minimal deltas)

- DoorStateChanged{thresholdId, state}

- RegionsRevealed{ids[]}

- ZoneRevealed{id}

- EntitySpawned{entity}

- EntityUpdated{id, changes{ field: value }}

- EntityDespawned{id}

- VariablesChanged{ entries{ key: value }}

- LogAppend{ eventId, type, payloadDigest } # optional for UI log

- Patches are idempotent; clients de-dup using eventId.

## 8) Ordering & Delivery

- Server guarantees in-order delivery per game session.

- On reconnect, client provides last seen eventId; server resends from that point.

- Patches refer to canonical IDs (thresholdId, entityId) stable within a game.

## 9) Transport & Encoding

- WS recommended for duplex; SSE is acceptable for one-way.

- Encoding: implementation-defined (e.g., MessagePack). Payloads should remain under a few KB per patch in typical play.

## 10) Snapshots & Replay

- Snapshot at least at start of each turn.

- Full replay = apply Snapshot then Events with id > snapshotEventId.

- Undo (optional) = roll back to prior Snapshot and re-apply a prefix of Events.

## 11) Performance Guidelines

- Initial join payload target: < 2 KB for base maps (excluding art).

- Average patch: < 200 bytes (door toggle, region reveal, small updates).

- Avoid per-tile updates; prefer Region- or Zone-level changes.

## 12) Test Checklist

- Join -> subscribe -> move -> door open -> reveal: client state matches server after each patch.

- Reconnect from eventId = N: state matches fresh join.

- Intent rejections produce no state changes.

- Out-of-order or duplicated patches are safely ignored.