
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Project Neo** is a ride-sharing mobile and web application similar to Uber/DoorDash, with a unique data sourcing mechanism: ride requests and offers are extracted from messaging platform groups (WhatsApp, Telegram, etc.) using automated workers.

### Key Differentiators
- **Multi-platform message sourcing**: Messages from groups/channels (WhatsApp, Telegram, etc.) are parsed to extract ride information
- **Location context awareness**: Certain locations have specific meanings based on group context
- **Local-first architecture**: Avoiding cloud providers (AWS, etc.) due to IP blocking concerns
- **Self-hosted infrastructure**: Running on local machines initially, with VPS option for later

**Reference TODO.md for full development roadmap and current progress.**

---

## ⚠️ Known Issues / Pending Manual Steps

1. **Node.js activation:**
   - Node.js 24.14.0 installed via nvm but needs activation in new shells
   - **Action required:** Run `nvm use 24.14.0` in each new terminal

2. **Database migrations not yet written:**
   - `supabase/migrations/` is empty — schema exists in models but no SQL migration files
   - **Action required:** Write migration files before `supabase db push` will do anything

---

## 🔍 Quick Health Check Commands

```bash
# Check versions
node --version          # Should be v24.14.0
go version             # Should be go1.25.x
bun --version          # Should be 1.3.11

# Start the full stack
cp .env.example .env   # First time only
supabase start         # Start Supabase (first time: ~2 min)
docker compose up -d --build  # Start app services

# Verify services
curl http://localhost:8083/health    # workers → {"status":"ok","service":"workers"}
curl http://localhost:8082/health    # graphql-api → {"status":"ok","service":"graphql-api"}
curl -X POST http://localhost:8082/query \
  -H "Content-Type: application/json" \
  -d '{"query":"{ health }"}'       # → {"data":{"health":"ok"}}

# Supabase Studio
open http://localhost:54323

# Stop everything
docker compose down
supabase stop

# Run all checks (CI parity)
bun run format:check   # Check formatting (JS/TS + Go)
bun run lint           # Lint all code (JS/TS + Go)
bun run build          # Build all packages
```

---

## 📚 Important Notes for Future Sessions

- **Commit scope naming:** Use `graphql-api` (not `api`) per commitlint config
- **No commit attribution footers:** Don't add "Generated with Claude Code" or "Co-Authored-By"
- **Small, atomic commits:** Each commit should be focused on a single change
- **Monorepo structure:** `apps/*` for deployable services, `packages/*` for shared libraries
- **Service ports:** graphql-api → `8082`, workers → `8083` (8080/8081 occupied by other local processes)
- **DATABASE_URL:** Must include `?sslmode=disable` — Supabase local Postgres does not use SSL
- **Docker build context:** Both Dockerfiles use repo root as context (`.`) to access `packages/shared-go` and `go.work`
- **No test files:** Do not create test files, test commands, or test targets in this project

---

## Tech Stack

### Mobile (Priority)
- **Framework**: Flutter
- **Platforms**: iOS + Android
- **Language**: Dart

### Web
- To be determined (Flutter Web or separate React/Next.js stack)

### Backend Services
- **Language**: Golang 1.25+ (high-performance, concurrent, compiled binaries)
- **Database**: Supabase (local instance) - PostgreSQL + PostgREST + Realtime
- **API Protocol**: GraphQL (using gqlgen - type-safe, code-first)
- **ORM**: Bun (query builder + struct mapping)
- **Message Source Automation**: Multi-platform support
  - WhatsApp: whatsmeow (Go library) or via Node.js worker if needed
  - Telegram: go-telegram-bot-api (official Go support)
  - Other platforms can be added as separate workers
  - Runs on local machine initially, self-hosted VPS later

### Monorepo & Tooling
- **Monorepo Tool**: Turborepo v1.13.0+ (polyglot - supports Go + JS/TS)
- **Package Manager**: Bun v1.3.11+ (for tooling, scripts, and future web app)
- **Backend Runtime**: Golang (compiled binaries - 2-3x faster than Node.js)
- **Go Version**: 1.25.0 - managed via go.work
- **Node.js Version**: 24.14.0 LTS (Krypton) - for tooling only
- **Deployment**: Docker-based containerization with multi-stage builds (golang:1.25-alpine → distroless)

---

## Architecture

### High-Level Components

```
┌─────────────────────────────────────────────┐
│  Message Sources (WhatsApp, Telegram, etc.) │
└──────────┬──────────────────────────────────┘
           │
           ▼
┌─────────────────────┐
│      Workers        │
│    (Go Binary)      │
│  - Goroutines       │
│  - Parse messages   │
│  - Extract rides    │
│  - NLP/Regex        │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  GraphQL API        │
│    (Go - gqlgen)    │
│  - Type-safe        │
│  - Subscriptions    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Supabase (Local)   │
│  - PostgreSQL       │
│  - Realtime         │
│  - Auth             │
└─────────────────────┘
           │
           ▼
┌─────────────────────┬─────────────────────┐
│   Flutter Mobile    │      Web App        │
│   (iOS + Android)   │   (Future Phase)    │
└─────────────────────┴─────────────────────┘
```

