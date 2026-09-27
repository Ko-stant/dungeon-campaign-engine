# Upgrade and Table-Companion Pivot - Progress Tracker

**Last Updated**: 2026-09-27 13:20 EDT
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

---

## Phase 0 - Branch and baseline [DONE 2026-09-27]

- [x] Branch `dce-table-only` created from `main` (899a473).
- [x] Backup of untracked `content/`, `assets/`, `docs/base_campaign_reference/`:
      `~/dce-backups/dce-content-assets-20260927-1250.tgz` (168 MB).
- [x] Baseline recorded with the pre-upgrade toolchain (`GOTOOLCHAIN=go1.25.0`).
      Raw outputs are in `docs/baseline/` (gitignored).
- [x] Golden regression test `cmd/server/baseline_golden_test.go` with
      `cmd/server/testdata/golden/quest01_core.json`. Covers board walls, region map,
      doors, blocked tiles, furniture, and line-of-sight grids (hero start, plus six
      corridor vantage points with every door open). Regenerate only on purpose:
      `go test ./cmd/server -run TestBaselineGolden -update`.
- [x] `cmd/server/main_test.go` TestMain skips the package when `content/` is absent,
      so a fresh clone does not panic.
- [ ] Synthetic `testdata/` content fixture: deferred to Phase 5/6, where the surviving
      `internal/content` loaders and the importer are written test-first. The
      `cmd/server` tests are multiplayer code that Phase 8 deletes.

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

### Browser smoke checklist (run via the Claude built-in browser)

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
      (`version: "2"`) that excludes `cmd/server/` until Phase 8; `internal/` and all
      new code are linted. Result: 0 issues. The v2 defaults found 8 issues, all in
      doomed `cmd/server` files; the 4 DEFERRED_BUGS warnings are also there.
- [x] Renamed `internal/geometry/adjacent_text.go` to `adjacent_test.go`. The test had
      never run, and it now surfaces a real bug: `BuildRegionMap` uses a right/bottom
      edge convention while `RegionsAcrossDoor` and the production board loader use
      left/top. It is skipped with an explanation and gets fixed in Phase 6.
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
- [x] 3c Tailwind v3.4.17 -> v4.3.3 via `@tailwindcss/cli` (no PostCSS/autoprefixer).
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

## Phase 4 - Fresh TS client foundation (test-first)
- [ ] tsconfig (TypeScript pinned 6.0.3; typescript-eslint doesn't support TS 7 yet)
- [ ] Bun build pipeline + watch with templ proxy reload
- [ ] `src/board/geometry.ts`, `types.ts`, `renderer.ts`, and the `/dev/board`
      comparison page

## Phase 5 - Postgres persistence
## Phase 6 - Map creator
- [ ] Unify the edge convention in `internal/geometry` (`BuildRegionMap`, dev layouts)
      and unskip `TestRegionsAcrossDoor`.
## Phase 7 - Companion tracker
## Phase 8 - Remove multiplayer, docs cleanup

---

## Open follow-ups
- The local nvm Node 22.18.0 is an x64 build under Rosetta (Node 20.19.1 is arm64).
  The project no longer needs Node, but reinstalling Node 22 as arm64 would avoid
  surprises in editor tooling.
- The Go static file server sends no cache headers, so browsers keep stale CSS/JS
  across rebuilds (seen during the Tailwind check). Phase 4 adds
  `Cache-Control: no-cache` in dev.
- Optional: pin the Tailwind v3 palette in `@theme` if the v4 color shift is unwanted.
- Move to TypeScript 7 when typescript-eslint supports it.
- Xcode.app 26.2 is behind CLT 26.6. Update from the App Store when convenient.
