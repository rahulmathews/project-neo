-- ============================================================
-- NOTIFY triggers for GraphQL subscriptions
-- ============================================================

-- Notify when a new ride is inserted
create or replace function notify_ride_added()
returns trigger language plpgsql as $$
begin
  perform pg_notify('rides_added', new.id::text);
  return new;
end;
$$;

create trigger ride_added_trigger
  after insert on rides
  for each row execute function notify_ride_added();

-- Notify when a ride is updated
create or replace function notify_ride_updated()
returns trigger language plpgsql as $$
begin
  perform pg_notify('rides_updated', new.id::text);
  return new;
end;
$$;

create trigger ride_updated_trigger
  after update on rides
  for each row execute function notify_ride_updated();

-- Notify when a match is updated
create or replace function notify_match_updated()
returns trigger language plpgsql as $$
begin
  perform pg_notify('matches_updated', new.id::text);
  return new;
end;
$$;

create trigger match_updated_trigger
  after update on matches
  for each row execute function notify_match_updated();
