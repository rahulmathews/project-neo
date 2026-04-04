# project-neo

## Quick Start

### Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) installed and running
- [Supabase CLI](https://supabase.com/docs/guides/cli/getting-started) — install via `scoop install supabase` or `winget install Supabase.CLI`
- [Bun](https://bun.sh) v1.3.11+
- [Go](https://golang.org) 1.24.4+

### First-time setup

```bash
# Install JS dependencies
bun install

# Initialize Supabase (first time only)
supabase init

# Copy environment config
cp .env.example .env
```

### Start the stack

```bash
# Start Supabase (first run downloads images, ~2 min)
supabase start

# Start app services
docker compose up -d --build
```

### Verify everything is running

```bash
curl http://localhost:8081/health
# → {"status":"ok","service":"workers"}

curl http://localhost:8080/health
# → {"status":"ok","service":"graphql-api"}

curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query":"{ health }"}'
# → {"data":{"health":"ok"}}
```

Open in browser:
- GraphQL Playground: http://localhost:8080/
- Supabase Studio: http://localhost:54323

### Stop

```bash
docker compose down
supabase stop
```

### Development checks

```bash
bun run format:check   # formatting
bun run lint           # linting
bun run build          # build all
```