# Message Parser Design

**Date:** 2026-04-06
**Scope:** `apps/workers/parser/`, `packages/shared-go/postgres/`, `supabase/migrations/`
**Status:** Approved

---

## Problem

Messages from WhatsApp groups are stored in the `messages` table with `parse_status = 'PENDING'`. Nothing currently reads them or creates `rides` rows. The full pipeline — capture → parse → ride — is incomplete.

---

## Goal

A real-time parser that:
1. Detects new pending messages via pg_notify
2. Extracts ride fields (type, locations, time, cost) using regex for structured messages and Claude Haiku for freeform ones
3. Resolves location text against `location_contexts` for the group
4. Creates a `rides` row on success
5. Records failure reason in `parse_error` for debugging and future improvement

---

## Architecture

The parser lives in `apps/workers/parser/` as a platform-agnostic package alongside `whatsapp/`. It is started in `run.go` alongside the WhatsApp connector. Location auto-suggestion is out of scope (separate spec).

```
WhatsApp handler stores message (parse_status='PENDING')
  → pg_notify fires 'messages_inserted' with message UUID

parser/listener.go
  → receives UUID via pg_notify
  → fetches full message row from DB
  → verifies parse_status == 'PENDING' (defensive check — skip if already processed)
  → calls extractor pipeline

parser/extractor.go (orchestrates)
  → parser/regex.go: attempt structured extraction
      → hit (ride type + at least one location found) → proceed
      → miss → parser/haiku.go: send to Claude Haiku
          → success → proceed
          → not a ride → mark SKIPPED, store parse_error, stop
          → failure → mark FAILED, store parse_error, stop

  → parser/location.go: resolve from/to text against location_contexts for group
      → match found → populate from_location_id / to_location_id
      → no match → leave IDs nil, store raw text only

  → parser/writer.go: insert rides row, update messages.parse_status
      → ride insert fires rides_added pg_notify → graphql-api subscriber notified (no extra work needed)
```

---

## Components

### `supabase/migrations/..._messages_notify_trigger.sql`

Fires on every insert into `messages`. The listener defensively checks `parse_status == 'PENDING'` after fetching the row to guard against duplicate processing.

```sql
create or replace function notify_message_inserted()
returns trigger language plpgsql as $$
begin
  perform pg_notify('messages_inserted', new.id::text);
  return new;
end;
$$;

create trigger message_inserted_trigger
  after insert on messages
  for each row execute function notify_message_inserted();
```

---

### `parser/listener.go`

Persistent `lib/pq` connection listening on `messages_inserted`. Uses `pq.NewListener` with its own DSN connection (separate from the bun query pool — same pattern as `graphql-api/internal/postgres/listener.go`).

Signature:
```go
func StartListener(ctx context.Context, databaseURL string, bunDB *bun.DB, logger *slog.Logger)
```

- `databaseURL` — passed to `pq.NewListener` for the persistent LISTEN connection
- `bunDB` — used by location.go and writer.go for queries and writes

On notification:
1. Parse UUID from payload
2. Fetch full `model.Message` from DB using `MessageStore.GetByID(ctx, id)` — see `packages/shared-go/postgres/message.go` modification below
3. **Verify `msg.ParseStatus == model.ParseStatusPending`** — skip if already handled
4. Call `extractor.Process(ctx, msg, bunDB, logger)`

---

### `parser/regex.go`

Handles structured multi-line messages. Extracts:
- **Ride type**: matches `need ride` / `ride available` (case-insensitive)
- **From/To**: matches `from [location] to [location]`
- **Time**: matches `now`, `HH:MM AM/PM`, `in N mins/hours`
- **Cost**: matches currency patterns (`$25`, `₹500`, `25 USD`)
- **Distance**: matches `N km` / `N miles`

Returns a `ParsedRide`. A result is a **hit** if ride type AND at least one location are present. Otherwise it's a **miss** — hand off to Haiku.

---

### `parser/haiku.go`

Called only when regex misses. Sends message content to `claude-haiku-4-5-20251001` with a structured system prompt:

```
You are a ride-sharing message parser. Extract ride information from the message.
Group context: {group_name}

Return ONLY a JSON object:
{
  "ride_type": "need_ride" | "ride_available" | null,
  "from_location": "string" | null,
  "to_location": "string" | null,
  "is_immediate": true | false,
  "departure_time_text": "string" | null,
  "cost": number | null,
  "currency": "string" | null,
  "not_a_ride_message": true | false
}

If this is not a ride message at all, set not_a_ride_message: true.
```

**Enum normalisation:** Haiku returns lowercase snake_case (`"need_ride"`, `"ride_available"`). The parser maps these to Go model constants (`model.RideTypeNeedRide`, `model.RideTypeRideAvailable`) after unmarshalling. Unknown values → treat as parse failure.

- If `not_a_ride_message: true` → mark `parse_status = 'SKIPPED'`, `parse_error = "not a ride message"`
- If `ride_type` is null or both locations are null → mark `parse_status = 'FAILED'`, `parse_error = "haiku: missing ride type or locations"`
- Otherwise → proceed

API key from `ANTHROPIC_API_KEY` env var. If unset → skip Haiku, mark `FAILED` with `parse_error = "ANTHROPIC_API_KEY not configured"`.

---

### `parser/location.go`

Takes raw location text (e.g. `"UCM"`, `"Airport"`) and a `group_id`. Queries the **`location_contexts`** table (not `locations`) for a case-insensitive match on `location_alias` within the group.

