# Message Parser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a real-time parser that listens for new `messages` rows via pg_notify, extracts ride data (regex first, Claude Haiku fallback), resolves location aliases, and creates `rides` rows.

**Architecture:** A SQL trigger fires `pg_notify('messages_inserted', id)` on every `messages` insert. `parser/listener.go` holds a persistent `lib/pq` LISTEN connection, fetches the message, checks it is still `PENDING`, and calls `extractor.Process`. The extractor tries regex, falls back to Haiku if needed, resolves locations against `location_contexts`, then writes the ride and updates `parse_status`.

**Tech Stack:** Go 1.25, `github.com/lib/pq` (LISTEN/NOTIFY), `github.com/anthropics/anthropic-sdk-go` (Claude Haiku), bun ORM, PostgreSQL (Supabase local)

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `supabase/migrations/20260406000000_messages_notify_trigger.sql` | Create | Fires `pg_notify('messages_inserted', id)` on every messages insert |
| `packages/shared-go/postgres/message.go` | Modify | Add `GetByID(ctx, id)` to `MessageStore` |
| `packages/shared-go/postgres/ride_write.go` | Create | `RideStore` with `InsertRide` — plain insert, fires existing `ride_added_trigger` |
| `apps/workers/parser/types.go` | Create | `ParsedRide` struct; `ErrNotARide` sentinel |
| `apps/workers/parser/regex.go` | Create | Structured extraction — ride type, from/to, time, cost, distance |
| `apps/workers/parser/haiku.go` | Create | Claude Haiku fallback; unmarshals JSON response into `ParsedRide` |
| `apps/workers/parser/location.go` | Create | Case-insensitive alias lookup in `location_contexts` → returns `*uuid.UUID` |
| `apps/workers/parser/writer.go` | Create | Inserts ride row; updates `messages.parse_status` on success/fail/skip |
| `apps/workers/parser/extractor.go` | Create | Orchestrates: regex → haiku → location → writer |
| `apps/workers/parser/listener.go` | Create | `StartListener` — persistent LISTEN loop, dispatches to extractor |
| `apps/workers/run.go` | Modify | Add `go parser.StartListener(ctx, databaseURL, bunDB, logger)` |
| `.env.example` | Modify | Add `ANTHROPIC_API_KEY=` |

---

## Task 1: Add `messages_inserted` pg_notify trigger

**Files:**
- Create: `supabase/migrations/20260406000000_messages_notify_trigger.sql`

- [ ] **Step 1: Create the migration file**

```sql
-- supabase/migrations/20260406000000_messages_notify_trigger.sql

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

- [ ] **Step 2: Apply the migration locally**

```bash
supabase migration up
```

Expected output: `Applying migration 20260406000000_messages_notify_trigger.sql... done`

If Supabase is not running: `supabase start` first.

- [ ] **Step 3: Commit**

```bash
git add supabase/migrations/20260406000000_messages_notify_trigger.sql
git commit -m "feat(database): add messages_inserted pg_notify trigger"
```

---

## Task 2: Add `MessageStore.GetByID` to shared-go

**Files:**
- Modify: `packages/shared-go/postgres/message.go`

`listener.go` will call this to fetch the full message row by UUID after receiving a pg_notify payload.

- [ ] **Step 1: Append `GetByID` to the end of `packages/shared-go/postgres/message.go`**

```go
// GetByID fetches a message by primary key. Returns (nil, nil) if not found.
func (s *MessageStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Message, error) {
	msg := new(model.Message)
	err := s.db.NewSelect().
		Model(msg).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get message by id: %w", err)
	}
	return msg, nil
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
cd packages/shared-go && go build ./...
```

Expected: no output (clean build).

- [ ] **Step 3: Commit**

```bash
git add packages/shared-go/postgres/message.go
git commit -m "feat(shared): add MessageStore.GetByID for parser listener"
```

---

## Task 3: Create `RideStore.InsertRide` in shared-go

**Files:**
- Create: `packages/shared-go/postgres/ride_write.go`

A concrete store for inserting rides — same pattern as `GroupStore`/`GroupSourceStore`. The insert automatically fires the existing `ride_added_trigger` in `20260404000000_notify_triggers.sql`, which notifies the graphql-api.

- [ ] **Step 1: Create `packages/shared-go/postgres/ride_write.go`**

```go
package postgres

