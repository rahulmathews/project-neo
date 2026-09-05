# MVP Implementation Plan

## Working agreement

- Do **not** create commits.
- Do **not** push branches or tags.
- Finish one phase checkpoint, verify it, and hand over exact staging and commit commands before starting the next implementation phase.
- Use conventional commits with the owner's configured Git identity; do not add co-author or generated-by footers.
- No seed or synthetic application data. Preserve the existing database and WhatsApp session; normal validation must not reset either.
- Leave every implementation change uncommitted for the repository owner to inspect and commit in deliberately small, atomic commits.
- Keep WhatsApp as the only production message source for the MVP.
- Do not expand scope into Telegram, matching, payments, chat, maps, notifications, broad profile/settings work, or other non-MVP features without explicit approval.

## MVP outcome

A fresh WhatsApp group message is ingested into `messages`, parsed into an available `rides` record, and displayed in the authenticated Flutter Ride Requests feed.

## Delivery phases

Keep phases 0–4 below. Split phase 0 into two independently reviewable checkpoints rather than accumulating backend and mobile changes together:

| Checkpoint | Scope | Exit gate |
| --- | --- | --- |
| 0A (current) | API readiness and consistent port configuration; current-state plan | Go build, API lint, formatting, rebuilt API reports database/listener health |
| 0B | Mobile launch/configuration, auth recovery and unfinished navigation | Android login reaches the feed with real groups; no unhandled session errors |
| 1 | Fresh WhatsApp message through database to mobile | One fresh real message traced through its linked ride to a visible card |
| 2 | Feed subscriptions and recovery | Add/status events, reconnect and group-switch checks pass without stale/duplicate cards |
| 3 | Parsing quality using existing failures | Categorized failures, selected fixes, and reviewed reprocessing of real messages |
| 4 | Release readiness | Project checks, Android acceptance, PR CI and release configuration reviewed |

Commit and PR guidance: one focused branch/PR per checkpoint, with small commits inside it. Stop to let the owner commit at the checkpoint. Pushing a feature branch for review is distinct from releasing to production.

## Repository and runtime baseline (September 5, 2026)

- Local `develop`: `fca8bf0`, matching GitHub `develop` when checked.
- PR #98 (backend stabilization) merged into `develop`; #100 merged `develop` into `main`; #101 completed the release. The earlier claim that the release train was stopped is obsolete.
- GitHub `develop` is three commits behind `main`, with no commits ahead. Current repo instructions and recent merges use feature branch -> `develop` -> `main`; keep that workflow for this delivery.
- Dependency PRs #102 (SQLite), #103 (Sentry), #104 (gqlparser) are open and have passing CI. Keep their upgrades separate from MVP behavior changes.
- The latest inspected `main` CI run passed. This does not validate new uncommitted changes.
- Both app containers and the core Supabase containers were running. Workers reported WhatsApp connected and parser listener ready; API reported listener ready.
- Database snapshot: 38 groups and 38 WhatsApp sources; 5,562 successful messages linked to rides, 558 skipped, 915 failed; 541 available and 2,122 expired rides. Counts change with live ingestion (a later query already showed 917 failures).
- Existing data demonstrates WhatsApp-to-database processing, not completion of fresh-message-to-mobile acceptance. Failed parses still need classification before claiming backend stability.
- Already implemented: supervised connectors/listener, bounded parsing, periodic pending-message recovery, ride expiry/revival, group resync, request/error logging, optional Sentry, complexity limits and subscription-drop metrics. Do not rebuild these from the stale TODO list.
- `AGENTS.md`, `apps/workers/whatsapp/a.js`, and this plan were untracked at the start of this session. Preserve the first two and stage only reviewed files explicitly.

### Phase 0A implementation

- API server and healthcheck share `PORT`, defaulting to `8082`.
- `/health` checks the application's SQL pool with a two-second timeout as well as listener state. A failure returns HTTP 503 and `status: degraded`.
- Successful JSON adds `database: ok`; the GraphQL `{ health }` field remains a liveness response.
- Verification is limited to build/lint/format and healthy-path runtime checks unless explicitly recorded otherwise. A database outage drill must not interrupt live ingestion as part of this checkpoint.

**Checkpoint result (September 5, 2026): ready for owner commit and feature-branch push.**

