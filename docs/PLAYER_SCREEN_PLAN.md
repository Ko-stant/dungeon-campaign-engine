# Player screen (TV) plan

**Last Updated**: 2026-10-05 20:55 EDT

## Goal
At the table the GM runs the tracker on the laptop and opens a second Chrome tab, the
**player screen**, full screen on the game-room TV. The GM resolves things on the physical
board, and while the players react, quickly updates the app; the TV follows live and shows
only what the heroes know.

## Decisions (GM, 2026-10-04)
- **Layout on the TV:** the whole map layout (room and corridor outlines, walls) is always
  drawn. Rooms and corridors are the plain room color until the GM marks them visible;
  visible ones turn a light, neutral "seen" color (new theme color).
- **Contents are separate from rooms:** monsters, furniture, blocked squares and doors appear
  on the TV only when the GM marks that piece visible (line of sight). Secret doors also
  need to be found. Traps appear once revealed, triggered or disarmed (a triggered pit stays
  for the rest of the quest); blocked squares added during play always appear. Quest notes, start
  squares and trap labels never appear.
- **Revealing a room** can bring its contents along: an optional toggle, like today's
  "reveal marks monsters seen", extended to furniture, blocked squares and the room's doors
  (never hidden traps or unfound secret doors).
- **Selecting:** the GM clicks a visible piece **on the TV window** (mouse on the second
  display); a card shows its info. Monsters: name, stats (Avoidance, hit dice, damage,
  traits such as ranged or "strikes 2 in a line"), effects (Poisoned...), "Wounded" while
  Faltering, and exact Body.
- **Monster Body** is shown by default; a **tracker toggle** (per session, changeable
  mid-game) hides it if the players vote that way.
- **Party panel** on one side: each hero's Body, Mind, Mana, status and effects; the round
  and a Fight badge.