import (
	"context"
	"fmt"

	"project-neo/shared/model"

	"github.com/uptrace/bun"
)

// RideStore is a concrete store for ride inserts (used by workers).
type RideStore struct {
	db *bun.DB
}

func NewRideStore(db *bun.DB) *RideStore {
	return &RideStore{db: db}
}

// InsertRide inserts a new ride row. The insert fires ride_added_trigger →
// pg_notify('rides_added', ...) → graphql-api subscribers notified automatically.
func (s *RideStore) InsertRide(ctx context.Context, ride *model.Ride) error {
	_, err := s.db.NewInsert().Model(ride).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert ride: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
cd packages/shared-go && go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add packages/shared-go/postgres/ride_write.go
git commit -m "feat(shared): add RideStore.InsertRide for workers parser"
```

---

## Task 4: Create `parser/types.go`

**Files:**
- Create: `apps/workers/parser/types.go`

Defines `ParsedRide` (shared across all parser files) and the `ErrNotARide` sentinel used by `haiku.go` to signal a non-ride message without treating it as a real error.

- [ ] **Step 1: Create `apps/workers/parser/types.go`**

```go
package parser

import (
	"errors"
	"time"

	"project-neo/shared/model"
)

// ErrNotARide is returned by extractWithHaiku when the message is clearly not
// a ride request or offer. The extractor marks parse_status = SKIPPED.
var ErrNotARide = errors.New("not a ride message")

// ParsedRide holds extraction results. GroupID and MessageID are NOT stored here —
// they come from the model.Message row passed through the pipeline.
type ParsedRide struct {
	RideType         model.RideType // NEED_RIDE | RIDE_AVAILABLE
	FromLocationText *string        // nil if not found; stored in rides.from_location_text
	ToLocationText   *string        // nil if not found; stored in rides.to_location_text
	IsImmediate      bool
	DepartureTime    *time.Time
	Cost             *float64
	Currency         *string  // nil → writer defaults to "USD"
	Distance         *float64
}
```

- [ ] **Step 2: Build workers to verify**

```bash
cd apps/workers && go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add apps/workers/parser/types.go
git commit -m "feat(workers): add ParsedRide struct and ErrNotARide sentinel"
```

---

## Task 5: Create `parser/regex.go`

**Files:**
- Create: `apps/workers/parser/regex.go`

Attempts structured extraction from multi-line ride messages. Returns a **hit** when ride type AND at least one location are found; otherwise a **miss** (hand off to Haiku).

- [ ] **Step 1: Create `apps/workers/parser/regex.go`**

```go
package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"project-neo/shared/model"
)

