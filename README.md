# Dungeon Campaign Engine

A companion app for a GM running HeroQuest games: build maps of any size, track a quest as it
plays out, show the players what their heroes know, and pick up exactly where you left off
next time.

> **Branch `online`:** this branch adds online play to the table companion on `main`: a Go
> rules engine, sign-in, player seats and a hosted site. It merges back into `main` when the
> GM wants the online features at the table. Plan and progress:
> `docs/ONLINE_AND_RULES_PLAN.md`.

At the table nothing is enforced: the GM decides, and the app records the result in a
readable log. Online, the rules engine gates what players can do from their seats, and the
GM can still override anything.

## At the table (as on `main`)

- **Map creator** (`/maps`): paint boards (rooms, corridors, solid rock, drawn walls), then
  lay out quests: doors, gates and secret doors, furniture, monsters, traps, lettered notes
  and start squares, with undo/redo and advisory checks.
- **Campaigns** (`/campaigns`): heroes that carry gold, items and notes between quests,
  ordered chapters (each quest on its own map), a loot list, the campaign's own monster stat
  lines, and a read-aloud script with audio clips per passage (stored in Postgres on this
  branch).
- **Custom content**: monsters (`/monsters`) and hero classes (`/classes`) with dice stats,
  mana and abilities with cooldowns.
- **Tracker** (`/play/{id}`): moves, Body and mana, doors, traps, what the heroes have seen,
  fights with rounds, cooldowns and effects, inventory and potions, travel between maps, and
  odds hints for the selected monster (advice only). Sessions resume from Postgres.
- **Player screen** (`/play/{id}/players`): a live, read-only view for the TV showing only
  what the heroes know.

## Online play (this branch)

- **Rules mode**, per session: the engine knows turns, movement, line of sight and dice,
  rolls the dice itself and lists the legal actions. Heroes take one turn each per round, in
  any order, then the GM moves the monsters. A session without rules stays the table game.
- **Sign-in** with Discord (or a local dev login). New users wait until an admin lets them in
  on `/members`. With sign-in off (`AUTH_MODE` unset) the app behaves like the table version.
- **Seats**: players join from `/lobby` or an invite link, pick a free hero (or the GM assigns
  one), and play from their own browser on the seat page (`/play/{id}/seat`). The GM runs the
  session from the tracker's rules console. How players join: `docs/ONLINE_PLAY_GUIDE.md`.
- **Hosting**: a Docker image (it contains the game content, so it goes only to a private
  registry) deployed on Render with a hosted Postgres; `make deploy`, `make hosted-backup`.

Still to come, in order: monster AI, an automated GM, a headless simulator with scripted bots,
and reinforcement-learning bots that play whole quests to find balance problems.

Desktop browsers only; phones aren't supported.

## Getting started

Requirements: Go 1.27+, Bun, Docker.

```bash
cp .env.example .env  # then set your own password
bun install
make tools            # air, gotestsum, golangci-lint, goose into ./.bin
make db-up            # Postgres 18 on POSTGRES_PORT
make dev              # http://localhost:7331 (live reload) or APP_PORT
```

To try online play locally, set `AUTH_MODE=dev` in `.env` (any name signs in, loopback only)
and add yourself to `AUTH_ADMINS` as `dev:<name>`.

Game content (`content/`, `assets/`) is not included; see `NOTICE.md`. With a copy in place,
`make import-content` loads the base board and a quest. Without it you can still paint boards
and use custom monsters and classes.

`make test` and `bun test` run the tests (`make test-db` adds the database tests);
`make lint` runs the Go and TypeScript linters and the spelling check.

If you also keep a `main` checkout, give this one its own database (another port and
database name in `.env`): the server migrates on start, and this branch's migrations must not
reach the table's database.

## Docs

- `CLAUDE.md`: architecture, conventions and every command.
- `docs/ONLINE_AND_RULES_PLAN.md`: the online plan, its phases and a running log.
- `docs/ONLINE_PLAY_GUIDE.md`: how players get into an online game.
- `docs/ROADMAP.md`: the table companion's roadmap.
- `docs/campaigns/three-plagues/`: the campaign being built with the app (contains spoilers).
