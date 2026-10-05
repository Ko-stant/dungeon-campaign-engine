# Player screen (TV) plan

**Last Updated**: 2026-10-05 01:20 EDT

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
- **Event feed:** closed by default, opened with a toggle on the TV. It shows a
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
   an "Open player screen" button.
3. **Player view on the server:** `tracker.PlayerView(state)`, a pure Go filter, and
   `GET /api/sessions/{id}/player`, plus a player WebSocket stream fed by `record()`.
   Events get a player-safe summary at `Apply` time (empty means not shown), stored with
   the event; old events show nothing.
4. **Player screen page:** `/play/{id}/players` (templ + `pages/players.ts`). Board in
   player style (seen tint, no GM marks), party panel, piece cards on click, the event
   feed toggle, a full-screen button, and reconnection.
5. **Polish:** monster "abilities" text on campaign monster stat lines for the cards, TV
   sizing (large type), and then the player docs and polls.
