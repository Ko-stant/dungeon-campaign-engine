# ADR-0004: Movement & Passability Policy
Date: 2025-09-03
Status: Proposed

## Change Log
- Clarified that **tables/low furniture are blocking for movement** (passability), but **do not block LoS**.
- Defined **ally-passable** to apply to **same-faction entities** (heroes with heroes, monsters with monsters). On the DM’s turn, monsters treat other monsters as allies.

## Context
- Movement is **orthogonal only**.
- You may move through same‑faction units (ally-passable), but may not **end** on their tile.
- You may **not** move through enemy NPCs, furniture, walls, or closed/locked/secret doors.
- Friendly NPCs (e.g., mercenaries) are considered allies of their designated faction (typically the heroes).
- Certain spells/items allow moving **through** walls/doors (still orthogonally) but never into out‑of‑play/void space; only into active rooms/corridors.

## Decision

### Adjacency
- `neighbors(tile)` yields up to **4 orthogonal** tiles.

### Occupancy & Passability
Movement considers **edge passability** and **tile occupancy** separately.

**Tile occupancy categories (relative to the mover):**

| Occupant                          | Pass‑through | Destination |
|-----------------------------------|--------------|-------------|
| **Ally (same faction)**           | Yes          | No          |
| **Friendly NPC** (ally to mover)  | Yes          | No          |
| **Enemy NPC**                     | No           | No          |
| **Furniture (all types)**         | No           | No          |
| **Empty**                         | Yes          | Yes         |

Notes:
- “Same faction” means **heroes with heroes** and **monsters with monsters**. On the DM’s turn, monsters treat other monsters as allies (ally-passable).
- Furniture (including low items like **tables**) **blocks movement** (no pass‑through, no destination).
  - **LoS behavior** is furniture‑specific: low furniture (tables) **does not block LoS**; tall/line furniture (bookcases, cupboards) **does block LoS**.

### Thresholds (Edges)
- Edges with `DoorSocket/Gate` must be **open** to traverse, unless the mover has a **Phasing** status/effect (see below).
- `SealedThreshold` is never traversable.

### Phasing (spell/item/status)
- Permits pass‑through on `wall` and `closed/locked door` **edges only**.
- Does **not** change tile occupancy rules (you still cannot end on an occupied tile and cannot pass through enemy units or furniture).
- Does **not** allow entry into tiles outside the **Active Play Area**.

### Active Play Area
- Defined by content as an allowlist of Regions/Zones plus active Links.
- Movement into tiles outside the Active Play Area is rejected.
- Rules may expand the Active Play Area (e.g., door reveals, link activation).

## Consequences
- Pathfinding must treat **ally‑occupied tiles** as **transit‑allowed but non‑terminal**: valid interior steps, invalid final destination.
- Furniture always blocks movement; only LoS uses the “low vs tall” distinction.
- Clear faction semantics: heroes/mercenaries vs monsters behave symmetrically.

## Alternatives Considered
- Making low furniture non‑blocking for movement: rejected; not per HQ rules.
- Allowing diagonal moves: rejected; not per HQ rules.

## Validation
- Unit tests:
  - Ally pass‑through for heroes and for monsters; destination on ally tile rejected.
  - Enemy blocks both pass‑through and destination.
  - Furniture blocks movement; tables do not block LoS while bookcases do.
  - Phasing allows traversal through closed doors/walls but still forbids entering occupied tiles or out‑of‑play tiles.
- Scenario tests:
  - Corridor with alternating allies/enemies/furniture matches expected reachable sets.
  - DM moving multiple monsters can thread through allied monsters but not end on them.

## Related ADRs
- ADR‑0001 (LoS): furniture LoS behavior aligns (tables non‑blocking; tall furniture blocking).
