# shared-go

Shared Go library consumed by both `workers` and `graphql-api`. Contains domain models, repository interfaces, and PostgreSQL implementations.

## Architecture

```
packages/shared-go/
├─ model/               Domain types (structs + enums)
│  ├─ user.go
│  ├─ group.go
│  ├─ group_source.go
│  ├─ message.go
│  ├─ ride.go
│  ├─ match.go
│  ├─ location.go
│  └─ inputs.go         Input types for create/update operations
├─ repository/          Interfaces — contract only, no implementation
│  ├─ user.go
│  ├─ group.go
│  ├─ ride.go
│  ├─ match.go
│  └─ location.go
└─ postgres/            PostgreSQL implementations (bun ORM)
   ├─ db.go             NewDB(dsn) — creates a *bun.DB
   ├─ user.go
   ├─ group.go
   ├─ group_source.go
   ├─ ride.go
   ├─ ride_write.go
   ├─ match.go
   ├─ location.go
   └─ message.go
```

Consumed by `apps/workers` and `apps/graphql-api` via `go.work`. Not deployed independently.

## Environment Variables

None. The database connection is passed in at construction time via `postgres.NewDB(dsn string)`.

## Running Locally

This is a library — no `main` package, not runnable directly. It is compiled into `workers` and `graphql-api`.

```bash
cd packages/shared-go

# Download dependencies
go mod download

# Vet the package
go vet ./...
```

## Repository Interfaces

| Interface | Methods | Description |
|-----------|---------|-------------|
| `UserRepository` | `GetByID`, `Upsert` | User profile read/write |
| `RideRepository` | `GetByID`, `List`, `ListByUser`, `Create`, `Update`, `Cancel`, `SetStatus` | Full ride CRUD |
| `MatchRepository` | `GetByID`, `ListByUser`, `Create`, `UpdateStatus` | Match lifecycle |
| `GroupRepository` | `List`, `GetByID`, `Create`, `ListLocationContexts`, `GetLocationContextByID`, `UpsertLocationContext` | Groups and their location aliases |
| `LocationRepository` | `GetByID`, `Search` | Location lookup and full-text search |

## Core Model Types

| Type | Description |
|------|-------------|
| `User` | Rider/driver profile: id, name, email, phone, role (RIDER/DRIVER/BOTH), avatar_url |
| `Group` | Messaging group: id, name, description, is_active |
| `GroupSource` | Platform connection for a group: source_type (whatsapp/telegram/etc.), source_identifier, last_parsed_at, is_active |
| `Message` | Raw ingested message: content, sender_identifier, timestamp, parse_status (pending/success/failed/skipped) |
| `Ride` | Ride posting: type (need_ride/ride_available), from/to location, departure_time, is_immediate, cost, seats_available, status |
| `Match` | Rider–driver pairing on a ride: status (pending/accepted/rejected/completed/cancelled), timestamps |
| `Location` | Named geographic point: name, lat/long, address, city, country |
| `LocationContext` | Group-specific alias mapping: location_alias → Location (e.g. "Station" → Central Train Station) |
