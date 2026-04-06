# supabase

Local Supabase instance for Project Neo. Provides PostgreSQL, Realtime, Auth, and Studio — all running as Docker containers managed by the Supabase CLI.

## Architecture

```
supabase/
├─ config.toml                                    All local Supabase settings
└─ migrations/
   ├─ 20260331000000_baseline.sql                 All 8 tables, RLS, triggers, indexes
   ├─ 20260404000000_notify_triggers.sql           pg_notify triggers for rides + matches
   ├─ 20260405000000_messages_content_hash.sql     content_hash column on messages
   ├─ 20260405000001_group_upsert_constraints.sql  Unique constraints for group upsert
   ├─ 20260406000000_messages_notify_trigger.sql   pg_notify trigger for messages
   └─ 20260406000001_messages_group_content_hash_idx.sql  Composite index on messages
```

Consumed by: `apps/workers` and `apps/graphql-api` via `DATABASE_URL`. The pg_notify triggers fire on `rides_added`, `rides_updated`, and `matches_updated` channels — graphql-api listens to these for real-time subscriptions.

## Environment Variables

| Variable | Description | Example | Required | Notes |
|----------|-------------|---------|----------|-------|
| `SUPABASE_JWT_SECRET` | JWT secret for token verification in graphql-api | `super-secret-jwt-token-with-at-least-32-characters-long` | Used by graphql-api | Copy from `supabase status` after starting |
| `OPENAI_API_KEY` | Enables AI features in Supabase Studio | `sk-...` | No | Studio works without it |

## Running Locally

```bash
# Start all Supabase containers (first run pulls ~8 images — allow ~2 min)
supabase start

# Stop all Supabase containers
supabase stop

# Apply pending migrations to the running database
supabase db push

# Reset database — drops everything and re-runs all migrations from scratch
supabase db reset

# Show service URLs, ports, and keys (including JWT secret)
supabase status
```

Supabase Studio: http://localhost:54323

## Configuration

Key settings in `config.toml`:

| Setting | Value | Notes |
|---------|-------|-------|
| `project_id` | `neo` | Suffix for all container names: `supabase_{service}_neo` |
| `api.port` | `54321` | PostgREST API |
| `db.port` | `54322` | PostgreSQL — use this in `DATABASE_URL` |
| `studio.port` | `54323` | Studio web UI |
| `inbucket.port` | `54324` | Email testing UI |
| `db.major_version` | `17` | Must match remote database version |
| `analytics.enabled` | `false` | Disabled locally |

Container names: `supabase_db_neo`, `supabase_kong_neo`, `supabase_auth_neo`, `supabase_rest_neo`, `supabase_realtime_neo`, `supabase_storage_neo`, `supabase_studio_neo`, `supabase_inbucket_neo`.

## Troubleshooting

**First `supabase start` is slow** — ~2 minutes on first run while Docker pulls the images. Subsequent starts are fast.

**Migrations not applied** — Run `supabase db push` after `supabase start` to apply any pending migrations.

**Changing `project_id` in config.toml** — Always run `supabase stop` first. After changing `project_id`, old containers become orphaned. Clean them up:

```bash
docker compose -p <old-project-id> down
```

**Cannot connect from Docker containers** — Use `host.docker.internal` instead of `localhost` in `DATABASE_URL` when connecting from inside a Docker container. Docker Desktop on Windows/Mac resolves `host.docker.internal` to the host machine automatically.
