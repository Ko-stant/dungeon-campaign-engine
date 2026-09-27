# Upgrade and Table-Companion Pivot - Progress Tracker

**Last Updated**: 2026-09-27 14:41 EDT
**Branch**: `dce-table-only`

Living checklist for the upgrade + pivot plan. Each step records what was done and how,
so work can resume after an interruption.

Goals, in order:
1. Baseline the current project.
2. Upgrade toolchains and packages, proving behaviour is unchanged at each step.
3. Fresh TypeScript client foundation (Bun, test-first). The old JS is reference only.
4. Postgres persistence, map creator (any board size, board + quest layers).
5. Single-GM companion tracker with a readable event log and resumable sessions.
6. Delete the multiplayer code.

## Resume here (current state)

**Status: all 8 phases are complete and committed on `dce-table-only`,** plus one
post-plan change: squares now count from (1,1) at the bottom-left and sizes read as
columns × rows (see "Post-plan: bottom-left coordinates" near the end). Nothing is
pushed or merged into `main`. The next work is in `docs/ROADMAP.md`.

What the app is now: a single-GM companion for in-person HeroQuest.
- `/maps` is the map creator. `/campaigns` lists campaigns and heroes and starts
  quests. `/play/{id}` is the live tracker. `/` redirects to `/campaigns`.
- The multiplayer engine (lobby, turns, rules, dice, legacy JS client) is deleted.
- Architecture, commands and conventions are in `CLAUDE.md`; working rules are in
  `docs/IMPORTANT.md`.

Commits (oldest first):

| Commit | Step |
|---|---|
| b5c9192 | Phase 0 baseline |
| cf26831 / 7f6cd0c | 2a go 1.27 / 2b go fix |
| 848fe30 / 1748aba | 2c modules / templ regen |
| cee5551 | 2d tools + golangci-lint v2 |
| a79a3af / 2c47285 | 3a Bun / 3b ESLint 10 |
| 11967ff | alternate preview config (later trimmed) |
| a12d824 | 3c Tailwind v4 |
| a816967 | 4 TS foundation |
| c5c8a33 | 5 Postgres 18 + store |
| 3eede05 | 6a map documents, catalogs, importer |
| 6fec549 | 6b map editor |
| 2fb7d7d | 7 tracker |
| 5c17cbe | 8 multiplayer removed, docs rewritten |
| 450cbad | docs: resume summary |
| 7a85e91 | post-plan: (1,1) at the bottom-left, columns × rows labels |
| 7bbd2e0 | post-plan: drawn walls on the board layer |

Test status after the bottom-left change:
- `make test`: 102 Go tests, 0 failures (DB tests skip without a URL).
- `make test-db`: 102 tests pass against Postgres.
- `bun test`: 106 tests pass.
- `make lint` (golangci-lint + ESLint + tsc): clean.

Local environment:
- Go 1.27.1 at `/usr/local/go`, Bun 1.4.2 (Homebrew), Postgres 18 in Docker.
- The container is `hq_postgres` on **port 5433**, volume `pgdata18`. Start it with
  `make db-up`.
- `.env` (gitignored) holds DATABASE_URL etc. and was changed from port 5432 to
  5433; the server loads it itself.
- `.claude/launch.json` config `dce-binary` runs `./build/dungeon-campaign-engine` on
  :8080, for the Claude preview browser. Run `make build` first.

Dev database contents (rebuilt 2026-09-27 14:40 for the bottom-left change; a backup of
the old top-left data is in `db/backups/backup-20260927-143900.sql`, gitignored):
- Boards: "HeroQuest Base Game Board" 26x19 (re-imported via `make import-content`) with
  quest "The Trial", and "Test Board 30x24" (landscape, rebuilt through the API) with
  quest "Test Quest".
- Campaign "Test Campaign" with heroes Grom (Barbarian) and Ilsa (Wizard), and a new
  active session "Game night 1" on The Trial (round 1, 3 events). The old session was
  removed because it was stored in top-left coordinates.
- Campaign "Test Numba 2" (no heroes, no sessions; created by the user, kept as is).

Browser-testing tips:
- Native `confirm()` dialogs are suppressed in the Claude browser (they return false).
  Stub them with `window.confirm = () => true` before clicking Complete quest or
  shrinking a board.
- Static files are served with `Cache-Control: no-cache`, so a plain reload picks up
  rebuilt CSS/JS.

Superseded notes below: Phases 0-4 describe the legacy code at the time. Later
phases replaced or removed parts of it; those spots are marked **(later: ...)**.

