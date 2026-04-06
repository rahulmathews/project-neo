# workers

Connects to messaging platform groups (WhatsApp, Telegram), ingests messages, parses them to extract ride information, and writes structured ride records to the database.

## Architecture

```
apps/workers/
├─ main.go                    Entry point
├─ run.go                     run() — init DB, build connectors, start health server + parser, graceful shutdown
├─ internal/
│  ├─ connector.go            Connector interface + NewConnectors factory
│  └─ store/
│     ├─ message.go           Saves raw messages to DB
│     └─ group_source.go      Upserts group/source records
├─ whatsapp/
│  ├─ client.go               WhatsApp connector (whatsmeow) — QR auth, group sync, message forwarding
│  └─ handler.go              Message event handler — stores message, triggers parser
└─ parser/
   ├─ listener.go             Polls DB for PENDING messages, dispatches Process()
   ├─ extractor.go            Process() — regex → Haiku → location resolve → write ride
   ├─ regex.go                Regex-based ride extraction
   ├─ haiku.go                Claude Haiku AI extraction (fallback when regex misses)
   ├─ location.go             Resolves location aliases via location_contexts table
   ├─ writer.go               Writes ride to DB, updates message parse status
   └─ types.go                ParsedRide type
```

Connects to: Supabase PostgreSQL (`DATABASE_URL`). Writes to `messages` and `rides`. Reads from `groups`, `group_sources`, `location_contexts`.

## Environment Variables

| Variable | Description | Example | Required | Notes |
|----------|-------------|---------|----------|-------|
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://postgres:postgres@localhost:54322/postgres?sslmode=disable` | Yes | Must include `?sslmode=disable` — local Supabase does not use SSL |
| `ANTHROPIC_API_KEY` | Claude Haiku API key for AI-based message parsing | `sk-ant-...` | No | If unset, messages that don't match regex are marked `FAILED` instead of parsed by AI |
| `PORT` | Health server bind port | `8083` | No | Defaults to `8083` |

## Running Locally

Prerequisites: Go 1.25+, Supabase running (`supabase start`), `.env` configured.

```bash
cd apps/workers

# Live reload (requires air — install via: make install-tools)
air

# Without live reload
go run .

# Build binary
go build -o bin/workers .
```

Health check: `curl http://localhost:8083/health`

On first run, a QR code is printed to stdout — scan it with **WhatsApp → Linked Devices → Link a Device** within 60 seconds. Subsequent runs resume the session silently from the database.

## Building & Docker

```bash
# Build binary (from this directory)
go build -o bin/workers .

# Docker build — must run from repo root (not apps/workers/)
cd <repo-root>
docker build -f apps/workers/Dockerfile -t workers .
```

The Dockerfile uses a multi-stage build (`golang:1.25-alpine` → `distroless/static:nonroot`) and copies `go.work` + `packages/shared-go` from the repo root to resolve the Go workspace.

## Message Parsing Pipeline

```
Incoming WhatsApp/Telegram message
        │
        ▼
  Store raw message (messages table, status: PENDING)
        │
        ▼
  parser.StartListener — polls for PENDING messages
        │
        ▼
  parser.Process(message)
        │
        ├── Step 1: extractWithRegex(content)
        │     match → parsed ride fields
        │     no match ↓
        ├── Step 2: extractWithHaiku(content, groupName)
        │     not-a-ride → mark SKIPPED
        │     error → mark FAILED
        │
        ▼
  Step 3: resolveLocation(fromText, groupID)
          resolveLocation(toText, groupID)
          (looks up location_contexts for group-specific aliases)
        │
        ▼
  Step 4: writeRide(message, parsed, fromLocationID, toLocationID)
          → inserts into rides table
          → updates message status → SUCCESS
```

## Supported Platforms

| Platform | Library | Auth Method |
|----------|---------|-------------|
| WhatsApp | [whatsmeow](https://github.com/tulir/whatsmeow) | QR code scan on first run; session resumed from DB thereafter |
| Telegram | go-telegram-bot-api (planned) | Bot token |

## Connector Lifecycle

```
NewConnectors() → creates WhatsApp client (and future connectors)
    │
    ▼
c.Start(ctx) → connects to WhatsApp, prints QR if no existing session
    │
    ▼
Listening for messages (events forwarded to handler → store → parser)
    │
    ▼
SIGINT / SIGTERM received
    │
    ▼
c.Stop() → disconnects from platform, waits for in-flight handlers (WaitGroup drain)
```

## Troubleshooting

**QR scan timeout** — The QR code expires after 60 seconds. Restart the service and scan the new QR code promptly.

**`DATABASE_URL` connection refused** — Ensure Supabase is running (`supabase start`) and that `DATABASE_URL` includes `?sslmode=disable`.

**Port 8083 already in use** — Set `PORT=<other-port>` in your environment and update `docker-compose.yml` accordingly.

**Docker build fails with "no required module"** — The Dockerfile must be built from the repo root, not from `apps/workers/`. Run: `docker build -f apps/workers/Dockerfile -t workers .` from the repo root.
