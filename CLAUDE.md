# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Project Neo** is a ride-sharing mobile and web application similar to Uber/DoorDash, with a unique data sourcing mechanism: ride requests and offers are extracted from messaging platform groups (WhatsApp, Telegram, etc.) using automated workers.

### Key Differentiators
- **Multi-platform message sourcing**: Messages from groups/channels (WhatsApp, Telegram, etc.) are parsed to extract ride information
- **Location context awareness**: Certain locations have specific meanings based on group context
- **Local-first architecture**: Avoiding cloud providers (AWS, etc.) due to IP blocking concerns
- **Self-hosted infrastructure**: Running on local machines initially, with VPS option for later

## Tech Stack

### Mobile (Priority)
- **Framework**: Flutter
- **Platforms**: iOS + Android
- **Language**: Dart

### Web
- To be determined (Flutter Web or separate React/Next.js stack)

### Backend Services
- **Database**: Supabase (local instance) - PostgreSQL + PostgREST + Realtime
- **API Protocol**: GraphQL
- **Message Source Automation**: Multi-platform support (WhatsApp, Telegram, etc.)
  - WhatsApp: whatsapp-web.js or baileys (⚠️ violates WhatsApp ToS, use at own risk)
  - Telegram: Telegram Bot API or MTProto
  - Other platforms can be added via workers
  - Runs on local machine initially, self-hosted VPS later

### Monorepo
- **Tool**: Turborepo
- **Package Manager**: (to be determined - pnpm recommended for monorepos)

## Architecture

### High-Level Components

```
┌─────────────────────────────────────────────┐
│  Message Sources (WhatsApp, Telegram, etc.) │
└──────────┬──────────────────────────────────┘
           │
           ▼
┌─────────────────────┐
│      Workers        │
│  (Node.js Service)  │
│  - Listen/Poll      │
│  - Parse messages   │
│  - Extract rides    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  GraphQL API        │
│  (Express/Fastify)  │
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

### Monorepo Structure (Planned)

```
project-neo/
├── apps/
│   ├── mobile/              # Flutter mobile app
│   ├── web/                 # Web app (future)
│   └── workers/             # WhatsApp automation service (Node.js)
├── packages/
│   ├── graphql-api/         # GraphQL server
│   ├── database/            # Supabase schemas, migrations
│   ├── shared-types/        # Shared TypeScript types
│   └── dart-models/         # Shared Dart models (if needed)
├── supabase/                # Supabase local configuration
├── turbo.json              # Turborepo configuration
└── CLAUDE.md               # This file
```

## Message Format

Messages in groups follow patterns like:
- **Timing**: "now" or specific time (e.g., "3:30 PM", "in 30 mins")
- **Type**: "Need ride" or "Ride available"
- **Route**: "from [Location A] to [Location B]"
- **Cost**: Price in local currency
- **Distance**: (Optional) "X km" or "X miles"
- **Location Context**: Group-specific location meanings

Example message:
```
Need ride now
From Downtown to Airport
$25
15km
```

### Parsing Strategy
- Use NLP/regex to extract structured data
- Map location names to coordinates (using location context)
- Store raw message + parsed fields
- Handle variations in message format

## Development Phases

### Phase 1 (MVP - Current Priority)
**Goal**: Parse WhatsApp messages and store rides locally with basic matching

Features:
- [ ] Workers service (local)
- [ ] Message parsing engine
- [ ] Local Supabase setup
- [ ] GraphQL API for CRUD operations
- [ ] Flutter mobile app with:
  - [ ] Ride listing (available rides)
  - [ ] Ride requests (need ride)
  - [ ] Basic user matching (riders ↔ drivers)
  - [ ] Real-time updates (new rides appear live)
- [ ] Location context configuration per group

### Phase 2 (Planned)
- Enhanced user profiles
- In-app messaging
- Route optimization
- Historical ride data and analytics
- Payment integration (local/cash initially)

### Phase 3 (Planned)
- Web application
- Advanced matching algorithms
- Driver ratings and reviews
- Multiple payment methods

### Phase 4+ (Future)
- VPS deployment for workers
- Push notifications
- Trip tracking and live location
- Multi-language support

## Database Schema (Conceptual)

### Core Tables

**`users`** - User profiles (riders + drivers)
- `id` (uuid, primary key)
- `email`, `phone`, `name`
- `role` (rider | driver | both)
- `avatar_url`
- `created_at`, `updated_at`

**`groups`** - Group/community information
- `id` (uuid, primary key)
- `name` (e.g., "Downtown Riders", "Airport Shuttle Group")
- `description`
- `is_active` (boolean)
- `created_at`, `updated_at`

**`group_sources`** - Data source configuration for each group
- `id` (uuid, primary key)
- `group_id` (foreign key → groups)
- `source_type` (whatsapp | telegram | discord | slack | manual)
- `source_identifier` (group ID, chat ID, channel ID, etc.)
- `last_parsed_at` (timestamp of last successful parse)
- `parse_frequency` (in minutes, for polling)
- `metadata` (jsonb - source-specific config like credentials, webhooks, etc.)
- `is_active` (boolean)
- `created_at`, `updated_at`

**`location_contexts`** - Location name mappings per group
- `id` (uuid, primary key)
- `group_id` (foreign key → groups)
- `location_alias` (e.g., "Station", "Mall", "Airport")
- `location_name` (full name, e.g., "Central Train Station")
- `latitude`, `longitude`
- `address` (optional)
- `metadata` (jsonb - additional info like landmarks, zones)
- `created_at`, `updated_at`

**`messages`** - Raw messages before parsing
- `id` (uuid, primary key)
- `group_id` (foreign key → groups)
- `source_message_id` (original message ID from platform)
- `sender_identifier` (phone, username, user ID from source)
- `content` (raw message text)
- `timestamp` (when message was sent)
- `parsed_at` (nullable, when it was processed)
- `parse_status` (pending | success | failed | skipped)
- `parse_error` (nullable, error details if parsing failed)
- `metadata` (jsonb - platform-specific data)
- `created_at`

**`rides`** - Parsed ride data
- `id` (uuid, primary key)
- `message_id` (foreign key → messages, nullable for manual entries)
- `group_id` (foreign key → groups)
- `type` (need_ride | ride_available)
- `from_location_id` (foreign key → location_contexts, nullable)
- `to_location_id` (foreign key → location_contexts, nullable)
- `from_location_text` (raw location text from message)
- `to_location_text` (raw location text from message)
- `departure_time` (timestamp, nullable if "now")
- `is_immediate` (boolean, for "now" requests)
- `cost` (decimal)
- `currency` (USD | EUR | INR, etc.)
- `distance` (nullable, in km)
- `seats_available` (nullable, for ride_available type)
- `status` (available | matched | completed | cancelled | expired)
- `poster_user_id` (foreign key → users, nullable)
- `created_at`, `updated_at`

**`matches`** - Rider-driver matches
- `id` (uuid, primary key)
- `ride_id` (foreign key → rides)
- `rider_id` (foreign key → users)
- `driver_id` (foreign key → users)
- `status` (pending | accepted | rejected | completed | cancelled)
- `matched_at`
- `accepted_at`, `completed_at`, `cancelled_at` (nullable)
- `created_at`, `updated_at`

**`locations`** - Master location data (optional, for autocomplete/suggestions)
- `id` (uuid, primary key)
- `name`
- `latitude`, `longitude`
- `address`
- `city`, `state`, `country`
- `metadata` (jsonb)
- `created_at`, `updated_at`

### Realtime Requirements
- New rides broadcast to active users
- Match notifications
- Ride status updates

## Development Commands

### Monorepo (once initialized)
```bash
# Install dependencies
pnpm install