### Monorepo Structure (Current)

```
project-neo/
├── apps/
│   ├── mobile/              # Flutter mobile app (Dart) — not started
│   ├── workers/             # Workers service (Go binary) — stub
│   │   ├── go.mod
│   │   ├── main.go
│   │   └── package.json
│   └── graphql-api/         # GraphQL API server (Go binary) — fully implemented
│       ├── go.mod
│       ├── main.go
│       ├── gqlgen.yml
│       ├── graph/           # Schema (.graphqls) + generated code + resolvers
│       └── internal/        # auth middleware, postgres broker/listener
├── packages/
│   └── shared-go/           # Shared Go code (models, repos, postgres impls)
│       ├── go.mod
│       ├── model/
│       ├── repository/
│       └── postgres/
├── supabase/                # Supabase local config (migrations/ is empty)
├── go.work                  # Go workspace (3 modules)
├── go.work.sum
├── docker-compose.yml       # Workers + GraphQL API (repo root context)
├── Makefile                 # Go tooling + Docker + Supabase targets
├── turbo.json
├── package.json
└── biome.json
```

### Real-Time Architecture

- **pg_notify**: PostgreSQL triggers fire `NOTIFY` on `rides_added`, `rides_updated`, `matches_updated` channels
- **Listener** (`internal/postgres/listener.go`): persistent connection, listens for NOTIFY, fetches entity, publishes to Broker
- **Broker** (`internal/postgres/broker.go`): in-memory pub/sub with buffered channels (capacity 4), fan-out to GraphQL subscribers

### Authentication

- Supabase JWT tokens (HMAC-SHA256)
- Middleware extracts `sub` (user UUID) and `role` claims
- Unauthenticated requests pass through; per-resolver auth enforcement

---

## GraphQL API

### Implemented Operations

**Queries:** `health`, `me`, `rides`, `ride`, `myRides`, `myMatches`, `groups`, `group`, `locations`

**Mutations:** `upsertUser`, `createRide`, `updateRide`, `cancelRide`, `acceptMatch`, `rejectMatch`, `completeMatch`, `cancelMatch`, `createGroup`, `upsertLocationContext`

**Subscriptions:** `rideAdded`, `rideStatusChanged`, `matchStatusChanged`

### Regenerating GraphQL Code

```bash
cd apps/graphql-api
go run github.com/99designs/gqlgen generate
```

---

## Database Schema

### Core Tables (designed, migrations not yet written)

**`users`** - `id`, `email`, `phone`, `name`, `role` (rider|driver|both), `avatar_url`, timestamps

**`groups`** - `id`, `name`, `description`, `is_active`, timestamps

**`group_sources`** - `id`, `group_id`, `source_type` (whatsapp|telegram|discord|slack|manual), `source_identifier`, `last_parsed_at`, `parse_frequency`, `metadata` (jsonb), `is_active`, timestamps

**`location_contexts`** - `id`, `group_id`, `location_alias`, `location_name`, `latitude`, `longitude`, `address`, `metadata` (jsonb), timestamps

**`messages`** - `id`, `group_id`, `source_message_id`, `sender_identifier`, `content`, `timestamp`, `parsed_at`, `parse_status` (pending|success|failed|skipped), `parse_error`, `metadata` (jsonb)

**`rides`** - `id`, `message_id`, `group_id`, `type` (need_ride|ride_available), `from/to_location_id`, `from/to_location_text`, `departure_time`, `is_immediate`, `cost`, `currency`, `distance`, `seats_available`, `status` (available|matched|completed|cancelled|expired), `poster_user_id`, timestamps

**`matches`** - `id`, `ride_id`, `rider_id`, `driver_id`, `status` (pending|accepted|rejected|completed|cancelled), `matched_at`, `accepted_at`, `completed_at`, `cancelled_at`, timestamps

**`locations`** - `id`, `name`, `latitude`, `longitude`, `address`, `city`, `state`, `country`, `metadata` (jsonb)

### Realtime Requirements
- pg_notify triggers needed on `rides` and `matches` tables (channels: `rides_added`, `rides_updated`, `matches_updated`)
- These must be part of the migration files

---

## Message Format

Messages in groups follow patterns like:
- **Timing**: "now" or specific time (e.g., "3:30 PM", "in 30 mins")
- **Type**: "Need ride" or "Ride available"
- **Route**: "from [Location A] to [Location B]"
- **Cost**: Price in local currency
- **Distance**: (Optional) "X km" or "X miles"

Example:
```
Need ride now
From Downtown to Airport
$25
15km
```

### Parsing Strategy
- NLP/regex to extract structured data
- Map location aliases to coordinates via `location_contexts` table
- Store raw message + parsed fields
- Handle variations in format

