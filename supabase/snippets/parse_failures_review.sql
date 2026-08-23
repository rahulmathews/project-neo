-- Failed-parse review (paste into Supabase Studio SQL editor, or psql -f).
-- Loop: review rows here → extend apps/workers/parser/regex.go →
-- `docker compose up -d --build workers` → run supabase/snippets/requeue_failed.sql.

select id, group_name, content, parse_error, retry_count, timestamp
from parse_failures
limit 100;

-- Accuracy trend by day:
-- select * from parse_stats;

-- Most common failure shapes (first line of the message):
-- select split_part(content, E'\n', 1) as first_line, count(*)
-- from parse_failures group by 1 order by 2 desc limit 25;
