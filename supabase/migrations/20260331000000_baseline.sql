-- Enable UUID extension
create extension if not exists "uuid-ossp";

-- ============================================================
-- users
-- ============================================================
create type user_role as enum ('rider', 'driver', 'both');

create table users (
  id          uuid primary key default uuid_generate_v4(),
  email       text unique,
  phone       text unique,
  name        text not null,
  role        user_role not null default 'both',
  avatar_url  text,
  created_at  timestamptz not null default now(),
  updated_at  timestamptz not null default now()
);

-- ============================================================
-- groups
-- ============================================================
create table groups (
  id          uuid primary key default uuid_generate_v4(),
  name        text not null,
  description text,
  is_active   boolean not null default true,
  created_at  timestamptz not null default now(),
  updated_at  timestamptz not null default now()
);

-- ============================================================
-- locations  (master geocoded places — autocomplete / resolution)
-- ============================================================
create table locations (
  id         uuid primary key default uuid_generate_v4(),
  name       text not null,
  latitude   numeric(10, 7) not null,
  longitude  numeric(10, 7) not null,
  address    text,
  city       text,
  state      text,
  country    text,
  metadata   jsonb not null default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

-- ============================================================
-- group_sources
-- ============================================================
create type source_type as enum ('whatsapp', 'telegram', 'discord', 'slack', 'manual');

create table group_sources (
  id                uuid primary key default uuid_generate_v4(),
  group_id          uuid not null references groups (id) on delete cascade,
  source_type       source_type not null,
  source_identifier text not null,
  last_parsed_at    timestamptz,
  parse_frequency   integer not null default 5, -- minutes
  metadata          jsonb not null default '{}',
  is_active         boolean not null default true,
  created_at        timestamptz not null default now(),
  updated_at        timestamptz not null default now()
);

-- ============================================================
-- location_contexts
-- ============================================================
create table location_contexts (
  id             uuid primary key default uuid_generate_v4(),
  group_id       uuid not null references groups (id) on delete cascade,
  location_alias text not null,
  location_name  text not null,
  location_id    uuid references locations (id) on delete set null,
  metadata       jsonb not null default '{}',
  created_at     timestamptz not null default now(),
  updated_at     timestamptz not null default now(),
  unique (group_id, location_alias)
);

-- ============================================================
-- messages
-- ============================================================
create type parse_status as enum ('pending', 'success', 'failed', 'skipped');

create table messages (
  id                uuid primary key default uuid_generate_v4(),
  group_id          uuid not null references groups (id) on delete cascade,
  source_message_id text,
  sender_identifier text,
  content           text not null,
  timestamp         timestamptz not null,
  parsed_at         timestamptz,
  parse_status      parse_status not null default 'pending',
  parse_error       text,
  metadata          jsonb not null default '{}',
  created_at        timestamptz not null default now(),
  unique (group_id, source_message_id)
);

-- ============================================================
-- rides
-- ============================================================
create type ride_type as enum ('need_ride', 'ride_available');
create type ride_status as enum ('available', 'matched', 'completed', 'cancelled', 'expired');

create table rides (
  id                 uuid primary key default uuid_generate_v4(),
  message_id         uuid references messages (id) on delete set null,
  group_id           uuid not null references groups (id) on delete cascade,
  type               ride_type not null,
  from_location_id   uuid references location_contexts (id) on delete set null,
  to_location_id     uuid references location_contexts (id) on delete set null,
  from_location_text text,
  to_location_text   text,
  departure_time     timestamptz,
  is_immediate       boolean not null default false,
  cost               numeric(10, 2),
  currency           text not null default 'USD',
  distance           numeric(10, 2), -- km
  seats_available    integer,
  status             ride_status not null default 'available',
  poster_user_id     uuid references users (id) on delete set null,
  created_at         timestamptz not null default now(),
  updated_at         timestamptz not null default now()
);

-- ============================================================
-- matches
-- ============================================================
create type match_status as enum ('pending', 'accepted', 'rejected', 'completed', 'cancelled');

create table matches (
  id           uuid primary key default uuid_generate_v4(),
  ride_id      uuid not null references rides (id) on delete cascade,
  rider_id     uuid not null references users (id) on delete cascade,
  driver_id    uuid not null references users (id) on delete cascade,
  status       match_status not null default 'pending',
  matched_at   timestamptz not null default now(),
  accepted_at  timestamptz,
  completed_at timestamptz,
  cancelled_at timestamptz,
  created_at   timestamptz not null default now(),
  updated_at   timestamptz not null default now()
);

-- ============================================================
-- updated_at triggers
-- ============================================================
create or replace function set_updated_at()
returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger users_updated_at          before update on users           for each row execute function set_updated_at();
create trigger groups_updated_at         before update on groups          for each row execute function set_updated_at();
create trigger group_sources_updated_at  before update on group_sources   for each row execute function set_updated_at();
create trigger location_contexts_updated_at before update on location_contexts for each row execute function set_updated_at();
create trigger rides_updated_at          before update on rides           for each row execute function set_updated_at();
create trigger matches_updated_at        before update on matches         for each row execute function set_updated_at();
create trigger locations_updated_at      before update on locations       for each row execute function set_updated_at();

-- ============================================================
-- indexes
-- ============================================================
create index on group_sources (group_id);
create index on location_contexts (group_id);
create index on messages (group_id, parse_status);
create index on messages (timestamp desc);
create index on rides (group_id, status);
create index on rides (type, status);
create index on rides (departure_time);
create index on matches (ride_id);
create index on matches (rider_id);
create index on matches (driver_id);

-- ============================================================
-- Row Level Security
-- ============================================================
alter table users            enable row level security;
alter table groups           enable row level security;
alter table group_sources    enable row level security;
alter table location_contexts enable row level security;
alter table messages         enable row level security;
alter table rides            enable row level security;
alter table matches          enable row level security;
alter table locations        enable row level security;

-- users: read own row; service role bypasses RLS
create policy "users: read own" on users
  for select using (auth.uid() = id);

create policy "users: update own" on users
  for update using (auth.uid() = id);

-- groups: all authenticated users can read active groups
create policy "groups: read active" on groups
  for select using (is_active = true and auth.role() = 'authenticated');

-- rides: all authenticated users can read available rides
create policy "rides: read available" on rides
  for select using (auth.role() = 'authenticated');

create policy "rides: insert own" on rides
  for insert with check (auth.uid() = poster_user_id);

create policy "rides: update own" on rides
  for update using (auth.uid() = poster_user_id);

-- matches: rider or driver can read their own matches
create policy "matches: read own" on matches
  for select using (auth.uid() = rider_id or auth.uid() = driver_id);

-- location_contexts: authenticated users can read
create policy "location_contexts: read" on location_contexts
  for select using (auth.role() = 'authenticated');

-- locations: authenticated users can read
create policy "locations: read" on locations
  for select using (auth.role() = 'authenticated');

-- messages: authenticated users can read (workers write via service role)
create policy "messages: read" on messages
  for select using (auth.role() = 'authenticated');