var (
	rideTypeRe = regexp.MustCompile(`(?i)\b(need\s+ride|ride\s+available)\b`)
	fromToRe   = regexp.MustCompile(`(?i)from\s+(.+?)\s+to\s+(.+?)(?:\n|$)`)
	nowRe      = regexp.MustCompile(`(?i)\bnow\b`)
	timeRe     = regexp.MustCompile(`(?i)\b(\d{1,2}:\d{2}\s*(?:AM|PM))\b`)
	inTimeRe   = regexp.MustCompile(`(?i)\bin\s+\d+\s*(?:min|mins|minutes|hour|hours|hr|hrs)\b`)
	costRe     = regexp.MustCompile(`(?i)(?:[$₹£€])(\d+(?:\.\d{1,2})?)|(\d+(?:\.\d{1,2})?)\s*(?:USD|INR|GBP|EUR)`)
	distanceRe = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:km|miles|mi)\b`)
)

// extractWithRegex attempts structured extraction from content.
// Returns (parsed, true) on a hit (ride type + at least one location found),
// or (nil, false) on a miss.
func extractWithRegex(content string) (*ParsedRide, bool) {
	parsed := &ParsedRide{}

	// Ride type
	if m := rideTypeRe.FindString(content); m != "" {
		if strings.Contains(strings.ToLower(m), "need") {
			parsed.RideType = model.RideTypeNeedRide
		} else {
			parsed.RideType = model.RideTypeRideAvailable
		}
	}

	// From / To locations
	if m := fromToRe.FindStringSubmatch(content); len(m) == 3 {
		from := strings.TrimSpace(m[1])
		to := strings.TrimSpace(m[2])
		parsed.FromLocationText = &from
		parsed.ToLocationText = &to
	}

	// Departure time
	if nowRe.MatchString(content) {
		parsed.IsImmediate = true
	} else if inTimeRe.MatchString(content) {
		parsed.IsImmediate = false
		// Relative time ("in 30 mins") — no absolute DepartureTime stored
	} else if m := timeRe.FindString(content); m != "" {
		parsed.IsImmediate = false
		for _, layout := range []string{"3:04 PM", "3:04PM"} {
			if t, err := time.Parse(layout, strings.TrimSpace(m)); err == nil {
				now := time.Now()
				dep := time.Date(now.Year(), now.Month(), now.Day(),
					t.Hour(), t.Minute(), 0, 0, now.Location())
				parsed.DepartureTime = &dep
				break
			}
		}
	}

	// Cost
	if m := costRe.FindStringSubmatch(content); m != nil {
		raw := m[1]
		if raw == "" {
			raw = m[2]
		}
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			parsed.Cost = &v
		}
	}

	// Distance
	if m := distanceRe.FindStringSubmatch(content); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			parsed.Distance = &v
		}
	}

	// Hit = ride type AND at least one location
	hit := parsed.RideType != "" &&
		(parsed.FromLocationText != nil || parsed.ToLocationText != nil)
	return parsed, hit
}
```

- [ ] **Step 2: Build workers**

```bash
cd apps/workers && go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add apps/workers/parser/regex.go
git commit -m "feat(workers): add regex-based ride message extractor"
```

---

## Task 6: Add Anthropic SDK dependency and create `parser/haiku.go`

**Files:**
- Modify: `apps/workers/go.mod` (via `go get`)
- Create: `apps/workers/parser/haiku.go`

Called only when regex misses. Uses `claude-haiku-4-5-20251001` to parse freeform messages. Returns `ErrNotARide` for clearly non-ride messages so the extractor marks them `SKIPPED` rather than `FAILED`.

- [ ] **Step 1: Add the Anthropic SDK to workers**

```bash
cd apps/workers && go get github.com/anthropics/anthropic-sdk-go@latest
```

Expected: `go.mod` and `go.sum` updated with the anthropic-sdk-go entry.

- [ ] **Step 2: Create `apps/workers/parser/haiku.go`**

```go
package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"project-neo/shared/model"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type haikuResponse struct {
	RideType          *string  `json:"ride_type"`
	FromLocation      *string  `json:"from_location"`
	ToLocation        *string  `json:"to_location"`
	IsImmediate       bool     `json:"is_immediate"`
	DepartureTimeText *string  `json:"departure_time_text"`
	Cost              *float64 `json:"cost"`
	Currency          *string  `json:"currency"`
	NotARideMessage   bool     `json:"not_a_ride_message"`
}

// extractWithHaiku calls Claude Haiku to parse a freeform message.
// Returns ErrNotARide if the message is not a ride request/offer.
// Returns an error wrapping "ANTHROPIC_API_KEY not configured" if the key is missing.
func extractWithHaiku(ctx context.Context, content string, groupName string, logger *slog.Logger) (*ParsedRide, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not configured")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	systemPrompt := fmt.Sprintf(`You are a ride-sharing message parser. Extract ride information from the message.
Group context: %s

Return ONLY a JSON object with no markdown formatting:
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