---

## Phase 0 - Branch and baseline [DONE 2026-09-27]

- [x] Branch `dce-table-only` created from `main` (899a473).
- [x] Backup of untracked `content/`, `assets/`, `docs/base_campaign_reference/`:
      `~/dce-backups/dce-content-assets-20260927-1250.tgz` (168 MB).
- [x] Baseline recorded with the pre-upgrade toolchain (`GOTOOLCHAIN=go1.25.0`).
      Raw outputs are in `docs/baseline/` (gitignored).
- [x] **(later: deleted in Phase 8 with the legacy code.)** Golden regression test
      `cmd/server/baseline_golden_test.go` with
      `cmd/server/testdata/golden/quest01_core.json`. Covers board walls, region map,
      doors, blocked tiles, furniture, and line-of-sight grids (hero start, plus six
      corridor vantage points with every door open). Regenerate only on purpose:
      `go test ./cmd/server -run TestBaselineGolden -update`.
- [x] `cmd/server/main_test.go` TestMain skips the package when `content/` is absent,
      so a fresh clone does not panic. **(later: replaced in Phase 8 by routing tests.)**
- [x] Content-independent tests. This was done differently from planned: the new
      packages use in-code synthetic fixtures (`fstest.MapFS` in `internal/content`,
      synthetic legacy defs in `internal/maps` and `internal/seed`). Tests that
      read the real `content/` skip when it is absent.

### Baseline results (Go 1.25.0, before any upgrade)

| Check | Result |
|---|---|
| `go build ./...` / `go vet ./...` | pass / pass |
| `go test ./...` | **19 pre-existing failures** in `cmd/server` (list below); geometry and protocol pass |
| `go test -race ./...` | same 19 failures, no data races reported |
| golangci-lint v1.64.8 (`.bin`) | 0 issues |
| `npm run test:js` | 16/16 pass |
| `npm run lint` (ESLint 9.36) | 7 errors (known) |
| `make build` | pass; templ generate produced no diff |

The 19 known-failing Go tests are all multiplayer turn-flow tests. Most fail with
"player player-1 cannot act right now", so they predate this work and are out of sync
with the turn system. They are **not being fixed** (Phase 8 deletes them). The bar
for every upgrade step is "no new failures":

```
TestGameFlow_CompleteHeroTurn          TestInstantActions_MultipleInstantActions
TestGameFlow_MovementAndActionOrder    TestMonsterSystem_CombatIntegration
TestGameFlow_ActionOncePerTurnEnforcement  TestMovement_OncePerTurn
TestGameFlow_InstantActionsUnlimited   TestTreasureSystem_QuestNoteResolution
TestGameFlow_TurnTransition            TestTurnTransition_StateReset
TestGameFlow_CombatIntegration         TestTurnTransition_MultiPlayer
TestInstantActions_DontConsumeMainAction   TestTurnTransition_PassTurnInstantEndsImmediately
TestInstantActions_OpenDoor            TestTurnTransition_ActionLimitsEnforced
TestInstantActions_Trading_RequiresAdjacency  TestTurnTransition_CompleteGameCycle
TestInstantActions_PassTurn_EndsTurn
```

### Browser smoke checklist (legacy app; historical, the lobby no longer exists)

Setup: `.claude/launch.json` config `dce-binary` runs `./build/dungeon-campaign-engine`.
- Use `localhost:8080` for the GM and `127.0.0.1:8080` for the hero. The `playerID`
  cookie beats the `?playerID=` query param, so two tabs on one host share an identity.
- The lobby "Start Game" and "Confirm Position" buttons use `window.confirm()`, which
  the built-in browser suppresses. Stub it first:
  `window.confirm = () => true`.

Steps and baseline outcome:
1. GM joins lobby, picks Game Master, Ready. OK
2. Hero joins, picks Barbarian, Ready. OK
3. GM starts game. GM lands on `/gm`, hero on `/`. OK (snapshots saved as
   `docs/baseline/{gm,hero}-snapshot.json`)
4. Hero: Select Starting Position, click a green tile, Confirm. OK. Position (4,15);
   "Confirm Position" is off-screen at 1024x768.
5. Roll Movement (2d6). OK (rolled 9)
6. Drag the hero token to (3,17). OK ("Moved to (3,17)", 3/9 used)
7. ArrowDown selects the adjacent door, `e` toggles it. OK: door opened and the
   corridor was revealed.
