# Connector Framework

All ride data enters Project Neo through **connectors** — platform-specific
workers that write raw messages into the `messages` table. Everything past
that insert is source-agnostic: a Postgres trigger (`messages_inserted`)
notifies the parser, which extracts a ride, dedupes it by semantic
fingerprint, and links the message to the canonical ride row.

```
WhatsApp ─┐
Telegram ─┼─► messages INSERT ─► pg_notify ─► parser ─► rides (canonical)
manual  ──┘        (only contract a connector must satisfy)
```

## The contract

A connector is any type satisfying `apps/workers/internal/connector.go`:

```go
type Connector interface {
    Name() string              // health/log identity, e.g. "whatsapp"
    Run(ctx context.Context) error // blocks while live; nil = unlinked, err = failure
    Stop()                     // blocks until in-flight handlers finish
}
```

Connectors are launched by the supervisor (`StartConnectorSupervisor`), which
runs one backoff-guarded loop per **registration** (see `registrations()` in
the same file). A registration bundles the connector's name, an `Enabled()`
env gate, and a `Build()` factory. The connector is rebuilt on every loop
iteration, so session loss (e.g. a WhatsApp unlink) lands back in the pairing
flow automatically. Per-connector status strings surface in
`GET :8083/health` under `"connectors"`.

To ingest, a connector:

1. Registers its group: `GroupStore.UpsertGroup(name)` +
   `GroupSourceStore.UpsertGroupSource(groupID, sourceType, identifier)`.
2. Writes each message via `internal/store.MessageWriter.Write(...)` with the
   `group_source_id` — dedupe by content hash and the parser trigger are
   handled from there.

### `group_sources.is_active` — the kill-switch

Set `is_active = false` on a `group_sources` row to stop ingesting that group.
Connectors filter their monitored set through `ListActive` on every (re)sync,
and upserts never flip the flag back on. WhatsApp re-syncs its joined groups
every `WHATSAPP_GROUP_SYNC_INTERVAL` (default 10m), which also picks up groups
joined or renamed mid-session.

### The manual "connector"

App-created rides (`createRide` mutation) intentionally **bypass** the message
pipeline: they insert directly into `rides` with `poster_user_id` set. There
is no messages row, no parse step, and no `source_message_id` — the mutation
is the manual entry path. The `MANUAL` value in the `source_type` enum is
reserved for a future ops flow that ingests pasted text through the parser.

## Adding a platform: Telegram walkthrough (planned next)

1. **Bot setup (operator):** create a bot with @BotFather, keep the token.
   Run `/setprivacy` → **Disabled** for the bot — with privacy mode on, bots
   do not receive normal group messages. Add the bot to the target group.
2. **Package:** `apps/workers/telegram/` with a `Client` mirroring
   `whatsapp.Client`: `Name() = "telegram"`, `Run` long-polls
   `go-telegram-bot-api/v5` `GetUpdates`, `Stop` stops the poller and waits.
3. **Ingest:** for each group/supergroup message: `UpsertGroup(chat title)`,
   `UpsertGroupSource(groupID, model.SourceTypeTelegram, chatID)`, then
   `MessageWriter.Write` with `sourceMessageID = strconv.Itoa(msg.MessageID)`,
   `senderIdentifier` = username or user ID, `timestamp = msg.Time()`.
   Filter through `ListActive(model.SourceTypeTelegram)` like WhatsApp does.
4. **Register:** add a `Registration{Name: "telegram", Enabled: TELEGRAM_ENABLED
   && TELEGRAM_BOT_TOKEN != "", Build: telegram.NewClient(...)}` entry in
   `registrations()`. Add `TELEGRAM_ENABLED` / `TELEGRAM_BOT_TOKEN` to
   compose + `.env.example`.

Unlike WhatsApp, the official Bot API needs no QR pairing, no session file,
and no anti-detection measures.

# Failed-Parse Review Runbook

Regex-only mode (`OLLAMA_ENABLED=false`) marks every message the patterns
cannot handle as `FAILED` with `parse_error = "no regex pattern matched
(regex-only mode)"`. Those rows are the feedback loop for growing the parser:

1. **Review** — open Supabase Studio (http://localhost:54323) → SQL editor →
   run `supabase/snippets/parse_failures_review.sql` (or browse the
   `parse_failures` view). `parse_stats` shows the daily accuracy trend.
2. **Fix** — extend the patterns in `apps/workers/parser/regex.go` to cover
   the failing shape.
3. **Rebuild** — `docker compose up -d --build workers`.
4. **Requeue** — `select requeue_failed_messages();` (snippet
   `requeue_failed.sql`; optionally scoped to one group or one message id).
   The reset to `PENDING` re-notifies the parser live — no restart needed —
   and rows drain within seconds (`docker compose logs -f workers` shows
   `parser: ride linked`).
5. **Triage non-rides** — anything that genuinely isn't a ride gets
   `mark_not_a_ride.sql` so it stops appearing in the review view.

Safety nets behind this loop: a periodic recovery sweep re-drives stale
`PENDING` rows every `PARSER_RECOVERY_INTERVAL` (default 5m), parse fan-out is
capped by `PARSER_MAX_CONCURRENT`, and bare clock times are interpreted in
`PARSER_TIMEZONE`.