If this is not a ride message at all, set not_a_ride_message: true.`, groupName)

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F[anthropic.Model]("claude-haiku-4-5-20251001"),
		MaxTokens: anthropic.F(int64(512)),
		System: anthropic.F([]anthropic.TextBlockParam{{
			Type: anthropic.F(anthropic.TextBlockParamTypeText),
			Text: anthropic.F(systemPrompt),
		}}),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(content)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("haiku api: %w", err)
	}

	// Extract text from first content block
	var raw string
	for _, block := range resp.Content {
		if block.Type == anthropic.ContentBlockTypeText {
			raw = block.Text
			break
		}
	}

	var hr haikuResponse
	if err := json.Unmarshal([]byte(raw), &hr); err != nil {
		return nil, fmt.Errorf("haiku: unmarshal response: %w", err)
	}

	if hr.NotARideMessage {
		return nil, ErrNotARide
	}

	if hr.RideType == nil || (hr.FromLocation == nil && hr.ToLocation == nil) {
		return nil, fmt.Errorf("haiku: missing ride type or locations")
	}

	parsed := &ParsedRide{
		IsImmediate:      hr.IsImmediate,
		Cost:             hr.Cost,
		Currency:         hr.Currency,
		FromLocationText: hr.FromLocation,
		ToLocationText:   hr.ToLocation,
	}

	switch *hr.RideType {
	case "need_ride":
		parsed.RideType = model.RideTypeNeedRide
	case "ride_available":
		parsed.RideType = model.RideTypeRideAvailable
	default:
		return nil, fmt.Errorf("haiku: unknown ride_type %q", *hr.RideType)
	}

	return parsed, nil
}
```

- [ ] **Step 3: Build workers to verify SDK resolves**

```bash
cd apps/workers && go build ./...
```

Expected: no output. If the Anthropic SDK API shape differs from the version installed, you may see type errors. The key patterns to check:
- `anthropic.F[anthropic.Model](...)` — wraps the model string
- `anthropic.F(int64(...))` — wraps MaxTokens
- `anthropic.TextBlockParamTypeText` — the type constant
- `block.Text` and `block.Type` on content blocks

Consult `go doc github.com/anthropics/anthropic-sdk-go` if needed.

- [ ] **Step 4: Commit**

```bash
git add apps/workers/go.mod apps/workers/go.sum apps/workers/parser/haiku.go
git commit -m "feat(workers): add Claude Haiku fallback extractor"
```

---

## Task 7: Create `parser/location.go`

**Files:**
- Create: `apps/workers/parser/location.go`

Queries `location_contexts` for a case-insensitive match on `location_alias` within the group. Returns `lc.ID` (the `location_contexts` primary key — **not** `lc.location_id`, which is the FK to the separate `locations` table). Returns `nil` if no alias matches.

- [ ] **Step 1: Create `apps/workers/parser/location.go`**

```go
package parser

import (
	"context"
	"log/slog"

	"project-neo/shared/model"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// resolveLocation looks up locationText in the location_contexts table for
// the given group. Returns lc.ID (the location_contexts PK) if found, nil otherwise.
// A nil result means the raw text will be stored as-is on the ride row.
func resolveLocation(ctx context.Context, db *bun.DB, locationText *string, groupID uuid.UUID, logger *slog.Logger) *uuid.UUID {
	if locationText == nil || *locationText == "" {
		return nil
	}

	var lc model.LocationContext
	err := db.NewSelect().
		Model(&lc).
		Where("group_id = ?", groupID).
		Where("LOWER(location_alias) = LOWER(?)", *locationText).
		Limit(1).
		Scan(ctx)
	if err != nil {
		// No match found or query error — leave ID nil, store raw text
		if err.Error() != "sql: no rows in result set" {
			logger.Warn("location resolve error", "text", *locationText, "group_id", groupID, "error", err)
		}
		return nil
	}

	return &lc.ID
}
```

- [ ] **Step 2: Build workers**

```bash
cd apps/workers && go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add apps/workers/parser/location.go
git commit -m "feat(workers): add location context resolver for parser"
```

