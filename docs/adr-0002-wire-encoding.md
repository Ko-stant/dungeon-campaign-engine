# ADR-0002: Wire Encoding — MessagePack First, Protobuf Later If Needed
Date: 2025-09-03
Status: Accepted (initial)

## Context
- Greenfield; fast iteration trumps strict schema lock-in.
- Transport is WS/SSE/HTTP; payloads are small (sub‑KB typical).
- Cross-language support required (Go server, JS client).

## Decision
Start with **MessagePack** for all over-the-wire payloads (snapshots, patches, intents, events).

### Envelope
Every message includes:
- `protocolVersion` (u8)
- `messageType` (u8 enum)
- `gameId` (string/uint)
- `seq` (monotonic per-connection for patches)
- `eventId` (when applicable)

### Compatibility discipline
- Avoid silent shape drift. Changes require:
  - Increment `protocolVersion` (minor for additive, major for breaking).
  - Keep old fields stable; add new fields with defaults.
- Document message shapes in `/docs/event-log-and-patch-protocol-v1.0.md`.

### Migration path to Protobuf
If we hit encoding/CPU bottlenecks:
- Introduce **dual-stack**: a `transport=protobuf` flag at handshake.
- Define `.proto` schemas for hot paths (Patch, Snapshot, Intent, Event).
- Maintain MessagePack for a deprecation window before switching defaults.

## Consequences
- + Rapid iteration, small binary payloads, simple tooling.
- – No schema enforcement at compile-time; discipline needed in docs/tests.

## Alternatives Considered
- **Protobuf** now: strong contracts but slower iteration and a steeper initial setup.
- **FlatBuffers/Cap’n Proto**: great for zero-copy, overkill for our small payloads.

## Validation
- Snapshot size kept under 2 KB typical; patches under 200 B typical.
- Encode/decode microbenchmarks in CI to catch regressions.