- **Event feed:** open by default (the players' vote, 2026-10-05; it was closed before), closed or reopened with the Events toggle on the TV. It shows a
  player-safe line per change and never mentions traps, GM notes, hidden monsters or
  anything the players can't see.
- **Secrecy on the server:** the player screen has its own API and stream carrying only the
  player view, so nothing secret reaches that browser tab (safe for phones later).
- Later: player docs and polls for the players (after this is built).

## Phases
1. **Visibility model** (Go tracker): a "visible to players" flag for doors
   (`DoorState.Seen`), furniture (`State.SeenFurniture`) and blocked squares
   (`State.SeenBlocks`); monsters keep `visibility`. A `seen.set` command shows or hides
   any piece; `area.reveal` with contents on also marks the room's furniture, blocked
   squares and doors seen. `State.Players.HideMonsterBody` with a `players.set` command.
   **Done 2026-10-05** (`internal/tracker/visibility.go`; the reveal summary now reads
   "Revealed Lair (seen: 2 monsters, 1 piece of furniture)"). Older sessions start with
   no doors, furniture or blocked squares seen.
1b. **Trap mechanics** (inserted 2026-10-05, done): blocked squares added during play
   (`State.AddedBlocks`, `block.add` / `block.remove`, kept per map; the players always
   see them) and a kind-aware `trap.trigger` (`internal/tracker/traps.go`): pits and long
   pits stay as permanent pits (triggered, shown to the players); a falling block turns its
   square into blocked squares; a spear trap is gone; a boulder is marked triggered, then
   the GM either drags it to where it stops and uses `trap.block` ("Turn into blocked
   squares"), or blocks the impact square with the "Block square" mode; other kinds
   (teleports, chests...) are marked triggered for the GM to keep or remove. The tracker
   has a Trigger button per trap (its label says what happens), "Turn into blocked
   squares", the "Block square" mode, and selecting an added block to clear it.
2. **GM controls** (tracker): select furniture too; a "Visible to players" toggle on the
   selection panel for every piece kind; pieces hidden from the players are drawn dimmed
   on the GM board; the reveal toggle covers contents; a header switch for monster Body;
   an "Open player screen" button. **Done 2026-10-05** except the "Open player screen"
   button, which moves to phase 4 (the page doesn't exist yet): furniture is selectable;
   monsters, furniture, quest blocked squares and doors have a "Show to players" /
   "Shown to players" button; with "Show what the heroes have seen" on, the GM board fades
   whatever the players haven't been shown; the reveal option is "Show contents too"; the
   header has "Players see monster Body".
3. **Player view on the server:** `tracker.PlayerView(state)`, a pure Go filter, and
   `GET /api/sessions/{id}/player`, plus a player WebSocket stream fed by `record()`.
   Events get a player-safe summary at `Apply` time (empty means not shown), stored with
   the event; old events show nothing.
   **Done 2026-10-05** (`internal/tracker/player.go`, `internal/app/player_api.go`,
   migration 00008 adds `session_event.player_summary`). The player view keeps the layout
   (no room names), discovered squares, shown doors (a found secret door is a plain door),
   shown furniture and blocked squares plus added ones, revealed/triggered/disarmed traps
   (no labels, never "trigger" markers), shown living monsters (stats, effects, Body unless
   hidden, Wounded when hurt and at a quarter of Body or less) and the heroes with their
   totals. Player lines come from an allow-list: heroes' Body/Mind/Mana and status, items,
   gold, abilities, rounds and fights, doors the players were shown opening or closing,
   monsters spotted, hit (numbers only while Body is shown) or slain, effects on heroes and
   shown monsters, travel, quest start and completion. Traps, notes, the script, GM log
   notes and anything hidden say nothing. `GET /api/sessions/{id}/player` gives the view
   and the last 30 lines; `/api/sessions/{id}/player-stream` pushes `{state, event,
   eventSeq}` on every change.
4. **Player screen page:** `/play/{id}/players` (templ + `pages/players.ts`). Board in
   player style (seen tint, no GM marks), party panel, piece cards on click, the event
   feed toggle, a full-screen button, and reconnection. **Done 2026-10-05**: the tracker
   header's "Open player screen" opens `/play/{id}/players` in a "player-screen" tab. The
   screen: header (quest, round, Fight badge, "Events (n new)", "Full screen", live dot);
   party panel (Body, Mind and Mana bars, effects, Fallen/Escaped); the board drawn from
   the player view (`internal/web/src/players/view.ts`; unexplored rooms plain gray,
   discovered squares the new light "seen" color, `BoardView.seenTiles`); clicking a piece
   or a hero shows its card (`players/cards.ts`: monsters' Body unless hidden, Wounded,
   stats line, Move, effects; heroes' stats and totals; furniture, trap and blocked-square
   names) in a right-hand column with the event feed, so nothing covers the map. The
   player API also sends a trimmed catalog (names, sizes, colors, artwork; never custom
   monsters' notes).
5. **Polish:** monster "abilities" text on campaign monster stat lines for the cards, TV
   sizing (large type), and then the player docs and polls. **Done 2026-10-05** except the
   player docs and polls, which need the GM: each campaign monster stat line has an
   "Abilities (shown to the players on the monster's card)" text (`MonsterCombat.Abilities`,
   500 characters; the tracker's monster panel shows it too, and `make fill-campaign`
   keeps it when `combat.json` has none). The player screen has A− / A+ text size buttons
   (100-200%, remembered by the TV's browser).

## Next (needs the GM)
- Player docs and polls (2026-10-05, done as published pages; sources in
  `docs/campaigns/three-plagues/players/`): the Players' Handbook
  (https://claude.ai/artifact/C1FiMQLNVA3pw5Jmvu7Ma2, `handbook.html`: the house rules,
  monster mechanics without per-monster numbers or spoilers, the TV, a sheet per hero with
  hit odds at Avoidance 6-16, a glossary) and Table Polls
  (https://claude.ai/artifact/LatPNDvxX5j7p3edeQgHNL, `polls.html`: narration audio vs
  the GM, monster Body on the TV, critical misses, monster crits, who tracks cooldowns
  and Determination, who describes special crits, the TV log open or closed; votes in the
  page's `db` under `votes/<viewer id>`, each player writes only their own). Players need
  claude.ai accounts and an email invite as Editor (no public link) to vote. Hero card
  art: the GM's originals are in `players/cards/*.png` (not in git, about 11 MB); the
  handbook publishes 800 px copies from `players/cards/web/`. Mind became Will
  (2026-10-05): the handbook has "Mind and magic" and the TV hero card a Will line.
- Fill in abilities text for the campaign's monster stat lines (campaign page) if wanted.
- Try the player screen on the real TV: text size, the "seen" color and the board size.

## Running log (overnight run, 2026-10-05)
The GM asked (2026-10-05, before bed) to commit phase 3, then work through the remaining
phases committing after each ("the same way": work commit + Docs hash commit), and push
to origin once all phases are done. Resume from the last line here after a compaction.
- 81ac447 phase 3 committed (Docs ad736f3). Next: phase 4, the player screen page.
- Phase 4 built and checked in the browser at 1920x1080 on a copy of the dev database
  (live updates, cards, feed, Body switch, Wounded). Committed ea027a1. Next: phase 5.
- Phase 5 built and checked in the browser (abilities text on a card, A+ text size).
  Committed 427a0da. All phases done; pushed to origin (main at 2398f6e, 2026-10-05).
- 2026-10-05 (later): b2d6247 (Docs 7b9d3df) committed the handbook, polls, web-sized
  card art and Mind-as-Will (Barbarian 3, Ranger 4, Rogue 5, Cleric 6; the GM applied
  `make fill-campaign` so the dev DB classes have them). Since then, uncommitted: the
  handbook's class base / gear / total sheets (published as version 4), the GM Bestiary
  (`scripts/gm-bestiary.ts`, `gm/bestiary.html`, published). The other session's tracker
  changes (corridor-drag reveal, 0-size monsters selectable) landed separately as 698a46c
  and 01ad0e0 (Docs c64f307).