8. Attack. NOT TESTABLE: no monsters are spawned on the board ("No monsters on the
   board").
9. Refresh the hero page. Wall/visibility rendering glitches after refresh (the known
   two-GameState bug).

Pre-existing UI bugs observed (not fixing; doomed code):
- GM header stays "Quest Setup / Waiting for game to start" after the hero's turn starts.
- GM quest label shows "dev-map".
- Barbarian panel shows Mind 6/6 while the header shows 2/2; weapon shows Dagger.
- Quest setup panel shows the placeholder "The Trial of Champions".

---

## Phase 1 - Machine toolchains [DONE 2026-09-27]

- [x] Go 1.27.1 via the official `go1.27.1.darwin-arm64.pkg` (sha256 verified) into
      `/usr/local/go`. The installer is kept at `~/dce-backups/installers/`.
- [x] Command Line Tools for Xcode 26.6 via `softwareupdate` (Homebrew required it).
      Xcode.app itself is still 26.2; Homebrew only warns about that.
- [x] Bun 1.4.2 via `brew install oven-sh/bun/bun`. No Rust or Zig toolchain is needed.
- [x] Rebuilt `~/go/bin` tools with Go 1.27.1: gopls, staticcheck, templ, air, goose.
- [x] `docker pull postgres:18` (for Phase 5).

## Phase 2 - Go upgrades [DONE 2026-09-27]

Verification for every step: build, vet, the golden baseline and the full test suite
(normal and `-race`). Each result was compared against the Phase 0 failure list:
same 19, no new failures.

- [x] 2a `go 1.27.0` in go.mod (cf26831).
- [x] 2b `go fix ./...` modernizers (7f6cd0c). The changes are maps.Copy,
      range-over-int, `interface{}` -> `any`, and one ineffective `omitempty` on a
      struct field removed. JSON output is unchanged.
- [x] 2c modules (848fe30): templ v0.3.943 -> v0.3.1020, coder/websocket v1.8.13 ->
      v1.8.15. The templ CLI is now a go.mod `tool` directive and the Makefile uses
      `go tool templ`, which fixes the generator/runtime version drift.
      Regenerated `*_templ.go` (1748aba). The only non-header change is that
      attribute values now go through `templ.ResolveAttributeValue`.
      Browser smoke re-run: the server-rendered GM and hero HTML is byte-identical
      to the baseline, and the snapshot JSON is identical except for dice rolls
      (compared order-insensitively, since Go map order is random).
- [x] 2d dev tools pinned in the Makefile: air v1.67.4, gotestsum v1.13.0, goose
      v3.28.0, golangci-lint v2.14.0 (the `/v2` module path). Added `.golangci.yml`
      (`version: "2"`) that excludes `cmd/server/` until Phase 8 **(later: exclusion
      removed in Phase 8; the config is now just `version: "2"`)**; `internal/` and all
      new code are linted. Result: 0 issues. The v2 defaults found 8 issues, all in
      doomed `cmd/server` files; the 4 DEFERRED_BUGS warnings are also there.
- [x] Renamed `internal/geometry/adjacent_text.go` to `adjacent_test.go`. The test had
      never run, and it now surfaces a real bug: `BuildRegionMap` uses a right/bottom
      edge convention while `RegionsAcrossDoor` and the production board loader use
      left/top. It is skipped with an explanation and gets fixed in Phase 6
      **(later: fixed in Phase 6a, and the package moved to `internal/legacy` in
      Phase 8)**.
      `BuildRegionMap`, `DevSegment` and `CorridorsAndRooms*` are not used by
      production code.

## Phase 3 - JS toolchain [DONE 2026-09-27]

- [x] 3a npm to Bun (a79a3af). `bun.lock` was migrated with identical versions and
      `package-lock.json` removed. The Makefile uses `bun run`. Tailwind v3 output was
      byte-identical.
- [x] 3b ESLint 10 (2c47285). eslint 10.11, @eslint/js 10.0.1,
      eslint-config-prettier 10.1.8, and `globals` 17.12 is now declared explicitly.
      The legacy `internal/web/static/js/**` is ignored as reference-only (ESLint 10's
      recommended set would report 11 errors there, up from 7, via the new
      `no-useless-assignment` rule). The home-made JS test framework and its 16
      geometry tests were deleted. `bun test --pass-with-no-tests` passes until
      Phase 4 adds tests.
- [x] 3c Tailwind v3.4.17 -> v4.3.3 via `@tailwindcss/cli`, with no PostCSS/autoprefixer
      (a12d824). **(later: in Phase 8 the legacy JS and `@source` globs were removed;
      Tailwind scans `internal/web/views/**/*.templ` and `internal/web/src/**/*.ts`.)**
  - Ran the official `@tailwindcss/upgrade` through npx; bunx could not load its
    native engine. It renamed classes in templ and JS: `rounded`->`rounded-sm`,
    `backdrop-blur-sm`->`backdrop-blur-xs`, `outline-none`->`outline-hidden`,
    `flex-shrink-0`->`shrink-0`, `bg-gradient-*`->`bg-linear-*`.
  - **Tool bugs caught and fixed:** it rewrote the Go string `"ring"` (a jewelry
    inventory slot) to `"ring-3"` in `cmd/server/inventory.go` (reverted). It
    hand-edited `*_templ.go` (reverted, then regenerated from `.templ`). Its `@theme`
    output referenced itself (`--color-surface: rgb(var(--color-surface))`).
  - Theme: the raw color channels are now `--rgb-*` in `:root`, and `@theme` exposes
    `--color-*` for utilities. The legacy canvas code (`rendering.js`,
    `entityRendering.js`, `types.js`) reads `--rgb-*`.
  - `@import 'tailwindcss' source(none)` plus explicit `@source` lines mirror the v3
    content globs exactly. The v3-compat base layer keeps the gray-200 default
    border color and `cursor: pointer` on buttons.
  - Tailwind runs on Bun's runtime (`bunx --bun`) because the local nvm Node 22.18.0
    is an **x64 (Rosetta) build**, which can't load the arm64 native modules.
  - Watch mode uses `--watch=always`. Plain `--watch` exits when stdin closes, as
    it does for background jobs in `make dev`.
  - **Verification:** computed-style fingerprints of every element on the lobby,
    GM and hero pages (v3 vs v4, same game state), with colors compared as
    rendered sRGB:
    - lobby 53/53, GM 165/165, hero 328/328 elements with **no width/height
      changes**, apart from a 4px-shorter GM quest-notes panel;
    - most color differences render identically;
    - the rest are v4's intentionally updated default palette (OKLCH), e.g.
      amber-600 (217,119,6)->(225,113,0) and green-400 (74,222,128)->(5,223,114);
    - margin changes come from v4's `space-y` mechanics (bottom margins instead of
      top) and don't move anything;
    - the spawn-monster modal fields use an undefined `bg-surface-dark` class, so
      they are now transparent instead of the browser's gray;
    - the canvas board renders the same.

