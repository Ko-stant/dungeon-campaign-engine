# Authoring & Content Packs — Spec v1.0
Status: Draft
Last updated: 2025-09-03

## 1) Scope & Goals
Define how maps, entities, thresholds, hazards, rules (TCE graphs), and assets are packaged and validated as data-only content.

## 2) Pack Manifest
```yaml
ContentPack:
  id: string              # e.g., "qhq.base"
  version: string         # semver
  engineMinVersion: string
  dependencies:
    - { id: string, version: string }
  locale: [ "en-US" ]
  assets: AssetIndex
  maps: [ MapDefinitionRef ]           # see geometry-topology-v1.0
  entities: [ EntityTemplate ]
  hazards: [ HazardTemplate ]
  rulesets: [ Ruleset ]                 # node graphs-
  constants: { key: value }             # e.g., movement minima
  strings: { key: { "en-US": "..." } }  # UI/DM text

AssetIndex:
  sprites:
    heroToken: "sprites/hero.png"
    doorClosed: "sprites/door_closed.png"
```

## 4) Maps & Segments

- Include MapDefinition objects (see geometry spec).

- For annexes, include additional Segments and Links.

- Provide zones for narrative convenience (e.g., "ThroneRoom", "AnnexEntry").

## 5) Entities & Hazards

- EntityTemplate: baseline stats, tags, abilities, token sprite key.

- HazardTemplate: shape, default state, detection/disarm rules, effects references.

## 6) Rulesets (node graphs)

A Ruleset includes:

- id, version

- triggers[] (entry nodes)

- nodes[] (conditions, effects, control)

- edges[] (graph wiring)

- variables with scope, type, defaults

- textKeys for prompts

## 7) Localization

All player-facing text must use strings keys, not inline literals.

## 8) Packaging & Hashing

- Packs are archives with a manifest (content.json/content.msgpack) + assets.

- Each file includes a SHA-256; the pack has a top-level hash for caching.

## 9) Validation Pipeline

- Schema validation (required fields, enums).

- Geometry lint (see geometry spec §7).

- Rules lint (see rules spec §10).

- Asset references exist and are within the pack.

- Dependency versions satisfied.

## 10) Migration & Versioning

- Minor version: non-breaking additions (new nodes, new zones).

- Major version: breaking geometry or variable changes; provide a migration note.

- Saved games reference pack id@version; engine refuses to load incompatible packs unless a migration exists.

## 11) Repository & IP Policy (authoring)

- Content packs containing third-party IP (rule text, art, trademarks) must not be published in the public engine repo.

- Keep such packs private or local. Use generic names and placeholder assets in public examples.

## 12) Example Skeleton

```
/packs/
  qhq.base/
    content.json
    maps/
      board01.json
      annex01.json
    sprites/
      hero.png
      door_closed.png
    rules/
      base_ruleset.json
    strings/
      en-US.json
```