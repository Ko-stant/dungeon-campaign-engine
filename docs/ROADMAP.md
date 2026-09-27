# Roadmap

**Last Updated**: 2026-09-27 14:20 EDT

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

## Next
### Table polish
- [ ] Tracker layout for smaller screens (collapsible side panels, bigger board).
- [ ] Drag to move pieces; keyboard nudges for the selected piece.
- [ ] Event log filters (round, hero, monster) and search.
- [ ] Monster detail panel with catalog stats (attack/defend/move) and notes.
- [ ] Friendlier advisory messages in the map creator (names instead of ids).

### Player TV view (read-only)
- [ ] `/table/{sessionId}`: heroes' view only (discovered squares, seen monsters,
      found doors, revealed traps), driven by the existing session stream.
- [ ] Optional big-screen mode (no side panels, large board).

### Helping the GM remember
- [ ] "Already searched" markers per room (treasure / traps / secret doors), per
      hero if wanted. (From the old roadmap's search tracking.)
- [ ] Line-of-sight suggestions when a door opens: offer to reveal what the
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
- [ ] Campaign quest list and progress (which quests are done).

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
