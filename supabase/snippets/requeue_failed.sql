-- Requeue FAILED messages for a live re-parse (no workers restart needed —
-- the message_reparse_trigger re-notifies the parser on the status reset).

-- All failed messages:
select requeue_failed_messages();

-- Only one group:
-- select requeue_failed_messages('00000000-0000-0000-0000-000000000000');

-- A single message by id:
-- update messages
--    set parse_status = 'PENDING', parse_error = null, retry_count = 0
--  where id = '00000000-0000-0000-0000-000000000000';
