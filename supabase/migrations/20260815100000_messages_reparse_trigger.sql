-- supabase/migrations/20260815100000_messages_reparse_trigger.sql
--
-- Make FAILED messages re-drivable without restarting the workers service.
-- The original messages_inserted trigger fires on INSERT only, so resetting
-- parse_status back to PENDING never re-notified the parser. This trigger
-- reuses notify_message_inserted() (20260406000000) for status resets.
--
-- Writer safety: the parser's status writers only ever move rows AWAY from
-- PENDING (SUCCESS/FAILED/SKIPPED), and incrementRetryCount never touches
-- parse_status — so normal pipeline writes cannot re-fire this trigger.

drop trigger if exists message_reparse_trigger on messages;
create trigger message_reparse_trigger
  after update of parse_status on messages
  for each row
  when (new.parse_status = 'PENDING' and old.parse_status is distinct from new.parse_status)
  execute function notify_message_inserted();

-- One-call requeue for the failed-parse review loop: fix a regex pattern,
-- rebuild workers, then `select requeue_failed_messages();` and watch the
-- rows drain live. Optionally scoped to a single group.
create or replace function requeue_failed_messages(p_group_id uuid default null)
returns integer
language sql
as $$
  with updated as (
    update messages
       set parse_status = 'PENDING',
           parse_error  = null,
           retry_count  = 0
     where parse_status = 'FAILED'
       and (p_group_id is null or group_id = p_group_id)
     returning 1
  )
  select count(*)::int from updated;
$$;
