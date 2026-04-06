-- supabase/migrations/20260406000000_messages_notify_trigger.sql

create or replace function notify_message_inserted()
returns trigger language plpgsql as $$
begin
  perform pg_notify('messages_inserted', new.id::text);
  return new;
end;
$$;

drop trigger if exists message_inserted_trigger on messages;
create trigger message_inserted_trigger
  after insert on messages
  for each row execute function notify_message_inserted();
