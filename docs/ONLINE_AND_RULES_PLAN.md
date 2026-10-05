# Online play, rules engine and bots plan

**Last Updated**: 2026-10-05 15:40 EDT
**Branch**: `online` (its own worktree, `../dungeon-campaign-engine-online`)

## Goal
The end goal is reinforcement-learning bots that play whole Three Plagues quests headlessly,
to find where the content is too hard, too easy or broken. Along the way the app grows:
- **online play**: the GM starts a session, friends sign in from their own browsers, and each
  player takes one turn per round, in any order, then the GM or monster phase;
- **an automated GM** that moves monsters, reveals, triggers traps, hands out loot and
  triggers narration;
- **content made by friends** (maps, quests, campaigns, items, monsters);
- far off, **generated campaigns**.

All of these need the same core first: a rules engine that knows turns, movement, line of
sight and dice, rolls the dice itself, and can list the legal actions.

Table mode stays as it is today. A session with no rules (`State.Rules == nil`) records only,
and even in rules mode the GM can override anything. Rules only gate commands from player
seats and bots.

## Decisions (GM, 2026-10-05)
- **The engine is Go.** It extends `internal/tracker`, as the single authority. The strike
  math in `internal/web/src/combat/simulate.ts` is ported to Go and parity-tested against
  TS.
- **Order:** the online track first (the rules core, then auth, seats, the player client and
  hosting), then monster AI and the auto-GM, then the headless sim and scripted bots, then RL.
- **RL stack:**
  - Python with Gymnasium drives a Go sim binary over JSON lines on stdin/stdout.
  - Training uses MaskablePPO (sb3-contrib), masked by the engine's legal actions.
- **Sign-in method:** decided when Phase 3 starts.
- **Isolation:** the branch lives in its own git worktree, with its own Postgres container
  (`hq_postgres_online`, port 5435, database `hq_online`), app port 8090 and templ proxy
  port 7341, all set in the worktree's `.env`.
  - The main checkout and the GM's `hq` database are never touched; switching branches in place
    would run the branch's migrations on `hq`, because the server migrates on start.
  - The GM's data is copied one way (a `pg_dump` from `hq_postgres` restored into
    `hq_postgres_online`), whenever a fresh copy is wanted.
  - `content/` and `assets/` are symlinks to the main checkout, read only.
- **Ruleset:** the engine targets Three Plagues rules, not the original game's. Other rulesets
  can come later as more configurations.
- **Online rules D1-D12:** see `docs/campaigns/three-plagues/ONLINE_RULES.md`.

## Phases
Each phase lands in a few sessions, is test-first, and records its commits here.

1. **Foundations in Go: geometry and combat (pure, parity-tested).** No `State` change.
   - **1a. Geometry** (`internal/maps`):
     - `adjacent.go`: `EdgeBetween`, `EdgeTiles`, `Neighbors4`;
     - `footprint.go`: `FootprintTiles`, `DoorCovers`;
     - `terrain.go`: `Terrain` (walls, open doors, blocked squares, furniture `BlocksMovement` /
       `BlocksLineOfSight`), `Passable`, `Open`;
     - `path.go`: `Reachable` (Dijkstra with a budget and occupancy), `Path` (deterministic
       tie-breaks), `WalkDistances`;
     - `sight.go`: `LineOfSight` with an `Attack` or `View` mode (D6, D7), `VisibleTiles`.
   - **1b. Combat** (new `internal/combat`, does not import the tracker):
     - `roller.go`: the `Roller` interface, `Mulberry32` (an exact port of `seededRoller`), and
       `Script` (a fixed die sequence);
     - `strike.go`: `HeroStrike`, `MonsterStrike`, `FalterAvoidance`, drawing dice in the same
       order as TS.
   - **1c. Parity:** `scripts/parity/gen.ts` (`bun run parity:gen`) writes
     `testdata/parity/*.json`; Go tests read the fixtures, and a bun test fails if the
     committed fixtures go stale.
   - **Exit:** the parity fixtures pass in both languages, and `make lint` and `make test`
     pass.