---

## Development Commands

### Monorepo
```bash
bun install           # Install JS/TS dependencies
go work sync          # Sync Go workspace
bun run build         # Build all packages
bun run dev           # Run all apps in dev mode
```

### Supabase Local
```bash
supabase start        # Start Supabase locally
supabase stop         # Stop Supabase
supabase db push      # Run migrations
supabase db reset     # Reset database
```

### Flutter Mobile App
```bash
cd apps/mobile
flutter pub get
flutter run -d ios
flutter run -d android
flutter build apk
```

### Workers Service (Go)
```bash
cd apps/workers
go mod download
air                             # Live reload (preferred)
go run .                        # Without Air
go build -o bin/workers .
```

### GraphQL API (Go)
```bash
cd apps/graphql-api
go mod download
go run github.com/99designs/gqlgen generate   # Regenerate from schema
air                             # Live reload (preferred)
go run .                        # Without Air
go build -o bin/graphql-api .
```

### Go Tooling
```bash
make install-tools    # Install golangci-lint, gofumpt, air
make lint-go          # Full lint (30+ linters)
make format-go        # Format (goimports + gofumpt)
make docker-up        # Start app containers
make docker-down      # Stop app containers
make dev-up           # Start Supabase + app containers
make dev-down         # Stop everything
```

---

## Critical Architectural Decisions

### Why Local-First?
- Cloud provider IPs (AWS, GCP, Azure) get blocked by certain services
- Full control over infrastructure
- Lower costs during development
- Easy transition to self-hosted VPS later

### Why Supabase Local?
- PostgreSQL with instant REST and GraphQL APIs
- Built-in realtime subscriptions
- Auth and storage included
- Can self-host in production

### Why Flutter?
- True cross-platform (iOS + Android from single codebase)
- Excellent performance (compiled to native)
- Rich UI component library (Material + Cupertino)

### Why Golang for Backend?
- 2-3x faster than Node.js/Bun; goroutines for concurrent message streams
- 50-70% lower memory usage
- Single compiled binary, no runtime dependencies
- Strong static typing

### Why Bun (for Tooling)?
- 3-5x faster installs than npm/pnpm
- All-in-one: package manager + runtime + bundler
- Works alongside Go seamlessly via Turborepo

### Why Biome?
- 10-25x faster than Prettier + ESLint combined
- Single tool for formatting AND linting

### Message Source Automation Risks
- **WhatsApp**: Unofficial libraries violate ToS — use dedicated accounts, rate limiting
- **Telegram**: Bot API is official and safe; MTProto requires proper auth
- **Other platforms**: Review ToS; use official APIs when available

---

## Development Workflow

### Conventional Commits

**Format**: `type(scope): subject`

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `build`, `ci`, `chore`

**Scopes**: `mobile`, `workers`, `graphql-api`, `database`, `shared`, `deps`, `release`, `docker`, `ci`, `docs`

**Rules:**
- Use imperative mood ("add" not "added")
- No attribution footers
- Keep commits small and focused

### Code Formatting

**JS/TS** — Biome: single quotes, 2 spaces, semicolons, 80 char width

**Go** — gofumpt + golangci-lint (30+ linters): `make format-go`, `make lint-go`

**Git Hooks (automatic):**
- `pre-commit`: Biome on JS/TS, gofumpt on Go
- `commit-msg`: conventional commits validation

### CI/CD Pipeline

**On every push/PR** (`.github/workflows/ci.yml`):
- Format check, lint, build verification for JS/TS and Go
- Turborepo cache optimization

**On merge to main** (`.github/workflows/release.yml`):
- Auto-create Version PR (Changesets)
- Create GitHub releases on Version PR merge

---

## Important Conventions

### Code Style
- **JavaScript/TypeScript**: Single quotes, 2 spaces, semicolons (Biome enforced)
- **Dart**: Follow official Dart style guide
- **JSON**: No trailing commas (Biome enforced)
- **Commits**: Conventional commits (commitlint enforced)

### API Design
- Use GraphQL subscriptions for realtime features
- Implement proper error handling and validation
- Rate limiting on all endpoints

### Security Considerations
- Never commit credentials (WhatsApp, API keys, etc.)
- Use environment variables for secrets
- `.gitignore` excludes sensitive files
- Validate all parsed data; sanitize user inputs

---

## Location Context System

Each group has specific location contexts where aliases map to coordinates. Configurable per group via `location_contexts` table.

Example:
- Group: "Downtown Riders"
  - "Station" → Central Train Station (lat/long)
  - "Mall" → City Center Mall (lat/long)
  - "Airport" → International Airport (lat/long)

---

## Resources

- Flutter: https://flutter.dev
- Supabase self-hosting: https://supabase.com/docs/guides/self-hosting/docker
- Turborepo: https://turbo.build/repo/docs
- gqlgen: https://gqlgen.com
- Bun ORM: https://bun.uptrace.dev
- GraphQL: https://graphql.org
