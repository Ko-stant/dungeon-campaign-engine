# Geometry & Topology — Spec v1.0
Status: Draft
Last updated: 2025-09-03

## 1) Scope & Goals
Define a geometry model for grid-based tactical play that:
- Supports NxN boards and expansion annexes.
- Models doors, gates, sealed “story” doors, and teleports.
- Enables deterministic reveal/movement and efficient path queries.
- Keeps runtime updates compact (region/fog, door states).

## 2) Core Concepts

### 2.1 Segment
A self-contained grid with its own tiles/edges. A Map may include multiple Segments (e.g., base board + annex).

### 2.2 Tile
An integer coordinate `(x, y)` in a Segment.

### 2.3 Edge
A potential adjacency between two tiles in the same Segment (vertical or horizontal). Edges may host Thresholds (e.g., a door).

### 2.4 Threshold (taxonomy)
An interactive boundary element that may or may not permit movement.

- **DoorSocket** — A door-capable edge between two tiles. Runtime state: `closed | open | locked | secret`.
  Invariant: connects exactly two Regions **within the same Segment**.
- **Gate/Portcullis** — Like a DoorSocket but may be one-way or timed by rules.
- **SealedThreshold** — Looks like a door; never passable. Used for narrative interactions.
- **PortalLink** — Interaction point that activates a Link (teleport/stairs). May be on a tile or edge.

### 2.5 Link
A connection that moves an entity across non-adjacent tiles or between Segments.
- Fields: `fromTile`, `toTile`, `oneWay | twoWay`, `active: bool`, optional conditions.
- Movement across Segments is **impassable unless a Link is active**.

### 2.6 Region
A maximal connected set of tiles **assuming all DoorSockets are closed**. Precomputed per Segment. Used for fog-of-war and cheap reachability.

### 2.7 Zone
A named, author-defined set of tiles for narrative and triggers. May span multiple Regions and Segments.

### 2.8 Addressing
All coordinates are segment-scoped. Canonical tile address: `(segmentId, x, y)`.
Edge address: `(segmentId, x, y, orientation)` where `orientation ∈ {vertical, horizontal}` and `(x,y)` is the lower/left tile of the pair.

## 3) Data Shapes (normative)

> Formats are illustrative; actual encoding (JSON, MessagePack, etc.) is up to the engine.

```yaml
MapDefinition:
  id: string
  version: string
  segments: SegmentDefinition[]
  links: LinkDefinition[]
  thresholds: ThresholdDefinition[]   # DoorSockets, Gates, Sealed, PortalLinks
  zones: ZoneDefinition[]

SegmentDefinition:
  id: string
  width: int
  height: int
  # Walls define edges that are never passable and cannot host a DoorSocket.
  walls:
    vertical: [ { x: int, y: int } ]     # edge between (x,y) and (x+1,y)
    horizontal: [ { x: int, y: int } ]   # edge between (x,y) and (x,y+1)
  regionMap: RegionMap                   # precomputed (see below)

ThresholdDefinition:
  id: string
  segmentId: string
  kind: enum [DoorSocket, Gate, SealedThreshold, PortalLink]
  edge?: { x: int, y: int, orientation: enum }  # for edge-based thresholds
  tile?: { x: int, y: int }                      # for tile-based PortalLinks
  initialState?: enum [closed, open, locked, secret, never]  # "never" = sealed
  oneWay?: boolean
  appearance?: { visible: boolean, spriteKey?: string }
  interaction?: { promptKey?: string, scriptRef?: string }

LinkDefinition:
  id: string
  from: { segmentId: string, x: int, y: int }
  to:   { segmentId: string, x: int, y: int }
  oneWay: boolean
  active: boolean
  conditions?: ConditionRef[]   # evaluated in rules

ZoneDefinition:
  id: string
  name: string
  tiles: [ { segmentId: string, x: int, y: int } ]

RegionMap:
  segmentId: string
  # Each tile maps to a regionId. Size = width*height. Out-of-board tiles omitted.
  tileRegionIds: int[]  # row-major, -1 for impassable/outside
  regions: [ { id: int, tileCount: int } ]
```

## 4) Invariants & Constraints

- DoorSocket edges always join two valid, distinct Regions within one Segment.

- SealedThreshold on a boundary does not connect Regions.

- Links may connect any two tiles (same or different Segment).

- The outer boundary of each Segment is impassable unless a Link crosses it.

- RegionMap is deterministic for a given Segment definition.

## 5) Core Queries (engine contracts)

- neighbors(tile, doorStates, links) -> [tile]

- regionOf(tile) -> regionId | null

- adjacentRegionsAcrossDoors(regionId) -> [{ edge, otherRegionId }]

- pathExists(a, b, doorStates, links) -> boolean

- tilesInRegion(regionId) -> [tile]

- los(a, b, blockers) -> boolean (algorithm pluggable; must be deterministic)

## 6) Fog & Reveal

- Default reveal granularity: Region. Effects may reveal by Region or Zone.

- Per-tile reveal may be layered later; Region-level is the required baseline.

## 7) Validation & Lint Rules

- Every DoorSocket references two tiles that are not separated by a permanent wall.

- No two Thresholds occupy the same edge or tile unless explicitly allowed (e.g., PortalLink on a DoorSocket is disallowed).

- Inactive Links must not create passable paths in neighbors.

- Zones must reference valid tiles; report empty Zones.

## 8) Examples (brief)

- Annex: Add a new Segment annex01. Place a PortalLink at the base board’s “secret door”; define a Link to annex01 entry tile, active=false until a puzzle is solved.

- Story door: Place a SealedThreshold on an edge; attach an interaction script; movement always blocked.

## 9) Versioning & Compatibility

- MapDefinition.version follows semver. Any geometry changes that alter Region IDs or passability require a minor or major bump and content pack migration notes.

## 10) Glossary

- Segment, Tile, Edge, Threshold, DoorSocket, Gate, SealedThreshold, PortalLink, Link, Region, Zone.