2. **Rules engine core** (`internal/tracker`):
   - **Rules state:** `State.Rules *RulesState`, holding:
     - the ruleset, its config and the RNG state;
     - the phase (heroes, monsters, over) and who has acted;
     - the active turn and the critical-miss skips;
     - a pending reaction window (D10) and what has been searched (D9);
     - the objectives and the outcome.
   - **Dice:** a seeded RNG lives in the state, so `Apply` stays pure. Every die drawn is logged
     in the event payload, and replaying the commands reproduces the game. GM commands may
     carry real dice (`dice: [...]`).
   - **Actor:** `Command.Actor{Kind gm|seat|bot|autogm, HeroID}`. The zero value is the GM, and
     the HTTP layer always sets it, never the client.
   - **Commands:**
     - GM: `rules.enable` and `rules.disable`;
     - seat: `turn.start`, `turn.roll-move`, `turn.move`, `turn.door`, `turn.attack`,
       `turn.ability`, `turn.search {kind: treasure|traps|doors}`, `turn.disarm`, `turn.end`,
       `reaction.use` and `reaction.pass`;
     - GM or autogm: `phase.monsters`, `monster.act`, `phase.end` (reusing the `round.advance`
       bookkeeping), `quest.end`.
   - **Legal actions:** `LegalActions(s, actor, catalog)` and `checkLegal` share their
     predicates. A property test checks that every listed action applies, and that unlisted
     seat commands are refused.
   - **Objectives:** an additive quest field (kill, escape, find, survive), with `CheckOutcome`
     after each rules-mode command.
   - **Events:** readable, e.g. "Bram attacks the Orc: 11+3 vs 12, hit for 9." The payload is
     `{command, actor, rolls, result}`.
   - **Exit:** a whole fight can be played through `Apply` with seat actors from a fixture
     state, and replayed from its seed.
3. **Auth, users, ownership:**
   - `internal/auth` behind an `Authenticator` interface;
   - `app_user` and `user_session` tables, and `owner_id` on campaigns, boards, quests, custom
     monsters and classes;
   - `AUTH_MODE=none` keeps local play as it is.
   - This is the branch's first migration; renumber it after any new ones on main.
   - **Exit:** write routes are owner-checked, with tests for 401 and 403.
4. **Seats, lobby, session lifecycle:**
   - a `session_seat` table (session, hero, user, invite token) and `/join/{token}`;
   - a seat command endpoint that sets the seat actor;
   - lobby, then active, then completed;
   - presence over the existing stream hub.
   - **Exit:** two browsers each claim a hero and play a turn.
5. **Online player client:**
   - `tracker.SeatView`: the player view, plus the seat's own hero, plus its legal actions;
   - a per-seat stream;
   - `pages/seat.ts`, with buttons driven by legal actions and move highlighting;
   - a GM rules console (phase, pending items, the monsters' turn).
   - Ability effects are resolved by the GM in this phase.
   - **Exit:** a whole quest is played online.
6. **Hosting and the content boundary:**
   - a multi-stage Dockerfile running a single instance, since session locks are in memory;
   - Postgres, Caddy for TLS, and backups (a small VPS with docker compose is the likely fit);
   - `CONTENT_MODE=online`: no `content/` or `assets/` are mounted, and pieces are drawn as
     shapes;
   - invite-only, shipping only original Three Plagues text and data;
   - legal advice before anything public.
   - **Exit:** the instance is deployed, and `/assets/` returns 404 there.
7. **Structured rules data:**
   - ability effects (ported from `simulate.ts`), trap effects, search rewards and reactions
     become data plus Go;
   - parity extends to abilities.
   - **Exit:** nothing in the Three Plagues kit still needs the GM to resolve it.
8. **Monster AI** (`internal/ai/monster`):
   - plans the whole monster phase under the tactical behavior (D12):
     - the order;
     - attacking before or after moving;
     - focusing the weakest or most exposed hero;
     - surrounding;
     - the attack-and-step-away rotation (D11);
     - archers keeping range;
   - a scored search over candidate plans, using the state's RNG;
   - per-type profiles written here.
   - **Exit:** a whole monster phase is generated, with golden replays.
9. **Auto-GM** (`internal/autogm`):
   - commands sent as the `autogm` actor: phases, monster turns, traps, loot, and `passage.read`
     triggers linked to rooms;
   - the GM can take over at any time.
   - **Exit:** a quest runs with no GM input.
10. **Headless sim and scripted bots:**
    - `cmd/sim`, a JSON-lines `reset` / `step` protocol;
    - scenarios exported from the database, so no `content/` is needed;
    - `internal/bots`;
    - `cmd/sim-batch` reports: win rate, deaths, rounds, class performance;
    - a statistical comparison with TS `runQuest`.
    - Balance is re-checked against the tactical monster AI.
    - **Exit:** a report from 10,000 Quest 1 runs.
11. **RL** (`python/dce_gym`):
    - a Gymnasium env over `cmd/sim`, masked from legal actions, trained with MaskablePPO;
    - a curriculum: single-room fights first, then Quest 1;
    - a report on balance gaps, comparing learned and scripted play.
12. **Content sharing:** packs with owners and visibility, validated by `Quest.Check`.
13. **Generated campaigns:** packs generated against the same validators.

## Resume here
- **Where things stand:** the worktree, the `hq_online` database (a copy of the GM's data) and
  the plan are set up. Phase 1 has not started.
- **Next:** Phase 1a, the geometry, test-first.

## Running log
- 2026-10-05: plan written. The online worktree has its own container, ports and `.env`.
  `docker-compose.yml` and the `Makefile` read the container name and ports from `.env`, with
  main's defaults.
