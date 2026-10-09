# Online play, rules engine and bots plan

**Last Updated**: 2026-10-08 20:38 EDT
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
     - **5c** (`9ddc831`): the GM rules console in the tracker's right column.
       - `GET /api/sessions/{id}/actions` (GM only) gives `tracker.LegalActions` for the GM;
         `tracker/rules.ts` (tested) sorts them for `ui/rulesConsole.ts`.
       - It shows where the round stands, "The monsters' turn" / "End the round", and an
         "End <hero>'s turn" override. In the monsters' phase each monster lists its attacks;
         selecting one highlights its moves, and clicking a highlighted square moves it by
         the rules.
       - The header's "Next round" (`round.advance`) in rules mode starts the next heroes'
         phase and closes an open turn (it had left the rules state in the old round).
       - Checked in the browser with a GM tab and a seat tab: live updates both ways, a
         goblin moved into a doorway and attacked, rounds ended both ways.
     - **Not yet done:** the Phase 5 exit.
6. **Hosting** (decisions by the GM, 2026-10-05):
   - **Host: Render** (a web service from a prebuilt image, plus Render Postgres 18). The image
     is built on the GM's machine and pushed to a **private Docker Hub repository** (the free
     plan's one private repo; each version is a tag). Render pulls it with a read-only Docker
     Hub token and deploys on a deploy hook (`make deploy`).
     - The database must be a **paid** instance (free Render Postgres is deleted after 30
       days); paid instances and the Starter web service never sleep, so a month between
       sessions is fine. The 3-day point-in-time restore of a Hobby workspace is enough, with
       our own dumps downloaded to the GM's machine.
     - Rough cost: web $7 + database about $6-10 (check the price when creating it).
     - Weighed and set aside: DigitalOcean (App Platform, its registry, Managed Postgres;
       about $32), Fly.io (no managed Postgres 18), Railway (private images need Pro; no
       managed Postgres), Heroku (frozen since February 2026; no disks), a VPS (we would run
       the OS, Postgres and backups), Supabase (no Postgres 18).
   - **Content: the image carries `content/` and `assets/`.** The Dockerfile in git has no
     content: `make image` passes them from the main checkout as named build contexts. The
     image never goes to GitHub, and the site is for the GM's group only: every page,
     `/assets/` included, needs a signed-in, approved member. (`CONTENT_MODE=online`, an
     image without HeroQuest material, is dropped for now; it comes back before anything
     public, with legal advice first.)
   - **Members: approved on first sign-in.** Anyone may sign in with Discord but waits on
     "Waiting for the GM" until an admin approves them on an admin page; admins
     (`AUTH_ADMINS`) are members already.
   - **Audio clips move into Postgres**, so no disk is needed and one backup covers all.
   - Postgres 18 is required: the schema uses `uuidv7()`.
   - **Steps:**
     1. Dockerfile (multi-stage, non-root, static files and migrations inside), `make image`
        and `make deploy`.
     2. Members: an approval table, the waiting page, the admin page; `/assets/` behind the
        guard.
     3. Audio in Postgres.
     4. Hosting housekeeping: `/healthz`, WebSocket keepalive pings (proxies close idle
        sockets), `PORT` as well as `APP_PORT`.
     5. Render setup (the GM creates the accounts, service and database; we write the
        settings and commands): `dce.kostant.dev`, `PUBLIC_URL`, the second Discord
        redirect, `AUTH_MODE=discord`, `AUTH_ADMINS`.
     6. The GM's data: a one-way copy of the Three Plagues campaign, boards and quests,
        owned by the GM's Discord user.
     7. Backups: `make` targets to download a dump and to restore one; one practice restore.
   - **Progress:**
     - **Step 1** (`7c76a44`): `Dockerfile`, `.dockerignore` (an allowlist), `make image`,
       `make push` (refuses a dirty tree) and `make deploy`. The image takes only the four
       catalog folders and `assets/tiles_cleaned` (15 MB). The amd64 image ran against
       `hq_online`. `.env` needs `DOCKER_IMAGE` (the private Docker Hub repo) and, once the
       Render service exists, `RENDER_DEPLOY_HOOK`. `2b1424a`: `make push` refuses a
       repository Docker Hub shows publicly (it creates a missing one on the first push,
       possibly public), checking before and after the push. The GM set `DOCKER_IMAGE` and
       ran `docker login`, then created the private repository
       `kostant/dungeon-campaign-engine`. First image pushed: tag `fb02146` (each tag is
       the commit it was built from).
     - **Step 2** (`59b3981`): members approved on first sign-in. `/waiting` for those not
       let in (or refused), `/members` for admins (Let in, Refuse, Remove), "Members (n)"
       in an admin's nav, `AUTH_MEMBERS=approve|open` (approve by default), migration 00013
       (`app_user.member_status`; earlier users start as members), admins approved at
       sign-in, and `/assets/` served through the guard. Checked in the browser.
     - **Step 3** (`aa2bac4`): audio clips in Postgres (migration 00014 `audio_clip`), served
       with ranges and an ETag; `AUDIO_DIR` is gone from the server. `make import-audio
       AUDIO_FOLDER=... [DB=...]` copies main's `AUDIO_DIR` folder into a database,
       skipping unchanged clips (campaign ids match because the databases are copies). The
       GM will record clips with ElevenLabs from the script text: upload them on main for
       game night, then import them into `hq_online` or the hosted database.
     - **Step 4** (`f52714f`): `GET /healthz` (no sign-in; 503 without the database), keepalive
       pings every 25 s on all three streams (one `listen` loop), and `PORT` before
       `APP_PORT` (the image sets `PORT=8080`). Checked with the image on `PORT=10000`.
     - **Step 5** (GM, 2026-10-05): Render is set up. Postgres 18 (`dce`, user `dce_ks_admin`,
       Virginia, $6 plan with 5 GB), the Starter web service from
       `docker.io/kostant/dungeon-campaign-engine` with a read-only Docker Hub token, health
       check `/healthz`, `dce.kostant.dev` (behind Cloudflare DNS), the second Discord
       redirect. The service's variables are in the worktree's gitignored `.env.render`;
       `RENDER_DEPLOY_HOOK` and `HOSTED_DATABASE_URL` (the External URL) are in its `.env`.
       Checked: `/healthz` ok, Discord sign-in works, signed-out visitors are sent to sign-in
       for pages, `/members` and `/assets/`, and the API answers 401.
     - **Step 6** (2026-10-05): the GM's data is on the hosted database, copied from main's
       `hq` (the GM's choice: it had 3 session events newer than `hq_online`):
       1. `pg_dump -Fc --no-owner --no-privileges` of `hq` (read-only).
       2. On the hosted database: `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`, then
          `pg_restore --no-owner --no-privileges` (run from the `hq_postgres_online`
          container with the URL passed through the environment, never printed).
       3. A redeploy (the deploy hook) migrated it from 10 to 14.
       4. `fill-campaign -apply` against it added class reach (nothing else changed).
       5. The GM's Discord user was made owner of the boards, campaigns and classes.
       - Counts matched `hq`: 4 campaigns, 5 boards, 4 quests, 3 sessions, 133 events.
       - **Don't repeat this now that the site is in use:** the copy replaces everything,
         including members, online sessions and audio clips made there. From here on, data
         flows to the hosted database by hand (`make import-audio ... DB=...`, or the app).
     - **Step 7** (`24b726c`): backups. `make hosted-backup` (a dump plus row counts in
       `db/backups/`), `make hosted-restore-check [FILE=...]` (restores into a throwaway
       Postgres 18 container and compares counts), `make hosted-restore FILE=...` (replaces
       the hosted database after typing "replace"; Render's 3-day point-in-time restore
       first). Practiced: the live database's first backup restored with every count
       matching. Take a backup after each session.
   - **Phase 6 done (2026-10-05):** live at `dce.kostant.dev`; anyone not approved is refused
     everywhere, `/assets/` included.
   - **Exit:** live at `dce.kostant.dev`; anyone not approved is refused everywhere,
     `/assets/` included.
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

## Merging back into main
Nothing here is urgent: `online` merges into main only when the GM wants the online features
at the table. Do these at that merge (and keep the list current as the branches drift):
- **Audio clips (GM, 2026-10-05).** Until the merge, **main is the source of truth for clips**:
  the GM records them with ElevenLabs and uploads them on main (files in main's
  `AUDIO_DIR/<campaign id>/`), then copies them to this branch with `make import-audio
  AUDIO_FOLDER=../dungeon-campaign-engine/audio` (unchanged clips are skipped; a clip in the
  folder replaces one with the same id in the database, so don't upload clips straight
  into `online`'s campaign page meanwhile).
  - After the merge main's server no longer reads `AUDIO_DIR`, so the clips must be in `hq`
    before the next game night. Plan: at merge time, have the server import `AUDIO_DIR` on
    start when the folder exists (the same skip-unchanged import, logged), so nothing is
    lost even if the step is forgotten; or run `make import-audio` against `hq` right after
    the merge. Check the reader plays them, then the folder can be archived.
  - Main's `Library.RemoveCampaign` (a deleted campaign's clip folder) has no counterpart
    here: `audio_clip` rows go with their campaign.
  - Code: main's file-based `internal/audio` and `internal/app/audio.go` give way to
    `online`'s; anything main adds to audio before then is redone on the database version.
- **Migrations: nothing to do.** Since 2026-10-08 both branches share one numbering: `main`
  carries `online`'s 00011-00014 byte for byte (applied to `hq` then, unused by main's code),
  and new migrations on either branch take the next free number (00015,
  `custom_hero_class.active`, came from `main`). A migration made here is copied to `main`
  straight away, and must be additive so main's code can ignore it.
