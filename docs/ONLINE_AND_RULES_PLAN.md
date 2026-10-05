# Online play, rules engine and bots plan

**Last Updated**: 2026-10-05 19:09 EDT
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
- **Sign-in (2026-10-05):**
  - Discord first (OAuth2, "identify" scope: Discord ID, username, avatar; no email).
  - Invite links decide who gets a seat. Signing in only says who someone is.
  - Logins are stored as `user_identity` (provider + subject) linked to `app_user`, so Google
    or passkeys can be added later without a migration.
  - Rolled-our-own passwords are ruled out.
  - The table keeps `AUTH_MODE=none`, and a dev login covers local testing and bots.
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
   - **Done 2026-10-05.**
     - Geometry (`5f564e8`): `internal/maps/{adjacent,footprint,terrain,path,sight}.go`.
     - Combat and parity (`f446ed7`): `internal/combat/{roller,strike}.go`, the
       `scripts/parity/` generator, and fixtures in `internal/{combat,maps}/testdata/parity/`.
       There are 1,220 hero and 520 monster strike cases, each replayed from its dice and from
       its seed. A mutation check confirmed that changing the hit rule or ignoring blocked
       squares fails them.
     - Moved to Phase 2: building attackers from session heroes and monsters (it belongs in
       `internal/tracker`, which imports `internal/combat`, not the other way round) and
       Faltering (it's applied to the Avoidance passed in).
2. **Rules engine core** (`internal/tracker`):
   - **Rules state:** `State.Rules *RulesState`, holding:
     - the ruleset, its config and the RNG state;
     - the phase (heroes, monsters, over) and who has acted;
     - the active turn and the critical-miss skips;
     - a pending reaction window (D10) and the furniture already searched for treasure
       (once per party, D9);
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
   - **Progress (2026-10-05):**
     - **2a, rules state and turns** (`5b9da83`): `rules.enable`/`rules.disable`, `Command.Actor`
       and `Command.Dice`, `turn.start`/`turn.roll-move`/`turn.end`, `phase.monsters`/`phase.end`.
       Rolls are logged in event payloads, and fights follow revealed monsters.
     - **2b, movement** (`703cc7b`): `turn.move` (a square or a checked path) and `turn.door`,
       view reveals after every step, traps that stop the move.
     - **Class reach** (`6a65785`): the class form, combat.json and `make fill-campaign`.
     - **2c, attacks** (`44113ba`): `turn.attack`, `monster.attack`, `monster.move`.
     - **Monster reach ruling** (`5fbe435`).
     - **2d, searching, disarming, outcomes** (`bdaf658`): `turn.search`, `turn.disarm`,
       `quest.end`, quest `objectives`.
     - **GM rulings on furniture, disarming and objectives** (`8d94651`): all furniture blocks
       movement; disarming steps onto a floor trap; quest `goal`, kill/collect/escape
       objectives, `turn.exit`, door keys.
     - **2e, legal actions** (`1c59df1`): `LegalActions`, sharing its checks with the commands.
       A property test plays seeded random games and holds the list to `Apply` in both
       directions. A replay test confirms a game repeats from its seed.
   - **Done 2026-10-05.**
3. **Auth, users, ownership:**
   - `internal/auth` behind an `Authenticator` interface;
   - `app_user` and `user_session` tables, and `owner_id` on campaigns, boards, quests, custom
     monsters and classes;
   - `AUTH_MODE=none` keeps local play as it is.
   - This is the branch's first migration; renumber it after any new ones on main.
   - **Exit:** write routes are owner-checked, with tests for 401 and 403.
   - **Done 2026-10-05.**
     - Store (`f17788e`): migration 00011, users, identities, login sessions, owners; a
       `store.Viewer` in the context makes new rows the viewer's and keeps lists to their own.
     - `internal/auth` (`a31184f`): the settings, Sign in with Discord (hand-written OAuth2,
       identify scope), tokens.
     - App (`7ed4f8d`): every route is registered through `guard.go`, whose rule comes from the
       route's pattern (owner or admin; a route without a rule panics). The sign-in routes,
       Sign out in the nav, and cross-origin protection.
     - **Real Discord sign-in works:** the GM signed in from Chrome on the worktree (port 8090)
       and is an admin through `AUTH_ADMINS`.
     - Reads are private too (scripts and maps are spoilers). Seats in Phase 4 will open
       sessions to their players.
4. **Seats, lobby, session lifecycle:**
   - a `session_seat` table (session, hero, user, invite token) and `/join/{token}`;
   - a seat command endpoint that sets the seat actor;
   - lobby, then active, then completed;
   - presence over the existing stream hub.
   - **Exit:** two browsers each claim a hero and play a turn.
   - **Decisions (GM, 2026-10-05):**
     - **A lobby, not invite links:** players sign in, see the games opened to them, pick a free
       hero or make their own, and join.
     - A player may play several heroes.
     - The GM starts online play; late joins are fine.
   - **Built differently from the sketch:** no `session_seat` table. A campaign hero's
     `userId` says who plays it (kept across GM saves), and a seat is every hero of yours in
     the session.
   - **Done 2026-10-05.**
     - `f7c45db`: the open flag, `hero.join`, and hero players.
     - `e3b3dd4`: the lobby, join page, seat API, the GM's Open/Start controls, and the
       "seated" guard rule.
     - `81c436e`: a quest-list leak found in the browser check.
     - Exit test: `TestFriendsJoinFromTheLobbyAndPlayATurn`.
   - **Moved to Phase 5:**
     - presence (who is connected; it belongs with the per-seat stream);
     - the player's own game screen;
     - the hero sheet in the seat view.
   - **Before hosting publicly (Phase 6):** a members allowlist, since any Discord account
     could join an open game.
5. **Online player client:**
   - `tracker.SeatView`: the player view, plus the seat's own hero, plus its legal actions;
   - a per-seat stream;
   - `pages/seat.ts`, with buttons driven by legal actions and move highlighting;
   - a GM rules console (phase, pending items, the monsters' turn).
   - Ability effects are resolved by the GM in this phase.
   - **Exit:** a whole quest is played online.
   - **Progress (2026-10-05):**
     - **5a** (`a3c9caf`):
       - `tracker.SeatView` (the seat's hero sheets and legal actions, the phase, whose turn).
       - The seat API returns the player view, feed and catalog, plus the seat and presence.
       - The `seat-stream` WebSocket pushes each player their own seat after every change, and
         presence. The GM tracker shows "Online: ..."; `tracker/stream.ts` tells changes from
         presence.
     - **5b** (`569d23b`):
       - `/play/{id}/seat` and `pages/seat.ts`: the board with reachable squares highlighted
         (click to move or attack), the hero sheet, action buttons, party, presence, and a
         "What happened" feed with dice.
       - `seat/model.ts` (tested).
       - The players' feed hears rules commands, sanitized in `tracker/player.go`
         `playerSafe`: no monster ids, no quest notes, nothing from unseen monsters.
     - **Browser check** (`43f3384`): a pretend player claimed two heroes, took turns, opened
       a door, moved through an ally and attacked by clicking the board. Fixed what it found:
       - Hero action labels name pieces as the players see them, with the way to them
         ("Attack the Goblin to the east"), never by id (`tracker/names.go`).
       - `playerSafe` drops door, furniture and trap ids and the GM's "(seen: ...)" counts.
       - The seat page stacks at phone width (board, then turn and feed, then sheet).
     - **Not yet done:**
       - **5c:** the GM rules console.
6. **Hosting and the content boundary:**
   - a multi-stage Dockerfile running a single instance, since session locks are in memory;
   - Postgres, Caddy for TLS, and backups (a small VPS with docker compose is the likely fit);
   - `CONTENT_MODE=online`: no `content/` or `assets/` are mounted, and pieces are drawn as
     shapes;
   - invite-only, shipping only original Three Plagues text and data;
   - legal advice before anything public.
   - **Hosting notes (GM, 2026-10-05):**
     - The GM owns `kostant.dev`. Candidates: a subdomain such as `dce.kostant.dev`, with
       `PUBLIC_URL` set to it and a second Discord redirect URL added in the portal.
     - Containerize the whole app. Candidates: Fly.io, Render or Railway (managed Postgres,
       WebSockets, a Dockerfile deploy), Heroku's container stack, or a small VPS (e.g.
       Hetzner) running docker compose (app, Postgres, Caddy for TLS).
     - Needs: a single app instance (the session locks are in memory), Postgres 18, persistent
       storage for the audio clips (`AUDIO_DIR`), backups, and the members allowlist before going public.
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
- **Where things stand (2026-10-05, end of a long session):**
  - Phases 1-4 are done, and Phase 5 is through 5b plus the seat page's browser check. All
    committed on `online`; the last code commit is `43f3384`.
  - Tests: 406 Go (`make test-db`) and 336 bun, with `make lint` clean.
  - The worktree's `.env` has `AUTH_MODE=discord` and `AUTH_ADMINS=discord:<the GM's id>`.
  - Main and the GM's `hq` database were never touched. Main's `.claude/launch.json` is
    restored.
- **Next, in order:**
  1. **5c, the GM rules console:**
     - A `ui/rulesConsole.ts` panel in the tracker's right column, above `renderSelection`.
     - It shows the phase, whose turn, and "The monsters' turn" / "End the round" buttons.
     - It lists the monsters' moves and attacks from `LegalActions` for the GM. That needs a
       new GM endpoint returning `tracker.LegalActions(state, Actor{}, cat)`, since the
       rules live only in Go.
     - Pure logic goes in `tracker/rules.ts` with tests. `presence` is already in
       `pages/tracker.ts`.
  2. **The Phase 5 exit:** play a whole small quest online (GM tracker plus two seat tabs).
     The classes in `hq_online` were copied before class reach existed, so every hero has
     adjacent reach there until `make fill-campaign CAMPAIGN="Three Plagues" APPLY=1` runs
     against the worktree's database (ask the GM first). The catalog classes (Barbarian,
     Elf...) have no combat stats, so their heroes can't attack in rules mode.
  3. **Phase 6, hosting:** see its hosting notes (`kostant.dev`, containers, a members
     allowlist).
- **Open decisions:** none pending.
- **Devices (GM, 2026-10-05):** desktop and laptop browsers only; phones aren't supported, so
  browser checks skip phone widths.
- **Browser checks** (the dev-mode procedure):
  1. Set `AUTH_MODE=dev` in the worktree's `.env`, `make build`, and add the temporary
     `dce-online` launch entry (below).
  2. Make the game as a pretend GM with a script (Python `urllib` with a cookie jar):
     `POST /auth/dev` (form `name=zz GM`), then a `zz` board, quest, campaign and session,
     `POST /play/{id}/open` and `/play/{id}/start`. GM commands go to
     `/api/sessions/{id}/commands`.
  3. Sign in as a pretend player in the browser pane at `127.0.0.1:8090` (its cookies are
     apart from `localhost`, so the GM and player don't share a sign-in), then join from
     the lobby.
  4. For attacks the heroes need classes with combat stats: copy rows into
     `custom_hero_class` owned by the pretend GM, and set the campaign's `monster_stats`.
  5. Clean up: delete the `zz` rows (session events, sessions, campaign, quests, boards,
     classes) and the `provider = 'dev'` users, then set `AUTH_MODE=discord` again.
- **Launch entry:** add a temporary `dce-online` entry to the main checkout's
  `.claude/launch.json` (`bash -c "cd ../dungeon-campaign-engine-online && exec
  ./build/dungeon-campaign-engine"`, port 8090) after `make build` in the worktree. Restore the
  file afterward. The GM signs in with Discord from their own browser at
  http://localhost:8090; the in-app browser pane is often hidden.

## Running log
- 2026-10-05: plan written. The online worktree has its own container, ports and `.env`.
  `docker-compose.yml` and the `Makefile` read the container name and ports from `.env`, with
  main's defaults.
- 2026-10-05: Phase 1 done.
  - `5f564e8`: geometry.
  - `f446ed7`: combat port and parity fixtures. Run `bun run parity:gen` after changing the TS
    strike math or the geometry.
- 2026-10-05: the GM settled the last two D9 points. No search of any kind while a monster is
  revealed (combat is active), and treasure is searched once per piece of furniture per party.
- 2026-10-05: Phase 2a-2c.
  - `5b9da83`: rules state, actors, turns, seeded dice.
  - `703cc7b`: movement, doors, reveals.
  - `6a65785`: class reach.
  - `44113ba`: attacks.
  - Ranged attacks have no range limit (GM). The GM endpoint always stamps the GM as the
    actor.
- 2026-10-05: Phase 2d.
  - `5fbe435`: monster reach is 2 squares in a straight line and never diagonal (GM).
  - `bdaf658`: searching, disarming, objectives and outcomes.
- 2026-10-05: `8d94651`, the GM's rulings.
  - All furniture blocks movement, and treasure is searched from beside a piece.
  - A floor trap is disarmed by stepping onto it.
  - Quests get a goal and kill/collect/escape objectives. Without objectives, clearing every
    monster completes the quest.
  - Door keys.
- 2026-10-05: Phase 2 done with 2e (`1c59df1`), `LegalActions` and its property and replay
  tests.
- 2026-10-05: `56513db`, map editor fields for the goal, objectives and door keys. Checked in
  the browser on a throwaway board in `hq_online`, then deleted.
- 2026-10-05: the GM chose Discord for sign-in (trade-offs weighed against Google, an email
  link, passkeys and passwords). Invite links gate seats.
- 2026-10-05: Phase 3 done.
  - `f17788e`: store.
  - `a31184f`: auth.
  - `7ed4f8d`: app guard and sign-in.
  - Real Discord sign-in checked by the GM.
- 2026-10-05: Phase 4 done.
  - `f7c45db`, `e3b3dd4`, `81c436e`.
  - A browser check as a pretend GM and player in dev mode, cleaned up afterward. It found
    and fixed a quest-list leak.
- 2026-10-05: Phase 5a and 5b.
- 2026-10-05: `43f3384`, the seat page's browser check and its fixes: player wording for
  actions and the feed, and a phone layout.
  - `a3c9caf`: seats, the seat stream, presence.
  - `569d23b`: the seat page and player lines for rules commands.
  - The seat page's browser check is still to do.
  - The GM noted the `kostant.dev` domain for hosting.