## Phase 4 - Fresh TS client foundation (test-first) [DONE 2026-09-27, a816967]

- [x] Tooling: TypeScript 6.0.3 (pinned; typescript-eslint 8.70 needs <6.1), typescript-eslint
      with `strictTypeChecked` + `stylisticTypeChecked`, `@types/bun`. `tsconfig.json` is
      strict with `noUncheckedIndexedAccess` and `exactOptionalPropertyTypes`.
      `make lint` = golangci-lint + ESLint + `tsc`. ESLint uses `defineConfig`
      (`tseslint.config` is deprecated).
- [x] Build pipeline: `scripts/build.ts` bundles every `internal/web/src/pages/*.ts` into
      `internal/web/static/dist/<name>.js` (gitignored, excluded from Air). `make build`
      runs it with `--minify`. `make dev` runs `watch:web`, which rebuilds on change and
      calls `go tool templ generate --notify-proxy` to reload the browser.
- [x] `internal/web.NoCache` (Go, test-first) sets `Cache-Control: no-cache` on `/static/`,
      so rebuilt CSS/JS is never served stale; unchanged files still get a cheap 304.
- [x] Test-first modules, 35 bun:test tests:
  - `src/board/geometry.ts`: grid metrics for any board size (no 26/19 fallback),
    tile rects, pixel-to-tile and pixel-to-edge hit-testing, edge/tile relations,
    rotated furniture footprints, furniture draw box, door marker rects.
  - `src/board/model.ts`: `BoardView` types. Regions are `-1` void, `0` corridor,
    `>0` room; walls are derived wherever neighbouring regions differ, with
    off-board treated as void.
  - `src/board/renderer.ts` (canvas, checked visually): draws a `BoardView` passed in,
    with no global state, so the editor, tracker and future TV view share it.
- [x] New `components.Page(title, entry)` templ shell for TS-driven pages.
- [x] Throwaway `/dev/board` + `/dev/board.json` (`cmd/server/dev_board.go`) **(later:
      deleted in Phase 8)**. It renders board.json + quest-01 through the new renderer. Verified:
      rooms, walls, doors, rotated furniture, monsters and blocked squares all in the
      right places; hover hit-testing reports `door-1` as `edge horizontal (3,18)` and
      tile (2,16) as region 20 (the starting room).
