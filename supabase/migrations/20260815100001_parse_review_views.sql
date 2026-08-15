-- supabase/migrations/20260815100001_parse_review_views.sql
--
-- Review surface for the failed-parse feedback loop, consumed via Supabase
-- Studio's SQL editor / table view (no app code reads these).
-- security_invoker: the view runs with the caller's privileges, so RLS on
-- messages/groups still applies for non-superuser roles.

create or replace view parse_failures
with (security_invoker = on) as
select m.id,
       m.group_id,
       g.name as group_name,
       m.content,
       m.parse_error,
       m.retry_count,
       m.timestamp
from messages m
join groups g on g.id = m.group_id
where m.parse_status = 'FAILED'
order by m.timestamp desc;

create or replace view parse_stats
with (security_invoker = on) as
select date_trunc('day', timestamp) as day,
       parse_status,
       count(*) as n
from messages
group by 1, 2
order by 1 desc, 2;
