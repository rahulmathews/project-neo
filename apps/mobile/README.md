# Project Neo — Mobile App

Flutter app (Riverpod + go_router + graphql_flutter + supabase_flutter). Auth via
Supabase, data via the Go GraphQL API, realtime via GraphQL websocket subscriptions.

## Prerequisites

- Backend running locally (from repo root):
  ```bash
  cp .env.example .env            # First time only — includes SUPABASE_JWKS_URL, needed so the
                                  # API can verify Supabase's ES256 user tokens
  supabase start                  # Postgres + Auth on :54321/:54322, applies migrations
  docker compose up -d graphql-api  # GraphQL API on :8082 (workers optional — needs WhatsApp session)
  ```
- Flutter SDK 3.44+, and for Android: Android SDK + an AVD (or a USB device).

## Configuration (compile-time defines)

The app reads four `--dart-define`s at build time — there is no .env file:

| Define | Purpose | Local value (Android emulator) |
|---|---|---|
| `SUPABASE_URL` | Supabase Auth/API | `http://10.0.2.2:54321` |
| `SUPABASE_ANON_KEY` | Supabase anon key | from `supabase status` (ANON_KEY) |
| `GRAPHQL_API_URL` | GraphQL HTTP endpoint | `http://10.0.2.2:8082/query` |
| `GRAPHQL_WS_URL` | GraphQL websocket endpoint | `ws://10.0.2.2:8082/query` |

`10.0.2.2` is how the Android emulator reaches the host machine. On a physical
device use your machine's LAN IP; on web/desktop use `localhost` / `127.0.0.1`.

## Run (Android emulator)

```bash
cd apps/mobile
flutter pub get
flutter run \
  --dart-define=SUPABASE_URL=http://10.0.2.2:54321 \
  --dart-define=SUPABASE_ANON_KEY=<ANON_KEY from supabase status> \
  --dart-define=GRAPHQL_API_URL=http://10.0.2.2:8082/query \
  --dart-define=GRAPHQL_WS_URL=ws://10.0.2.2:8082/query
```

Sign up with any email/password — local Supabase auto-confirms emails. The Rides
tab lists parsed `NEED_RIDE` messages for the selected group. There is no seed
data: groups and rides appear only after the workers service pairs with WhatsApp
and parses real messages, so an empty feed on a fresh database is expected.

## Code generation

Riverpod providers use code-gen; generated `*.g.dart` files are checked in.
After changing an annotated provider:

```bash
dart run build_runner build --delete-conflicting-outputs
```
