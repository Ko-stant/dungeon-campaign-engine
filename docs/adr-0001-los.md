# ADR-0001: Line of Sight (LoS) Algorithm
Date: 2025-09-03
Status: Proposed

## Context
- LoS is center-of-tile → center-of-tile.
- Blockers: walls, closed/locked/secret doors, NPCs (hostile and friendly), other Heroes, and *some* furniture. Low furniture like tables do not block.
- DM may override when it’s close.
- Engine must be deterministic and fast; works on a grid with explicit edges/thresholds and entity occupancy.

## Decision
Adopt **2D DDA ([Amanatides & Woo](http://www.cse.yorku.ca/~amana/research/grid.pdf)) grid traversal** (“ray marching”) from source tile center to target tile center, with **edge-aware occlusion**:

1. Compute the continuous ray between centers.
2. Traverse grid cells using DDA; at each step:
   - If crossing a grid **edge**, check that edge’s threshold:
     - Block if `wall`, `DoorSocket` state ∈ {closed, locked, secret}, or `Gate` closed/locked.
   - If entering a **cell** with an entity whose `blocksLoS=true`, block.
3. **Tie handling (ambiguous cases)**:
   - If the ray exactly coincides with an edge or corner within ε, mark as **Ambiguous**.
   - Engine defaults to **blocked** on ambiguity (closed-on-ties) for consistency.
   - Surface ambiguity to DM for optional override.
4. Furniture sets `blocksLoS` based on template. Defaults:
   - Tables/low items: false
   - Tall/line items (bookcases, fireplaces): true
5. Provide three evaluation modes (quest-configurable):
   - `strict`: closed-on-ties (default)
   - `permissive`: open-on-ties
   - `dm-adjudicated`: mark Ambiguous and require manual decision/UI override

## Consequences
- Deterministic, O(tiles crossed) time; stable across platforms.
- Edge-aware checks align with our geometry model (Thresholds on edges).
- Ambiguity surfaced explicitly; no hidden “magic” rulings.

## Alternatives Considered
- **Bresenham supercover**: simpler but awkward with edge thresholds and tie cases.
- **Shadow-casting / Permissive FOV**: great for area FOV, less direct for per-pair LoS and edge thresholds.
- **Ray sampling**: approximate; risks false positives/negatives on ties.

## Validation
- Unit tests for:
  - Door closed vs open along path
  - NPC in intervening cell blocks LoS
  - Table does not block; bookcase does
  - Corner and along-edge cases produce Ambiguous in `dm-adjudicated`
- Golden tests for known tricky maps.
- Performance check: ≤ 20 µs per LoS on commodity hardware.

## Links
- Geometry & Topology v1.0 (Thresholds, edges, regions)