- [x] Documented the test-first convention and new commands in `CLAUDE.md` and
      `docs/IMPORTANT.md`.

## Phase 5 - Postgres persistence [DONE 2026-09-27, c5c8a33]

- [x] `docker-compose.yml`: postgres:16 -> **18.6** on a **new** volume `pgdata18`, mounted at
      `/var/lib/postgresql` as the 18 image expects. The old Postgres 16 volume
      `dungeon-campaign-engine_pgdata` is **left untouched**; delete it by hand if unwanted.
      The host port moved to **5433** (`.env`: POSTGRES_PORT, DATABASE_URL, GOOSE_DBSTRING)
      because another project's Postgres 17 container holds 5432. The missing `./db/init`
      mount was removed.
- [x] Migrations: the never-used `00001`/`00002` were replaced by `00001_init.sql` with
      tables `board`, `quest`, `campaign`, `game_session` (current state as jsonb +
      event_seq) and `session_event` (append-only, unique per session+seq), keyed by
      `uuidv7()`. `db.Migrations` embeds them; `store.Migrate` runs goose as a library.
- [x] `internal/store` (pgx/v5): board/quest/campaign/session CRUD. `RecordEvent`
      atomically replaces session state and appends the event (rollback tested).
      Malformed ids return ErrNotFound; deleting a board still used by a quest returns
      ErrInUse. The ON DELETE RESTRICT violation is SQLSTATE 23001, not 23503.
- [x] `internal/store/storetest`: each DB test gets a throwaway migrated schema in the dev
      database. `make test-db` runs the tests with the DB **(later: all packages, via
      `$(GO_PACKAGES)`)**. Tests skip without a URL.
- [x] Importer: `internal/seed.ImportLegacy` (idempotent, DB-tested) plus
      `cmd/import-content` / `make import-content [QUEST=...]`. The base board and
      "The Trial" are imported into the dev database.

## Phase 6 - Map creator [DONE 2026-09-27, 6a 3eede05, 6b 6fec549]
- [x] 6a documents (Go, test-first) in `internal/maps`:
  - `Board`: version, width, height (1..200), row-major regions (-1 void, 0 corridor,
    >0 room) and rooms. **(later: squares count from (1,1) at the bottom-left and
    regions start from the bottom row; see "Post-plan: bottom-left coordinates")** Walls are derived with the same rule as the TS client.
    `Checksum()` covers the layout only (not room names).
  - `Quest`: board checksum, doors (edge, normal/secret, open/closed), blocked-square
    rects, furniture (catalog type, quarter-turn rotation), monsters (optional body/mind
    overrides), traps (hidden/revealed/triggered/disarmed, optional furniture), lettered
    notes, start tiles, wandering monster.
  - `Validate()` rejects malformed data. `Check(board, sizes)` returns **advisory**
    issues only (door into rock, piece on void, board edited since the quest was
    saved, ...); it never blocks saving.
  - `BoardFromLegacy` / `QuestFromLegacy` convert the old JSON: blocking walls become
    rects, notes are sorted by letter, and start tiles are the tiles of the starting
    room. The real base board + quest-01 convert with **zero issues** (content-gated
    test).
- [x] `internal/content`: hero/monster/furniture catalogs from any `fs.FS` (tests use
      in-memory synthetic files). Monster stats come from `content/monsters/*.json`.
      Real content: 12 furniture, 8 monsters, 9 heroes.
- [x] Edge convention unified: deleted the unused `BuildRegionMap`, `DevSegment` and
      `CorridorsAndRooms*` (the only code using the right/bottom convention) and rewrote
      `TestRegionsAcrossDoor` against the real convention. It runs now instead of
      being skipped.
