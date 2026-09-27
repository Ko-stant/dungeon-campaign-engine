# Dungeon Campaign Engine

A companion app for running in-person HeroQuest games: build maps of any size, track a
quest as it plays out at the table, and pick up exactly where you left off next time.

- **Map creator** - paint boards (rooms, corridors, solid rock), then lay out quests:
  doors and secret doors, furniture, monsters, traps, notes and start squares.
- **Tracker** - heroes' body/mind/gold/equipment, monster sightings and wounds, doors,
  traps, discovered areas and a readable log of everything that happened. Nothing is
  enforced; the GM decides and the app remembers.

## Getting started

Requirements: Go 1.27+, Bun, Docker.

```bash
bun install
make tools          # air, gotestsum, golangci-lint, goose into ./.bin
make db-up          # Postgres 18 on port 5433 (see .env)
make dev            # http://localhost:7331 (live reload) or :8080
```

Create `.env` with `DATABASE_URL`, `POSTGRES_*` and `GOOSE_*` settings (see the Makefile's
database targets). Game content (`content/`, `assets/`) is not included; see `NOTICE.md`.

See `CLAUDE.md` for architecture and commands, and `docs/ROADMAP.md` for what's next.
