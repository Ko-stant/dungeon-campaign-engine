# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a dungeon campaign engine for HeroQuest built with Go, featuring real-time gameplay through WebSockets and a web frontend using Templ templates with TailwindCSS.

## Common Commands

### Development
- `make dev` - Start development mode with hot reloading (runs tailwind watch, templ watch, and air)
- `make tools` - Install all development tools (air, templ, gotestsum, golangci-lint, goose)
- `make run` - Build and run the server once
- `make build` - Build production binary to `./build/dungeon-campaign-engine`

### Testing and Quality
- `make test` - Run tests with gotestsum
- `make test-race` - Run tests with race detection
- `make cover` - Generate test coverage report
- `make lint` - Run golangci-lint
- `make fmt` - Format and vet Go code

### Database (PostgreSQL via Docker Compose)
- `make db-up` - Start PostgreSQL container
- `make db-migrate-up` - Run pending migrations
- `make db-migrate-down` - Rollback last migration
- `make db-migrate-new` - Create new migration file
- `make db-psql` - Connect to database shell

### Frontend Assets
- `npm run tailwind:build` - Build TailwindCSS (production)
- `npm run tailwind:watch` - Watch and rebuild TailwindCSS (development)

## Architecture

### Core Components

**Game State Management** (`cmd/server/main.go`):
- Central `GameState` struct manages dungeon layout, entities, doors, and visibility
- Real-time updates via WebSocket protocol with sequence numbers
- Thread-safe operations using mutexes

**Geometry System** (`internal/geometry/`):
- `Segment` - Represents dungeon layout with walls, doors, and dimensions
- `RegionMap` - Maps tiles to room/corridor regions for visibility calculations
- Line-of-sight algorithm using grid traversal for door visibility
- Procedural dungeon generation with corridors and rooms

**Protocol** (`internal/protocol/`):
- Event-driven architecture with `IntentEnvelope` (client→server) and `PatchEnvelope` (server→client)
- Snapshot system for initial game state delivery
- Real-time patches for movement, door state, visibility, and region discovery

**Web Frontend** (`internal/web/`):
- Templ templates for server-side rendering
- WebSocket client for real-time game updates
- TailwindCSS for styling (built from `internal/web/static/styles/index.css`)

### Key Patterns

**Visibility System**:
- Regions are "known" (discovered), "revealed" (accessible), or "visible" (currently in line-of-sight)
- Opening doors reveals connected rooms and updates visibility calculations

**Movement Validation**:
- Checks map boundaries, wall collisions, and door states
- Uses edge-based collision detection system

**Template System**:
- Uses `a-h/templ` for type-safe HTML generation
- Templates must be generated before building: `templ generate -path=./internal/web/views`

## Game Master Overrides & Custom Rules

Beyond standard HeroQuest mechanics, this engine is designed to support Game Master agency and custom rule variations:

### Monster Placement Flexibility
- **Delayed Monster Reveal**: Monsters may not be immediately revealed when doors open, even if technically "visible"
- GM can choose to place monsters on-demand when heroes enter rooms, particularly for enemies in corners or areas outside initial line-of-sight
- Allows for dramatic timing and narrative control over encounters

### Custom Dice Rolling Rules
- **Double Dice Effects** (custom house rules):
  - Double 1s: Roll one fewer attack die on next attack
  - Double 2-5: Can reroll one attack die
  - Double 6s: Can reroll 2 attack dice
- System should support toggling these custom rules on/off per campaign

### Player Negotiation & Dynamic Rewards
- **Bargaining System**: GM can offer alternative outcomes based on dice rolls or player actions
- Example: Allow players to risk/gamble rewards (e.g., "roll double 6s to keep treasure chest")
- Support for ad-hoc rule modifications during gameplay to maintain engagement

### Line of Sight Overrides
- **Flexible LOS Rules**: GM can override strict "center-of-tile to center-of-tile" calculations
- Allow attacks/actions that would normally be blocked by allies or monsters
- Provide narrative context for rule bends (e.g., "Knight ducks to allow crossbow shot")
- Prioritize fun and roleplay over rigid rule adherence

### Implementation Considerations
- All GM overrides should be optional toggles, not replacing core mechanics
- Need UI controls for GM to make real-time rule adjustments
- Consider logging override decisions for campaign consistency
- Custom rules should be per-campaign configurable, not global settings