---

## Task 8: Create `parser/writer.go`

**Files:**
- Create: `apps/workers/parser/writer.go`

Inserts the ride row and updates `messages.parse_status`. Uses `RideStore.InsertRide` (which fires `ride_added_trigger` automatically). All three status update paths (success, failed, skipped) live here.

- [ ] **Step 1: Create `apps/workers/parser/writer.go`**

```go
package parser

import (
	"context"
	"log/slog"
	"time"

	sharedpostgres "project-neo/shared/postgres"
	"project-neo/shared/model"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// writeRide assembles a model.Ride from the parsed result, inserts it, and
// marks the message as SUCCESS. On any error, marks FAILED.
func writeRide(
	ctx context.Context,
	db *bun.DB,
	msg *model.Message,
	parsed *ParsedRide,
	fromLocationID *uuid.UUID,
	toLocationID *uuid.UUID,
	logger *slog.Logger,
) {
	ride := &model.Ride{
		ID:               uuid.New(),
		MessageID:        &msg.ID,
		GroupID:          msg.GroupID,
		Type:             parsed.RideType,
		FromLocationID:   fromLocationID,
		ToLocationID:     toLocationID,
		FromLocationText: parsed.FromLocationText,
		ToLocationText:   parsed.ToLocationText,
		DepartureTime:    parsed.DepartureTime,
		IsImmediate:      parsed.IsImmediate,
		Cost:             parsed.Cost,
		Currency:         currencyOrDefault(parsed.Currency),
		Distance:         parsed.Distance,
		Status:           model.RideStatusAvailable,
	}

	rideStore := sharedpostgres.NewRideStore(db)
	if err := rideStore.InsertRide(ctx, ride); err != nil {
		logger.Error("writer: insert ride", "msg_id", msg.ID, "error", err)
		markFailed(ctx, db, msg.ID, "ride insert failed: "+err.Error(), logger)
		return
	}

	markSuccess(ctx, db, msg.ID, logger)
	logger.Info("parser: ride created", "ride_id", ride.ID, "msg_id", msg.ID, "type", ride.Type)
}

func markSuccess(ctx context.Context, db *bun.DB, msgID uuid.UUID, logger *slog.Logger) {
	now := time.Now()
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("parse_status = ?", model.ParseStatusSuccess).
		Set("parsed_at = ?", now).
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: mark success", "msg_id", msgID, "error", err)
	}
}

func markFailed(ctx context.Context, db *bun.DB, msgID uuid.UUID, reason string, logger *slog.Logger) {
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("parse_status = ?", model.ParseStatusFailed).
		Set("parse_error = ?", reason).
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: mark failed", "msg_id", msgID, "error", err)
	}
}

func markSkipped(ctx context.Context, db *bun.DB, msgID uuid.UUID, logger *slog.Logger) {
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("parse_status = ?", model.ParseStatusSkipped).
		Set("parse_error = ?", "not a ride message").
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: mark skipped", "msg_id", msgID, "error", err)
	}
}

func currencyOrDefault(c *string) string {
	if c == nil || *c == "" {
		return "USD"
	}
	return *c
}
```

- [ ] **Step 2: Build workers**

```bash
cd apps/workers && go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add apps/workers/parser/writer.go
git commit -m "feat(workers): add parser writer — ride insert and message status updates"
```

---

## Task 9: Create `parser/extractor.go`

**Files:**
- Create: `apps/workers/parser/extractor.go`

Orchestrates the full pipeline: regex → haiku fallback → location resolution → write. Handles all three outcome paths (success, failed, skipped). Fetches group name for Haiku context.

- [ ] **Step 1: Create `apps/workers/parser/extractor.go`**