# Build all packages
turbo build

# Run all apps in dev mode
turbo dev

# Run specific app
turbo dev --filter=mobile
turbo dev --filter=workers
```

### Supabase Local
```bash
# Start Supabase locally
supabase start

# Stop Supabase
supabase stop

# Run migrations
supabase db push

# Reset database
supabase db reset
```

### Flutter Mobile App
```bash
cd apps/mobile

# Install dependencies
flutter pub get

# Run on iOS
flutter run -d ios

# Run on Android
flutter run -d android

# Build APK
flutter build apk

# Build iOS
flutter build ios
```

### Workers Service
```bash
cd apps/workers

# Install dependencies
pnpm install

# Run in development
pnpm dev

# Run in production
pnpm start
```

## Critical Architectural Decisions

### Why Local-First?
- Cloud provider IPs (AWS, GCP, Azure) get blocked by certain services
- Full control over infrastructure
- Lower costs during development
- Easy transition to self-hosted VPS later

### Why Supabase Local?
- PostgreSQL with instant REST and GraphQL APIs
- Built-in realtime subscriptions
- Auth and storage included
- Familiar ecosystem for web developers
- Can self-host in production

### Why Flutter?
- True cross-platform (iOS + Android from single codebase)
- Excellent performance (compiled to native)
- Rich UI component library (Material + Cupertino)
- Growing ecosystem

### Why Turborepo?
- Fast incremental builds with caching
- Simple configuration
- Works well with mixed language repos (Dart + TypeScript)

### Message Source Automation Risks
- **WhatsApp**: Using unofficial libraries (whatsapp-web.js, baileys) violates WhatsApp ToS
  - Account may be banned
  - Mitigation: Use dedicated accounts, rate limiting, avoid spam behavior
  - Future: Consider WhatsApp Business API (requires approval + costs)
- **Telegram**: Bot API is official and safe, MTProto requires proper authentication
- **Other platforms**: Review ToS and use official APIs when available

## Important Conventions

### Code Style
- **Dart**: Follow official Dart style guide
- **TypeScript**: Use ESLint + Prettier
- **Commits**: Conventional commits format

### State Management (Flutter)
- To be decided: Riverpod, Bloc, or Provider

### API Design
- Use GraphQL subscriptions for realtime features
- Implement proper error handling and validation
- Rate limiting on all endpoints

### Security Considerations
- Never commit WhatsApp credentials
- Implement proper authentication
- Validate all parsed data
- Sanitize user inputs
- Secure local database access

## Location Context System

Each group has specific location contexts where certain location names map to actual coordinates or areas. This must be configurable per group.

Example:
- Group: "Downtown Riders"
  - "Station" → Central Train Station (lat/long)
  - "Mall" → City Center Mall (lat/long)
  - "Airport" → International Airport (lat/long)

Configuration stored in the `location_contexts` table, linked to groups via `group_id`.

## Next Steps for Setup

When ready to initialize the project:
1. Initialize Turborepo structure
2. Set up Supabase local instance
3. Create Flutter app skeleton
4. Set up workers service
5. Design and implement database schema
6. Create GraphQL API layer
7. Implement message parsing engine
8. Build Flutter UI for Phase 1 features

## Resources

- Flutter: https://flutter.dev
- Supabase: https://supabase.com/docs/guides/self-hosting/docker
- Turborepo: https://turbo.build/repo/docs
- whatsapp-web.js: https://github.com/pedroslopez/whatsapp-web.js
- GraphQL: https://graphql.org
