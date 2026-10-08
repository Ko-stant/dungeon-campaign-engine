# Dungeon Campaign Engine

A companion app for a GM running in-person HeroQuest games: build maps of any size, track a
quest as it plays out at the table, show the players what their heroes know on a TV, and
pick up exactly where you left off next time.

Nothing is enforced. The GM decides, and the app records the result in a readable log.

- **Map creator** (`/maps`): paint boards (rooms, corridors, solid rock, drawn walls), then
  lay out quests: doors, gates and secret doors, furniture, monsters, traps, lettered notes
  and start squares, with undo/redo and advisory checks.
- **Campaigns** (`/campaigns`): heroes that carry gold, items and notes between quests,
  ordered chapters (each quest on its own map), a loot list, the campaign's own monster stat
  lines, and a read-aloud script with audio clips per passage.
- **Custom content**: monsters (`/monsters`) and hero classes (`/classes`) with dice stats,
  mana and abilities with cooldowns.
- **Tracker** (`/play/{id}`): moves, Body and mana, doors, traps, what the heroes have seen,
  fights with rounds, cooldowns and effects, inventory and potions, travel between maps, and
  odds hints for the selected monster (advice only). Sessions resume from Postgres.
- **Player screen** (`/play/{id}/players`): a live, read-only view for the TV showing only
  what the heroes know.

## Getting started

Requirements: Go 1.27+, Bun, Docker.

```bash
cp .env.example .env  # then set your own password
bun install
make tools            # air, gotestsum, golangci-lint, goose into ./.bin
make db-up            # Postgres 18 on port 5433
make dev              # http://localhost:7331 (live reload) or :8080
```

Game content (`content/`, `assets/`) is not included; see `NOTICE.md`. With a copy in place,
`make import-content` loads the base board and a quest. Without it you can still paint boards
and use custom monsters and classes.

`make test` and `bun test` run the tests; `make lint` runs the Go and TypeScript linters and
the spelling check.

## Docs

- `CLAUDE.md`: architecture, conventions and every command.
- `docs/ROADMAP.md`: what's done and what's next.
- `docs/campaigns/three-plagues/`: the campaign being built with the app (contains spoilers).