- `go build ./apps/graphql-api/... ./apps/workers/... ./packages/shared-go/...` passed.
- API `golangci-lint run --config ../../.golangci.yml ./...` passed.
- `gofumpt -l apps/graphql-api/main.go` produced no output; `git diff --check` passed.
- `docker compose up -d --build graphql-api` succeeded. Docker reports both application containers healthy.
- Rebuilt API returned HTTP 200 with `database: ok`, `listener: ok`, `status: ok`.
- Workers still returned HTTP 200 with WhatsApp connected and parser listener ready.
- No database reset, seeding, WhatsApp session reset, commits, or pushes were performed in this checkpoint.

Suggested branch: `fix/api-readiness`. Commit API code + API README together as
`fix(graphql-api): check database readiness and align service ports`.
Commit `TODO.md` + this plan together as
`docs: define phased MVP delivery checkpoints`.
Push that branch and open a PR into `develop`; require its CI to pass before
merging. Start phase 0B after this checkpoint is committed/reviewed.

### 0. Make the existing vertical slice runnable

- Configure Android **development-only** access to the documented local HTTP Supabase and GraphQL endpoints; production remains HTTPS-only.
- Align the GraphQL API's direct-run port with Compose and mobile configuration (`8082`), or make the required override explicit everywhere.
- Make Ride Requests the polished landing experience and hide or clearly mark unfinished navigation destinations.

**Completion criteria:** an Android emulator can authenticate locally, query groups and rides through GraphQL, and reach the Ride Requests feed without entering unfinished features.

### 1. Prove WhatsApp ingestion end to end

- Document the operator sequence: migrations, local services, WhatsApp QR pairing, group membership, valid fresh message, database inspection, and mobile verification.
- Document accepted ride-message patterns based on the parser's actual regex support.
- Use fresh real WhatsApp messages for acceptance. If pairing blocks validation, record the blocker and keep the existing data; do not substitute seeded messages or rides.

**Completion criteria:** a new WhatsApp message creates a successful `messages` row linked to an available `rides` row, and the selected group shows its request card.

### 2. Improve feed behavior

- Consume existing `rideAdded` and `rideStatusChanged` GraphQL subscriptions in the Flutter feed.
- Handle reconnection and selected-group changes without duplicate or stale cards.
- Retain pull-to-refresh as recovery and keep loading, empty, authentication, and error states clear.

**Completion criteria:** while the feed is open, newly parsed requests appear without manual refresh and no-longer-available requests disappear.

### 3. Harden parsing with real messages

- Review parse failures from actual target groups; prioritize regex coverage before increasing LLM dependence.
- Persist Ollama fallback `departure_time_text` using the same configured timezone semantics as regex parsing.
- Decide and document the treatment of incomplete routes: reject, send to review, or present a limited card.
- Use the existing failure-review and requeue workflow to make parser operations observable.

**Completion criteria:** representative WhatsApp messages consistently yield usable origin/destination/time data; failures are observable and recoverable.

### 4. Release readiness

- Run project format, lint, and build checks.
- Execute a documented manual Android end-to-end validation.
- Ensure local HTTP allowances are development-only and production configuration uses HTTPS.

## Evidence-based loose ends to address

- Android cleartext development policy already exists in `src/debug/AndroidManifest.xml`; verify the merged debug manifest and release HTTPS configuration in phase 0B/4.
- `apps/mobile/lib/features/rides/screens/rides_screen.dart`: uses queries/manual refresh but not already-implemented subscriptions.
- `apps/workers/internal/connector.go`: a connector registry/supervisor is implemented; WhatsApp remains the only concrete MVP source. Keep operations guidance current.
- `apps/workers/parser/ollama.go`: receives `departure_time_text` but does not map it to `ParsedRide.DepartureTime`.
- `apps/mobile/lib/core/widgets/app_drawer.dart`: exposes unfinished non-MVP screens.
- `apps/graphql-api/main.go`: phase 0A aligns the direct-run and healthcheck ports and adds SQL-pool readiness.

## Validation approach

No test files, test commands, or test targets should be added. Use the project checks and a manual vertical-slice validation during implementation:

```bash
git status --short
bun run format:check
bun run lint
cd apps/graphql-api && go build .
cd apps/workers && go build .
cd apps/mobile && dart format --output=none --set-exit-if-changed lib/
cd apps/mobile && flutter analyze
```

Then pair WhatsApp, send a supported fresh ride request, verify the `messages` and `rides` records in Supabase Studio, and confirm the resulting card in the Android app.