- [x] 6b map editor (`/maps`, `/maps/{id}/edit`):
  - `internal/app` (Go, DB-tested with httptest): JSON API
    - `GET /api/catalog`
    - `GET/POST /api/boards`, `GET/PUT/DELETE /api/boards/{id}`
    - `GET/POST /api/boards/{id}/quests`, `GET/PUT/DELETE /api/quests/{id}`

    Saving a quest re-binds it to the current board; responses include advisory
    `issues`. A board with quests can't be deleted (409). There are also
    server-rendered `/maps` (list + create form) and editor shell pages.
  - `internal/dotenv` (tested): the server loads `.env`, so DATABASE_URL works under
    make dev, go run and the preview tool. `cmd/server/app_mount.go` mounts the app
    (migrate, store, catalog) next to the legacy routes, and answers 503 with
    instructions if the database is missing.
  - TS, test-first (77 bun tests in total):
    - `maps/types.ts` mirrors the Go JSON; `maps/api.ts` is a typed client with an
      injectable fetch.
    - `editor/model.ts` has immutable operations: paint, rectangle, new/rename room
      (empty rooms pruned), resize (top-left anchored), cycle door
      none/normal/secret, place/move/rotate/remove pieces, lettered notes, start
      squares, `itemsAt`, `toBoardView`.
    - `editor/history.ts`: undo/redo with dirty tracking. `editor/tools.ts`: what
      each tool does on click/drag, plus `lineTiles` for gap-free brush strokes.
  - `pages/mapEditor.ts` does the wiring:
    - **Board layer**: corridor / solid rock / room brush, rectangle fill, rooms
      list with rename and tile counts.
    - **Quest layer**: select/move, door, blocked squares, furniture (type +
      rotation), monster, trap, note (text editor), start square, erase.
    - Header with resize, undo/redo and save; quest picker/create; checks panel;
      shortcuts Ctrl/Cmd+S, Z, Shift+Z, Y, R, Delete, Esc; a warning before leaving
      with unsaved changes.
  - Renderer additions: start squares, lettered note markers, drag previews.
  - Tailwind now also scans `internal/web/src/**/*.ts` (it was missing, so TS-only
    classes were not generated).
  - **Verified in the browser**:
    - Created a 24x30 board through the form, rectangle-filled a corridor, added
      and renamed two rooms, and saved; it survived a reload.
    - Created a quest and placed a door, monster, trap, furniture, start squares and
      a note with text. Dragged the monster, rotated the furniture with R, and
      saved with Cmd+S. The API shows every change persisted.
    - A monster placed on rock produced the advisory "is on solid rock" issue.
    - The imported base board + "The Trial" render fully in the editor.
    - Two bugs found and fixed during testing: the header layout shifted as the
      hover readout appeared (it's now an overlay), and a panel re-render on
      pointer-up stole focus from the note textarea.
  - Deferred polish: JSON export/import download, tracing image, friendlier issue
    wording (names instead of ids), monster stat overrides in the selection panel.

## Phase 7 - Companion tracker [MVP DONE 2026-09-27, 2fb7d7d]

- [x] 7a `internal/tracker` (Go, test-first):
  - `State` holds frozen board + quest copies, round, heroes (position/placed,
    body/mind and max, gold, equipment, notes, status), monsters (catalog stats or
    quest overrides, hidden/seen, alive), live door states (secret doors start
    not found), trap states, used notes, and discovered squares (tile-level, so a
    future TV view can use them).
  - `NewSession` puts heroes on the start squares in order and reveals the
    starting area.
  - `Apply(state, command)` supports: `move`, `hero.update`, `monster.add|update|remove`,
    `door.set` (re-closing allowed), `trap.set` (any state to any state),
    `area.reveal`, `tiles.reveal|hide`, `note.consume`, `round.advance|set`,
    `log.note`. It never enforces rules, only rejects malformed commands, and
    returns a readable event such as "Moved Grom (Barbarian) from (1,14) to (4,17)".
    It never mutates its input.
  - `CarryOver` copies gold, equipment and notes back to the campaign heroes.
- [x] 7b server (`internal/app`, DB-tested):
  - Campaign API: `GET/POST /api/campaigns`, `GET/PUT /api/campaigns/{id}` (hero
    classes are validated and ids assigned).
  - Session API: `POST /api/campaigns/{id}/sessions` (start + "session.start"
    event), `GET /api/sessions/{id}`, and
    `POST /api/sessions/{id}/commands`. Commands are applied under a per-session
    lock, then the state and event are saved in one transaction.
  - `GET /api/sessions/{id}/events?after=N`.
  - `POST .../complete` carries progress to the campaign; a completed session
    refuses commands (409) until `.../reopen`.
  - `GET .../stream` is a WebSocket that pushes every change to all open tabs.
  - Pages: `/campaigns` (list + resume active sessions + create),
    `/campaigns/{id}` (heroes add/remove, start a quest, session list), and
    `/play/{id}` (tracker shell).
- [x] 7c tracker page (TS):
  - `tracker/view.ts`, `interaction.ts`, `format.ts` and `api.ts` are
    test-first (95 bun tests in total).
  - `pages/tracker.ts` includes:
    - Round counter and next round.
    - A fog toggle showing what the heroes have discovered.
    - Hero cards: body/mind ± buttons, gold, status, equipment and notes.
    - Board modes: select/move (click a door to open or close it), reveal area,
      hide square, add monster.
    - A selection panel for monsters (body, seen/hidden, kill/revive, remove),
      doors (open/close, found) and traps (4 states).
    - Quest notes with used toggles, a free-text log entry, and a live event log.
    - A WebSocket live indicator with reconnect and catch-up, and
      complete/reopen.
  - Hidden monsters are drawn dimmed. Door clicks use a 35% edge tolerance
    (tested) because squares are small at the table.
- [x] Verified in the browser:
  - Created a campaign and added Grom (Barbarian) and Ilsa (Wizard).
  - Started "The Trial"; the heroes appear on the stairway start squares, all 24
    monsters are hidden, and fog shows only the start room.
  - Took a Body point, moved Grom, opened and then re-closed door-1, added a
    free-text log note, and advanced to round 2.
  - Reloaded the page and everything resumed exactly.
- Deferred polish: tracker layout at narrow widths (the board is small at 1024px
  with both 20rem panels), drag-to-move, line-of-sight reveal suggestions, event
  log filters, monster stat panel, player TV view.

## Phase 8 - Remove multiplayer, docs cleanup [DONE 2026-09-27, 5c17cbe]

- [x] Deleted the legacy multiplayer server:
  - `cmd/server` except the new `main.go` + `app_mount.go`: lobby, turn/election
    systems, rules engine, dice, debug routes, legacy game state, the golden
    baseline test and its 19 known-failing tests.
  - `internal/protocol`, `internal/ws`, and the legacy templ views/components
    (only `components/page.templ` remains).
  - The vanilla JS client (`internal/web/static/js`) and the `/dev/board`
    comparison page.

  About 22k lines removed. The line-of-sight code is kept in git history for
  the roadmap's reveal suggestions.
- [x] `internal/geometry` became `internal/legacy`: only the board/quest JSON readers
      the importer needs.
- [x] New `cmd/server/main.go` (tested):
  - `/` redirects to `/campaigns`; `/lobby` etc. are gone (404).
  - `/static/` is served no-cache and `/assets/` from `assets/`.
  - App routes answer 503 with instructions when the database is missing.
  - Graceful shutdown.
- [x] Lint and styles: the golangci-lint exclusion, the ESLint legacy ignore and the
      legacy-only CSS are removed. Tailwind now scans only templ views + TS
      sources.
- [x] `make test` has zero failures (the 19 legacy failures went with the legacy
      code). `make test-db` covers all packages.
- [x] `.claude/launch.json` was trimmed to the single `dce-binary` config.
- [x] Docs:
  - `CLAUDE.md` and `README.md` are rewritten for the companion app.
  - New `docs/ROADMAP.md`, with the still-relevant ideas from the old roadmap
    carried over.
  - `docs/IMPORTANT.md` is updated (no USE_LOBBY; the GM is never blocked).
  - Deleted the election/turn docs, `DEFERRED_BUGS.md`,
    `CONTENT_EFFECTS_IMPLEMENTATION_PLAN.md` and `DEVELOPMENT_ROADMAP.md`.
- [x] Verified in the browser: `/` goes to campaigns, the in-progress session resumes
      after a server restart with identical state and log, all board art loads
      through the new `/assets/` route, and the map creator lists both boards.

## Post-plan: bottom-left coordinates, columns × rows [DONE 2026-09-27]
Asked for: the board's coordinates start at (1,1) in the lower-left corner, and the
default shape is landscape. Decisions (asked and answered):
- Coordinates change in **storage too**, not just on screen.
- Landscape: keep 26 × 19 as the default, label sizes "Columns (width)" and
  "Rows (height)" with a note that the long side runs left to right, and rebuild the
  portrait test board as 30 columns × 24 rows. (The user writes sizes rows-first, e.g.
  "24x30" meant 24 rows by 30 columns.)

What changed (tests written first, then code):
- Convention: x runs 1..width left to right, y runs 1..height bottom to top. Regions
  are row-major from the bottom row (`(y-1)*width + (x-1)`). A horizontal edge (x,y) is
  now the **bottom** side of tile (x,y) (vertical is still the left side), so the
  "which two squares does an edge separate" rule is unchanged: (x-1,y)|(x,y) and
  (x,y-1)|(x,y). Furniture and blocked squares anchor at their bottom-left square.
- Versions: board/quest documents `maps.CurrentVersion` 1 -> 2, session state
  `tracker.StateVersion` 1 -> 2. `Validate` rejects older documents with a
  "re-import or recreate" message, and `tracker.Apply` refuses a version 1 state, so old
  top-left data is never silently misread. No automatic upgrade was written: the only
  stored data was imported or test data.
- Go: `Board.OnBoard/Index/TileAt` helpers; `Walls`, `RegionAt`, `Quest.Check` door and
  area bounds, tracker bounds/indexes use them. Event summaries and advisory messages
  print the stored (now bottom-left) coordinates directly.
- Legacy import: `internal/maps/legacy.go` converts from the top-left, 0-based legacy
  files (tile -> (x+1, H-y); horizontal edge -> (x+1, H-y+1); blocks -> bottom-left
  anchor). `QuestFromLegacy` and `seed.ImportLegacy` now take a furniture `SizeLookup`
  (the rotated height is needed for the anchor); `cmd/import-content` loads the catalog.
  The real base board + quest-01 still convert with zero placement issues.
- TS: `board/geometry.ts` is the only place rows flip (pixel <-> square): `tileRect`,
  new `footprintRect` and `edgeSegment`, `pixelToTile`, `pixelToEdge`, `doorRect`,
  `furnitureDrawBox` (now takes metrics and returns pixels). `board/model.ts` gains
  `tileAt` and `onBoard`; the renderer, editor model (resize now keeps the bottom-left,
  so new rows appear on top), tracker hover and map editor use them. `DOC_VERSION = 2`
  in `maps/types.ts`.
- UI: the new-board form and editor header say columns × rows; the help text explains
  landscape and the (1,1) origin. The form's grid columns widened to fit the labels.
- Verified in the browser: hover reads (1,1) at the bottom-left and (26,19) at the
  top-right of the base board; "Top-left" rooms are at the top; The Trial's furniture
  and doors sit where they did; click-to-move in the tracker moves up the screen as y
  grows and logs "Moved Grom (Barbarian) from (3,2) to (2,4)"; the 30x24 test board
  renders landscape.

## Post-plan: drawn walls [DONE 2026-09-27]
Found while building the first real map: on the physical board, two stretches of
corridor are sometimes separated only by a wall (e.g. between columns 10 and 11 for
y=1..8, and between rows 2 and 3 at x=24). Derived walls only appear where regions
differ, so the workaround was a column of solid rock, which wastes squares.

What changed (tests first):
- Board documents gain `drawnWalls` (interior edges, omitted when empty). No version
  bump: older version 2 boards simply have none. `Validate` rejects boundary,
  off-board, badly oriented and duplicate drawn walls.
- Go: `Board.IsWall` (derived or drawn), `IsInteriorEdge`; `Walls()` appends drawn
  walls that are not already derived; `Checksum()` includes drawn walls (order-free),
  unchanged for boards without them, so quests are flagged when walls are drawn after
  they were saved. `Quest.Check` no longer reports `door-same-region` for a door on a
  drawn wall (a door between two corridors).
- TS: `deriveWalls(cols, rows, regions, drawn)`; `BoardView.drawnWalls`;
  `toggleWall` (ignores the outer edge and edges already walled by differing regions;
  always removes an existing drawn wall); `resizeBoard` drops drawn walls that stop
  being interior; the tracker draws the frozen board's drawn walls.
- Editor: a **Wall** brush on the Board tab. Click an edge to draw a wall, click again
  to remove it; clicking an edge that is already a wall shows a status message.
- Not browser-verified yet: the user was running their own server at the time. Unit
  tests cover the model, tools and view; the renderer change is the extra argument
  to `deriveWalls`.

---

## Open follow-ups
- The local nvm Node 22.18.0 is an x64 build under Rosetta (Node 20.19.1 is arm64).
  The project no longer needs Node, but reinstalling Node 22 as arm64 would avoid
  surprises in editor tooling.
- Optional: pin the Tailwind v3 palette in `@theme` if the v4 color shift is unwanted.
- Move to TypeScript 7 when typescript-eslint supports it (also in docs/ROADMAP.md).
- The old Postgres 16 Docker volume `dungeon-campaign-engine_pgdata` is unused;
  remove it with `docker volume rm dungeon-campaign-engine_pgdata` once you're sure
  nothing in it matters.
- Xcode.app 26.2 is behind CLT 26.6. Update from the App Store when convenient.