Returns `*uuid.UUID` — specifically `lc.ID` (the `location_contexts` primary key), **not** `lc.LocationID` (which is the FK to the `locations` table). This value maps directly to `rides.from_location_id` / `rides.to_location_id`. Returns `nil` if no alias match is found. Called independently for from and to locations. When nil, the raw text is stored in `from_location_text`/`to_location_text` on the ride row.

---

### `parser/writer.go`

Assembles and writes the result. `GroupID` and `MessageID` come from the `model.Message` passed through the pipeline (not from `ParsedRide`):

**On success:**
```go
ride := &model.Ride{
    ID:               uuid.New(),
    MessageID:        &msg.ID,
    GroupID:          msg.GroupID,
    Type:             parsed.RideType,
    FromLocationID:   fromLocationID,   // *uuid.UUID, may be nil
    ToLocationID:     toLocationID,     // *uuid.UUID, may be nil
    FromLocationText: parsed.FromLocationText,
    ToLocationText:   parsed.ToLocationText,
    DepartureTime:    parsed.DepartureTime,
    IsImmediate:      parsed.IsImmediate,
    Cost:             parsed.Cost,
    Currency:         currencyOrDefault(parsed.Currency), // defaults to "USD" if nil
    Distance:         parsed.Distance,
    Status:           model.RideStatusAvailable,
}
```

**Currency default:** `parsed.Currency` is `*string` (nullable from extraction). Writer defaults to `"USD"` when nil, matching `rides.currency NOT NULL DEFAULT 'USD'` in the schema.

Inserts ride via `shared-go/postgres` `InsertRide`. The insert automatically fires the existing `ride_added_trigger` → `pg_notify('rides_added', ...)` → graphql-api listener → GraphQL subscribers notified. No additional work needed for realtime propagation.

Then updates `messages.parse_status = 'SUCCESS'`, `messages.parsed_at = now()`.

**On failure:** updates `messages.parse_status = 'FAILED'`, `messages.parse_error = reason`

**On skip:** updates `messages.parse_status = 'SKIPPED'`, `messages.parse_error = "not a ride message"`

---

### `packages/shared-go/postgres/ride_write.go`

New file. `InsertRide(ctx context.Context, ride *model.Ride) error` — plain insert. Workers uses this concrete store directly, same pattern as `GroupStore`/`GroupSourceStore`. The graphql-api has its own `rideRepository.Create` via the interface and is unaffected.

The insert automatically fires the existing `ride_added_trigger` (confirmed in `supabase/migrations/20260404000000_notify_triggers.sql`) → `pg_notify('rides_added', ...)` → graphql-api listener → GraphQL subscribers notified.

---

### `packages/shared-go/postgres/message.go` (modify)

Add `GetByID` to the existing `MessageStore`:

```go
func (s *MessageStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Message, error)
```

Plain select by primary key. Returns `(nil, nil)` if not found (listener skips gracefully). Used by `listener.go` to fetch the full message row after receiving the pg_notify payload.

---

## ParsedRide Struct

```go
// ParsedRide holds extraction results. GroupID and MessageID are NOT here —
// they come from the model.Message row and are added by writer.go.
type ParsedRide struct {
    RideType         model.RideType  // NEED_RIDE | RIDE_AVAILABLE
    FromLocationText *string         // nil if not found; matches model.Ride.FromLocationText (*string)
    ToLocationText   *string         // nil if not found; matches model.Ride.ToLocationText (*string)
    IsImmediate      bool
    DepartureTime    *time.Time
    Cost             *float64
    Currency         *string         // nil → writer defaults to "USD"
    Distance         *float64
}
```

---

## parse_status Transitions

Using constants from `model.ParseStatus` (uppercase):

```
PENDING
  ├─ regex or haiku extracted ride type + location → SUCCESS
  ├─ extraction failed (looked like a ride but fields missing) → FAILED + parse_error
  └─ clearly not a ride message → SKIPPED + parse_error="not a ride message"
```

- `FAILED` = worth investigating to improve patterns
- `SKIPPED` = expected, no action needed

---

## Wire-up in `run.go`

Insert after `srv := startHealthServer(port, logger)` and before `waitForShutdown(cancel, connectors, srv, logger)`:

```go
// Import: "project-neo/workers/parser"
go parser.StartListener(ctx, databaseURL, bunDB, logger)
```

`databaseURL` and `bunDB` are already in scope at this call site (declared earlier in `run()`).

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ANTHROPIC_API_KEY` | No (degrades gracefully) | Claude Haiku API key. If unset, Haiku fallback is skipped and unmatched messages go to `FAILED`. |

Add `ANTHROPIC_API_KEY=` (empty) to `.env.example`, alongside the other service configuration variables (after `DATABASE_URL`).

---

## New Files

| File | Action |
|------|--------|
| `supabase/migrations/..._messages_notify_trigger.sql` | Create |
| `apps/workers/parser/listener.go` | Create |
| `apps/workers/parser/extractor.go` | Create |
| `apps/workers/parser/regex.go` | Create |
| `apps/workers/parser/haiku.go` | Create |
| `apps/workers/parser/location.go` | Create |
| `apps/workers/parser/writer.go` | Create |
| `packages/shared-go/postgres/ride_write.go` | Create |
| `packages/shared-go/postgres/message.go` | Modify (add `GetByID` method to `MessageStore`) |
| `apps/workers/run.go` | Modify (add `go parser.StartListener(...)`) |
| `.env.example` | Modify (add `ANTHROPIC_API_KEY=`) |

---

## Out of Scope

- Location auto-suggestion (separate spec)
- Re-parsing failed messages (reset `parse_status='PENDING'` manually to reprocess)
- Telegram / other source parsing (same parser handles them automatically via `messages` table)
