# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.
Also read `docs/IMPORTANT.md` (working rules) before making changes.

## Project Overview

Dungeon Campaign Engine is a **single-GM companion app for in-person HeroQuest**. The GM builds
maps in the browser, runs a quest at the table, and records outcomes (moves, body points, doors,
traps, monsters seen) so a session can be resumed later with a readable event log.

It deliberately **enforces no game rules**: the GM may re-close doors, un-trigger traps, revive
monsters or move anything anywhere. The app only records the result and describes each change.

Stack: Go 1.27 (net/http, templ, pgx), Postgres 18 (Docker), TypeScript 6 bundled by Bun,
Tailwind CSS v4, canvas rendering.

## Common Commands

### Everyday
- `make db-up` - Start Postgres (host port 5433; see `.env`)
- `make dev` - Tailwind watch + TS watch + templ proxy + Air hot reload (app on :8080, proxy on :7331)
- `make import-content [QUEST=base/quests/quest-01.json]` - Import the legacy base board + a quest (idempotent)
- `make build` - Production build to `./build/dungeon-campaign-engine`

### Testing and quality
- `make test` - Go tests (database tests skip without a URL)
- `make test-db` - Go tests including database tests (each uses a throwaway schema)
- `make test-race` - Go tests with the race detector
- `bun test` / `make test-js` - TypeScript unit tests (bun:test)
- `make lint` - golangci-lint v2 + ESLint 10 (typescript-eslint) + `tsc`
- `make fmt` - gofmt + go vet

### Frontend
- `bun install` - JS dependencies (Bun is the package manager and runtime; Node is not needed)
- `bun run build:web` / `bun run watch:web` - Bundle `internal/web/src/pages/*.ts` into `internal/web/static/dist/`
- `bun run tailwind:build` / `bun run tailwind:watch` - Tailwind v4 (config is CSS-first in `internal/web/static/styles/index.css`)

### Database
- `make db-migrate-up` / `make db-migrate-down` / `make db-migrate-new` - goose CLI (the server also migrates on start)
- `make db-psql`, `make db-backup`, `make db-restore`

## Architecture

- `cmd/server` - Entry point: static files, `/` redirects to `/campaigns`, mounts `internal/app`.
- `cmd/import-content` - Imports legacy `content/board.json` + quest JSON as map documents.
- `internal/app` - HTTP layer: JSON APIs, templ pages, per-session command locking, WebSocket stream.
  - Maps: `/maps`, `/maps/{id}/edit`, `/api/boards...`, `/api/quests...`, `/api/catalog`
  - Custom monsters: `/monsters` (server-rendered forms; table `custom_monster`). `/api/catalog`
    merges them into `monsters` with `custom-<uuid>` ids; use `Server.catalogFor(ctx)`, not
    `s.catalog`, wherever monsters are looked up.
  - Tracker: `/campaigns`, `/campaigns/{id}`, `/play/{id}`, `/api/campaigns...`, `/api/sessions/{id}/(commands|events|complete|reopen|stream)`
- `internal/maps` - Board and quest documents (Go), validation, advisory `Check`, legacy converters.
- `internal/tracker` - Session `State`, `NewSession`, `Apply(state, command)` -> new state + readable event, `CarryOver`.
- `internal/store` - Postgres (pgx) persistence; `RecordEvent` atomically saves state + event. `storetest` gives tests a throwaway schema.
- `internal/content` - Hero/monster/furniture catalogs from `content/` (any `fs.FS`).
- `internal/legacy` - Readers for the original board/quest JSON formats (import only).
- `internal/seed` - Idempotent legacy import. `internal/dotenv` - `.env` loader. `internal/web` - `NoCache` helper, templ views, static assets, TS sources.
- `db/migrations` - goose SQL, embedded via `db.Migrations`.

### Client (`internal/web/src`)
- `board/` - geometry (metrics, hit-testing, footprints), `BoardView` model (derived walls), canvas renderer.
- `maps/` - document types + API client. `editor/` - pure editing model, tools, undo history.
- `tracker/` - session types, view builder, click interaction, event formatting, API client.
- `pages/` - thin DOM wiring per page (`mapEditor.ts`, `tracker.ts`), bundled to `static/dist/`.

### Conventions
- Squares count from **(1,1) at the bottom-left**: x runs 1..width left to right, y runs
  1..height bottom to top. This is also what is stored (board/quest document version 2,
  session state version 2); older top-left documents are rejected, not misread.
- Regions are row-major from the bottom row up (index `(y-1)*width + (x-1)`): `-1` void
  (solid rock), `0` corridor, `>0` room id.
- A vertical edge (x,y) is the left side of tile (x,y); a horizontal edge is its bottom side.
- Furniture and blocked squares are anchored at their bottom-left square and extend right and up.
- Only `board/geometry.ts` functions that take `GridMetrics` deal in screen pixels (y down);
  they are the one place rows are flipped. Legacy `content/` files are top-left, 0-based and
  are converted in `internal/maps/legacy.go`.
- Sizes are columns × rows (width × height); landscape boards have more columns than rows.
- Walls are derived wherever neighbouring regions differ (off-board = void). The GM can also
  draw walls on interior edges (`drawnWalls` on the board, e.g. between two corridors that
  touch); those are the only stored walls. Use `Board.IsWall` / `deriveWalls(..., drawn)` so
  both kinds count.
- A session stores frozen copies of its board and quest, so map edits never change a game in progress.
- Every tracker change is one command -> one event row; corrections are just more events.

## Development Rules
- **Test-first**: write the failing Go test / `bun:test` before the implementation. Keep pure logic
  (models, geometry, commands, formatting) separate from DOM/canvas and HTTP wiring.
- Never hand-edit `*_templ.go`; change the `.templ` file and run `go tool templ generate`.
- `content/` and `assets/` are gitignored (copyrighted HeroQuest material). Tests must not require
  them (content-dependent tests skip without them).
- GM-facing behaviour: never add a rule check that blocks the GM; at most show advice.
