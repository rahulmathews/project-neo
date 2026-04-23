[![CI](https://github.com/rahulmathews/project-neo/actions/workflows/ci.yml/badge.svg)](https://github.com/rahulmathews/project-neo/actions/workflows/ci.yml)
[![workers](https://img.shields.io/github/v/tag/rahulmathews/project-neo?filter=workers-*&label=workers)](https://github.com/rahulmathews/project-neo/releases?q=workers)
[![graphql-api](https://img.shields.io/github/v/tag/rahulmathews/project-neo?filter=graphql-api-*&label=graphql-api)](https://github.com/rahulmathews/project-neo/releases?q=graphql-api)
[![mobile](https://img.shields.io/github/v/tag/rahulmathews/project-neo?filter=mobile-*&label=mobile)](https://github.com/rahulmathews/project-neo/releases?q=mobile)
[![Dependabot](https://img.shields.io/badge/Dependabot-enabled-brightgreen.svg)](https://github.com/rahulmathews/project-neo/network/updates)

# Project Neo

Ride-sharing app that sources ride requests and offers from messaging platform groups (WhatsApp, Telegram, etc.).

## Architecture

```
┌─────────────────────────────────────────────┐
│  Message Sources (WhatsApp, Telegram, etc.) │
└──────────┬──────────────────────────────────┘
           │
           ▼
┌─────────────────────┐
│      Workers        │
│    (Go Binary)      │
│  - Parse messages   │
│  - Extract rides    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  GraphQL API        │
│    (Go - gqlgen)    │
│  - Queries          │
│  - Mutations        │
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

## Tech Stack

| Service | Language / Framework |
|---------|----------------------|
| Workers | Go 1.25, whatsmeow, go-telegram-bot-api |
| GraphQL API | Go 1.25, gqlgen, Bun ORM |
| Shared library | Go 1.25 |
| Database | Supabase (PostgreSQL + Realtime + Auth) |
| Mobile | Flutter (Dart) |
| Tooling | Bun, Turborepo, Biome |

## Prerequisites

- **Node.js 24.14.0** via nvm — run `nvm use 24.14.0` in each new shell
- **Go 1.25+** — [golang.org/dl](https://golang.org/dl) — `go version`
- **Bun 1.3.11+** — [bun.sh](https://bun.sh) — `bun --version`
- **Flutter** — [flutter.dev](https://flutter.dev/docs/get-started/install) — `flutter --version`
- **Docker** with Docker Compose
- **Supabase CLI** — `supabase --version`

## Setup

```bash
# 1. Clone
git clone <repo-url>
cd project-neo

# 2. Install JS tooling dependencies
bun install

# 3. Environment
cp .env.example .env
# Edit .env — fill in SUPABASE_JWT_SECRET after step 4
```

```bash
# 4. Start Supabase (first run pulls images — ~2 min)
supabase start

# Copy the "JWT secret" from the output above into SUPABASE_JWT_SECRET in .env
# Or run: supabase status
```

```bash
# 5. Apply database migrations
supabase db push
```

```bash
# 6. Start app services
docker compose up -d --build
```

```bash
# 7. Verify
curl http://localhost:8083/health    # → {"status":"ok","service":"workers"}
curl http://localhost:8082/health    # → {"status":"ok","service":"graphql-api"}
```

## Services

| Service | Description | Port | README |
|---------|-------------|------|--------|
| workers | Connects to WhatsApp/Telegram, parses messages, extracts rides | 8083 | [apps/workers](apps/workers/README.md) |
| graphql-api | GraphQL API for rides, matches, groups | 8082 | [apps/graphql-api](apps/graphql-api/README.md) |
| shared-go | Shared Go models and repository interfaces | — | [packages/shared-go](packages/shared-go/README.md) |
| supabase | Local PostgreSQL + Realtime + Auth + Studio (54323) | 54321 | [supabase](supabase/README.md) |
| mobile | Flutter mobile app (iOS + Android) — auth, drawer nav, GraphQL client | — | [apps/mobile](apps/mobile/README.md) |

## Development Commands

```bash
# Run all services in dev mode
bun run dev

# Check formatting (JS/TS + Go)
bun run format:check

# Lint all code
bun run lint

# Build all packages
bun run build
```

## Contributing

This project uses [Conventional Commits](https://www.conventionalcommits.org/).

**Format:** `type(scope): subject`

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `build`, `ci`, `chore`

**Scopes:** `mobile`, `workers`, `graphql-api`, `database`, `shared`, `deps`, `release`, `docker`, `ci`, `docs`

**Rules:**
- Imperative mood: "add" not "added"
- No attribution footers
- Keep commits small and focused