- **Table mode.** Rules mode is opt-in per session (`State.Rules == nil` is the table game)
  and `AUTH_MODE` unset keeps sign-in off, so the table companion should behave as before:
  play a quick table session after the merge, and add the table-mode golden test (Phase 1c)
  if it still isn't written.
- **Merge main into `online` regularly**, so the final merge stays small. Last done
  2026-10-08 (`5ecd9ee`: the shared migration numbering, class deactivation (00015) and
  deletes behind a Yes/No dialog; conflicts in the class, catalog, campaign and script
  code, and the guard now checks a session delete against the session's owner). Before that
  2026-10-07 (`bc06ee9`: table testing fixes, monster movement, hero sheets, housekeeping;
  the tracker page's imports conflicted, and both branches had added `Monster.Movement`,
  now one field), then `2493d61` (the ESLint fix for `docs/**/*.js`). Before that
  2026-10-06 (`9cafaa6`).

## Resume here
- **Where things stand (2026-10-05, end of a long session):**
  - Phases 1-4 and 6 are done (the site is live at `dce.kostant.dev`), and Phase 5 is built
    through 5c; only its exit (a whole quest online) is left. All committed on `online`; the last code commit is `9ddc831`.
  - Tests: 408 Go (`make test-db`) and 340 bun, with `make lint` clean.
  - The worktree's `.env` has `AUTH_MODE=discord` and `AUTH_ADMINS=discord:<the GM's id>`.
  - Main and the GM's `hq` database were never touched. Main's `.claude/launch.json` is
    restored.
- **Next, in order:**
  1. **The Phase 5 exit, now on the hosted site:** play a whole small quest online (the GM
     tracker plus friends' seats),
     ideally with the GM and friends signed in with Discord. The GM started Phase 6 first,
     so this can be played on the hosted site once it is up. The GM ran `make fill-campaign`
     on `hq_online`, so its Three Plagues classes have their reach. The catalog classes
     (Barbarian, Elf...) have no combat stats, so their heroes can't attack in rules mode.
     Known gaps to expect: ability effects, trap effects and search rewards are resolved by
     the GM (Phase 7); no reaction prompts yet (D10).
  2. **Phase 6, hosting: done** (Render, a private Docker Hub image, members approved on
     first sign-in, audio in Postgres): live at `dce.kostant.dev` with the GM's data and
     backups. Deploy changes with `make deploy`; back up with `make hosted-backup`.
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
- 2026-10-05: the GM ran `make fill-campaign` on `hq_online` (class reach now there).
  `9ddc831`, Phase 5c: the GM rules console; "Next round" ends the round by the rules.
- 2026-10-05: Phase 6 decided: Render with a private Docker Hub image that carries the
  content, members approved on first sign-in, audio in Postgres. `7c76a44`, step 1: the
  image; `2b1424a`, the public-repository guard; `59b3981`, step 2: members; `aa2bac4`,
  step 3: audio in Postgres; `f52714f`, step 4: health check, pings, `PORT`. Steps 5 (Render set up by the GM) and 6 (the GM's data
  copied from `hq`) done the same evening; the site is live. `24b726c`, step 7: backups. Phase 6
  done.
  - `a3c9caf`: seats, the seat stream, presence.
  - `569d23b`: the seat page and player lines for rules commands.
  - The seat page's browser check is still to do.
  - The GM noted the `kostant.dev` domain for hosting.
- 2026-10-06: the GM's first ElevenLabs clips (P0-01 to P0-04, Q1-01 to Q1-03, Q1-N1 to
  Q1-N3; 27 MB) imported from main's `audio/` into `hq_online` and the hosted database, each
  checked byte for byte. Main merged into `online` (`9cafaa6`), and the revised script loaded
  into both (identical to `hq`'s). A hosted backup was taken first.
- 2026-10-06: deployed `f36ffd3` (the merge from main, with the player screen's log kept
  open) with `make deploy`; live within about 30 seconds, `/healthz` ok.
- 2026-10-06: the GM's first test with a friend found an empty lobby: online play was started
  at 01:12 but the session opened to players only at 01:22. `8ef9da2`: **Start online play**
  now also opens the session, the row says "Open/Closed to players", and the GM chooses who
  plays each hero (**Played by** on the campaign page, for covering an absent friend; the
  hero's usual player is kept). The browser check found and fixed a real bug: the seat page
  dropped a seat sent for the event it already showed, so a handed-over hero appeared only on
  reload. How players join, scenario by scenario: `docs/ONLINE_PLAY_GUIDE.md`.
- 2026-10-06: deployed `3e9517a` (Played by, start-opens, the seat fix) with `make deploy`, which
  lives only in the online worktree; live within about 30 seconds.
- 2026-10-07: deployed `84e0994` (the two merges from main: table testing fixes, monster
  movement, hero sheets, housekeeping, the ESLint fix; no new migrations) with
  `make deploy`, after a hosted backup (`hosted-20261007-221121.dump`). `/healthz` ok, and the
  served `tracker.js` matches the image's byte for byte.
