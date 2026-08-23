-- Manually classify a reviewed FAILED message as not-a-ride so it stops
-- showing up in parse_failures (mirrors the parser's SKIPPED terminal state).

-- update messages
--    set parse_status = 'SKIPPED', parse_error = 'manual review: not a ride'
--  where id = '00000000-0000-0000-0000-000000000000';

-- Bulk variant for an obvious non-ride phrase:
-- update messages
--    set parse_status = 'SKIPPED', parse_error = 'manual review: not a ride'
--  where parse_status = 'FAILED' and content ilike '%apartment%';