```go
package parser

import (
	"context"
	"errors"
	"log/slog"

	"project-neo/shared/model"

	"github.com/uptrace/bun"
)

// Process runs the full extraction pipeline for a single PENDING message.
// It is safe to call concurrently — each call operates on its own message ID.
func Process(ctx context.Context, msg *model.Message, db *bun.DB, logger *slog.Logger) {
	// Fetch group name for Haiku context (best-effort — empty string on failure)
	groupName := fetchGroupName(ctx, db, msg)

	// Step 1: try regex extraction
	parsed, hit := extractWithRegex(msg.Content)
	if !hit {
		// Step 2: regex miss → try Haiku
		var err error
		parsed, err = extractWithHaiku(ctx, msg.Content, groupName, logger)
		if err != nil {
			if errors.Is(err, ErrNotARide) {
				logger.Info("parser: skipped (not a ride)", "msg_id", msg.ID)
				markSkipped(ctx, db, msg.ID, logger)
				return
			}
			logger.Warn("parser: extraction failed", "msg_id", msg.ID, "error", err)
			markFailed(ctx, db, msg.ID, err.Error(), logger)
			return
		}
	}

	// Step 3: resolve locations
	fromID := resolveLocation(ctx, db, parsed.FromLocationText, msg.GroupID, logger)
	toID := resolveLocation(ctx, db, parsed.ToLocationText, msg.GroupID, logger)

	// Step 4: write ride + update message status
	writeRide(ctx, db, msg, parsed, fromID, toID, logger)
}

// fetchGroupName queries the group name for Haiku context. Returns empty string on error.
func fetchGroupName(ctx context.Context, db *bun.DB, msg *model.Message) string {
	var g model.Group
	if err := db.NewSelect().
		Model(&g).
		Column("name").
		Where("id = ?", msg.GroupID).
		Scan(ctx); err != nil {
		return ""
	}
	return g.Name
}
```

- [ ] **Step 2: Build workers**

```bash
cd apps/workers && go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add apps/workers/parser/extractor.go
git commit -m "feat(workers): add parser extractor pipeline (regex → haiku → location → write)"
```

---

## Task 10: Add `lib/pq` dependency and create `parser/listener.go`

**Files:**
- Modify: `apps/workers/go.mod` (via `go get`)
- Create: `apps/workers/parser/listener.go`

The listener holds a persistent `lib/pq` LISTEN connection (separate from the bun pool, same pattern as `graphql-api/internal/postgres/listener.go`). On each notification it fetches the message, confirms it is still `PENDING`, and calls `Process` in a goroutine.

- [ ] **Step 1: Add `lib/pq` to workers**

```bash
cd apps/workers && go get github.com/lib/pq
```

Expected: `go.mod` and `go.sum` updated.

- [ ] **Step 2: Create `apps/workers/parser/listener.go`**

```go
package parser

import (
	"context"
	"log/slog"
	"time"

	sharedpostgres "project-neo/shared/postgres"
	"project-neo/shared/model"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/uptrace/bun"
)

// StartListener opens a persistent LISTEN connection on 'messages_inserted' and
// dispatches each notification to the extractor pipeline. Blocks until ctx is cancelled.
func StartListener(ctx context.Context, databaseURL string, bunDB *bun.DB, logger *slog.Logger) {
	msgStore := sharedpostgres.NewMessageStore(bunDB)

	listener := pq.NewListener(databaseURL, 10*time.Second, time.Minute,
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				logger.Error("parser pg listener event", "event", ev, "error", err)
			}
		},
	)
	defer func() {
		if err := listener.Close(); err != nil {
			logger.Error("parser listener close", "error", err)
		}
	}()

	if err := listener.Listen("messages_inserted"); err != nil {
		logger.Error("parser listener: listen failed", "error", err)
		return
	}
	logger.Info("message parser listener started")

	for {
		select {
		case <-ctx.Done():
			return
		case n := <-listener.Notify:
			if n == nil {
				// nil means the connection was re-established after a drop — safe to continue
				continue
			}
			id, err := uuid.Parse(n.Extra)
			if err != nil {
				logger.Warn("parser listener: invalid uuid payload", "payload", n.Extra)
				continue
			}
			go handleNotification(ctx, id, msgStore, bunDB, logger)
		}
	}
}

func handleNotification(ctx context.Context, id uuid.UUID, msgStore *sharedpostgres.MessageStore, db *bun.DB, logger *slog.Logger) {
	msg, err := msgStore.GetByID(ctx, id)
	if err != nil {
		logger.Error("parser listener: fetch message", "id", id, "error", err)
		return
	}
	if msg == nil {
		return // message not found — skip
	}
	if msg.ParseStatus != model.ParseStatusPending {
		return // already handled (defensive check)
	}
	Process(ctx, msg, db, logger)
}
```

