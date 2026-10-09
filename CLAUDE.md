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

### Hosting (branch `online`, docs/ONLINE_AND_RULES_PLAN.md Phase 6)
- `make image` - Build the app image locally (`Dockerfile`); `content/` and `assets/` go in as
  named build contexts, only the folders the app reads. Needs `DOCKER_IMAGE` in `.env`.
- `make push` / `make deploy` - Push the tag to the private Docker Hub repo, then start the
  Render deploy (`RENDER_DEPLOY_HOOK` in `.env`, a secret). The image holds HeroQuest
  material: never push it to GitHub or anywhere public.
- `make hosted-backup` / `make hosted-restore-check [FILE=...]` / `make hosted-restore FILE=...` -
  Back up the hosted database (`HOSTED_DATABASE_URL` in `.env`, a secret) to `db/backups/`,
  practice-restore a backup into a throwaway container, or replace the hosted database (asks
  first). Never copy local data over the hosted database: it holds members and online play.

### Testing and quality
- `make test` - Go tests (database tests skip without a URL)
- `make test-db` - Go tests including database tests (each uses a throwaway schema)
- `make test-race` - Go tests with the race detector
- `bun test` / `make test-js` - TypeScript unit tests (bun:test)
- `make lint` - golangci-lint v2 + ESLint 10 (typescript-eslint) + `tsc` + American-spelling check
  (`bun run spelling`, word lists in `scripts/spelling.ts`)
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
  - Custom hero classes: `/classes` (table `custom_hero_class`): dice stats, accuracy, mana,
    class exclusives and abilities (cooldown in rounds and/or mana cost), and the Three Plagues
    combat stats (hit dice, crit range, damage, avoidance, mitigation, mana per fight round),
    frozen into session heroes as `Combat`. Merged into the catalog's `heroes` as
    `custom-<uuid>` by `catalogFor` too; look classes up there. Classes are never deleted:
    `/classes/{id}/deactivate` and `/reactivate` set `custom_hero_class.active`; a deactivated
    class (`Inactive` in the catalog) is left out of the add-hero picker, and heroes who have
    it keep it.
  - Deletes: every delete or remove button carries `data-confirm="<question>"` (optionally
    `data-confirm-yes`); `confirm.js` (`pages/confirm.ts`, `ui/confirm.ts`) loads on every page
    and opens a Yes/No dialog where only Yes goes ahead. The TS pages call `confirmDialog`.
    Campaigns, sessions and boards (with their quests) are deleted from their pages
    (`internal/app/delete.go`); quests and boards also from the map editor
    (`DELETE /api/boards/{id}?withQuests=true`).
  - Campaign monster stats (column `campaign.monster_stats`): a campaign's own stat line per
    monster type (Body and `content.MonsterCombat`), edited on the campaign page
    (`/campaigns/{id}/monsters...`). Session handlers use `Server.campaignCatalog(ctx,
    campaignID)` (catalogFor with the stat lines laid over), and session monsters freeze them
    as `combat`. A stat line's `abilities` text is shown to the players on the TV card. A
    stat line's `movement` (0 keeps the catalog's) is frozen on session monsters as `movement`.
  - Tracker: `/campaigns`, `/campaigns/{id}`, `/play/{id}`, `/api/campaigns...`, `/api/sessions/{id}/(commands|travel|events|complete|reopen|stream)`
  - Inventory: the party shares one purse (`campaign.gold`, the session's `State.Gold`,
    `gold.set`); campaign heroes carry `items` (name, quantity, notes, kind, stats, equipped)
    between quests. Equipped items add their stats (damage, Accuracy, avoidance, mitigation,
    mana, mana regen) to the hero's class combat stats: totals are computed, never stored
    (`Hero.CombatTotals`, `ManaCap`, `ManaRegen`; TS `combatTotals`, `manaCap`).
    `/campaigns/{id}/gold` and `/campaigns/{id}/heroes/{heroId}/items...` forms, `item.*`
    tracker commands (`item.equip` too). Shared pure helpers in `internal/tracker/inventory.go`.
    The gold box takes `+25`, `-10` or `=40` (a bare number is refused). The campaign page's
    item form also takes `loot_id` (a pick from the loot list: its kind, stats and use, unequipped). Usable items
    (`healBody`, `restoreMana`: potions) are spent by `item.use`, which heals in one event.
  - Loot list (`campaign.loot`, migration 00010; `internal/app/loot.go`): items with their
    kind and stats ready, edited on the campaign page (`/campaigns/{id}/loot...`), served by
    `GET /api/campaigns/{id}/loot` to the tracker's per-hero loot picker
    (`internal/web/src/tracker/loot.ts`). `make fill-campaign` merges combat.json's `loot`
    (matched by name; the GM's own entries stay).
  - Read-aloud script: one Markdown text per campaign (`campaign.script`) parsed by
    `internal/script`; saved with `POST /campaigns/{id}/script`, served parsed by
    `GET /api/campaigns/{id}/script`; the tracker's Read aloud panel logs `passage.read` and
    tags each passage with the letters of the active map's quest notes that hold its text
    (`noteLabels` in `internal/web/src/tracker/script.ts`).
    The Three Plagues script is a folder of numbered part files,
    `docs/campaigns/three-plagues/script/` (`script.Assemble` joins them);
    `make load-script CAMPAIGN="Three Plagues"` saves it to the campaign. Edit one part
    file at a time rather than reading them all. `make narration-text [ONLY=P0,Q1]`
    (`internal/narration`) writes paste-ready text for voicing it into `narration/`.
  - Audio clips for the script: named after passage ids (`Q2-03.mp3`; `Q3-09a` is an extra
    clip of Q3-09), kept in Postgres (table `audio_clip`, `internal/store/audio.go`;
    naming and formats in `internal/audio`); `GET /api/campaigns/{id}/audio`,
    `GET /audio/{campaign}/{file}` (ranges and ETag), upload/delete forms on the campaign
    page. On branch `online`; main still keeps them as files in `AUDIO_DIR/<campaign id>/`,
    which `make import-audio AUDIO_FOLDER=...` copies into a database.
  - Abilities in play: session heroes copy mana and abilities from their class;
    `ability.use` / `ability.reset` track cooldowns (`cooldowns`: ability id -> ready round).
  - Fights (`internal/tracker/fights.go`): `fight.start`/`fight.end` set `State.Fight`. In a
    fight `round.advance` finishes cooldowns, regenerates mana (`Combat.ManaRegen`) and counts
    effects down; `fight.end` drops cooldowns to 1-2 rounds left (0 for a 1-round cooldown), and
    out of a fight they stop there. Generic effects
    (`effect.add`/`effect.remove`, `Effects` on heroes and monsters) only count down and remind.
  - Player screen (TV): `/play/{id}/players` (`pages/players.ts`, `internal/web/src/players/`)
    shows only `tracker.PlayerView` (`internal/tracker/player.go`) from
    `GET /api/sessions/{id}/player` and `/api/sessions/{id}/player-stream`
    (`internal/app/player_api.go`); events carry a player-safe `player_summary` from an
    allow-list, and sightings apart in `player_spotted` (`tracker.PlayerLine` joins them):
    removing a living monster takes its sighting back (a killed one's stays). The GM shows pieces with `seen.set` (doors, furniture, blocked squares,
    monsters); anything new that the players could see must be added to the allow-lists
    there, never sent from the GM state. Plan and decisions: `docs/PLAYER_SCREEN_PLAN.md`.
  - Odds hints (`internal/web/src/tracker/odds.ts`, `ui/odds.ts`): the selected monster's
    panel shows each hero's chance to hit it and attacks to finish it, and its chance to hit
    each hero, from the exact combat math in `internal/web/src/combat/odds.ts`. Advice only.
  - Campaign fill (`internal/campaignfill`, `cmd/fill-campaign`): `make fill-campaign
    CAMPAIGN="Three Plagues" [APPLY=1]` loads `docs/campaigns/three-plagues/combat.json`
    (class stats and abilities, monster stat lines, starting kits, the loot list; the simulator's
    `scripts/combat-config.ts` reads the same file) into the classes, the campaign and its
    heroes. A dry run without `APPLY=1`; run it for real only when the GM asks.
  - Campaign chapters (table `campaign_chapter`): ordered quests, each on its own board;
    `/campaigns/{id}/chapters...` and `/campaigns/{id}/maps` forms, `GET /api/campaigns/{id}/chapters`.
  - Multi-map sessions: the `State` top-level fields are the active map; `OtherMaps` keeps maps
    the party left. `tracker.Travel` swaps them (heroes keep stats; round continues).
- `internal/dice` - Dice expressions (`2d6+1`, d4..d20); mirrored by `internal/web/src/dice/`.
- `internal/combat` (branch `online`) - Strike math ported from `combat/simulate.ts`: seeded
  `Mulberry32`, `Script` (fixed dice), `Log`, `HeroStrike`, `MonsterStrike`. Parity fixtures
  in `internal/{combat,maps}/testdata/parity/` come from `scripts/parity/` (`bun run
  parity:gen` after changing the TS strike math or geometry; a bun test fails when stale).
- `internal/auth` + `internal/app/guard.go`, `signin.go` (branch `online`): sign-in.
  - `AUTH_MODE` none (the default, the table companion, nothing changes), discord, or dev
    (any name, loopback only).
  - Also `PUBLIC_URL`, `DISCORD_CLIENT_ID`/`DISCORD_CLIENT_SECRET`, and `AUTH_ADMINS`
    (`discord:<id>,dev:<name>`).
  - Every app route is registered through the guard, which reads what the route touches from
    its pattern (add new path shapes to `accessRule`, or registration panics).
  - Owners are board, campaign, custom monster and custom class (quests and sessions follow
    theirs).
  - Handlers taking an id from a request body call `mayUseFromRequest`.
  - Membership: only members get past the guard (`AUTH_MEMBERS=approve`, the default: new
    users wait on `/waiting` until an admin lets them in on `/members`; `open` lets everyone
    in). Admin-only routes are listed in `adminRoute`. `/assets/` goes through the guard.
- Online play in the app (branch `online`; how players join, for the GM:
  `docs/ONLINE_PLAY_GUIDE.md`). Who plays a hero is the campaign hero's `userId` (players
  pick free heroes; the GM sets any hero's player on the campaign page, `played_by.go`):
  `/lobby`, `/join/{id}`, the seat page
  `/play/{id}/seat` (`pages/seat.ts`, `seat/model.ts`; `tracker.SeatView`, seat API and
  stream in `internal/app/seat.go`), and the GM's rules console in the tracker
  (`ui/rulesConsole.ts`, `tracker/rules.ts`, from `GET /api/sessions/{id}/actions`). Hero
  action labels and the players' feed name pieces as the players see them, never by id
  (`tracker/names.go`, `playerSafe`). Desktop browsers only; phones aren't supported.
- `internal/maps` geometry for rules (branch `online`): `Terrain` (walls, open doors, blocked
  and sight-blocking squares), `Reachable`/`Path` (orthogonal, deterministic), `LineOfSight`
  (pieces block when given; lenient corners), `VisibleTiles`, `WalkDistances`.
- `internal/maps` - Board and quest documents (Go), validation, advisory `Check`, legacy converters.
- `internal/tracker` - Session `State`, `NewSession`, `Apply(state, command)` -> new state + readable event, `CarryOver`.
- `internal/store` - Postgres (pgx) persistence; `RecordEvent` atomically saves state + event. `storetest` gives tests a throwaway schema.
- `internal/content` - Hero/monster/furniture/trap catalogs from `content/` (any `fs.FS`).
  Monsters and traps may cover several squares (`gridSize`). A quest trap whose kind has no
  `content/traps/` entry is a single-square marker (e.g. `chest`, `teleport`, `other`).
- `internal/legacy` - Readers for the original board/quest JSON formats (import only).
- `internal/seed` - Idempotent legacy import. `internal/dotenv` - `.env` loader. `internal/web` - `NoCache` helper, templ views, static assets, TS sources.
- `db/migrations` - goose SQL, embedded via `db.Migrations`. **One numbering for `main` and
  `online`:** a migration made on `online` is copied to `main` straight away (byte for byte,
  same number), and new ones on either branch take the next free number. Migrations must be
  additive (new tables, nullable or defaulted columns) so the other branch's code can ignore
  them; `main` carries `online`'s 00011-00014 (users, session open, members, audio clips)
  unused until the merge.

### Campaign work in progress
- The Three Plagues campaign (story, script, maps, rules) lives in
  `docs/campaigns/three-plagues/`; progress and next steps are in
  `docs/NARRATIVE_AND_CLASSES_TODO.md` ("Resume here").
- The `online` branch (rules engine, online play, bots) is a separate worktree,
  `../dungeon-campaign-engine-online`, with its own `.env`: container `hq_postgres_online`,
  port 5435, database `hq_online` (a one-way copy of the GM's data), app on :8090. Never
  check out `online` in the main checkout: the server migrates on start and would migrate
  the GM's `hq`. Plan and progress: `docs/ONLINE_AND_RULES_PLAN.md`; online rules:
  `docs/campaigns/three-plagues/ONLINE_RULES.md`.
- The GM's boards, quests and campaigns in the dev database are theirs: don't edit them
  unless asked. To check UI changes in the browser, use a throwaway database (create
  `hq_preview` in the `hq_postgres` container, add a temporary `.claude/launch.json` entry
  with `DATABASE_URL` pointing at it and `APP_PORT=8091`, then restore the file and drop
  the database).

### Client (`internal/web/src`)
- `board/` - geometry (metrics, hit-testing, footprints), `BoardView` model (derived walls), canvas renderer.
- `maps/` - document types + API client. `editor/` - pure editing model, tools, undo history.
- `tracker/` - session types, view builder, click interaction, event formatting, API client.
- `pages/` - thin DOM wiring per page (`mapEditor.ts`, `tracker.ts`), bundled to `static/dist/`.
- Tracker layout: collapsible sections start open or closed per `tracker/panels.ts`
  (`defaultOpen`; hero abilities and inventory, and the right panel's lists, start closed);
  `[` and `]` hide the sidebars (remembered in localStorage); the mode bar keeps a fixed
  height so switching modes never resizes the board. The fight button pulses when
  `tracker/fightHint.ts` suggests starting a fight (a monster comes into sight or loses
  Body, or a hero misses; compared with a "calm" picture kept per session in localStorage)
  or ending one (no living monster in sight). Advice only.

### Conventions
- Squares count from **(1,1) at the bottom-left**: x runs 1..width left to right, y runs
  1..height bottom to top. This is also what is stored (board/quest document version 2,
  session state version 2); older top-left documents are rejected, not misread.
- Regions are row-major from the bottom row up (index `(y-1)*width + (x-1)`): `-1` void
  (solid rock), `0` corridor, `>0` room id.
- A vertical edge (x,y) is the left side of tile (x,y); a horizontal edge is its bottom side.
- Doors may be two edges wide (`span: 2`): the stored edge plus the next one along the wall,
  right for horizontal and up for vertical (`Door.Edges()` in Go, `doorEdges`/`doorCovers` in
  TS; look doors up with `doorCovers`, not by exact edge). Kind `exit` leads off the map and
  is not flagged on the board's edge or against solid rock.
- Furniture, catalog traps, monsters and blocked squares are anchored at their bottom-left square
  and extend right and up. Furniture and traps may be rotated (0/90/180/270); monsters are not.
- Only `board/geometry.ts` functions that take `GridMetrics` deal in screen pixels (y down);
  they are the one place rows are flipped. Legacy `content/` files are top-left, 0-based and
  are converted in `internal/maps/legacy.go`.
- Sizes are columns × rows (width × height); landscape boards have more columns than rows.
- Walls are derived wherever neighboring regions differ (off-board = void). The GM can also
  draw walls on interior edges (`drawnWalls` on the board, e.g. between two corridors that
  touch); those are the only stored walls. Use `Board.IsWall` / `deriveWalls(..., drawn)` so
  both kinds count.
- A session stores frozen copies of its board and quest, so map edits never change a game in progress.
- Every tracker change is one command -> one event row; corrections are just more events.
- Trap state "removed" is live-only (taken off the board during play). Kind "trigger" is a
  built-in marker the GM uses for their own effects; catalog traps with `movable` (the
  boulder) can be moved during play.

## Development Rules
- **Test-first**: write the failing Go test / `bun:test` before the implementation. Keep pure logic
  (models, geometry, commands, formatting) separate from DOM/canvas and HTTP wiring.
- Never hand-edit `*_templ.go`; change the `.templ` file and run `go tool templ generate`.
- `content/` and `assets/` are gitignored (copyrighted HeroQuest material). Tests must not require
  them (content-dependent tests skip without them).
- GM-facing behavior: never add a rule check that blocks the GM; at most show advice.
- **American English** everywhere: code, identifiers, comments, UI text, docs, the campaign
  script and commit messages (color, behavior, center, gray, traveled, organize, toward).
  `make lint` fails on British spellings; mark a line `spelling:allow` only for outside data
  that must keep its spelling. Proper nouns (e.g. Greyford) are left as named.
