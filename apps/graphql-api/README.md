# graphql-api

GraphQL API server for Project Neo. Exposes ride, match, group, user, and location operations. Supports real-time subscriptions via WebSocket, powered by PostgreSQL NOTIFY triggers.

## Architecture

```
apps/graphql-api/
├─ main.go                              Entry point — wires DB, broker, listener, HTTP server
├─ gqlgen.yml                           gqlgen code generation config
├─ graph/
│  ├─ query.graphqls                    Query definitions
│  ├─ mutation.graphqls                 Mutation definitions + input types
│  ├─ subscription.graphqls             Subscription definitions
│  ├─ types.graphqls                    Domain types (Ride, Match, User, etc.)
│  ├─ generated/generated.go            gqlgen-generated code — do not edit manually
│  ├─ model/models_gen.go               gqlgen-generated model types
│  └─ resolvers/
│     ├─ resolver.go                    Resolver struct (holds all repos + broker)
│     ├─ query.resolvers.go             Query implementations
│     ├─ mutation.resolvers.go          Mutation implementations
│     ├─ subscription.resolvers.go      Subscription implementations
│     └─ types.resolvers.go             Field resolver implementations
└─ internal/
   ├─ auth/middleware.go                JWT auth middleware (Supabase HMAC-SHA256)
   ├─ postgres/broker.go                In-memory pub/sub — fans out NOTIFY events to GraphQL subscribers
   └─ postgres/listener.go              Persistent pg_notify listener — feeds broker on ride/match changes
```

Connects to: Supabase PostgreSQL (`DATABASE_URL`). Reads/writes all tables. Listens on `rides_added`, `rides_updated`, `matches_updated` NOTIFY channels.

## Environment Variables

| Variable | Description | Example | Required | Notes |
|----------|-------------|---------|----------|-------|
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://postgres:postgres@localhost:54322/postgres?sslmode=disable` | Yes | Must include `?sslmode=disable` |
| `SUPABASE_JWT_SECRET` | JWT secret for auth middleware | `super-secret-jwt-token-with-at-least-32-characters-long` | Yes | Get from `supabase status` |
| `PORT` | HTTP server bind port | `8082` | No | Defaults to `8080`; set to `8082` in Docker via compose |

## Running Locally

Prerequisites: Go 1.25+, Supabase running (`supabase start`), `.env` configured.

```bash
cd apps/graphql-api

# Live reload (requires air — install via: make install-tools)
air

# Without live reload
PORT=8082 go run .

# Build binary
go build -o bin/graphql-api .
```

- GraphQL Playground: http://localhost:8082/
- Health check: `curl http://localhost:8082/health`

To regenerate GraphQL code after schema changes:

```bash
go run github.com/99designs/gqlgen generate
```

## Building & Docker

```bash
# Build binary (from this directory)
go build -o bin/graphql-api .

# Docker build — must run from repo root (not apps/graphql-api/)
cd <repo-root>
docker build -f apps/graphql-api/Dockerfile -t graphql-api .
```

Multi-stage build: `golang:1.25-alpine` → `distroless/static:nonroot`. Copies `go.work` and `packages/shared-go` from repo root to resolve the Go workspace.

## API Surface

### Queries

| Operation | Description |
|-----------|-------------|
| `health` | Returns `"ok"` — liveness check |
| `me` | Returns the authenticated user's profile |
| `rides(groupId, type?, status?, limit?, offset?)` | Lists rides for a group, optionally filtered by type and status |
| `ride(id)` | Returns a single ride by ID |
| `myRides(limit?, offset?)` | Lists rides posted by the authenticated user |
| `myMatches(limit?, offset?)` | Lists matches for the authenticated user |
| `groups` | Lists all groups |
| `group(id)` | Returns a single group by ID (includes location contexts) |
| `locations(query)` | Full-text search for locations |

### Mutations

| Operation | Description |
|-----------|-------------|
| `upsertUser(input)` | Creates or updates the authenticated user's profile |
| `createRide(input)` | Creates a new ride posting |
| `updateRide(id, input)` | Updates departure time, cost, or seats on a ride |
| `cancelRide(id)` | Cancels a ride (poster only) |
| `acceptMatch(rideId)` | Driver accepts a pending match on a ride |
| `rejectMatch(matchId)` | Rejects a pending match |
| `completeMatch(matchId)` | Marks a match as completed |
| `cancelMatch(matchId)` | Cancels an active match |
| `createGroup(input)` | Creates a new group |
| `upsertLocationContext(input)` | Creates or updates a location alias for a group |

### Subscriptions

| Operation | Description |
|-----------|-------------|
| `rideAdded(groupId)` | Emits when a new ride is added to the group |
| `rideStatusChanged(groupId)` | Emits when a ride's status changes in the group |
| `matchStatusChanged` | Emits when any of the authenticated user's matches changes status |

## Troubleshooting

**`DATABASE_URL` connection refused** — Ensure Supabase is running (`supabase start`) and `DATABASE_URL` includes `?sslmode=disable`.

**Port conflict with local processes** — Ports 8080 and 8081 are occupied on the dev machine. When running locally without Docker, explicitly set `PORT=8082`.

**Docker build fails with "no required module"** — Build must run from repo root: `docker build -f apps/graphql-api/Dockerfile -t graphql-api .`

**Subscriptions not receiving events** — Verify pg_notify triggers are applied (`supabase db push`) and that `DATABASE_URL` points to the running Supabase instance.