- [ ] **Step 3: Build workers**

```bash
cd apps/workers && go build ./...
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add apps/workers/go.mod apps/workers/go.sum apps/workers/parser/listener.go
git commit -m "feat(workers): add pg_notify listener for message parser"
```

---

## Task 11: Wire up in `run.go` and update `.env.example`

**Files:**
- Modify: `apps/workers/run.go`
- Modify: `.env.example`

Starts the parser listener as a background goroutine alongside the WhatsApp connector. `databaseURL` and `bunDB` are already in scope at the call site.

- [ ] **Step 1: Add import and goroutine to `apps/workers/run.go`**

Add `"project-neo/workers/parser"` to the imports block. Then insert the goroutine between `srv := startHealthServer(port, logger)` and `waitForShutdown(cancel, connectors, srv, logger)`:

```go
// in run() function, between startHealthServer and waitForShutdown:
go parser.StartListener(ctx, databaseURL, bunDB, logger)
```

The full `run()` function after the change:

```go
func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	bunDB, sqlDB, err := initDB(databaseURL)
	if err != nil {
		return fmt.Errorf("database init: %w", err)
	}
	defer func() {
		if closeErr := bunDB.Close(); closeErr != nil {
			logger.Error("failed to close database", "error", closeErr)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connectors, err := buildConnectors(ctx, bunDB, sqlDB, logger)
	if err != nil {
		return fmt.Errorf("build connectors: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	srv := startHealthServer(port, logger)

	go parser.StartListener(ctx, databaseURL, bunDB, logger)

	waitForShutdown(cancel, connectors, srv, logger)
	return nil
}
```

- [ ] **Step 2: Add `ANTHROPIC_API_KEY` to `.env.example`**

Add the following line after the `DATABASE_URL` line in `.env.example`:

```
# Claude Haiku API key — used by message parser for freeform messages
# If unset, unmatched messages are marked FAILED instead of parsed by AI
ANTHROPIC_API_KEY=
```

- [ ] **Step 3: Build workers**

```bash
cd apps/workers && go build ./...
```

Expected: no output (clean build).

- [ ] **Step 4: Run lint**

```bash
make lint-go
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add apps/workers/run.go .env.example
git commit -m "feat(workers): wire up message parser listener in run.go"
```

---

## Task 12: End-to-End Smoke Test

Verify the full pipeline works in Docker.

- [ ] **Step 1: Rebuild and restart workers**

```bash
docker compose up -d --build workers
docker compose logs -f workers
```

Expected log lines:
```
level=INFO msg="message parser listener started"
level=INFO msg="whatsapp connector started" groups=N
```

- [ ] **Step 2: Check a PENDING message gets parsed**

In Supabase Studio (`http://localhost:54323`) or psql, check messages have been processed:

```sql
SELECT id, content, parse_status, parse_error
FROM messages
ORDER BY created_at DESC
LIMIT 10;
```

Expected: messages with `parse_status = 'SUCCESS'` or `'SKIPPED'` (not `'PENDING'` indefinitely).

- [ ] **Step 3: Check a ride was created**

```sql
SELECT id, group_id, type, from_location_text, to_location_text, status
FROM rides
ORDER BY created_at DESC
LIMIT 5;
```

Expected: ride rows created from parsed messages.

- [ ] **Step 4: Check SKIPPED messages**

Messages that are clearly not ride-related should have:
- `parse_status = 'SKIPPED'`
- `parse_error = 'not a ride message'`
