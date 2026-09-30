# Roadmap

**Last Updated**: 2026-09-29 21:30 EDT

The engine is a single-GM companion for in-person HeroQuest: build maps, run a
quest at the table, and keep a resumable record of what happened. It never
enforces rules; the GM decides, and the app remembers.

## Done (see UPGRADE_AND_PIVOT_TODO.md for details)
- Map creator: boards of any size (void / corridor / rooms, derived walls) and
  quest layers (doors, secret doors, blocked squares, furniture, monsters, traps,
  lettered notes, start squares), with undo/redo and advisory checks.
- Campaigns: heroes that carry gold, equipment and notes between quests.
- Tracker: live session state, GM commands with a readable event log, fog of
  what the heroes have seen, resumable from Postgres, live sync across tabs.

## Done after the plan (2026-09-27)
- Squares count from (1,1) at the bottom-left; sizes read columns × rows.
- Drawn walls on the board layer (e.g. between two corridors).
- Room colors; exit squares; gates and locked doors; purple secret doors.
- Blocked squares that hide a secret door, removable (and restorable) during play.
- Custom monster library (`/monsters`): color, size up to 4x4, stats, notes.
- Teleport squares and teleport traps; doors across corridors are no longer flagged.
- Multi-map campaigns: ordered chapters (quest + its map), maps grouped by campaign,
  and mid-game travel between maps within one session.

## Done 2026-09-28
- Quest-book tiles: 31 monsters, 8 furniture and 5 traps from five quest packs
  (Jungles of Delthrak, Mage of the Mirror, Rise of the Dread Moon, The Ogre Horde,
  Kellar's Keep). Catalog monsters read `gridSize` (2x1 Giant Ape, Ogre Lord, ...).
- Trap catalog (`content/traps/`): trap artwork, multi-square and rotatable traps.
- Traps can be removed during play, the boulder moves, trap labels, and a "trigger"
  marker for the GM's own effects.
- Reveal a room with its monsters (everything but traps); pick squares and reveal them
  (and the monsters on them) in one step.
- [ ] Enter stats for the new monsters (their files say 0 = not entered yet).
- Dice expressions (`2d6+1`, d4..d20) in Go and TypeScript.
- Custom hero classes (`/classes`): dice stats, accuracy, mana, class exclusives and
  abilities with cooldowns and mana costs. Next steps for the "Three Plagues" campaign are
  tracked in `NARRATIVE_AND_CLASSES_TODO.md`.
- Hero inventory (gold and items) on the campaign page and in the tracker; ability use,
  mana and cooldowns in the tracker.
- Map creator: note list on the Quest tab (removing a note relabels the rest), previews of
  furniture, traps and monsters at the chosen rotation, two-square doors and gates, and an
  exit door kind for the board's edge; N for a new room and "New room for each shape"
  (a shape started on a room extends it).
- Read-aloud script per campaign and a Read aloud panel with a large-type reader in the
  tracker, with audio clip playback per passage (files named after passage ids).

## Next
### Table polish
- [ ] Tracker layout for smaller screens (collapsible side panels, bigger board).
- [ ] Drag to move pieces; keyboard nudges for the selected piece.
- [ ] Event log filters (round, hero, monster) and search.
- [ ] Monster detail panel with catalog stats (attack/defend/move) and notes.
- [ ] Map checks for multi-square monsters: `Quest.Check` only checks a monster's
      anchor square, so a 2x2 custom monster hanging off the board or into rock is
      not flagged yet.
- [ ] Drag to draw a run of walls (drawn walls are one click per edge today).
- [ ] Friendlier advisory messages in the map creator (names instead of ids).

### Player TV view (read-only)
- [ ] `/table/{sessionId}`: heroes' view only (discovered squares, seen monsters,
      found doors, revealed traps), driven by the existing session stream.
- [ ] Optional big-screen mode (no side panels, large board).

### Helping the GM remember
- [ ] "Already searched" markers per room (treasure / traps / secret doors), per
      hero if wanted. (From the old roadmap's search tracking.)
- [ ] Line-of-sight suggestions when a door opens (must treat drawn walls as walls): offer to reveal what the
      heroes can see. The legacy LOS code is in git history (`cmd/server/visibility.go`
      before commit "Phase 8"), and would be ported into `internal/maps`.
- [ ] Wandering monster reminder from the quest's wandering monster type.

### Heroes and campaigns
- [ ] Structured equipment and artifacts (from `content/equipment`, `content/artifacts`)
      instead of free text; show stat modifiers as reminders, not enforcement.
- [ ] Spell tracking per hero (spells chosen, used this quest).
- [ ] Potions and consumables with a "used" toggle.
- [ ] Hero death and replacement across a campaign (new hero, TPK handling,
      replaying a quest). (From the old roadmap's hero death section.)
- [x] Campaign quest list and progress (which quests are done). (Chapters.)
- [ ] Per-hero travel: today the whole party travels together; splitting the party
      across maps is not modelled.

### Map creator
- [ ] Export/import board + quests as a JSON bundle (backup and sharing).
- [ ] Tracing image under the grid for copying a printed map.
- [ ] Duplicate a board or quest.
- [ ] Import the remaining original quests (`make import-content QUEST=...`) once
      their JSON exists.

### GM house rules (optional reminders, never enforced)
- [ ] Per-campaign notes on house rules (e.g. doubles on dice, bargains) shown in
      the tracker. The old custom dice-rule ideas become reminders, since dice are
      rolled at the table.

### Tooling
- [ ] Move to TypeScript 7 once typescript-eslint supports it.
- [ ] CI (GitHub Actions): Go tests with a Postgres service, bun test, lint.
